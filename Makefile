#
# Copyright (c) 2014-2017 10X Genomics, Inc. All rights reserved.
#
# Build a Go package with git version embedding.
#

PWD=$(shell pwd)
GOBINS=$(notdir $(wildcard cmd/m*))
REPO=github.com/martian-lang/martian
GOLIBTESTS=$(addprefix test-, $(notdir $(wildcard martian/*)))
GOBINTESTS=$(addprefix test-, $(GOBINS))
GOTESTS=$(GOLIBTESTS) $(GOBINTESTS) test-all
VERSION=$(shell git describe --tags --always --dirty)
RELEASE=false
SRC_ROOT=$(abspath $(dir $(PWD))../..)
GO_FLAGS=-ldflags "-X $(REPO)/martian/util.__VERSION__='$(VERSION)' -X $(REPO)/martian/util.__RELEASE__='$(RELEASE)'" -gcflags "-trimpath $(SRC_ROOT)" -mod=vendor

unexport GOPATH
export GO111MODULE=on
export GOBIN=$(PWD)/bin

.PHONY: $(GOBINS) grammar web $(GOTESTS) govet all-bins $(GOBIN)/sum_squares longtests mrs integration_prereqs

#
# Targets for development builds.
#
all: grammar all-bins web test mrs

$(GOBIN)/goyacc: vendor/golang.org/x/tools/cmd/goyacc/yacc.go
	go install -mod=vendor golang.org/x/tools/cmd/goyacc

martian/syntax/grammar.go: $(GOBIN)/goyacc martian/syntax/grammar.y
	PATH=$(GOBIN):$(PATH) go generate ./martian/syntax

martian/test/sum_squares/types.go: PATH:=$(GOBIN):$(PATH)
martian/test/sum_squares/types.go: test/split_test_go/pipeline_stages.mro mro2go
	go generate ./martian/test/sum_squares

$(GOBIN)/sum_squares: martian/test/sum_squares/sum_squares.go \
                      martian/test/sum_squares/types.go
	go install $(GO_FLAGS) ./martian/test/sum_squares

grammar: martian/syntax/grammar.go

$(GOBINS):
	go install $(GO_FLAGS) ./cmd/$@

mrs: $(GOBIN)/mrs

$(GOBIN)/mrs: mrp
	rm -f $(GOBIN)/mrs && ln -s mrp $(GOBIN)/mrs


all-bins:
	go install $(GO_FLAGS) ./cmd/...

NPM_CMD=install
ifeq ($(CI),true)
	NPM_CMD=ci
endif

web:
	(cd web/martian && npm $(NPM_CMD) && node_modules/gulp/bin/gulp.js)

mrt:
	cp scripts/mrt $(GOBIN)/mrt

$(GOLIBTESTS): test-%:
	go test -v ./martian/$*

$(GOBINTESTS): test-%:
	go test -v ./cmd/$*

WEB_FILES=web/martian/serve web/martian/templates/graph.html

$(WEB_FILES): web

ADAPTERS=$(wildcard adapters/python/*.py) $(wildcard adapters/python/*/*.py)
JOBMANAGERS=$(wildcard jobmanagers/*.py) \
			$(wildcard jobmanagers/*.json) \
			$(wildcard jobmanagers/*.template.example)

PRODUCT_NAME:=martian-$(VERSION)-$(shell uname -is | tr "A-Z " "a-z-")

$(PRODUCT_NAME).tar.%: $(addprefix bin/, $(GOBINS)) $(ADAPTERS) $(JOBMANAGERS) $(WEB_FILES)
	tar --owner=0 --group=0 --transform "s/^./$(PRODUCT_NAME)/" -caf $@ $(addprefix ./, $^)

tarball: $(PRODUCT_NAME).tar.xz $(PRODUCT_NAME).tar.gz

test-all:
	go test -v ./martian/... ./cmd/...

govet:
	go vet ./martian/... ./cmd/...

test: test-all govet $(GOBIN)/sum_squares

integration_prereqs: mrp mrjob $(ADAPTERS) test/martian_test.py $(JOBMANAGERS)

test/split_test/pipeline_test: test/split_test/split_test.json \
                               integration_prereqs
	test/martian_test.py $<

test/split_test_go/pipeline_test: test/split_test_go/split_test.json \
                                  integration_prereqs $(GOBIN)/sum_squares
	test/martian_test.py $<

test/split_test_go/disable_pipeline_test: test/split_test_go/disable_test.json \
                                          integration_prereqs $(GOBIN)/sum_squares
	test/martian_test.py $<

test/exit_test/pipeline_test: test/exit_test/exit_test.json \
                              integration_prereqs
	test/martian_test.py $<

test/files_test/pipeline_test: test/files_test/files_test.json \
                               integration_prereqs
	test/martian_test.py $<

test/retain_test/pipeline_test: test/retain_test/retain_test.json \
                                integration_prereqs
	test/martian_test.py $<

test/fork_test/fail/pipeline_fail: test/fork_test/fail1_test.json \
                                   integration_prereqs
	test/martian_test.py $<

test/fork_test/ar_fail/pipeline_fail: test/fork_test/autoretry_fail.json \
                                      integration_prereqs
	test/martian_test.py $<

test/fork_test/pass/pipeline_test: test/fork_test/fork_test.json \
                                   integration_prereqs
	test/martian_test.py $<

test/fork_test/retry/pipeline_test: test/fork_test/retry_test.json \
                                    integration_prereqs
	test/martian_test.py $<

test/fork_test/ar_pass/pipeline_test: test/fork_test/autoretry_pass.json \
                                      integration_prereqs
	test/martian_test.py $<

longtests: test/split_test/pipeline_test \
           test/split_test_go/pipeline_test \
           test/split_test_go/disable_pipeline_test \
           test/exit_test/pipeline_test \
           test/files_test/pipeline_test \
           test/retain_test/pipeline_test \
           test/fork_test/pass/pipeline_test \
           test/fork_test/retry/pipeline_test \
           test/fork_test/ar_pass/pipeline_test \
           test/fork_test/fail/pipeline_fail \
           test/fork_test/ar_fail/pipeline_fail

clean:
	rm -rf $(GOBIN)
	rm -rf $(dir $(GOBIN))pkg
	rm -rf web/martian/node_modules
