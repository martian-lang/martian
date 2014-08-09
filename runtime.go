package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	_ "reflect"
	"runtime"
	_ "time"
)

/****************************************************
 * Helpers
 */
func mkdirAsync(p string, done chan bool) int {
	go func() {
		fmt.Println("start", p)
		os.Mkdir(p, 0700)
		//time.Sleep(time.Microsecond * 1)
		//runtime.Gosched()
		fmt.Println("done ", p)
		done <- true
	}()
	return 1
}

/****************************************************
 * Runtime Model Classes
 */
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
	value     interface{}
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
			parentBinding := self.node.parent.Node().argbindings[valueExp.id]
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
		self.value = bindStm.exp.(*ValExp).value
		// Unwrap array values.

	}
	return self
}

func (self *Binding) resolve(argPermute map[string]interface{}) interface{} {
	self.waiting = false
	if self.mode == "value" {
		if argPermute == nil {
			return self.value
		}
		if self.sweep {
			return argPermute[self.id]
		} else {
			return self.value
		}
	}
	if argPermute == nil {
		return nil
	}
	matchedFork := self.boundNode.Node().matchFork(argPermute)
	outputs := matchedFork.metadata.readSync("outs")
	if output, ok := outputs[self.output]; ok {
		return output
	}
	self.waiting = true
	return nil
}

type Metadata struct {
	path      string
	contents  map[string]interface{}
	filesPath string
}

func NewMetadata(p string) *Metadata {
	self := &Metadata{}
	self.path = p
	self.filesPath = path.Join(p, "files")
	return self
}

func (self *Metadata) mkdirs(done chan bool) int {
	count := mkdirAsync(self.path, done)
	count += mkdirAsync(self.filesPath, done)
	return count
}

func (self *Metadata) cache() {
	if _, ok := self.contents["complete"]; ok {
		return
	}
	self.contents = map[string]interface{}
    
}

type Chunk struct {
	stagestance *Stagestance
	fork        *Fork
	index       int
	chunkDef    map[string]int
	path        string
	fqname      string
	metadata    *Metadata
}

func NewChunk(nodable Nodable, fork *Fork, index int, chunkDef map[string]int) *Chunk {
	self := &Chunk{}
	self.stagestance = nodable.(*Stagestance)
	self.fork = fork
	self.index = index
	self.chunkDef = chunkDef
	self.path = path.Join(fork.path, fmt.Sprintf("chnk%d", index))
	self.fqname = fork.fqname + fmt.Sprintf(".chnk%d", index)
	self.metadata = NewMetadata(self.path)
	if !self.stagestance.split {
		// If we're not splitting, just set the sole chunk's filesPath
		// to the filesPath of the parent fork, to save a pseudo-join copy.
		self.metadata.filesPath = self.fork.metadata.filesPath
	}
	done := make(chan bool)
	count := self.mkdirs(done)
	for i := 0; i < count; i++ {
		<-done
	}
	return self
}

func (self *Chunk) mkdirs(done chan bool) int {
	return self.metadata.mkdirs(done)
}

type Fork struct {
	node           *Node
	index          int
	path           string
	fqname         string
	metadata       *Metadata
	split_metadata *Metadata
	join_metadata  *Metadata
	chunks         []*Chunk
	argPermute     map[string]interface{}
}

func NewFork(nodable Nodable, index int, argPermute []int) *Fork {
	self := &Fork{}
	self.node = nodable.Node()
	self.index = index
	self.path = path.Join(self.node.path, fmt.Sprintf("fork%d", index))
	self.fqname = self.node.fqname + fmt.Sprintf(".fork%d", index)
	self.metadata = NewMetadata(self.path)
	self.split_metadata = NewMetadata(path.Join(self.path, "split"))
	self.join_metadata = NewMetadata(path.Join(self.path, "join"))
	self.chunks = []*Chunk{}
	return self
}

func (self *Fork) mkdirs(done chan bool) int {
	count := 0
	count += self.metadata.mkdirs(done)
	count += self.split_metadata.mkdirs(done)
	count += self.join_metadata.mkdirs(done)
	return count
}

func (self *Fork) getState() string {
	return ""
}

/****************************************************
 * Node Classes
 */
type Nodable interface {
	Node() *Node
}

type Node struct {
	parent        Nodable
	rt            *Runtime
	kind          string
	name          string
	fqname        string
	path          string
	metadata      *Metadata
	outparams     *Params
	argbindings   map[string]*Binding
	sweepbindings []*Binding
	subnodes      map[string]Nodable
	prenodes      map[string]Nodable
	forks         []*Fork
}

