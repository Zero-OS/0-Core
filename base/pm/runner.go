package pm

// #cgo LDFLAGS: -lcap
// #include <sys/capability.h>
import "C"
import (
	"fmt"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/utils"
	"runtime"
	"sync"
	"syscall"
	"time"
)

const (
	StreamBufferSize = 1000

	meterPeriod = 30 * time.Second
)

type Runner interface {
	Command() *core.Command
	Terminate()
	Process() process.Process
	Wait() *core.JobResult
	StartTime() int64

	start(unprivileged bool)
}

type runnerImpl struct {
	manager *PM
	command *core.Command
	factory process.ProcessFactory
	kill    chan int

	process   process.Process
	hooks     []RunnerHook
	startTime time.Time

	waitOnce sync.Once
	result   *core.JobResult
	wg       sync.WaitGroup
}

/*
NewRunner creates a new runner object that is bind to this PM instance.

:manager: Bind this runner to this PM instance
:command: Command to run
:factory: Process factory associated with command type.
:hooksDelay: Fire the hooks after this delay in seconds if the process is still running. Basically it's a delay for if the
            command didn't exit by then we assume it's running successfully
            values are:
            	- 1 means hooks are only called when the command exits
            	0   means use default delay (default is 2 seconds)
            	> 0 Use that delay

:hooks: Optionals hooks that are called if the process is considered RUNNING successfully.
        The process is considered running, if it ran with no errors for 2 seconds, or exited before the 2 seconds passes
        with SUCCESS exit code.
*/
func NewRunner(manager *PM, command *core.Command, factory process.ProcessFactory, hooks ...RunnerHook) Runner {
	runner := &runnerImpl{
		manager: manager,
		command: command,
		factory: factory,
		kill:    make(chan int),
		hooks:   hooks,
	}

	runner.wg.Add(1)
	return runner
}

func (runner *runnerImpl) Command() *core.Command {
	return runner.command
}

func (runner *runnerImpl) timeout() <-chan time.Time {
	var timeout <-chan time.Time
	if runner.command.MaxTime > 0 {
		timeout = time.After(time.Duration(runner.command.MaxTime) * time.Second)
	}
	return timeout
}

//set the bounding set for current thread, of course this is un-reversable once set on the
//pm it affects all threads from now on.
func (process *runnerImpl) setUnprivileged() {
	//drop bounding set for children.
	bound := []uintptr{
		C.CAP_SETPCAP,
		C.CAP_SYS_MODULE,
		C.CAP_SYS_RAWIO,
		C.CAP_SYS_PACCT,
		C.CAP_SYS_ADMIN,
		C.CAP_SYS_NICE,
		C.CAP_SYS_RESOURCE,
		C.CAP_SYS_TIME,
		C.CAP_SYS_TTY_CONFIG,
		C.CAP_AUDIT_CONTROL,
		C.CAP_MAC_OVERRIDE,
		C.CAP_MAC_ADMIN,
		C.CAP_NET_ADMIN,
		C.CAP_SYSLOG,
		C.CAP_DAC_READ_SEARCH,
		C.CAP_LINUX_IMMUTABLE,
		C.CAP_NET_BROADCAST,
		C.CAP_IPC_LOCK,
		C.CAP_IPC_OWNER,
		C.CAP_SYS_PTRACE,
		C.CAP_SYS_BOOT,
		C.CAP_LEASE,
		C.CAP_WAKE_ALARM,
		C.CAP_BLOCK_SUSPEND,
	}

	for _, c := range bound {
		syscall.Syscall6(syscall.SYS_PRCTL, syscall.PR_CAPBSET_DROP,
			c, 0, 0, 0, 0)
	}
}

