package plugin

import "github.com/g8os/core0/base/pm/process"

type Plugin struct {
	Domain   string
	Commands map[string]process.Runnable
}
