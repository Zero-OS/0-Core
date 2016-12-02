package logger

import (
	"encoding/json"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	"github.com/g8os/core0/base/utils"
	"github.com/garyburd/redigo/redis"
	"strings"
	"time"
)

const (
	RedisLoggerQueue = "core.logs"
	defaultBatchSize = 100000
)

// redisLogger send log to redis queue
type redisLogger struct {
	coreID    uint64
	pool      *redis.Pool
	defaults  []int
	batchSize int
}

// NewRedisLogger creates new redis logger handler
func NewRedisLogger(coreID uint64, address string, password string, defaults []int, batchSize int) Logger {
	if batchSize == 0 {
		batchSize = defaultBatchSize
	}
	network := "unix"
	if strings.Index(address, ":") > 0 {
		network = "tcp"
	}

	rl := &redisLogger{
		coreID:    coreID,
		pool:      utils.NewRedisPool(network, address, password),
		defaults:  defaults,
		batchSize: batchSize,
	}

	return rl
}

func (l *redisLogger) Log(cmd *core.Command, msg *stream.Message) {
	if len(l.defaults) > 0 && !utils.In(l.defaults, msg.Level) {
		return
	}
	data := map[string]interface{}{
		"core":    l.coreID,
		"command": *cmd,
		"message": stream.Message{
			// need to copy this first because we don't want to
			// modify the epoch value of original `msg`
			Epoch:   msg.Epoch / int64(time.Millisecond),
			Message: msg.Message,
			Level:   msg.Level,
		},
	}

	bytes, err := json.Marshal(data)
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

	if err := db.Send("LTRIM", RedisLoggerQueue, -1*l.batchSize, -1); err != nil {
		log.Errorf("Failed to truncate log message to `%v` err: `%v`", l.batchSize, err)
	}
}
