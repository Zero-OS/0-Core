package pm

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
