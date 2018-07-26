// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check types.

package syntax

// Build type table, starting with builtins. Duplicates allowed.
func (global *Ast) compileTypes() error {
	for _, builtinType := range builtinTypes {
		global.TypeTable[builtinType.Id] = builtinType
	}
	for _, userType := range global.UserTypes {
		global.TypeTable[userType.Id] = userType
		global.UserTypeTable[userType.Id] = userType
	}
	return nil
}

func (global *Ast) isUserType(t string) bool {
	_, ok := global.UserTypeTable[t]
	return ok
}

func (global *Ast) checkTypeMatch(paramType string, valueType string) bool {
	return (valueType == KindNull ||
		paramType == valueType ||
		(paramType == KindPath && valueType == KindString) ||
		(paramType == KindFile && valueType == KindString) ||
		(paramType == KindFloat && valueType == KindInt) ||
		// Allow implicit cast between string and user file type
		(global.isUserType(paramType) &&
			(valueType == KindString || valueType == KindFile)) ||
		(global.isUserType(valueType) &&
			(paramType == KindString || paramType == KindFile)))
}
