//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

package api

// Information required to query metadata for a specific pipestance.
type MetadataForm struct {
	Path string `json:"path"`
	Name string `json:"name"`
}
