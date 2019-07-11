package core

import (
	"fmt"
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

		// The string ID for a fork.
		//
		// For a map-type fork, this will be "fork.<key>".  For an array-type
		// fork it will be "forkN" where N is the array index.
		forkString() string
	}

	mapKeyFork string

	arrayIndexFork int

	ForkSourcePart struct {
		Source syntax.Exp
		Id     ForkIdPart
	}

	// A set of expressions and the corresponding index values for each one,
	// which identifies a specific fork.
	ForkId []*ForkSourcePart

	expSetBuilder struct {
		Exps []syntax.Exp
		set  map[syntax.Exp]struct{}
	}
)

func (s *expSetBuilder) Add(exp syntax.Exp) {
	if s.Exps == nil {
		s.Exps = []syntax.Exp{exp}
	} else if len(s.Exps) == 0 {
		s.Exps = append(s.Exps, exp)
	}
	if s.set == nil {
		s.set = make(map[syntax.Exp]struct{}, 1)
		for _, e := range s.Exps {
			s.set[e] = struct{}{}
		}
	}
	if _, ok := s.set[exp]; !ok {
		s.set[exp] = struct{}{}
		s.Exps = append(s.Exps, exp)
	}
}

func (s *expSetBuilder) AddMany(exp []syntax.Exp) {
	if len(exp) == 0 {
		return
	}
	if len(s.Exps) == 0 {
		s.Exps = exp
	}
	if s.set == nil {
		s.set = make(map[syntax.Exp]struct{}, len(s.Exps)+len(exp))
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
		if n := n.getNode(); n != nil && len(n.forkRoots) > 0 {
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

func makeForkIdParts(src syntax.Exp) []*ForkSourcePart {
	switch src := src.(type) {
	case *syntax.SweepExp:
		re := make([]ForkSourcePart, len(src.Value))
		result := make([]*ForkSourcePart, len(src.Value))
		for i := range result {
			r := arrayIndexFork(i)
			re[i].Source = src
			re[i].Id = &r
			result[i] = &re[i]
		}
		return result
	case *syntax.ArrayExp:
		re := make([]ForkSourcePart, len(src.Value))
		result := make([]*ForkSourcePart, len(src.Value))
		for i := range result {
			r := arrayIndexFork(i)
			re[i].Source = src
			re[i].Id = &r
			result[i] = &re[i]
		}
		return result
	case *syntax.MapExp:
		re := make([]ForkSourcePart, 0, len(src.Value))
		result := make([]*ForkSourcePart, 0, len(src.Value))
		for k := range result {
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
	default:
		return []*ForkSourcePart{
			&ForkSourcePart{
				Source: src,
			},
		}
	}
}

// Computes the cartesian product of possible values for ForkIds.
func MakeForkIds(srcs []syntax.Exp) []ForkId {
	if len(srcs) == 0 {
		return nil
	}
	suffixes := MakeForkIds(srcs[1:])
	these := makeForkIdParts(srcs[0])
	var result []ForkId
	if len(suffixes) == 0 {
		result = make([]ForkId, len(these))
		for i, part := range these {
			result[i] = ForkId{part}
		}
	} else {
		result = make([]ForkId, 0, len(these)*len(suffixes))
		for _, suffix := range suffixes {
			for _, part := range these {
				id := make(ForkId, 1, len(srcs))
				id[0] = part
				result = append(result, append(id, suffix...))
			}
		}
	}
	return result
}

// Get the ForkId from an upstream stage with the given fork sources which
// corresponds to this fork.
//
// This ForkId must include every Source contained in upstream.
func (f ForkId) Match(upstream []syntax.Exp) ForkId {
	if len(upstream) == 0 {
		return nil
	}
	result := make(ForkId, len(upstream))
	for i, src := range upstream {
		if r, err := f.matchPart(src); err != nil {
			panic(fmt.Sprintf("resolving fork part %d from %v: %v",
				i, src.GoString(), err))
		} else {
			result[i] = r
		}
	}
	return result
}

func (f ForkId) matchPart(src syntax.Exp) (*ForkSourcePart, error) {
	for _, part := range f {
		if part.Source == src {
			return part, nil
		}
	}
	return nil, fmt.Errorf("no match found")
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
	switch id := f.Id.(type) {
	case *arrayIndexFork:
		oid, ok := o.Id.(*arrayIndexFork)
		return ok && *oid == *id
	case mapKeyFork:
		oid, ok := o.Id.(mapKeyFork)
		return ok && oid == id
	default:
		panic("bad id type")
	}
}

// String returns "fork.<key>".
func (k mapKeyFork) String() string {
	return k.forkString()
}

// forkString returns "fork.<key>".
func (k mapKeyFork) forkString() string {
	return "fork." + string(k)
}

// String returns "fork<i>".
func (i *arrayIndexFork) String() string {
	return i.forkString()
}

// forkString returns "fork<i>".
func (i *arrayIndexFork) forkString() string {
	if i == nil {
		return defaultFork
	}
	j := int(*i)
	if j == 0 {
		return defaultFork
	}
	var buf strings.Builder
	if j < 10 {
		buf.Grow(5)
	} else {
		buf.Grow(6)
	}
	buf.WriteString("fork")
	buf.WriteString(strconv.Itoa(j))
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
// avoid memory allocation.
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
	switch sid0 := f.Id.(type) {
	case *arrayIndexFork:
		i := int(*sid0)
		if i < 0 {
			return defaultFork, fmt.Errorf(
				"invalid fork index %d", i)
		}
		var alen int
		switch s0 := f.Source.(type) {
		case *syntax.SweepExp:
			alen = len(s0.Value)
			if i >= alen {
				return defaultFork, fmt.Errorf(
					"len(sweep) == %d <= index %d",
					len(s0.Value), i)
			}
		case *syntax.ArrayExp:
			alen = len(s0.Value)
			if i >= alen {
				return defaultFork, fmt.Errorf(
					"len(array) == %d <= index %d",
					len(s0.Value), i)
			}
		case *syntax.MapExp:
			return defaultFork, fmt.Errorf(
				"can't get integer index of map source")
		case *syntax.RefExp:
			return defaultFork, fmt.Errorf(
				"fork is not fully resolved")
		default:
			return defaultFork, fmt.Errorf(
				"can't index into fork source type %T", s0)
		}
		if i == 0 {
			return defaultFork, nil
		}
		var buf strings.Builder
		w := util.WidthForInt(alen)
		buf.Grow(4 + w)
		if _, err := buf.WriteString("fork"); err != nil {
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
	case mapKeyFork:
		switch s0 := f.Source.(type) {
		case *syntax.MapExp:
			if _, ok := s0.Value[string(sid0)]; !ok {
				return defaultFork, fmt.Errorf("no key %q in source",
					string(sid0))
			}
			return sid0.forkString(), nil
		case *syntax.ArrayExp, *syntax.SweepExp:
			return defaultFork, fmt.Errorf(
				"can't get map key from array source")
		case *syntax.RefExp:
			return defaultFork, fmt.Errorf(
				"fork is not fully resolved")
		default:
			return defaultFork, fmt.Errorf(
				"can't get map index from fork source type %T", s0)
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
		switch id0 := part.Id.(type) {
		case *arrayIndexFork:
			j := int(*id0)
			if j < 0 {
				return true, fmt.Errorf("invalid fork index %d", j)
			}
			forkIndex += forkDim * j
			switch s0 := part.Source.(type) {
			case *syntax.SweepExp:
				forkDim *= len(s0.Value)
				if j >= len(s0.Value) {
					return true, fmt.Errorf("len(sweep) == %d <= index %d",
						len(s0.Value), j)
				}
			case *syntax.ArrayExp:
				forkDim *= len(s0.Value)
				if j >= len(s0.Value) {
					return true, fmt.Errorf("len(sweep) == %d <= index %d",
						len(s0.Value), j)
				}
			case *syntax.RefExp:
				return true, fmt.Errorf(
					"fork is not fully resolved")
			case *syntax.MapExp:
				return true, fmt.Errorf(
					"can't get integer interface into map type")
			default:
				return true, fmt.Errorf(
					"can't index into fork source type %T", s0)
			}
		case mapKeyFork:
			if i == 0 {
				// Write this fork id.
				switch s0 := part.Source.(type) {
				case *syntax.MapExp:
					if _, ok := s0.Value[string(id0)]; !ok {
						return true, fmt.Errorf("no key %q in source",
							string(id0))
					}
				case *syntax.ArrayExp, *syntax.SweepExp:
					return true, fmt.Errorf("can't get map key from array source")
				case *syntax.RefExp:
					return true, fmt.Errorf(
						"fork is not fully resolved")
				default:
					return true, fmt.Errorf(
						"can't get map index from fork source type %T", s0)
				}
				if _, err := buf.WriteString("fork."); err != nil {
					return true, err
				}
				if _, err := buf.WriteString(string(id0)); err != nil {
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
