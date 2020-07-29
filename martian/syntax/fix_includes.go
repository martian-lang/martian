//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Fix includes, inspired by https://include-what-you-use.org/
//

package syntax

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/util"
)

func (parser *Parser) FixIncludes(source *Ast, mropath []string) error {
	seen := make(map[string]*SourceFile, len(source.Files)+len(source.Includes))
	incPaths := make([]string, 0, len(mropath)+1)
	seenPaths := make(map[string]struct{}, len(mropath))
	var srcFile *SourceFile
	for _, p := range mropath {
		if _, ok := seenPaths[p]; !ok {
			incPaths = append(incPaths, p)
		}
	}
	for k, v := range source.Files {
		seen[k] = v
		d := filepath.Dir(v.FullPath)
		if _, ok := seenPaths[d]; !ok {
			incPaths = append(incPaths, d)
		}
		srcFile = v
	}
	if closure, err := parser.getIncludes(srcFile, source.Includes,
		incPaths, seen); err != nil {
		return err
	} else {
		if err := uncheckedMakeTables(source, closure); err != nil {
			util.PrintError(err, "include", "WARNING: compile errors")
			util.PrintInfo("include", "         Attempting to fix...")
		}
		var ut []*UserType
		if closure != nil {
			ut = closure.UserTypes
		}
		needed, optional, missingTypes, missingCalls := getRequiredIncludes(source, ut)
		extraIncs, extraTypes, err := parser.findMissingIncludes(seen,
			missingTypes, missingCalls,
			incPaths)
		for _, file := range extraIncs {
			if _, ok := needed[file.FullPath]; !ok {
				needed[file.FullPath] = file
			}
		}
		delete(needed, srcFile.FileName)
		delete(needed, srcFile.FullPath)

		incLookup := make(map[string]*SourceFile, len(needed))
		for _, f := range needed {
			if len(f.IncludedFrom) > 0 && f.IncludedFrom[0].File == srcFile {
				incLookup[f.FileName] = f
			} else if p, _, err := IncludeFilePath(f.FullPath, incPaths); err == nil {
				incLookup[p] = f
			} else {
				incLookup[f.FileName] = f
			}
		}
		optionalLookup := make(map[string]*SourceFile, len(optional))
		for _, f := range optional {
			if len(f.IncludedFrom) > 0 && f.IncludedFrom[0].File == srcFile {
				optionalLookup[f.FileName] = f
			} else if p, _, err := IncludeFilePath(f.FullPath, incPaths); err == nil {
				optionalLookup[p] = f
			} else {
				optionalLookup[f.FileName] = f
			}
		}
		fixIncludes(source, incLookup, optionalLookup, extraTypes)
		return err
	}
}

