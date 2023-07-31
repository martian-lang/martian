// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Martian job managers for local and remote (SGE, LSF, etc) modes.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

// JobManager provides an interface for executing jobs.
type JobManager interface {
	execJob(shellCmd string,
		args []string,
		env map[string]string,
		md *Metadata,
		res *JobResources,
		fqname, shellName string,
		preflight bool)
	endJob(*Metadata)

	// Given a list of candidate job IDs, returns a list of jobIds which may be
	// still queued or running, as well as the stderr output of the queue check.
	// If this job manager doesn't know how to check the queue or the query
	// fails, it simply returns the list it was given.
	checkQueue([]string, context.Context) ([]string, string)
	// Returns true if checkQueue does something useful.
	hasQueueCheck() bool
	// Returns the amount of time to wait, after a job is found to be unknown
	// to the job manager, before declaring the job dead.  This is to protect
	// against races between NFS caching in the directories Martian watches and
	// whatever the queue manager uses to syncronize state.
	queueCheckGrace() time.Duration

	// Update resource availability.
	//
	// For local mode, this means free memory and possibly loadavg.
	//
	// For remote job managers, this means maxjobs.
	refreshResources(localMode bool) error
	GetSystemReqs(*JobResources) JobResources
	GetMaxCores() int
	GetMaxMemGB() int
	GetSettings() *JobManagerSettings

	// Reset the max jobs semaphore.
	resetMaxJobs()
	// Re-add a job to the max jobs semaphore.
	reattach(*Metadata)
}

// Set environment variables which control thread count.  Do not override
// envs from above.
func threadEnvs(self JobManager, threads int,
	envs map[string]string) map[string]string {
	thr := strconv.Itoa(threads)
	newEnvs := make(map[string]string)
	for _, env := range self.GetSettings().ThreadEnvs {
		newEnvs[env] = thr
	}
	for key, value := range envs {
		newEnvs[key] = value
	}
	return newEnvs
}

//
// Helper functions for job manager file parsing
//

type JobModeEnv struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type JobModeJson struct {
	Cmd             string        `json:"cmd"`
	Args            []string      `json:"args,omitempty"`
	QueueQuery      string        `json:"queue_query,omitempty"`
	ResourcesOpt    string        `json:"resopt"`
	JobEnvs         []*JobModeEnv `json:"envs"`
	QueueQueryGrace int           `json:"queue_query_grace_secs,omitempty"`
	AlwaysVmem      bool          `json:"mem_is_vmem,omitempty"`
}

type JobManagerSettings struct {
	ThreadEnvs    []string `json:"thread_envs"`
	ThreadsPerJob int      `json:"threads_per_job"`
	MemGBPerJob   int      `json:"memGB_per_job"`
	ExtraVmemGB   int      `json:"extra_vmem_per_job,omitempty"`
}

type JobManagerJson struct {
	JobSettings *JobManagerSettings            `json:"settings"`
	JobModes    map[string]*JobModeJson        `json:"jobmodes"`
	ProfileMode map[ProfileMode]*ProfileConfig `json:"profiles"`
}

type jobManagerConfig struct {
	jobSettings      *JobManagerSettings
	queueQueryCmd    string
	jobResourcesOpt  string
	jobTemplate      string
	jobCmd           string
	jobCmdArgs       []string
	queueQueryGrace  time.Duration
	alwaysVmem       bool
	threadingEnabled bool
}

func getJobConfig(profileMode ProfileMode) (*JobManagerJson, error) {
	jobPath := util.RelPath(path.Join("..", "jobmanagers"))

	// Check for existence of job manager JSON file
	jobJsonFile := path.Join(jobPath, "config.json")
	b, err := os.ReadFile(jobJsonFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Job manager config file %s does not exist.",
				jobJsonFile)
		} else {
			return nil, fmt.Errorf(
				"Error reading job manager config file %s.\n%w",
				jobJsonFile, err)
		}
	}
	util.LogInfo("jobmngr", "Job config = %s", jobJsonFile)

	// Parse job manager JSON file
	var jobJson *JobManagerJson
	if err := json.Unmarshal(b, &jobJson); err != nil {
		return nil, fmt.Errorf(
			"Job manager config file %s does not contain valid JSON: %w",
			jobJsonFile, err)
	}

	// Validate settings fields
	jobSettings := jobJson.JobSettings
	if jobSettings == nil {
		return nil, fmt.Errorf(
			"Job manager config file %s should contain 'settings' field.",
			jobJsonFile)
	}
	if jobSettings.ThreadsPerJob <= 0 {
		return nil, fmt.Errorf(
			"Job manager config file %s contains invalid default threads per job.",
			jobJsonFile)
	}
	if jobSettings.MemGBPerJob <= 0 {
		return nil, fmt.Errorf(
			"Job manager config %s contains invalid default memory (GB) per job.",
			jobJsonFile)
	}

	if profileMode != "" && profileMode != DisableProfile {
		if _, ok := jobJson.ProfileMode[profileMode]; !ok {
			return nil, fmt.Errorf(
				"Invalid profile mode: %s. Valid profile modes: %s",
				profileMode, allProfileModes(jobJson.ProfileMode))
		}
	}
	return jobJson, nil
}

