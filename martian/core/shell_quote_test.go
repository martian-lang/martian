// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

import "testing"

func TestAppendShellSafeQuote(t *testing.T) {
	buf := make([]byte, 0, 48)
	check := func(src, expect string) {
		if s := string(appendShellSafeQuote(buf, src)); s != expect {
			t.Errorf("%s != %s", s, expect)
		}
	}
	check(`abc123`, `"abc123"`)
	check(`a"$\bc`, `"a\"\$\\bc"`)
	check("\a\b\n\t\r\v☺\277", `"\a\b\n\t\r\v☺\277"`)
}
