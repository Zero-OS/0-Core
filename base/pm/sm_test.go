package pm

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

//StateMachine
func TestNewStateMachine(t *testing.T) {
	state := NewStateMachine()
	if !assert.Implements(t, (*StateMachine)(nil), state) {
		t.Fatal()
	}
}

func Wait(state StateMachine, timeout int, keys ...string) (bool, error) {
	ch := make(chan bool)
	go func() {
		ch <- state.Wait(keys...)
	}()

	defer close(ch)

	var s bool
	select {
	case s = <-ch:
	case <-time.After(time.Duration(timeout) * time.Second):
		return false, fmt.Errorf("Timed out waiting from wait to unlock")
	}

	return s, nil
}

func Test_NonBlockingWait(t *testing.T) {
	state := NewStateMachine()

	s, err := Wait(state, 2)
	if err != nil {
		t.Fatal(err)
	}

	if !assert.True(t, s) {
		t.Fatal()
	}
}

func Test_BlockingWait_SatisfiedConditions_True(t *testing.T) {
	state := NewStateMachine("init")

	state.Release("init", true)

	s, err := Wait(state, 2, "init")
	if err != nil {
		t.Fatal(err)
	}

	if !assert.True(t, s) {
		t.Fatal()
	}
}

func Test_BlockingWait_SatisfiedConditions_False(t *testing.T) {
	state := NewStateMachine("init")

	state.Release("init", false)

	s, err := Wait(state, 2, "init")
	if err != nil {
		t.Fatal(err)
	}

	if !assert.False(t, s) {
		t.Fatal()
	}
}

func Test_BlockingWait_UNSatisfiedConditions(t *testing.T) {
	state := NewStateMachine("init")

	//waiting for a condition that didn't (and never) happen
	s, err := Wait(state, 2, "init")
	if !assert.Error(t, err) {
		t.Fail()
	}

	if !assert.False(t, s) {
		t.Fail()
	}
}

func Test_BlockingWait_DelayedSatisfiedConditions(t *testing.T) {
	state := NewStateMachine("init")

	//waiting for a condition that didn't (and never) happen
	go func() {
		time.Sleep(1 * time.Second)
		state.Release("init", true)
	}()

	s, err := Wait(state, 3, "init")
	if !assert.Nil(t, err) {
		t.Fail()
	}

	if !assert.True(t, s) {
		t.Fail()
	}
}

func Test_BlockingWait_DelayedSatisfiedConditions_Multiple(t *testing.T) {
	state := NewStateMachine("init", "net")

	//waiting for a condition that didn't (and never) happen
	go func() {
		time.Sleep(1 * time.Second)
		state.Release("init", true)
		time.Sleep(100 * time.Millisecond)
		state.Release("net", true)
	}()

	s, err := Wait(state, 3, "init", "net")
	if !assert.Nil(t, err) {
		t.Fail()
	}

	if !assert.True(t, s) {
		t.Fail()
	}
}

func Test_BlockingWait_ComplexDependency(t *testing.T) {

	state := NewStateMachine("x", "a", "b", "c", "d")
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		state.Wait("x")
		state.Release("a", true)
		wg.Done()
	}()

	go func() {
		state.Wait("x")
		state.Release("b", true)
		wg.Done()
	}()

	go func() {
		state.Wait("x", "b")
		state.Release("d", true)
		wg.Done()
	}()

	go func() {
		state.Wait("a", "b")
		state.Release("c", true)
		wg.Done()
	}()

	//start the chain reaction
	state.Release("x", true)

	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatal("Timedout")
	}
}
