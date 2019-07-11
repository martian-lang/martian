//
// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.
//
// Post-processing logic for pipelines.
//

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func (self *Fork) postProcess() error {
	// Handle formal output parameters
	pipestancePath := self.node.parent.getNode().path
	outsPath := path.Join(pipestancePath, "outs")

	// Handle multi-fork sweeps
	if len(self.node.forks) > 1 {
		outsPath = path.Join(outsPath, self.id)
		util.Print("\nOutputs (%s):\n", self.id)
	} else {
		util.Print("\nOutputs:\n")
	}

	// Create the fork-specific outs/ folder
	util.MkdirAll(outsPath)

	paramList := self.OutParams().List

	// Get fork's output parameter values
	outs := make(LazyArgumentMap, len(paramList))
	err := self.metadata.ReadInto(OutsFile, &outs)

	errs := self.handleOuts(paramList, outs, pipestancePath, outsPath)
	if err != nil {
		errs = append(errs, err)
	}
	util.Print("\n")

	self.printAlarms()
	return errs.If()
}

func (self *Fork) handleOuts(paramList []*syntax.OutParam,
	outs LazyArgumentMap,
	pipestancePath, outsPath string) syntax.ErrorList {
	// Error message accumulator
	var errs syntax.ErrorList

	// Calculate longest key name for alignment
	keyWidth := 0
	for _, param := range paramList {
		// Print out the param help and value
		key := param.GetHelp()
		if len(key) == 0 {
			key = param.GetId()
		}
		if len(key) > keyWidth {
			keyWidth = len(key)
		}
	}
	indent := bytes.Repeat([]byte{' '}, keyWidth+1) // "- keyWidth: "
	var result bytes.Buffer
	// Iterate through output parameters
	for _, param := range paramList {
		id := param.GetId()
		key := param.GetOutName()
		if len(key) == 0 {
			key = id
		}
		if _, err := result.WriteString("- "); err != nil {
			errs = append(errs, err)
		}
		if _, err := result.WriteString(key); err != nil {
			errs = append(errs, err)
		}
		if _, err := result.WriteRune(':'); err != nil {
			errs = append(errs, err)
		}
		// Pull the param value from the fork _outs
		// If value not available, report null
		value := outs[id]
		if err := handleOutParam(&result, &param.StructMember,
			param.IsFile(), value,
			self.node.top.types,
			pipestancePath, outsPath,
			indent[:keyWidth+1-len(key)],
			indent[:2]); err != nil {
			errs = append(errs, err)
		}
		if _, err := result.WriteRune('\n'); err != nil {
			errs = append(errs, err)
		}
		util.PrintBytes(result.Bytes())
		result.Reset()
	}
	return errs
}

func handleOutParam(w *bytes.Buffer, param *syntax.StructMember,
	isFile syntax.FileKind,
	value json.RawMessage, lookup *syntax.TypeLookup,
	pipestancePath, outsPath string,
	pad, indent []byte) error {
	if value == nil || bytes.Equal(value, nullBytes) {
		if _, err := w.Write(pad); err != nil {
			return err
		}
		_, err := w.WriteString("null")
		return err
	}
	switch isFile {
	case syntax.KindIsDirectory:
		return makeOutDir(w, value, param,
			lookup, pipestancePath, outsPath,
			pad, indent)
	case syntax.KindIsFile:
		if _, err := w.Write(pad); err != nil {
			return err
		}
		// Make sure value is a string
		var filePath string
		if err := json.Unmarshal(value, &filePath); err != nil {
			w.Write(value)
			return err
		}
		return moveOutFile(w, param,
			filePath, pipestancePath, outsPath)
	case syntax.KindMayContainPaths:
		if param.Tname.ArrayDim > 0 {
			if _, err := w.Write(pad); err != nil {
				return err
			}
			var arr []json.RawMessage
			if err := json.Unmarshal(value, &arr); err != nil || len(arr) == 0 {
				return fmtJson(w, value)
			}
			return fmtArray(w, arr, indent)
		}
	}
	if _, err := w.Write(pad); err != nil {
		return err
	}
	// Check if it can be a string, and unmarshal as such if possible.
	return fmtJson(w, value)
}

