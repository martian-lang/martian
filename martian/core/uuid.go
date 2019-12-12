// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// Based on the API presented by github.com/satori/go.uuid, however only
// implements creation of v4 uuids, and does not bring in as many
// dependencies.

// Background note: for several martian tools including mrjob, satori/uuid
// was the only thing causing the standard library net package to be linked
// into the binary.  That was adding 1.8MB to the binary size, which we don't
// need.

package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// UUID representation compliant with specification
// described in RFC 4122.
type UUID [16]byte

func safeRandom(dest []byte) {
	if _, err := rand.Read(dest); err != nil {
		panic(err)
	}
}

// NewV4 returns random generated UUID.
func NewUUID() UUID {
	u := UUID{}
	safeRandom(u[:])
	u[6] = (u[6] & 0x0f) | (4 << 4)
	u[8] = (u[8] & 0xbf) | 0x80

	return u
}

// Bytes returns bytes slice representation of UUID.
func (u UUID) Bytes() []byte {
	return u[:]
}

// Returns canonical string representation of UUID:
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (u UUID) String() string {
	if b, err := u.MarshalText(); err != nil {
		panic(err)
	} else {
		return string(b)
	}
}

// MarshalText implements the encoding.TextMarshaler interface.
// The encoding is the same as returned by String.
func (u UUID) MarshalText() (text []byte, err error) {
	buf := make([]byte, 36)

	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])
	return buf, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// Following formats are supported:
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
// "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
// "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
func (u *UUID) UnmarshalText(text []byte) error {
	if len(text) < 32 {
		return fmt.Errorf("uuid string too short: %q", string(text))
	}

	t := text[:]
	if string(t[:9]) == "urn:uuid:" {
		t = t[9:]
	} else if t[0] == '{' {
		t = t[1:]
		if t[len(t)-1] != '}' {
			return fmt.Errorf("uuid %q: missing closing brace",
				string(text))
		}
		t = t[:len(t)-1]
	}
	if len(t) != 8+1+4+1+4+1+4+1+12 {
		return fmt.Errorf("uuid string too long: %q", text)
	}

	if t[8] != '-' || t[8+1+4] != '-' || t[8+4+4+2] != '-' || t[8+4+4+4+3] != '-' {
		return fmt.Errorf(
			"invalid uuid string format %q: expected dashes were %q",
			string(text), []byte{t[8], t[8+1+4], t[8+4+4+2], t[8+4+4+4+3]})
	}

	if _, err := hex.Decode(u[:4], t[:8]); err != nil {
		return err
	}
	if _, err := hex.Decode(u[4:6], t[9:9+4]); err != nil {
		return err
	}
	if _, err := hex.Decode(u[6:8], t[14:14+4]); err != nil {
		return err
	}
	if _, err := hex.Decode(u[8:10], t[19:19+4]); err != nil {
		return err
	}
	if _, err := hex.Decode(u[10:], t[24:24+12]); err != nil {
		return err
	}
	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (u UUID) MarshalBinary() ([]byte, error) {
	return u[:], nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
// It will return error if the slice isn't 16 bytes long.
func (u *UUID) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("uuid must be exactly 16 bytes long, got %d bytes", len(data))
	}
	copy(u[:], data)
	return nil
}

// Scan implements the sql.Scanner interface.
// A 16-byte slice is handled by UnmarshalBinary, while
// a longer byte slice or a string is handled by UnmarshalText.
func (u *UUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		if len(src) == 16 {
			return u.UnmarshalBinary(src)
		}
		return u.UnmarshalText(src)

	case string:
		return u.UnmarshalText([]byte(src))
	}

	return fmt.Errorf("cannot convert %T to UUID",
		src)
}
