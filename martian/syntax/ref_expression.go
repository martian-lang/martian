// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Reference expressions refer to outputs of stages or pipelines, or inputs
// to pipelines.

package syntax

import (
	"fmt"
	"strconv"
)

type (
	// A RefExp represents a value that is a reference to a pipeline input or
	// a call output.
	//
	// When resolving a reference, first simplify MergeOver and OutputIndex, by
	// combining the first element of each into a ForkIndex component.
	//
	// Next, compute the merge dimensions MergeOver. Each dimension in
	// MergeOver, in order, will produce an array or typed map of references.
	// That is, if MergeOver's first element is an array of length 2, then the
	// result will be an array of lenth 2, where each element is the same as the
	// original reference, but with that dimension removed from MergeOver and
	// added to ForkIndex (with the appropriate index).
	//
	// Next, evaluate the OutputId to select the subset fields from strutures.
	// These project over arrays and typed maps, so if one has a structure
	//
	//   {
	//       a: {
	//           "b": {
	//               c: [
	//                   {
	//                       d: 1,
	//                   },
	//               ],
	//           }
	//       }
	//   }
	//
	// then a reference STRUCT.a.c.d would resolve to
	//
	//   {
	//       "b": [1]
	//   }
	//
	// If any stage outputs are encountered along the way, select the fork of
	// that stage indicated by ForkIndex.  Exactly one fork should match - every
	// forking dimension used by any subreference of a reference must either
	// be in ForkIndex or MergeOver (which eventually resolves to ForkIndex).
	//
	// Finally, take any remaining elements from OutputIndex and use them to
	// select the array elements or map indicies from the resulting object.  If
	// the index is not a compile-time constant, use the upstream fork's
	// ForkIndex[idx.IndexSource()]
	RefExp struct {
		Node AstNode
		Kind ExpKind

		// For KindSelf, the name of the pipeline input parameter.  For
		// KindCall, the call's Id within the pipeline
		Id string

		// The binding path through the referred-to call or input.
		OutputId string

		// The dimensions over which to combine the reference path's forks.
		//
		// For example, if pipeline P is mapped over a map input1, and
		// stage S uses input1 and also maps over an array input2, then if
		// ForkIndex is empty, MergeOver must contain both input1's source and
		// input2's source (in that order), and the reference is to a map of
		// arrays of S structures.  If ForkIndex contains an index into input1,
		// then MergeOver contains input2 and the result of the reference would
		// be an array of S.
		MergeOver []MapCallSource `json:"merge_over,omitempty"`

		// Which fork of a mapped call this reference refers to.  This is not
		// set when compiling mro, as mro does not have syntax for indexing into
		// collections.  Instead, it is set when resolving a call graph.
		//
		// If a stage forks on dimensions not mentioned here, those dimensions
		// must be present in MergeOver.
		ForkIndex map[MapCallSource]CollectionIndex `json:"fork_index,omitempty"`

		// If a reference into a stage's outputs results in an array or map,
		// these indicies are used to select from that result.  These indicies
		// are applied after selecting one or more specific forks of a call
		// with ForkIndex, but before combining the results if more than one
		// fork is being used.
		//
		// That is, if a stage S has outputs
		//
		//  {
		//      a: {
		//          "c": {
		//              b: [1,3,4]
		//          }
		//      }
		//  }
		//
		// Then the result of S.a.b is a map of arrays {"c": [1,3,4]}.  The
		// OutputIndex must proceed in that nesting order.  So an output index
		// ["c"] would return [1,3,4] and ["c",1] would return 3, but [1,"c"]
		// would be an error.
		OutputIndex []CollectionIndex `json:"output_index,omitempty"`
	}

	// An index into an array or map.
	CollectionIndex interface {
		// Either ModeArrayCall or ModeMapCall
		Mode() CallMode

		// The map key to use.
		// Panics if IndexSource() is non-nil or Mode() != ModeMapCall.
		MapKey() string

		// The map key to use.
		// Panics if IndexSource() is non-nil or Mode() != ModeArrayCall.
		ArrayIndex() int

		// If non-nil, then this index is a placeholder.  The corresponding fork
		// index of the stage or RuntimeMap/RuntimeArray doing the lookup should
		// be used in its place.
		IndexSource() MapCallSource
	}

	mapKeyIndex string
	arrayIndex  int

	unknownIndex struct {
		src MapCallSource
	}
)

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) File() *SourceFile { return s.Node.Loc.File }
func (s *RefExp) Line() int         { return s.Node.Loc.Line }
func (s *RefExp) getKind() ExpKind  { return s.Kind }

