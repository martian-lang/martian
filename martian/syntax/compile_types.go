// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check types.

package syntax

import (
	"fmt"
	"strings"
)

// CompileTypes initializes the TypeTable for the Ast object, starting with
// builtins.
//
// Duplicate declarations are allowed for user-defined file types.
// For struct types and callables, duplicates are allowed (at this stage) if
// and only if they are functionally identical.
func (global *Ast) CompileTypes() error {
	var errs ErrorList
	global.TypeTable.init(len(global.UserTypes) + len(global.StructTypes) + len(global.Callables.List))
	for _, userType := range global.UserTypes {
		if err := global.TypeTable.AddUserType(userType); err != nil {
			errs = append(errs, err)
		}
	}
	for _, structType := range global.StructTypes {
		if err := structType.compile(global); err != nil {
			errs = append(errs, err)
		}
		if err := global.TypeTable.AddStructType(structType); err != nil {
			errs = append(errs, err)
		}
	}
	for _, callable := range global.Callables.List {
		if len(callable.GetOutParams().List) > 0 {
			structType := structFromCallable(callable)
			if err := structType.compile(global); err != nil {
				errs = append(errs, err)
			}
			if err := global.TypeTable.AddStructType(structType); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
}

func (st *StructType) compile(global *Ast) error {
	if len(st.Members) < 1 {
		return global.err(st, "EmptyStructError: struct has no fields")
	}
	st.Table = make(map[string]*StructMember, len(st.Members))
	outNames := make(map[string]struct{}, len(st.Members))
	var errs ErrorList
	for _, member := range st.Members {
		// Check for duplicates
		if existing, ok := st.Table[member.Id]; ok {
			var msg strings.Builder
			fmt.Fprintf(&msg,
				"DuplicateNameError: field '%s' was already declared when encountered again",
				member.Id)
			if node := existing.getNode(); node != nil {
				msg.WriteString(".\n  Previous declaration at ")
				node.Loc.writeTo(&msg, "      ")
				msg.WriteRune('\n')
			}
			errs = append(errs, global.err(member, msg.String()))
		} else {
			st.Table[member.Id] = member
		}
		if err := member.compile(st, global); err != nil {
			errs = append(errs, err)
		}
		if out := member.GetOutFilename(); out != "" {
			if _, ok := outNames[out]; ok {
				errs = append(errs, global.err(member,
					"DuplicateNameError: member '%s' has output name '%s' which was already used",
					member.Id, out))
			} else {
				outNames[out] = struct{}{}
			}
		}
	}
	return errs.If()
}

func (member *StructMember) CacheIsFile(t Type) {
	switch t.(type) {
	case *BuiltinType, *UserType:
		member.isComplex = false
	default:
		member.isComplex = true
	}
	member.isFile = t.IsFile()
}

func (member *StructMember) compile(st *StructType, global *Ast) error {
	if member.Tname.Tname == st.Id {
		return global.err(member, fmt.Sprintf(
			"TypeError: field %q of struct type %q cannot be of type %q",
			member.Id, st.Id, st.Id))
	} else if t := global.TypeTable.Get(member.Tname); t == nil {
		return global.err(member, fmt.Sprintf(
			"TypeError: unknown type %q for parameter %q",
			member.Tname.String(), member.Id))
	} else {
		member.CacheIsFile(t)
		switch member.isFile {
		case KindMayContainPaths:
			if st.isFile == KindIsNotFile {
				st.isFile = KindMayContainPaths
			}
		case KindIsFile, KindIsDirectory:
			if st.isFile == KindIsNotFile || st.isFile == KindMayContainPaths {
				st.isFile = KindIsDirectory
			}
			if member.OutName != "" {
				if err := checkLegalFilename(member.OutName); err != nil {
					return &wrapError{
						innerError: fmt.Errorf("out %s name %q for field %s is not "+
							"legal under Microsoft Windows operating systems "+
							"and may cause issues for users who export their "+
							"results to such filesystems: %v",
							member.isFile.String(),
							member.OutName, member.Id, err),
						loc: member.Node.Loc,
					}
				}
			}
		}
		return nil
	}
}
