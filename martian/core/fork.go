package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

type (
	// Represents a component of a fork ID, which
	ForkIdPart interface {
		fmt.Stringer
		syntax.CollectionIndex

		// The string ID for a fork.
		//
		// For a map-type fork, this will be "fork.<key>".  For an array-type
		// fork it will be "forkN" where N is the array index.
		forkString() string
	}

	mapKeyFork string

	arrayIndexFork int

	undeterminedFork struct{}

	ForkSourcePart struct {
		Source syntax.MapCallSource
		Id     ForkIdPart
	}

	// A set of expressions and the corresponding index values for each one,
	// which identifies a specific fork.
	ForkId []*ForkSourcePart

	ForkIdSet struct {
		List  []ForkId
		Table map[syntax.MapCallSource][]*ForkSourcePart
	}

	expSetBuilder struct {
		Exps []syntax.MapCallSource
		set  map[syntax.MapCallSource]struct{}
	}
)

func convertForkPart(f syntax.CollectionIndex) ForkIdPart {
	switch f := f.(type) {
	case mapKeyFork:
		return f
	case arrayIndexFork:
		return f
	case undeterminedFork:
		return f
	}
	if f.IndexSource() != nil {
		return undeterminedFork{}
	}
	switch f.Mode() {
	case syntax.ModeArrayCall:
		return arrayIndexFork(f.ArrayIndex())
	case syntax.ModeMapCall:
		return mapKeyFork(f.MapKey())
	}
	panic("invalid index")
}

func (s *expSetBuilder) Add(exp syntax.MapCallSource) {
	if s.Exps == nil {
		s.Exps = []syntax.MapCallSource{exp}
	} else if len(s.Exps) == 0 {
		s.Exps = append(s.Exps, exp)
	}
	if s.set == nil {
		s.set = make(map[syntax.MapCallSource]struct{}, 1)
		for _, e := range s.Exps {
			s.set[e] = struct{}{}
		}
	}
	if _, ok := s.set[exp]; !ok {
		s.set[exp] = struct{}{}
		s.Exps = append(s.Exps, exp)
	}
}

func (s *expSetBuilder) AddMany(exp []syntax.MapCallSource) {
	if len(exp) == 0 {
		return
	}
	if len(s.Exps) == 0 {
		s.Exps = exp
	}
	if s.set == nil {
		s.set = make(map[syntax.MapCallSource]struct{}, len(s.Exps)+len(exp))
		for _, e := range s.Exps {
			s.set[e] = struct{}{}
		}
	}
	for _, e := range exp {
		if _, ok := s.set[e]; !ok {
			s.set[e] = struct{}{}
			s.Exps = append(s.Exps, e)
		}
	}
}

// Adds the expression or subexpressions of the given expression which
// are fork roots.  This does not include reference expressions where the
// reference might be forked.
//
// At the moment this just means sweep expressions.
func (s *expSetBuilder) AddForkRoots(exp syntax.Exp) {
	if exp == nil {
		return
	}
	switch exp := exp.(type) {
	case *syntax.SweepExp:
		s.Add(exp)
	case *syntax.ArrayExp:
		for _, se := range exp.Value {
			s.AddForkRoots(se)
		}
	case *syntax.MapExp:
		for _, se := range exp.Value {
			s.AddForkRoots(se)
		}
	}
}

func (s *expSetBuilder) AddPrenodes(prenodes map[string]Nodable) {
	if len(prenodes) == 0 {
		return
	}
	keys := make([]string, 0, len(prenodes))
	for k, n := range prenodes {
		if n == nil {
			continue
		}
		if n := n.getNode(); n != nil && n.swept && len(n.forkRoots) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		s.AddMany(prenodes[k].getNode().forkRoots)
	}
}

func (s *expSetBuilder) AddBindings(bindings map[string]*syntax.ResolvedBinding) {
	if len(bindings) == 0 {
		return
	}
	keys := make([]string, 0, len(bindings))
	for k := range bindings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s.AddForkRoots(bindings[k].Exp)
	}
}

func (fork ForkId) SourceIndexMap() map[syntax.MapCallSource]syntax.CollectionIndex {
	if len(fork) == 0 {
		return nil
	}
	result := make(map[syntax.MapCallSource]syntax.CollectionIndex, len(fork))
	for _, part := range fork {
		if _, ok := part.Id.(*undeterminedFork); !ok {
			result[part.Source] = part.Id
		}
	}
	return result
}

