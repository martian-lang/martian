//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
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
