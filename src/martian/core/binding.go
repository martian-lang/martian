//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime. This is where the action happens.
//
package core

import (
	"martian/syntax"
)

//=============================================================================
// Binding
//=============================================================================
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

func newBinding(node *Node, bindStm *syntax.BindStm, returnBinding bool) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.Id
	self.tname = bindStm.Tname
	self.sweep = bindStm.Sweep
	self.sweepRootId = bindStm.Id
	self.waiting = false
	switch valueExp := bindStm.Exp.(type) {
	case *syntax.RefExp:
		if valueExp.Kind == "self" {
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
			self.id = bindStm.Id
			self.valexp = "self." + valueExp.Id
		} else if valueExp.Kind == "call" {
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
		self.mode = "value"
		self.parentNode = node
		self.boundNode = node
		self.value = bindStm.Exp.ToInterface()
	}
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
