package logger

import (
	"encoding/json"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/siddontang/ledisdb/ledis"
)

const (
	RedisLoggerQueue  = "core.logs"
	MaxRedisQueueSize = 100000
)

// redisLogger send log to redis queue
type redisLogger struct {
	coreID   uint16
	db       *ledis.DB
	defaults []int
	size     int64

	ch chan *LogRecord
}

// NewRedisLogger creates new redis logger handler
func NewLedisLogger(coreID uint16, db *ledis.DB, defaults []int, size int64) Logger {
	if size == 0 {
		size = MaxRedisQueueSize
	}

	rl := &redisLogger{
		coreID:   coreID,
		db:       db,
		defaults: defaults,
		size:     size,
		ch:       make(chan *LogRecord, MaxRedisQueueSize),
	}

	go rl.pusher()
	return rl
}

func (l *redisLogger) Log(cmd *core.Command, msg *stream.Message) {
	if !IsLoggableCmd(cmd, msg) {
		return
	}

	l.LogRecord(&LogRecord{
		Core:    l.coreID,
		Command: cmd.ID,
		Message: msg,
	})
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

func (l *redisLogger) push() error {
	for {
		record := <-l.ch

		bytes, err := json.Marshal(record)
		if err != nil {
			continue
		}

		if _, err := l.db.RPush([]byte(RedisLoggerQueue), bytes); err != nil {
			return err
		}

		if err := l.db.LTrim([]byte(RedisLoggerQueue), -1*l.size, -1); err != nil {
			return err
		}
	}
}
