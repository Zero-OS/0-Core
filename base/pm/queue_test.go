package pm

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestQueue_Start(t *testing.T) {
	var q Queue
	ch := q.Start()

	if ok := assert.NotNil(t, ch); !ok {
		t.Fatal()
	}
}

func TestQueue_Push(t *testing.T) {
	var q Queue
	ch := q.Start()

	lock := make(chan int)
	failed := true
	go func() {
		select {
		case <-ch:
			failed = false
		case <-time.After(1 * time.Second):
		}
		lock <- 0
	}()

	q.Push(&jobImb{command: &Command{}})
	<-lock

	if ok := assert.False(t, failed); !ok {
		t.Fatal()
	}
}

func TestQueue_PushQueued(t *testing.T) {
	var q Queue
	ch := q.Start()

	lock := make(chan int)

	failed := true
	go func() {
		select {
		case <-ch:
			failed = false
		case <-time.After(1 * time.Second):
		}
		lock <- 0
	}()

	q.Push(&jobImb{command: &Command{
		Queue: "test",
	}})

	q.Push(&jobImb{command: &Command{
		Queue: "test",
	}})

	<-lock
	if ok := assert.False(t, failed); !ok {
		t.Fatal()
	}

	failed = true
	go func() {
		select {
		case <-ch:
			failed = false
		case <-time.After(1 * time.Second):
		}
		lock <- 0
	}()

	<-lock
	//command not available because first one was never notified as finished
	if ok := assert.True(t, failed); !ok {
		t.Fatal()
	}

	if ok := assert.NotNil(t, q.queues); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, 2, q.queues["test"].Len()); !ok {
		t.Fatal()
	}

	failed = true
	go func() {
		select {
		case <-ch:
			failed = false
		case <-time.After(1 * time.Second):
		}
		lock <- 0
	}()

	q.Notify(&jobImb{command: &Command{
		Queue: "test",
	}})

	<-lock

	if ok := assert.Equal(t, 1, q.queues["test"].Len()); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, failed); !ok {
		t.Fatal()
	}

}
