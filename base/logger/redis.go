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
	}

	return rl
}

func (l *redisLogger) Log(cmd *core.Command, msg *stream.Message) {
	if len(l.defaults) > 0 && !utils.In(l.defaults, msg.Level) {
		return
	}

	l.LogRecord(&LogRecord{
		Core:    l.coreID,
		Command: cmd.ID,
		Message: msg,
	})
}

func (l *redisLogger) LogRecord(record *LogRecord) {
	bytes, err := json.Marshal(record)
	if err != nil {
		log.Errorf("Failed to serialize message for redis logger: %s", err)
		return
	}

	l.sendLog(bytes)
}

func (l *redisLogger) sendLog(bytes []byte) {
	db := l.pool.Get()
	defer db.Close()

	if err := db.Send("RPUSH", RedisLoggerQueue, bytes); err != nil {
		log.Errorf("Failed to push log message to redis: %s", err)
	}

	if err := db.Send("LTRIM", RedisLoggerQueue, -1*l.queueSize, -1); err != nil {
		log.Errorf("Failed to truncate log message to `%v` err: `%v`", l.queueSize, err)
	}
}
