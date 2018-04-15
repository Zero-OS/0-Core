package pm

import (
	"testing"

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

	log.Info("waiting for command to exit")
	result := job.Wait()
	if ok := assert.Equal(t, StateSuccess, result.State); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, stdin+"\n", result.Streams.Stdout()); !ok {
		t.Error()
	}
}

func TestJobHooks(t *testing.T) {
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

	log.Info("waiting for command to exit")
	result := job.Wait()
	if ok := assert.Equal(t, StateSuccess, result.State); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, stdin+"\n", result.Streams.Stdout()); !ok {
		t.Error()
	}
}
