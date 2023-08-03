//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Post-processing logic for pipelines.
//

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func (self *Fork) postProcess(ctx context.Context) error {
	defer trace.StartRegion(ctx, "Fork_postProcess").End()

	ro := self.node.call.ResolvedOutputs()
	if ro == nil {
		return nil
	}
	if rro, err := ro.BindingPath("", self.forkId.SourceIndexMap(),
		self.node.top.types); err != nil {
		return err
	} else {
		ro = rro
	}

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

	var errs syntax.ErrorList
	var newOuts json.Marshaler
	switch ro.Type.(type) {
	case *syntax.ArrayType:
		var arr []json.RawMessage
		if err := self.metadata.ReadInto(OutsFile, &arr); err != nil {
			errs = append(errs, err)
		}
		noutArr := make(marshallerArray, len(arr))
		for i, elem := range arr {
			k := strconv.Itoa(i)
			util.Print("Fork %s:\n", k)
			nout, err := self.processStructOuts(pipestancePath,
				path.Join(outsPath, k), elem)
			if err != nil {
				errs = append(errs, err)
			}
			noutArr[i] = nout
		}
		newOuts = noutArr
	case *syntax.TypedMapType:
		var outs LazyArgumentMap
		if err := self.metadata.ReadInto(OutsFile, &outs); err != nil {
			errs = append(errs, err)
		}
		noutMap := make(MarshalerMap, len(outs))
		for k, elem := range outs {
			util.Print("Fork \"%s\":\n", k)
			nout, err := self.processStructOuts(pipestancePath,
				path.Join(outsPath, k), elem)
			if err != nil {
				errs = append(errs, err)
			}
			noutMap[k] = nout
		}
		newOuts = noutMap
	default:
		if b, err := self.metadata.readRawBytes(OutsFile); err != nil {
			errs = append(errs, err)
		} else {
			newOuts, err = self.processStructOuts(
				pipestancePath, outsPath, b)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	// Rewrite the outs json file with the updated locations.
	if err := self.metadata.WriteAtomic(OutsFile, newOuts); err != nil {
		errs = append(errs, err)
	}
	self.printAlarms()
	return errs.If()
}

func (self *Fork) processStructOuts(pipestancePath, outsPath string,
	outputs json.RawMessage) (LazyArgumentMap, error) {
	var errs syntax.ErrorList
	paramList := self.OutParams().List
	for _, p := range paramList {
		if k := p.IsFile(); k == syntax.KindIsFile || k == syntax.KindIsDirectory {
			// Create the fork-specific outs/ folder
			if err := util.MkdirAll(outsPath); err != nil {
				errs = append(errs, err)
			}
			break
		}
	}

	// Get fork's output parameter values
	var outs LazyArgumentMap
	if err := json.Unmarshal(outputs, &outs); err != nil {
		errs = append(errs, err)
	}

	newOuts, oerrs := self.handleOuts(paramList, outs, pipestancePath, outsPath)
	if len(oerrs) != 0 {
		if len(errs) == 0 {
			errs = oerrs
		} else {
			errs = append(errs, oerrs...)
		}
	}
	util.Print("\n")

	return newOuts, errs.If()
}

func (self *Fork) handleOuts(paramList []*syntax.OutParam,
	outs LazyArgumentMap,
	pipestancePath, outsPath string) (LazyArgumentMap, syntax.ErrorList) {
	var errs syntax.ErrorList

	// Calculate longest key name for alignment
	keyWidth := 0
	// Move output files into the pipestance outs directory.
	newOuts := make(LazyArgumentMap, len(paramList))
	for _, param := range paramList {
		out := outs[param.Id]

		k := param.IsFile()
		if out != nil {
			switch k {
			case syntax.KindIsDirectory, syntax.KindIsFile:
				if !bytes.Equal(out, nullBytes) {
					var result bytes.Buffer
					result.Grow(len(out))
					err := moveOutFiles(&result,
						&param.StructMember, k, out, self.node.top.types,
						pipestancePath, outsPath)
					if err != nil {
						errs = append(errs, err)
					}
					newOuts[param.Id] = result.Bytes()
				} else {
					newOuts[param.Id] = out
				}
			default:
				newOuts[param.Id] = out
			}
		}
		// Print out the param help and value
		if kw := len(param.GetDisplayName()); kw > keyWidth {
			keyWidth = kw
		}
	}

	indent := bytes.Repeat([]byte{' '}, keyWidth+1) // "- keyWidth: "

	var result bytes.Buffer
	// Iterate through output parameters
	for _, param := range paramList {
		id := param.GetId()
		key := param.GetDisplayName()
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
		value := newOuts[id]
		if err := printOutParam(&result, &param.StructMember,
			param.IsFile(), value,
			self.node.top.types,
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
	return newOuts, errs
}

func moveOutFiles(w *bytes.Buffer, param *syntax.StructMember,
	isFile syntax.FileKind,
	value json.RawMessage,
	lookup *syntax.TypeLookup,
	pipestancePath, outsPath string) error {
	if value == nil || bytes.Equal(value, nullBytes) {
		_, err := w.Write(nullBytes)
		return err
	}
	switch isFile {
	case syntax.KindIsFile:
		return moveOutFile(w, param,
			value, pipestancePath, outsPath)
	case syntax.KindIsDirectory:
		return moveOutDir(w,
			value, param, lookup,
			pipestancePath, outsPath)
	default:
		_, err := w.Write(value)
		return err
	}
}

func moveOutDir(w *bytes.Buffer, value json.RawMessage,
	member *syntax.StructMember, lookup *syntax.TypeLookup,
	pipestancePath, outsPath string) error {
	t := lookup.Get(member.Tname)
	outPath := path.Join(outsPath, member.GetOutFilename())
	if member.Tname.ArrayDim > 0 {
		if at, ok := t.(*syntax.ArrayType); !ok {
			if _, err := w.Write(value); err != nil {
				return err
			}
			return fmt.Errorf("expected array, got %T", t)
		} else {
			return moveOutArrayDir(w, value,
				at, member, lookup,
				pipestancePath, outPath)
		}
	}
	var valueMap LazyArgumentMap
	if err := json.Unmarshal(value, &valueMap); err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return fmt.Errorf("value was not a %s: %v",
			member.Tname.String(), err)
	}
	if len(valueMap) == 0 {
		_, err := w.WriteString("{}")
		return err
	}
	if _, err := w.WriteRune('{'); err != nil {
		return err
	}
	var errs syntax.ErrorList
	writeKey := func(i int, k string) {
		if i != 0 {
			if _, err := w.WriteRune(','); err != nil {
				errs = append(errs, err)
			}
		}
		if b, err := json.Marshal(k); err != nil {
			errs = append(errs, err)
			if _, err := w.WriteString("<err>"); err != nil {
				errs = append(errs, err)
			}
		} else if _, err := w.Write(b); err != nil {
			errs = append(errs, err)
		}
		if _, err := w.WriteRune(':'); err != nil {
			errs = append(errs, err)
		}
	}
	switch t := t.(type) {
	case *syntax.TypedMapType:
		keys := make([]string, 0, len(valueMap))
		for k := range valueMap {
			if err := syntax.IsLegalUnixFilename(k); err != nil {
				util.PrintError(err, "cannot create out directory %q", k)
			} else {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		p := syntax.StructMember{
			Tname: t.Elem.TypeId(),
		}
		p.CacheIsFile(t.Elem)
		for i, k := range keys {
			writeKey(i, k)
			p.Id = k
			if err := moveOutFiles(w,
				&p,
				t.Elem.IsFile(),
				valueMap[k],
				lookup,
				pipestancePath,
				outPath); err != nil {
				errs = append(errs, err)
			}
		}
	case *syntax.StructType:
		keys := make([]string, len(t.Members))
		for i, m := range t.Members {
			keys[i] = m.Id
		}
		sort.Strings(keys)
		for i, k := range keys {
			writeKey(i, k)
			m := t.Table[k]
			if err := moveOutFiles(w,
				m,
				m.IsFile(),
				valueMap[k],
				lookup,
				pipestancePath,
				outPath); err != nil {
				errs = append(errs, err)
			}
		}
	default:
		errs = append(errs, fmt.Errorf("bad directory type %T", t))
	}
	if _, err := w.WriteRune('}'); err != nil {
		errs = append(errs, err)
	}
	return errs.If()
}

func moveOutArrayDir(w *bytes.Buffer, value json.RawMessage,
	t *syntax.ArrayType,
	member *syntax.StructMember, lookup *syntax.TypeLookup,
	pipestancePath, outPath string) error {
	var valueArr []json.RawMessage
	if err := json.Unmarshal(value, &valueArr); err != nil {
		if err := fmtJson(w, value); err != nil {
			return err
		}
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
	p := syntax.StructMember{
		Tname: t.Elem.TypeId(),
	}
	p.CacheIsFile(t.Elem)
	width := util.WidthForInt(len(valueArr))
	var errs syntax.ErrorList
	for i, v := range valueArr {
		if i != 0 {
			if _, err := w.WriteRune(','); err != nil {
				errs = append(errs, err)
			}
		}
		k := fmt.Sprintf("%0*d", width, i)
		p.Id = k
		if err := moveOutFiles(w,
			&p,
			t.Elem.IsFile(),
			v,
			lookup,
			pipestancePath,
			outPath); err != nil {
			errs = append(errs, err)
		}
	}
	if _, err := w.WriteRune(']'); err != nil {
		errs = append(errs, err)
	}
	return errs.If()
}

// Move files to the top-level pipestance outs directory.
func moveOutFile(w *bytes.Buffer, param *syntax.StructMember,
	value json.RawMessage, pipestancePath, outsPath string) error {
	var filePath string
	if err := json.Unmarshal(value, &filePath); err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	}
	if filePath == "" {
		_, err := w.Write(nullBytes)
		return err
	}
	// If file doesn't exist (e.g. stage just didn't create it)
	// then report null
	if info, err := os.Lstat(filePath); os.IsNotExist(err) {
		_, err := w.Write(nullBytes)
		return err
	} else if err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	} else if info.Mode()&os.ModeSymlink != 0 {
		if err := os.MkdirAll(outsPath, 0775); err != nil {
			return err
		}
		// The source is a symlink, so we will put a symlink in outs/
		return copyOutSymlink(w, param, value, filePath, pipestancePath, outsPath)
	}

	// Generate the outs path for this param
	outPath := path.Join(outsPath, param.GetOutFilename())

	// Only continue if path to be copied is inside the pipestance
	if absFilePath, err := filepath.Abs(filePath); err == nil {
		if absPipestancePath, err := filepath.Abs(pipestancePath); err == nil {
			if !strings.Contains(absFilePath, absPipestancePath) {
				if _, err := w.Write(value); err != nil {
					return err
				}
				if err := os.MkdirAll(outsPath, 0775); err != nil {
					return err
				}
				// But we still want a symlink in outs/
				return os.Symlink(absFilePath, outPath)
			}
		}
	}

	// If this param has already been moved to outs/, we're done
	if _, err := os.Stat(outPath); err == nil {
		_, err := w.Write(value)
		return err
	}
	if err := os.MkdirAll(outsPath, 0775); err != nil {
		return err
	}
	// If source file exists, move it to outs/
	if err := os.Rename(filePath, outPath); err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	}

	// Generate the relative path from files/ to outs/
	relPath, err := filepath.Rel(filepath.Dir(filePath), outPath)
	if err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	}

	// Symlink it back to the original files/ folder
	if err := os.Symlink(relPath, filePath); err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	}

	if b, err := json.Marshal(outPath); err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	} else if _, err := w.Write(b); err != nil {
		return err
	}
	return err
}

