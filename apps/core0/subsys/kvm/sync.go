package kvm

import (
	"fmt"
	"sync"
	"time"
)

//NewSync create a new sync type
func NewSync() *Sync {
	return &Sync{
		m: make(map[syncKey]chan struct{}),
	}
}

//Sync helper type to process device remove events.
type Sync struct {
	m map[syncKey]chan struct{}
	s sync.Mutex
}

type syncKey [2]string

//Release a waiting routine if exists
func (s *Sync) Release(uuid, alias string) {
	s.s.Lock()
	defer s.s.Unlock()

	ch, ok := s.m[syncKey{uuid, alias}]
	if !ok {
		return
	}

	select {
	case ch <- struct{}{}:
	default:
	}
}

//Expect notify the sync type that we expect an event for deleting this device
func (s *Sync) Expect(uuid string, alias string) {
	s.s.Lock()
	defer s.s.Unlock()

	s.m[syncKey{uuid, alias}] = make(chan struct{}, 1)
}

//Unexpect forgets about those keys
func (s *Sync) Unexpect(uuid, alias string) {
	s.s.Lock()
	defer s.s.Unlock()
	ch, ok := s.m[syncKey{uuid, alias}]
	if !ok {
		return
	}

	close(ch)
	delete(s.m, syncKey{uuid, alias})
}

//Wait waits for the event to arrive
func (s *Sync) Wait(uuid, alias string, timeout time.Duration) error {
	s.s.Lock()
	ch, ok := s.m[syncKey{uuid, alias}]
	if !ok {
		return fmt.Errorf("was not expecting this event")
	}
	s.s.Unlock()

	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout")
	}
}
