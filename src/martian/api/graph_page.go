//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

package api

// Information needed to process the graph web page template.
type GraphPage struct {
	InstanceName string
	Container    string
	Pname        string
	Psid         string
	Admin        bool
	AdminStyle   bool
	Release      bool
	Auth         string
}
