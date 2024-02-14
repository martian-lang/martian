//go:build !go1.22

// Due to changes in how go's encoding/json package handles `\b` and `\f`
// in go 1.22, some tests won't work on older versions of go.
// To simplify their implementation, we have this constant defined in
// build-constrained files.

package syntax

const isGo122 = false
