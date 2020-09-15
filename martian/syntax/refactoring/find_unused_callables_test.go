package refactoring

import (
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestFindUnusedCallables(t *testing.T) {
	var parser syntax.Parser

	_, _, ast, err := parser.Compile("testdata/resolve_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	callables := FindUnusedCallables(StringSet{"POINT_MAPPER": struct{}{}},
		[]*syntax.Ast{ast})

	if len(callables) > 0 {
		t.Errorf("Expected nothing, got %d", len(callables))
	}

	callables = FindUnusedCallables(StringSet{"POINT_PIPE": struct{}{}},
		[]*syntax.Ast{ast})

	if len(callables) != 3 {
		t.Fatalf("%d != 3", len(callables))
	}
	if callables[0].GetId() != "POINT_MAPPER" &&
		callables[1].GetId() != "POINT_MAPPER" &&
		callables[2].GetId() != "POINT_MAPPER" {
		t.Error("Expected POINT_MAPPER to be unused")
	}
	if callables[0].GetId() != "POINT_USER" &&
		callables[1].GetId() != "POINT_USER" &&
		callables[2].GetId() != "POINT_USER" {
		t.Error("Expected POINT_USER to be unused")
	}
}
