// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Code for resolving the pipeline graph.

package syntax

import (
	"sort"
	"strings"
)

// CallGraphStage represents a stage in a call graph.
type CallGraphStage struct {
	Parent   *CallGraphPipeline `json:"-"`
	Fqid     string             `json:"fqid"`
	Comments []string           `json:"comments,omitempty"`
	call     *CallStm
	stage    *Stage
	Inputs   ResolvedBindingMap `json:"inputs"`
	Outputs  *ResolvedBinding   `json:"outputs"`
	Disable  []Exp              `json:"disabled,omitempty"`
	Forks    ForkRootList       `json:"fork_roots,omitempty"`
	split    *SplitExp
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
	if s.split == nil {
		return nil
	}
	return s.split.Source
}

func (s *CallGraphStage) Split() *SplitExp {
	return s.split
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
		lookup)
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
	node.split, err = unifyMapSources(node.call, ins, node.Disable)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid,
			Err: err,
		})
	}
	if node.split != nil {
		found := false
		for i := len(mapped) - 1; i >= 0; i-- {
			if mapped[i].call == node.split.Call {
				found = true
				break
			}
		}
		if !found {
			mapped = append(mapped, node)
		}
	}
	node.Inputs = ins
	node.resolveForks(mapped, node)
	return errs.If()
}

type sortedSplitList []*SplitExp

func (arr sortedSplitList) Len() int {
	return len(arr)
}

func (arr sortedSplitList) Swap(i, j int) {
	arr[i], arr[j] = arr[j], arr[i]
}

func (arr sortedSplitList) Less(i, j int) bool {
	if f1 := arr[i].File(); f1 != nil {
		if f2 := arr[j].File(); f2 != nil {
			if f1.FileName < f2.FileName {
				return true
			} else if f2.FileName < f1.FileName {
				return false
			}
		} else {
			return true
		}
	} else if f2 := arr[j].File(); f2 != nil {
		return false
	}
	return arr[i].Line() < arr[j].Line()
}

