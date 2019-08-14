// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Methods to resolve argument and output bindings.

import (
	"encoding/json"

	"github.com/martian-lang/martian/martian/syntax"
)

//=============================================================================
// Binding
//=============================================================================

// Holds information about the value of an input arguemnt, either hard-coded
// into the MRO or bound to the output of another node.
type Binding struct {
	node        *Node
	id          string
	tname       string
	sweepRootId string
	sweep       bool
	waiting     bool
	valexp      string
	mode        string
	parentNode  Nodable
	boundNode   Nodable
	output      string
	value       interface{}
}

// An exportable version of Binding.
type BindingInfo struct {
	Id          string      `json:"id"`
	Type        string      `json:"type"`
	ValExp      string      `json:"valexp"`
	Mode        string      `json:"mode"`
	Output      string      `json:"output"`
	Sweep       bool        `json:"sweep"`
	SweepRootId string      `json:"sweepRootId"`
	Node        interface{} `json:"node"`
	MatchedFork interface{} `json:"matchedFork"`
	Value       interface{} `json:"value"`
	Waiting     bool        `json:"waiting"`
}

func (self *Binding) preBind(exp syntax.Exp, sweep, returnBinding bool) {
	switch valueExp := exp.(type) {
	case *syntax.RefExp:
		if valueExp.Kind == syntax.KindSelf {
			var parentBinding *Binding
			if returnBinding {
				parentBinding = self.node.argbindings[valueExp.Id]
			} else {
				parentBinding = self.node.parent.getNode().argbindings[valueExp.Id]
			}
			if parentBinding != nil {
				self.node = parentBinding.node
				self.tname = parentBinding.tname
				self.sweep = parentBinding.sweep
				self.sweepRootId = parentBinding.sweepRootId
				self.waiting = parentBinding.waiting
				self.mode = parentBinding.mode
				self.parentNode = parentBinding.parentNode
				self.boundNode = parentBinding.boundNode
				self.output = parentBinding.output
				self.value = parentBinding.value
			}
			self.valexp = "self." + valueExp.Id
		} else if valueExp.Kind == syntax.KindCall {
			if returnBinding {
				self.parentNode = self.node.subnodes[valueExp.Id]
				self.boundNode, self.output, self.mode, self.value = self.node.findBoundNode(
					valueExp.Id, valueExp.OutputId, "reference", nil)
			} else {
				self.parentNode = self.node.parent.getNode().subnodes[valueExp.Id]
				self.boundNode, self.output, self.mode, self.value = self.node.parent.getNode().findBoundNode(
					valueExp.Id, valueExp.OutputId, "reference", nil)
			}
			if valueExp.OutputId == "default" {
				self.valexp = valueExp.Id
			} else {
				self.valexp = valueExp.Id + "." + valueExp.OutputId
			}
		}
	case *syntax.ValExp:
		if !sweep && valueExp.Kind == syntax.KindArray {
			subexps := valueExp.Value.([]syntax.Exp)
			valueBindings := make([]*Binding, 0, len(subexps))
			useValue := true
			for _, innerExp := range subexps {
				b := &Binding{
					node:        self.node,
					id:          self.id,
					sweepRootId: self.sweepRootId,
				}
				b.preBind(innerExp, false, returnBinding)
				// If all of the expressions are value bindings then we can
				// just use a direct value binding and ignore the array.
				// Otherwise we'll need to recurse when resolving.
				if b.mode != "value" {
					useValue = false
				}
				valueBindings = append(valueBindings, b)
			}
			if !useValue {
				self.mode = "array"
				self.parentNode = self.node
				self.boundNode = self.node
				self.value = valueBindings
				return
			}
		}
		self.mode = "value"
		self.parentNode = self.node
		self.boundNode = self.node
		self.value = valueExp.ToInterface()
	}
}

func newBinding(node *Node, bindStm *syntax.BindStm, returnBinding bool) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.Id
	self.tname = bindStm.Tname
	self.sweep = bindStm.Sweep
	self.sweepRootId = bindStm.Id
	self.waiting = false
	self.preBind(bindStm.Exp, bindStm.Sweep, returnBinding)
	return self
}

func NewBinding(node *Node, bindStm *syntax.BindStm) *Binding {
	return newBinding(node, bindStm, false)
}

func NewReturnBinding(node *Node, bindStm *syntax.BindStm) *Binding {
	return newBinding(node, bindStm, true)
}

