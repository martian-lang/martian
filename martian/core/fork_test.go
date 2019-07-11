package core

import (
	"fmt"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestExpSetBuilderAddBindings(t *testing.T) {
	var finder expSetBuilder
	exps := []syntax.Exp{
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.BoolExp{Value: true},
				&syntax.BoolExp{Value: false},
			},
		},
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.StringExp{Value: "bar"},
				&syntax.StringExp{Value: "baz"},
			},
		},
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.IntExp{Value: 2},
				&syntax.IntExp{Value: 3},
			},
		},
	}
	finder.AddBindings(map[string]*syntax.ResolvedBinding{
		"a": &syntax.ResolvedBinding{
			Exp: &syntax.StringExp{Value: `foo`},
		},
		"b": &syntax.ResolvedBinding{
			Exp: exps[0],
		},
		"c": &syntax.ResolvedBinding{
			Exp: &syntax.ArrayExp{
				Value: []syntax.Exp{
					&syntax.StringExp{Value: "foo"},
					exps[1],
				},
			},
		},
		"d": &syntax.ResolvedBinding{
			Exp: &syntax.MapExp{
				Value: map[string]syntax.Exp{
					"1": &syntax.IntExp{Value: 1},
					"2": exps[2],
				},
			},
		},
	})
	if len(finder.Exps) != len(exps) {
		t.Errorf("Expected %d roots, got %d", len(exps), len(finder.Exps))
	}
	for i, e := range exps {
		if finder.Exps[i] != e {
			t.Errorf("Expected %s, got %s", e.GoString(), finder.Exps[i].GoString())
		}
	}
}

func TestExpSetBuilderAddMany(t *testing.T) {
	var finder expSetBuilder
	exps := []syntax.Exp{
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.BoolExp{Value: true},
				&syntax.BoolExp{Value: false},
			},
		},
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.StringExp{Value: "bar"},
				&syntax.StringExp{Value: "baz"},
			},
		},
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.IntExp{Value: 2},
				&syntax.IntExp{Value: 3},
			},
		},
	}

	finder.AddMany(exps)
	if len(finder.Exps) != len(exps) {
		t.Errorf("Expected %d roots, got %d", len(exps), len(finder.Exps))
	}
	for i, e := range exps {
		if finder.Exps[i] != e {
			t.Errorf("Expected %s, got %s", e.GoString(), finder.Exps[i].GoString())
		}
	}
}

func TestMakeForkIds(t *testing.T) {
	exps := []syntax.Exp{
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.BoolExp{Value: true},
				&syntax.BoolExp{Value: false},
			},
		},
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.StringExp{Value: "bar"},
				&syntax.StringExp{Value: "baz"},
			},
		},
		&syntax.SweepExp{
			Value: []syntax.Exp{
				&syntax.IntExp{Value: 2},
				&syntax.IntExp{Value: 3},
			},
		},
	}
	ids := MakeForkIds(exps)
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
	if len(ids) != 8 {
		t.Errorf("expected %d ids, got %d", len(expect), len(ids))
	}
	for i, id := range ids {
		if !id.Equal(expect[i]) {
			t.Errorf("expected %v, got %v", expect[i], id)
		}
	}
	for i, id := range ids {
		if s, err := id.ForkIdString(); err != nil {
			t.Error(err)
		} else if s != fmt.Sprintf("fork%d", i) {
			t.Errorf("expected fork%d, got %s", i, s)
		}
	}
}
