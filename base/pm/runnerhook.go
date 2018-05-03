package pm

import (
	"bytes"
	"regexp"
	"sync"
	"time"

	"github.com/zero-os/0-core/base/pm/stream"
)

//RunnerHook is a per process event handler
type RunnerHook interface {
	//Tick is called if certain amount has passed sicne process starting
	Tick(delay time.Duration)
	//Message is called on each received message from the process
	Message(msg *stream.Message)
	//Exit is called on process exit
	Exit(state JobState)
	//PID is executed once the child process is started and got an PID
	PID(pid int)
}

//NOOPHook empty handler
type NOOPHook struct{}

func (h *NOOPHook) Tick(delay time.Duration)    {}
func (h *NOOPHook) Message(msg *stream.Message) {}
func (h *NOOPHook) Exit(state JobState)         {}
func (h *NOOPHook) PID(pid int)                 {}

//DelayHook called after a certain amount of time passes from process start
type DelayHook struct {
	NOOPHook
	o sync.Once

	Delay  time.Duration
	Action func()
}

func (h *DelayHook) Tick(delay time.Duration) {
	if delay > h.Delay {
		h.o.Do(h.Action)
	}
}

//ExitHook is called when the process exits
type ExitHook struct {
	NOOPHook
	o sync.Once

	Action func(bool)
}

func (h *ExitHook) Exit(state JobState) {
	s := false
	if state == StateSuccess {
		s = true
	}

	h.o.Do(func() {
		h.Action(s)
	})
}

//PIDHook is called if a process got a PID
type PIDHook struct {
	NOOPHook
	o sync.Once

	Action func(pid int)
}

func (h *PIDHook) PID(pid int) {
	h.o.Do(func() {
		h.Action(pid)
	})
}

//MatchHook is called if a message matches a given pattern
type MatchHook struct {
	NOOPHook
	Match  string
	Action func(msg *stream.Message)

	io sync.Once
	p  *regexp.Regexp
	o  sync.Once
}

func (h *MatchHook) Message(msg *stream.Message) {
	h.io.Do(func() {
		p, e := regexp.CompilePOSIX(h.Match)
		if e != nil {
			log.Errorf("Failed to compile regexp pattern '%s'", h.Match)
			return
		}
		h.p = p
	})

	if h.p == nil {
		return
	}

	if h.p.MatchString(msg.Message) {
		h.o.Do(func() {
			h.Action(msg)
			h.p = nil
		})
	}
}

//StreamHook captures full stdout and stderr of a process.
type StreamHook struct {
	NOOPHook
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

func (h *StreamHook) append(buf *bytes.Buffer, msg *stream.Message) {
	if buf.Len() > 0 {
		buf.WriteByte('\n')
	}
	buf.WriteString(msg.Message)
}

func (h *StreamHook) Message(msg *stream.Message) {
	if msg.Meta.Level() == stream.LevelStdout {
		h.append(&h.Stdout, msg)
	} else if msg.Meta.Level() == stream.LevelStderr {
		h.append(&h.Stderr, msg)
	}

	//ignore otherwise.
}
