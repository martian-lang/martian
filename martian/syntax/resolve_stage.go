// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Code for resolving the pipeline graph.

package syntax

import (
	"sort"
	"strings"
)

// CallGraphStage represents a stage in a call graph.
type CallGraphStage struct {
	Parent  *CallGraphPipeline `json:"-"`
	Fqid    string             `json:"fqid"`
	call    *CallStm
	stage   *Stage
	Inputs  ResolvedBindingMap `json:"inputs"`
	Outputs *ResolvedBinding   `json:"outputs"`
	Disable []Exp              `json:"disabled,omitempty"`
	Forks   ForkRootList       `json:"fork_roots,omitempty"`
	Source  MapCallSource      `json:"-"`
}

// Kind returns KindStage.
func (c *CallGraphStage) Kind() CallGraphNodeType {
	return KindStage
}

// GetParent returns the pipeline, if any, which contains this node.
func (c *CallGraphStage) GetParent() *CallGraphPipeline {
	return c.Parent
}

// GetChildren returns nil.  Stages don't have children.
func (c *CallGraphStage) GetChildren() []CallGraphNode {
	return nil
}

// DisableBindings returns the set of expressions which could disable a
// stage or pipeline.
//
// A stage or pipeline is disabled if it is called like
//
//   call STAGE (
//       ...
//   ) using (
//       disabled = FOO.value,
//   )
//
// or if its parent pipeline is disabled.
//
// If an expression disabling this node or any of its parent pipelines evaluates
// to a constant true, then only that constant expression will be returned.
// Otherwise, the set of reference expressions which could disable this node or
// any of its parents is returned.  Constants evaluating to false are omitted.
func (c *CallGraphStage) Disabled() []Exp {
	return c.Disable
}

// Get the set of mapped calls, which may include this node or any of
// its ancestor nodes, for which this node must also be split.
func (c *CallGraphStage) ForkRoots() ForkRootList {
	return c.Forks
}

// The fully-qualified ID suffix for this node. That is
// e.g. ...id(grandparent).id(parent).id(this).
//
// Unlike the FQName() used in core.Node, this one lacks the
// ID.PipestanceId. prefix, but is otherwise the same.
func (c *CallGraphStage) GetFqid() string {
	return c.Fqid
}

// The ast node for this stage.
func (c *CallGraphStage) Callable() Callable {
	return c.stage
}

// The call, either in the context of the parent or the top-level call.
func (c *CallGraphStage) Call() *CallStm {
	return c.call
}

// The resolved input bindings for this node.  References are followed
// until either a literal expression is found or a stage node is
// encountered.
func (c *CallGraphStage) ResolvedInputs() ResolvedBindingMap {
	if c == nil {
		return nil
	}
	return c.Inputs
}

// ResolvedOutputs returns the resolved input bindings for this node.
//
// References are followed until either a literal expression is found or a
// stage node is encountered.  For stage nodes, this will simply be the stage
// FQID and output name.
func (c *CallGraphStage) ResolvedOutputs() *ResolvedBinding {
	return c.Outputs
}

// Retained values, exempt from VDR.
func (c *CallGraphStage) Retained() []*RefExp {
	if r := c.stage.Retain; r != nil && len(r.Params) > 0 {
		result := make([]*RefExp, 0, len(r.Params))
		for _, rp := range r.Params {
			result = append(result, &RefExp{
				Node:     rp.Node,
				Id:       c.Fqid,
				OutputId: rp.Id,
			})
		}
		return result
	}
	return nil
}

func (s *CallGraphStage) MapSource() MapCallSource {
	return s.Source
}

func makeFqid(prefix string, call *CallStm, parent *CallGraphPipeline) string {
	fqid := call.Id
	if parent != nil {
		fqidlen := len(call.Id) + len(prefix)
		fqidParts := []string{call.Id}
		for p := parent; p != nil; p = p.GetParent() {
			fqidlen += 1 + len(p.Call().Id)
			fqidParts = append(fqidParts, p.Call().Id)
		}
		var buf strings.Builder
		buf.Grow(fqidlen)
		if _, err := buf.WriteString(prefix); err != nil {
			panic(err)
		}
		for i := len(fqidParts) - 1; i >= 0; i-- {
			if _, err := buf.WriteString(fqidParts[i]); err != nil {
				panic(err)
			}
			if i != 0 {
				if _, err := buf.WriteRune('.'); err != nil {
					panic(err)
				}
			}
		}
		fqid = buf.String()
	} else if prefix != "" {
		fqid = prefix + fqid
	}
	return fqid
}