func makeForkIdParts(src syntax.MapCallSource) []*ForkSourcePart {
	switch src.CallMode() {
	case syntax.ModeArrayCall:
		alen := src.ArrayLength()
		if alen >= 0 {
			re := make([]ForkSourcePart, src.ArrayLength())
			result := make([]*ForkSourcePart, src.ArrayLength())
			for i := range result {
				r := arrayIndexFork(i)
				re[i].Source = src
				re[i].Id = r
				result[i] = &re[i]
			}
			return result
		}
	case syntax.ModeMapCall:
		keys := src.Keys()
		re := make([]ForkSourcePart, len(keys))
		result := make([]*ForkSourcePart, len(keys))
		for k := range keys {
			r := mapKeyFork(k)
			re = append(re, ForkSourcePart{
				Source: src,
				Id:     r,
			})
			result = append(result, &re[len(re)-1])
		}
		sort.Slice(result, func(i, j int) bool {
			return result[i].Id.(mapKeyFork) < result[j].Id.(mapKeyFork)
		})
		return result
	}

	return []*ForkSourcePart{
		{
			Source: src,
			Id:     dummyForkId,
		},
	}
}

var dummyForkId undeterminedFork

// Computes the cartesian product of possible values for ForkIds.
func (set *ForkIdSet) MakeForkIds(allNodes map[string]*Node, srcs []syntax.MapCallSource) {
	if len(srcs) == 0 {
		return
	}
	set.Table = make(map[syntax.MapCallSource][]*ForkSourcePart, len(srcs))
	count := 1
	if len(srcs) == 1 {
		these := makeForkIdParts(srcs[0])
		set.Table[srcs[0]] = these
		set.List = make([]ForkId, len(these))
		for i, part := range these {
			set.List[i] = ForkId{part}
		}
		return
	}
	for _, src := range srcs {
		these := makeForkIdParts(src)
		set.Table[src] = these
		count *= len(these)
	}
	// Single allocation for the backing array
	idBlock := make([]*ForkSourcePart, count*len(srcs))
	// List of IDs is sliced from idBlock, each with length 0
	// and capacity len(srcs)
	set.List = make([]ForkId, count)
	for i := range set.List {
		set.List[i] = ForkId(idBlock[0:0:len(srcs)])
		idBlock = idBlock[len(srcs):]
	}
	stride := 1
	for _, src := range srcs {
		i := 0
		these := set.Table[src]
		for i < len(set.List) {
			for _, p := range these {
				for j := 0; j < stride; j++ {
					set.List[i] = append(set.List[i], p)
					i++
				}
			}
		}
		stride *= len(these)
	}
}

// Get the ForkId from an upstream stage with the given fork sources which
// corresponds to this fork.
//
// This ForkId must include every Source contained in upstream.
func (f ForkId) Match(ref map[syntax.MapCallSource]syntax.CollectionIndex,
	upstream []syntax.MapCallSource) (ForkId, error) {
	if len(upstream) == 0 {
		return nil, nil
	}
	result := make(ForkId, len(upstream))
	for i, src := range upstream {
		if r, err := f.matchPart(src); err != nil {
			if j := ref[src]; j != nil {
				if j.IndexSource() == nil {
					result[i] = &ForkSourcePart{
						Id:     convertForkPart(j),
						Source: src,
					}
				} else {
					return result, &elementError{
						element: "unknown index for " + j.IndexSource().GoString(),
						inner:   err,
					}
				}
			} else {
				return result, err
			}
		} else {
			result[i] = r
		}
	}
	return result, nil
}

// Matches returns true if the IDs match for every fork element with matching
// sources.
func (f ForkId) Matches(other ForkId) bool {
	for _, p1 := range f {
		if len(other) == 0 {
			return true
		}
		for j, p2 := range other {
			if p1.Source == p2.Source {
				if p1.Id != p2.Id &&
					p1.Id.IndexSource() == nil &&
					p2.Id.IndexSource() == nil {
					return false
				}
				other = other[j+1:]
				break
			}
		}
	}
	return true
}

