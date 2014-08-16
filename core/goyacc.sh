#!/bin/bash
#
# Compiles grammar.y into grammar.go and cleans up.
# To simplify the build process we run this during
# development time and check-in the generated .go
# file so we do not have run go tool yacc at build
# time.

go tool yacc -p "mm" -o grammar.go grammar.y && rm y.output