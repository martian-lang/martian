(function() {
  var app, renderGraph;

  app = angular.module('app', ['ui.bootstrap', 'ngClipboard']);

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
      g.addNode(node.name, node);
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
      d3.select(this).attr('ng-class', "[node.name=='" + id + "'?'seled':'',nodes['" + id + "'].state]");
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

  app.controller('MarioGraphCtrl', function($scope, $compile, $http, $interval) {
    $scope.pname = pname;
    $scope.psid = psid;
    $scope.admin = admin;
    $scope.adminstyle = adminstyle;
    $scope.urlprefix = adminstyle ? '/admin' : '/';
    $http.get("/api/get-state/" + container + "/" + pname + "/" + psid).success(function(state) {
      $scope.nodes = _.indexBy(state.nodes, 'name');
      $scope.error = state.error;
      return renderGraph($scope, $compile);
    });
    $scope.id = null;
    $scope.forki = 0;
    $scope.chunki = 0;
    $scope.mdviews = {
      fork: '',
      split: '',
      join: '',
      chunk: ''
    };
    $scope.showRestart = true;
    $scope.showLog = false;
    if (admin) {
      $scope.stopRefresh = $interval(function() {
        return $scope.refresh();
      }, 30000);
    }
    $scope.copyToClipboard = function() {
      return '';
    };
    $scope.selectNode = function(id) {
      $scope.id = id;
      $scope.node = $scope.nodes[id];
      $scope.forki = 0;
      $scope.chunki = 0;
      return $scope.mdviews = {
        fork: '',
        split: '',
        join: '',
        chunk: ''
      };
    };
    $scope.restart = function() {
      $scope.showRestart = false;
      return $http.post("/api/restart/" + container + "/" + pname + "/" + psid + "/" + $scope.node.fqname).success(function(data) {
        console.log(data);
        return $scope.stopRefresh = $interval(function() {
          return $scope.refresh();
        }, 3000);
      }).error(function() {
        $scope.showRestart = true;
        return alert('mrp is no longer running.\n\nPlease run mrp again with the --noexit option to continue running the pipeline.');
      });
    };
    $scope.selectMetadata = function(view, name, path) {
      return $http.post("/api/get-metadata/" + container + "/" + pname + "/" + psid, {
        path: path,
        name: name
      }, {
        transformResponse: function(d) {
          return d;
        }
      }).success(function(metadata) {
        return $scope.mdviews[view] = metadata;
      });
    };
    return $scope.refresh = function() {
      return $http.get("/api/get-state/" + container + "/" + pname + "/" + psid).success(function(state) {
        $scope.nodes = _.indexBy(state.nodes, 'name');
        if ($scope.id) {
          $scope.node = $scope.nodes[$scope.id];
        }
        $scope.showRestart = true;
        return $scope.error = state.error;
      }).error(function() {
        console.log('Server responded with an error for /api/get-state, so stopping auto-refresh.');
        return $interval.cancel($scope.stopRefresh);
      });
    };
  });

}).call(this);
