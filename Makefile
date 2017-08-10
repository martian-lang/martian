#
# Copyright (c) 2014-2017 10X Genomics, Inc. All rights reserved.
#
# Build a Go package with git version embedding.
#

GOBINS=mrc mrf mrg mrp mrs mrt_helper mrstat mrjob
GOTESTS=$(addprefix test-, $(GOBINS) core)
VERSION=$(shell git describe --tags --always --dirty)
RELEASE=false
GO_FLAGS=-ldflags "-X martian/core.__VERSION__='$(VERSION)' -X martian/core.__RELEASE__='$(RELEASE)'"

export GOPATH=$(shell pwd)

.PHONY: $(GOBINS) grammar web $(GOTESTS) bin/sum_squares

#
# Targets for development builds.
#
all: grammar $(GOBINS) web test

bin/goyacc: src/vendor/golang.org/x/tools/cmd/goyacc/yacc.go
	go install vendor/golang.org/x/tools/cmd/goyacc

src/martian/core/grammar.go: bin/goyacc src/martian/core/grammar.y
	bin/goyacc -p "mm" -o src/martian/core/grammar.go src/martian/core/grammar.y && rm y.output

bin/sum_squares: test/split_test_go/stages/sum_squares/sum_squares.go
	go build -o $@ $<

grammar: src/martian/core/grammar.go

$(GOBINS):
	go install $(GO_FLAGS) martian/$@

web:
	(cd web/martian && npm install && gulp)

mrt:
	cp scripts/mrt bin/mrt

$(GOTESTS): test-%:
	go test -v martian/$*

test: $(GOTESTS) bin/sum_squares

clean:
	rm -rf $(GOPATH)/bin
	rm -rf $(GOPATH)/pkg
