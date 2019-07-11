// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

//
// Struct
//
func (self *StructType) format(printer *printer) {
	printer.printComments(&self.Node, "")

	typeWidth := 0
	idWidth := 0
	for _, m := range self.Members {
		typeWidth = max(typeWidth, m.Tname.strlen())
		idWidth = max(idWidth, len(m.Id))
	}

	printer.Printf("struct %s(\n", self.Id)
	for _, m := range self.Members {
		m.format(printer, typeWidth, idWidth)
	}
	printer.mustWriteString(")\n")
}

func (member *StructMember) format(printer *printer, typeWidth int, idWidth int) {
	printer.printComments(member.getNode(), INDENT)

	// Common columns up to type name.
	printer.mustWriteString(INDENT)
	member.Tname.writeTo(printer)
	for i := member.Tname.strlen(); i < typeWidth; i++ {
		printer.mustWriteRune(' ')
	}
	printer.mustWriteRune(' ')
	printer.mustWriteString(member.Id)
	if member.OutName != "" {
		for i := len(member.Id); i < idWidth; i++ {
			printer.mustWriteRune(' ')
		}
		printer.mustWriteRune(' ')
		quoteString(printer, member.OutName)
	}
	printer.mustWriteString(",\n")
}

//
// Filetype
//
func (self *UserType) format(printer *printer) {
	printer.printComments(&self.Node, "")
	printer.Printf("filetype %s;\n", self.Id)
}
