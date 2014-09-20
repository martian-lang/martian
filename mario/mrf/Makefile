#
# Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
#
# Build a Go package with git version embedding.
#

VERSION = $(shell git describe --tags --always --dirty)

all: main

main:
	go build -ldflags "-X main.__VERSION__ $(VERSION)"