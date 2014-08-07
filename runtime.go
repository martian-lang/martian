package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type Nodable interface {
	Node() *Node
}

type Binding struct {
	node      *Node
	id        string
	tname     string
	sweep     bool
	waiting   bool
	valexp    string
	mode      string
	boundNode Nodable
	output    string
	value     *ValExp
}

func NewBinding(node *Node, bindStm *BindStm) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.id
	self.tname = bindStm.tname
	self.sweep = bindStm.sweep
	self.waiting = false
	switch valueExp := bindStm.exp.(type) {
	case *RefExp:
		if valueExp.kind == "self" {
			parentBinding := node.parent.Node().argbindings[valueExp.id]
			self.node = parentBinding.node
			self.tname = parentBinding.tname
			self.sweep = parentBinding.sweep
			self.waiting = parentBinding.waiting
			self.mode = parentBinding.mode
			self.id = bindStm.id
			self.valexp = "self." + valueExp.id
		} else if valueExp.kind == "call" {
			self.mode = "reference"
			self.boundNode = self.node.parent.Node().subnodes[valueExp.id]
			self.output = valueExp.outputId
			if valueExp.outputId == "default" {
				self.valexp = valueExp.id
			} else {
				self.valexp = valueExp.id + "." + valueExp.outputId
			}
		}
	case *ValExp:
		self.mode = "value"
		self.boundNode = node
		self.value = valueExp
		// Unwrap array values.

	}
	return self
}

type Metadata struct {
}

func NewMetadata() *Metadata {
	return &Metadata{}
}

type Node struct {
	parent      Nodable
	rt          *Runtime
	kind        string
	name        string
	fqname      string
	path        string
	metadata    *Metadata
	outparams   *Params
	argbindings map[string]*Binding
	subnodes    map[string]Nodable
	prenodes    map[string]Nodable
}

func NewNode(parent Nodable, kind string, callStm *CallStm, callables *Callables) *Node {
	self := &Node{}
	self.parent = parent
	self.rt = parent.Node().rt
	self.kind = kind
	self.name = callStm.id
	self.fqname = parent.Node().fqname + "." + self.name
	self.path = path.Join(parent.Node().path, self.name)
	self.metadata = NewMetadata()
	self.outparams = callables.table[self.name].OutParams()
	self.argbindings = map[string]*Binding{}
	for id, bindStm := range callStm.bindings.table {
		self.argbindings[id] = NewBinding(self, bindStm)
	}
	return self
}

func (self *Node) Node() *Node {
	return self
}

type TopNode struct {
	node          *Node
	rt            *Runtime
	fqname        string
	subnodes      map[string]Nodable
	sweepbindings []*Binding
}

func NewTopNode(rt *Runtime, psid string, path string) *TopNode {
	return &TopNode{
		node:          &Node{},
		rt:            rt,
		fqname:        "ID" + psid,
		subnodes:      map[string]Nodable{},
		sweepbindings: []*Binding{},
	}
}

func (self *TopNode) Node() *Node {
	return self.node
}

type Pipestance struct {
	node        *Node
	subnodes    []Nodable
	retbindings []Binding
	prenodes    []Nodable
}

func NewPipestance(parent Nodable, callStm *CallStm, callables *Callables) *Pipestance {
	self := &Pipestance{}
	self.node = NewNode(parent, "pipeline", callStm, callables)

	return self
}

type Job interface {
}

type LocalJob struct {
}

func NewLocalJob() Job {
	return &LocalJob{}
}

type SGEJob struct {
}

func NewSGEJob() Job {
	return &SGEJob{}
}

type Runtime struct {
	mroPath        string
	stagecodePath  string
	libPath        string
	adaptersPath   string
	jobConstructor func() Job
	ptreeTable     map[string]*Ast
	srcTable       map[string]string
	typeTable      map[string]string
	codeVersion    string
	/* queue goes here */
}

func NewRuntime(jobMode string, pipelinesPath string) *Runtime {
	JOBRUNNER_TABLE := map[string]func() Job{
		"local": NewLocalJob,
		"sge":   NewSGEJob,
	}

	thispath, _ := filepath.Abs(path.Dir(os.Args[0]))
	return &Runtime{
		mroPath:        path.Join(pipelinesPath, "mro"),
		stagecodePath:  path.Join(pipelinesPath, "stages"),
		libPath:        path.Join(pipelinesPath, "lib"),
		adaptersPath:   path.Join(thispath, "..", "adapters"),
		jobConstructor: JOBRUNNER_TABLE[jobMode],
		ptreeTable:     map[string]*Ast{},
		srcTable:       map[string]string{},
		typeTable:      map[string]string{},
		codeVersion:    "",
	}
}

func (self *Runtime) Instantiate(psid string, src string, pipestancePath string) (*Pipestance, error) {
	global, err := ParseCall(src)
	if err != nil {
		return nil, err
	}
	ptree, ok := self.ptreeTable[global.call.Id()]
	if !ok {
		return nil, &MarioError{fmt.Sprintf("PipelineNotFoundError: '%s'", global.call.Id())}
	}
	pipestance := NewPipestance(NewTopNode(self, psid, pipestancePath), global.call, ptree.callables)
	return pipestance, nil
}

func (self *Runtime) Compile(fname string) (*Ast, error) {
	processedSrc, ptree, err := ParseFile(path.Join(self.mroPath, fname))
	if err != nil {
		return nil, err
	}
	for _, pipeline := range ptree.pipelines {
		self.ptreeTable[pipeline.Id()] = ptree
		self.srcTable[pipeline.Id()] = processedSrc
	}
	return ptree, nil
}

func (self *Runtime) CompileAll() (int, error) {
	dirs, _ := ioutil.ReadDir(self.mroPath)
	count := 0
	for _, dir := range dirs {
		if !(path.Ext(dir.Name()) == ".mro") {
			continue
		}
		if path.Base(dir.Name())[0:1] == "_" {
			continue
		}
		_, err := self.Compile(path.Join(dir.Name()))
		if err != nil {
			return 0, err
		}
		count += 1
	}
	return count, nil
}

func main() {
	runtime := NewRuntime("sge", "/Users/alex/Home/git/pipelines/src")
	count, err := runtime.CompileAll()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("Successfully compiled %d mro files.", count)
}
