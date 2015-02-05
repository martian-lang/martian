(function() {
  var addColumns, addRow, app, humanize, renderChart, renderGraph, _humanizeBytes, _humanizeTime, _humanizeUnits, _humanizeWithSuffixes;

  app = angular.module('app', ['ui.bootstrap', 'ngClipboard', 'googlechart']);

  app.filter('shorten', function() {
    return function(s) {
      s = s + "";
      if (s.length < 71) {
        return s;
      } else {
        return s.substr(0, 30) + " ... " + s.substr(s.length - 50);
      }
    };
  });

  renderGraph = function($scope, $compile) {
    var edge, g, maxX, node, scale, _i, _j, _k, _len, _len1, _len2, _ref, _ref1, _ref2;
    g = new dagreD3.Digraph();
    _ref = _.values($scope.nodes);
    for (_i = 0, _len = _ref.length; _i < _len; _i++) {
      node = _ref[_i];
      node.label = node.name;
      g.addNode(node.fqname, node);
    }
    _ref1 = _.values($scope.nodes);
    for (_j = 0, _len1 = _ref1.length; _j < _len1; _j++) {
      node = _ref1[_j];
      _ref2 = node.edges;
      for (_k = 0, _len2 = _ref2.length; _k < _len2; _k++) {
        edge = _ref2[_k];
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
    var column, row, _i, _len;
    row = [name];
    for (_i = 0, _len = columns.length; _i < _len; _i++) {
      column = columns[_i];
      row.push({
        v: stats[column],
        f: humanize(stats[column], units)
      });
    }
    return chart.data.push(row);
  };

  addColumns = function(chart, columns) {
    var column, _i, _len, _results;
    chart.data = [['Stages']];
    _results = [];
    for (_i = 0, _len = columns.length; _i < _len; _i++) {
      column = columns[_i];
      _results.push(chart.data[0].push(column.replace(/_/g, ' ')));
    }
    return _results;
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
    var suffix, _ref;
    _ref = _humanizeWithSuffixes(num, ['seconds', 'minutes', 'hours'], 60), num = _ref[0], suffix = _ref[1];
    num = num.toFixed(2);
    return num.toString() + ' ' + suffix;
  };

  _humanizeBytes = function(num) {
    var suffix, _ref;
    _ref = _humanizeWithSuffixes(num, ['B', 'KB', 'MB', 'GB', 'TB'], 1024), num = _ref[0], suffix = _ref[1];
    num = Math.round(num);
    return num.toString() + ' ' + suffix;
  };

  _humanizeUnits = function(num, units) {
    var c, i, l, s, _i, _ref;
    if (num >= 1000) {
      num = Math.round(num);
      s = '';
      l = num.toString().length;
      _ref = num.toString();
      for (i = _i = _ref.length - 1; _i >= 0; i = _i += -1) {
        c = _ref[i];
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
    var chart, chunk, fork, height, name, node, pnode, stage, stages, _i, _j, _len, _len1, _ref;
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
        return stage.name;
      });
      for (_i = 0, _len = stages.length; _i < _len; _i++) {
        stage = stages[_i];
        name = $scope.pnodes[stage.fqname].name;
        fork = $scope.pnodes[stage.fqname].forks[stage.forki];
        addRow(chart, columns, name, units, fork.fork_stats);
      }
    }
    if (pnode.type === "stage") {
      fork = pnode.forks[$scope.forki];
      _ref = fork.chunks;
      for (_j = 0, _len1 = _ref.length; _j < _len1; _j++) {
        chunk = _ref[_j];
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
    var selected, tab, _ref;
    $scope.pname = pname;
    $scope.psid = psid;
    $scope.admin = admin;
    $scope.adminstyle = adminstyle;
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
    _ref = $scope.tabs;
    for (tab in _ref) {
      selected = _ref[tab];
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
    $scope.getActiveTab = function() {
      var _ref1;
      _ref1 = $scope.tabs;
      for (tab in _ref1) {
        selected = _ref1[tab];
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
        return alert('mrp is no longer running.\n\nPlease run mrp again with the --noexit option to continue running the pipeline.');
      });
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
