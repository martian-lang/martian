package core

import (
	"fmt"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestMakeForkIds(t *testing.T) {
	exps := []syntax.MapCallSource{
		&syntax.ArrayExp{
			Value: []syntax.Exp{
				&syntax.BoolExp{Value: true},
				&syntax.BoolExp{Value: false},
			},
		},
		&syntax.ArrayExp{
			Value: []syntax.Exp{
				&syntax.StringExp{Value: "bar"},
				&syntax.StringExp{Value: "baz"},
			},
		},
		&syntax.ArrayExp{
			Value: []syntax.Exp{
				&syntax.IntExp{Value: 2},
				&syntax.IntExp{Value: 3},
			},
		},
	}
	var ids ForkIdSet
	ids.MakeForkIds(nil, exps)
	mkId := func(i, j, k arrayIndexFork) ForkId {
		return ForkId{
			&ForkSourcePart{
				Source: exps[0],
				Id:     &i,
			},
			&ForkSourcePart{
				Source: exps[1],
				Id:     &j,
			},
			&ForkSourcePart{
				Source: exps[2],
				Id:     &k,
			},
		}
	}
	expect := []ForkId{
		mkId(0, 0, 0),
		mkId(1, 0, 0),
		mkId(0, 1, 0),
		mkId(1, 1, 0),
		mkId(0, 0, 1),
		mkId(1, 0, 1),
		mkId(0, 1, 1),
		mkId(1, 1, 1),
	}
	if len(ids.List) != 8 {
		t.Errorf("expected %d ids, got %d", len(expect), len(ids.List))
	}
	for i, id := range ids.List {
		if !id.Equal(expect[i]) {
			t.Errorf("expected %v, got %v", expect[i].GoString(), id.GoString())
		}
	}
	for i, id := range ids.List {
		if s, err := id.ForkIdString(); err != nil {
			t.Error(err)
		} else if s != fmt.Sprintf("fork%d", i) {
			t.Errorf("expected fork%d, got %s", i, s)
		}
	}
}
