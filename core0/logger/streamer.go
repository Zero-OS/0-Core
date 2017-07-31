package logger

import (
	"encoding/json"
	"fmt"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/zero-os/0-core/base/pm/stream"
)

const (
	MaxStreamRedisQueueSize = 1000
	StreamRedisQueueTTL     = 60
)

// redisLogger send Message to redis queue
type streamLogger struct {
	db   *ledis.DB
	size int64

	ch chan *LogRecord
}

// NewRedisLogger creates new redis logger handler
func NewStreamLogger(db *ledis.DB, size int64) Logger {
	if size == 0 {
		size = MaxStreamRedisQueueSize
	}

	rl := &streamLogger{
		db:   db,
		size: size,
		ch:   make(chan *LogRecord, MaxStreamRedisQueueSize),
	}

	go rl.pusher()
	return rl
}

func (l *streamLogger) LogRecord(record *LogRecord) {
	if !record.Message.Meta.Is(stream.StreamFlag) {
		//stream flag is not set
		return
	}

	l.ch <- record
}

func (l *streamLogger) pusher() {
	for {
		if err := l.push(); err != nil {
			//we don't sleep to avoid blocking the logging channel and to not slow down processes.
		}
	}
}

func (l *streamLogger) push() error {
	for {
		record := <-l.ch
		bytes, err := json.Marshal(record)
		if err != nil {
			continue
		}

		queue := fmt.Sprintf("stream:%s", record.Command)
		if _, err := l.db.RPush([]byte(queue), bytes); err != nil {
			return err
		}

		if err := l.db.LTrim([]byte(queue), -1*l.size, -1); err != nil {
			return err
		}

		l.db.LExpire([]byte(queue), StreamRedisQueueTTL)
	}
}