func (self *Node) Node() *Node { return self }

func NewNode(parent Nodable, kind string, callStm *CallStm, callables *Callables) *Node {
	self := &Node{}
	self.parent = parent

	self.rt = parent.Node().rt
	self.kind = kind
	self.name = callStm.id
	self.fqname = parent.Node().fqname + "." + self.name
	self.path = path.Join(parent.Node().path, self.name)
	self.metadata = NewMetadata(self.path)

	self.outparams = callables.table[self.name].OutParams()
	self.argbindings = map[string]*Binding{}
	self.subnodes = map[string]Nodable{}
	self.prenodes = map[string]Nodable{}

	for id, bindStm := range callStm.bindings.table {
		binding := NewBinding(self, bindStm)
		self.argbindings[id] = binding
		if binding.mode == "reference" && binding.boundNode != nil {
			self.prenodes[binding.boundNode.Node().name] = binding.boundNode
		}
	}
	// Do not set state = getState here, or else nodes will wrongly report complete before the first refreshMetadata call
	return self
}

func (self *Node) mkdirs(done chan bool) int {
	count := mkdirAsync(self.path, done)
	for _, fork := range self.forks {
		count += fork.mkdirs(done)
	}
	for _, subnode := range self.subnodes {
		count += subnode.Node().mkdirs(done)
	}
	return count
}

// State and dataflow management (synchronous)
func (self *Node) getState() string {
	complete := true
	for _, fork := range self.forks {
		if fork.getState() != "complete" {
			complete = false
			break
		}
	}
	if complete {
		return "complete"
	}
	for _, fork := range self.forks {
		if fork.getState() == "failed" {
			return "failed"
		}
	}
	for _, prenode := range self.prenodes {
		if prenode.Node().getState() != "complete" {
			return "waiting"
		}
	}
	return "running"
}

// Sweep management
func (self *Node) buildForks(bindings []*Binding) {
	// Use a map to uniquify bindings by id.
	bindingTable := map[string]*Binding{}

	// Add local sweep bindings.
	for _, binding := range bindings {
		if binding.sweep {
			bindingTable[binding.id] = binding
		}
	}
	// Add upstream sweep bindings (from prenodes).
	for _, prenode := range self.prenodes {
		for _, binding := range prenode.Node().sweepbindings {
			bindingTable[binding.id] = binding
		}
	}
	// Add all unique bindings to self.sweepbindings.
	paramIds := []string{}
	argRanges := []interface{}{}
	for _, binding := range bindingTable {
		self.sweepbindings = append(self.sweepbindings, binding)
		paramIds = append(paramIds, binding.id)
		argRanges = append(argRanges, binding.resolve(nil))
	}

	// Build out argument permutations.
	//for _, binding: =
}

func (self *Node) matchFork(targetArgPermute map[string]interface{}) *Fork {
	if targetArgPermute == nil {
		return nil
	}
	for _, fork := range self.forks {
		every := true
		for paramId, argValue := range fork.argPermute {
			if targetArgPermute[paramId] != argValue {
				every = false
				break
			}
		}
		if every {
			return fork
		}
	}
	return nil
}

type Pipestance struct {
	node        *Node
	retbindings []Binding
}

func (self *Pipestance) Node() *Node { return self.node }

func NewPipestance(parent Nodable, callStm *CallStm, callables *Callables) *Pipestance {
	self := &Pipestance{}
	self.node = NewNode(parent, "pipeline", callStm, callables)

	pipeline := callables.table[self.node.name].(*Pipeline)
	for _, subcallStm := range pipeline.calls {
		callable := callables.table[subcallStm.id]
		switch callable.(type) {
		case *Stage:
			self.node.subnodes[subcallStm.id] = NewPythonStagestance(self, subcallStm, callables)
		case *Pipeline:
			self.node.subnodes[subcallStm.id] = NewPipestance(self, subcallStm, callables)
		}

	}
	return self
}

type Stagestance struct {
	node          *Node
	stagecodePath string
	split         bool
	// buildForks
}

func (self *Stagestance) Node() *Node { return self.node }

