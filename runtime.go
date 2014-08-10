package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	_ "reflect"
	"runtime"
	"sync"
	"time"
)

/****************************************************
 * Helpers
 */
func mkdir(p string) {
	err := os.Mkdir(p, 0700)
	if err != nil {
		fmt.Println(err)
	}
}

func cartesianProduct(valueSets []interface{}) []interface{} {
	perms := []interface{}{[]interface{}{}}
	for _, valueSet := range valueSets {
		newPerms := []interface{}{}
		for _, perm := range perms {
			for _, value := range valueSet.([]interface{}) {
				perm := perm.([]interface{})
				newPerm := make([]interface{}, len(perm))
				copy(newPerm, perm)
				newPerm = append(newPerm, value)
				newPerms = append(newPerms, newPerm)
			}
		}
		perms = newPerms
	}
	return perms
}

/*
func main() {
	aoa := []interface{}{
		[]interface{}{ "always" },
		[]interface{}{ "a", "b" },
		[]interface{}{ "red", "blue" },
		[]interface{}{ 1, 2, 3 },
		//[]interface{}{ "tree", "bush" },
		//[]interface{}{ "cat", "dog" },
	}
	fmt.Println(cartesianProduct(aoa))
}
*/

/****************************************************
 * Runtime Model Classes
 */
type Parameter struct {
}

func NewParameter() {

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
	value     interface{}
}

func NewBinding(node *Node, bindStm *BindStm) *Binding {
	self := &Binding{}
	self.node = node
	//fmt.Println("HAHA", self.id, self.node.parent.Node())
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
			self.boundNode = parentBinding.boundNode
			self.output = parentBinding.output
			self.value = parentBinding.value
			self.id = bindStm.id
			self.valexp = "self." + valueExp.id
		} else if valueExp.kind == "call" {
			self.mode = "reference"
			//fmt.Println("HAAaaAAAAAAAA", valueExp.id, self.node.parent.Node())
			self.boundNode = self.node.parent.Node().subnodes[valueExp.id]
			//fmt.Println("ho", self.boundNode)
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
		if self.value != nil && valueExp.kind == "array" {
			value := []interface{}{}
			for _, e := range self.value.([]Exp) {
				value = append(value, e.(*ValExp).value)
			}
			self.value = value
		}
	}
	return self
}

func NewReturnBinding(node *Node, bindStm *BindStm) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.id
	self.tname = bindStm.tname
	self.mode = "reference"
	valueExp := bindStm.exp.(*RefExp)
	self.boundNode = self.node.subnodes[valueExp.id] // from node, NOT parent; this is diff from Binding
	self.output = valueExp.outputId
	if valueExp.outputId == "default" {
		self.valexp = valueExp.id
	} else {
		self.valexp = valueExp.id + "." + valueExp.outputId
	}
	return self
}