func fmtJson(w *bytes.Buffer, value json.RawMessage) error {
	var str string
	if json.Unmarshal(value, &str) == nil {
		_, err := w.WriteString(str)
		return err
	}
	// For numbers, arrays, and maps.
	return json.Compact(w, value)
}

// Decend one level down into arrays which might contain file names, so we don't
// make super-long output lines.
func fmtArray(w *bytes.Buffer, arr []json.RawMessage, indent []byte) error {
	if _, err := w.WriteString("[\n"); err != nil {
		return err
	}
	newIndent := append(indent, ' ', ' ')
	for _, elem := range arr {
		if _, err := w.Write(newIndent); err != nil {
			return err
		}
		if err := fmtJson(w, elem); err != nil {
			return err
		}
		if _, err := w.WriteRune('\n'); err != nil {
			return err
		}
	}
	if _, err := w.Write(indent); err != nil {
		return err
	}
	if _, err := w.WriteRune(']'); err != nil {
		return err
	}
	return nil
}

func makeNewIndent(indent []byte, keyLen int) []byte {
	newIndent := indent
	if cap(newIndent) < keyLen || cap(newIndent) < len(indent)+2 {
		if len(indent)+2 > keyLen {
			newIndent = make([]byte, len(indent), 2*len(indent)+2)
		} else {
			newIndent = make([]byte, len(indent), keyLen)
		}
		copy(newIndent, indent)
	}
	return append(newIndent, ' ', ' ')
}