func (s *RefExp) inheritComments() bool { return false }
func (s *RefExp) getSubnodes() []AstNodable {
	return nil
}

func (*RefExp) HasRef() bool {
	return true
}
func (*RefExp) HasSplit() bool {
	return false
}

// Combine OutputIndex keys with MergeOver keys to select forks.
func (ref *RefExp) Simplify() {
	for len(ref.MergeOver) > 0 && len(ref.OutputIndex) > 0 {
		if ref.ForkIndex == nil {
			ref.ForkIndex = make(map[MapCallSource]CollectionIndex, 1)
		}
		if ref.MergeOver[0].CallMode() != ref.OutputIndex[0].Mode() {
			panic("merging over " + ref.MergeOver[0].GoString() +
				" but with index of type " + ref.OutputIndex[0].Mode().String())
		}
		ref.ForkIndex[ref.MergeOver[0]] = ref.OutputIndex[0]
		ref.MergeOver = ref.MergeOver[1:]
		ref.OutputIndex = ref.OutputIndex[1:]
	}
}

func (ref *RefExp) updateForks(fork map[MapCallSource]CollectionIndex) (*RefExp, error) {
	result := ref
	var errs ErrorList
	for src, j := range fork {
		if i, ok := ref.ForkIndex[src]; !ok {
			if ref == result {
				r2 := *ref
				result = &r2
				result.ForkIndex = make(map[MapCallSource]CollectionIndex, len(fork)+1)
				for k, v := range ref.ForkIndex {
					result.ForkIndex[k] = v
				}
			}
			result.ForkIndex[src] = j
		} else if i.IndexSource() != nil && j.IndexSource() == nil {
			if result == ref {
				newFork := make(map[MapCallSource]CollectionIndex, len(ref.ForkIndex))
				for k, v := range ref.ForkIndex {
					newFork[k] = v
				}
				r2 := *ref
				r2.ForkIndex = newFork
				result = &r2
			}
			result.ForkIndex[src] = j
		} else if i != j {
			errs = append(errs, &bindingError{
				Msg: fmt.Sprint("inconsistent index ", i, " vs ", j),
			})
		}
	}
	return result, errs.If()
}

// Recursively convert merged references into arrays or maps of keyed
// references.  This is not correct for references which have not yet been
// resolved to a stage output.
func (ref *RefExp) ExpandMerges() Exp {
	ref.Simplify()
	if len(ref.MergeOver) > 0 {
		if src := ref.MergeOver[0]; src.KnownLength() {
			switch src.CallMode() {
			case ModeArrayCall:
				newResult := ArrayExp{
					valExp: valExp{Node: ref.Node},
					Value:  make([]Exp, src.ArrayLength()),
				}
				for i := range newResult.Value {
					fork := make(map[MapCallSource]CollectionIndex, len(ref.ForkIndex)+1)
					for k, v := range ref.ForkIndex {
						fork[k] = v
					}
					fork[src] = arrayIndex(i)
					newRef := *ref
					newRef.MergeOver = ref.MergeOver[1:]
					newRef.ForkIndex = fork
					newResult.Value[i] = newRef.ExpandMerges()
				}
				return &newResult
			case ModeMapCall:
				newResult := MapExp{
					valExp: valExp{Node: ref.Node},
					Value:  make(map[string]Exp, len(src.Keys())),
				}
				for i := range src.Keys() {
					fork := make(map[MapCallSource]CollectionIndex, len(ref.ForkIndex)+1)
					for k, v := range ref.ForkIndex {
						fork[k] = v
					}
					ii := mapKeyIndex(i)
					fork[src] = ii
					newRef := *ref
					newRef.MergeOver = ref.MergeOver[1:]
					newRef.ForkIndex = fork
					newResult.Value[i] = newRef.ExpandMerges()
				}
				return &newResult
			}
		}
	}
	return ref
}

// CallMode Returns the call mode for a call which depends on this source.
func (r *RefExp) CallMode() CallMode {
	if r == nil || len(r.MergeOver) == 0 {
		return ModeSingleCall
	}
	if r.MergeOver[0] == r {
		panic("self-split")
	}
	return r.MergeOver[0].CallMode()
}

// KnownLength returns false, as stage output lengths are never known.
func (r *RefExp) KnownLength() bool {
	if r == nil || len(r.MergeOver) == 0 {
		return false
	}
	return r.MergeOver[0].KnownLength()
}

