package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

// Limit of JSON read size in bytes.
const readSizeLimit int64 = 2 * 1024 * 1024

type (
	// Represents a component of a fork ID, for a map-called stage or
	// pipeline.
	ForkIdPart interface {
		fmt.Stringer
		syntax.CollectionIndex

		// The string ID for a fork.
		//
		// For a map-type fork, this will be "fork.<key>".  For an array-type
		// fork it will be "forkN" where N is the array index.
		forkString() string
	}

	// Represents a range of allowed values for a ForkSourcePart.
	ForkIdRange interface {
		Allow(syntax.CollectionIndex) error
		Length() int
	}

	mapKeyFork string

	arrayIndexFork int

	undeterminedFork struct{}

	emptyFork struct{}

	mapSourceRange struct {
		Source syntax.MapCallSource
	}

	arrayLengthRange int

	mapKeyRange []string

	ForkSourcePart struct {
		Split *syntax.SplitExp
		Id    ForkIdPart
		Range ForkIdRange
	}

	// A set of expressions and the corresponding index values for each one,
	// which identifies a specific fork.
	ForkId []*ForkSourcePart

	ForkIdSet struct {
		Table map[*syntax.CallStm][]*ForkSourcePart
		List  []ForkId
	}
)

func convertForkPart(f syntax.CollectionIndex) ForkIdPart {
	switch f := f.(type) {
	case mapKeyFork:
		return f
	case arrayIndexFork:
		return f
	case emptyFork:
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

func (fork ForkId) SourceIndexMap() map[*syntax.CallStm]syntax.CollectionIndex {
	if len(fork) == 0 {
		return nil
	}
	result := make(map[*syntax.CallStm]syntax.CollectionIndex, len(fork))
	for _, part := range fork {
		result[part.Split.Call] = part.Id
	}
	return result
}

func countForkParts(src syntax.MapCallSource,
	index map[*syntax.CallStm]syntax.CollectionIndex,
	lookup *syntax.TypeLookup) int {
	if src.KnownLength() {
		switch src.CallMode() {
		case syntax.ModeArrayCall:
			return src.ArrayLength()
		case syntax.ModeMapCall:
			return len(src.Keys())
		}
	}
	switch src := src.(type) {
	case *syntax.SplitExp:
		if src.Source.KnownLength() {
			switch src.Source.CallMode() {
			case syntax.ModeArrayCall:
				c := 0
				if old, ok := index[src.Call]; ok {
					defer func() { index[src.Call] = old }()
				} else if index != nil {
					defer delete(index, src.Call)
				}
				for i := 0; i < src.Source.ArrayLength(); i++ {
					if index == nil {
						index = make(map[*syntax.CallStm]syntax.CollectionIndex)
					}
					index[src.Call] = arrayIndexFork(i)
					sub, err := src.BindingPath("", index, lookup)
					if err != nil || sub == src {
						c++
					} else if subs, ok := sub.(syntax.MapCallSource); ok {
						c += countForkParts(subs, index, lookup)
					} else {
						c++
					}
				}
				return c
			case syntax.ModeMapCall:
				c := 0
				if old, ok := index[src.Call]; ok {
					defer func() { index[src.Call] = old }()
				} else if index != nil {
					defer delete(index, src.Call)
				}
				for i := range src.Source.Keys() {
					if index == nil {
						index = make(map[*syntax.CallStm]syntax.CollectionIndex)
					}
					index[src.Call] = mapKeyFork(i)
					sub, err := src.BindingPath("", index, lookup)
					if err != nil || sub == src {
						c++
					} else if subs, ok := sub.(syntax.MapCallSource); ok {
						c += countForkParts(subs, index, lookup)
					} else {
						c++
					}
				}
				return c
			}
		}
	case *syntax.MergeExp:
		return countForkParts(src.MergeOver, index, lookup)
	}
	return 1
}

func makeForkIdParts(split *syntax.SplitExp,
	lookup *syntax.TypeLookup) ([]*ForkSourcePart, int) {
	switch split.Source.CallMode() {
	case syntax.ModeArrayCall:
		if alen := split.Source.ArrayLength(); alen >= 0 {
			re := make([]ForkSourcePart, alen)
			result := make([]*ForkSourcePart, alen)
			for i := range result {
				r := arrayIndexFork(i)
				re[i].Split = split
				re[i].Id = r
				result[i] = &re[i]
			}
			return result, 0
		}
	case syntax.ModeMapCall:
		if keys := split.Source.Keys(); keys != nil {
			re := make([]ForkSourcePart, 0, len(keys))
			result := make([]*ForkSourcePart, 0, len(keys))
			for k := range keys {
				r := mapKeyFork(k)
				re = append(re, ForkSourcePart{
					Split: split,
					Id:    r,
				})
				result = append(result, &re[len(re)-1])
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i].Id.(mapKeyFork) < result[j].Id.(mapKeyFork)
			})
			return result, 0
		}
	case syntax.ModeNullMapCall:
		return []*ForkSourcePart{
			{
				Split: split,
				Id:    dummyForkId,
			},
		}, 0
	}
	return []*ForkSourcePart{
		{
			Split: split,
			Id:    dummyForkId,
		},
	}, countForkParts(split.Source, nil, lookup)
}