// Copies a symlink to the outs directory.  If the symlink was absolute, this
// is simple, but if it was relative it needs to be converted to be relative
// to the location in the outs directory.
func copyOutSymlink(w *bytes.Buffer, param *syntax.StructMember,
	value json.RawMessage, filePath, pipestancePath, outsPath string) error {
	// Generate the outs path for this param
	outPath := path.Join(outsPath, param.GetOutFilename())

	// Only continue if path to be copied is inside the pipestance
	if absFilePath, err := filepath.Abs(filePath); err == nil {
		if absPipestancePath, err := filepath.Abs(pipestancePath); err == nil {
			if !strings.Contains(absFilePath, absPipestancePath) {
				if _, err := w.Write(value); err != nil {
					return err
				}
				// But we still want a symlink
				return os.Symlink(absFilePath, outPath)
			}
		}
	}

	// If this param has already been moved to outs/, we're done
	if _, err := os.Stat(outPath); err == nil {
		if b, err := json.Marshal(outPath); err != nil {
			if _, err := w.Write(value); err != nil {
				return err
			}
			return err
		} else {
			_, err := w.Write(b)
			return err
		}
	}
	var p string
	p, err := os.Readlink(filePath)
	if err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	}
	ap := p
	if !filepath.IsAbs(ap) {
		ap = filepath.Clean(filepath.Join(filepath.Dir(filePath), p))
		// resolve when the pointed-to file is a symlink, but unlike
		// filepath.EvalSymlinks we don't want to walk up the tree.
		for rp, err := os.Readlink(ap); err == nil; rp, err = os.Readlink(ap) {
			p = ap
			if filepath.IsAbs(rp) {
				ap = rp
				break
			} else {
				ap = filepath.Clean(filepath.Join(filepath.Dir(ap), rp))
			}
		}
	}
	if pb, err := json.Marshal(ap); err != nil {
		if _, err := w.Write(value); err != nil {
			return err
		}
		return err
	} else {
		// Use the destination of the symlink, not the location in outs.
		if _, err := w.Write(pb); err != nil {
			return err
		}
		if filepath.IsAbs(p) {
			return os.Symlink(p, outPath)
		} else if rel, err := filepath.Rel(filepath.Dir(outPath), ap); err != nil {
			return err
		} else {
			return os.Symlink(rel, outPath)
		}
	}
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