func unifyMapSources(call *CallStm, ins map[string]*ResolvedBinding, disable []Exp) (*SplitExp, error) {
	splits := make(map[*SplitExp]struct{})
	for _, b := range ins {
		findSplitsForCall(b.Exp, call, splits)
	}
	for _, b := range disable {
		findSplitsForCall(b, call, splits)
	}
	// Sort the splits before merging, to make the resolution repeatable.
	splitList := make(sortedSplitList, 0, len(splits))
	for sp := range splits {
		splitList = append(splitList, sp)
	}
	sort.Sort(splitList)
	var root MapCallSource
	var errs ErrorList
	var ref MapCallSource
	for _, sp := range splitList {
		if f, err := mergeSplitSource(root, sp); err != nil {
			errs = append(errs, err)
		} else {
			root = f
		}
	}
	if err := errs.If(); err != nil {
		return nil, err
	}
	if root != nil {
		if ref != nil && (ref.KnownLength() || !root.KnownLength()) {
			root = ref
		}
		for _, sp := range splitList {
			sp.setSource(root)
		}
	}
	if len(splitList) > 0 {
		return splitList[0], nil
	} else {
		return nil, nil
	}
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
		sp.Source = f
	}
	return root, nil
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
		if localRoot != nil {
			if _, ok := splits[localRoot.call]; ok {
				node.Forks = append(node.Forks, localRoot)
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
	r, err := resolveExp(d.Exp, d.Tname, parentInputs, siblings, lookup)
	if err != nil {
		return disable, &bindingError{
			Msg: "BindingError: disabled expression for " + node.Fqid,
			Err: err,
		}
	}
	disable, err = resolveDisableExp(r.Exp, disable)
	if err != nil {
		err = &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: control binding for " + node.Fqid,
				Err: err,
			},
			loc: node.call.Node.Loc,
		}
	}
	return disable, err
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
		case *MergeExp:
			return resolveDisableExp(v.Value, disable)
		}
		return resolveDisableExp(r.Value, disable)
	case *DisabledExp:
		for _, e := range disable {
			if r.Disabled.equal(e) == nil {
				return resolveDisableExp(r.Value, disable)
			}
		}
		var buf strings.Builder
		buf.WriteString("BindingError: disabled was bound to value ")
		buf.WriteString(r.Value.GoString())
		if buf.Len() < 50 {
			buf.WriteByte(' ')
		} else {
			buf.WriteString("\n    ")
		}
		buf.WriteString("which would be disabled at runtime if ")
		buf.WriteString(r.Disabled.GoString())
		buf.WriteString(" is true")
		if len(disable) > 0 {
			buf.WriteString(".\n      Call is also disabled by")
			for _, e := range disable {
				if len(disable) > 1 {
					buf.WriteString("\n        ")
				} else {
					buf.WriteRune(' ')
				}
				buf.WriteString(e.GoString())
			}
		}
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: buf.String(),
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
	var errs ErrorList
	if err := node.resolveInputs(siblings, mapped, lookup); err != nil {
		errs = append(errs, err)
	}

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
			if node.split != nil {
				switch node.split.Source.CallMode() {
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
				if exp == &ref {
					src := node.split.Source
					if s, err := MergeMapCallSources(&BoundReference{
						Exp:  &ref,
						Type: lookup.Get(tid),
					}, src); err != nil {
						errs = append(errs, err)
					} else {
						src = s
					}
					ref.Forks = make(map[*CallStm]CollectionIndex, len(node.Forks))
					ref.Forks[node.call] = unknownIndex{src: src}
					exp = &MergeExp{
						Call:      node,
						MergeOver: src,
						Value:     exp,
					}
				}
			}
			for i := len(node.Forks) - 1; i >= 0; i-- {
				fs := node.Forks[i]
				if fs != node {
					if ref.Forks == nil {
						ref.Forks = make(map[*CallStm]CollectionIndex, len(node.Forks)-i)
					}
					ref.Forks[fs.call] = unknownIndex{src: fs.MapSource()}
				}
			}
			exp, err := node.makeDisabled(exp, lookup)
			if err != nil {
				errs = append(errs, err)
			}
			node.Outputs = &ResolvedBinding{
				Exp:  exp,
				Type: lookup.Get(tid),
			}
		}
	}
	return errs.If()
}

func (node *CallGraphStage) makeDisabled(exp Exp,
	lookup *TypeLookup) (Exp, error) {
	var errs ErrorList
	if len(node.Disable) > 0 {
		start := 0
		if node.Parent != nil {
			start = len(node.Parent.Disable)
		}
		for _, d := range node.Disable[start:] {
			e, err := wrapDisabled(d, exp, lookup)
			if err != nil {
				errs = append(errs, err)
			}
			exp = e
		}
	}
	return exp, errs.If()
}

func wrapDisabled(d, exp Exp, lookup *TypeLookup) (Exp, error) {
	if exp.getKind() == KindNull {
		return exp, nil
	}
	var errs ErrorList
	switch d := d.(type) {
	case *DisabledExp:
		return d.makeDisabledExp(d.Value, exp)
	case *RefExp:
		return &DisabledExp{
			Disabled: d,
			Value:    exp,
		}, nil
	case *BoolExp:
		if d.Value {
			return &NullExp{valExp: d.valExp}, nil
		}
	case *SplitExp:
		switch v := d.Value.(type) {
		case *RefExp:
			exp = &DisabledExp{
				Disabled: v,
				Value:    exp,
			}
		case *ArrayExp:
			arr := *v
			arr.Value = make([]Exp, len(arr.Value))
			fork := make(map[*CallStm]CollectionIndex, 1)
			for i := range v.Value {
				v, err := wrapDisabled(v.Value[i], exp, lookup)
				if err != nil {
					errs = append(errs, err)
				} else {
					fork[d.Call] = arrayIndex(i)
					v, err = v.BindingPath("", fork, lookup)
					if err != nil {
						errs = append(errs, err)
					}
				}
				arr.Value[i] = v
			}
			se := *d
			se.Value = &arr
			exp = &se
		case *MapExp:
			m := *v
			m.Value = make(map[string]Exp, len(m.Value))
			fork := make(map[*CallStm]CollectionIndex, 1)
			for k, vv := range v.Value {
				v, err := wrapDisabled(vv, exp, lookup)
				if err != nil {
					errs = append(errs, err)
				} else {
					fork[d.Call] = mapKeyIndex(k)
					v, err = v.BindingPath("", fork, lookup)
					if err != nil {
						errs = append(errs, err)
					}
				}
				m.Value[k] = v
			}
			se := *d
			se.Value = &m
			exp = &se
		default:
			return exp, &bindingError{
				Msg: "invalid disable binding " + string(d.getKind()),
			}
		}
	}
	return exp, errs.If()
}