var dummyForkId undeterminedFork

// Computes the cartesian product of possible values for ForkIds.
func (set *ForkIdSet) MakeForkIds(srcs syntax.ForkRootList,
	lookup *syntax.TypeLookup) {
	if len(srcs) == 0 {
		return
	}
	set.Table = make(map[*syntax.CallStm][]*ForkSourcePart, len(srcs))
	if len(srcs) == 1 {
		these, extra := makeForkIdParts(srcs[0].Split(), lookup)
		set.Table[srcs[0].Call()] = these
		set.List = make([]ForkId, len(these), len(these)+extra)
		for i, part := range these {
			set.List[i] = ForkId{part}
		}
		return
	}
	count := 1
	extra := 0
	for _, srcNode := range srcs {
		these, e := makeForkIdParts(srcNode.Split(), lookup)
		set.Table[srcNode.Call()] = these
		count *= len(these)
		if e == 0 {
			extra *= len(these)
		} else {
			extra += e
		}
	}
	// Single allocation for the backing array
	idBlock := make([]*ForkSourcePart, count*len(srcs))
	// List of IDs is sliced from idBlock, each with length 0
	// and capacity len(srcs)
	set.List = make([]ForkId, count, count+extra)
	for i := range set.List {
		set.List[i] = ForkId(idBlock[0:0:len(srcs)])
		idBlock = idBlock[len(srcs):]
	}
	stride := 1
	for _, src := range srcs {
		i := 0
		these := set.Table[src.Call()]
		if these != nil {
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
	if extra >= 0 {
		set.expandStaticForks(lookup)
	}
}

func (set *ForkIdSet) expandStaticForks(lookup *syntax.TypeLookup) {
	index := make(map[*syntax.CallStm]syntax.CollectionIndex, len(set.List[0]))
	for i := 0; i < len(set.List); i++ {
		fork := set.List[i]
		newForks := fork.expandStaticForks(index, set.List[len(set.List):], lookup)
		for ; len(newForks) > 0; newForks = fork.expandStaticForks(index,
			set.List[len(set.List):], lookup) {
			set.List = append(set.List, newForks...)
		}
	}
}

func (fork ForkId) expandStaticForks(index map[*syntax.CallStm]syntax.CollectionIndex,
	slice []ForkId,
	lookup *syntax.TypeLookup) []ForkId {
	for i := 0; i < len(fork); i++ {
		part := fork[i]
		if part.Id.IndexSource() != nil {
			if ids := fork.expandStaticForkPart(i, part,
				part.Split, index, slice, lookup); len(ids) > 0 {
				return ids
			} else if part.Id == arrayIndexFork(-1) {
				return nil
			}
		}
	}
	return nil
}

// Get the (statically-resolved) source for a fork.
func (fork ForkId) getForkSrc(split *syntax.SplitExp,
	index map[*syntax.CallStm]syntax.CollectionIndex,
	lookup *syntax.TypeLookup) syntax.MapCallSource {
	if split.Source.KnownLength() || len(fork) == 0 {
		return split.Source
	}
	for _, part := range fork {
		if part.Id.IndexSource() == nil {
			if id, ok := index[part.Split.Call]; ok {
				defer func() { index[part.Split.Call] = id }()
			} else {
				defer delete(index, part.Split.Call)
			}
			index[part.Split.Call] = part.Id
		}
	}
	exp, err := split.BindingPath("", index, lookup)
	if err != nil {
		panic("resolving source for " + split.GoString() +
			" of fork " + fork.GoString() + ": " + err.Error())
	}
	switch exp := exp.(type) {
	case *syntax.SplitExp:
		if exp.Call == split.Call {
			if ss, ok := exp.Value.(syntax.MapCallSource); ok {
				return ss
			} else {
				return exp.Source
			}
		} else {
			return exp
		}
	case syntax.MapCallSource:
		return exp
	}
	return split.Source
}

func (fork ForkId) expandStaticForkPart(i int, part *ForkSourcePart,
	split *syntax.SplitExp,
	index map[*syntax.CallStm]syntax.CollectionIndex,
	result []ForkId,
	lookup *syntax.TypeLookup) []ForkId {
	forkSrc := fork.getForkSrc(split, index, lookup)
	if forkSrc == nil || !forkSrc.KnownLength() {
		return nil
	}
	var r ForkIdRange = mapSourceRange{Source: forkSrc}
	if part.Range != nil {
		r = part.Range
	}
	if r.Length() <= 0 {
		p := *part
		fork[i] = &p
		p.Range = r
		p.Id = emptyFork{}
		return nil
	}
	if s, ok := forkSrc.(*syntax.SplitExp); ok {
		split = s
	}
	switch forkSrc.CallMode() {
	case syntax.ModeArrayCall:
		if forkSrc.ArrayLength() == 1 {
			p := *part
			p.Range = r
			p.Id = arrayIndexFork(0)
			p.Split = split
			fork[i] = &p
			return nil
		}
		parts := make([]ForkSourcePart, r.Length())
		ids := make([]*ForkSourcePart, len(fork)*(len(parts)-1))
		for j := range parts {
			parts[j] = *part
			parts[j].Range = r
			parts[j].Split = split
			parts[j].Id = arrayIndexFork(j)
			if j == 0 {
				fork[i] = &parts[0]
			} else {
				id := ids[:len(fork)]
				ids = ids[len(fork):]
				for k := range id {
					if k == i {
						id[k] = &parts[j]
					} else {
						id[k] = fork[k]
					}
				}
				result = append(result, id)
			}
		}
		return result
	case syntax.ModeMapCall:
		keyMap := forkSrc.Keys()
		if len(keyMap) == 1 {
			p := *part
			p.Range = r
			for k := range keyMap {
				p.Id = mapKeyFork(k)
			}
			p.Split = split
			fork[i] = &p
			return nil
		}
		keys := make([]string, 0, len(keyMap))
		for k := range keyMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]ForkSourcePart, len(keys))
		ids := make(ForkId, len(fork)*(len(parts)-1))
		for j := range parts {
			parts[j] = *part
			parts[j].Range = r
			parts[j].Split = split
			parts[j].Id = mapKeyFork(keys[j])
			if j == 0 {
				fork[i] = &parts[0]
			} else {
				id := ids[:len(fork)]
				ids = ids[len(fork):]
				for k := range id {
					if k == i {
						id[k] = &parts[j]
					} else {
						id[k] = fork[k]
					}
				}
				result = append(result, id)
			}
		}
		return result
	}
	panic("invalid fork mode " + forkSrc.CallMode().String())
}

