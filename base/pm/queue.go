package pm

import (
	"container/list"
	"sync"
)

/**
Queue is used for sequential cmds exectuions
*/
type Queue struct {
	queues map[string]*list.List
	ch     chan Job
	lock   sync.RWMutex
	o      sync.Once
}

func (q *Queue) Start() <-chan Job {
	q.o.Do(func() {
		q.queues = make(map[string]*list.List)
		q.ch = make(chan Job)
	})

	return q.ch
}

func (q *Queue) Push(job Job) {
	q.lock.Lock()
	defer q.lock.Unlock()

	name := job.Command().Queue
	if name == "" {
		q.ch <- job
		return
	}

	queue, ok := q.queues[name]
	if !ok {
		queue = list.New()
		q.queues[name] = queue
	}

	log.Debugf("pushing job %v to queue: %s", job.Command(), name)
	queue.PushBack(job)
	if queue.Len() == 1 {
		//first job in the queue
		q.ch <- job
	}
}

func (q *Queue) Notify(job Job) {
	q.lock.Lock()
	defer q.lock.Unlock()
	name := job.Command().Queue
	queue, ok := q.queues[name]
	if !ok {
		return
	}
	queue.Remove(queue.Front())
	if queue.Len() == 0 {
		delete(q.queues, name)
		return
	}

	next := queue.Front().Value.(Job)
	q.ch <- next
}
