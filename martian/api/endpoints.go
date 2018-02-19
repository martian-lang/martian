//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// API query paths.

package api // import "github.com/martian-lang/martian/martian/api"

const (
	// Gets top-level information about a pipestance.
	QueryGetInfo = "/api/get-info"

	// Gets top-level information about a pipestance and all of its nodes.
	QueryGetState = "/api/get-state"

	// Gets information about a pipestance's performance.
	QueryGetPerf = "/api/get-perf"

	// Get the contents of a specific metadata file.
	QueryGetMetadata = "/api/get-metadata"

	// Restarts a failed pipestance.
	QueryRestart = "/api/restart"

	// Get the contents of a pipestance's top-level metadata.
	QueryGetMetadataTop = "/api/get-metadata-top/"

	// Terminate a running pipestance.
	QueryKill = "/api/kill"

	// Register an instance of mrp with an mrv host.
	QueryRegisterMrv = "/register"

	// Register (or re-register) an instance of mrp with an Enterprise host.
	QueryRegisterEnterprise = "/api/register"
)
