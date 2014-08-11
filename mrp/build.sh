#!/bin/bash

go tool yacc -p "mm" -o ../core/grammar.go ../core/grammar.y && rm y.output && go build
