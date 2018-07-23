// +build production

package transport

import "github.com/zero-os/0-core/base/pm"

/*
Blocked commands that are not only accepted via the transport

We can't disable those command internally because they are used intensevly
by the core0 itself to run utilities and other processes. (also configuration files)
*/
var blockedCommands = []string{
	"core.system", "bash",
}

func (sink *Sink) allowedCommnad(cmd *pm.Command) bool {
	for _, blocked := range blockedCommands {
		if cmd.Command == blocked {
			return false
		}
	}

	return true
}
