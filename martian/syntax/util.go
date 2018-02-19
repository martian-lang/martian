//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//

package syntax

func SearchPipestanceParams(pipestance *Ast, what string) interface{} {
	b1 := pipestance.Call.Bindings.Table[what]
	if b1 == nil {
		return nil
	} else {
		return b1.Exp.(*ValExp).Value
	}
}
