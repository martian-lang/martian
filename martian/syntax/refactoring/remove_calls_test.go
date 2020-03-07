package refactoring

import (
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestRemoveUnusedCalls(t *testing.T) {
	var parser syntax.Parser

	_, _, ast, err := parser.Compile("testdata/resolve_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	edit := RemoveAllUnusedCalls([]*syntax.Ast{ast})

	/*
	   Outputs of call PIPE4 of pipeline POINT_MAPPER are not used
	   Outputs of call PIPE5 of pipeline POINT_MAPPER are not used
	   Outputs of call PIPE6 of pipeline POINT_MAPPER are not used
	   Outputs of call POINT_USER of pipeline POINT_MAPPER are not used
	   Input mp of pipeline POINT_MAPPER is no longer used
	   Input disable4 of pipeline POINT_MAPPER is no longer used
	   Input disable_user of pipeline POINT_MAPPER is no longer used
	*/

	mapIns := ast.Callables.Table["POINT_MAPPER"].GetInParams()

	if mapIns.Table["mp"] == nil {
		t.Error("expected input mp for POINT_MAPPER")
	}
	if edit == nil {
		t.Fatal("Expected edits", edit)
	}
	if c, err := edit.Apply(ast); err != nil {
		t.Fatal(err)
	} else if c < 7 {
		t.Errorf("%d < 7", c)
	}
	if mapIns.Table["mp"] != nil {
		t.Error("expected input mp for POINT_MAPPER to have been removed")
	}
}
