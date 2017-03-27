package process

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	"syscall"
)

/*
Runnable represents a runnable built in function that can be managed by the process manager.
*/
type Runnable func(*core.Command) (interface{}, error)

/*
internalProcess implements a Procss interface and represents an internal (go) process that can be managed by the process manager
*/
type internalProcess struct {
	runnable Runnable
	cmd      *core.Command
}

/*
internalProcessFactory factory to build Runnable processes
*/
func NewInternalProcessFactory(runnable Runnable) ProcessFactory {
	factory := func(_ PIDTable, cmd *core.Command) Process {
		return &internalProcess{
			runnable: runnable,
			cmd:      cmd,
		}
	}

	return factory
}

/*
Cmd returns the internal process command
*/
func (process *internalProcess) Command() *core.Command {
	return process.cmd
}

/*
Run runs the internal process
*/
func (process *internalProcess) Run() (<-chan *stream.Message, error) {

	channel := make(chan *stream.Message)

	go func(channel chan *stream.Message) {
		defer func() {
			if err := recover(); err != nil {
				m, _ := json.Marshal(fmt.Sprintf("%v", err))
				channel <- &stream.Message{
					Level:   stream.LevelResultJSON,
					Message: string(m),
				}
				channel <- stream.MessageExitError
			}

			close(channel)
		}()

		value, err := process.runnable(process.cmd)
		msg := stream.Message{
			Level: stream.LevelResultJSON,
		}

		if err != nil {
			m, _ := json.Marshal(err.Error())
			msg.Message = string(m)
		} else {
			m, _ := json.Marshal(value)
			msg.Message = string(m)
		}

		channel <- &msg
		if err != nil {
			channel <- stream.MessageExitError
		} else {
			channel <- stream.MessageExitSuccess
		}

	}(channel)

	return channel, nil
}

/*
Kill kills internal process (not implemented)
*/
func (process *internalProcess) Kill() error {
	//you can't kill an internal process.
	return nil
}

func (process *internalProcess) Signal(sig syscall.Signal) error {
	return nil
}
