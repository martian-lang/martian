// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"sort"

	"github.com/martian-lang/martian/martian/syntax"
)

func FindUnusedCallables(topCalls StringSet, asts []*syntax.Ast) []syntax.Callable {
	unused := make(map[decId]syntax.Callable)
	used := make(map[decId]*syntax.Pipeline, len(topCalls))
	for _, ast := range asts {
		for _, callable := range ast.Callables.List {
			if topCalls.Contains(callable.GetId()) {
				if p, ok := callable.(*syntax.Pipeline); ok && p != nil {
					used[makeDecId(p)] = p
				}
			} else if !HasKeepComment(callable) {
				unused[makeDecId(callable)] = callable
			}
		}
	}
	for len(used) > 0 && len(unused) > 0 {
		newUsed := make(map[decId]*syntax.Pipeline, len(unused))
		for _, pipe := range used {
			for _, callable := range pipe.Callables.Table {
				id := makeDecId(callable)
				if p, ok := callable.(*syntax.Pipeline); ok && p != nil {
					newUsed[id] = p
				}
				delete(unused, id)
			}
		}
		used = newUsed
	}
	if len(unused) == 0 {
		return nil
	}
	result := make([]syntax.Callable, 0, len(unused))
	for _, c := range unused {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool {
		ri, rj := result[i], result[j]
		if f, g := syntax.DefiningFile(ri), syntax.DefiningFile(rj); f < g {
			return true
		} else if f > g {
			return false
		}
		return ri.Line() < rj.Line()
	})
	return result
}
