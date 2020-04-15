(function() {
  var _humanizeBytes, _humanizeTime, _humanizeUnits, _humanizeWithSuffix,
    addColumns, addRow, app, humanize, renderChart, renderGraph;

  app = angular.module("app", ["ui.bootstrap", "googlechart"]);

  app.filter("shorten", function() {
    return function(s, expand) {
      s = s + "";
      if (s.length < 71 || expand) {
        return s;
      } else {
        return s.substr(0, 30) + " ... " + s.substr(s.length - 50);
      }
    };
  });

  renderGraph = function($scope, $compile) {
    var g = new dagreD3.graphlib.Graph({
      directed: true
    }).setGraph({});
    g.ranker = "tight-tree";
    g.edgesep = 0;
    g.nodeSep = 30;
    g.marginx = 10;
    g.setDefaultEdgeLabel(function() {
      return {
        minlen: 1,
        curve: d3.curveBasis
      };
    });
    var ref = _.values($scope.nodes);
    for (var j = 0, len = ref.length; j < len; j++) {
      var node = ref[j];
      node.label = node.name;
      g.setNode(node.fqname, node);
    }
    ref = _.values($scope.nodes);
    for (var k = 0, len1 = ref.length; k < len1; k++) {
      var node = ref[k];
      var ref2 = node.edges;
      for (var m = 0, len2 = ref2.length; m < len2; m++) {
        var edge = ref2[m];
        g.setEdge(edge.from, edge.to);
      }
    }
    $scope.render($scope.graph, g);
    var maxX = 0.0;
    var maxY = 0.0;
    d3.selectAll("g.node").each(function(id) {
      var coords, element, xCoord, yCoord;
      element = d3.select(this);
      element.classed(g.node(id).type, true);
      element.attr("ng-click", "selectNode('" + id + "')");
      element.attr(
        "ng-class",
        "[node.fqname=='" + id + "'?'seled':'',nodes['" + id + "'].state]");
      coords = element.attr("transform").substr(10).split(",");
      xCoord = parseFloat(coords[0]);
      yCoord = parseFloat(coords[1]);
      if (xCoord > maxX) {
        maxX = xCoord;
      }
      if (yCoord > maxY) {
        return maxY = yCoord;
      }
    });
    maxX += 100;
    if (maxX < 720.0) {
      maxX = 720.0;
    }
    var scale = 720.0 / maxX;
    maxY += 100;
    if (maxY < 700) {
      maxY = 700;
    }
    $scope.svg.attr("width", "750px").attr("height", maxY.toString() + "px");
    $scope.graph.attr("transform", "translate(5,5) scale(" + scale + ")");
    $scope.graph.selectAll("g.node.stage rect").attr("rx", 20).attr("ry", 20);
    $scope.graph.selectAll("g.node.pipeline rect").attr("rx", 0).attr("ry", 0);
    $scope.zoom(g, 750, maxY, scale);
    return $compile($scope.top.contents())($scope);
  };

  addRow = function(chart, columns, name, units, stats) {
    var column, j, len, row;
    row = [name];
    for (j = 0, len = columns.length; j < len; j++) {
      column = columns[j];
      row.push({ v: stats[column], f: humanize(stats[column], units) });
    }
    return chart.data.push(row);
  };

  addColumns = function(chart, columns) {
    var column, j, len, results;
    chart.data = [["Stages"]];
    results = [];
    for (j = 0, len = columns.length; j < len; j++) {
      column = columns[j];
      results.push(chart.data[0].push(column.replace(/_/g, " ")));
    }
    return results;
  };

  humanize = function(num, units) {
    var s;
    if (units === "bytes") {
      s = _humanizeBytes(num);
    } else if (units === "kilobytes") {
      s = _humanizeBytes(num * 1024);
    } else if (units === "seconds") {
      s = _humanizeTime(num);
    } else {
      s = _humanizeUnits(num, units);
    }
    return s.trim();
  };

  _humanizeWithSuffix = function(num, suffixes, base, precision) {
    var i;
    i = 0;
    while (num > base && i < suffixes.length - 1) {
      num = num / base;
      i += 1;
    }
    return num.toFixed(precision) + " " + suffixes[i];
  };

  _humanizeTime = function(num) {
    return _humanizeWithSuffix(num, ["seconds", "minutes", "hours"], 60, 2);
  };

  _humanizeBytes = function(num) {
    var precision = (num > 1073741824) ? 1 : 0;
    return _humanizeWithSuffix(
      num, ["B", "KB", "MB", "GB", "TB"], 1024, precision);
  };

  _humanizeUnits = function(num, units) {
    var c, i, j, l, ref, s;
    if (num >= 1000) {
      num = Math.round(num);
      s = "";
      l = num.toString().length;
      ref = num.toString();
      for (i = j = ref.length - 1; j >= 0; i = j += -1) {
        c = ref[i];
        s = c + s;
        if ((l - i) % 3 === 0 && i > 0) {
          s = "," + s;
        }
      }
    } else {
      if (num % 1 !== 0) {
        num = num.toFixed(2);
      }
      s = num.toString();
    }
    return s + " " + units;
  };

  renderChart = function($scope, columns, units) {
    var chart, pnode;
    pnode = $scope.pnode;
    chart = {};
    chart.type = $scope.charttype;
    addColumns(chart, columns);
    if (pnode.type === "pipeline") {
      var stages = _.sortBy(pnode.forks[$scope.forki].stages, function(stage) {
        return [stage.name, stage.fqname];
      });
      for (var j = 0, len = stages.length; j < len; j++) {
        var stage = stages[j];
        var name = $scope.pnodes[stage.fqname].name;
        var fork = $scope.pnodes[stage.fqname].forks[stage.forki];
        addRow(chart, columns, name, units, fork.fork_stats);
      }
    }
    if (pnode.type === "stage") {
      var fork = pnode.forks[$scope.forki];
      var ref = fork.chunks;
      for (var k = 0, len1 = ref.length; k < len1; k++) {
        var chunk = ref[k];
        addRow(chart, columns, "chunk " + chunk.index, units, chunk.chunk_stats);
      }
      if (fork.split_stats) {
        addRow(chart, columns, "split", units, fork.split_stats);
      }
      if (fork.join_stats) {
        addRow(chart, columns, "join", units, fork.join_stats);
      }
    }
    var height = Math.max(chart.data.length * 20, 100);
    chart.options = {
      legend: "none",
      height: height,
      chartArea: { width: "40%", height: "90%" }
    };
    return chart;
  };

  app.controller("MartianGraphCtrl", function($scope, $compile, $http, $interval) {
    var auth, j, len, ref, ref1, selected, tab, v, zoom;
    $scope.pname = pname;
    $scope.psid = psid;
    $scope.admin = admin;
    $scope.adminstyle = adminstyle;
    $scope.release = release;
    $scope.urlprefix = adminstyle ? "/admin" : "/";
    $scope.files = files;
    auth = "";
    ref = window.location.search.substring(1).split("&");
    for (j = 0, len = ref.length; j < len; j++) {
      v = ref[j];
      if (v.split("=")[0] === "auth") {
        auth = "?" + v;
        break;
      }
    }
    $scope.top = angular.element(document.querySelector("#top"));
    $scope.svg = d3.select("div svg");
    $scope.graph = $scope.svg.select("g");
    $scope.render = dagreD3.render();
    zoom = d3.zoom().on("zoom", function() {
      return $scope.graph.attr("transform", d3.event.transform);
    });
    $scope.svg.call(zoom);
    $scope.zoom = function(g, width, height, scale) {
      return $scope.svg.call(
        zoom.transform, d3.zoomIdentity.translate(5, 5).scale(scale));
    };
    $http.get("/api/get-state/" + container + "/" + pname + "/" + psid + auth)
      .then(function(r) {
        var state;
        state = r.data;
        $scope.topnode = state.nodes[0];
        $scope.nodes = _.keyBy(state.nodes, "fqname");
        $scope.info = state.info;
        return renderGraph($scope, $compile);
      });
    $http
      .get(
        "/api/list-metadata-top/" + container + "/" + pname + "/" + psid +
        auth)
      .then(function(r) {
        return $scope.files = r.data;
      });
    $scope.id = null;
    $scope.forki = 0;
    $scope.chunki = 0;
    $scope.mdviews = { forks: {}, split: {}, join: {}, chunks: {} };
    $scope.expand = { node: {}, forks: {}, chunks: {} };
    $scope.mdfilters =
      ["profile_cpu_bin", "profile_line_bin", "profile_mem_bin", "heartbeat"];
    $scope.showRestart = true;
    $scope.showLog = false;
    $scope.perf = false;
    $scope.charts = {};
    $scope.charttype = "BarChart";
    $scope.tabs = {
      summary: true,
      time: false,
      cpu: false,
      io: false,
      iorate: false,
      memory: false,
      jobs: false,
      vdr: false
    };
    $scope.chartopts = {
      time: { columns: ["usertime", "systemtime"], units: "seconds" },
      cpu: { columns: ["core_hours"] },
      memory: { columns: ["maxrss"], units: "kilobytes" },
      io: { columns: ["total_blocks", "in_blocks", "out_blocks"] },
      iorate:
        { columns: ["total_blocks_rate", "in_blocks_rate", "out_blocks_rate"] },
      jobs: { columns: ["num_jobs"] },
      vdr: { columns: ["vdr_bytes"], units: "bytes" }
    };
    if (admin) {
      $scope.stopRefresh = $interval(function() {
        return $scope.refresh();
      }, 30000);
    }
    $scope.$watch("perf", function() {
      if ($scope.perf) {
        return $http
          .get("/api/get-perf/" + container + "/" + pname + "/" + psid + auth)
          .then(function(state) {
            $scope.pnodes = _.keyBy(state.data.nodes, "fqname");
            return $scope.pnode = $scope.pnodes[$scope.topnode.fqname];
          });
      }
    });
    ref1 = $scope.tabs;
    for (tab in ref1) {
      selected = ref1[tab];
      $scope.$watch("tabs." + tab, function() {
        return $scope.getChart();
      });
    }
    $scope.$watch("forki", function() {
      if ($scope.perf) {
        return $scope.getChart();
      }
    });
    $scope.humanize = function(name, units) {
      var fork;
      fork = $scope.pnode.forks[$scope.forki];
      return humanize(fork.fork_stats[name], units);
    };
    $scope.humanizeFromNode = function(name, units) {
      var node;
      node = $scope.pnode;
      return humanize(node[name], units);
    };
    $scope.getActiveTab = function() {
      var ref2;
      ref2 = $scope.tabs;
      for (tab in ref2) {
        selected = ref2[tab];
        if (selected) {
          return tab;
        }
      }
    };
    $scope.getChart = function() {
      var active, columns, units;
      active = $scope.getActiveTab();
      if ($scope.chartopts[active]) {
        columns = $scope.chartopts[active].columns;
        units =
          $scope.chartopts[active].units ? $scope.chartopts[active].units : "";
        return $scope.charts[$scope.forki] = renderChart($scope, columns, units);
      }
    };
    $scope.setChartType = function(charttype) {
      $scope.charttype = charttype;
      return $scope.getChart();
    };
    $scope.copyToClipboard = function() {
      return "";
    };
    $scope.selectNode = function(id) {
      $scope.id = id;
      $scope.node = $scope.nodes[id];
      $scope.forki = 0;
      $scope.chunki = 0;
      $scope.mdviews = { forks: {}, split: {}, join: {}, chunks: {} };
      $scope.expand = { node: {}, forks: {}, chunks: {} };
      if ($scope.perf) {
        $scope.pnode = $scope.pnodes[id];
        return $scope.getChart();
      }
    };
    $scope.restart = function() {
      $scope.showRestart = false;
      return $http
        .post("/api/restart/" + container + "/" + pname + "/" + psid + auth)
        .then(
          function(data) {
            return $scope.stopRefresh = $interval(function() {
              return $scope.refresh();
            }, 3000);
          },
          function(data, error) {
            $scope.showRestart = true;
            return alert(
              "Restart failed: error " + status + " (" + data +
              ").  mrp may no longer be running.\n\n" +
              "Please run mrp again with the --noexit option to continue running the pipeline.");
          });
    };
    $scope.expandString = function(view, index, name) {
      if ($scope.expand[view][index] == null) {
        $scope.expand[view][index] = {};
      }
      return $scope.expand[view][index][name] = true;
    };
    $scope.selectMetadata = function(view, index, name, path) {
      return $http
        .post(
          "/api/get-metadata/" + container + "/" + pname + "/" + psid + auth,
          { path: path, name: name },
          {
            transformResponse: function(d) {
              return d;
            }
          })
        .then(function(r) {
          return $scope.mdviews[view][index] = r.data;
        });
    };
    $scope.filterMetadata = function(name) {
      var found;
      found = _.find($scope.mdfilters, function(md) {
        return md === name;
      });
      return !found;
    };
    return $scope.refresh = function() {
      $http.get("/api/get-state/" + container + "/" + pname + "/" + psid + auth)
        .then(
          function(r) {
            var state;
            state = r.data;
            $scope.nodes = _.keyBy(state.nodes, "fqname");
            if ($scope.id) {
              $scope.node = $scope.nodes[$scope.id];
            }
            $scope.info = state.info;
            return $scope.showRestart = true;
          },
          function(data, status) {
            console.log(
              "Server responded with error " + status + ": " + data +
              " for /api/get-state, so stopping auto-refresh.");
            return $interval.cancel($scope.stopRefresh);
          });
      return $http
        .get(
          "/api/list-metadata-top/" + container + "/" + pname + "/" + psid +
          auth)
        .then(function(files) {
          return $scope.files = files.data;
        });
    };
  });
}).call(this);
