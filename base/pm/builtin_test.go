package pm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zero-os/0-core/base/pm/stream"
)

func TestGetProcessFactory(t *testing.T) {
	factory := GetProcessFactory(&Command{Command: "wrong"})

	if ok := assert.Nil(t, factory); !ok {
		t.Error()
	}

	//CommandSystem is a built in command it is always available
	factory = GetProcessFactory(&Command{Command: CommandSystem})
	if ok := assert.NotNil(t, factory); !ok {
		t.Fatal()
	}
}

func TestRegisterBuiltIn(t *testing.T) {
	runnable := func(cmd *Command) (interface{}, error) {
		return nil, nil
	}

	name := "test.builtin.1"
	RegisterBuiltIn(name, runnable)

	cmd := Command{
		Command: name,
	}

	factory := GetProcessFactory(&cmd)
	if ok := assert.NotNil(t, factory); !ok {
		t.Fatal()
	}

	process := factory(nil, &cmd)

	_, ok := process.(*internalProcess)

	if !assert.True(t, ok) {
		t.Fatal()
	}
}

func TestRegisterBuiltInWithCtx(t *testing.T) {
	runnable := func(ctx *Context) (interface{}, error) {
		return nil, nil
	}

	name := "test.builtin.ctx"
	RegisterBuiltInWithCtx(name, runnable)

	cmd := Command{
		Command: name,
	}

	factory := GetProcessFactory(&cmd)
	if ok := assert.NotNil(t, factory); !ok {
		t.Fatal()
	}

	process := factory(nil, &cmd)

	_, ok := process.(*internalProcess)

	if !assert.True(t, ok) {
		t.Fatal()
	}
}

func TestExtension(t *testing.T) {
	RegisterExtension("test.extension", "ls", "/", nil, nil)

	cmd := Command{
		Command:   "test.extension",
		Arguments: MustArguments(M{}),
	}

	factory := GetProcessFactory(&cmd)
	if ok := assert.NotNil(t, factory); !ok {
		t.Fatal()
	}

	var pidTable TestingPIDTable
	process := factory(&pidTable, &cmd)

	_, ok := process.(*extensionProcess)

	if !assert.True(t, ok) {
		t.Fatal()
	}

	ch, err := process.Run()

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	var msgs []*stream.Message
	for msg := range ch {
		msgs = append(msgs, msg)
	}

	if ok := assert.NotEmpty(t, msgs); !ok {
		t.Fatal()
	}

	term := msgs[len(msgs)-1]
	if ok := assert.True(t, term.Meta.Is(stream.ExitSuccessFlag)); !ok {
		t.Error()
	}
}
