(function() {
  var _humanizeBytes, _humanizeTime, _humanizeUnits, _humanizeWithSuffixes, addColumns, addRow, app, humanize, renderChart, renderGraph;

  app = angular.module('app', ['ui.bootstrap', 'ngClipboard', 'googlechart']);

  app.filter('shorten', function() {
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
    var edge, g, j, k, len, len1, len2, m, maxX, node, ref, ref1, ref2, scale;
    g = new dagreD3.Digraph();
    ref = _.values($scope.nodes);
    for (j = 0, len = ref.length; j < len; j++) {
      node = ref[j];
      node.label = node.name;
      g.addNode(node.fqname, node);
    }
    ref1 = _.values($scope.nodes);
    for (k = 0, len1 = ref1.length; k < len1; k++) {
      node = ref1[k];
      ref2 = node.edges;
      for (m = 0, len2 = ref2.length; m < len2; m++) {
        edge = ref2[m];
        g.addEdge(null, edge.from, edge.to, {});
      }
    }
    (new dagreD3.Renderer()).zoom(false).run(g, d3.select("g"));
    maxX = 0.0;
    d3.selectAll("g.node").each(function(id) {
      var xCoord;
      d3.select(this).classed(g.node(id).type, true);
      d3.select(this).attr('ng-click', "selectNode('" + id + "')");
      d3.select(this).attr('ng-class', "[node.fqname=='" + id + "'?'seled':'',nodes['" + id + "'].state]");
      xCoord = parseFloat(d3.select(this).attr('transform').substr(10).split(',')[0]);
      if (xCoord > maxX) {
        return maxX = xCoord;
      }
    });
    maxX += 100;
    if (maxX < 750.0) {
      maxX = 750.0;
    }
    scale = 750.0 / maxX;
    d3.selectAll("g#top").each(function(id) {
      return d3.select(this).attr('transform', 'translate(5,5) scale(' + scale + ')');
    });
    d3.selectAll("g.node.stage rect").each(function(id) {
      return d3.select(this).attr('rx', 20).attr('ry', 20);
    });
    d3.selectAll("g.node.pipeline rect").each(function(id) {
      return d3.select(this).attr('rx', 0).attr('ry', 0);
    });
    return $compile(angular.element(document.querySelector('#top')).contents())($scope);
  };

  addRow = function(chart, columns, name, units, stats) {
    var column, j, len, row;
    row = [name];
    for (j = 0, len = columns.length; j < len; j++) {
      column = columns[j];
      row.push({
        v: stats[column],
        f: humanize(stats[column], units)
      });
    }
    return chart.data.push(row);
  };

  addColumns = function(chart, columns) {
    var column, j, len, results;
    chart.data = [['Stages']];
    results = [];
    for (j = 0, len = columns.length; j < len; j++) {
      column = columns[j];
      results.push(chart.data[0].push(column.replace(/_/g, ' ')));
    }
    return results;
  };

  humanize = function(num, units) {
    var s;
    if (units === 'bytes') {
      s = _humanizeBytes(num);
    } else if (units === 'kilobytes') {
      s = _humanizeBytes(num * 1024);
    } else if (units === 'seconds') {
      s = _humanizeTime(num);
    } else {
      s = _humanizeUnits(num, units);
    }
    return s.trim();
  };

  _humanizeWithSuffixes = function(num, suffixes, base) {
    var i;
    i = 0;
    while (num > base && i < suffixes.length - 1) {
      num = num / base;
      i += 1;
    }
    return [num, suffixes[i]];
  };

  _humanizeTime = function(num) {
    var ref, suffix;
    ref = _humanizeWithSuffixes(num, ['seconds', 'minutes', 'hours'], 60), num = ref[0], suffix = ref[1];
    num = num.toFixed(2);
    return num.toString() + ' ' + suffix;
  };

  _humanizeBytes = function(num) {
    var ref, suffix;
    ref = _humanizeWithSuffixes(num, ['B', 'KB', 'MB', 'GB', 'TB'], 1024), num = ref[0], suffix = ref[1];
    num = Math.round(num);
    return num.toString() + ' ' + suffix;
  };

  _humanizeUnits = function(num, units) {
    var c, i, j, l, ref, s;
    if (num >= 1000) {
      num = Math.round(num);
      s = '';
      l = num.toString().length;
      ref = num.toString();
      for (i = j = ref.length - 1; j >= 0; i = j += -1) {
        c = ref[i];
        s = c + s;
        if ((l - i) % 3 === 0 && i > 0) {
          s = ',' + s;
        }
      }
    } else {
      if (num % 1 !== 0) {
        num = num.toFixed(2);
      }
      s = num.toString();
    }
    return s + ' ' + units;
  };

  renderChart = function($scope, columns, units) {
    var chart, chunk, fork, height, j, k, len, len1, name, node, pnode, ref, stage, stages;
    if ($scope.node) {
      node = $scope.node;
    } else {
      node = $scope.topnode;
    }
    pnode = $scope.pnode;
    chart = {};
    chart.type = $scope.charttype;
    addColumns(chart, columns);
    if (pnode.type === "pipeline") {
      stages = _.sortBy(pnode.forks[$scope.forki].stages, function(stage) {
        return [stage.name, stage.fqname];
      });
      for (j = 0, len = stages.length; j < len; j++) {
        stage = stages[j];
        name = $scope.pnodes[stage.fqname].name;
        fork = $scope.pnodes[stage.fqname].forks[stage.forki];
        addRow(chart, columns, name, units, fork.fork_stats);
      }
    }
    if (pnode.type === "stage") {
      fork = pnode.forks[$scope.forki];
      ref = fork.chunks;
      for (k = 0, len1 = ref.length; k < len1; k++) {
        chunk = ref[k];
        addRow(chart, columns, 'chunk ' + chunk.index, units, chunk.chunk_stats);
      }
      if (fork.split_stats) {
        addRow(chart, columns, 'split', units, fork.split_stats);
      }
      if (fork.join_stats) {
        addRow(chart, columns, 'join', units, fork.join_stats);
      }
    }
    height = Math.max(chart.data.length * 20, 100);
    chart.options = {
      legend: 'none',
      height: height,
      chartArea: {
        width: '40%',
        height: '90%'
      }
    };
    return chart;
  };

  app.controller('MartianGraphCtrl', function($scope, $compile, $http, $interval) {
    var ref, selected, tab;
    $scope.pname = pname;
    $scope.psid = psid;
    $scope.admin = admin;
    $scope.adminstyle = adminstyle;
    $scope.release = release;
    $scope.urlprefix = adminstyle ? '/admin' : '/';
    $http.get("/api/get-state/" + container + "/" + pname + "/" + psid).success(function(state) {
      $scope.topnode = state.nodes[0];
      $scope.nodes = _.indexBy(state.nodes, 'fqname');
      $scope.info = state.info;
      return renderGraph($scope, $compile);
    });
    $scope.id = null;
    $scope.forki = 0;
    $scope.chunki = 0;
    $scope.mdviews = {
      forks: {},
      split: {},
      join: {},
      chunks: {}
    };
    $scope.expand = {
      node: {},
      forks: {},
      chunks: {}
    };
    $scope.mdfilters = ['profile_cpu_bin', 'profile_line_bin', 'profile_mem_bin', 'heartbeat'];
    $scope.showRestart = true;
    $scope.showLog = false;
    $scope.perf = false;
    $scope.charts = {};
    $scope.charttype = 'BarChart';
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
      time: {
        columns: ['usertime', 'systemtime'],
        units: 'seconds'
      },
      cpu: {
        columns: ['core_hours']
      },
      memory: {
        columns: ['maxrss'],
        units: 'kilobytes'
      },
      io: {
        columns: ['total_blocks', 'in_blocks', 'out_blocks']
      },
      iorate: {
        columns: ['total_blocks_rate', 'in_blocks_rate', 'out_blocks_rate']
      },
      jobs: {
        columns: ['num_jobs']
      },
      vdr: {
        columns: ['vdr_bytes'],
        units: 'bytes'
      }
    };
    if (admin) {
      $scope.stopRefresh = $interval(function() {
        return $scope.refresh();
      }, 30000);
    }
    $scope.$watch('perf', function() {
      if ($scope.perf) {
        return $http.get("/api/get-perf/" + container + "/" + pname + "/" + psid).success(function(state) {
          $scope.pnodes = _.indexBy(state.nodes, 'fqname');
          return $scope.pnode = $scope.pnodes[$scope.topnode.fqname];
        });
      }
    });
    ref = $scope.tabs;
    for (tab in ref) {
      selected = ref[tab];
      $scope.$watch('tabs.' + tab, function() {
        return $scope.getChart();
      });
    }
    $scope.$watch('forki', function() {
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
      var ref1;
      ref1 = $scope.tabs;
      for (tab in ref1) {
        selected = ref1[tab];
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
        units = $scope.chartopts[active].units ? $scope.chartopts[active].units : '';
        return $scope.charts[$scope.forki] = renderChart($scope, columns, units);
      }
    };
    $scope.setChartType = function(charttype) {
      $scope.charttype = charttype;
      return $scope.getChart();
    };
    $scope.copyToClipboard = function() {
      return '';
    };
    $scope.selectNode = function(id) {
      $scope.id = id;
      $scope.node = $scope.nodes[id];
      $scope.forki = 0;
      $scope.chunki = 0;
      $scope.mdviews = {
        forks: {},
        split: {},
        join: {},
        chunks: {}
      };
      $scope.expand = {
        node: {},
        forks: {},
        chunks: {}
      };
      if ($scope.perf) {
        $scope.pnode = $scope.pnodes[id];
        return $scope.getChart();
      }
    };
    $scope.restart = function() {
      $scope.showRestart = false;
      return $http.post("/api/restart/" + container + "/" + pname + "/" + psid).success(function(data) {
        return $scope.stopRefresh = $interval(function() {
          return $scope.refresh();
        }, 3000);
      }).error(function() {
        $scope.showRestart = true;
        console.log('Server responded with an error for /api/restart, so stopping auto-refresh.');
        $interval.cancel($scope.stopRefresh);
        return alert('mrp is no longer running.\n\nPlease run mrp again with the --noexit option to continue running the pipeline.');
      });
    };
    $scope.expandString = function(view, index, name) {
      if ($scope.expand[view][index] == null) {
        $scope.expand[view][index] = {};
      }
      return $scope.expand[view][index][name] = true;
    };
    $scope.selectMetadata = function(view, index, name, path) {
      return $http.post("/api/get-metadata/" + container + "/" + pname + "/" + psid, {
        path: path,
        name: name
      }, {
        transformResponse: function(d) {
          return d;
        }
      }).success(function(metadata) {
        return $scope.mdviews[view][index] = metadata;
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
      return $http.get("/api/get-state/" + container + "/" + pname + "/" + psid).success(function(state) {
        $scope.nodes = _.indexBy(state.nodes, 'fqname');
        if ($scope.id) {
          $scope.node = $scope.nodes[$scope.id];
        }
        $scope.info = state.info;
        return $scope.showRestart = true;
      }).error(function() {
        console.log('Server responded with an error for /api/get-state, so stopping auto-refresh.');
        return $interval.cancel($scope.stopRefresh);
      });
    };
  });

}).call(this);