func NewStagestance(parent Nodable, callStm *CallStm, callables *Callables) *Stagestance {
	self := &Stagestance{}
	self.node = NewNode(parent, "stage", callStm, callables)
	fmt.Println("==", self.node.name)
	stage := callables.table[self.node.name].(*Stage)
	self.stagecodePath = path.Join(self.node.rt.stagecodePath, stage.src.path)
	self.split = len(stage.splitParams.list) > 0
	/* buildforks */
	return self
}

/****************************************************
 * Python Adapter
 */
type PythonStagestance struct {
	stagestance   *Stagestance
	stagecodeLang string
	adaptersPath  string
	libPath       string
}

func (self *PythonStagestance) Node() *Node { return self.stagestance.node }

func NewPythonStagestance(parent Nodable, callStm *CallStm, callables *Callables) *PythonStagestance {
	self := &PythonStagestance{}
	self.stagestance = NewStagestance(parent, callStm, callables)
	self.stagecodeLang = "Python"
	self.adaptersPath = path.Join(self.stagestance.node.rt.adaptersPath, "python")
	self.libPath = path.Join(self.stagestance.node.rt.libPath, "python")
	/* check for __init__.py */
	return self
}

/****************************************************
 * Job Execution Classes
 */
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

/****************************************************
 * Runtime
 */
type TopNode struct {
	node *Node
}

func (self *TopNode) Node() *Node { return self.node }

func NewTopNode(rt *Runtime, psid string, p string) *TopNode {
	self := &TopNode{}
	self.node = &Node{}
	self.node.path = p
	self.node.rt = rt
	self.node.fqname = "ID." + psid
	return self
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
	callStm := global.call

	ptree, ok := self.ptreeTable[callStm.Id()]
	if !ok {
		return nil, &MarioError{fmt.Sprintf("PipelineNotFoundError: '%s'", callStm.Id())}
	}
	pipeline := ptree.callables.table[callStm.Id()].(*Pipeline)
	if err := callStm.bindings.check(global, pipeline, pipeline.InParams()); err != nil {
		return nil, err
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

func (self *Runtime) InvokeWithSource(psid string, src string, pipestancePath string) (*Pipestance, error) {
	pipestance, err := self.Instantiate(psid, src, pipestancePath)
	if err != nil {
		return nil, err
	}
	done := make(chan bool)
	count := pipestance.Node().mkdirs(done)
	fmt.Println(count)
	for i := 0; i < count; i++ {
		<-done
	}
	return pipestance, nil
}

/****************************************************
 * Main Routine
 */
func main() {
	runtime.GOMAXPROCS(4)
	rt := NewRuntime("sge", "/Users/alex/Home/git/pipelines/src")
	count, err := rt.CompileAll()
	if err != nil {
		fmt.Println(err.Error())
	}
	callSrc :=
		`call ANALYTICS(
    read_path = [path("/mnt/analysis/marsoc/pipestances/HA911ADXX/PREPROCESS/HA911ADXX/1.4.1/PREPROCESS/DEMULTIPLEX/fork0/files/demultiplexed_fastq_path")],
    sample_indices = ["GTTGTAGT","TTCTATGC","CAACCCAA","CCGATTAG"],
    lanes = null,
    genome = "PhiX",
    targets_file = null,
    confident_regions = null,
    trim_length = 0,
    barcode_whitelist = "737K-april-2014",
    primers = ["R1-alt2:TTGCTCATTCCCTACACGACGCTCTTCCGATCT","R2RC:GTGACTGGAGTTCAGACGTGTGCTCTTCCGATCT","Alt2-10N:AATGATACGGCGACCACCGAGATCTACACTAGATCGCTTGCTCATTCCCTACACGACGCTCTTCCGATCTNNNNNNNNNN","P7RC:CAAGCAGAAGACGGCATACGAGAT","P5:AATGATACGGCGACCACCGAGA"],
    variant_results = null,
    sample_id = "2444",
    lena_url = "lena.10x.office",
)`
		/*
		   `call PREPROCESS(
		      run_path = path("/mnt/sequencing/test/sequencers/miseq00V/mini-bcl"),
		      #run_path = path("/Users/alex/tmp/netapp/sequencing/sequencers/miseq00V/mini-bcl"),
		      seq_run_id = "mini-bcl",
		      lena_url = null,
		   )`
		*/
	PIPESTANCE_PATH := "./HA191"
	os.MkdirAll(PIPESTANCE_PATH, 0700)
	_, err = rt.InvokeWithSource("HA191", callSrc, PIPESTANCE_PATH)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("Successfully compiled %d mro files.", count)
}
