/*jslint node: true */
/*global ZeroClipboard */
'use strict';

angular.module('ngClipboard', []).
  provider('ngClip', function() {
    var self = this;
    this.path = '/js/ZeroClipboard.swf';
    return {
      setPath: function(newPath) {
       self.path = newPath;
      },
      $get: function() {
        return {
          path: self.path
        };
      }
    };
  }).
  run(['ngClip', function(ngClip) {
    ZeroClipboard.config({
      moviePath: ngClip.path,
      trustedDomains: ["*"],
      allowScriptAccess: "always",
      forceHandCursor: true
    });
  }]).
  directive('clipCopy', ['ngClip', function (ngClip) {
    return {
      scope: {
        clipCopy: '&',
        clipClick: '&'
      },
      restrict: 'A',
      link: function (scope, element, attrs) {
        // Create the clip object
        var clip = new ZeroClipboard(element);
        /*if (attrs.clipCopy == "") {*/
          scope.clipCopy = function(scope) {
            console.log(element[0].nextElementSibling.innerText)
            return element[0].nextElementSibling.innerText;
          };
        /*}*/
        clip.on( 'load', function(client) {
          var onDataRequested = function (client) {
            client.setText(scope.$eval(scope.clipCopy));
            if (angular.isDefined(attrs.clipClick)) {
              scope.$apply(scope.clipClick);
            }
          };
          client.on('dataRequested', onDataRequested);

          scope.$on('$destroy', function() {
            client.off('dataRequested', onDataRequested);
            client.unclip(element);
          });
        });
      }
    };
  }]);
