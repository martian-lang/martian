// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

//
// Binding
//
func (self *BindStm) format(printer *printer, prefix string, idWidth int) {
	printer.printComments(self.getNode(), prefix+INDENT)
	printer.printComments(self.Exp.getNode(), prefix+INDENT)

	printer.mustWriteString(prefix)
	printer.mustWriteString(INDENT)
	printer.mustWriteString(self.Id)
	for i := len(self.Id); i < idWidth; i++ {
		printer.mustWriteRune(' ')
	}
	printer.mustWriteString(" = ")
	self.Exp.format(printer, prefix+INDENT)
	printer.mustWriteRune(',')
	printer.mustWriteString(NEWLINE)
}

func (self *BindStms) format(printer *printer, prefix string) {
	printer.printComments(self.getNode(), prefix)
	idWidth := 0
	for _, bindstm := range self.List {
		if len(bindstm.Id) < 30 {
			idWidth = max(idWidth, len(bindstm.Id))
		}
		if bindstm.Id == "*" {
			break
		}
	}
	for _, bindstm := range self.List {
		bindstm.format(printer, prefix, idWidth)
		if bindstm.Id == "*" {
			break
		}
	}
}

//
// Parameter
//
func paramFormat(printer *printer, param Param, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	printer.printComments(param.getNode(), INDENT)
	id := param.GetId()
	if id == "default" {
		id = ""
	}

	// Generate column alignment paddings.
	tname := param.GetTname()
	typePad := strings.Repeat(" ", typeWidth-tname.strlen())

	// Common columns up to type name.
	printer.mustWriteString(INDENT)
	printer.mustWriteString(param.getMode())
	for i := len(param.getMode()); i < modeWidth; i++ {
		printer.mustWriteRune(' ')
	}
	printer.mustWriteRune(' ')
	printer.mustWriteString(tname.str())

	// Add id if not default.
	if id != "" {
		printer.mustWriteString(typePad)
		printer.mustWriteRune(' ')
		printer.mustWriteString(id)
	}

	// Add help string if it exists.
	if len(param.GetHelp()) > 0 {
		if id == "" {
			printer.mustWriteString(typePad)
			printer.mustWriteRune(' ')
		}
		for i := len(id); i < idWidth; i++ {
			printer.mustWriteRune(' ')
		}
		printer.mustWriteString(`  "`)
		printer.mustWriteString(param.GetHelp())
		printer.mustWriteRune('"')
	}

	// Add outname string if it exists.
	if len(param.GetOutName()) > 0 {
		if param.GetHelp() == "" {
			for i := len(id); i < idWidth; i++ {
				printer.mustWriteRune(' ')
			}
			printer.mustWriteString(`  ""`)
		}
		for i := len(param.GetHelp()); i < helpWidth; i++ {
			printer.mustWriteRune(' ')
		}
		printer.mustWriteString(`  "`)
		printer.mustWriteString(param.GetOutName())
		printer.mustWriteRune('"')
	}
	printer.mustWriteString(",\n")
}

