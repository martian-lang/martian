// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// Directives to generate grammar.go from grammar.yacc

// +build generate
//go:generate go run golang.org/x/tools/cmd/goyacc -l -p "mm" -o grammar.go grammar.y
//go:generate rm -f y.output
//go:generate gofmt -s -w grammar.go

package syntax

// Ensure go mod tidy doesn't remove tools.  Of course this is a binary, not
// an importable package; that's why this file is tagged +build generate
import _ "golang.org/x/tools/cmd/goyacc"
