//go:build go1.27

// Due to changes in how go's encoding/json package handles invalid UTF-8
// in go 1.27, some tests won't work on older versions of go.
// To simplify their implementation, we have this constant defined in
// build-constrained files.

package syntax

const isGo127 = true
