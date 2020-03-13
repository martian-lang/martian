//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//

package syntax

// GenerateCall creates a CallStm calling the given Callable with the given
// inputs.  Missing inputs are null.
func GenerateCall(target Callable, args map[string]Exp) *CallStm {
	c := &CallStm{
		Modifiers: &Modifiers{Bindings: new(BindStms)},
		Id:        target.GetId(),
		DecId:     target.GetId(),
		Bindings: &BindStms{
			List: make([]*BindStm, 0, len(target.GetInParams().List)),
			Table: make(map[string]*BindStm,
				len(target.GetInParams().List)),
		},
	}
	for _, arg := range target.GetInParams().List {
		v := args[arg.GetId()]
		if v == nil {
			v = new(NullExp)
		}
		b := BindStm{
			Id:    arg.GetId(),
			Exp:   v,
			Tname: arg.GetTname(),
		}
		c.Bindings.List = append(c.Bindings.List, &b)
		c.Bindings.Table[b.Id] = &b
	}
	return c
}

// GenerateAbstractCall creates a CallStm calling the given Callable with
// arbitrary simulated inputs appropriate for static analysis of call graphs.
//
// Because some arguments may be mapped over or used to disable sub-pipelines,
// rather than just giving null as inputs the arguments given to each pipeline
// input parameter are as follows:
//
//   - boolean inputs get false.
//   - Untyped maps, strings, numeric types and file types get null.
//   - Arrays get a single element.
//   - Typed maps get a single element with a key named "null".
func GenerateAbstractCall(target Callable, types *TypeLookup) *CallStm {
	c := &CallStm{
		Modifiers: &Modifiers{Bindings: new(BindStms)},
		Id:        target.GetId(),
		DecId:     target.GetId(),
		Bindings: &BindStms{
			List: make([]*BindStm, 0, len(target.GetInParams().List)),
			Table: make(map[string]*BindStm,
				len(target.GetInParams().List)),
		},
	}
	for _, arg := range target.GetInParams().List {
		typ := types.Get(arg.Tname)
		v := abstractExpForType(typ, &arg.Node, types)
		b := BindStm{
			Id:    arg.GetId(),
			Exp:   v,
			Tname: arg.GetTname(),
		}
		c.Bindings.List = append(c.Bindings.List, &b)
		c.Bindings.Table[b.Id] = &b
	}
	return c
}

func abstractExpForType(typ Type, node *AstNode, types *TypeLookup) ValExp {
	switch typ := typ.(type) {
	case *ArrayType:
		exp := abstractExpForType(typ.Elem, node, types)
		for i := int16(0); i < typ.Dim; i++ {
			exp = &ArrayExp{
				valExp: valExp{Node: *node},
				Value:  []Exp{exp},
			}
		}
		return exp
	case *TypedMapType:
		exp := abstractExpForType(typ.Elem, node, types)
		return &MapExp{
			valExp: valExp{Node: *node},
			Kind:   KindMap,
			Value: map[string]Exp{
				"null": exp,
			},
		}
	case *StructType:
		exp := &MapExp{
			valExp: valExp{Node: *node},
			Kind:   KindStruct,
			Value:  make(map[string]Exp, len(typ.Members)),
		}
		for _, member := range typ.Members {
			exp.Value[member.Id] = abstractExpForType(
				types.Get(member.Tname),
				node,
				types)
		}
		return exp
	case *BuiltinType:
		if typ.Id == KindBool {
			return &BoolExp{
				valExp: valExp{Node: *node},
			}
		}
	}
	return &NullExp{
		valExp: valExp{Node: *node},
	}
}
