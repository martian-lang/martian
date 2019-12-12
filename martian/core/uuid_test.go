package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestBytes(t *testing.T) {
	u := UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

	bytes1 := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

	if !bytes.Equal(u.Bytes(), bytes1) {
		t.Errorf("Incorrect bytes representation for UUID: %s", u)
	}
}

func TestUnmarshalText(t *testing.T) {
	check := func(t *testing.T, val string) {
		var u UUID
		t.Helper()
		if err := u.UnmarshalText([]byte(val)); err != nil {
			t.Errorf("could not unmarshal %q: %v", val, err)
		} else if b, err := u.MarshalText(); err != nil {
			t.Errorf("could not marshal %q: %v", val, err)
		} else if s := string(b); s != strings.Trim(strings.TrimPrefix(val, "urn:uuid:"), "{}") {
			t.Errorf("%q != %q", s, val)
		} else if s2 := u.String(); s2 != s {
			t.Errorf("%q != %q", s2, s)
		}
	}
	check(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	check(t, "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}")
	check(t, "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	for _, invalid := range []string{
		// Not hex.
		"6bz7b810-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b810-9zad-11d1-80b4-00c04fd430c8",
		"6ba7b810-9dad-11z1-80b4-00c04fd430c8",
		"6ba7b810-9dad-11d1-80z4-00c04fd430c8",
		"6ba7b810-9dad-11d1-80b4-00cz4fd430c8",
		// too short
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		// too long
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8=",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}f",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c800c04fd430c8",
		// No dashes
		"6ba7b8109dad11d180b400c04fd430c8",
		"6ba7b8109dad11d180b400c04fd430c86ba7b8109dad11d180b400c04fd430c8",
		// urn and braces
		"urn:uuid:{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		// Misplaced or incorrect dashes
		"6ba7b8109-dad-11d1-80b4-00c04fd430c8",
		"6ba7b810-9dad1-1d1-80b4-00c04fd430c8",
		"6ba7b810-9dad-11d18-0b4-00c04fd430c8",
		"6ba7b810-9dad-11d1-80b40-0c04fd430c8",
		"6ba7b810+9dad+11d1+80b4+00c04fd430c8",
		"6ba7b810-9dad11d180b400c04fd430c8",
		"6ba7b8109dad-11d180b400c04fd430c8",
		"6ba7b8109dad11d1-80b400c04fd430c8",
		"6ba7b8109dad11d180b4-00c04fd430c8",
	} {
		var u UUID
		if err := u.UnmarshalText([]byte(invalid)); err == nil {
			t.Error("expected invalid: ", invalid)
		}
	}

}

func TestScanBinary(t *testing.T) {
	u := UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	b1 := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

	u1 := UUID{}
	err := u1.Scan(b1)
	if err != nil {
		t.Errorf("Error unmarshaling UUID: %s", err)
	}

	if !bytes.Equal(u[:], u1[:]) {
		t.Errorf("UUIDs should be equal: %s and %s", u, u1)
	}

	b2 := []byte{}
	u2 := UUID{}

	err = u2.Scan(b2)
	if err == nil {
		t.Errorf("Should return error unmarshalling from empty byte slice, got %s", err)
	}
}

func TestScanString(t *testing.T) {
	u := UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	s1 := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	u1 := UUID{}
	err := u1.Scan(s1)
	if err != nil {
		t.Errorf("Error unmarshaling UUID: %s", err)
	}

	if !bytes.Equal(u[:], u1[:]) {
		t.Errorf("UUIDs should be equal: %s and %s", u, u1)
	}

	s2 := ""
	u2 := UUID{}

	err = u2.Scan(s2)
	if err == nil {
		t.Errorf("Should return error trying to unmarshal from empty string")
	}
}

func TestScanText(t *testing.T) {
	u := UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	b1 := []byte("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	u1 := UUID{}
	err := u1.Scan(b1)
	if err != nil {
		t.Errorf("Error unmarshaling UUID: %s", err)
	}

	if !bytes.Equal(u[:], u1[:]) {
		t.Errorf("UUIDs should be equal: %s and %s", u, u1)
	}

	b2 := []byte("")
	u2 := UUID{}

	err = u2.Scan(b2)
	if err == nil {
		t.Errorf("Should return error trying to unmarshal from empty string")
	}
}

func TestScanUnsupported(t *testing.T) {
	u := UUID{}

	err := u.Scan(true)
	if err == nil {
		t.Errorf("Should return error trying to unmarshal from bool")
	}
}
