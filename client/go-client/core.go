package client

import (
	"fmt"
	"syscall"

	"github.com/google/shlex"
)

type Job struct {
	Command   Command `json:"cmd"`
	CPU       float64 `json:"cpu"`
	RSS       int64   `json:"rss"`
	Swap      int64   `json:"swap"`
	VMS       int64   `json:"vms"`
	StartTime int64   `json:"starttime"`
}

type JobStats struct {
	CPU   float64 `json:"cpu"`
	RSS   uint64  `json:"rss"`
	VMS   uint64  `json:"vms"`
	Swap  uint64  `json:"swap"`
	Debug string  `json:"debug,ommitempty"`
}

type Process struct {
	Command    string    `json:"cmdline"`
	Createtime uint64    `json:"createtime"`
	Cpu        CPUStats  `json:"cpu"`
	PID        ProcessId `json:"pid"`
	PPID       ProcessId `json:"ppid"`
	RSS        uint64    `json:"rss"`
	Swap       uint64    `json:"swap"`
	VMS        uint64    `json:"vms"`
}

type CPUStats struct {
	GuestNice float64 `json:"guestnice"`
	Idle      float64 `json:"idle"`
	IoWait    float64 `json:"iowait"`
	Irq       float64 `json:"irq"`
	Nice      float64 `json:"nice"`
	SoftIrq   float64 `json:"softirq"`
	Steal     float64 `json:"steal"`
	Stolen    float64 `json:"stolen"`
	System    float64 `json:"system"`
	User      float64 `json:"user"`
}

type CoreManager interface {
	System(cmd string, env map[string]string, cwd string, stdin string, opt ...Option) (JobId, error)
	SystemArgs(cmd string, args []string, env map[string]string, cwd string, stdin string, opt ...Option) (JobId, error)
	Bash(bash string, stdin string, opt ...Option) (JobId, error)
	Ping() error
	Jobs() ([]Job, error)
	Job(job JobId) (*Job, error)
	KillJob(job JobId, signal syscall.Signal) error
	KillAllJobs() error
	Process(pid ProcessId) (*Process, error)
	ProcessAlive(pid ProcessId) (bool, error)
	Processes() ([]Process, error)
	KillProcess(pid ProcessId, signal syscall.Signal) error
	State() (*JobStats, error)
}

type coreMgr struct {
	cl Client
}

func Core(client Client) CoreManager {
	return &coreMgr{client}
}

func (s *coreMgr) SystemArgs(cmd string, args []string, env map[string]string, cwd string, stdin string, opt ...Option) (JobId, error) {
	return s.cl.Raw("core.system", A{
		"name":  cmd,
		"args":  args,
		"dir":   cwd,
		"stdin": stdin,
		"env":   env,
	}, opt...)
}

func (s *coreMgr) System(cmd string, env map[string]string, cwd string, stdin string, opt ...Option) (JobId, error) {
	parts, err := shlex.Split(cmd)
	if err != nil {
		return JobId(""), err
	}

	if len(parts) == 0 {
		return JobId(""), fmt.Errorf("empty command")
	}
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	return s.SystemArgs(parts[0], args, env, cwd, stdin, opt...)
}

func (s *coreMgr) Bash(bash, stdin string, opt ...Option) (JobId, error) {
	return s.cl.Raw("bash", A{
		"script": bash,
		"stdin":  stdin,
	}, opt...)
}

func (s *coreMgr) Ping() error {
	_, err := sync(s.cl, "core.ping", A{})
	return err
}

func (s *coreMgr) Jobs() ([]Job, error) {
	res, err := sync(s.cl, "job.list", A{})
	if err != nil {
		return nil, err
	}

	var jobs []Job
	if err := res.Json(&jobs); err != nil {
		return nil, err
	}

	return jobs, err
}

func (s *coreMgr) Job(job JobId) (*Job, error) {
	res, err := sync(s.cl, "job.list", A{
		"id": job,
	})
	if err != nil {
		return nil, err
	}

	var jobs []Job
	if err := res.Json(&jobs); err != nil {
		return nil, err
	}

	if len(jobs) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return &jobs[0], err
}

func (s *coreMgr) KillJob(job JobId, signal syscall.Signal) error {
	_, err := sync(s.cl, "job.kill", A{
		"id":     job,
		"signal": signal,
	})

	return err
}

func (s *coreMgr) KillAllJobs() error {
	if res, err := sync(s.cl, "job.killall", A{}); res != nil && res.State == StateKilled {
		return nil
	} else {
		return err
	}
}

// Process gets process that has pid
func (s *coreMgr) Process(pid ProcessId) (*Process, error) {
	res, err := sync(s.cl, "process.list", A{
		"pid": pid,
	})
	if err != nil {
		return nil, err
	}

	var processes []Process
	if err := res.Json(&processes); err != nil {
		return nil, err
	}

	if len(processes) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return &processes[0], err
}

// ProcessAlive checks if process exists
func (s *coreMgr) ProcessAlive(pid ProcessId) (bool, error) {
	res, err := sync(s.cl, "process.list", A{
		"pid": pid,
	})
	if err != nil {
		return false, err
	}

	var processes []Process
	if err := res.Json(&processes); err != nil {
		return false, err
	}

	return len(processes) > 0, nil
}

// Processes List all processes
func (s *coreMgr) Processes() ([]Process, error) {
	res, err := sync(s.cl, "process.list", A{})
	if err != nil {
		return nil, err
	}

	var processes []Process
	if err := res.Json(&processes); err != nil {
		return nil, err
	}

	return processes, err
}

func (s *coreMgr) KillProcess(pid ProcessId, signal syscall.Signal) error {
	_, err := sync(s.cl, "process.kill", A{
		"pid":    pid,
		"signal": signal,
	})

	return err
}

func (s *coreMgr) State() (*JobStats, error) {
	res, err := sync(s.cl, "core.state", A{})
	if err != nil {
		return nil, err
	}

	var jobStats JobStats
	if err := res.Json(&jobStats); err != nil {
		return nil, err
	}

	return &jobStats, err
}