func (self *InParams) getWidths() (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, param := range self.List {
		modeWidth = max(modeWidth, len(param.getMode()))
		tname := param.GetTname()
		typeWidth = max(typeWidth, tname.strlen())
		if len(param.GetId()) < 35 {
			idWidth = max(idWidth, len(param.GetId()))
		}
		if len(param.GetHelp()) < 25 {
			helpWidth = max(helpWidth, len(param.GetHelp()))
		}
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func (self *OutParams) getWidths() (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, param := range self.List {
		modeWidth = max(modeWidth, len(param.getMode()))
		tname := param.GetTname()
		typeWidth = max(typeWidth, tname.strlen())
		if len(param.GetId()) < 35 {
			idWidth = max(idWidth, len(param.GetId()))
		}
		if len(param.GetHelp()) < 25 {
			helpWidth = max(helpWidth, len(param.GetHelp()))
		}
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func measureParamsWidths(paramsList ...Params) (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, params := range paramsList {
		mw, tw, iw, hw := params.getWidths()
		modeWidth = max(modeWidth, mw)
		typeWidth = max(typeWidth, tw)
		idWidth = max(idWidth, iw)
		helpWidth = max(helpWidth, hw)
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func (self *InParams) format(printer *printer, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	for _, param := range self.List {
		paramFormat(printer, param, modeWidth, typeWidth, idWidth, helpWidth)
	}
}

func (self *OutParams) format(printer *printer, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	for _, param := range self.List {
		paramFormat(printer, param, modeWidth, typeWidth, idWidth, helpWidth)
	}
}

//
// Pipeline, Call, Return
//
func (self *Pipeline) format(printer *printer) {
	printer.printComments(&self.Node, "")

	modeWidth, typeWidth, idWidth, helpWidth := measureParamsWidths(
		self.InParams, self.OutParams,
	)

	printer.mustWriteString("pipeline ")
	printer.mustWriteString(self.Id)
	printer.mustWriteString("(\n")
	self.InParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.OutParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	printer.mustWriteString(")\n{")
	if err := self.topoSort(); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: formatting pipeline ",
			self.Id, ": ", err)
	}
	for _, callstm := range self.Calls {
		printer.mustWriteString(NEWLINE)
		callstm.format(printer, INDENT)
	}
	printer.mustWriteString(NEWLINE)
	self.Ret.format(printer)
	if self.Retain != nil {
		printer.mustWriteString(NEWLINE)
		self.Retain.format(printer)
	}
	printer.mustWriteString("}\n")
}

func (self *CallStm) format(printer *printer, prefix string) {
	printer.printComments(&self.Node, prefix)
	printer.mustWriteString(prefix)
	if self.CallMode() != ModeSingleCall {
		printer.mustWriteString("map ")
	}
	printer.mustWriteString("call ")
	printer.mustWriteString(self.DecId)
	if self.Id != self.DecId {
		printer.mustWriteString(" as ")
		printer.mustWriteString(self.Id)
	}
	printer.mustWriteRune('(')
	if len(self.Bindings.List) > 0 {
		printer.mustWriteRune('\n')
		self.Bindings.format(printer, prefix)
		printer.mustWriteString(prefix)
	}

	if self.Modifiers != nil && (self.Modifiers.Bindings != nil &&
		len(self.Modifiers.Bindings.List) > 0 ||
		self.Modifiers.Local || self.Modifiers.Preflight || self.Modifiers.Volatile) {
		if self.Modifiers.Bindings == nil {
			self.Modifiers.Bindings = &BindStms{
				Node: self.Node,
			}
		}
		printer.mustWriteString(") using (\n")
		// Convert unbound-form mods to bound form.
		// Because we remove elements from the binding table if they're
		// static, we can't just use the table to see if they're needed.
		var foundMods Modifiers
		for _, binding := range self.Modifiers.Bindings.List {
			switch binding.Id {
			case local:
				foundMods.Local = true
			case preflight:
				foundMods.Preflight = true
			case volatile:
				foundMods.Volatile = true
			}
		}
		if self.Modifiers.Local && !foundMods.Local {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "local",
					Exp: &BoolExp{
						valExp: valExp{Node: self.Modifiers.Bindings.Node},
						Value:  true,
					},
				})
		}
		if self.Modifiers.Preflight && !foundMods.Preflight {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "preflight",
					Exp: &BoolExp{
						valExp: valExp{Node: self.Modifiers.Bindings.Node},
						Value:  true,
					},
				})
		}
		if self.Modifiers.Volatile && !foundMods.Volatile {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "volatile",
					Exp: &BoolExp{
						valExp: valExp{Node: self.Modifiers.Bindings.Node},
						Value:  true,
					},
				})
		}
		sort.Slice(self.Modifiers.Bindings.List, func(i, j int) bool {
			return self.Modifiers.Bindings.List[i].Id < self.Modifiers.Bindings.List[j].Id
		})
		self.Modifiers.Bindings.format(printer, prefix)
		printer.mustWriteString(prefix)
	}
	printer.mustWriteString(")\n")
}

func (self *ReturnStm) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.mustWriteString(INDENT)
	printer.mustWriteString("return (\n")
	self.Bindings.format(printer, INDENT)
	printer.mustWriteString(INDENT)
	printer.mustWriteString(")\n")
}

func (self *PipelineRetains) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.mustWriteString(INDENT)
	printer.mustWriteString("retain (\n")
	for _, ref := range self.Refs {
		printer.mustWriteString(INDENT)
		printer.mustWriteString(INDENT)
		ref.format(printer, INDENT+INDENT)
		printer.mustWriteString(",\n")
	}
	printer.mustWriteString(INDENT)
	printer.mustWriteString(")\n")
}

//
// Stage
//
func (self *Stage) format(printer *printer) {
	printer.printComments(&self.Node, "")

	modeWidth, typeWidth, idWidth, helpWidth := measureParamsWidths(
		self.InParams, self.OutParams, self.ChunkIns, self.ChunkOuts,
	)
	modeWidth = max(modeWidth, len("src"))

	printer.mustWriteString("stage ")
	printer.mustWriteString(self.Id)
	printer.mustWriteString("(\n")
	self.InParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.OutParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.Src.format(printer, modeWidth, typeWidth, idWidth)
	if idWidth > 30 || helpWidth > 20 {
		_, _, idWidth, helpWidth = measureParamsWidths(
			self.ChunkIns, self.ChunkOuts)
	}
	if self.Split {
		printer.mustWriteString(") split (\n")
		self.ChunkIns.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
		self.ChunkOuts.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	}
	if self.Resources != nil {
		self.Resources.format(printer)
	}
	if self.Retain != nil {
		self.Retain.format(printer)
	}
	printer.mustWriteString(")\n")
}