func alwaysDisable(disable []Exp) []Exp {
	if len(disable) >= 1 {
		if v, ok := disable[0].(*BoolExp); ok && v.Value {
			return disable[:1]
		}
	}
	return []Exp{&trueExp}
}

func (node *CallGraphStage) resolveInputs(siblings map[string]*ResolvedBinding,
	mapped ForkRootList,
	lookup *TypeLookup) error {
	var errs ErrorList
	var parentInputs map[string]*ResolvedBinding
	var disable []Exp
	if parent := node.Parent; parent != nil {
		parentInputs = parent.Inputs
		disable = parent.Disable
	}
	ins, err := node.call.Bindings.resolve(parentInputs, siblings,
		lookup, true)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid,
			Err: err,
		})
	}
	if node.isEmptyMapping() {
		node.Disable = alwaysDisable(disable)
	} else {
		node.Disable, err = node.resolveDisable(disable, parentInputs,
			siblings, lookup)
		if err != nil {
			errs = append(errs, &wrapError{
				innerError: err,
				loc:        node.Call().Node.Loc,
			})
		}
	}
	node.Source, err = unifyMapSources(node.call, ins, node.Disable)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid,
			Err: err,
		})
	}
	node.Inputs = ins
	node.resolveForks(mapped, node)
	if node.Source != nil {
		match := false
		for _, f := range node.Forks {
			if f.MapSource() == node.Source {
				match = true
				break
			}
		}
		if !match {
			for _, f := range mapped {
				if f.MapSource() == node.Source {
					match = true
					node.Forks = append(node.Forks, f)
					break
				}
			}
		}
		if !match {
			node.Forks = append(node.Forks, node)
		}
	}
	return errs.If()
}

func unifyMapSources(call *CallStm, ins map[string]*ResolvedBinding, disable []Exp) (MapCallSource, error) {
	splits := make(map[*SplitExp]struct{})
	for _, b := range ins {
		findSplitsForCall(b.Exp, call, splits)
	}
	for _, b := range disable {
		findSplitsForCall(b, call, splits)
	}
	// Sort the splits before merging, to make the resolution repeatable.
	splitList := make([]*SplitExp, 0, len(splits))
	for sp := range splits {
		splitList = append(splitList, sp)
	}
	sort.Slice(splitList, func(i, j int) bool {
		if f1 := splitList[i].Node.Loc.File; f1 != nil {
			if f2 := splitList[j].Node.Loc.File; f2 != nil {
				if f1.FileName < f2.FileName {
					return true
				} else if f2.FileName < f1.FileName {
					return false
				}
			} else {
				return true
			}
		} else if f2 := splitList[j].Node.Loc.File; f2 != nil {
			return false
		}
		return splitList[i].Node.Loc.Line < splitList[j].Node.Loc.Line
	})
	var root MapCallSource
	var errs ErrorList
	var ref MapCallSource
	for _, sp := range splitList {
		if ref == nil || !ref.KnownLength() {
			if r, ok := sp.Source.(refMapResolver); ok && (ref == nil || sp.Source.KnownLength()) {
				rr := r.resolveMapSource(sp.CallMode())
				if rr.CallMode() == ModeSingleCall {
					ref = &ReferenceMappingSource{
						Ref:  rr.(*RefExp),
						Mode: sp.Source.CallMode(),
					}
				} else {
					ref = rr
				}
			}
			if r, ok := sp.Value.(*RefExp); ok && (ref == nil || r.KnownLength()) {
				rr := r.resolveMapSource(sp.CallMode())
				if rr.CallMode() == ModeSingleCall {
					ref = &ReferenceMappingSource{
						Ref:  rr.(*RefExp),
						Mode: sp.Source.CallMode(),
					}
				} else {
					ref = rr
				}
			}
		}
		if f, err := mergeSplitSource(root, sp); err != nil {
			errs = append(errs, err)
		} else {
			root = f
		}
	}
	if err := errs.If(); err != nil {
		return root, err
	}
	if root != nil {
		if ref != nil && (ref.KnownLength() || !root.KnownLength()) {
			root = ref
		}
		for _, sp := range splitList {
			sp.Source = root
		}
	}
	return root, nil
}

