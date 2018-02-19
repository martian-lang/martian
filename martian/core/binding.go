// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Methods to resolve argument and output bindings.

import (
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
	sweep       bool
	sweepRootId string
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

func (self *Binding) resolve(argPermute map[string]interface{}) interface{} {
	self.waiting = false
	if self.mode == "value" {
		if argPermute == nil {
			// In this case we want to get the raw value, which might be a sweep array.
			return self.value
		}
		// Replace literal sweep ranges with specific permuted argument values.
		if self.sweep {
			// This needs to use self.sweepRootId because argPermute
			// is populated with sweepRootId's (not just id's) in buildForks.
			// This is required for proper forking when param names don't match.
			return argPermute[self.sweepRootId]
		} else {
			return self.value
		}
	} else if self.mode == "array" {
		innerBinds := self.value.([]*Binding)
		result := make([]interface{}, 0, len(innerBinds))
		for _, binding := range innerBinds {
			if r := binding.resolve(argPermute); binding.waiting {
				self.waiting = true
				return nil
			} else {
				result = append(result, r)
			}
		}
		return result
	}
	if argPermute == nil {
		return nil
	}
	if self.boundNode != nil {
		matchedFork := self.boundNode.getNode().matchFork(argPermute)
		outputs, ok := matchedFork.metadata.read(OutsFile).(map[string]interface{})
		if ok {
			output, ok := outputs[self.output]
			if ok {
				return output
			}
		}
	}
	self.waiting = true
	return nil
}

func (self *Binding) serializeState(argPermute map[string]interface{}) *BindingInfo {
	var node interface{} = nil
	var matchedFork interface{} = nil
	if self.boundNode != nil {
		node = self.boundNode.getNode().name
		f := self.boundNode.getNode().matchFork(argPermute)
		if f != nil {
			matchedFork = f.index
		}
	}
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
		Value:       self.resolve(argPermute),
		Waiting:     self.waiting,
	}
}

func resolveBindings(bindings map[string]*Binding, argPermute map[string]interface{}) map[string]interface{} {
	resolvedBindings := map[string]interface{}{}
	for id, binding := range bindings {
		resolvedBindings[id] = binding.resolve(argPermute)
	}
	return resolvedBindings
}
