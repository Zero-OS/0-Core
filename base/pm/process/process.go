package process

import (
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	"github.com/op/go-logging"
	"syscall"
)

const (
	CommandSystem = "core.system"
)

var (
	log = logging.MustGetLogger("process")
)

type GetPID func() (int, error)

type PIDTable interface {
	//Register atomic registration of PID. MUST grantee that that no wait4 will happen
	//on any of the child process until the register operation is done.
	Register(g GetPID) error
	WaitPID(pid int) syscall.WaitStatus
}

//ProcessStats holds process cpu and memory usage
type ProcessStats struct {
	CPU   float64 `json:"cpu"`
	RSS   uint64  `json:"rss"`
	VMS   uint64  `json:"vms"`
	Swap  uint64  `json:"swap"`
	Debug string  `json:"debug,ommitempty"`
}

//Process interface
type Process interface {
	Command() *core.Command
	Run() (<-chan *stream.Message, error)
	Kill() error //==Signal(SIGTERM)
	Signal(sig syscall.Signal) error
}

type Stater interface {
	Stats() *ProcessStats
}

type ProcessFactory func(PIDTable, *core.Command) Process