func mergeSplitSource(root MapCallSource, sp *SplitExp) (MapCallSource, error) {
	switch src := sp.Value.(type) {
	case MapCallSource:
		if f, err := MergeMapCallSources(root, src); err != nil {
			return root, err
		} else {
			root = f
		}
	}
	if f, err := MergeMapCallSources(root, sp.Source); err != nil {
		return root, err
	} else {
		root = f
	}
	return root, nil
}

func updateMapSources(call *CallStm, root MapCallSource, exp Exp) Exp {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			ee := *exp
			ee.Source = root
			ee.Value = updateMapSources(call, root, exp.Value)
			return &ee
		}
		if e := updateMapSources(call, root, exp.Value); e != exp.Value {
			ee := *exp
			ee.Value = e
			return &ee
		}
	case *ArrayExp:
		if hasSplit(exp, call) != nil {
			arr := *exp
			arr.Value = make([]Exp, len(exp.Value))
			for i, v := range exp.Value {
				arr.Value[i] = updateMapSources(call, root, v)
			}
			return &arr
		}
	case *MapExp:
		if hasSplit(exp, call) != nil {
			arr := *exp
			arr.Value = make(map[string]Exp, len(exp.Value))
			for i, v := range exp.Value {
				arr.Value[i] = updateMapSources(call, root, v)
			}
			return &arr
		}
	case *DisabledExp:
		disabled := updateMapSources(call, root, exp.Disabled)
		inner := updateMapSources(call, root, exp.Value)
		e, err := exp.makeDisabledExp(disabled, inner)
		if err != nil {
			panic(err)
		}
		return e
	}
	return exp
}

func (node *CallGraphStage) resolveForks(mapped ForkRootList, localRoot *CallGraphStage) {
	if len(mapped) == 0 {
		return
	}
	splits := make(map[*CallStm]struct{}, len(mapped))
	for _, input := range node.Inputs {
		findSplitCalls(input.Exp, splits, false)
	}
	for _, d := range node.Disable {
		findSplitCalls(d, splits, false)
	}
	if len(splits) > 0 {
		if localRoot == nil {
			node.Forks = make(ForkRootList, 0, len(splits))
		} else {
			node.Forks = make(ForkRootList, 0, len(splits)+1)
		}
		for _, src := range mapped {
			if src == localRoot {
				continue
			}
			if _, ok := splits[src.Call()]; ok {
				node.Forks = append(node.Forks, src)
			}
		}
	}
}

var trueExp = BoolExp{Value: true}

func (node *CallGraphStage) resolveDisable(disable []Exp,
	parentInputs, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) ([]Exp, error) {
	mod := node.call.Modifiers
	if mod == nil {
		return disable, nil
	}
	for len(disable) >= 1 &&
		disable[0].getKind() == KindBool {
		if disable[0].(*BoolExp).Value {
			return disable[:1], nil
		} else {
			disable = disable[1:]
		}
	}
	if node.call.KnownLength() {
		// Map over an empty set is equivalent to disabled.
		switch node.call.CallMode() {
		case ModeArrayCall:
			if node.call.ArrayLength() == 0 {
				return []Exp{&trueExp}, nil
			}
		case ModeMapCall:
			if len(node.call.Keys()) == 0 {
				return []Exp{&trueExp}, nil
			}
		case ModeNullMapCall:
			return []Exp{&trueExp}, nil
		}
	}
	bind := mod.Bindings
	if bind == nil {
		return disable, nil
	}
	d := bind.Table[disabled]
	if d == nil {
		return disable, nil
	}
	r, err := resolveExp(d.Exp, d.Tname, parentInputs, siblings, lookup, true)
	if err != nil {
		return disable, &bindingError{
			Msg: "BindingError: disabled control binding",
			Err: err,
		}
	}
	return resolveDisableExp(r.Exp, disable)
}

func resolveDisableExp(r Exp, disable []Exp) ([]Exp, error) {
	switch r := r.(type) {
	case *RefExp:
		for _, e := range disable {
			if e == r {
				return disable, nil
			}
		}
		v := make([]Exp, len(disable), len(disable)+1)
		copy(v, disable)
		return append(v, r), nil
	case *NullExp:
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled cannot be bound to a null value.",
			},
			loc: r.getNode().Loc,
		}
	case *BoolExp:
		if r.Value {
			return []Exp{r}, nil
		}
		return disable, nil
	case *SplitExp:
		switch v := r.Value.(type) {
		case *ArrayExp:
			return resolveDisableArray(r, v.Value, disable)
		case *MapExp:
			if v.Kind != KindMap {
				return disable, &wrapError{
					innerError: &bindingError{
						Msg: "BindingError: cannot split " + string(v.Kind),
					},
					loc: r.getNode().Loc,
				}
			}
			return resolveDisableMap(r, v.Value, disable)
		}
		return resolveDisableExp(r.Value, disable)
	case *DisabledExp:
		for _, e := range disable {
			if r.Disabled.equal(e) {
				return resolveDisableExp(r.Value, disable)
			}
		}
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled was bound to a value that may be disabled at runtime",
			},
			loc: r.getNode().Loc,
		}
	default:
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled control binding was not boolean",
			},
			loc: r.getNode().Loc,
		}
	}
}