func makeOutDir(w *bytes.Buffer, value json.RawMessage,
	member *syntax.StructMember, lookup *syntax.TypeLookup,
	pipestancePath, outsPath string,
	pad, indent []byte) error {
	t := lookup.Get(member.Tname)
	outPath := path.Join(outsPath, member.GetOutFilename())
	if err := os.Mkdir(outPath, 0775); err != nil {
		fmtJson(w, value)
		return err
	}
	if member.Tname.ArrayDim > 0 {
		if at, ok := t.(*syntax.ArrayType); !ok {
			return fmt.Errorf("expected array, got %T", t)
		} else {
			if _, err := w.Write(pad); err != nil {
				return err
			}
			return makeOutArrayDir(w, value,
				at, member, lookup,
				pipestancePath, outPath,
				indent)
		}
	}
	var valueMap LazyArgumentMap
	if err := json.Unmarshal(value, &valueMap); err != nil {
		fmtJson(w, value)
		return fmt.Errorf("value was not a %s: %v",
			member.Tname.String(), err)
	}

	makePad := func(indent []byte, size int) []byte {
		pad := indent
		if len(pad) > size {
			pad = indent[:size]
		} else {
			for len(pad) < size {
				pad = append(pad, ' ')
			}
		}
		return pad
	}
	var errs syntax.ErrorList
	keyLen := 0
	switch t := t.(type) {
	case *syntax.TypedMapType:
		if _, err := w.Write(pad); err != nil {
			return err
		}
		if len(valueMap) == 0 {
			_, err := w.WriteString("{}")
			return err
		}
		if _, err := w.WriteString("{\n"); err != nil {
			return err
		}
		keys := make([]string, 0, len(valueMap))
		for k := range valueMap {
			if err := syntax.IsLegalUnixFilename(k); err != nil {
				util.PrintError(err, "cannot create out directory %q", k)
			} else {
				keys = append(keys, k)
				if keyLen < len(k) {
					keyLen = len(k)
				}
			}
		}
		sort.Strings(keys)
		newIndent := makeNewIndent(indent, keyLen)
		p := syntax.StructMember{
			Tname: t.Elem.GetId(),
		}
		p.CacheIsFile(t.Elem)
		for _, k := range keys {
			if _, err := w.Write(newIndent); err != nil {
				errs = append(errs, err)
			}
			if _, err := w.WriteString(k); err != nil {
				errs = append(errs, err)
			}
			if _, err := w.WriteRune(':'); err != nil {
				errs = append(errs, err)
			}
			pad := makePad(newIndent, 1+keyLen-len(k))
			p.OutName = k
			if err := handleOutParam(w,
				&p,
				t.Elem.IsFile(),
				valueMap[k],
				lookup,
				pipestancePath,
				outPath,
				pad, newIndent); err != nil {
				errs = append(errs, err)
			}
			if _, err := w.WriteRune('\n'); err != nil {
				errs = append(errs, err)
			}
		}
		if _, err := w.Write(indent); err != nil {
			errs = append(errs, err)
		}
		if _, err := w.WriteRune('}'); err != nil {
			errs = append(errs, err)
		}
	case *syntax.StructType:
		if _, err := w.WriteRune('\n'); err != nil {
			errs = append(errs, err)
		}
		keys := make([]string, len(t.Members))
		for i, m := range t.Members {
			keys[i] = m.GetOutName()
			if len(keys[i]) == 0 {
				keys[i] = m.Id
			}
			if f := len(keys[i]); f > keyLen {
				keyLen = f
			}
		}
		newIndent := makeNewIndent(indent, keyLen)
		for i, k := range keys {
			if i != 0 {
				if _, err := w.WriteRune('\n'); err != nil {
					errs = append(errs, err)
				}
			}
			if _, err := w.Write(newIndent); err != nil {
				errs = append(errs, err)
			}
			if _, err := w.WriteString(k); err != nil {
				errs = append(errs, err)
			}
			if _, err := w.WriteRune(':'); err != nil {
				errs = append(errs, err)
			}
			pad := makePad(newIndent, 1+keyLen-len(k))
			if err := handleOutParam(w,
				t.Members[i],
				t.Members[i].IsFile(),
				valueMap[t.Members[i].Id],
				lookup,
				pipestancePath,
				outPath,
				pad, newIndent); err != nil {
				errs = append(errs, err)
			}
		}
	default:
		errs = append(errs, fmt.Errorf("bad directory type %T", t))
	}
	return errs.If()
}

func makeOutArrayDir(w *bytes.Buffer, value json.RawMessage,
	t *syntax.ArrayType,
	member *syntax.StructMember, lookup *syntax.TypeLookup,
	pipestancePath, outPath string,
	indent []byte) error {
	var valueArr []json.RawMessage
	if err := json.Unmarshal(value, &valueArr); err != nil {
		fmtJson(w, value)
		return fmt.Errorf("value was not a %s: %v",
			member.Tname.String(), err)
	}
	if len(valueArr) == 0 {
		_, err := w.WriteString("[]")
		return err
	}
	if _, err := w.WriteString("[\n"); err != nil {
		return err
	}
	width := util.WidthForInt(len(valueArr))
	newIndent := makeNewIndent(indent, width)
	p := syntax.StructMember{
		Tname: t.Elem.GetId(),
	}
	p.CacheIsFile(t.Elem)
	var errs syntax.ErrorList
	for i, v := range valueArr {
		if _, err := w.Write(newIndent); err != nil {
			errs = append(errs, err)
		}
		k := fmt.Sprintf("%0*d", width, i)
		if _, err := w.WriteString(k); err != nil {
			errs = append(errs, err)
		}
		if _, err := w.WriteRune(':'); err != nil {
			errs = append(errs, err)
		}
		p.Id = k
		if err := handleOutParam(w,
			&p,
			t.Elem.IsFile(),
			v,
			lookup,
			pipestancePath,
			outPath,
			newIndent[:1], newIndent); err != nil {
			errs = append(errs, err)
		}
		if _, err := w.WriteRune('\n'); err != nil {
			errs = append(errs, err)
		}
	}
	if _, err := w.Write(indent); err != nil {
		errs = append(errs, err)
	}
	if _, err := w.WriteRune(']'); err != nil {
		errs = append(errs, err)
	}
	return errs.If()
}