func (self *Binding) resolve(argPermute map[string]interface{}, readSize int64) (interface{}, error) {
	self.waiting = false
	if self.mode == "value" {
		if argPermute == nil {
			// In this case we want to get the raw value, which might be a sweep array.
			return self.value, nil
		}
		// Replace literal sweep ranges with specific permuted argument values.
		if self.sweep {
			// This needs to use self.sweepRootId because argPermute
			// is populated with sweepRootId's (not just id's) in buildForks.
			// This is required for proper forking when param names don't match.
			return argPermute[self.sweepRootId], nil
		} else {
			return self.value, nil
		}
	} else if self.mode == "array" {
		innerBinds := self.value.([]*Binding)
		result := make([]interface{}, 0, len(innerBinds))
		for _, binding := range innerBinds {
			if r, err := binding.resolve(argPermute, readSize); err != nil {
				return nil, err
			} else if binding.waiting {
				self.waiting = true
				return nil, nil
			} else {
				result = append(result, r)
			}
		}
		return result, nil
	}
	if argPermute == nil {
		return nil, nil
	}
	if self.boundNode != nil {
		matchedFork := self.boundNode.getNode().matchFork(argPermute)
		if outputs, err := matchedFork.metadata.read(OutsFile, readSize); err != nil {
			return nil, err
		} else if outputs != nil {
			output, ok := outputs[self.output]
			if ok {
				return output, nil
			}
		}
	}
	self.waiting = true
	return nil, nil
}

func (self *Binding) serializeState(argPermute map[string]interface{}, readSize int64) (*BindingInfo, error) {
	var node interface{} = nil
	var matchedFork interface{} = nil
	if self.boundNode != nil {
		node = self.boundNode.getNode().name
		f := self.boundNode.getNode().matchFork(argPermute)
		if f != nil {
			matchedFork = f.index
		}
	}
	v, err := self.resolve(argPermute, readSize)
	return &BindingInfo{
		Id:          self.id,
		Type:        self.tname,
		ValExp:      self.valexp,
		Mode:        self.mode,
		Output:      self.output,
		Sweep:       self.sweep,
		SweepRootId: self.sweepRootId,
		Node:        node,
		MatchedFork: matchedFork,
		Value:       v,
		Waiting:     self.waiting,
	}, err
}

func resolveBindings(bindings map[string]*Binding, argPermute map[string]interface{}, readSize int64) (LazyArgumentMap, error) {
	resolvedBindings := make(LazyArgumentMap, len(bindings))
	var errs syntax.ErrorList
	for id, binding := range bindings {
		if v, err := binding.resolve(argPermute, readSize); err != nil {
			errs = append(errs, err)
		} else if b, err := json.Marshal(v); err != nil {
			errs = append(errs, err)
		} else {
			resolvedBindings[id] = b
		}
	}
	return resolvedBindings, errs.If()
}

func (pipestance *Pipestance) retain(ref *syntax.RefExp) {
	if ref.Kind == syntax.KindSelf {
		parentBinding := pipestance.node.argbindings[ref.Id]
		if parentBinding != nil {
			if nodable := parentBinding.boundNode; nodable != nil {
				if node := nodable.getNode(); node != nil {
					for _, fork := range node.forks {
						if fileArgs := fork.fileArgs; fileArgs != nil {
							if children := fileArgs[ref.OutputId]; children != nil {
								children[nil] = struct{}{}
							} else {
								fileArgs[ref.OutputId] = map[Nodable]struct{}{nil: {}}
							}
						} else {
							fork.fileArgs = map[string]map[Nodable]struct{}{
								ref.OutputId: {nil: {}},
							}
						}
					}
				}
			}
		}
	} else if ref.Kind == syntax.KindCall {
		if boundNode, outputId, _, _ := pipestance.node.findBoundNode(
			ref.Id, ref.OutputId, "reference", nil); boundNode != nil {
			if node := boundNode.getNode(); node != nil {
				for _, fork := range node.forks {
					if fileArgs := fork.fileArgs; fileArgs != nil {
						if children := fileArgs[outputId]; children != nil {
							children[nil] = struct{}{}
						} else {
							fileArgs[outputId] = map[Nodable]struct{}{nil: {}}
						}
					} else {
						fork.fileArgs = map[string]map[Nodable]struct{}{
							outputId: {nil: {}},
						}
					}
				}
			}
		}
	}
}