// Get the ForkId from an upstream stage with the given fork sources which
// corresponds to this fork.
//
// This ForkId must include every Source contained in upstream.
func (f ForkId) Match(ref map[*syntax.CallStm]syntax.CollectionIndex,
	upstream []*syntax.CallStm) (ForkId, error) {
	if len(upstream) == 0 {
		return nil, nil
	}
	result := make(ForkId, len(upstream))
	for i, src := range upstream {
		if r, err := f.matchPart(src); err != nil {
			if j, ok := ref[src]; ok && j != nil {
				if j.IndexSource() == nil {
					result[i] = &ForkSourcePart{
						Id: convertForkPart(j),
						Split: &syntax.SplitExp{
							Value:  &syntax.MergeExp{MergeOver: src},
							Call:   src,
							Source: src,
						},
					}
					if src.CallMode() != result[i].Id.Mode() {
						// Should not be possible - checked during static analysis.
						panic(result[i].GoString() + " from " + j.Mode().String())
					}
				} else {
					return result, &elementError{
						element: "unknown index for " + j.IndexSource().GoString(),
						inner:   err,
					}
				}
			} else {
				found := false
				for s, j := range ref {
					if j != nil && s == src {
						if j.IndexSource() == nil {
							found = true
							result[i] = &ForkSourcePart{
								Id: convertForkPart(j),
								Split: &syntax.SplitExp{
									Value:  &syntax.MergeExp{MergeOver: src},
									Call:   src,
									Source: src,
								},
							}
							if src.CallMode() != result[i].Id.Mode() {
								// Should not be possible - checked during static analysis.
								panic(result[i].GoString() + " from " + j.Mode().String())
							}
						} else {
							return result, &elementError{
								element: "unknown index for " + j.IndexSource().GoString(),
								inner:   err,
							}
						}
					}
				}
				if !found {
					return result, err
				}
			}
		} else {
			result[i] = r
		}
	}
	return result, nil
}

// Matches returns true if the IDs match for every fork element with matching
// sources where the source is determined.
func (f ForkId) Matches(other ForkId) bool {
	for _, p1 := range f {
		if len(other) == 0 {
			return true
		}
		for j, p2 := range other {
			if p1.Split.Call == p2.Split.Call {
				if p1.Id.IndexSource() == nil &&
					p2.Id.IndexSource() == nil &&
					!indexEqual(p1.Id, p2.Id) {
					return false
				}
				// Don't re-check these parts on the next pass.  Ordering
				// should be consistent due to nesting.
				other = other[j+1:]
				break
			}
		}
	}
	return true
}

