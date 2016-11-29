package logger

import (
	"encoding/json"
	"github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/pm/stream"
	"github.com/g8os/core.base/utils"
	"github.com/garyburd/redigo/redis"
	"time"
)

const (
	redisLoggerQueue = "core.logs"
)

type redisLogger struct {
	coreNum   int
	pool      *redis.Pool
	defaults  []int
	batchSize int
}

func NewRedisLogger(coreNum int, address string, password string, defaults []int, batchSize int) Logger {
	return &redisLogger{
		coreNum:   coreNum,
		pool:      utils.NewRedisPool("tcp", address, password),
		defaults:  defaults,
		batchSize: batchSize,
	}
}

func (l *redisLogger) Log(cmd *core.Command, msg *stream.Message) {
	if len(l.defaults) > 0 && !utils.In(l.defaults, msg.Level) {
		return
	}

	db := l.pool.Get()
	defer db.Close()

	data := map[string]interface{}{
		"core":    l.coreNum,
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

	if err := db.Send("RPUSH", redisLoggerQueue, bytes); err != nil {
		log.Errorf("Failed to push log message to redis: %s", err)
	}

	if err := db.Send("LTRIM", redisLoggerQueue, -1*l.batchSize, -1); err != nil {
		log.Errorf("Failed to truncate log message to `%v` err: `%v`", l.batchSize, err)
	}
}
