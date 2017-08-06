package pm

import (
	"github.com/zero-os/0-core/base/pm/stream"
	"syscall"
)

const (
	CommandSystem = "core.system"
)

type GetPID func() (int, error)

type PIDTable interface {
	//PIDTable atomic registration of PID. MUST grantee that that no wait4 will happen
	//on any of the child p until the register operation is done.
	RegisterPID(g GetPID) error
	WaitPID(pid int) syscall.WaitStatus
}

//ProcessStats holds p cpu and memory usage
type ProcessStats struct {
	CPU   float64 `json:"cpu"`
	RSS   uint64  `json:"rss"`
	VMS   uint64  `json:"vms"`
	Swap  uint64  `json:"swap"`
	Debug string  `json:"debug,ommitempty"`
}

//Process interface
type Process interface {
	Command() *Command
	Run() (<-chan *stream.Message, error)
}

type Signaler interface {
	Process
	Signal(sig syscall.Signal) error
}

type Stater interface {
	Process
	Stats() *ProcessStats
}

type ProcessFactory func(PIDTable, *Command) Process
