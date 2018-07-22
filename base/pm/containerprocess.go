package pm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"

	psutils "github.com/shirou/gopsutil/process"
	"github.com/zero-os/0-core/base/pm/stream"
)

//ContainerCommandArguments arguments for container command
type ContainerCommandArguments struct {
	Name        string            `json:"name"`
	Dir         string            `json:"dir"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	HostNetwork bool              `json:"host_network"`
	Chroot      string            `json:"chroot"`
	Log         string            `json:"log"`
}

func (c *ContainerCommandArguments) String() string {
	return fmt.Sprintf("%s %v %s", c.Name, c.Args, c.Chroot)
}

//Channel is a 2 way communication channel that is mainly used
//to talk to the main containerd process `coreX`
type Channel interface {
	io.ReadWriteCloser
}

type channel struct {
	r *os.File
	w *os.File
	o sync.Once
}

func (c *channel) Close() error {
	c.o.Do(func() {
		c.r.Close()
		c.w.Close()
	})

	return nil
}

func (c *channel) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *channel) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

//ContainerProcess interface
type ContainerProcess interface {
	Process
	Channel() Channel
}

type containerProcessImpl struct {
	cmd     *Command
	args    ContainerCommandArguments
	pid     int
	process *psutils.Process
	ch      *channel

	table PIDTable
}

//NewContainerProcess creates a new contained process, used soley from
//the container subsystem. Clients can't create container process directly they
//instead has to go throught he container subsystem which does most of the heavy
//lifting.
func NewContainerProcess(table PIDTable, cmd *Command) Process {
	process := &containerProcessImpl{
		cmd:   cmd,
		table: table,
	}

	json.Unmarshal(*cmd.Arguments, &process.args)
	return process
}

func (p *containerProcessImpl) Command() *Command {
	return p.cmd
}

func (p *containerProcessImpl) Channel() Channel {
	return p.ch
}

func (p *containerProcessImpl) Signal(sig syscall.Signal) error {
	if p.process != nil {
		return syscall.Kill(-int(p.process.Pid), sig)
	}

	return fmt.Errorf("process not found")
}

//GetStats gets stats of an external process
func (p *containerProcessImpl) Stats() *ProcessStats {
	stats := ProcessStats{}

	defer func() {
		if r := recover(); r != nil {
			log.Warningf("processUtils panic: %s", r)
		}
	}()

	ps := p.process
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

	return &stats
}

func (p *containerProcessImpl) GetPID() int32 {
	ps := p.process
	if ps == nil {
		return -1
	}

	return ps.Pid
}

func (p *containerProcessImpl) setupChannel() (*os.File, *os.File, error) {
	lr, lw, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	rr, rw, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	p.ch = &channel{
		r: lr,
		w: rw,
	}

	return rr, lw, nil
}

func (p *containerProcessImpl) Run() (ch <-chan *stream.Message, err error) {
	//we don't do lookup on the name because the name
	//is only available under the chroot
	name := p.args.Name

	var env []string

	if len(p.args.Env) > 0 {
		for k, v := range p.args.Env {
			env = append(env, fmt.Sprintf("%v=%v", k, v))
		}
	}

	channel := make(chan *stream.Message)
	ch = channel
	defer func() {
		if err != nil {
			close(channel)
		}
	}()

	var wg sync.WaitGroup

	var flags uintptr = syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS

	if !p.args.HostNetwork {
		flags |= syscall.CLONE_NEWNET
	}

	r, w, err := p.setupChannel()
	if err != nil {
		return nil, err
	}
	var logf *os.File
	if len(p.args.Log) != 0 {
		logf, err = os.OpenFile(p.args.Log, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
	}

	attrs := os.ProcAttr{
		Dir: p.args.Dir,
		Env: env,
		Files: []*os.File{
			nil, logf, logf, r, w,
		},
		Sys: &syscall.SysProcAttr{
			Chroot:     p.args.Chroot,
			Cloneflags: flags,
			Setsid:     true,
		},
	}

	var ps *os.Process
	args := []string{name}
	args = append(args, p.args.Args...)
	_, err = p.table.RegisterPID(func() (int, error) {
		ps, err = os.StartProcess(name, args, &attrs)
		if err != nil {
			return 0, err
		}

		return ps.Pid, nil
	})

	if err != nil {
		return
	}

	p.pid = ps.Pid
	psProcess, _ := psutils.NewProcess(int32(p.pid))
	p.process = psProcess

	go func(channel chan *stream.Message) {
		//make sure all outputs are closed before waiting for the process
		defer close(channel)
		state := p.table.WaitPID(p.pid)
		//wait for all streams to finish copying
		wg.Wait()
		ps.Release()
		if err := p.ch.Close(); err != nil {
			log.Errorf("failed to close container channel: %s", err)
		}

		code := state.ExitStatus()
		log.Debugf("Process %s exited with state: %d", p.cmd, code)
		if code == 0 {
			channel <- &stream.Message{
				Meta: stream.NewMeta(stream.LevelStdout, stream.ExitSuccessFlag),
			}
		} else {
			channel <- &stream.Message{
				Meta: stream.NewMetaWithCode(uint32(code), stream.LevelStderr, stream.ExitErrorFlag),
			}
		}

	}(channel)

	return channel, nil
}
