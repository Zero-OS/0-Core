package logger

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/core0/transport"
	"sync"
)

const (
	MaxRedisQueueSize = 1000
)

// redisLogger send Message to redis queue
type redisLogger struct {
	sink     *transport.Sink
	defaults []uint16
	size     int64
	buffer   *stream.Buffer
	queues   map[string]struct{}
	m        sync.RWMutex

	ch chan *LogRecord
}

// NewRedisLogger creates new redis logger handler
func NewLedisLogger(sink *transport.Sink, defaults []uint16, size int64) Logger {
	if size == 0 {
		size = MaxRedisQueueSize
	}

	rl := &redisLogger{
		sink:     sink,
		defaults: defaults,
		size:     size,
		buffer:   stream.NewBuffer(MaxStreamRedisQueueSize),
		queues:   make(map[string]struct{}),
		ch:       make(chan *LogRecord, MaxRedisQueueSize),
	}

	pm.RegisterBuiltIn("logger.subscribe", rl.subscribe)
	pm.RegisterBuiltIn("logger.unsubscribe", rl.unSubscribe)

	go rl.pusher()
	return rl
}

func (l *redisLogger) LogRecord(record *LogRecord) {
	if !IsLoggable(l.defaults, record.Message) {
		return
	}
	l.ch <- record
}

func (l *redisLogger) pusher() {
	for {
		if err := l.push(); err != nil {
			//we don't sleep to avoid blocking the logging channel and to not slow down processes.
		}
	}
}

func (l *redisLogger) unSubscribe(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Queue string `json:"queue"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if len(args.Queue) == 0 {
		return nil, fmt.Errorf("queue is required")
	}

	l.m.Lock()
	defer l.m.Unlock()
	delete(l.queues, args.Queue)

	return nil, nil
}

func (l *redisLogger) subscribe(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Queue string `json:"queue"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if len(args.Queue) == 0 {
		args.Queue = uuid.New()
	}

	args.Queue = fmt.Sprintf("logger:%s", args.Queue)

	go func() {
		//copying a 100,000 records can take too much,
		//so we run this in go routine, so the caller
		//can start reading logs immediately and he doesn't have
		//to wait until all logs are copied.
		if err := l.Subscribe(args.Queue); err != nil {
			log.Errorf("failed to subscribe to queue: %s", err)
		}
	}()

	return args.Queue, nil
}

func (l *redisLogger) Subscribe(queue string) error {
	l.m.Lock()
	defer l.m.Unlock()
	if _, ok := l.queues[queue]; ok {
		return nil
	}

	l.queues[queue] = struct{}{}

	//flush backlog
	for v := l.buffer.Front(); v != nil; v = v.Next() {
		bytes, err := json.Marshal(v.Value)
		if err != nil {
			continue
		}

		if _, err := l.sink.RPush([]byte(queue), bytes); err != nil {
			return err
		}
	}

	if err := l.sink.LTrim([]byte(queue), -1*l.size, -1); err != nil {
		return err
	}

	return nil
}

func (l *redisLogger) pushQueues(record *LogRecord) error {
	l.m.RLock()
	defer l.m.RUnlock()
	l.buffer.Append(record)
	if len(l.queues) == 0 {
		return nil
	}

	bytes, err := json.Marshal(record)
	if err != nil {
		return err
	}

	for queue := range l.queues {
		if _, err := l.sink.RPush([]byte(queue), bytes); err != nil {
			return err
		}

		if err := l.sink.LTrim([]byte(queue), -1*l.size, -1); err != nil {
			return err
		}
	}

	return nil
}

func (l *redisLogger) push() error {
	for {
		record := <-l.ch
		if err := l.pushQueues(record); err != nil {
			log.Errorf("failed to push logs to queue: %s", err)
		}
	}
}