// ArrayLength returns nil, as stage output lengths are never known.
func (r *RefExp) ArrayLength() int {
	if r == nil || len(r.MergeOver) == 0 {
		return -1
	}
	return r.MergeOver[0].ArrayLength()
}

// Keys returns nil, as stage output keys are never known.
func (r *RefExp) Keys() map[string]Exp {
	if r == nil || len(r.MergeOver) == 0 {
		return nil
	}
	return r.MergeOver[0].Keys()
}

type refMapResolver interface {
	resolveMapSource(CallMode) MapCallSource
}

func (r *RefExp) resolveMapSource(mode CallMode) MapCallSource {
	if len(r.MergeOver) < 1 {
		if r.CallMode() != mode {
			return &ReferenceMappingSource{
				Ref:  r,
				Mode: mode,
			}
		}
		return r
	}
	if rr, ok := r.MergeOver[0].(refMapResolver); ok {
		m := rr.resolveMapSource(mode)
		if m.CallMode() != mode {
			return &ReferenceMappingSource{
				Ref:  m.(*RefExp),
				Mode: mode,
			}
		}
		return m
	}
	if r.MergeOver[0] == nil {
		panic("nil source in ref")
	}
	if r.MergeOver[0].CallMode() != mode {
		return &ReferenceMappingSource{
			Ref:  r.MergeOver[0].(*RefExp),
			Mode: mode,
		}
	}
	return r.MergeOver[0]
}

// A wrapper for a reference to an array or map output element which can be
// used as a mapping source.
type ReferenceMappingSource struct {
	Ref  *RefExp  `json:"ref"`
	Mode CallMode `json:"kind"`
}

// CallMode Returns the call mode for a call which depends on this source.
func (r *ReferenceMappingSource) CallMode() CallMode {
	return r.Mode
}

// KnownLength returns false, as stage output lengths are never known.
func (r *ReferenceMappingSource) KnownLength() bool {
	return r.Ref.KnownLength()
}

// ArrayLength returns nil, as stage output lengths are never known.
func (r *ReferenceMappingSource) ArrayLength() int {
	return r.Ref.ArrayLength()
}

// Keys returns nil, as stage output keys are never known.
func (r *ReferenceMappingSource) Keys() map[string]Exp {
	return r.Ref.Keys()
}

func (r *ReferenceMappingSource) GoString() string {
	return r.Mode.String() + " of " + r.Ref.GoString()
}

func (r *ReferenceMappingSource) resolveMapSource(mode CallMode) MapCallSource {
	if mode != r.Mode {
		panic("incompatible modes")
	}
	if len(r.Ref.MergeOver) < 1 {
		return r
	}
	if rr, ok := r.Ref.MergeOver[0].(refMapResolver); ok {
		if m := rr.resolveMapSource(mode); m == r.Ref {
			return r
		} else if m.CallMode() != mode {
			return &ReferenceMappingSource{
				Ref:  m.(*RefExp),
				Mode: mode,
			}
		}
	}
	if r.Ref.MergeOver[0] == nil {
		panic("nil source in ref")
	}
	if r.Ref.MergeOver[0].CallMode() != mode {
		return &ReferenceMappingSource{
			Ref:  r.Ref.MergeOver[0].(*RefExp),
			Mode: mode,
		}
	}
	return r.Ref.MergeOver[0]
}

func (mapKeyIndex) Mode() CallMode {
	return ModeMapCall
}
func (i mapKeyIndex) MapKey() string {
	return string(i)
}
func (mapKeyIndex) ArrayIndex() int {
	panic("map key can't be used as array index")
}
func (mapKeyIndex) IndexSource() MapCallSource {
	return nil
}
func (k mapKeyIndex) String() string {
	return string(k)
}

func (arrayIndex) Mode() CallMode {
	return ModeArrayCall
}
func (arrayIndex) MapKey() string {
	panic("array index can't be used as a map key")
}
func (i arrayIndex) ArrayIndex() int {
	return int(i)
}
func (arrayIndex) IndexSource() MapCallSource {
	return nil
}
func (i arrayIndex) String() string {
	return strconv.Itoa(int(i))
}

func (i unknownIndex) Mode() CallMode {
	return i.src.CallMode()
}
func (unknownIndex) MapKey() string {
	panic("key unknown")
}
func (unknownIndex) ArrayIndex() int {
	panic("array index unknown")
}
func (i unknownIndex) IndexSource() MapCallSource {
	return i.src
}

func (i unknownIndex) MarshalText() ([]byte, error) {
	return append([]byte("unknown "), i.src.CallMode().String()...), nil
}