func (f ForkId) UnmatchedParts(upstream syntax.ForkRootList) syntax.ForkRootList {
	if len(upstream) == 0 {
		return nil
	}
	if len(f) == 0 {
		return upstream
	}
	// If we have a sequence of map sources, the most common case is that we
	// match a contiguous subset of them.  In those cases, we want to return a
	// subslice rather than allocating a new one.

	match := func(i int, upstream syntax.ForkRootList) bool {
		if len(upstream) > i {
			_, err := f.matchPart(upstream[i].MapSource())
			return err == nil
		}
		return false
	}
	// Trim the matched parts off of the front.
	for match(0, upstream) {
		upstream = upstream[1:]
	}
	// Trim the matched parts off the end.
	for len(upstream) > 1 && match(len(upstream)-1, upstream) {
		upstream = upstream[:len(upstream)-1]
	}
	if len(upstream) <= 2 {
		return upstream
	}
	result := upstream[:1]
	if cap(result) != cap(upstream) {
		panic(cap(result))
	}
	for i := 1; i < len(upstream)-1; i++ {
		if _, err := f.matchPart(upstream[i].MapSource()); err != nil {
			// if we haven't reallocated, this is a no-op.
			result = append(result, upstream[i])
		} else if cap(result) == cap(upstream) {
			// Need to allocate a new buffer.  It's capacity will be sufficient
			// to hold all but one element of upstream.  We know we're
			// skipping at least one, so it won't ever need to grow, and being
			// smaller than the length of upstream means the capacity will no
			// longer match either.
			result = make(syntax.ForkRootList, i-1, len(upstream)-1)
			copy(result, upstream[:i])
		}
	}
	return append(result, upstream[len(upstream)-1])
}

type forkMatchNotFoundError struct {
	src  syntax.MapCallSource
	fork string
}

func (err forkMatchNotFoundError) Error() string {
	return "no match found for " + err.src.GoString() + " in " + err.fork
}

func (f ForkId) matchPart(src syntax.MapCallSource) (*ForkSourcePart, error) {
	for _, part := range f {
		if part.Source == src {
			return part, nil
		}
	}
	return nil, forkMatchNotFoundError{
		src:  src,
		fork: f.GoString(),
	}
}

// Returns true if the two fork Ids are equal.
func (f ForkId) Equal(o ForkId) bool {
	if len(f) != len(o) {
		return false
	}
	for i, fi := range f {
		if !fi.Equal(o[i]) {
			return false
		}
	}
	return true
}

func (f *ForkSourcePart) Equal(o *ForkSourcePart) bool {
	if f == o {
		return true
	} else if f == nil || o == nil {
		return false
	}
	if f.Source != o.Source {
		return false
	}
	if f.Id == nil {
		return o.Id == nil
	}
	if f.Id.Mode() != o.Id.Mode() {
		return false
	}
	if f.Id.IndexSource() != nil || o.Id.IndexSource() != nil {
		return true
	}
	switch f.Id.Mode() {
	case syntax.ModeArrayCall:
		return f.Id.ArrayIndex() == o.Id.ArrayIndex()
	case syntax.ModeMapCall:
		return f.Id.MapKey() == o.Id.MapKey()
	}
	panic("bad id type")
}

// String returns "fork.<key>".
func (k mapKeyFork) String() string {
	return k.forkString()
}

// forkString returns "fork.<key>".
func (k mapKeyFork) forkString() string {
	return "fork." + string(k)
}

func (mapKeyFork) ArrayIndex() int {
	panic("not an array index")
}

func (mapKeyFork) IndexSource() syntax.MapCallSource {
	return nil
}

func (k mapKeyFork) MapKey() string {
	return string(k)
}

func (mapKeyFork) Mode() syntax.CallMode {
	return syntax.ModeMapCall
}

// String returns "fork<i>".
func (i arrayIndexFork) String() string {
	return i.forkString()
}

// forkString returns "fork<i>".
func (i arrayIndexFork) forkString() string {
	if i < 0 {
		return defaultFork
	}
	j := int(i)
	if j == 0 {
		return defaultFork
	}
	var buf strings.Builder
	if j < 10 {
		buf.Grow(5)
	} else {
		buf.Grow(6)
	}
	buf.WriteString(defaultFork[:4])
	buf.WriteString(strconv.Itoa(j))
	return buf.String()
}