// Compile the type and callable tables, but do not enforce uniqueness or
// check anything else.  It does ensure that the first mention is the one that
// will be in the table.
func uncheckedMakeTables(top *Ast, included *Ast) error {
	for _, callable := range top.Callables.List {
		if top.Callables.Table == nil {
			top.Callables.Table = make(map[string]Callable, len(top.Callables.List))
		}
		if _, ok := top.Callables.Table[callable.GetId()]; !ok {
			top.Callables.Table[callable.GetId()] = callable
		}
	}
	var errs ErrorList

	for _, userType := range top.UserTypes {
		if top.TypeTable.baseTypes == nil {
			top.TypeTable.init(len(top.UserTypes) + len(top.StructTypes) + len(top.Callables.List))
		}
		if err := top.TypeTable.AddUserType(userType); err != nil {
			errs = append(errs, err)
		}
	}
	for _, structType := range top.StructTypes {
		if top.TypeTable.baseTypes == nil {
			top.TypeTable.init(len(top.UserTypes) + len(top.StructTypes) + len(top.Callables.List))
		}
		if err := top.TypeTable.AddStructType(structType); err != nil {
			errs = append(errs, err)
		}
	}
	for _, callable := range top.Callables.List {
		if len(callable.GetOutParams().List) > 0 {
			structType := structFromCallable(callable)
			if top.TypeTable.baseTypes == nil {
				top.TypeTable.init(len(top.UserTypes) + len(top.StructTypes) + len(top.Callables.List))
			}
			if err := top.TypeTable.AddStructType(structType); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if included != nil {
		// Add included callables and types to table, but not to list.

		for _, callable := range included.Callables.List {
			if top.Callables.Table == nil {
				top.Callables.Table = make(map[string]Callable, len(included.Callables.List))
			}
			if _, ok := top.Callables.Table[callable.GetId()]; !ok {
				top.Callables.Table[callable.GetId()] = callable
			}
		}
		for _, userType := range included.UserTypes {
			if top.TypeTable.baseTypes == nil {
				top.TypeTable.init(
					len(included.UserTypes) +
						len(included.StructTypes) +
						len(included.Callables.List))
			}
			if err := top.TypeTable.AddUserType(userType); err != nil {
				errs = append(errs, err)
			}
		}
		for _, structType := range included.StructTypes {
			if top.TypeTable.baseTypes == nil {
				top.TypeTable.init(
					len(included.UserTypes) +
						len(included.StructTypes) +
						len(included.Callables.List))
			}
			if err := top.TypeTable.AddStructType(structType); err != nil {
				errs = append(errs, err)
			}
		}
		for _, callable := range included.Callables.List {
			if top.TypeTable.baseTypes == nil {
				top.TypeTable.init(len(included.UserTypes) +
					len(included.StructTypes) +
					len(included.Callables.List))
			}
			if err := top.TypeTable.AddStructType(structFromCallable(callable)); err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, structType := range top.StructTypes {
		if err := structType.compile(top); err != nil {
			errs = append(errs, err)
		}
	}
	for _, callable := range top.Callables.List {
		if len(callable.GetOutParams().List) > 0 {
			structType := structFromCallable(callable)
			if err := structType.compile(top); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
}

func isTransitivelyIncluded(file *SourceFile, included, usedTransitively map[string]*SourceFile) bool {
	if _, ok := included[file.FullPath]; ok {
		return true
	} else if _, ok := usedTransitively[file.FullPath]; ok {
		return true
	}
	for _, from := range file.IncludedFrom {
		if from != nil && from.File != nil &&
			len(from.File.IncludedFrom) > 0 &&
			isTransitivelyIncluded(from.File, included, nil) {
			if usedTransitively != nil {
				usedTransitively[file.FullPath] = file
			}
			return true
		}
	}
	return false
}

func addIncludesForInParamTypes(source *Ast, params *InParams,
	unknownTypes map[string]*UserType,
	required, usedTransitively map[string]*SourceFile, allowTransitive bool) {
	for _, param := range params.List {
		tName := param.GetTname()
		if t := source.TypeTable.Get(TypeId{Tname: tName.Tname}); t != nil {
			// Don't worry about builtin types
			if tn, ok := t.(AstNodable); ok {
				if srcFile := tn.getNode().Loc.File; srcFile != param.File() {
					switch t := t.(type) {
					case *UserType:
						if !allowTransitive || !isTransitivelyIncluded(srcFile,
							required, nil) {
							// Just re-define it locally.
							unknownTypes[tName.Tname] = t
						}
					default:
						if !allowTransitive || !isTransitivelyIncluded(srcFile,
							required, usedTransitively) {
							// Structs, etc
							required[srcFile.FullPath] = srcFile
						}
					}
				}
			}
		} else {
			unknownTypes[tName.Tname] = &UserType{
				Id: tName.Tname,
			}
		}
	}
}

func addIncludesForOutParamTypes(source *Ast, params *OutParams,
	unknownTypes map[string]*UserType,
	required, usedTransitively map[string]*SourceFile, allowTransitive bool) {
	for _, param := range params.List {
		addIncludesForMemberType(source, &param.StructMember, unknownTypes,
			required, usedTransitively, allowTransitive)
	}
}

func addIncludesForMemberType(source *Ast, param *StructMember,
	unknownTypes map[string]*UserType,
	required, usedTransitively map[string]*SourceFile, allowTransitive bool) {
	tName := param.Tname
	if t := source.TypeTable.Get(TypeId{Tname: tName.Tname}); t != nil {
		// Don't worry about builtin types
		if tn, ok := t.(AstNodable); ok {
			if srcFile := tn.getNode().Loc.File; srcFile != param.File() {
				switch t := t.(type) {
				case *UserType:
					if !allowTransitive || !isTransitivelyIncluded(srcFile,
						required, nil) {
						// Just re-define it locally.
						unknownTypes[tName.Tname] = t
					}
				default:
					if !allowTransitive || !isTransitivelyIncluded(srcFile,
						required, usedTransitively) {
						// Structs, etc
						required[srcFile.FullPath] = srcFile
					}
				}
			}
		}
	} else {
		unknownTypes[tName.Tname] = &UserType{
			Id: tName.Tname,
		}
	}
}

// Get the set of includes which are required for this source AST,
// as well as the set of types and callables which remain undefined.
func getRequiredIncludes(source *Ast, userTypes []*UserType) (
	required, optional map[string]*SourceFile,
	unknownTypes map[string]*UserType,
	unknownCallables map[string]struct{}) {
	required = make(map[string]*SourceFile, 1+len(source.Includes))
	for k, v := range source.Files {
		required[k] = v
		required[v.FullPath] = v
	}
	unknownTypes = make(map[string]*UserType)
	unknownCallables = make(map[string]struct{})
	if source.Call != nil {
		if call := source.Callables.Table[source.Call.DecId]; call != nil {
			file := call.getNode().Loc.File
			required[file.FullPath] = file
		} else {
			unknownCallables[source.Call.DecId] = struct{}{}
		}
	}
	for _, pipeline := range source.Pipelines {
		for _, call := range pipeline.Calls {
			if c := source.Callables.Table[call.DecId]; c != nil {
				file := c.getNode().Loc.File
				required[file.FullPath] = file
			} else {
				unknownCallables[call.DecId] = struct{}{}
			}
		}
	}
	optional = make(map[string]*SourceFile, len(source.Includes))
	for _, pipeline := range source.Pipelines {
		addIncludesForInParamTypes(source, pipeline.InParams, unknownTypes, required, optional, true)
		addIncludesForOutParamTypes(source, pipeline.OutParams, unknownTypes, required, optional, true)
	}
	// Check that the input and output types for all stages are declared.
	for _, stage := range source.Stages {
		addIncludesForInParamTypes(source, stage.InParams, unknownTypes, required, optional, false)
		addIncludesForInParamTypes(source, stage.ChunkIns, unknownTypes, required, optional, false)
		addIncludesForOutParamTypes(source, stage.OutParams, unknownTypes, required, optional, false)
		addIncludesForOutParamTypes(source, stage.ChunkOuts, unknownTypes, required, optional, false)
	}
	for _, structType := range source.StructTypes {
		for _, member := range structType.Members {
			addIncludesForMemberType(source, member, unknownTypes, required, optional, false)
		}
	}
	if len(unknownTypes) > 0 && len(userTypes) > 0 {
		// Check that the required types weren't brought in by the includes for
		// struct types.
		var excess []string
		isAlreadyIncluded := func(name string) bool {
			if ty := source.TypeTable.Get(TypeId{Tname: name}); ty != nil {
				if tn, ok := ty.(AstNodable); ok && isTransitivelyIncluded(
					tn.getNode().Loc.File, required, optional) {
					return true
				} else if ok {
					// The type may have been declared in other included files
					// besides the first one which is listed on the type info.
					for _, t := range userTypes {
						if t.Id == name {
							if isTransitivelyIncluded(t.Node.Loc.File, required, optional) {
								return true
							}
						}
					}
				}
			}
			return false
		}
		for name := range unknownTypes {
			if isAlreadyIncluded(name) {
				excess = append(excess, name)
			}
		}
		if len(excess) == len(unknownTypes) {
			unknownTypes = nil
		} else {
			for _, name := range excess {
				delete(unknownTypes, name)
			}
		}
	}
	return required, optional, unknownTypes, unknownCallables
}

func (parser *Parser) findMissingIncludes(seenFiles map[string]*SourceFile,
	neededTypes map[string]*UserType,
	neededCallables map[string]struct{},
	incPaths []string) ([]*SourceFile, []Type, error) {
	if len(neededTypes) == 0 && len(neededCallables) == 0 {
		return nil, nil, nil
	}
	// Types may be declared in multiple files.  We'd prefer not to include
	// one file that declares a type if a file that's included for a callable
	// also declares it.
	neededFiles := make([]*SourceFile, 0, len(neededCallables))
	var errs ErrorList
	for _, incPath := range incPaths {
		if files, err := ioutil.ReadDir(incPath); err != nil {
			errs = append(errs, err)
		} else {
			for _, finfo := range files {
				if !finfo.IsDir() && filepath.Ext(finfo.Name()) == ".mro" {
					absPath, _ := filepath.Abs(filepath.Join(incPath, finfo.Name()))
					if _, ok := seenFiles[absPath]; ok {
						continue
					}
					seenFiles[absPath] = nil
					if src, err := ioutil.ReadFile(absPath); err == nil {
						// Parse and generate the AST.
						srcFile := SourceFile{
							FileName: filepath.Base(absPath),
							FullPath: absPath,
						}
						if ast, err := yaccParse(src, &srcFile, parser.getIntern()); err == nil {
							needed := false
							for _, callable := range ast.Callables.List {
								if _, ok := neededCallables[callable.GetId()]; ok {
									util.PrintInfo("include",
										"Found %s in %s\n",
										callable.GetId(), absPath)
									needed = true
									delete(neededCallables, callable.GetId())
								}
							}
							for _, st := range ast.StructTypes {
								if _, ok := neededTypes[st.GetId()]; ok {
									util.PrintInfo("include",
										"Found %s in %s\n",
										st.Id, absPath)
									needed = true
									delete(neededTypes, st.Id)
								}
							}
							if needed {
								for _, t := range ast.UserTypes {
									delete(neededTypes, t.Id)
								}
								neededFiles = append(neededFiles, &srcFile)
							} else {
								for _, ut := range ast.UserTypes {
									if t, ok := neededTypes[ut.GetId()]; ok {
										if t.getNode().Loc.File == nil {
											neededTypes[t.GetId()] = ut
										}
									}
								}
							}
						}
					}
				}
				if len(neededCallables) == 0 {
					break
				}
			}
		}
		if len(neededCallables) == 0 {
			break
		}
	}
	types := make([]Type, 0, len(neededTypes))
	for _, t := range neededTypes {
		types = append(types, t)
	}
	for c := range neededCallables {
		errs = append(errs, fmt.Errorf(
			"Could not find a definition for a stage or pipeline %s",
			c))
	}
	return neededFiles, types, errs.If()
}

// Add required includes, remove unnecessary ones, and sort them.
func fixIncludes(source *Ast, needed, optional map[string]*SourceFile, extraTypes []Type) {
	// Grab the scope comments off the first node, so that we can reattach them post-sort.
	var scopeComments []*commentBlock
	if len(source.Includes) > 0 {
		scopeComments = source.Includes[0].Node.scopeComments
		source.Includes[0].Node.scopeComments = nil
	}
	var loc SourceLoc
	newIncludes := make([]*Include, 0, len(needed))
	for _, inc := range source.Includes {
		if _, ok := needed[inc.Value]; ok {
			newIncludes = append(newIncludes, inc)
			delete(needed, inc.Value)
		} else if _, ok := optional[inc.Value]; ok {
			newIncludes = append(newIncludes, inc)
			delete(optional, inc.Value)
		}
		loc = inc.Node.Loc
	}
	if loc.File == nil {
		for _, f := range source.Files {
			loc.File = f
			break
		}
	}
	var selfName string
	if loc.File != nil {
		selfName = strings.Trim(strings.TrimSuffix(filepath.Base(loc.File.FileName), ".mro"), "_")
	}
	for f := range needed {
		newIncludes = append(newIncludes, &Include{
			Node: AstNode{
				Loc: loc,
			},
			Value: f,
		})
	}
	sort.Slice(newIncludes, func(i, j int) bool {
		dir1, base1 := filepath.Split(newIncludes[i].Value)
		dir2, base2 := filepath.Split(newIncludes[j].Value)
		// Sort same-directory includes after ones from other directories.
		if len(dir1) == 0 && len(dir2) > 0 {
			return false
		} else if len(dir2) == 0 && len(dir1) > 1 {
			return true
		}
		// Sort by directories.
		if dir1 < dir2 {
			return true
		} else if dir2 < dir1 {
			return false
		}
		// Sort underscore-prefixed files after others.
		// By convention these are "private".
		p1 := strings.HasPrefix(base1, "_")
		p2 := strings.HasPrefix(base2, "_")
		if p1 != p2 {
			return p2
		}
		// Sort files which contain this file's name last, e.g.
		//   _my_pipeline_stages.mro
		// in
		//   my_pipeline.mro
		if selfName != "" {
			p1 = strings.Contains(base1, selfName)
			p2 = strings.Contains(base2, selfName)
			if p1 != p2 {
				return p2
			}
		}
		return newIncludes[i].Value < newIncludes[j].Value
	})
	if len(newIncludes) > 0 {
		newIncludes[0].Node.scopeComments = append(scopeComments,
			newIncludes[0].Node.scopeComments...)
	}
	if len(extraTypes) > 0 {
		for _, t := range source.UserTypes {
			loc = t.Node.Loc
		}
		sort.Slice(extraTypes, func(i, j int) bool {
			// sort UserTypes before structs
			if ui, ok := extraTypes[i].(*UserType); ok {
				if uj, ok := extraTypes[j].(*UserType); !ok {
					return true
				} else {
					return ui.Id < uj.Id
				}
			} else if _, ok := extraTypes[j].(*UserType); ok {
				return false
			}
			return extraTypes[i].TypeId().Tname < extraTypes[j].TypeId().Tname
		})
		for _, t := range extraTypes {
			switch t := t.(type) {
			case *UserType:
				source.UserTypes = append(source.UserTypes, &UserType{
					Id: t.Id,
					Node: AstNode{
						Loc:      loc,
						Comments: t.Node.Comments,
						// Don't include scope comments.
					},
				})
			case *StructType:
				source.StructTypes = append(source.StructTypes, &StructType{
					Id: t.Id,
					Node: AstNode{
						Loc:      loc,
						Comments: t.Node.Comments,
						// Don't include scope comments.
					},
					Members: t.Members,
				})
			default:
				panic(fmt.Sprintf("unexpected extra type %T", t))
			}
		}
	}
	source.Includes = newIncludes
}
