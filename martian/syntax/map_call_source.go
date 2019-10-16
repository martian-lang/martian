// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// check that map/array dimensions are consistent for all calls in the AST.

package syntax

import (
	"fmt"
	"strings"
)

type CallMode int

const (
	// This call invokes the callable once
	ModeSingleCall = CallMode(iota)
	// This call invokes the callable on an array of inputs, so the
	// result is an array of outputs.
	ModeArrayCall
	// This call invokes the callable on a map of inputs, so the result is
	// a map of outputs.
	ModeMapCall
	// This call invokes the callable on a set of inputs, but it is not known
	// yet what kind of inputs those are because the call has not yet been
	// compiled.
	ModeUnknownMapCall
	// This call invokes the callableon an expression which evaluates to null,
	// which could be either an array or a map but either way has no elements.
	ModeNullMapCall
)

func (m CallMode) MarshalText() ([]byte, error) {
	switch m {
	case ModeSingleCall:
		return nil, nil
	case ModeArrayCall:
		return []byte(KindArray), nil
	case ModeMapCall:
		return []byte(KindMap), nil
	case ModeUnknownMapCall:
		return []byte("unknown"), nil
	}
	return nil, fmt.Errorf("invalid map call mode %d", m)
}

func (m *CallMode) UnmarshalText(b []byte) error {
	if len(b) == 0 || isNullBytes(b) {
		*m = 0
		return nil
	}
	switch ExpKind(b) {
	case KindArray:
		*m = ModeArrayCall
	case KindMap:
		*m = ModeMapCall
	case "unknown":
		*m = ModeUnknownMapCall
	default:
		return fmt.Errorf("invalid map call mode %s", string(b))
	}
	return nil
}

func (m CallMode) String() string {
	switch m {
	case ModeSingleCall:
		return "simple"
	case ModeArrayCall:
		return string(KindArray)
	case ModeMapCall:
		return string(KindMap)
	case ModeUnknownMapCall:
		return "unknown"
	}
	return "invalid"
}

// A MapCallSource determines the type and dimension of a map call.
type MapCallSource interface {
	// CallMode Returns the call mode for a call which depends on this source.
	CallMode() CallMode

	// KnownLength returns true if the source is an array with a known length
	// or is a map with a known set of keys.
	KnownLength() bool

	// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
	// the length of the array referred to by this source.  Otherwise it will
	// return -1.
	ArrayLength() int

	// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
	// a map[string]Exp with the same keys which any call mapping over this
	// source would have.  The values are arbitrary.  Otherwise, it will return
	// nil.
	Keys() map[string]Exp

	GoString() string
}

// placeholderMapSource is added to map calls by the parser so that they can be
// identified easily, but does not actually provide mapping information.  The
// compiler will replace it with a real source.
type placeholderMapSource struct{}

var mapSourcePlaceholder placeholderMapSource