func printOutParam(w *bytes.Buffer, param *syntax.StructMember,
	isFile syntax.FileKind,
	value json.RawMessage, lookup *syntax.TypeLookup,
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
		return printOutDir(w, value, param,
			lookup, pad, indent)
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
		_, err := w.WriteString(filePath)
		return err
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

func printOutDir(w *bytes.Buffer, value json.RawMessage,
	member *syntax.StructMember, lookup *syntax.TypeLookup,
	pad, indent []byte) error {
	t := lookup.Get(member.Tname)
	if member.Tname.ArrayDim > 0 {
		if at, ok := t.(*syntax.ArrayType); !ok {
			return fmt.Errorf("expected array, got %T", t)
		} else {
			if _, err := w.Write(pad); err != nil {
				return err
			}
			return printOutArrayDir(w, value,
				at, member, lookup,
				indent)
		}
	}
	var valueMap LazyArgumentMap
	if err := json.Unmarshal(value, &valueMap); err != nil {
		if err := fmtJson(w, value); err != nil {
			return err
		}
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
			keys = append(keys, k)
			if keyLen < len(k) {
				keyLen = len(k)
			}
		}
		sort.Strings(keys)
		newIndent := makeNewIndent(indent, keyLen)
		p := syntax.StructMember{
			Tname: t.Elem.TypeId(),
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
			if err := printOutParam(w,
				&p,
				t.Elem.IsFile(),
				valueMap[k],
				lookup,
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
			keys[i] = m.GetDisplayName()
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
			if err := printOutParam(w,
				t.Members[i],
				t.Members[i].IsFile(),
				valueMap[t.Members[i].Id],
				lookup,
				pad, newIndent); err != nil {
				errs = append(errs, err)
			}
		}
	default:
		errs = append(errs, fmt.Errorf("bad directory type %T", t))
	}
	return errs.If()
}

func printOutArrayDir(w *bytes.Buffer, value json.RawMessage,
	t *syntax.ArrayType,
	member *syntax.StructMember, lookup *syntax.TypeLookup,
	indent []byte) error {
	var valueArr []json.RawMessage
	if err := json.Unmarshal(value, &valueArr); err != nil {
		if err := fmtJson(w, value); err != nil {
			return err
		}
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
		Tname: t.Elem.TypeId(),
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
		if err := printOutParam(w,
			&p,
			t.Elem.IsFile(),
			v,
			lookup,
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
