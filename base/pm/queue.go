package pm

import (
	"container/list"
	"github.com/zero-os/0-core/base/pm/core"
	"sync"
)

/**
cmdQueueManager is used for sequential cmds exectuions
*/
type cmdQueueManager struct {
	queues   map[string]*list.List
	signal   chan string
	consumer chan *core.Command
	producer chan *core.Command
	lock     sync.Mutex
}

/*
NewCmdQueueManager creates a new instace of the queue manager. Normally only the PM should call this.
*/
func newCmdQueueManager() *cmdQueueManager {
	mgr := &cmdQueueManager{
		queues:   make(map[string]*list.List),
		signal:   make(chan string),
		consumer: make(chan *core.Command),
		producer: make(chan *core.Command),
	}

	//start queue dispatcher
	go mgr.dispatcherLoop()

	//start the producer
	go mgr.producerLoop()

	return mgr
}

func (mgr *cmdQueueManager) dispatcherLoop() {
	for {
		cmd := <-mgr.consumer

		if cmd.Queue == "" {
			//not a queued command just log a warning and continue
			log.Warningf("Queue manager received a command with no set queue (%s)", cmd)
			continue
		}

		log.Infof("Pushing command to queue '%s'", cmd.Queue)
		//push command to correct queue
		mgr.lock.Lock()
		queue, ok := mgr.queues[cmd.Queue]
		if !ok {
			log.Debugf("Queue '%s' doesn't exist, initializing...", cmd.Queue)
			queue = list.New()
			mgr.queues[cmd.Queue] = queue
		}
		mgr.lock.Unlock()
		//push the command to the queue.
		queue.PushBack(cmd)

		if !ok {
			//since we just create this queue. We signal that it's ready
			//to produce the first queued command. Think of it as intial start
			//condition. When this command exists, it will auto signal the next command
			//and so on.
			mgr.signal <- cmd.Queue
		}
	}
}

//producer gets the next job to run and send it over channel
func (mgr *cmdQueueManager) producerLoop() {
	for {
		queueName := <-mgr.signal

		log.Debugf("Queue '%s' ready...", queueName)
		mgr.lock.Lock()
		queue, ok := mgr.queues[queueName]
		if !ok {
			//Proceed on a queue that doesn't exist anymore.
			mgr.lock.Unlock()
			continue
		}

		if queue.Len() == 0 {
			//last command on this queue exited successfully.
			//we can safely delete it.
			log.Infof("Cleaning up  queue '%s'", queueName)
			delete(mgr.queues, queueName)
			mgr.lock.Unlock()
			continue
		}

		mgr.lock.Unlock()

		next := queue.Remove(queue.Front()).(*core.Command)
		mgr.producer <- next
	}
}

func (mgr *cmdQueueManager) Push(cmd *core.Command) {
	mgr.consumer <- cmd
}

func (mgr *cmdQueueManager) Notify(cmd *core.Command) {
	if cmd.Queue == "" {
		//nothing to do
		return
	}

	mgr.signal <- cmd.Queue
}

func (mgr *cmdQueueManager) Producer() <-chan *core.Command {
	return mgr.producer
}
