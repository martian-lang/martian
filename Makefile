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
GO_FLAGS=-ldflags "-X '$(REPO)/martian/util.__VERSION__=$(VERSION)' -X $(REPO)/martian/util.__RELEASE__='$(RELEASE)'" -gcflags "-trimpath $(SRC_ROOT)"

unexport GOPATH
export GO111MODULE=on
export GOBIN=$(PWD)/bin

.PHONY: $(GOBINS) grammar web $(GOTESTS) coverage.out govet all-bins $(GOBIN)/sum_squares longtests integration_prereqs vscode vscode-test

#
# Targets for development builds.
#
all: grammar all-bins web test vscode

martian/syntax/grammar.go: martian/syntax/grammar.y martian/syntax/lexer.go
	go generate ./martian/syntax

martian/test/sum_squares/types.go: PATH:=$(GOBIN):$(PATH)
martian/test/sum_squares/types.go: test/split_test_go/pipeline_stages.mro mro2go
	go generate ./martian/test/sum_squares

$(GOBIN)/sum_squares: martian/test/sum_squares/sum_squares.go \
                      martian/test/sum_squares/types.go
	go install $(GO_FLAGS) ./martian/test/sum_squares

grammar: martian/syntax/grammar.go

$(GOBINS):
	go install $(GO_FLAGS) ./cmd/$@

all-bins:
	go install $(GO_FLAGS) ./cmd/...

NPM_CMD=install
ifeq ($(CI),true)
	NPM_CMD=ci
endif

web:
	(cd web/martian && npm $(NPM_CMD) --frozen-lockfile --no-optional && npm run-script build)

vscode:
	(cd tools/syntax/vscode && npm $(NPM_CMD) --frozen-lockfile --no-optional && npm run-script compile)

vscode-test: vscode
	(cd tools/syntax/vscode && npm run-script check-lint)

$(GOLIBTESTS): test-%:
	go test -v ./martian/$*

$(GOBINTESTS): test-%:
	go test -v ./cmd/$*

WEB_FILES=web/martian/serve

$(WEB_FILES): web

ADAPTERS=$(wildcard adapters/python/*.py) $(wildcard adapters/python/*/*.py)
JOBMANAGERS=$(wildcard jobmanagers/*.py) \
			$(wildcard jobmanagers/*.json) \
			$(wildcard jobmanagers/*.template.example)

$(addprefix bin/, $(GOBINS)): all-bins

PRODUCT_NAME:=martian-$(VERSION)-$(shell uname -is | tr "A-Z " "a-z-")

TARBALLS:=$(addprefix $(PRODUCT_NAME).tar, .gz .xz)

$(TARBALLS): $(addprefix bin/, $(GOBINS)) $(ADAPTERS) $(JOBMANAGERS) $(WEB_FILES)
	git status || echo "no git status"
	tar --owner=0 --group=0 --transform "s/^\\./$(PRODUCT_NAME)/" -caf $@ $(addprefix ./, $^)

tarball: $(TARBALLS)

test-all: martian/syntax/grammar.go | martian/test/sum_squares/types.go
	go test -race ./martian/... ./cmd/...

coverage.out: martian/syntax/grammar.go | martian/test/sum_squares/types.go
	go test -coverprofile=coverage.out \
	        -coverpkg=./martian/... \
	        ./martian/... ./cmd/...

coverage.html: coverage.out
	go tool cover -html=coverage.out -o coverage.html

cover: coverage.html

govet: martian/syntax/grammar.go | martian/test/sum_squares/types.go
	go vet ./martian/... ./cmd/...

test: test-all govet $(GOBIN)/sum_squares vscode-test

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

test/struct_test/pipeline_test: test/struct_test/struct_test.json \
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

test/map_test/pipeline_test: test/map_test/map_test.json \
                             integration_prereqs
	test/martian_test.py $<

test/disable_test/pipeline_test: test/disable_test/disable_test.json \
                                 integration_prereqs
	test/martian_test.py $<


test/retry_test/pipeline_test: test/retry_test/autoretry_pass.json \
                               integration_prereqs
	test/martian_test.py $<

longtests: test/split_test/pipeline_test \
           test/split_test_go/pipeline_test \
           test/split_test_go/disable_pipeline_test \
           test/exit_test/pipeline_test \
           test/files_test/pipeline_test \
           test/retain_test/pipeline_test \
           test/struct_test/pipeline_test \
           test/fork_test/pass/pipeline_test \
           test/fork_test/retry/pipeline_test \
           test/fork_test/ar_pass/pipeline_test \
           test/fork_test/fail/pipeline_fail \
           test/fork_test/ar_fail/pipeline_fail \
           test/map_test/pipeline_test \
		   test/disable_test/pipeline_test \
		   test/retry_test/pipeline_test

clean:
	rm -rf $(GOBIN)
	rm -rf $(dir $(GOBIN))pkg
	rm -rf web/martian/node_modules
	rm -rf web/martian/build
	rm -rf web/martian/serve
	make -C tools/syntax/vscode clean
