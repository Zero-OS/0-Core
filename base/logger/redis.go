package logger

import (
	"encoding/json"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	"github.com/g8os/core0/base/utils"
	"github.com/garyburd/redigo/redis"
	"strings"
)

const (
	RedisLoggerQueue  = "core.logs"
	MaxRedisQueueSize = 100000
)

// redisLogger send log to redis queue
type redisLogger struct {
	coreID    uint16
	pool      *redis.Pool
	defaults  []int
	queueSize int

	ch chan *LogRecord
}

// NewRedisLogger creates new redis logger handler
func NewRedisLogger(coreID uint16, address string, password string, defaults []int, batchSize int) Logger {
	if batchSize == 0 {
		batchSize = MaxRedisQueueSize
	}
	network := "unix"
	if strings.Index(address, ":") > 0 {
		network = "tcp"
	}

	rl := &redisLogger{
		coreID:    coreID,
		pool:      utils.NewRedisPool(network, address, password),
		defaults:  defaults,
		queueSize: batchSize,
		ch:        make(chan *LogRecord, MaxRedisQueueSize),
	}

	go rl.pusher()
	return rl
}

func (l *redisLogger) Log(cmd *core.Command, msg *stream.Message) {
	if !IsLoggable(l.defaults, cmd, msg) {
		return
	}

	l.LogRecord(&LogRecord{
		Core:    l.coreID,
		Command: cmd.ID,
		Message: msg,
	})
}

func (l *redisLogger) LogRecord(record *LogRecord) {
	l.ch <- record
}

func (l *redisLogger) pusher() {
	for {
		if err := l.push(); err != nil {
			log.Errorf("redis log pusher error: %s", err)
			//we don't sleep to avoid blocking the logging channel and to not slow down processes.
		}
	}
}

func (l *redisLogger) push() error {
	db := l.pool.Get()
	defer db.Close()

	for {
		record := <-l.ch
		log.Debugf("received log message: %v", record)

		bytes, err := json.Marshal(record)
		if err != nil {
			log.Errorf("Failed to serialize message for redis logger: %s", err)
			continue
		}

		if _, err := db.Do("RPUSH", RedisLoggerQueue, bytes); err != nil {
			return err
		}

		if _, err := db.Do("LTRIM", RedisLoggerQueue, -1*l.queueSize, -1); err != nil {
			return err
		}
	}
}
