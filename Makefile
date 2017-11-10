#
# Copyright (c) 2014-2017 10X Genomics, Inc. All rights reserved.
#
# Build a Go package with git version embedding.
#

GOBINS=mrc mrf mrg mrp mrs mrt_helper mrstat mrjob
GOLIBTESTS=$(addprefix test-, core util syntax adapter)
GOBINTESTS=$(addprefix test-, $(GOBINS))
GOTESTS=$(GOLIBTESTS) $(GOBINTESTS) test-all
VERSION=$(shell git describe --tags --always --dirty)
RELEASE=false
GO_FLAGS=-ldflags "-X martian/util.__VERSION__='$(VERSION)' -X martian/util.__RELEASE__='$(RELEASE)'"

export GOPATH=$(shell pwd)

.PHONY: $(GOBINS) grammar web $(GOTESTS) govet bin/sum_squares longtests

#
# Targets for development builds.
#
all: grammar $(GOBINS) web test

bin/goyacc: src/vendor/golang.org/x/tools/cmd/goyacc/yacc.go
	go install vendor/golang.org/x/tools/cmd/goyacc

src/martian/syntax/grammar.go: bin/goyacc src/martian/syntax/grammar.y
	bin/goyacc -p "mm" -o src/martian/syntax/grammar.go src/martian/syntax/grammar.y && rm y.output

bin/sum_squares: test/split_test_go/stages/sum_squares/sum_squares.go
	go build -o $@ $<

grammar: src/martian/syntax/grammar.go

$(GOBINS):
	go install $(GO_FLAGS) martian/cmd/$@

web:
	(cd web/martian && npm install && gulp)

mrt:
	cp scripts/mrt bin/mrt

$(GOLIBTESTS): test-%:
	go test -v martian/$*

$(GOBINTESTS): test-%:
	go test -v martian/cmd/$*

WEB_FILES=web/martian/serve web/martian/templates/graph.html

$(WEB_FILES): web

ADAPTERS=$(wildcard adapters/python/*.py) $(wildcard adapters/python/*/*.py)
JOBMANAGERS=$(wildcard jobmanagers/*.py) \
			$(wildcard jobmanagers/*.json) \
			$(wildcard jobmanagers/*.template.example)

PRODUCT_NAME:=martian-$(VERSION)-$(shell uname -is | tr "A-Z " "a-z-")

$(PRODUCT_NAME).tar.%: $(addprefix bin/, $(GOBINS)) $(ADAPTERS) $(JOBMANAGERS) $(WEB_FILES)
	tar --owner=0 --group=0 --transform "s/^./$(PRODUCT_NAME)/" -caf $@ $(addprefix ./, $^)

tarball: $(PRODUCT_NAME).tar.gz

test-all:
	go test -v martian/...

govet:
	go tool vet src/martian

test: test-all govet bin/sum_squares

longtests: bin/sum_squares mrp mrjob
	test/martian_test.py test/split_test/split_test.json
	test/martian_test.py test/split_test_go/split_test.json
	test/martian_test.py test/files_test/files_test.json
	test/martian_test.py test/fork_test/fork_test.json
	test/martian_test.py test/fork_test/fail1_test.json
	test/martian_test.py test/fork_test/retry_test.json
	test/martian_test.py test/fork_test/autoretry_pass.json
	test/martian_test.py test/fork_test/autoretry_fail.json

clean:
	rm -rf $(GOPATH)/bin
	rm -rf $(GOPATH)/pkg
