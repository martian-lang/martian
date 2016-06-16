package core

// Facilities to search an AST

func SearchPipestanceParams(pipestance *Ast, what string) interface{} {
	b1 := pipestance.Call.Bindings.Table[what]
	if b1 == nil {
		return nil
	} else {
		return b1.Exp.(*ValExp).Value
	}
}