func (node *CallGraphStage) unsplit(lookup *TypeLookup) error {
	if node.isAlwaysDisabled() {
		for _, binding := range node.Inputs {
			if sp, ok := binding.Exp.(*SplitExp); ok {
				binding.Exp = &NullExp{
					valExp: sp.valExp,
				}
			}
		}
	}
	if node.Outputs == nil {
		return nil
	}
	var errs ErrorList
	e, err := node.Outputs.Exp.BindingPath("", nil, lookup)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid + " outputs",
			Err: err,
		})
	}
	node.Outputs.Exp = e
	for k, binding := range node.Inputs {
		// Ensure inputs can be scanned for refs, and also that their
		// types are cached.  Otherwise, at runtime mrp may end up trying to cache
		// the types concurrently.
		if _, err := binding.FindRefs(lookup); err != nil {
			errs = append(errs, &bindingError{
				Msg: node.Fqid + " input " + k,
				Err: err,
			})
		}
	}
	return errs.If()
}

func findSplitCalls(exp Exp, result map[*CallStm]struct{}, onlyUnknown bool) {
	switch exp := exp.(type) {
	case *SplitExp:
		if !onlyUnknown || !exp.mapSource().KnownLength() {
			result[exp.GetCall()] = struct{}{}
		}
		findSplitCalls(exp.innerValue(), result, onlyUnknown)
	case *MergeExp:
		_, ok := result[exp.GetCall()]
		findSplitCalls(exp.Value, result, onlyUnknown)
		// if exp.ForkNode != nil {
		// 	findSplitCalls(exp.ForkNode, result, onlyUnknown)
		// }
		if !ok {
			// splits for this call which are below this level are merged.
			delete(result, exp.GetCall())
		}
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
		findSplitCalls(exp.Disabled, result, onlyUnknown)
	case *RefExp:
		for c, i := range exp.Forks {
			if i.IndexSource() != nil {
				result[c] = struct{}{}
			}
		}
	}
}

func findSplitsForCall(exp Exp, call *CallStm, result map[*SplitExp]struct{}) {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			result[exp] = struct{}{}
		}
		findSplitsForCall(exp.Value, call, result)
	case *MergeExp:
		findSplitsForCall(exp.Value, call, result)
		// if exp.ForkNode != nil {
		// 	findSplitsForCall(exp.ForkNode, call, result)
		// }
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
		findSplitsForCall(exp.Disabled, call, result)
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
	if node.split != nil {
		return node.split.Source.KnownLength()
	}
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (node *CallGraphStage) ArrayLength() int {
	if node.split != nil {
		return node.split.Source.ArrayLength()
	}
	return node.call.ArrayLength()
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (node *CallGraphStage) Keys() map[string]Exp {
	if node.split != nil {
		return node.split.Source.Keys()
	}
	return node.call.Keys()
}

func (node *CallGraphStage) GoString() string {
	return node.Fqid
}
