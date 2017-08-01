package pm

import (
	"fmt"
	"sync"
)

type waitMachineImpl struct {
	states map[string]bool
	keys   map[string]*stateW

	m sync.Mutex
}

type stateW struct {
	ch chan struct{}
	s  bool
}

type stateMachine interface {
	Wait(key ...string) bool
	WaitAll()
	Release(ket string, state bool) error
}

func newStateMachine(keys ...string) stateMachine {
	s := &waitMachineImpl{
		keys: make(map[string]*stateW),
	}

	for _, key := range keys {
		s.keys[key] = &stateW{
			ch: make(chan struct{}),
		}
	}

	return s
}

func (s *waitMachineImpl) WaitAll() {
	for k := range s.keys {
		s.Wait(k)
	}
}

func (s *waitMachineImpl) Wait(keys ...string) bool {
	r := true
	for _, k := range keys {
		w, ok := s.keys[k]
		if !ok {
			continue
		}
		<-w.ch
		r = r && w.s
	}

	return r
}

func (s *waitMachineImpl) Release(key string, state bool) error {
	w, ok := s.keys[key]
	if !ok {
		return fmt.Errorf("key not found")
	}

	w.s = state
	defer func() {
		//recover from closing a closed channel
		recover()
	}()
	close(w.ch)

	return nil
}
