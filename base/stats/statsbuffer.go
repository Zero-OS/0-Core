package stats

import (
	"github.com/g8os/core0/base/utils"
	"github.com/garyburd/redigo/redis"
	"github.com/op/go-logging"
	"time"
)

const (
	Counter    Operation = "A"
	Difference Operation = "D"
)

var (
	log = logging.MustGetLogger("stats")
)

type Operation string

/*
StatsBuffer implements a buffering and flushing mechanism to buffer statsd messages
that are collected via the process manager. Flush happens when buffer is full or a certain time passes since last flush.

The StatsBuffer.Handler should be registers as StatsFlushHandler on the process manager object.
*/
type StatsFlusher interface {
	Handler(operation Operation, key string, value float64, tags string)
}

type Stats struct {
	Operation Operation
	Key       string
	Value     float64
	Tags      string
}

type redisStatsBuffer struct {
	buffer utils.Buffer
	pool   *redis.Pool

	sha string
}

func NewRedisStatsBuffer(address string, password string, capacity int, flushInt time.Duration) StatsFlusher {
	pool := utils.NewRedisPool("tcp", address, password)

	redisBuffer := &redisStatsBuffer{
		pool: pool,
	}

	redisBuffer.buffer = utils.NewBuffer(capacity, flushInt, redisBuffer.onFlush)

	return redisBuffer
}

func (r *redisStatsBuffer) Handler(op Operation, key string, value float64, tags string) {
	r.buffer.Append(Stats{
		Operation: op,
		Key:       key,
		Value:     value,
		Tags:      tags,
	})
}

func (r *redisStatsBuffer) onFlush(stats []interface{}) {
	if len(stats) == 0 {
		return
	}

	db := r.pool.Get()
	defer db.Close()
	now := time.Now().Unix()

	for _, s := range stats {
		stat := s.(Stats)
		if err := db.Send("EVALSHA", r.sha, 1, stat.Key, stat.Value, now, stat.Operation, stat.Tags, ""); err != nil {
			log.Errorf("failed to report stats to redis: %s", err)
		}
	}
}
