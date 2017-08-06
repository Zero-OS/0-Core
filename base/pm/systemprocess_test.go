package pm

import (
	"github.com/stretchr/testify/assert"
	"github.com/zero-os/0-core/base/pm/stream"
	"syscall"
	"testing"
)

type table struct {
}

func (t *table) RegisterPID(g GetPID) error {
	_, err := g()
	if err != nil {
		return err
	}

	return nil
}

func (t *table) WaitPID(pid int) syscall.WaitStatus {
	var status syscall.WaitStatus
	syscall.Wait4(pid, &status, 0, nil)
	return status
}

func TestSystemProcess_Run(t *testing.T) {
	ps := NewSystemProcess(&table{}, &Command{
		Arguments: MustArguments(
			SystemCommandArguments{
				Name: "echo",
				Args: []string{"hello", "world"},
			},
		),
	})

	ch, err := ps.Run()

	if ok := assert.Nil(t, err); !ok {
		t.Fatal(err)
	}

	if ok := assert.NotNil(t, ch); !ok {
		t.Fatal()
	}
	var messages []*stream.Message
	for msg := range ch {
		messages = append(messages, msg)
	}

	if ok := assert.Len(t, messages, 2); !ok { //the 2nd is for termination message
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello world", messages[0].Message); !ok {
		t.Error()
	}
}

func TestSystemProcess_RunStderr(t *testing.T) {
	ps := NewSystemProcess(&table{}, &Command{
		Arguments: MustArguments(
			SystemCommandArguments{
				Name: "sh",
				Args: []string{"-c", "echo 'hello world' 1>&2"},
			},
		),
	})

	ch, err := ps.Run()

	if ok := assert.Nil(t, err); !ok {
		t.Fatal(err)
	}

	if ok := assert.NotNil(t, ch); !ok {
		t.Fatal()
	}
	var messages []*stream.Message
	for msg := range ch {
		messages = append(messages, msg)
	}

	if ok := assert.Len(t, messages, 2); !ok { //the 2nd is for termination message
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello world", messages[0].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(2), messages[0].Meta.Level()); !ok {
		t.Error()
	}
}

func TestSystemProcess_RunStdin(t *testing.T) {
	ps := NewSystemProcess(&table{}, &Command{
		Arguments: MustArguments(
			SystemCommandArguments{
				Name:  "cat",
				StdIn: "hello world",
			},
		),
	})

	ch, err := ps.Run()

	if ok := assert.Nil(t, err); !ok {
		t.Fatal(err)
	}

	if ok := assert.NotNil(t, ch); !ok {
		t.Fatal()
	}
	var messages []*stream.Message
	for msg := range ch {
		messages = append(messages, msg)
	}

	if ok := assert.Len(t, messages, 2); !ok { //the 2nd is for termination message
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello world", messages[0].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(1), messages[0].Meta.Level()); !ok {
		t.Error()
	}

	//check exit status as well
	if ok := assert.True(t, messages[1].Meta.Is(stream.ExitSuccessFlag)); !ok {
		t.Error()
	}
}
