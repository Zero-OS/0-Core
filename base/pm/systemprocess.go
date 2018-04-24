package pm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	psutils "github.com/shirou/gopsutil/process"
	"github.com/zero-os/0-core/base/pm/stream"
)

type SystemCommandArguments struct {
	Name  string            `json:"name"`
	Dir   string            `json:"dir"`
	Args  []string          `json:"args"`
	Env   map[string]string `json:"env"`
	StdIn string            `json:"stdin"`
}

func (s *SystemCommandArguments) String() string {
	return fmt.Sprintf("%v %s %v (%s)", s.Env, s.Name, s.Args, s.Dir)
}

type systemProcessImpl struct {
	cmd     *Command
	args    SystemCommandArguments
	pid     int
	process *psutils.Process

	table PIDTable
}

func NewSystemProcess(table PIDTable, cmd *Command) Process {
	process := &systemProcessImpl{
		cmd:   cmd,
		table: table,
	}

	json.Unmarshal(*cmd.Arguments, &process.args)
	return process
}

func (p *systemProcessImpl) Command() *Command {
	return p.cmd
}

//GetStats gets stats of an external p
func (p *systemProcessImpl) Stats() *ProcessStats {
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

	stats.Debug = fmt.Sprintf("%d", p.process.Pid)

	return &stats
}

func (p *systemProcessImpl) Signal(sig syscall.Signal) error {
	if p.process == nil {
		return fmt.Errorf("process not found")
	}

	kill := int(p.process.Pid)
	if !p.cmd.Flags.NoSetPGID {
		gid, err := syscall.Getpgid(kill)
		if err != nil {
			return err
		}
		kill = -gid
	}

	return syscall.Kill(kill, sig)

}

func (p *systemProcessImpl) Run() (ch <-chan *stream.Message, err error) {
	var stdin, stdout, stderr *os.File

	name, err := exec.LookPath(p.args.Name)
	if err != nil {
		return nil, NotFoundError(err)
	}

	var env []string

	if len(p.args.Env) > 0 {
		env = append(env, os.Environ()...)
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

	var toClose []*os.File
	var input *os.File
	if len(p.args.StdIn) != 0 {
		stdin, input, err = os.Pipe()
		if err != nil {
			return nil, err
		}
	} else {
		stdin, err = os.Open(os.DevNull)
		if err != nil {
			return nil, err
		}
	}

	toClose = append(toClose, stdin)

	if !p.cmd.Flags.NoOutput {
		handler := func(m *stream.Message) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("error while writing output: %s", err)
				}
			}()
			channel <- m
		}

		var outRead, errRead *os.File
		outRead, stdout, err = os.Pipe()
		if err != nil {
			return nil, err
		}

		errRead, stderr, err = os.Pipe()
		if err != nil {
			return nil, err
		}
		toClose = append(toClose, stdout, stderr)
		wg.Add(2)
		stream.Consume(&wg, outRead, 1, handler)
		stream.Consume(&wg, errRead, 2, handler)
	}

	attrs := os.ProcAttr{
		Dir: p.args.Dir,
		Env: env,
		Files: []*os.File{
			stdin, stdout, stderr,
		},
		Sys: &syscall.SysProcAttr{
			Setpgid: !p.cmd.Flags.NoSetPGID,
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
		for _, f := range toClose {
			f.Close()
		}
		return ps.Pid, nil
	})

	if err != nil {
		return
	}

	p.pid = ps.Pid
	psProcess, _ := psutils.NewProcess(int32(p.pid))
	p.process = psProcess

	if input != nil {
		//write data to command stdin.
		io.WriteString(input, p.args.StdIn)
		input.Close()
	}

	go func(channel chan *stream.Message) {
		//make sure all outputs are closed before waiting for the p
		defer close(channel)
		state := p.table.WaitPID(p.pid)
		//wait for all streams to finish copying
		wg.Wait()
		ps.Release()
		code := state.ExitStatus()
		log.Debugf("Process %s exited with state: %d", p.cmd, code)
		if code == 0 {
			channel <- &stream.Message{
				Meta: stream.NewMeta(stream.LevelStdout, stream.ExitSuccessFlag),
			}
		} else {
			channel <- &stream.Message{
				Meta: stream.NewMetaWithCode(uint32(1000+code), stream.LevelStderr, stream.ExitErrorFlag),
			}
		}
	}(channel)

	return channel, nil
}