func (runner *runnerImpl) run(unprivileged bool) (jobresult *core.JobResult) {
	runner.startTime = time.Now()
	jobresult = core.NewBasicJobResult(runner.command)
	jobresult.State = core.StateError

	defer func() {
		jobresult.StartTime = int64(time.Duration(runner.startTime.UnixNano()) / time.Millisecond)
		endtime := time.Now()

		jobresult.Time = endtime.Sub(runner.startTime).Nanoseconds() / int64(time.Millisecond)

		if err := recover(); err != nil {
			jobresult.State = core.StateError
			jobresult.Critical = fmt.Sprintf("PANIC(%v)", err)
		}
	}()

	runner.process = runner.factory(runner, runner.command)

	ps := runner.process
	runtime.LockOSThread()
	if unprivileged {
		runner.setUnprivileged()
	}
	channel, err := ps.Run()
	runtime.UnlockOSThread()

	if err != nil {
		//this basically means process couldn't spawn
		//which indicates a problem with the command itself. So restart won't
		//do any good. It's better to terminate it immediately.
		jobresult.Data = err.Error()
		return jobresult
	}

	var result *stream.Message
	var critical string

	stdoutBuffer := stream.NewBuffer(StreamBufferSize)
	stderrBuffer := stream.NewBuffer(StreamBufferSize)

	timeout := runner.timeout()

	handlersTicker := time.NewTicker(1 * time.Second)
	defer handlersTicker.Stop()
loop:
	for {
		select {
		case <-runner.kill:
			if ps, ok := ps.(process.Signaler); ok {
				ps.Signal(syscall.SIGTERM)
			}
			jobresult.State = core.StateKilled
			break loop
		case <-timeout:
			if ps, ok := ps.(process.Signaler); ok {
				ps.Signal(syscall.SIGKILL)
			}
			jobresult.State = core.StateTimeout
			break loop
		case <-handlersTicker.C:
			d := time.Now().Sub(runner.startTime)
			for _, hook := range runner.hooks {
				go hook.Tick(d)
			}
		case message := <-channel:
			if utils.In(stream.ResultMessageLevels, message.Level) {
				result = message
			} else if message.Level == stream.LevelExitState {
				jobresult.State = message.Message
				break loop
			} else if message.Level == stream.LevelStdout {
				stdoutBuffer.Append(message.Message)
			} else if message.Level == stream.LevelStderr {
				stderrBuffer.Append(message.Message)
			} else if message.Level == stream.LevelCritical {
				critical = message.Message
			}

			for _, hook := range runner.hooks {
				go hook.Message(message)
			}

			//by default, all messages are forwarded to the manager for further processing.
			runner.manager.msgCallback(runner.command, message)
		}
	}

	runner.process = nil

	//consume channel to the end to allow process to cleanup properly
	for _ = range channel {
		//noop.
	}

	if result != nil {
		jobresult.Level = result.Level
		jobresult.Data = result.Message
	}

	jobresult.Streams = core.Streams{
		stdoutBuffer.String(),
		stderrBuffer.String(),
	}

	jobresult.Critical = critical

	return jobresult
}

func (runner *runnerImpl) start(unprivileged bool) {
	runs := 0
	var result *core.JobResult
	defer func() {
		if result != nil {
			runner.result = result
			runner.manager.resultCallback(runner.command, result)

			runner.waitOnce.Do(func() {
				runner.wg.Done()
			})
		}

		runner.manager.cleanUp(runner)
	}()

loop:
	for {
		result = runner.run(unprivileged)

		for _, hook := range runner.hooks {
			hook.Exit(result.State)
		}

		if runner.command.Protected {
			//immediate restart
			log.Debugf("Respawning protected service")
			continue
		}

		if result.State == core.StateKilled {
			//we never restart a killed process.
			break
		}

		restarting := false
		var restartIn time.Duration

		if result.State != core.StateSuccess && runner.command.MaxRestart > 0 {
			runs++
			if runs < runner.command.MaxRestart {
				log.Debugf("Restarting '%s' due to upnormal exit status, trials: %d/%d", runner.command, runs+1, runner.command.MaxRestart)
				restarting = true
				restartIn = 1 * time.Second
			}
		}

		if runner.command.RecurringPeriod > 0 {
			restarting = true
			restartIn = time.Duration(runner.command.RecurringPeriod) * time.Second
		}

		if restarting {
			log.Debugf("Recurring '%s' in %s", runner.command, restartIn)
			select {
			case <-time.After(restartIn):
			case <-runner.kill:
				log.Infof("Command %s Killed during scheduler sleep", runner.command)
				result.State = core.StateKilled
				break loop
			}
		} else {
			break
		}
	}
}

func (runner *runnerImpl) Terminate() {
	runner.kill <- 1
}

func (runner *runnerImpl) Process() process.Process {
	return runner.process
}

func (runner *runnerImpl) Wait() *core.JobResult {
	runner.wg.Wait()
	return runner.result
}

//implement PIDTable
//intercept pid registration to fire the correct hooks.
func (runner *runnerImpl) Register(g process.GetPID) error {
	return runner.manager.Register(func() (int, error) {
		pid, err := g()
		if err != nil {
			return 0, err
		}

		for _, hook := range runner.hooks {
			go hook.PID(pid)
		}

		return pid, err
	})
}

func (runner *runnerImpl) WaitPID(pid int) syscall.WaitStatus {
	return runner.manager.WaitPID(pid)
}

func (runner *runnerImpl) StartTime() int64 {
	return int64(time.Duration(runner.startTime.UnixNano()) / time.Millisecond)
}
