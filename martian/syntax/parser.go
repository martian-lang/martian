//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO semantic checking.
//

package syntax

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/martian-lang/martian/martian/util"
)

//
// Semantic Checking Methods
//
func (global *Ast) err(nodable AstNodable, msg string, v ...interface{}) error {
	return &AstError{global, nodable.getNode(), fmt.Sprintf(msg, v...)}
}

func (global *Ast) compile() error {
	if err := global.CompileTypes(); err != nil {
		return err
	}

	// Check for duplicate names amongst callables.
	if err := global.Callables.compile(global); err != nil {
		return err
	}

	if err := global.compileStages(); err != nil {
		return err
	}

	if err := global.compilePipelineDecs(); err != nil {
		return err
	}

	if err := global.compilePipelineArgs(); err != nil {
		return err
	}

	if err := global.compileCall(); err != nil {
		return err
	}

	return nil
}

func (src *SrcParam) FindPath(searchPaths []string) (string, error) {
	if filepath.IsAbs(src.Path) {
		_, err := os.Stat(src.Path)
		if err != nil {
			return src.Path, &wrapError{
				innerError: err,
				loc:        src.Node.Loc,
			}
		}
		return src.Path, nil
	}
	if src.Node.Loc.File != nil {
		p := filepath.Join(filepath.Dir(src.Node.Loc.File.FullPath), src.Path)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if p, found := util.SearchPaths(src.Path, searchPaths); !found {
		return src.Path, &wrapError{
			innerError: fmt.Errorf(
				"SourcePathError: searched (%s) but stage source path not found '%s'",
				strings.Join(searchPaths, ", "), src.Path),
			loc: src.Node.Loc,
		}
	} else {
		return p, nil
	}
}

func (global *Ast) checkSrcPaths(stagecodePaths []string) error {
	var errs ErrorList
	for _, stage := range global.Stages {
		// Exempt exec stages
		if stage.Src.Lang != "exec" && stage.Src.Lang != "comp" {
			if _, err := stage.Src.FindPath(stagecodePaths); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
}

func (src *SourceFile) checkIncludes(fullPath string, inc *SourceLoc) error {
	var errs ErrorList
	if fullPath == src.FullPath {
		errs = append(errs, &wrapError{
			innerError: fmt.Errorf("Include cycle: %s included", src.FullPath),
			loc:        *inc,
		})
	} else {
		for _, parent := range src.IncludedFrom {
			if err := parent.File.checkIncludes(fullPath, inc); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
}

// A Parser object allows the ParseSourceBytes and Compile methods
// to cache state if repeatedly invoked.
//
// The Parser object is NOT thread safe.
type Parser struct {
	intern *stringIntern
}

// ParseSource parses a souce string into an ast.
//
// src is the mro source code.
//
// srcpath is the path to the source code file (if applicable), used for
// debugging information.
//
// incpaths is the orderd set of search paths to use when resolving include
// directives.
//
// if checksrc is true, then the parser will verify that stage src values
// refer to code that actually exists.
//
// Deprecated: Use ParseSourceBytes instead.
func ParseSource(src string, srcPath string,
	incPaths []string, checkSrc bool) (string, []string, *Ast, error) {
	return ParseSourceBytes([]byte(src), srcPath, incPaths, checkSrc)
}

// ParseSourceBytes parses a source byte array into an ast.
//
// src is the mro source code.
//
// srcpath is the path to the source code file (if applicable), used for
// debugging information.
//
// incpaths is the orderd set of search paths to use when resolving include
// directives.
//
// if checksrc is true, then the parser will verify that stage src values
// refer to code that actually exists.
func ParseSourceBytes(src []byte, srcPath string,
	incPaths []string, checkSrc bool) (string, []string, *Ast, error) {
	var parser Parser
	return parser.ParseSourceBytes(src, srcPath, incPaths, checkSrc)
}

func (parser *Parser) getIntern() *stringIntern {
	if parser == nil {
		return makeStringIntern()
	} else if parser.intern == nil {
		parser.intern = makeStringIntern()
	}
	return parser.intern
}

// ParseSourceBytes parses a source byte array into an ast.
//
// src is the mro source code.
//
// srcpath is the path to the source code file (if applicable), used for
// debugging information.
//
// incpaths is the orderd set of search paths to use when resolving include
// directives.
//
// if checksrc is true, then the parser will verify that stage src values
// refer to code that actually exists.
func (parser *Parser) ParseSourceBytes(src []byte, srcPath string,
	incPaths []string, checkSrc bool) (string, []string, *Ast, error) {
	srcPath, absPath, err := IncludeFilePath(srcPath, incPaths)
	if err != nil {
		return "", nil, nil, err
	}
	srcFile := SourceFile{
		FileName: srcPath,
		FullPath: absPath,
	}
	if ast, err := parser.parseSource(src, &srcFile, incPaths[:len(incPaths):len(incPaths)],
		map[string]*SourceFile{absPath: &srcFile}); err != nil {
		return "", nil, ast, err
	} else {
		err := ast.compile()
		ifnames := make([]string, len(ast.Includes))
		for i, inc := range ast.Includes {
			ifnames[i] = inc.Value
		}
		if checkSrc {
			stagecodePaths := filepath.SplitList(os.Getenv("PATH"))
			seenPaths := make(map[string]struct{}, len(incPaths)+len(stagecodePaths))
			for f := range ast.Files {
				p := filepath.Dir(f)
				if _, ok := seenPaths[p]; !ok {
					stagecodePaths = append(stagecodePaths, p)
					seenPaths[p] = struct{}{}
				}
			}
			if srcerr := ast.checkSrcPaths(stagecodePaths); srcerr != nil {
				err = ErrorList{err, srcerr}.If()
			}
		}
		return ast.format(false), ifnames, ast, err
	}
}

// UncheckedParse loads an Ast from source bytes, but does not follow includes
// or perform any semantic verification.
func (parser *Parser) UncheckedParse(src []byte, srcPath string) (*Ast, error) {
	absPath, _ := filepath.Abs(srcPath)
	srcFile := SourceFile{
		FileName: srcPath,
		FullPath: absPath,
	}
	return yaccParse(src, &srcFile, parser.getIntern())
}

// UncheckedParseIncludes loads an Ast from source bytes, including processing
// include directives, but does not perform any further semantic verification.
func (parser *Parser) UncheckedParseIncludes(src []byte,
	srcPath string, incPaths []string) (*Ast, error) {
	srcPath, absPath, err := IncludeFilePath(srcPath, incPaths)
	if err != nil {
		return nil, err
	}
	srcFile := SourceFile{
		FileName: srcPath,
		FullPath: absPath,
	}
	return parser.parseSource(src, &srcFile, incPaths[:len(incPaths):len(incPaths)],
		map[string]*SourceFile{absPath: &srcFile})
}

func (parser *Parser) parseSource(src []byte, srcFile *SourceFile, incPaths []string,
	processedIncludes map[string]*SourceFile) (*Ast, error) {
	// Parse the source into an AST and attach the comments.
	ast, err := yaccParse(src, srcFile, parser.getIntern())
	if err != nil {
		return nil, err
	}

	iasts, err := parser.getIncludes(srcFile, ast.Includes, incPaths, processedIncludes)
	if iasts != nil {
		if err := ast.merge(iasts); err != nil {
			return nil, err
		}
	}
	return ast, err
}

func (parser *Parser) getIncludes(srcFile *SourceFile, includes []*Include, incPaths []string,
	processedIncludes map[string]*SourceFile) (*Ast, error) {
	if len(includes) == 0 {
		return nil, nil
	}
	// Add the source file's own folder to the include path for
	// resolving both @includes and stage src paths.
	srcDir := filepath.Dir(srcFile.FullPath)
	incPaths = append(incPaths, srcDir)

	var errs ErrorList
	var iasts *Ast
	seen := make(map[string]struct{}, len(includes))
	for _, inc := range includes {
		if ifpath, err := util.FindUniquePath(inc.Value, incPaths); err != nil {
			errs = append(errs, &FileNotFoundError{
				name:  inc.Value,
				loc:   inc.Node.Loc,
				inner: err,
				paths: strings.Join(incPaths, ":"),
			})
		} else {
			absPath, _ := filepath.Abs(ifpath)
			if _, ok := seen[absPath]; ok {
				errs = append(errs, &wrapError{
					innerError: fmt.Errorf("%s included multiple times",
						inc.Value),
					loc: inc.Node.Loc,
				})
			}
			seen[absPath] = struct{}{}

			if absPath == srcFile.FullPath {
				errs = append(errs, &wrapError{
					innerError: fmt.Errorf("%s includes itself", srcFile.FullPath),
					loc:        inc.Node.Loc,
				})
			} else if iSrcFile := processedIncludes[absPath]; iSrcFile != nil {
				iSrcFile.IncludedFrom = append(iSrcFile.IncludedFrom, &inc.Node.Loc)
				if err := srcFile.checkIncludes(absPath, &inc.Node.Loc); err != nil {
					errs = append(errs, err)
				}
			} else {
				iSrcFile = &SourceFile{
					FileName:     inc.Value,
					FullPath:     absPath,
					IncludedFrom: []*SourceLoc{&inc.Node.Loc},
				}
				processedIncludes[absPath] = iSrcFile
				if b, err := ioutil.ReadFile(iSrcFile.FullPath); err != nil {
					errs = append(errs, &wrapError{
						innerError: err,
						loc:        inc.Node.Loc,
					})
				} else {
					iast, err := parser.parseSource(b, iSrcFile,
						incPaths[:len(incPaths)-1], processedIncludes)
					// The last element of the array may have been overwritten.
					// Restore it.
					incPaths[len(incPaths)-1] = srcDir
					errs = append(errs, err)
					if iast != nil {
						if iasts == nil {
							iasts = iast
						} else {
							// x.merge(y) puts y's stuff before x's.
							if err := iast.merge(iasts); err != nil {
								errs = append(errs, err)
							}
							iasts = iast
						}
					}
				}
			}
		}
	}
	return iasts, errs.If()
}

// Get the mropath-relative and absolute paths for a file name,
// which may or may not be an aboslute file name.
func IncludeFilePath(filename string, mroPaths []string) (rel, abs string, err error) {
	abs, err = filepath.Abs(filename)
	if err != nil || len(mroPaths) == 0 {
		return filename, abs, err
	}
	rdir := filepath.Dir(filename)
	adir := filepath.Dir(abs)
	// Check for direct include before looking at deeper include paths.
	for _, p := range mroPaths {
		if rdir == p || adir == p {
			return filepath.Base(filename), abs, nil
		}
	}
	for _, p := range mroPaths {
		if p == "" {
			// working directory is in MROPATH.
			if !filepath.IsAbs(filename) {
				return filename, abs, nil
			}
		} else if strings.HasPrefix(rdir, p) {
			// Relative path to directory containing filename is in MROPATH
			if p[len(p)-1] == '/' {
				return filename[len(p):], abs, nil
			} else if rdir[len(p)] == '/' {
				return filename[len(p)+1:], abs, nil
			}
		} else if strings.HasPrefix(adir, p) {
			// Absolute path of directory containing filename is in MROPATH
			if p[len(p)-1] == '/' {
				return abs[len(p):], abs, nil
			} else if adir[len(p)] == '/' {
				return abs[len(p)+1:], abs, nil
			}
		}
		ap, err := filepath.Abs(p)
		if err != nil {
			return filename, abs, err
		}
		if ap == adir {
			return filepath.Base(filename), abs, nil
		}
		if strings.HasPrefix(adir, ap) {
			if adir[len(ap)] == '/' {
				return abs[len(ap)+1:], abs, nil
			}
		}
	}
	return filename, abs, nil
}

// Compile an MRO file in cwd or mroPaths.
//
// fpath is the path (absolute or relative to the current working directory) of
// the source file.
//
// mroPaths specifies additional paths in which to search files requested with
// @include
//
// If checkcSrcPath is true, an error will be returned if the src parameter in
// a stage definition does not refer to an existing path.
//
// Returns the combined source (after processing all includes), the transitive
// closure of all includes, the compiled AST, or an error if applicable.
func Compile(fpath string,
	mroPaths []string, checkSrcPath bool) (string, []string, *Ast, error) {
	var parser Parser
	return parser.Compile(fpath, mroPaths, checkSrcPath)
}

// Compile an MRO file in cwd or mroPaths.
//
// fpath is the path (absolute or relative to the current working directory) of
// the source file.
//
// mroPaths specifies additional paths in which to search files requested with
// @include
//
// If checkcSrcPath is true, an error will be returned if the src parameter in
// a stage definition does not refer to an existing path.
//
// Returns the combined source (after processing all includes), the transitive
// closure of all includes, the compiled AST, or an error if applicable.
func (parser *Parser) Compile(fpath string,
	mroPaths []string, checkSrcPath bool) (string, []string, *Ast, error) {

	if data, err := ioutil.ReadFile(fpath); err != nil {
		return "", nil, nil, err
	} else {
		return parser.ParseSourceBytes(data, fpath, mroPaths, checkSrcPath)
	}
}

// Parse a byte string to a *ValExp object.
func (parser *Parser) ParseValExp(data []byte) (ValExp, error) {
	return parseExp(data, &SourceFile{FileName: "[]byte"}, parser.getIntern())
}
