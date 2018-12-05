package mgr

import (
	"fmt"

	"github.com/threefoldtech/0-core/base/pm"
)

//implement internal processes

/*
Global command ProcessConstructor registery
*/
var factories = map[string]pm.ProcessFactory{
	pm.CommandSystem: NewSystemProcess,
}

//GetProcessFactory gets a process factory from command name
func GetProcessFactory(cmd *pm.Command) pm.ProcessFactory {
	return factories[cmd.Command]
}

//Register registers a command process factory
func Register(name string, factory pm.ProcessFactory) {
	if _, ok := factories[name]; ok {
		panic(fmt.Sprintf("command registered with same name: %s", name))
	}
	factories[name] = factory
}

/*
RegisterExtension registers a new command (extension) so it can be executed via commands
*/
func RegisterExtension(cmd string, exe string, workdir string, cmdargs []string, env map[string]string) error {
	if _, ok := factories[cmd]; ok {
		return fmt.Errorf("job factory with the same name already registered: %s", cmd)
	}

	Register(cmd, extensionProcessFactory(exe, workdir, cmdargs, env))
	return nil
}

//RegisterBuiltIn registers a built in function
func RegisterBuiltIn(name string, runnable Runnable) {
	Register(name, NewInternalProcess(runnable))
}

//RegisterBuiltInWithCtx registers a built in function that accepts a command and a context
func RegisterBuiltInWithCtx(name string, runnable RunnableWithCtx) {
	Register(name, NewInternalProcessWithCtx(runnable))
}
