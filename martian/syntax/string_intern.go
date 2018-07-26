// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Implements string interning for the parser.

package syntax

import "bytes"

type stringIntern struct {
	internSet map[string]string
}

// makeStringIntern creates a stringIntern object, prepopulated with string
// constants which are expected to be frequently used.  The use of such
// constants is desireable because they don't need to be allocated on the
// heap.
func makeStringIntern() (v *stringIntern) {
	v = &stringIntern{
		internSet: make(map[string]string, 64),
	}
	v.internSet[local] = strict
	v.internSet[preflight] = strict
	v.internSet[volatile] = strict
	v.internSet[disabled] = strict
	v.internSet[strict] = strict
	v.internSet[default_out_name] = default_out_name
	v.internSet[abr_python] = abr_python
	v.internSet[abr_exec] = abr_exec
	v.internSet[abr_compiled] = abr_compiled
	v.internSet[string(KindMap)] = string(KindMap)
	v.internSet[string(KindFloat)] = string(KindFloat)
	v.internSet[string(KindInt)] = string(KindInt)
	v.internSet[string(KindString)] = string(KindString)
	v.internSet[string(KindBool)] = string(KindBool)
	v.internSet[string(KindNull)] = string(KindNull)
	v.internSet[string(KindFile)] = string(KindFile)
	v.internSet[string(KindPath)] = string(KindPath)
	return v
}

func (store *stringIntern) GetString(value string) string {
	if len(value) == 0 {
		return ""
	}
	if s, ok := store.internSet[value]; ok {
		return s
	} else {
		store.internSet[value] = value
		return value
	}
}

func (store *stringIntern) Get(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	// The compiler special-cases string([]byte) used as a map key.
	// See golang issue #3512
	if s, ok := store.internSet[string(value)]; ok {
		return s
	} else {
		s = string(value)
		store.internSet[s] = s
		return s
	}
}

var quoteBytes = []byte(`"`)

func (store *stringIntern) unquote(value []byte) string {
	return store.Get(bytes.Replace(value, quoteBytes, nil, -1))
}

func unquote(qs []byte) string {
	return string(bytes.Replace(qs, quoteBytes, nil, -1))
}
