package syntax

import "testing"

func TestExpBindingPath(t *testing.T) {
	var parser Parser
	exp, err := parser.ParseValExp([]byte(`{
	a: {b: "foo"},
	c: {
		"d": {
			e: "bar",
		},
		"f": {
			e: "baz",
		},
	},
	d: [
		{e:"bar"},
		{e:"baz"},
	],
	f: STAGE.out1,
}`))
	if err != nil {
		t.Fatal(err)
	}

	check := func(p, e string) {
		t.Helper()
		actual, err := exp.BindingPath(p, nil, nil)
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := parser.ParseValExp([]byte(e))
		if err != nil {
			t.Error(err)
		} else if err := actual.equal(expected); err != nil {
			t.Errorf("%s != %s: %v",
				actual.GoString(), expected.GoString(), err)
		}
	}
	check("a.b", `"foo"`)
	check("c.e", `{"d":"bar", "f":"baz"}`)
	check("d.e", `["bar", "baz"]`)
	ref := RefExp{
		Id:       "STAGE",
		OutputId: "out1.bar",
		Kind:     KindCall,
	}
	if e, err := exp.BindingPath("f.bar", nil, nil); err != nil {
		t.Error(err)
	} else if err := ref.equal(e); err != nil {
		t.Errorf("%s != %s: %v", e.GoString(), ref.GoString(), err)
	}
}

func TestResolvedBindingBindingPath(t *testing.T) {
	var parser Parser
	exp, err := parser.ParseValExp([]byte(`{
	a: {b: "foo"},
	c: {
		"d": {
			e: "bar",
		},
		"f": {
			e: "baz",
		},
	},
	d: [
		{e:"bar"},
		{e:"baz"},
	],
}`))
	if err != nil {
		t.Fatal(err)
	}
	ast := testGood(t, `
struct Es(
	string e,
)

struct As(
	string b,
)

struct Top(
	As a,
	map<Es> c,
	Es[] d,
)
`)
	if ast == nil {
		return
	}
	rb := ResolvedBinding{
		Exp:  exp,
		Type: ast.TypeTable.Get(TypeId{Tname: "Top"}),
	}
	if rb.Type == nil {
		t.Fatal("could not get type")
	}
	check := func(p, e string) {
		t.Helper()
		actual, err := rb.BindingPath(p, nil, &ast.TypeTable)
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := parser.ParseValExp([]byte(e))
		if err != nil {
			t.Error(err)
		} else if err := actual.Exp.equal(expected); err != nil {
			t.Errorf("%s != %s: %v", actual.Exp.GoString(), expected.GoString(), err)
		}
	}
	check("a.b", `"foo"`)
	check("c.e", `{"d":"bar", "f":"baz"}`)
	check("d.e", `["bar", "baz"]`)
}
