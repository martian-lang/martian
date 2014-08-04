#!/bin/bash

go tool yacc -p "mm" -o grammar.go grammar.y && go build && time ./margo
