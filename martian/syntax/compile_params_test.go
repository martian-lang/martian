package syntax

import "testing"

func TestCheckLegalFileName(t *testing.T) {
	legal := func(s string) {
		t.Helper()
		if err := checkLegalFilename(s); err != nil {
			t.Errorf("expected %q to be legal: %v", s, err)
		}
	}
	illegal := func(s string) {
		t.Helper()
		if err := checkLegalFilename(s); err == nil {
			t.Errorf("expected %q to be illegal", s)
		}
	}
	legal("abc.d")
	legal(".foo")
	legal("abc%2F")
	illegal("abc.")
	illegal("abc ")
	illegal("a\x00b")
	illegal("a\bc")
	illegal("abc\f")
	illegal("ab\nc")
	illegal("ab\rc")
	illegal("ab\tc")
	illegal("ab\vc")
	illegal("ab|c")
	illegal("ab\\c")
	illegal("ab/c")
	illegal("ab:c")

	illegal("aux")
	illegal("AUX")
	illegal("aux.c")
	legal("aux~.c")
	legal("aux1.c")
	legal("aux1")

	legal("lpt")
	legal("LPT")
	legal("lpt.foo")
	illegal("lpt1.foo")
	illegal("lpt2")
	illegal("LPT3")
	legal("lpt4~.foo")

	const thirtyTwo = "012345678901234567890123456789012"
	legal(thirtyTwo)
	illegal(thirtyTwo + thirtyTwo + thirtyTwo + thirtyTwo + thirtyTwo)
}
