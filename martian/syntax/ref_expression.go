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
	// result will be an array of length 2, where each element is the same as the
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
	// select the array elements or map indices from the resulting object.  If
	// the index is not a compile-time constant, use the upstream fork's
	// ForkIndex[idx.IndexSource()].
	RefExp struct {
		Node AstNode
		Kind ExpKind

		// For KindSelf, the name of the pipeline input parameter.  For
		// KindCall, the call's Id within the pipeline
		Id string

		// The binding path through the referred-to call or input.
		OutputId string

		// Which fork of a mapped call this reference refers to.  This is not
		// set when compiling mro, as mro does not have syntax for indexing into
		// collections.  Instead, it is set when resolving a call graph.
		//
		// Every dimension over which this stage is forked should be included
		// in this map.
		Forks map[*CallStm]CollectionIndex `json:"fork_index,omitempty"`
	}

	// An index into an array or map.
	CollectionIndex interface {
		fmt.GoStringer

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

// Implementation for Exp

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

func (ref *RefExp) updateForks(fork map[*CallStm]CollectionIndex) (*RefExp, error) {
	result := ref
	makeCopy := func() {
		if result == ref {
			newFork := make(map[*CallStm]CollectionIndex, len(ref.Forks))
			for k, v := range ref.Forks {
				newFork[k] = v
			}
			r2 := *ref
			r2.Forks = newFork
			result = &r2
		}
	}
	var errs ErrorList
	if len(ref.Forks) > 0 {
		for src, j := range fork {
			if i, ok := ref.Forks[src]; ok {
				if i.IndexSource() != nil {
					if j.IndexSource() == nil {
						makeCopy()
						result.Forks[src] = j
					} else if m, err := MergeMapCallSources(i.IndexSource(), j.IndexSource()); err != nil {
						errs = append(errs, &bindingError{
							Msg: "merge dimension " + src.GoString(),
							Err: err,
						})
					} else if _, ok := i.(unknownIndex); ok && m != i.IndexSource() {
						makeCopy()
						result.Forks[src] = unknownIndex{src: m}
					}
				} else if j.IndexSource() == nil && !indexEqual(i, j) {
					errs = append(errs, &bindingError{
						Msg: fmt.Sprint("inconsistent index ", i.GoString(), " vs ", j.GoString()),
					})
				}
			}
		}
	}
	return result, errs.If()
}

// Implementation for MapCallSource

func (*RefExp) CallMode() CallMode {
	return ModeUnknownMapCall
}

func (*RefExp) KnownLength() bool {
	return false
}

func (*RefExp) ArrayLength() int {
	return -1
}

func (*RefExp) Keys() map[string]Exp {
	return nil
}

// Index stuff

func indexEqual(i, j CollectionIndex) bool {
	if i == j {
		return true
	}
	if s := i.IndexSource(); s != j.IndexSource() || s != nil {
		return false
	}
	if m := i.Mode(); m != j.Mode() {
		return false
	} else if m == ModeArrayCall {
		return i.ArrayIndex() == j.ArrayIndex()
	} else if m == ModeMapCall {
		return i.MapKey() == j.MapKey()
	} else {
		return true
	}
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
func (k mapKeyIndex) GoString() string {
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
func (i arrayIndex) GoString() string {
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
func (i unknownIndex) GoString() string {
	return "unknown " + i.src.CallMode().String()
}
