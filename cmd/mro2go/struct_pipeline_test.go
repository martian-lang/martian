// Code generated by mro2go testdata/struct_pipeline.mro; DO NOT EDIT.

package main

// A structure to encode and decode the STUFF struct.
type Stuff struct {
	Bar int `json:"bar"`
	// file
	File1 string `json:"file1"`
	// txt file
	File2 string `json:"file2"`
}

// A structure to encode and decode the CREATOR struct.
type Creator struct {
	Bar *Stuff `json:"bar"`
	// help text
	//
	// output_name.file
	//
	// txt file
	File3 string `json:"file3"`
}

// Called by OUTER.
type Inner struct {
	Bar *Creator `json:"bar"`
	// description
	//
	// another_file.txt
	//
	// txt file
	OutFile  string              `json:"out_file"`
	Results1 map[string]*Creator `json:"results1"`
	// help text
	//
	// output_name
	Results2 map[string]*Creator `json:"results2"`
}

//
// OUTER
//

// A structure to encode and decode args to the OUTER pipeline.
type OuterArgs struct {
	Foo int `json:"foo"`
}

// CallName returns the name of this pipeline as defined in the .mro file.
func (*OuterArgs) CallName() string {
	return "OUTER"
}

// MroFileName returns the name of the .mro file which defines this pipeline.
func (*OuterArgs) MroFileName() string {
	return "testdata/struct_pipeline.mro"
}

// A structure to encode and decode outs from the OUTER pipeline.
type OuterOuts struct {
	// txt file
	Text  string `json:"text"`
	Inner *Inner `json:"inner"`
}
