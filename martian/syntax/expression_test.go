// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"testing"
)

func TestHasSplit(t *testing.T) {
	exp := ArrayExp{
		Value: []Exp{
			new(IntExp),
			new(MapExp),
			&MapExp{
				Value: map[string]Exp{
					"foo": new(IntExp),
				},
			},
			&MapExp{
				Value: map[string]Exp{
					"foo": new(IntExp),
					"bar": &SplitExp{
						Value: &ArrayExp{
							Value: []Exp{
								new(IntExp),
								new(FloatExp),
							},
						},
					},
				},
			},
			new(ArrayExp),
			&ArrayExp{
				Value: []Exp{new(IntExp)},
			},
			&SplitExp{
				Value: &ArrayExp{
					Value: []Exp{
						new(IntExp),
						new(FloatExp),
					},
				},
			},
			new(RefExp),
		},
	}
	if !exp.HasSplit() {
		t.Error("expected true")
	}
	for i, e := range []bool{
		false, // int
		false, // empty map
		false, // map
		true,  // map
		false, // empty array
		false, // array
		true,  // array
		false, // ref
	} {
		if exp.Value[i].HasSplit() != e {
			t.Errorf("expected %v for value %s",
				e, exp.Value[i].GoString())
		}
	}
}

func TestFindRefs(t *testing.T) {
	exp := ArrayExp{
		Value: []Exp{
			new(IntExp),
			new(MapExp),
			&MapExp{
				Value: map[string]Exp{
					"foo": new(IntExp),
				},
			},
			&MapExp{
				Value: map[string]Exp{
					"foo": new(IntExp),
					"bar": &RefExp{Id: "bar"},
				},
			},
			new(ArrayExp),
			&ArrayExp{
				Value: []Exp{new(IntExp)},
			},
			&RefExp{Id: "foo"},
		},
	}
	refs := exp.FindRefs()
	if len(refs) != 2 {
		t.Errorf("expected 2 refs, found %d", len(refs))
	} else {
		for i, e := range []string{
			"bar",
			"foo",
		} {
			if refs[i].Id != e {
				t.Errorf("expected %s, got %s", e, refs[i].Id)
			}
		}
	}
}