// CallMode Returns the call mode for a call which depends on this source.
func (*placeholderMapSource) CallMode() CallMode {
	return ModeUnknownMapCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (*placeholderMapSource) KnownLength() bool {
	return false
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (*placeholderMapSource) ArrayLength() int {
	return -1
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (*placeholderMapSource) Keys() map[string]Exp {
	return nil
}

func (*placeholderMapSource) GoString() string {
	return "placeholder"
}

// placeholderArrayMapSource is used as a map source when the array length
// is not known.
type placeholderArrayMapSource struct {
	placeholderMapSource
}

// CallMode Returns the call mode for a call which depends on this source.
func (*placeholderArrayMapSource) CallMode() CallMode {
	return ModeArrayCall
}

// placeholderMapMapSource is used as a map source when the keys are
// not known.
type placeholderMapMapSource struct {
	placeholderMapSource
}

// CallMode Returns the call mode for a call which depends on this source.
func (*placeholderMapMapSource) CallMode() CallMode {
	return ModeMapCall
}

// CallMode Returns the call mode for a call which depends on this source.
func (*NullExp) CallMode() CallMode {
	return ModeNullMapCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (*NullExp) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (*NullExp) ArrayLength() int {
	return 0
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (*NullExp) Keys() map[string]Exp {
	return nil
}

var emptyMap MapExp
var emptyArray ArrayExp

// MapCallSet contains a set of MapCallSources which must share the same array
// length or key set.  If a map call splits on multiple input parameters, then
// the corresponding sources, as well as the call itself, all belong to the same
// set.  information about collections that a map call maps
// over.  These must either all be maps with the same set of keys, or they
// must be arrays of the same length.
//
// In many cases, the array length or map keys are not known at static analysis
// time.  However, if one input in the set is known then the rest must have
// the same keys or length.
type MapCallSet struct {
	// The Master is either an arbitrarily chosen expression with known array
	// length or map keys, or a stage output parameter if none in the set are
	// known.
	Master MapCallSource `json:"primary_source"`

	// The set of all sources in the call set.
	Sources map[MapCallSource]struct{} `json:"-"`
}

// CallMode Returns the call mode for a call which depends on this source.
func (m *MapCallSet) CallMode() CallMode {
	if m == nil || m.Master == nil {
		return ModeSingleCall
	}
	return m.Master.CallMode()
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (m *MapCallSet) KnownLength() bool {
	if m == nil || m.Master == nil {
		return false
	}
	return m.Master.KnownLength()
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (m *MapCallSet) ArrayLength() int {
	if m == nil || m.Master == nil {
		return -1
	}
	return m.Master.ArrayLength()
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (m *MapCallSet) Keys() map[string]Exp {
	if m == nil || m.Master == nil {
		return nil
	}
	return m.Master.Keys()
}

func (m *MapCallSet) GoString() string {
	return "set " + m.Master.GoString() + fmt.Sprintf("%p", m.Master)
}

func MergeMapCallSources(a, b MapCallSource) (MapCallSource, error) {
	// Handle trivial cases
	if a == b {
		return a, nil
	}
	switch a := a.(type) {
	case *CallStm:
		src, err := MergeMapCallSources(a.Mapping, b)
		if err != nil {
			a.Mapping = src
		}
		return src, err
	case *RefExp:
		if len(a.MergeOver) > 0 {
			src, err := MergeMapCallSources(a.MergeOver[0], b)
			if err != nil || src == a.MergeOver[0] {
				return src, err
			} else if src != a {
				switch a.MergeOver[0].(type) {
				case *placeholderArrayMapSource, *placeholderMapMapSource, *placeholderMapSource:
					mo := make([]MapCallSource, len(a.MergeOver))
					mo[0] = src
					copy(mo[1:], a.MergeOver[1:])
					a.MergeOver = mo
				}
			}
			if src == b {
				return b, nil
			}
		}
	}
	switch b := b.(type) {
	case *CallStm:
		src, err := MergeMapCallSources(a, b.Mapping)
		if err != nil {
			b.Mapping = src
		}
		return src, err
	case *RefExp:
		if len(b.MergeOver) > 0 {
			src, err := MergeMapCallSources(a, b.MergeOver[0])
			if err != nil || src == b.MergeOver[0] {
				return src, err
			} else if src != b {
				switch b.MergeOver[0].(type) {
				case *placeholderArrayMapSource, *placeholderMapMapSource, *placeholderMapSource:
					mo := make([]MapCallSource, len(b.MergeOver))
					mo[0] = src
					copy(mo[1:], b.MergeOver[1:])
					b.MergeOver = mo
				}
			}
			if src == a {
				return a, nil
			}
		}
	}
	if b == nil || b.CallMode() == ModeSingleCall || b.CallMode() == ModeUnknownMapCall {
		return a, nil
	}
	if a == nil || a.CallMode() == ModeSingleCall || a.CallMode() == ModeUnknownMapCall {
		return b, nil
	}
	// Check consistency
	if am, bm := a.CallMode(), b.CallMode(); am != bm && am != ModeNullMapCall && bm != ModeNullMapCall {
		return nil, fmt.Errorf("cannot split over both arrays and maps")
	} else if bm == ModeNullMapCall {
		return a, nil
	} else if am == ModeNullMapCall {
		return b, nil
	}
	switch b.(type) {
	case *placeholderArrayMapSource, *placeholderMapMapSource, *placeholderMapSource:
		return a, nil
	}
	switch a.(type) {
	case *placeholderArrayMapSource, *placeholderMapMapSource, *placeholderMapSource:
		return b, nil
	}
	if a.KnownLength() && b.KnownLength() {
		switch a.CallMode() {
		case ModeArrayCall:
			if la, lb := a.ArrayLength(), b.ArrayLength(); la != lb {
				return nil, fmt.Errorf("array length mismatch %d vs %d",
					la, lb)
			}
		case ModeMapCall:
			ka, kb := a.Keys(), b.Keys()
			if len(ka) != len(kb) {
				return nil, fmt.Errorf("map length mismatch %d vs %d",
					len(ka), len(kb))
			}
			for k := range ka {
				if _, ok := kb[k]; !ok {
					return nil, fmt.Errorf("map key missing %q", k)
				}
			}
		case ModeNullMapCall:
			switch b.CallMode() {
			case ModeArrayCall:
				if b.ArrayLength() != 0 {
					return nil, fmt.Errorf("array was not empty")
				}
			case ModeMapCall:
				if len(b.Keys()) != 0 {
					return nil, fmt.Errorf("map was not empty")
				}
			}
		}
	}
	if as, ok := a.(*MapCallSet); ok {
		switch bs := b.(type) {
		case *MapCallSet:
			// Both are sets, merge them.
			if as.KnownLength() || !bs.KnownLength() {
				for s := range bs.Sources {
					as.Sources[s] = struct{}{}
				}
				return as, nil
			} else {
				for s := range as.Sources {
					bs.Sources[s] = struct{}{}
				}
				return bs, nil
			}
		default:
			as.Sources[b] = struct{}{}
			if !as.KnownLength() && b.KnownLength() {
				if b.CallMode() == ModeNullMapCall {
					if m := as.CallMode(); m == ModeArrayCall {
						as.Master = &emptyArray
					} else if m == ModeMapCall {
						as.Master = &emptyMap
					} else {
						as.Master = b
					}
				} else {
					as.Master = b
				}
			}
			return as, nil
		}
	} else if bs, ok := b.(*MapCallSet); ok {
		bs.Sources[a] = struct{}{}
		if !bs.KnownLength() && a.KnownLength() {
			if a.CallMode() == ModeNullMapCall {
				if m := bs.CallMode(); m == ModeArrayCall {
					bs.Master = &emptyArray
				} else if m == ModeMapCall {
					bs.Master = &emptyMap
				} else {
					bs.Master = a
				}
			} else {
				bs.Master = a
			}
		}
		return bs, nil
	} else {
		set := MapCallSet{
			Sources: map[MapCallSource]struct{}{
				a: {},
				b: {},
			},
			Master: a,
		}
		if a.CallMode() == ModeNullMapCall {
			switch b.CallMode() {
			case ModeArrayCall:
				set.Master = &emptyArray
			case ModeMapCall:
				set.Master = &emptyMap
			}
		}
		if !set.KnownLength() {
			if b.KnownLength() {
				if b.CallMode() == ModeNullMapCall {
					switch a.CallMode() {
					case ModeArrayCall:
						set.Master = &emptyArray
					case ModeMapCall:
						set.Master = &emptyMap
					}
				} else {
					set.Master = b
				}
			} else if ra, ok := set.Master.(*RefExp); ok {
				// Prefer fully-qualified refs to those which aren't.
				if !strings.ContainsRune(ra.Id, '.') {
					if rb, ok := b.(*RefExp); ok {
						if strings.ContainsRune(rb.Id, '.') {
							set.Master = b
						}
					}
				}
			} else if _, ok := b.(*RefExp); ok {
				// Prefer ref over not ref
				set.Master = b
			}
		} else if b.KnownLength() {
			if _, ok := set.Master.(*RefExp); !ok {
				if _, ok := b.(*RefExp); ok {
					// Prefer ref over not ref
					set.Master = b
				}
			}
		}
		return &set, nil
	}
}

// CallMode Returns the call mode for a call which depends on this source.
func (c *CallStm) CallMode() CallMode {
	m := c.Mapping
	if m == nil {
		return ModeSingleCall
	}
	return m.CallMode()
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (c *CallStm) KnownLength() bool {
	m := c.Mapping
	if m == nil {
		return false
	}
	return m.KnownLength()
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (c *CallStm) ArrayLength() int {
	m := c.Mapping
	if m == nil {
		return -1
	}
	return m.ArrayLength()
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (c *CallStm) Keys() map[string]Exp {
	m := c.Mapping
	if m == nil {
		return nil
	}
	return m.Keys()
}

// CallMode Returns the call mode for a call which depends on this source.
func (a *ArrayExp) CallMode() CallMode {
	return ModeArrayCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (a *ArrayExp) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (a *ArrayExp) ArrayLength() int {
	if a == nil {
		return 0
	}
	return len(a.Value)
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (a *ArrayExp) Keys() map[string]Exp {
	return nil
}

// CallMode Returns the call mode for a call which depends on this source.
func (a *SweepExp) CallMode() CallMode {
	return ModeArrayCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (a *SweepExp) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (a *SweepExp) ArrayLength() int {
	if a == nil {
		return 0
	}
	return len(a.Value)
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (a *SweepExp) Keys() map[string]Exp {
	return nil
}

// CallMode Returns the call mode for a call which depends on this source.
func (a *MapExp) CallMode() CallMode {
	return ModeMapCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (a *MapExp) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (a *MapExp) ArrayLength() int {
	return -1
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (a *MapExp) Keys() map[string]Exp {
	if a == nil {
		return make(map[string]Exp)
	}
	return a.Value
}

type InconsistentMapCallError struct {
	Pipeline string
	Call     *CallStm
	Inner    error
}

func (err *InconsistentMapCallError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Inner
}

func (err *InconsistentMapCallError) writeTo(w stringWriter) {
	if err.Pipeline == "" {
		mustWriteString(w, "inconsistent split inputs in top-level call to ")
		mustWriteString(w, err.Call.DecId)
	} else {
		mustWriteString(w, "inconsistent split inputs in call to ")
		mustWriteString(w, err.Call.DecId)
		if err.Call.DecId != err.Call.Id {
			mustWriteString(w, " as ")
			mustWriteString(w, err.Call.Id)
		}
		mustWriteString(w, " in pipeline ")
		mustWriteString(w, err.Pipeline)
	}
	mustWriteString(w, "\n    at ")
	err.Call.Node.Loc.writeTo(w, "        ")
	if err.Inner != nil {
		mustWriteString(w, "\nCause: ")
		if ew, ok := err.Inner.(errorWriter); ok {
			ew.writeTo(w)
		} else {
			mustWriteString(w, err.Inner.Error())
		}
	}
}

func (err *InconsistentMapCallError) Error() string {
	var buff strings.Builder
	buff.Grow(150 + len(err.Pipeline) + len(err.Call.Id) + len(err.Call.DecId))
	err.writeTo(&buff)
	return buff.String()
}