// Move files to the top-level pipestance outs directory.
func moveOutFile(w *bytes.Buffer, param *syntax.StructMember,
	filePath, pipestancePath, outsPath string) error {
	// If file doesn't exist (e.g. stage just didn't create it)
	// then report null
	if info, err := os.Lstat(filePath); os.IsNotExist(err) {
		_, err := w.WriteString("null")
		return err
	} else if info.Mode()&os.ModeSymlink != 0 {
		return copyOutSymlink(w, param, filePath, pipestancePath, outsPath)
	}

	// Generate the outs path for this param
	outPath := path.Join(outsPath, param.GetOutFilename())

	// Only continue if path to be copied is inside the pipestance
	if absFilePath, err := filepath.Abs(filePath); err == nil {
		if absPipestancePath, err := filepath.Abs(pipestancePath); err == nil {
			if !strings.Contains(absFilePath, absPipestancePath) {
				if _, err := w.WriteString(filePath); err != nil {
					return err
				}
				// But we still want a symlink
				return os.Symlink(absFilePath, outPath)
			}
		}
	}

	// If this param has already been moved to outs/, we're done
	if _, err := os.Stat(outPath); err == nil {
		_, err := w.WriteString(filePath)
		return err
	}

	// If source file exists, move it to outs/
	if err := os.Rename(filePath, outPath); err != nil {
		if _, err := w.WriteString(filePath); err != nil {
			return err
		}
		return err
	}

	// Generate the relative path from files/ to outs/
	relPath, err := filepath.Rel(filepath.Dir(filePath), outPath)
	if err != nil {
		if _, err := w.WriteString(filePath); err != nil {
			return err
		}
		return err
	}

	// Symlink it back to the original files/ folder
	if err := os.Symlink(relPath, filePath); err != nil {
		if _, err := w.WriteString(filePath); err != nil {
			return err
		}
		return err
	}

	if _, err := w.WriteString(outPath); err != nil {
		return err
	}
	return err
}

// Copies a symlink to the outs directory.  If the symlink was absolute, this
// is simple, but if it was relative it needs to be converted to be relative
// to the location in the outs directory.
func copyOutSymlink(w *bytes.Buffer, param *syntax.StructMember,
	filePath, pipestancePath, outsPath string) error {
	// Generate the outs path for this param
	outPath := path.Join(outsPath, param.GetOutFilename())

	// Only continue if path to be copied is inside the pipestance
	if absFilePath, err := filepath.Abs(filePath); err == nil {
		if absPipestancePath, err := filepath.Abs(pipestancePath); err == nil {
			if !strings.Contains(absFilePath, absPipestancePath) {
				if _, err := w.WriteString(filePath); err != nil {
					return err
				}
				// But we still want a symlink
				return os.Symlink(absFilePath, outPath)
			}
		}
	}

	// If this param has already been moved to outs/, we're done
	if _, err := os.Stat(outPath); err == nil {
		_, err := w.WriteString(filePath)
		return err
	}
	if p, err := os.Readlink(filePath); err != nil {
		return err
	} else if filepath.IsAbs(p) {
		if _, err := w.WriteString(outPath); err != nil {
			return err
		}
		return os.Symlink(p, outPath)
	} else if rel, err := filepath.Rel(filepath.Dir(outPath),
		filepath.Join(filepath.Dir(filePath), p)); err != nil {
		return err
	} else {
		if _, err := w.WriteString(outPath); err != nil {
			return err
		}
		return os.Symlink(rel, outPath)
	}
}
