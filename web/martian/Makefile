serve: serve/favicon.ico serve/graph.js \
       serve/graph.html \
       serve/css/main.css

NODE_MODULES=node_modules/d3/dist/d3.min.js \
	node_modules/clean-css-cli/bin/cleancss \
	node_modules/html-minifier-terser/cli.js

$(NODE_MODULES): node_modules

node_modules: yarn.lock package.json
	yarn install --no-audit --frozen-lockfile --ignore-optional

serve/graph.js: node_modules/d3/dist/d3.min.js \
                node_modules/dagre-d3/dist/dagre-d3.min.js \
                node_modules/angular/angular.min.js \
                node_modules/angular-ui-bootstrap/ui-bootstrap-tpls.min.js \
                node_modules/lodash/lodash.min.js \
                node_modules/angular-google-chart/ng-google-chart.min.js \
                client/graph.js | $(NODE_MODULES)
	mkdir -p serve
	cat $^ | node_modules/terser/bin/terser | gzip --best > $@

build/css/main.css: res/css/main.css node_modules/clean-css-cli/bin/cleancss
	mkdir -p build/css
	node_modules/clean-css-cli/bin/cleancss -o $@ $<

serve/css/main.css: build/css/main.css
	mkdir -p serve/css
	gzip --best -c $< > $@

serve/graph.html: templates/graph.html node_modules/html-minifier-terser/cli.js
	mkdir -p serve
	node_modules/html-minifier-terser/cli.js -o $@ \
	--remove-comments --collapse-whitespace \
	--continue-on-parse-error --minify-css true \
	--minify-js '{ "mangle": false }' $<

serve/favicon.ico: res/favicon.ico
	mkdir -p serve
	gzip --best -c $< > $@

clean:
	rm -rf node_modules build serve
