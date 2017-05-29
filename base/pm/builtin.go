package pm

//implement internal processes

import (
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
)

/*
Global command ProcessConstructor registery
*/
var CmdMap = map[string]process.ProcessFactory{
	process.CommandSystem: process.NewSystemProcess,
}

/*
NewProcess creates a new process from a command
*/
func GetProcessFactory(cmd *core.Command) process.ProcessFactory {
	return CmdMap[cmd.Command]
}

/*
RegisterCmd registers a new command (extension) so it can be executed via commands
*/
func RegisterCmd(cmd string, exe string, workdir string, cmdargs []string, env map[string]string) {
	CmdMap[cmd] = process.NewExtensionProcessFactory(exe, workdir, cmdargs, env)
}

/*
UnregisterCmd removes an extension from the global registery
*/
func UnregisterCmd(cmd string) {
	delete(CmdMap, cmd)
}
