package pm

import (
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/zero-os/0-core/base/pm/stream"

	"github.com/stretchr/testify/assert"
)

func TestJob(t *testing.T) {
	New()

	stdin := "hello world"
	cmd := Command{
		Command: CommandSystem,
		Arguments: MustArguments(
			SystemCommandArguments{
				Name:  "cat",
				StdIn: stdin,
			},
		),
	}

	job := newTestJob(&cmd, NewSystemProcess)

	job.start(false)

	result := job.Wait()
	if ok := assert.Equal(t, StateSuccess, result.State); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, stdin+"\n", result.Streams.Stdout()); !ok {
		t.Error()
	}
}

func TestJobMaxRestart(t *testing.T) {
	New()

	var counter int
	var action = func(cmd *Command) (interface{}, error) {
		counter++
		return nil, fmt.Errorf("error")
	}

	cmd := Command{
		MaxRestart: 3,
	}

	job := newTestJob(&cmd, internalProcessFactory(action))

	job.start(false)

	result := job.Wait()
	if ok := assert.Equal(t, StateError, result.State); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, 3, counter); !ok {
		t.Error()
	}
}

func TestJobMessages(t *testing.T) {
	New()

	var action = func(ctx *Context) (interface{}, error) {
		ctx.Log("debug message", stream.LevelDebug)
		ctx.Log("stdout message", stream.LevelStdout)
		ctx.Log("stderr message", stream.LevelStderr)

		return "result data", nil
	}

	cmd := Command{}

	job := newTestJob(&cmd, internalProcessFactoryWithCtx(action))

	var logs []*stream.Message
	var subscriber = func(msg *stream.Message) {
		logs = append(logs, msg)
	}

	job.Subscribe(subscriber)
	job.start(false)

	log.Info("waiting for command to exit")
	result := job.Wait()
	if ok := assert.Equal(t, StateSuccess, result.State); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, `"result data"`, result.Data); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, `stdout message`, result.Streams.Stdout()); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, `stderr message`, result.Streams.Stderr()); !ok {
		t.Error()
	}

	//Other levels (like debug) are forwarded to the logger instead, only stdout, stderr and return messages are captured
	//by the result object
	//also any subscriber to the job will get the messages, as we did above
	if ok := assert.Len(t, logs, 4); !ok {
		t.Fatal()
	}

	msg := logs[0] //first message
	if ok := assert.Equal(t, `debug message`, msg.Message); !ok {
		t.Error()
	}
}

func TestJobTimeout(t *testing.T) {
	New()

	cmd := Command{
		Command: CommandSystem,
		Arguments: MustArguments(
			SystemCommandArguments{
				Name: "sleep",
				Args: []string{"10s"},
			},
		),
		MaxTime: 1,
	}

	job := newTestJob(&cmd, NewSystemProcess)

	job.start(false)

	result := job.Wait()
	if ok := assert.Equal(t, StateTimeout, result.State); !ok {
		t.Error()
	}
}
func TestJobSignal(t *testing.T) {
	New()

	cmd := Command{
		Command: CommandSystem,
		Arguments: MustArguments(
			SystemCommandArguments{
				Name: "sleep",
				Args: []string{"10s"},
			},
		),
	}

	job := newTestJob(&cmd, NewSystemProcess)

	go func() {
		time.Sleep(time.Second)
		job.Signal(syscall.SIGINT)
	}()

	job.start(false)

	result := job.Wait()
	if ok := assert.Equal(t, StateError, result.State); !ok {
		t.Error()
	}
}

func TestJobMaxRecurring(t *testing.T) {
	New()

	var counter int
	var action = func(cmd *Command) (interface{}, error) {
		counter++
		return nil, nil
	}

	cmd := Command{
		RecurringPeriod: 1,
	}

	job := newTestJob(&cmd, internalProcessFactory(action))

	go func() {
		time.Sleep(4 * time.Second)
		job.Signal(syscall.SIGKILL)
	}()

	job.start(false)

	//it will never reach here.
	result := job.Wait()
	if ok := assert.Equal(t, StateKilled, result.State); !ok {
		t.Error()
	}

	if ok := assert.InDelta(t, 4, counter, 1); !ok {
		t.Error()
	}
}

func TestJobHooks(t *testing.T) {
	t.Skip()
	New()

	cmd := Command{
		Command: CommandSystem,
		Arguments: MustArguments(
			SystemCommandArguments{
				Name: "sleep",
				Args: []string{"2s"},
			},
		),
	}

	var delayCalled, exitCalled, pidCalled bool
	var delay = func() {
		delayCalled = true
	}

	var exit = func(s bool) {
		exitCalled = true
	}

	var pid = func(i int) {
		pidCalled = true
	}

	hooks := []RunnerHook{
		&DelayHook{Delay: time.Second, Action: delay},
		&ExitHook{Action: exit},
		&PIDHook{Action: pid},
	}

	job := newTestJob(&cmd, NewSystemProcess, hooks...)

	job.start(false)

	result := job.Wait()
	if ok := assert.Equal(t, StateSuccess, result.State); !ok {
		t.Error()
	}

	if ok := assert.True(t, delayCalled && exitCalled && pidCalled); !ok {
		t.Error()
	}
}