func (i arrayIndexFork) ArrayIndex() int {
	return int(i)
}

func (arrayIndexFork) IndexSource() syntax.MapCallSource {
	return nil
}

func (arrayIndexFork) MapKey() string {
	panic("not a map key")
}

func (arrayIndexFork) Mode() syntax.CallMode {
	return syntax.ModeArrayCall
}

func (undeterminedFork) String() string {
	return defaultFork
}

func (undeterminedFork) forkString() string {
	return defaultFork
}

func (undeterminedFork) ArrayIndex() int {
	panic("unknown index")
}

func (f undeterminedFork) IndexSource() syntax.MapCallSource {
	return f
}

func (undeterminedFork) MapKey() string {
	panic("unknown index")
}

func (undeterminedFork) CallMode() syntax.CallMode {
	return syntax.ModeUnknownMapCall
}

func (undeterminedFork) KnownLength() bool {
	return false
}

func (undeterminedFork) GoString() string {
	return "unknown index"
}

func (undeterminedFork) ArrayLength() int {
	return -1
}
func (undeterminedFork) Keys() map[string]syntax.Exp {
	return nil
}

func (undeterminedFork) Mode() syntax.CallMode {
	return syntax.ModeUnknownMapCall
}

func (p *ForkSourcePart) GoString() string {
	if p.Id.IndexSource() != nil {
		return p.Source.GoString() + ":" + p.Id.String()
	}
	switch p.Id.Mode() {
	case syntax.ModeArrayCall:
		return p.Source.GoString() + ":" + strconv.Itoa(p.Id.ArrayIndex())
	case syntax.ModeMapCall:
		return p.Source.GoString() + ":" + p.Id.MapKey()
	default:
		return p.Source.GoString() + ":" + p.Id.String()
	}
}

func (id ForkId) GoString() string {
	var buf strings.Builder
	buf.Grow(2 + 25*len(id))
	if _, err := buf.WriteRune('['); err != nil {
		panic(err)
	}
	for i, p := range id {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				panic(err)
			}
		}
		if _, err := buf.WriteString(p.GoString()); err != nil {
			panic(err)
		}
	}
	if _, err := buf.WriteRune(']'); err != nil {
		panic(err)
	}
	return buf.String()
}

// Most of the time there is only one fork, so use a shared constant for
// the ID.
const defaultFork = "fork0"

// ForkId computes the fork ID for a fully-resolved fork.
//
// If the fork source expression is of type SweepExp or ArrayExp,
// e.g. [sweep1, sweep2, sweep3], then the fork IDs are "forkN" with N
// going from 0 to the product of the lengths of the sweeps.  The concrete
// values are taken in the order
//
//  source:   sweep1 sweep2 sweep3
//   fork0:     0      0      0
//   fork1:     1      0      0
//   ...
//   forkN:     N      0      0
//   forkN+1:   0      1      0
//   ...
//   fork2*N:   N      1      0
//   ...
//   forkN*M:   N      M      0
//   ...
//   forkN*M*L: N      M      L
//
// Expressions which are of typed map type resolve to "fork.<key>"
// for each map key.
//
// If there are multiple map sources, or both map and array sources,
// IDs are concatenated with '/' as a separator character.
func (f ForkId) ForkIdString() (string, error) {
	if len(f) == 0 {
		return defaultFork, nil
	}
	if len(f) == 1 {
		return f[0].ForkIdString()
	}
	var buf strings.Builder
	if isDefault, err := f.forkId(&buf, 0); isDefault {
		return defaultFork, err
	} else {
		return buf.String(), err
	}
}

