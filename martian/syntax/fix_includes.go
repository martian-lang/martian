//
// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.
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
		needed, missingTypes, missingCalls := getRequiredIncludes(source)
		extraIncs, extraTypes, err := parser.findMissingIncludes(seen,
			missingTypes, missingCalls,
			incPaths)
		for _, file := range extraIncs {
			if _, ok := needed[file.FileName]; !ok {
				needed[file.FileName] = file
			}
		}
		delete(needed, srcFile.FileName)
		delete(needed, srcFile.FullPath)
		fixIncludes(source, needed, extraTypes)
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
	if err := top.CompileTypes(); err != nil {
		errs = append(errs, err)
	}
	if included != nil {
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
				top.TypeTable.init(len(included.UserTypes) + len(included.StructTypes) + len(included.Callables.List))
			}
			if err := top.TypeTable.AddUserType(userType); err != nil {
				errs = append(errs, err)
			}
		}
		for _, structType := range included.StructTypes {
			if top.TypeTable.baseTypes == nil {
				top.TypeTable.init(len(included.UserTypes) + len(included.StructTypes) + len(included.Callables.List))
			}
			if err := top.TypeTable.AddStructType(structType); err != nil {
				errs = append(errs, err)
			}
		}
		for _, callable := range included.Callables.List {
			if top.TypeTable.baseTypes == nil {
				top.TypeTable.init(len(included.UserTypes) + len(included.StructTypes) + len(included.Callables.List))
			}
			if err := top.TypeTable.AddStructType(structFromCallable(callable)); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
}

// Get the set of includes which are required for this source AST,
// as well as the set of types and callables which remain undefined.
func getRequiredIncludes(source *Ast) (map[string]*SourceFile,
	map[string]Type, map[string]struct{}) {
	required := make(map[string]*SourceFile, 1+len(source.Includes))
	for k, v := range source.Files {
		required[k] = v
	}
	unknownTypes := make(map[string]Type)
	unknownCallables := make(map[string]struct{})
	if source.Call != nil {
		if call := source.Callables.Table[source.Call.DecId]; call != nil {
			required[call.getNode().Loc.File.FileName] = call.getNode().Loc.File
		} else {
			unknownCallables[source.Call.DecId] = struct{}{}
		}
	}
	for _, pipeline := range source.Pipelines {
		for _, call := range pipeline.Calls {
			if c := source.Callables.Table[call.DecId]; c != nil {
				required[c.getNode().Loc.File.FileName] = c.getNode().Loc.File
			} else {
				unknownCallables[call.DecId] = struct{}{}
			}
		}
	}
	// Check that the input and output types for all stages are declared.
	// For pipelines, we can assume that their input/output types match
	// those of the stages, meaning we don't need to worry about them.
	for _, stage := range source.Stages {
		for _, params := range []*InParams{
			stage.InParams,
			stage.ChunkIns,
		} {
			for _, param := range params.List {
				tName := param.GetTname()
				if t := source.TypeTable.Get(TypeId{Tname: tName.Tname}); t != nil {
					// Don't worry about builtin types
					if tn, ok := t.(AstNodable); ok {
						if srcFile := tn.getNode().Loc.File; srcFile != param.File() {
							if _, ok := required[srcFile.FileName]; !ok {
								unknownTypes[tName.Tname] = t
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
		for _, params := range []*OutParams{
			stage.OutParams,
			stage.ChunkOuts,
		} {
			for _, param := range params.List {
				tName := param.GetTname()
				if t := source.TypeTable.Get(TypeId{Tname: tName.Tname}); t != nil {
					// Don't worry about builtin types
					if tn, ok := t.(AstNodable); ok {
						if srcFile := tn.getNode().Loc.File; srcFile != param.File() {
							if _, ok := required[srcFile.FileName]; !ok {
								unknownTypes[tName.Tname] = t
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
	}
	return required, unknownTypes, unknownCallables
}

func (parser *Parser) findMissingIncludes(seenFiles map[string]*SourceFile,
	neededTypes map[string]Type,
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
							if needed {
								for _, t := range ast.UserTypes {
									delete(neededTypes, t.Id)
								}
								neededFiles = append(neededFiles, &srcFile)
							} else {
								for _, ut := range ast.UserTypes {
									if t, ok := neededTypes[ut.GetId().Tname]; ok {
										if tn, ok := t.(AstNodable); ok {
											if tn.getNode().Loc.File == nil {
												neededTypes[t.GetId().Tname] = ut
											}
										}
									}
								}
								for _, st := range ast.StructTypes {
									if t, ok := neededTypes[st.GetId().Tname]; ok {
										if tn, ok := t.(AstNodable); ok {
											if tn.getNode().Loc.File == nil {
												neededTypes[t.GetId().Tname] = st
											}
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
func fixIncludes(source *Ast, needed map[string]*SourceFile, extraTypes []Type) {
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
		// Sort underscore-prefixed files after others.
		// By convention these are "private".
		p1 := strings.HasPrefix(newIncludes[i].Value, "_")
		p2 := strings.HasPrefix(newIncludes[j].Value, "_")
		if p1 != p2 {
			return p2
		}
		// Sort files which contain this file's name last, e.g.
		//   _my_pipeline_stages.mro
		// in
		//   my_pipeline.mro
		if selfName != "" {
			p1 = strings.Contains(newIncludes[i].Value, selfName)
			p2 = strings.Contains(newIncludes[j].Value, selfName)
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
			return extraTypes[i].GetId().Tname < extraTypes[j].GetId().Tname
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