func indexEqual(i, j syntax.CollectionIndex) bool {
	if i == j {
		return true
	}
	if m := i.Mode(); m == syntax.ModeNullMapCall {
		return true
	} else if n := j.Mode(); n == syntax.ModeNullMapCall {
		return true
	} else if s := i.IndexSource(); s != j.IndexSource() || s != nil {
		return false
	} else if m != n {
		return false
	} else if m == syntax.ModeArrayCall {
		return i.ArrayIndex() == j.ArrayIndex()
	} else if m == syntax.ModeMapCall {
		return i.MapKey() == j.MapKey()
	} else {
		return true
	}
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
			_, err := f.matchPart(upstream[i].Call())
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
		if _, err := f.matchPart(upstream[i].Call()); err != nil {
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
	src  *syntax.CallStm
	fork string
}

func (err forkMatchNotFoundError) Error() string {
	return "no match found for " + err.src.GoString() + " in " + err.fork
}

// matchPart returns the part of the fork ID corresponding to the given call.
func (f ForkId) matchPart(src *syntax.CallStm) (*ForkSourcePart, error) {
	for _, part := range f {
		if part.Split.Call == src {
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

func (f *ForkSourcePart) GetRange() ForkIdRange {
	if f.Split.Source.KnownLength() {
		return mapSourceRange{Source: f.Split.Source}
	}
	if f.Range != nil {
		return f.Range
	}
	panic("unknown range")
}

func (r mapSourceRange) Allow(i syntax.CollectionIndex) error {
	if i.IndexSource() != nil {
		if i.IndexSource() != r.Source {
			return fmt.Errorf("mismatched sources %s vs %s",
				i.IndexSource().GoString(), r.Source.GoString())
		}
	}
	if i.Mode() != r.Source.CallMode() {
		return fmt.Errorf("mismatched modes %s vs %s",
			i.Mode().String(), r.Source.CallMode().String())
	}
	switch i.Mode() {
	case syntax.ModeArrayCall:
		j := i.ArrayIndex()
		if j < 0 {
			return fmt.Errorf("invalid fork array index %d", j)
		}
		alen := r.Source.ArrayLength()
		if j >= alen {
			return fmt.Errorf(
				"len(sweep) == %d <= index %d",
				alen, j)
		}
	case syntax.ModeMapCall:
		k := i.MapKey()
		if _, ok := r.Source.Keys()[k]; !ok {
			return fmt.Errorf("no key %q in source", k)
		}
	default:
		panic("invalid index mode " + i.Mode().String())
	}
	return nil
}

func (r mapSourceRange) Length() int {
	if !r.Source.KnownLength() {
		return -1
	}
	switch r.Source.CallMode() {
	case syntax.ModeArrayCall:
		return r.Source.ArrayLength()
	case syntax.ModeMapCall:
		return len(r.Source.Keys())
	case syntax.ModeSingleCall:
		return 1
	}
	return 0
}

func (r arrayLengthRange) Allow(i syntax.CollectionIndex) error {
	j := i.ArrayIndex()
	if j < 0 {
		return fmt.Errorf("invalid fork array index %d", j)
	}
	if j >= int(r) {
		return fmt.Errorf(
			"len(sweep) == %d <= index %d",
			int(r), j)
	}
	return nil
}

func (r arrayLengthRange) Length() int {
	return int(r)
}

func (r mapKeyRange) Allow(i syntax.CollectionIndex) error {
	x := i.MapKey()
	for _, k := range r {
		if x == k {
			return nil
		}
	}
	return fmt.Errorf("no key %q in source", x)
}

func (r mapKeyRange) Length() int {
	return len(r)
}

func (f *ForkSourcePart) Equal(o *ForkSourcePart) bool {
	if f == o {
		return true
	} else if f == nil || o == nil {
		return false
	}
	if f.Split.Call != o.Split.Call {
		return false
	}
	if f.Id == nil {
		return o.Id == nil
	}
	if o.Id.Mode() == syntax.ModeNullMapCall {
		return true
	} else if f.Id.Mode() == syntax.ModeNullMapCall {
		return true
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

// GoString returns "fork.<key>".
func (k mapKeyFork) GoString() string {
	return k.forkString()
}

// makeKeySafe returns a "safe" version of the key.
//
// Keys are escaped using RFC 3986 "percent encoding" to make them safe to use
// as path names.
func makeKeySafe(k string) string {
	return url.PathEscape(k)
}

// writeSafeKey writes a "safe" version of the key to te given buffer.
//
// Keys are escaped using RFC 3986 "percent encoding" to make them safe to use
// as path names.
func writeSafeKey(buf *strings.Builder, k string) {
	k = url.PathEscape(k)
	_, err := buf.WriteString(k)
	if err != nil {
		panic(err)
	}
}

// forkString returns "fork.<key>".
//
// Keys are escaped using RFC 3986 "percent encoded" to make them safe to use
// as path names, with '.' additionally being encoded according to the same
// scheme (to %2E), because of the way journal files are parsed later.
func (k mapKeyFork) forkString() string {
	return "fork_" + makeKeySafe(string(k))
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

// GoString returns "fork<i>".
func (i arrayIndexFork) GoString() string {
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

func (emptyFork) String() string {
	return "empty"
}

func (emptyFork) GoString() string {
	return "empty"
}

func (emptyFork) forkString() string {
	return defaultFork
}

func (emptyFork) ArrayIndex() int {
	return -1
}

func (emptyFork) IndexSource() syntax.MapCallSource {
	return nil
}

func (emptyFork) MapKey() string {
	return ""
}

func (emptyFork) Mode() syntax.CallMode {
	return syntax.ModeNullMapCall
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
		if p.Split.Call == nil {
			return p.Split.GoString() + ":" + p.Id.GoString()
		}
		return p.Split.Call.Id + ":" + p.Split.GoString() + ":unmatched"
	}
	switch p.Id.Mode() {
	case syntax.ModeArrayCall:
		return p.Split.Call.Id + ":" + p.Split.GoString() + ":" + strconv.Itoa(p.Id.ArrayIndex())
	case syntax.ModeMapCall:
		return p.Split.Call.Id + ":" + p.Split.GoString() + ":" + p.Id.MapKey()
	default:
		return p.Split.Call.Id + ":" + p.Split.GoString() + ":unknown(" + p.Id.String() + ")"
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
//	source:   sweep1 sweep2 sweep3
//	 fork0:     0      0      0
//	 fork1:     1      0      0
//	 ...
//	 forkN:     N      0      0
//	 forkN+1:   0      1      0
//	 ...
//	 fork2*N:   N      1      0
//	 ...
//	 forkN*M:   N      M      0
//	 ...
//	 forkN*M*L: N      M      L
//
// Expressions which are of typed map type resolve to "fork_<key>"
// for each map key, with keys "percent encoded" as per RFC 3986 to ensure
// safety for use in path names.
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
	if f.Split == nil {
		return defaultFork, fmt.Errorf("nil fork split")
	} else if f.Split.Source == nil {
		return defaultFork, fmt.Errorf("nil fork source")
	}
	if f.Split.Source.CallMode() == syntax.ModeSingleCall {
		return defaultFork, nil
	}
	switch f.Id.(type) {
	case undeterminedFork:
		return defaultFork, nil
	case emptyFork:
		return defaultFork, nil
	case syntax.CollectionIndex:
		if f.Id.IndexSource() != nil {
			return defaultFork, fmt.Errorf(
				"unresolved fork index")
		}
		if f.Split.Source.CallMode() != f.Id.Mode() {
			return defaultFork, fmt.Errorf(
				"mismatched index type %s with source %s",
				f.Id.Mode(), f.Split.Source.CallMode().String())
		}
		switch f.Split.Source.CallMode() {
		case syntax.ModeArrayCall:
			i := f.Id.ArrayIndex()
			if i < 0 {
				return defaultFork, fmt.Errorf("invalid array index %d", i)
			}
			if f.Split.Source.KnownLength() {
				alen := f.Split.Source.ArrayLength()
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
			if f.Split.Source.KnownLength() {
				if _, ok := f.Split.Source.Keys()[k]; !ok {
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
		} else if part.Split == nil || part.Split.Source == nil {
			return true, fmt.Errorf("nil fork source")
		}
		if part.Id.IndexSource() != nil {
			continue
		}
		r := part.GetRange()
		alen := r.Length()
		if alen < 0 {
			return true, &forkResolutionError{
				Msg: "fork is not fully resolved: unknown length for " +
					part.Split.Source.GoString() + " validating index " +
					part.Id.GoString(),
			}
		} else if alen == 0 {
			return forkIndex == 0, nil
		}
		if err := r.Allow(part.Id); err != nil {
			return forkIndex == 0, err
		}
		switch part.Id.Mode() {
		case syntax.ModeArrayCall:
			if alen != part.Split.Source.ArrayLength() && alen > 1 {
				// a variable-length source.  We can't easily combine them into
				// a single sequential index scheme because different forks
				// of one part may have different lengths for another part.
				if i != 0 {
					if err := f.writeForkIndex(buf, forkDim, forkIndex); err != nil {
						return true, err
					} else if _, err := buf.WriteRune('_'); err != nil {
						return true, err
					}
					forkIndex = 0
					forkDim = 1
				}
			}
			forkIndex += forkDim * part.Id.ArrayIndex()
			if alen == 0 {
				return forkIndex == 0, nil
			}
			if err := r.Allow(part.Id); err != nil {
				return forkIndex == 0, err
			}
			forkDim *= alen
		case syntax.ModeMapCall:
			if i == 0 {
				// Write this fork id.
				if _, err := buf.WriteString("fork_"); err != nil {
					return true, err
				}
				writeSafeKey(buf, part.Id.MapKey())
				if start+i < len(f)-1 {
					if _, err := buf.WriteRune('/'); err != nil {
						return true, err
					}
					if isDefault, err := f.forkId(buf, start+i+1); err != nil {
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
			} else if err := f.writeForkIndex(buf, forkDim, forkIndex); err != nil {
				return true, err
			} else if _, err := buf.WriteRune('/'); err != nil {
				return true, err
			}
			return f.forkId(buf, start+i+1)
		default:
			panic("invalid source type")
		}
	}
	if forkIndex == 0 && buf.Len() == 0 {
		return true, nil
	}
	err := f.writeForkIndex(buf, forkDim, forkIndex)
	return false, err
}

func (ForkId) writeForkIndex(buf *strings.Builder, forkDim, forkIndex int) error {
	if forkDim < 10 && forkIndex == 0 {
		_, err := buf.WriteString(defaultFork)
		return err
	}
	if _, err := buf.WriteString("fork"); err != nil {
		return err
	}
	return writePaddedIndex(buf, forkDim, forkIndex)
}

func writePaddedIndex(buf *strings.Builder, forkDim, forkIndex int) error {
	w := util.WidthForInt(forkDim - 1)
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
	if v == nil {
		return 0, nil
	}
	switch v := v.(type) {
	case *syntax.ArrayExp:
		return len(v.Value), nil
	case *syntax.NullExp:
		return 0, nil
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
	switch v := v.(type) {
	case fmt.GoStringer:
		return 0, fmt.Errorf("can't take length of %s %s",
			val.Type().String(), v.GoString())
	case fmt.Stringer:
		return 0, fmt.Errorf("can't take length of %s %s",
			val.Type().String(), v.String())
	default:
		return 0, fmt.Errorf("can't take length of %s",
			val.Type().String())
	}
}

func getUnknownKeys(v json.Marshaler) (mapKeyRange, error) {
	if v == nil {
		return nil, nil
	}
	switch v := v.(type) {
	case *syntax.MapExp:
		if len(v.Value) == 0 {
			return nil, nil
		}
		keys := make(mapKeyRange, 0, len(v.Value))
		for k := range v.Value {
			keys = append(keys, k)
		}
		return keys, nil
	case *syntax.NullExp:
		return nil, nil
	case MarshalerMap:
		if len(v) == 0 {
			return nil, nil
		}
		keys := make(mapKeyRange, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys, nil
	case LazyArgumentMap:
		if len(v) == 0 {
			return nil, nil
		}
		keys := make(mapKeyRange, 0, len(v))
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
		keys := make(mapKeyRange, 0, len(m))
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
		keys := make(mapKeyRange, 0, len(mk))
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
	return self.forkId.getUnmatchedForkParts(bNode)
}

func (forkId ForkId) getUnmatchedForkParts(bNode *Node) []*ForkSourcePart {
	if rl := forkId.UnmatchedParts(bNode.call.ForkRoots()); len(rl) > 0 {
		bNode = bNode.top.allNodes[rl[0].GetFqid()]
		parts := make([]*ForkSourcePart, 0, len(bNode.forks))
		for _, f := range bNode.forks {
			if forkId.Matches(f.forkId) {
				for _, part := range f.forkId {
					if part.Split.Call == rl[0].Call() {
						parts = append(parts, part)
					}
				}
			}
		}
		return parts
	}
	return nil
}

func (self *Fork) getDisabledSource(exp *syntax.DisabledExp) (syntax.Exp, error) {
	// If the source was disabled, we've got a null fork.
	ready, result, err := self.node.top.resolve(exp.Disabled,
		self.node.top.types.Get(syntax.TypeId{Tname: syntax.KindBool}),
		self.forkId, readSizeLimit)
	if err != nil {
		return nil, err
	}
	if !ready {
		panic("disabled binding " + exp.Disabled.GoString() + " was not ready")
	}
	if result == nil {
		return nil, nil
	}
	switch result := result.(type) {
	case *syntax.BoolExp:
		if result.Value {
			return nil, nil
		}
		return exp.Value, nil
	case *syntax.NullExp:
		return nil, nil
	case json.RawMessage:
		var b bool
		if err := json.Unmarshal(result, &b); err != nil {
			return nil, err
		}
		if b {
			return nil, nil
		}
		return exp.Value, nil
	default:
		panic(fmt.Sprintf("invalid type %T for disabled binding", result))
	}
}

func (self *Fork) expandForkPart(must bool,
	i int, part *ForkSourcePart,
	split *syntax.SplitExp, result []ForkId) ([]ForkId, error) {
	if self.metadata != nil {
		if s, _ := self.metadata.getState(); s == DisabledState {
			return nil, nil
		}
	}
	exp, err := split.BindingPath("", self.forkId.SourceIndexMap(),
		self.node.top.types)
	if err != nil {
		return nil, err
	}
	if sp, ok := exp.(*syntax.SplitExp); ok {
		return self.expandForkPartFromSource(must, i, part, split, sp.Source, result[len(result):])
	} else {
		return nil, nil
	}
}

func (self *Fork) expandForkPartFromSource(must bool,
	i int, part *ForkSourcePart,
	split *syntax.SplitExp,
	src syntax.MapCallSource, result []ForkId) ([]ForkId, error) {
	switch src := src.(type) {
	case *syntax.MergeExp:
		if src.ForkNode != nil {
			return self.expandForkFromRef(must,
				i, part, split, src.ForkNode, result)
		}
		return self.expandForkPartFromSource(must,
			i, part, split, src.MergeOver, result)
	case *syntax.MapCallSet:
		return self.expandForkPartFromSource(must,
			i, part, split, src.Master, result)
	case syntax.Exp:
		return self.expandForkPartFromExp(must,
			i, part, split, src, result)
	case *syntax.BoundReference:
		return self.expandForkFromRef(must,
			i, part, split, src.Exp, result)
	}
	return nil, fmt.Errorf(
		"invalid source %s for undetermined %s (computing forks for %s)",
		src.GoString(),
		split.GoString(),
		self.fqname)
}

func (self *Fork) expandForkPartFromExp(must bool, i int, part *ForkSourcePart,
	split *syntax.SplitExp, exp syntax.Exp, result []ForkId) ([]ForkId, error) {
	if exp == nil {
		part.Id = arrayIndexFork(0)
		self.updateId(self.forkId)
		self.writeDisable()
		return nil, nil
	}
	switch exp := exp.(type) {
	case *syntax.NullExp:
		part.Id = emptyFork{}
		self.updateId(self.forkId)
		self.writeDisable()
		return nil, nil
	case *syntax.RefExp:
		return self.expandForkFromRef(must, i, part, split, exp, result)
	case *syntax.MergeExp:
		if exp.ForkNode != nil {
			return self.expandForkFromRef(must,
				i, part, split, exp.ForkNode, result)
		}
		return self.expandForkPartFromSource(must,
			i, part, split, exp.MergeOver, result)
	case *syntax.DisabledExp:
		if d, ok := exp.Disabled.(*syntax.SplitExp); ok && d.Call == split.Call {
			return self.expandForkPartFromExp(must,
				i, part, split, d.Value, result)
		}
		ee, err := self.getDisabledSource(exp)
		if err != nil {
			return nil, err
		}
		return self.expandForkPartFromExp(must, i, part, split, ee, result)
	case *syntax.ArrayExp:
		return self.expandForkFromObj(i, part, split, exp, exp, result)
	case *syntax.MapExp:
		return self.expandForkFromObj(i, part, split, exp, exp, result)
	case *syntax.SplitExp:
		return self.expandForkPartFromSplit(must, i, part, split, exp, result)
	}
	return nil, fmt.Errorf(
		"invalid source %s for undetermined %s (computing forks for %s)",
		exp.GoString(),
		split.GoString(),
		self.fqname)
}

func (self *Fork) expandForkPartFromSplit(must bool, i int, part *ForkSourcePart,
	split *syntax.SplitExp, exp *syntax.SplitExp,
	result []ForkId) ([]ForkId, error) {
	if split.Call == exp.Call {
		return self.expandForkPartFromExp(must,
			i, part,
			split, exp.Value,
			result)
	}
	for _, id := range self.forkId {
		if id.Split.Call == exp.Call {
			if id.Id.IndexSource() != nil {
				if must {
					panic(exp.GoString() + " was not not resolved by " + id.GoString())
				}
				return result, nil
			}
			obj, err := self.expandForkSplitInnerPart(
				split, exp.Value, id.Id)
			if err != nil {
				return nil, err
			}
			if e, ok := obj.(syntax.Exp); ok {
				result, err = self.expandForkPartFromExp(must,
					i, part, split, e, result)
			} else {
				result, err = self.expandForkFromObj(i, part, split, obj, exp, result)
			}
			if err != nil {
				err = &forkResolutionError{
					Msg: "in split source " + exp.GoString(),
					Err: err,
				}
			}
			return result, err
		}
	}
	panic("call " + exp.Call.GoString() +
		" not found in " + self.forkId.GoString())
}

func (self *Fork) expandForkSplitInnerPart(
	split *syntax.SplitExp, exp syntax.Exp,
	index ForkIdPart) (json.Marshaler, error) {
	switch exp := exp.(type) {
	case *syntax.ArrayExp:
		return getElement(exp, index)
	case *syntax.MapExp:
		return getElement(exp, index)
	case *syntax.DisabledExp:
		ee, err := self.getDisabledSource(exp)
		if err != nil {
			return nil, err
		}
		if ee == nil {
			return nil, nil
		}
		return self.expandForkSplitInnerPart(
			split, exp.Value, index)
	case *syntax.SplitExp:
		for _, id := range self.forkId {
			if id.Split.Call == exp.Call {
				if id.Id.IndexSource() != nil {
					panic(exp.GoString() + " was not not resolved by " + id.GoString())
				}
				obj, err := self.expandForkSplitInnerPart(
					split, exp.Value, id.Id)
				if err != nil {
					return nil, err
				}
				return getElement(obj, index)
			}
		}
		panic("call " + exp.Call.GoString() + " not found in " + self.forkId.GoString())
	case *syntax.RefExp:
		ready, obj, err := self.node.top.resolveRef(exp, nil, self.forkId, readSizeLimit)
		if err != nil {
			return nil, err
		}
		if !ready {
			return nil, fmt.Errorf("%s is not ready", exp.GoString())
		}
		return getElement(obj, index)
	}
	return nil, fmt.Errorf(
		"invalid source %s for undetermined %s (computing forks for %s)",
		exp.GoString(),
		split.GoString(),
		self.fqname)
}

func (self *Fork) expandForkFromRef(must bool, i int,
	part *ForkSourcePart,
	split *syntax.SplitExp,
	ref *syntax.RefExp,
	result []ForkId) ([]ForkId, error) {
	bNode := self.node.top.allNodes[ref.Id]
	if bNode == nil {
		panic("invalid reference to " + ref.Id)
	}
	if bNode != self.node {
		bNode.expandForks(must)
	}
	if parts := self.getUnmatchedForkParts(bNode); len(parts) > 0 {
		flen := len(self.forkId)
		if flen == 0 {
			panic("No forks to expand parts into")
		}
		// Allocate the results in a single block, which we're going to
		// take slices out of, to make life easier for the GC.
		rep := make(ForkId, flen*(len(parts)-1))
		if cap(result) < len(parts)-1 {
			result = make([]ForkId, 0, len(parts)-1)
		}
		for j, part := range parts {
			if j == 0 {
				self.forkId[i] = part
				self.updateId(self.forkId)
			} else {
				result = append(result, rep[:flen:flen])
				rep = rep[flen:]
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
		return nil, fmt.Errorf(
			"fork %s matched %d out of %d forks of %s, need exactly 1",
			self.forkId.GoString(),
			len(matchedForks), len(bNode.forks), bNode.GetFQName())
	}
	ready, obj, err := matchedForks[0].resolveRef(ref, nil,
		bNode.call.Call().DecId, readSizeLimit)
	if err != nil {
		return nil, &elementError{
			element: "evaluating mapping source " + ref.GoString(),
			inner:   err,
		}
	}
	if !ready {
		return nil, nil
	}
	return self.expandForkFromObj(i, part, split, obj, ref, result)
}

func (self *Fork) expandForkFromObj(
	i int, part *ForkSourcePart,
	split *syntax.SplitExp,
	obj json.Marshaler,
	ref fmt.GoStringer,
	result []ForkId) ([]ForkId, error) {
	if obj == nil {
		if len(self.node.forks)-1 > self.index {
			pc := *part
			part = &pc
			self.forkId[i] = part
		}
		part.Id = emptyFork{}
		self.updateId(self.forkId)
		self.writeDisable()
		return nil, nil
	}
	switch split.Source.CallMode() {
	case syntax.ModeArrayCall:
		n, err := getUnknownLength(obj)
		if err != nil {
			return nil, err
		}
		part.Range = arrayLengthRange(n)
		if split.Source.KnownLength() && split.Source.ArrayLength() != n {
			return nil, &elementError{
				element: fmt.Sprint(
					"expected ", split.Source.ArrayLength(),
					"elements, but ", ref.GoString(),
					" had ", n),
			}
		}
		if n == 0 {
			if len(self.node.forks)-1 > self.index {
				pc := *part
				part = &pc
				self.forkId[i] = part
			}
			part.Id = emptyFork{}
			self.updateId(self.forkId)
			self.writeDisable()
			return nil, nil
		} else if n == 1 {
			if len(self.node.forks)-1 > self.index {
				pc := *part
				part = &pc
				self.forkId[i] = part
			}
			part.Id = arrayIndexFork(0)
			if self.forkId[i] != part {
				panic("not editing the right part")
			}
			self.updateId(self.forkId)
			return nil, nil
		}
		re := make([]ForkSourcePart, n)
		flen := len(self.forkId)
		if flen == 0 {
			flen = 1
		}
		rep := make(ForkId, flen*n)
		if cap(result) < n-1 {
			result = make([]ForkId, 0, n-1)
		}
		for j := 0; j < n; j++ {
			r := arrayIndexFork(j)
			re[j].Split = split
			re[j].Id = r
			re[j].Range = part.Range
			if j == 0 {
				self.forkId[i] = &re[j]
				self.updateId(self.forkId)
			} else {
				result = append(result, rep[0:flen:flen])
				rep = rep[flen:]
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
		part.Range = keys
		if split.Source.KnownLength() && len(split.Source.Keys()) != len(keys) {
			return nil, &elementError{
				element: fmt.Sprint(
					"expected ", len(split.Source.Keys()),
					"keys, but ", ref.GoString(),
					" had ", len(keys)),
			}
		}
		if len(keys) == 0 {
			if len(self.node.forks)-1 > self.index {
				pc := *part
				part = &pc
				self.forkId[i] = part
			}
			part.Id = emptyFork{}
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
		flen := len(self.forkId)
		if flen == 0 {
			flen = 1
		}
		rep := make(ForkId, flen*(len(keys)))
		if cap(result) < len(keys)-1 {
			result = make([]ForkId, 0, len(keys)-1)
		}
		for j := 0; j < len(keys); j++ {
			r := mapKeyFork(keys[j])
			re[j].Split = split
			re[j].Id = r
			re[j].Range = part.Range
			if j == 0 {
				self.forkId[i] = &re[j]
				self.updateId(self.forkId)
			} else {
				result = append(result, rep[0:flen:flen])
				rep = rep[flen:]
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
	panic("invalid fork mode " + split.Source.CallMode().String())
}

func (self *Fork) expand(must bool) ([]ForkId, error) {
	if len(self.forkId) == 0 && len(self.node.call.ForkRoots()) > 0 {
		if split := self.node.call.Split(); split != nil {
			if src := split.Source; src != nil && !src.KnownLength() {
				part := &ForkSourcePart{
					Split: split,
					Id:    undeterminedFork{},
				}
				self.forkId = ForkId{part}
				ids, err := self.expandForkPart(must, 0, part, split,
					self.node.forkIds.List)
				if err != nil {
					self.forkId = nil
				}
				return ids, err
			}
		}
	}
	for i := 0; i < len(self.forkId); i++ {
		part := self.forkId[i]
		if part.Id.IndexSource() != nil {
			if ids, err := self.expandForkPart(must, i, part, part.Split,
				self.node.forkIds.List); err != nil ||
				len(ids) > 0 {
				return ids, err
			}
		}
	}
	return nil, nil
}