// Returns the fork ID string for a single fork component.
//
// Normally this is just called by ForkId.ForkIdString(), as a special case to
// avoid memory allocation in the common case where fork sources aren't nested.
func (f *ForkSourcePart) ForkIdString() (string, error) {
	if f == nil {
		return defaultFork, fmt.Errorf("nil part")
	}
	if f.Id == nil {
		return defaultFork, fmt.Errorf("nil fork id")
	}
	if f.Source == nil {
		return defaultFork, fmt.Errorf("nil fork source")
	}
	if f.Source.CallMode() == syntax.ModeSingleCall {
		return defaultFork, nil
	}
	switch f.Id.(type) {
	case undeterminedFork:
		return defaultFork, nil
	case syntax.CollectionIndex:
		if f.Id.IndexSource() != nil {
			return defaultFork, fmt.Errorf(
				"unresolved fork index")
		}
		if f.Source.CallMode() != f.Id.Mode() {
			return defaultFork, fmt.Errorf(
				"mismatched index type %s with source %s",
				f.Id.Mode(), f.Source.CallMode().String())
		}
		switch f.Source.CallMode() {
		case syntax.ModeArrayCall:
			i := f.Id.ArrayIndex()
			if i < 0 {
				return defaultFork, fmt.Errorf("invalid array index %d", i)
			}
			if f.Source.KnownLength() {
				alen := f.Source.ArrayLength()
				if i >= alen {
					return defaultFork, fmt.Errorf(
						"len(sweep) == %d <= index %d",
						alen, i)
				}
			}
			if i == 0 {
				return defaultFork, nil
			}
			var buf strings.Builder
			w := util.WidthForInt(i)
			buf.Grow(4 + w)
			if _, err := buf.WriteString(defaultFork[:4]); err != nil {
				return defaultFork, err
			}
			for x := util.WidthForInt(i); x < w; x++ {
				if _, err := buf.WriteRune('0'); err != nil {
					return defaultFork, err
				}
			}
			if _, err := buf.WriteString(strconv.Itoa(i)); err != nil {
				return defaultFork, err
			}
			return buf.String(), nil
		case syntax.ModeMapCall:
			k := f.Id.MapKey()
			if f.Source.KnownLength() {
				if _, ok := f.Source.Keys()[k]; !ok {
					return defaultFork, fmt.Errorf("no key %q in source",
						k)
				}
			}
			return f.Id.forkString(), nil
		default:
			return defaultFork, nil
		}
	default:
		panic("invalid source type")
	}
}

// Recursively build fork ID.  If it's the default fork (fork0) then return true
// to avoid the allocation in the builder.
func (f ForkId) forkId(buf *strings.Builder, start int) (bool, error) {
	forkIndex := 0
	forkDim := 1
	for i, part := range f[start:] {
		if part == nil {
			return true, fmt.Errorf("nil part")
		} else if part.Id == nil {
			return true, fmt.Errorf("nil fork id")
		} else if part.Source == nil {
			return true, fmt.Errorf("nil fork source")
		}
		if part.Id.IndexSource() != nil {
			continue
		}
		switch part.Id.Mode() {
		case syntax.ModeArrayCall:
			if !part.Source.KnownLength() {
				return true, fmt.Errorf(
					"fork is not fully resolved")
			}
			j := part.Id.ArrayIndex()
			if j < 0 {
				return true, fmt.Errorf("invalid fork index %d", j)
			}
			forkIndex += forkDim * j
			if part.Source.CallMode() != syntax.ModeArrayCall {
				return true, fmt.Errorf(
					"can't get integer index of %s source",
					part.Source.CallMode().String())
			}
			alen := part.Source.ArrayLength()
			if j >= alen {
				return true, fmt.Errorf(
					"len(sweep) == %d <= index %d",
					alen, j)
			}
			forkDim *= alen
		case syntax.ModeMapCall:
			if !part.Source.KnownLength() {
				return true, fmt.Errorf(
					"fork is not fully resolved")
			}
			if i == 0 {
				// Write this fork id.
				if part.Source.CallMode() != syntax.ModeMapCall {
					return true, fmt.Errorf(
						"can't get map index from fork source %s",
						part.Source.CallMode().String())
				}
				if _, ok := part.Source.Keys()[part.Id.MapKey()]; !ok {
					return true, fmt.Errorf("no key %q in source",
						part.Id.MapKey())
				}
				if _, err := buf.WriteString("fork."); err != nil {
					return true, err
				}
				if _, err := buf.WriteString(part.Id.MapKey()); err != nil {
					return true, err
				}
				if i < len(f)-1 {
					if _, err := buf.WriteRune('/'); err != nil {
						return true, err
					}
					if isDefault, err := f.forkId(buf, start+1); err != nil {
						return true, err
					} else if isDefault {
						_, err := buf.WriteString(defaultFork)
						return false, err
					}
					return false, nil
				} else {
					return false, nil
				}
				// Otherwise, flush the fork index and recurse.
			} else if forkIndex == 0 {
				if _, err := buf.WriteString(defaultFork); err != nil {
					return true, err
				}
				if _, err := buf.WriteRune('/'); err != nil {
					return true, err
				}
			} else if err := f.writeForkIndex(buf, forkDim, forkIndex); err != nil {
				return true, err
			} else if _, err := buf.WriteRune('/'); err != nil {
				return true, err
			}
			return f.forkId(buf, start+i)
		default:
			panic("invalid source type")
		}
	}
	if forkIndex == 0 {
		return true, nil
	} else if err := f.writeForkIndex(buf, forkDim, forkIndex); err != nil {
		return true, err
	}
	return false, nil
}

