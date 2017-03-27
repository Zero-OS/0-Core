package process

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	psutils "github.com/shirou/gopsutil/process"
	"os/exec"
	"syscall"
)

type ContainerCommandArguments struct {
	Name        string            `json:"name"`
	Dir         string            `json:"dir"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	HostNetwork bool              `json:"host_network"`
	Chroot      string            `json:"chroot"`
}

type containerProcessImpl struct {
	cmd     *core.Command
	args    ContainerCommandArguments
	pid     int
	process *psutils.Process

	table PIDTable
}

func NewContainerProcess(table PIDTable, cmd *core.Command) Process {
	process := &containerProcessImpl{
		cmd:   cmd,
		table: table,
	}

	json.Unmarshal(*cmd.Arguments, &process.args)
	return process
}

func (process *containerProcessImpl) Command() *core.Command {
	return process.cmd
}

func (process *containerProcessImpl) Signal(sig syscall.Signal) error {
	if process.process != nil {
		return syscall.Kill(int(process.process.Pid), sig)
	}

	return fmt.Errorf("process not found")
}

func (process *containerProcessImpl) Kill() error {
	return process.Signal(syscall.SIGTERM)
}

//GetStats gets stats of an external process
func (process *containerProcessImpl) Stats() *ProcessStats {
	stats := ProcessStats{}
	stats.Cmd = process.cmd

	defer func() {
		if r := recover(); r != nil {
			log.Warningf("processUtils panic: %s", r)
		}
	}()

	ps := process.process
	if ps == nil {
		return &stats
	}
	ps.CPUAffinity()
	cpu, err := ps.Percent(0)
	if err == nil {
		stats.CPU = cpu
	}

	mem, err := ps.MemoryInfo()
	if err == nil {
		stats.RSS = mem.RSS
		stats.VMS = mem.VMS
		stats.Swap = mem.Swap
	}

	stats.Debug = fmt.Sprintf("%d", process.process.Pid)

	return &stats
}

func (process *containerProcessImpl) Run() (<-chan *stream.Message, error) {
	cmd := exec.Command(process.args.Name,
		process.args.Args...)
	cmd.Dir = process.args.Dir

	var flags uintptr = syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS

	if !process.args.HostNetwork {
		flags |= syscall.CLONE_NEWNET
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     process.args.Chroot,
		Cloneflags: flags,
		Setsid:     true,
	}

	for k, v := range process.args.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k, v))
	}

	err := process.table.Register(func() (int, error) {
		err := cmd.Start()
		if err != nil {
			return 0, err
		}

		return cmd.Process.Pid, nil
	})

	if err != nil {
		log.Errorf("Failed to start process(%s): %s", process.cmd.ID, err)
		return nil, err
	}

	channel := make(chan *stream.Message)

	process.pid = cmd.Process.Pid
	psProcess, _ := psutils.NewProcess(int32(process.pid))
	process.process = psProcess

	go func(channel chan *stream.Message) {
		//make sure all outputs are closed before waiting for the process
		//to exit.
		defer close(channel)

		state := process.table.WaitPID(process.pid)

		log.Infof("Process %s exited with state: %d", process.cmd, state.ExitStatus())

		if state.ExitStatus() == 0 {
			channel <- stream.MessageExitSuccess
		} else {
			channel <- stream.MessageExitError
		}
	}(channel)

	return channel, nil
}
