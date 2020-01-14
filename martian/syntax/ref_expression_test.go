// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Reference expressions refer to outputs of stages or pipelines, or inputs
// to pipelines.

package syntax

import (
	"testing"
)

func TestRefExp_ExpandMerges(t *testing.T) {
	src := ArrayExp{
		Value: []Exp{
			&IntExp{Value: 1},
			&IntExp{Value: 1},
			&IntExp{Value: 1},
		},
	}
	ref := RefExp{
		Kind:      KindCall,
		Id:        "FOO",
		OutputId:  "BAR",
		MergeOver: []MapCallSource{&src},
	}
	merged := ref.ExpandMerges()
	if arr, ok := merged.(*ArrayExp); !ok {
		t.Errorf("wrong type %T", merged)
	} else if len(arr.Value) != 3 {
		t.Errorf("wrong length %d != 3", len(arr.Value))
	} else {
		for i, v := range arr.Value {
			if r, ok := v.(*RefExp); !ok {
				t.Errorf("element %d was a %T", i, v)
			} else {
				if len(r.MergeOver) > 0 {
					t.Errorf("%d merges", len(r.MergeOver))
				}
				if len(r.ForkIndex) == 0 {
					t.Errorf("no forks")
				}
			}
		}
	}
}
