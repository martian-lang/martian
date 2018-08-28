#
# Copyright (c) 2014-2017 10X Genomics, Inc. All rights reserved.
#
# Build a Go package with git version embedding.
#

GOBINS=$(notdir $(wildcard cmd/*))
REPO=github.com/martian-lang/martian
GOLIBTESTS=$(addprefix test-, $(notdir $(wildcard martian/*)))
GOBINTESTS=$(addprefix test-, $(GOBINS))
GOTESTS=$(GOLIBTESTS) $(GOBINTESTS) test-all
VERSION=$(shell git describe --tags --always --dirty)
RELEASE=false
GO_FLAGS=-ldflags "-X $(REPO)/martian/util.__VERSION__='$(VERSION)' -X $(REPO)/martian/util.__RELEASE__='$(RELEASE)'"

export GOPATH=$(shell pwd)
export GO111MODULE=off

.PHONY: $(GOBINS) grammar web $(GOTESTS) govet all-bins bin/sum_squares longtests mrs

#
# Targets for development builds.
#
all: grammar all-bins web test mrs

bin/goyacc: vendor/golang.org/x/tools/cmd/goyacc/yacc.go
	go install vendor/golang.org/x/tools/cmd/goyacc

martian/syntax/grammar.go: bin/goyacc martian/syntax/grammar.y
	PATH=$(GOPATH)/bin:$(PATH) go generate $(REPO)/martian/syntax

martian/test/sum_squares/types.go: PATH:=$(GOPATH)/bin:$(PATH)
martian/test/sum_squares/types.go: test/split_test_go/pipeline_stages.mro mro2go
	go generate $(REPO)/martian/test/sum_squares

bin/sum_squares: martian/test/sum_squares/sum_squares.go \
                 martian/test/sum_squares/types.go
	go install $(GO_FLAGS) $(REPO)/martian/test/sum_squares

grammar: martian/syntax/grammar.go

$(GOBINS):
	go install $(GO_FLAGS) $(REPO)/cmd/$@

mrs: bin/mrs

bin/mrs: mrp
	rm -f bin/mrs && ln -s mrp bin/mrs


all-bins:
	go install $(GO_FLAGS) $(addprefix $(REPO)/, $(wildcard cmd/*))

NPM_CMD=install
ifeq ($(CI),true)
	NPM_CMD=ci
endif

web:
	(cd web/martian && npm $(NPM_CMD) && node_modules/gulp/bin/gulp.js)

mrt:
	cp scripts/mrt bin/mrt

$(GOLIBTESTS): test-%:
	go test -v $(REPO)/martian/$*

$(GOBINTESTS): test-%:
	go test -v $(REPO)/cmd/$*

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
	go test -v $(addprefix $(REPO)/, $(wildcard martian/* cmd/*))

govet:
	go tool vet martian

test: test-all govet bin/sum_squares

test/split_test/pipeline_test: mrp mrjob $(ADAPTERS)
	test/martian_test.py test/split_test/split_test.json

test/split_test_go/pipeline_test: mrp mrjob $(ADAPTERS) bin/sum_squares
	test/martian_test.py test/split_test_go/split_test.json

test/split_test_go/disable_pipeline_test: mrp mrjob $(ADAPTERS) bin/sum_squares
	test/martian_test.py test/split_test_go/disable_test.json

test/files_test/pipeline_test: mrp mrjob $(ADAPTERS)
	test/martian_test.py test/files_test/files_test.json

test/fork_test/pipeline_fail: test/fork_test/pipeline_test
	test/martian_test.py test/fork_test/fail1_test.json
	test/martian_test.py test/fork_test/autoretry_fail.json

test/fork_test/pipeline_test: mrp mrjob $(ADAPTERS)
	test/martian_test.py test/fork_test/fork_test.json
	test/martian_test.py test/fork_test/retry_test.json
	test/martian_test.py test/fork_test/autoretry_pass.json

longtests: test/split_test/pipeline_test \
           test/split_test_go/pipeline_test \
           test/split_test_go/disable_pipeline_test \
           test/files_test/pipeline_test \
           test/fork_test/pipeline_test \
           test/fork_test/pipeline_fail

clean:
	rm -rf $(GOPATH)/bin
	rm -rf $(GOPATH)/pkg
	rm -rf web/martian/node_modules