func (self *Resources) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.mustWriteString(") using (\n")
	// Pad depending on which arguments are present.
	// mem_gb   = x,
	// special  = y
	// threads  = y,
	// volatile = z,
	var memPad, threadPad string
	if self.VolatileNode != nil {
		memPad = "  "
		threadPad = " "
	} else if self.VMemNode != nil ||
		self.SpecialNode != nil ||
		self.ThreadNode != nil {
		memPad = " "
	}
	if self.MemNode != nil {
		printer.printComments(self.MemNode, INDENT)
		printer.mustWriteString(INDENT)
		printer.mustWriteString("mem_gb")
		printer.mustWriteString(memPad)
		printer.mustWriteString(" = ")
		formatGB(&printer.buf, self.MemGB)
		printer.mustWriteString(",\n")
	}
	if self.SpecialNode != nil {
		printer.printComments(self.SpecialNode, INDENT)
		printer.mustWriteString(INDENT)
		printer.mustWriteString("special")
		printer.mustWriteString(threadPad)
		printer.mustWriteString(` = "`)
		printer.mustWriteString(self.Special)
		printer.mustWriteString("\",\n")
	}
	if self.ThreadNode != nil {
		printer.printComments(self.ThreadNode, INDENT)
		printer.mustWriteString(INDENT)
		printer.mustWriteString("threads")
		printer.mustWriteString(threadPad)
		printer.Printf(" = %g,\n", self.Threads)
	}
	if self.VMemNode != nil {
		printer.printComments(self.VMemNode, INDENT)
		printer.mustWriteString(INDENT)
		printer.mustWriteString("vmem_gb")
		printer.mustWriteString(threadPad)
		printer.mustWriteString(" = ")
		formatGB(&printer.buf, self.VMemGB)
		printer.mustWriteString(",\n")
	}
	if self.VolatileNode != nil {
		printer.printComments(self.VolatileNode, INDENT)
		printer.mustWriteString(INDENT)
		if self.StrictVolatile {
			printer.mustWriteString("volatile = strict,\n")
		} else {
			printer.mustWriteString("volatile = false,\n")
		}
	}
}

// formatGB prints a floating point value, without exponential representation,
// using the minimum number of decimal digits required such that
//
//     formatGB(buf, roundUpTo(gb, 1024))
//     return roundUpTo(parseFloat32(buf.Bytes()), 1024)
//
// leaves the value unchanged.
func formatGB(buf *strings.Builder, gb float32) {
	if gb == 0 {
		if err := buf.WriteByte('0'); err != nil {
			panic(err)
		}
		return
	} else if gb < 0 {
		if err := buf.WriteByte('-'); err != nil {
			panic(err)
		}
		gb = -gb
	}
	mb := int64(gb * 1024)
	var b [20]byte
	if _, err := buf.Write(strconv.AppendInt(b[:0], mb/1024, 10)); err != nil {
		panic(err)
	}
	mb = mb % 1024
	if mb == 0 {
		return
	}
	// The fractional part of a GB, as a fixed point number 0.XXXX
	// Under all circumstances, 4 digits is sufficient accuracy.
	decFrac := (mb * 10000) / 1024
	digits := 4
	scale := int64(10000)
	// Figure out how many digits are actually required.
	for digits > 0 && ((decFrac-decFrac%10)*1024+scale-1)/scale == mb {
		scale /= 10
		decFrac /= 10
		digits--
	}
	if digits == 0 {
		return
	}
	if err := buf.WriteByte('.'); err != nil {
		panic(err)
	}
	for i := digits - 1; i >= 0; i-- {
		v := byte(decFrac % 10)
		if v == 0 && i+1 == digits {
			digits = i
		}
		b[i] = v + '0'
		decFrac = decFrac / 10
	}
	if _, err := buf.Write(b[:digits]); err != nil {
		panic(err)
	}
}

func (self *RetainParams) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.mustWriteString(") retain (\n")
	for _, param := range self.Params {
		printer.printComments(&param.Node, INDENT)
		printer.mustWriteString(INDENT)
		printer.mustWriteString(param.Id)
		printer.mustWriteString(",\n")
	}
}

func (self *SrcParam) format(printer *printer, modeWidth int, typeWidth int, idWidth int) {
	printer.printComments(&self.Node, INDENT)
	printer.mustWriteString(INDENT)
	printer.mustWriteString("src ")
	for i := 0; i < modeWidth-len("src"); i++ {
		printer.mustWriteRune(' ')
	}
	printer.mustWriteString(string(self.Lang))
	for i := 0; i < typeWidth-len(string(self.Lang)); i++ {
		printer.mustWriteRune(' ')
	}
	printer.mustWriteString(` "`)
	printer.mustWriteString(self.Path)
	for _, arg := range self.Args {
		printer.mustWriteRune(' ')
		printer.mustWriteString(arg)
	}
	printer.mustWriteString("\",\n")
}

//
// Callable
//
func (self *Callables) format(printer *printer) {
	if self == nil {
		return
	}
	for i, callable := range self.List {
		if i != 0 {
			printer.mustWriteString(NEWLINE)
		}
		callable.format(printer)
	}
}