func (self *Binding) resolve(argPermute map[string]interface{}) interface{} {
	self.waiting = false
	if self.mode == "value" {
		if argPermute == nil {
			// In this case we want to get the raw value, which might be
			// a sweep array.
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
	outputs := matchedFork.metadata.read("outs").(map[string]interface{})
	output, _ := outputs[self.output]
	self.waiting = true
	return output
}

func resolveBindings(bindings map[string]*Binding, argPermute map[string]interface{}) map[string]interface{} {
	resolvedBindings := map[string]interface{}{}
	for id, binding := range bindings {
		resolvedBindings[id] = binding.resolve(argPermute)
	}
	return resolvedBindings
}
func makeOutArgs(outParams *Params, filesPath string) map[string]interface{} {
	args := map[string]interface{}{}
	for id, param := range outParams.table {
		switch param.Tname() {
		case "file":
			args[id] = path.Join(filesPath, param.Id()+"."+param.Tname())
		case "path":
			args[id] = path.Join(filesPath, param.Id())
		default:
		}
	}
	return args
}

type Metadata struct {
	path      string
	contents  map[string]bool
	filesPath string
}

func NewMetadata(p string) *Metadata {
	self := &Metadata{}
	self.path = p
	self.filesPath = path.Join(p, "files")
	return self
}

func (self *Metadata) glob() []string {
	paths, _ := filepath.Glob(path.Join(self.path, "_*"))
	return paths
}

func (self *Metadata) mkdirs() {
	mkdir(self.path)
	mkdir(self.filesPath)
}

func (self *Metadata) cache() {
	if !self.exists("complete") {
		self.contents = map[string]bool{}
		paths := self.glob()
		for _, p := range paths {
			self.contents[path.Base(p)[1:]] = true
		}
	}
}

func (self *Metadata) restartIfFailed() {
	if self.exists("errors") {
		self.contents = map[string]bool{}
		paths := self.glob()
		for _, p := range paths {
			os.Remove(p)
		}
	}
}

func (self *Metadata) makePath(name string) string {
	return path.Join(self.path, "_"+name)
}
func (self *Metadata) exists(name string) bool {
	_, ok := self.contents[name]
	return ok
}
func (self *Metadata) readRaw(name string) string {
	bytes, _ := ioutil.ReadFile(self.makePath(name))
	return string(bytes)
}
func (self *Metadata) read(name string) interface{} {
	var v interface{}
	json.Unmarshal([]byte(self.readRaw(name)), v)
	return v
}
func (self *Metadata) writeRaw(name string, text string) {
	ioutil.WriteFile(self.makePath(name), []byte(text), 0600)
}
func (self *Metadata) write(name string, object interface{}) {
	bytes, _ := json.Marshal(object)
	self.writeRaw(name, string(bytes))
}

func (self *Metadata) append(name string, text string) {
	f, _ := os.OpenFile(self.makePath(name), os.O_WRONLY|os.O_CREATE, 0700)
	f.Write([]byte(text))
	f.Close()
}

func (self *Metadata) writeTime(name string) {
	self.writeRaw(name, time.Now().Format("2006-01-02 15:04:05"))
}

func (self *Metadata) remove(name string) {
	os.Remove(self.makePath(name))
}

type Chunk struct {
	node        *Node
	stagestance *Stagestance
	fork        *Fork
	index       int
	chunkDef    map[string]interface{}
	path        string
	fqname      string
	metadata    *Metadata
}

func NewChunk(nodable Nodable, fork *Fork, index int, chunkDef map[string]interface{}) *Chunk {
	self := &Chunk{}
	self.node = nodable.Node()
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
	// we have to mkdirs here because runtime might have been interrupted after chunk_defs were
	// written but before next step interval caused the actual creation of the chnk folders.
	// in that scenario, upon restart the fork step would try to write _args into chnk folders
	// that don't exist.
	self.mkdirs()
	return self
}

func (self *Chunk) mkdirs() {
	self.metadata.mkdirs()
}

func (self *Chunk) getState() string {
	if self.metadata.exists("errors") {
		return "failed"
	}
	if self.metadata.exists("complete") {
		return "complete"
	}
	if self.metadata.exists("log") {
		return "running"
	}
	if self.metadata.exists("jobinfo") {
		return "queued"
	}
	return "ready"
}

func (self *Chunk) step() {
	if self.getState() == "ready" {
		resolvedBindings := resolveBindings(self.node.argbindings, self.fork.argPermute)
		for id, value := range self.chunkDef {
			resolvedBindings[id] = value
		}
		self.metadata.write("args", resolvedBindings)
		self.metadata.write("outs", makeOutArgs(self.node.outparams, self.metadata.filesPath))
		// runjob
	}
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

func NewFork(nodable Nodable, index int, argPermute map[string]interface{}) *Fork {
	self := &Fork{}
	self.node = nodable.Node()
	self.index = index
	self.path = path.Join(self.node.path, fmt.Sprintf("fork%d", index))
	self.fqname = self.node.fqname + fmt.Sprintf(".fork%d", index)
	self.metadata = NewMetadata(self.path)
	self.split_metadata = NewMetadata(path.Join(self.path, "split"))
	self.join_metadata = NewMetadata(path.Join(self.path, "join"))
	self.chunks = []*Chunk{}
	self.argPermute = argPermute
	return self
}

func (self *Fork) collectMetadatas() []*Metadata {
	metadatas := []*Metadata{self.metadata, self.split_metadata, self.join_metadata}
	for _, chunk := range self.chunks {
		metadatas = append(metadatas, chunk.metadata)
	}
	return metadatas
}

func (self *Fork) mkdirs() {
	self.metadata.mkdirs()
	self.split_metadata.mkdirs()
	self.join_metadata.mkdirs()
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
	}
	for _, binding := range self.argbindings {
		if binding.mode == "reference" && binding.boundNode != nil {
			self.prenodes[binding.boundNode.Node().name] = binding.boundNode
		}
	}
	// Do not set state = getState here, or else nodes will wrongly report complete before the first refreshMetadata call
	return self
}

/*
func (self *Node) mkdirs() {
	mkdir(self.path)
	for _, fork := range self.forks {
		fork.mkdirs()
	}
	for _, subnode := range self.subnodes {
		subnode.Node().mkdirs()
	}
	fmt.Println("got done", self.name)
}
*/

/*
func (self *Node) mkdirs() {
	mkdir(self.path)
	done := make(chan bool, 1)
	for _, fork := range self.forks {
		go func(fork *Fork) {
			fork.mkdirs()
			done <- true
		}(fork)
	}
	for i := 0; i < len(self.forks); i++ {
		<-done
	}
	done = make(chan bool)
	for _, subnode := range self.subnodes {
		go func(subnode Nodable) {
			subnode.Node().mkdirs()
			done <- true
		}(subnode)
	}
	for j := 0; j < len(self.subnodes); j++ {
		<-done
	}
	fmt.Println("got done", self.name)
}
*/
func (self *Node) mkdirs(wg *sync.WaitGroup) {
	fmt.Println("hello", self.name)
	mkdir(self.path)
	for _, fork := range self.forks {
		wg.Add(1)
		go func(f *Fork) {
			defer wg.Done()
			f.mkdirs()
		}(fork)
	}
	for _, subnode := range self.subnodes {
		wg.Add(1)
		go func(n Nodable) {
			defer wg.Done()
			n.Node().mkdirs(wg)
		}(subnode)
	}
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
func (self *Node) buildForks(bindings map[string]*Binding) {
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

	for _, binding := range bindingTable {
		self.sweepbindings = append(self.sweepbindings, binding)
	}

	// Add all unique bindings to self.sweepbindings.
	paramIds := []string{}
	argRanges := []interface{}{}
	for _, binding := range self.sweepbindings {
		//	self.sweepbindings = append(self.sweepbindings, binding)
		paramIds = append(paramIds, binding.id)
		argRanges = append(argRanges, binding.resolve(nil))
	}

	// Build out argument permutations.
	for i, valPermute := range cartesianProduct(argRanges) {
		argPermute := map[string]interface{}{}
		for j, paramId := range paramIds {
			argPermute[paramId] = valPermute.([]interface{})[j]
		}
		self.forks = append(self.forks, NewFork(self, i, argPermute))
	}
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

func (self *Node) collectMetadatas() []*Metadata {
	metadatas := []*Metadata{self.metadata}
	for _, fork := range self.forks {
		metadatas = append(metadatas, fork.collectMetadatas()...)
	}
	return metadatas
}

func (self *Node) refreshMetadata(done chan bool) int {
	metadatas := self.collectMetadatas()
	for _, metadata := range metadatas {
		go func(m *Metadata) {
			m.cache()
			done <- true
		}(metadata)
	}
	return len(metadatas)
}

func (self *Node) restartFailedMetadatas(done chan bool) int {
	metadatas := self.collectMetadatas()
	for _, metadata := range metadatas {
		go func(m *Metadata) {
			m.restartIfFailed()
			done <- true
		}(metadata)
	}
	return len(metadatas)
}

type Stagestance struct {
	node          *Node
	stagecodePath string
	split         bool
}

func (self *Stagestance) Node() *Node { return self.node }

func NewStagestance(parent Nodable, callStm *CallStm, callables *Callables) *Stagestance {
	self := &Stagestance{}
	self.node = NewNode(parent, "stage", callStm, callables)
	//fmt.Println("==", self.node.name)
	stage := callables.table[self.node.name].(*Stage)
	self.stagecodePath = path.Join(self.node.rt.stagecodePath, stage.src.path)
	self.split = len(stage.splitParams.list) > 0
	self.node.buildForks(self.node.argbindings)
	return self
}

type Pipestance struct {
	node        *Node
	retbindings map[string]*Binding
}

func (self *Pipestance) Node() *Node { return self.node }

func NewPipestance(parent Nodable, callStm *CallStm, callables *Callables) *Pipestance {
	self := &Pipestance{}
	self.node = NewNode(parent, "pipeline", callStm, callables)

	// Build subcall tree.
	pipeline := callables.table[self.node.name].(*Pipeline)
	for _, subcallStm := range pipeline.calls {
		callable := callables.table[subcallStm.id]
		switch callable.(type) {
		case *Stage:
			self.node.subnodes[subcallStm.id] = NewStagestance(self.Node(), subcallStm, callables)
		case *Pipeline:
			self.node.subnodes[subcallStm.id] = NewPipestance(self.Node(), subcallStm, callables)
		}
	}

	// Also depends on stages bound to return values.
	self.retbindings = map[string]*Binding{}
	for id, bindStm := range pipeline.ret.bindings.table {
		binding := NewReturnBinding(self.node, bindStm)
		self.retbindings[id] = binding
		if binding.mode == "reference" && binding.boundNode != nil {
			self.node.prenodes[binding.boundNode.Node().name] = binding.boundNode
		}
	}

	self.node.buildForks(self.retbindings)
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
	paths, err := filepath.Glob(self.mroPath + "/[^_]*.mro")
	if err != nil {
		return 0, err
	}
	for _, p := range paths {
		_, err := self.Compile(path.Base(p))
		if err != nil {
			return 0, err
		}
	}
	return len(paths), nil
}

func (self *Runtime) InvokeWithSource(psid string, src string, pipestancePath string) (*Pipestance, error) {
	pipestance, err := self.Instantiate(psid, src, pipestancePath)
	if err != nil {
		return nil, err
	}

	//pipestance.Node().mkdirs()
	var wg sync.WaitGroup
	pipestance.Node().mkdirs(&wg)
	wg.Wait()
	return pipestance, nil
}

/****************************************************
 * Main Routine
 */
func main() {
	runtime.GOMAXPROCS(4)
	rt := NewRuntime("sge", "/Users/aywong/Home/Work/10X/git/pipelines/src")
	count, err := rt.CompileAll()
	if err != nil {
		fmt.Println(err.Error())
	}
	callSrc :=
		`call ANALYTICS(
    read_path = [path("/mnt/analysis/marsoc/pipestances/HA911ADXX/PREPROCESS/HA911ADXX/1.4.1/PREPROCESS/DEMULTIPLEX/fork0/files/demultiplexed_fastq_path")],
    sample_indices = ["GTTGTAGT","TTCTATGC","CAACCCAA","CCGATTAG"],
    lanes = null,
    genome = sweep(["PhiX","hg19"]),
    targets_file = null,
    confident_regions = null,
    trim_length = sweep([0,10]),
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