func (ForkId) writeForkIndex(buf *strings.Builder, forkDim, forkIndex int) error {
	if _, err := buf.WriteString("fork"); err != nil {
		return err
	}
	w := util.WidthForInt(forkDim)
	id := strconv.Itoa(forkIndex)
	for x := len(id); x < w; x++ {
		if _, err := buf.WriteRune('0'); err != nil {
			return err
		}
	}
	_, err := buf.WriteString(id)
	return err
}

func getUnknownLength(v json.Marshaler) (int, error) {
	switch v := v.(type) {
	case *syntax.ArrayExp:
		return len(v.Value), nil
	case marshallerArray:
		return len(v), nil
	case json.RawMessage:
		if len(v) <= 2 || bytes.Equal(v, nullBytes) {
			return 0, nil
		}
		var m marshallerArray
		err := json.Unmarshal(v, &m)
		return len(m), err
	}
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	if val.Kind() == reflect.Array {
		return val.Len(), nil
	}
	return 0, fmt.Errorf("can't take length of %s",
		val.Type().String())
}

func getUnknownKeys(v json.Marshaler) ([]string, error) {
	switch v := v.(type) {
	case *syntax.MapExp:
		if len(v.Value) == 0 {
			return nil, nil
		}
		keys := make([]string, 0, len(v.Value))
		for k := range v.Value {
			keys = append(keys, k)
		}
		return keys, nil
	case MarshalerMap:
		if len(v) == 0 {
			return nil, nil
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys, nil
	case LazyArgumentMap:
		if len(v) == 0 {
			return nil, nil
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys, nil
	case json.RawMessage:
		if len(v) < 2 || bytes.Equal(v, nullBytes) {
			return nil, nil
		}
		var m LazyArgumentMap
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		return keys, nil
	}
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	if val.Kind() == reflect.Map {
		mk := val.MapKeys()
		keys := make([]string, 0, len(mk))
		for _, k := range mk {
			if k.Kind() != reflect.String {
				return keys, fmt.Errorf("%s is not a string key", k.String())
			}
			keys = append(keys, k.String())
		}
		return keys, nil
	}
	return nil, fmt.Errorf("can't take keys of %s",
		val.Type().String())
}

func (self *Fork) getUnmatchedForkParts(bNode *Node) []*ForkSourcePart {
	if rl := self.forkId.UnmatchedParts(bNode.call.ForkRoots()); len(rl) > 0 {
		bNode = self.node.top.allNodes[rl[0].GetFqid()]
		parts := make([]*ForkSourcePart, 0, len(bNode.forks))
		for _, f := range bNode.forks {
			for _, part := range f.forkId {
				if part.Source == rl[0].MapSource() {
					parts = append(parts, part)
				}
			}
		}
		return parts
	}
	return nil
}

func (self *Fork) expandForkPart(i int, part *ForkSourcePart,
	src syntax.MapCallSource) ([]ForkId, error) {
	var ref *syntax.RefExp
	switch r := src.(type) {
	case *syntax.RefExp:
		ref = r
	case *syntax.ReferenceMappingSource:
		ref = r.Ref
	default:
		panic("invalid source for undetermined split " +
			src.GoString() + " (computing forks for " +
			self.fqname + ")")
	}
	bNode := self.node.top.allNodes[ref.Id]
	if bNode == nil {
		panic("invalid reference to " + ref.Id)
	}
	if parts := self.getUnmatchedForkParts(bNode); len(parts) > 0 {
		rep := make(ForkId, len(self.forkId)*(len(parts)))
		result := make([]ForkId, len(parts)-1)
		for j, part := range parts {
			if j == 0 {
				self.forkId[i] = part
				self.updateId(self.forkId)
			} else {
				result[j-1] = rep[j*len(self.forkId) : (j+1)*len(self.forkId) : (j+1)*len(self.forkId)]
				fid := result[j-1]
				for k, p := range self.forkId {
					if k == i {
						fid[k] = part
					} else {
						fid[k] = p
					}
				}
			}
		}
		return result, nil
	}
	matchedForks := bNode.matchForks(self.forkId)
	if len(matchedForks) != 1 {
		return nil, fmt.Errorf("matched %d out of %d forks of %s, need exactly 1",
			len(matchedForks), len(bNode.forks), bNode.GetFQName())
	}
	ready, obj, err := matchedForks[0].resolveRef(ref, nil, self.forkId,
		bNode.top.types, bNode.call.Call().DecId, 1024*1024)
	if err != nil || !ready {
		return nil, &elementError{
			element: "evaluating mapping source " + ref.GoString(),
			inner:   err,
		}
	}
	if obj == nil {
		part.Id = arrayIndexFork(0)
		self.updateId(self.forkId)
		self.writeDisable()
		return nil, nil
	}
	switch src.CallMode() {
	case syntax.ModeArrayCall:
		n, err := getUnknownLength(obj)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			part.Id = arrayIndexFork(0)
			self.updateId(self.forkId)
			self.writeDisable()
			return nil, nil
		} else if n == 1 {
			part.Id = arrayIndexFork(0)
			self.updateId(self.forkId)
			return nil, nil
		}
		re := make([]ForkSourcePart, n)
		rep := make(ForkId, len(self.forkId)*n)
		result := make([]ForkId, n-1)
		for j := 0; j < n; j++ {
			r := arrayIndexFork(j)
			re[j].Source = src
			re[j].Id = r
			if j == 0 {
				self.forkId[i] = &re[j]
				self.updateId(self.forkId)
			} else {
				result[j-1] = rep[j*len(self.forkId) : (j+1)*len(self.forkId) : (j+1)*len(self.forkId)]
				fid := result[j-1]
				for k, p := range self.forkId {
					if k == i {
						fid[k] = &re[j]
					} else {
						fid[k] = p
					}
				}
			}
		}
		return result, nil
	case syntax.ModeMapCall:
		keys, err := getUnknownKeys(obj)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 {
			part.Id = mapKeyFork("")
			self.updateId(self.forkId)
			self.writeDisable()
			return nil, nil
		}
		if len(keys) == 1 {
			part.Id = mapKeyFork(keys[0])
			self.updateId(self.forkId)
			return nil, nil
		}
		sort.Strings(keys)
		re := make([]ForkSourcePart, len(keys))
		rep := make(ForkId, len(self.forkId)*(len(keys)))
		result := make([]ForkId, len(keys)-1)
		for j := 0; j < len(keys); j++ {
			r := mapKeyFork(keys[j])
			re[j].Source = src
			re[j].Id = r
			if j == 0 {
				self.forkId[i] = &re[j]
				self.updateId(self.forkId)
			} else {
				result[j-1] = rep[j*len(self.forkId) : (j+1)*len(self.forkId) : (j+1)*len(self.forkId)]
				fid := result[j-1]
				for k, p := range self.forkId {
					if k == i {
						fid[k] = &re[j]
					} else {
						fid[k] = p
					}
				}
			}
		}
		return result, nil
	}
	panic("invalid fork mode " + src.CallMode().String())
}

func (self *Fork) expand() ([]ForkId, error) {
	if len(self.forkId) == 0 {
		if src := self.node.call.MapSource(); src != nil && !src.KnownLength() {
			part := &ForkSourcePart{
				Source: src,
				Id:     undeterminedFork{},
			}
			self.forkId = ForkId{part}
			ids, err := self.expandForkPart(0, part, src)
			if err != nil {
				self.forkId = nil
			}
			return ids, err
		}
	}
	for i, part := range self.forkId {
		if part.Id.IndexSource() != nil {
			src := part.Source
			if ids, err := self.expandForkPart(i, part, src); err != nil ||
				len(ids) > 0 {
				return ids, err
			}
		}
	}
	return nil, nil
}
