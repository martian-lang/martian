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
		} else if !actual.equal(expected) {
			t.Errorf("%s != %s", actual.GoString(), expected.GoString())
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
	} else if !ref.equal(e) {
		t.Errorf("%s != %s", e.GoString(), ref.GoString())
	}
}

func TestExpBindingPathIndex(t *testing.T) {
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

	check := func(p, e string, index []CollectionIndex) {
		t.Helper()
		actual, err := exp.BindingPath(p, nil, index)
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := parser.ParseValExp([]byte(e))
		if err != nil {
			t.Error(err)
		} else if !actual.equal(expected) {
			t.Errorf("%s != %s", actual.GoString(), expected.GoString())
		}
	}
	check("c.e", `"bar"`, []CollectionIndex{mapKeyIndex("d")})
	check("d.e", `"bar"`, []CollectionIndex{arrayIndex(0)})
	if _, err := exp.BindingPath("c.e", nil, []CollectionIndex{arrayIndex(1)}); err == nil {
		t.Error("expected failure")
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
		actual, err := rb.BindingPath(p, nil, nil, &ast.TypeTable)
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := parser.ParseValExp([]byte(e))
		if err != nil {
			t.Error(err)
		} else if !actual.Exp.equal(expected) {
			t.Errorf("%s != %s", actual.Exp.GoString(), expected.GoString())
		}
	}
	check("a.b", `"foo"`)
	check("c.e", `{"d":"bar", "f":"baz"}`)
	check("d.e", `["bar", "baz"]`)
}
