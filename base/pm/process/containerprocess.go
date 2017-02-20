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
	Name   string            `json:"name"`
	Dir    string            `json:"dir"`
	Args   []string          `json:"args"`
	Env    map[string]string `json:"env"`
	Chroot string            `json:"chroot"`
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

func (process *containerProcessImpl) Kill() {
	//should force system process to exit.
	if process.process != nil {
		process.process.Kill()
	}
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

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     process.args.Chroot,
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		Setpgid:    true,
	}

	for k, v := range process.args.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k, v))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = process.table.Register(func() (int, error) {
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

	msgInterceptor := func(msg *stream.Message) {
		if msg.Level == stream.LevelExitState {
			//the level exit state is for internal use only, shouldn't
			//be sent by the app itself, if found, we change the level to err.
			msg.Level = stream.LevelStderr
		}

		channel <- msg
	}

	// start consuming outputs.
	outConsumer := stream.NewConsumer(stdout, 1)
	outConsumer.Consume(msgInterceptor)

	errConsumer := stream.NewConsumer(stderr, 2)
	errConsumer.Consume(msgInterceptor)

	go func(channel chan *stream.Message) {
		//make sure all outputs are closed before waiting for the process
		//to exit.
		defer close(channel)

		<-outConsumer.Signal()
		<-errConsumer.Signal()
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
