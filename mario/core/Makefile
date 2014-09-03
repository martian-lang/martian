#
# Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
#
# Compiles grammar.y into grammar.go and cleans up. To simplify the build 
# process we run this during development time and check-in the generated 
# .go file so we do not have run go tool yacc at build time.
#

VERSION = $(shell git describe --tags --always --dirty)

all: grammar

grammar: grammar.y
	go tool yacc -p "mm" -o grammar.go grammar.y && rm y.output