func resolveDisableArray(r Exp, v, disable []Exp) ([]Exp, error) {
	if len(v) == 0 {
		return disable, nil
	} else if len(v) == 1 {
		return resolveDisableExp(v[0], disable)
	}
	allFalse := true
	allTrue := true
	for _, e := range v {
		switch e := e.(type) {
		case *RefExp, *NullExp:
			allTrue = false
			allFalse = false
		case *BoolExp:
			if e.Value {
				allFalse = false
			} else {
				allTrue = false
			}
		default:
			return disable, &wrapError{
				innerError: &bindingError{
					Msg: "BindingError: disabled control binding was not boolean",
				},
				loc: e.getNode().Loc,
			}
		}
	}
	if allFalse {
		return disable, nil
	}
	if allTrue {
		return []Exp{v[0]}, nil
	}
	result := make([]Exp, len(disable), len(disable)+1)
	copy(result, disable)
	return append(result, r), nil
}

func resolveDisableMap(r Exp, v map[string]Exp, disable []Exp) ([]Exp, error) {
	if len(v) == 0 {
		return disable, nil
	} else if len(v) == 1 {
		for _, e := range v {
			return resolveDisableExp(e, disable)
		}
	}
	allFalse := true
	allTrue := true
	for _, e := range v {
		switch e := e.(type) {
		case *RefExp, *NullExp:
			allTrue = false
			allFalse = false
		case *BoolExp:
			if e.Value {
				allFalse = false
			} else {
				allTrue = false
			}
		default:
			return disable, &wrapError{
				innerError: &bindingError{
					Msg: "BindingError: disabled control binding was not boolean",
				},
				loc: e.getNode().Loc,
			}
		}
	}
	if allFalse {
		return disable, nil
	}
	if allTrue {
		for _, e := range v {
			return []Exp{e}, nil
		}
	}
	result := make([]Exp, len(disable), len(disable)+1)
	copy(result, disable)
	return append(result, r), nil
}

func (node *CallGraphStage) resolve(siblings map[string]*ResolvedBinding,
	mapped ForkRootList, lookup *TypeLookup) error {
	err := node.resolveInputs(siblings, mapped, lookup)

	if len(node.stage.OutParams.List) > 0 {
		if node.isAlwaysDisabled() {
			// constantly-disabled stages always output null.  No need to
			// propagate references.
			node.Outputs = &ResolvedBinding{
				Exp: &NullExp{
					valExp: valExp{Node: node.stage.Node},
				},
				Type: lookup.Get(TypeId{Tname: node.stage.Id}),
			}
			node.Disable = alwaysDisable(node.Disable)
		} else {
			tid := TypeId{Tname: node.stage.Id}
			ref := RefExp{
				Node: node.stage.Node,
				Kind: KindCall,
				Id:   node.Fqid,
			}
			var exp Exp = &ref
			for i := len(node.Forks) - 1; i >= 0; i-- {
				fs := node.Forks[i]
				ref.MergeOver = append(ref.MergeOver, fs.MapSource())
				switch fs.MapSource().CallMode() {
				case ModeArrayCall:
					tid.ArrayDim++
				case ModeMapCall:
					tid.MapDim = tid.ArrayDim + 1
					tid.ArrayDim = 0
				case ModeNullMapCall:
					exp = &NullExp{
						valExp: valExp{Node: node.stage.Node},
					}
				}
				if _, ok := exp.(*NullExp); !ok && fs.MapSource() != node.MapSource() {
					// Forking because the parent pipeline forked, so other
					// calls in the same pipeline should also fork.
					// These splits will be removed by unsplit later.
					exp = &SplitExp{
						valExp: valExp{Node: fs.Call().Node},
						Call:   fs.Call(),
						Source: fs.MapSource(),
						Value:  exp,
					}
				}
			}
			node.Outputs = &ResolvedBinding{
				Exp:  exp,
				Type: lookup.Get(tid),
			}
		}
	}
	return err
}