func verifyJobManager(jobMode string, jobJson *JobManagerJson, memGBPerCore int) (jobManagerConfig, error) {
	if jobMode == localMode {
		// Local job mode only needs to verify settings parameters
		return jobManagerConfig{
			jobSettings: jobJson.JobSettings,
		}, nil
	}

	var jobTemplateFile string
	var jobErrorMsg string

	jobModeJson, ok := jobJson.JobModes[jobMode]
	if ok {
		jobPath := util.RelPath(path.Join("..", "jobmanagers"))
		jobTemplateFile = path.Join(jobPath, jobMode+".template")
		exampleJobTemplateFile := jobTemplateFile + ".example"
		jobErrorMsg = fmt.Sprintf(`Job manager template file %s does not exist.

To set up a job manager template, please follow instructions in %s.`,
			jobTemplateFile, exampleJobTemplateFile)
	} else {
		jobTemplateFile = jobMode
		jobMode = strings.TrimSuffix(path.Base(jobTemplateFile), ".template")

		jobModeJson, ok = jobJson.JobModes[jobMode]
		if !strings.HasSuffix(jobTemplateFile, ".template") || !ok {
			return jobManagerConfig{}, fmt.Errorf(
				"Job manager template file %s must be named <name_of_job_manager>.template.",
				jobTemplateFile)
		}
		jobErrorMsg = fmt.Sprintf(
			"Job manager template file %s does not exist.",
			jobTemplateFile)
	}

	jobCmd := jobModeJson.Cmd
	if len(jobModeJson.Args) == 0 {
		util.LogInfo("jobmngr", "Job submit command = %s",
			jobCmd)
	} else {
		util.LogInfo("jobmngr", "Job submit comand = %s %s",
			jobCmd, strings.Join(jobModeJson.Args, " "))
	}

	jobResourcesOpt := jobModeJson.ResourcesOpt
	util.LogInfo("jobmngr", "Job submit resources option = %s",
		jobResourcesOpt)

	// Check for existence of job manager template file
	b, err := os.ReadFile(jobTemplateFile)
	if os.IsNotExist(err) {
		return jobManagerConfig{}, errors.New(jobErrorMsg)
	}
	util.LogInfo("jobmngr", "Job template = %s", jobTemplateFile)
	jobTemplate := string(b)

	// Check if template includes threading.
	jobThreadingEnabled := false
	if strings.Contains(jobTemplate, "__MRO_THREADS__") {
		jobThreadingEnabled = true
	} else if memGBPerCore > 0 {
		util.Println(`
CLUSTER MODE WARNING:
   Thread reservations are not enabled in your job template.  The
   --mempercore option will have no effect.
Please consult the documentation for details.
`)
	}

	// Check if memory reservations or mempercore are enabled
	if !strings.Contains(jobTemplate, "__MRO_MEM_GB") &&
		!strings.Contains(jobTemplate, "__MRO_MEM_MB") &&
		memGBPerCore <= 0 {
		util.Println(`
CLUSTER MODE WARNING:
   Memory reservations are not enabled in your job template.
   To avoid memory over-subscription, we highly recommend that you enable
   memory reservations on your cluster, or use the --mempercore option.
Please consult the documentation for details.
`)
	}

	// Verify job command exists
	if _, err := exec.LookPath(jobCmd); err != nil {
		return jobManagerConfig{}, fmt.Errorf(
			"Job command '%s' not found in %q",
			jobCmd, os.Getenv("PATH"))
	}

	// Verify environment variables
	envs := [][]string{}
	for _, entry := range jobModeJson.JobEnvs {
		envs = append(envs, []string{entry.Name, entry.Description})
	}
	util.EnvRequire(envs, true)

	var queueGrace time.Duration
	if jobModeJson.QueueQuery != "" {
		queueGrace = time.Duration(jobModeJson.QueueQueryGrace) * time.Second
		// Default to 1 hour.
		if queueGrace == 0 {
			queueGrace = time.Hour
		}
	}

	return jobManagerConfig{
		jobSettings:      jobJson.JobSettings,
		jobCmd:           jobCmd,
		jobCmdArgs:       jobModeJson.Args,
		alwaysVmem:       jobModeJson.AlwaysVmem,
		queueQueryCmd:    jobModeJson.QueueQuery,
		queueQueryGrace:  queueGrace,
		jobResourcesOpt:  jobResourcesOpt,
		jobTemplate:      jobTemplate,
		threadingEnabled: jobThreadingEnabled,
	}, nil
}
