package plugin

import "github.com/g8os/core0/base/pm/process"

const (
	Version_1 Version = "1.0"
	Current   Version = Version_1
)

type Version string

type Manifest struct {
	//Domain to prefix command names
	Domain string

	//Plugin interface version
	Version Version
}

type Commands map[string]process.Runnable
