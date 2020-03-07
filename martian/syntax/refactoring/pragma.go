// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"regexp"

	"github.com/martian-lang/martian/martian/syntax"
)

var keepRegexp = regexp.MustCompile(`(?i:\bkeep\b)|(?i:\brequired\b)`)

func HasKeepComment(node syntax.AstNodable) bool {
	for _, c := range syntax.GetComments(node) {
		if keepRegexp.MatchString(c) {
			return true
		}
	}
	return false
}
