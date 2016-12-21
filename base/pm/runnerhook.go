package pm

import (
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	"regexp"
	"sync"
	"time"
)

type RunnerHook interface {
	Tick(delay time.Duration)
	Message(msg *stream.Message)
	Exit(state string)
	PID(pid int)
}

type NOOPHook struct {
}

func (h *NOOPHook) Tick(delay time.Duration)    {}
func (h *NOOPHook) Message(msg *stream.Message) {}
func (h *NOOPHook) Exit(state string)           {}
func (h *NOOPHook) PID(pid int)                 {}

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

type ExitHook struct {
	NOOPHook
	o sync.Once

	Action func(bool)
}

func (h *ExitHook) Exit(state string) {
	s := false
	if state == core.StateSuccess {
		s = true
	}

	h.o.Do(func() {
		h.Action(s)
	})
}

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
