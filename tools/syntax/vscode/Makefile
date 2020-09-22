TSC=./node_modules/.bin/tsc

MODULES=extension format_provider mro
OUTS=$(addprefix out/src/, $(addsuffix .js, $(MODULES)) $(addsuffix .js.map, $(MODULES)))
INS=$(addprefix src/, $(addsuffix .ts, $(MODULES)))

.PHONY: all

all: $(OUTS) syntaxes/Martianlang.tmLanguage.json

$(OUTS): $(INS) tsconfig.json
	$(TSC) -p ./

syntaxes/Martianlang.tmLanguage.json: syntaxes/Martianlang.YAML-tmLanguage
	npx js-yaml $< > $@

clean:
	rm -rf out/ node_modules/