func (node *CallGraphStage) unsplit() error {
	if node.Outputs == nil {
		return nil
	}
	for s, ok := node.Outputs.Exp.(*SplitExp); ok; s, ok = node.Outputs.Exp.(*SplitExp) {
		node.Outputs.Exp = s.Value
	}
	return nil
}

func findSplitCalls(exp Exp, result map[*CallStm]struct{}, onlyUnknown bool) {
	switch exp := exp.(type) {
	case *SplitExp:
		if !onlyUnknown || !exp.Source.KnownLength() {
			result[exp.Call] = struct{}{}
		}
		findSplitCalls(exp.Value, result, onlyUnknown)
	case *ArrayExp:
		for _, v := range exp.Value {
			findSplitCalls(v, result, onlyUnknown)
		}
	case *MapExp:
		for _, v := range exp.Value {
			findSplitCalls(v, result, onlyUnknown)
		}
	case *DisabledExp:
		findSplitCalls(exp.Value, result, onlyUnknown)
	}
}

func findSplitsForCall(exp Exp, call *CallStm, result map[*SplitExp]struct{}) {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			result[exp] = struct{}{}
		}
		findSplitsForCall(exp.Value, call, result)
	case *ArrayExp:
		for _, v := range exp.Value {
			findSplitsForCall(v, call, result)
		}
	case *MapExp:
		for _, v := range exp.Value {
			findSplitsForCall(v, call, result)
		}
	case *DisabledExp:
		findSplitsForCall(exp.Value, call, result)
	}
}

// Returns true if the node never runs.  Only returns a correct result if
// inputs have been resolved already.
//
// It is not legal to specify a constantly-disabled call in MRO directly,
// however it can happen if the call to a pipeline includes a literal value
// which results in one of the calls by that pipeline being disabled.
func (node *CallGraphStage) isAlwaysDisabled() bool {
	if len(node.Disable) >= 1 && node.Disable[0].getKind() == KindBool {
		// false is trimmed out of the disable list, so it must be true.
		return true
	}
	return node.isEmptyMapping()
}

func (node *CallGraphStage) isEmptyMapping() bool {
	if m := node.call.CallMode(); m == ModeSingleCall {
		return false
	} else if m == ModeNullMapCall {
		return true
	}
	for _, input := range node.Inputs {
		if exp, ok := input.Exp.(*SplitExp); ok {
			if exp.IsEmpty() {
				return true
			}
		}
	}
	return false
}

func hasSplit(exp Exp, call MapCallSource) Exp {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			return exp.InnerValue()
		}
		return hasSplit(exp.Value, call)
	case *ArrayExp:
		var result Exp
		for _, v := range exp.Value {
			if v := hasSplit(v, call); v != nil {
				switch v := v.(type) {
				case *RefExp:
					result = v
				case *NullExp:
					if result == nil {
						result = v
					}
				default:
					return v
				}
			}
		}
		return result
	case *MapExp:
		var result Exp
		for _, v := range exp.Value {
			if v := hasSplit(v, call); v != nil {
				switch v := v.(type) {
				case *RefExp:
					result = v
				case *NullExp:
					if result == nil {
						result = v
					}
				default:
					return v
				}
			}
		}
		return result
	case *DisabledExp:
		return hasSplit(exp.Value, call)
	}
	return nil
}

func (node *CallGraphStage) NodeClosure() map[string]CallGraphNode {
	return map[string]CallGraphNode{
		node.Fqid: node,
	}
}

func (node *CallGraphStage) nodeClosure(m map[string]CallGraphNode) {
	m[node.Fqid] = node
}

// CallMode Returns the call mode for a call which depends on this source.
func (node *CallGraphStage) CallMode() CallMode {
	return node.call.CallMode()
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (node *CallGraphStage) KnownLength() bool {
	if node.call.KnownLength() {
		return true
	}
	if node.Source != nil {
		return node.Source.KnownLength()
	}
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (node *CallGraphStage) ArrayLength() int {
	if node.Source != nil {
		return node.Source.ArrayLength()
	}
	return node.call.ArrayLength()
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (node *CallGraphStage) Keys() map[string]Exp {
	if node.Source != nil {
		return node.Source.Keys()
	}
	return node.call.Keys()
}

func (node *CallGraphStage) GoString() string {
	return node.Fqid
}
