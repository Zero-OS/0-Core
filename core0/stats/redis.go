package stats

import (
	"github.com/zero-os/0-core/base/utils"
	"github.com/zero-os/0-core/core0/assets"
	"github.com/garyburd/redigo/redis"
	"github.com/op/go-logging"
	"strings"
	"time"
)



var (
	log = logging.MustGetLogger("stats")
)



/*
StatsBuffer implements a buffering and flushing mechanism to buffer statsd messages
that are collected via the process manager. Flush happens when buffer is full or a certain time passes since last flush.

The StatsBuffer.Handler should be registers as StatsFlushHandler on the process manager object.
*/
type Aggregator interface {
	Aggregate(operation string, key string, value float64, tags string)
}

type Stats struct {
	Operation Operation `json:"operation"`
	Key       string    `json:"key"`
	Value     float64   `json:"value"`
	Tags      string    `json:"tags"`
}

type redisStatsBuffer struct {
	buffer utils.Buffer
	pool   *redis.Pool

	sha string
}

func NewRedisStatsAggregator(address string, password string, capacity int, flushInt time.Duration) (Aggregator, error) {
	pool := utils.NewRedisPool("tcp", address, password)

	redisBuffer := &redisStatsBuffer{
		pool: pool,
	}

	redisBuffer.buffer = utils.NewBuffer(capacity, flushInt, redisBuffer.onFlush)

	if err := redisBuffer.init(); err != nil {
		return nil, err
	}

	return redisBuffer, nil
}

func (r *redisStatsBuffer) init() error {
	data, err := assets.Asset("scripts/stat.lua")
	if err != nil {
		return err
	}

	db := r.pool.Get()
	defer db.Close()

	sha, err := redis.String(db.Do("SCRIPT", "LOAD", string(data)))
	if err != nil {
		return err
	}

	r.sha = sha
	return nil
}

func (r *redisStatsBuffer) Aggregate(op string, key string, value float64, tags string) {
	log.Debugf("STATS: %s(%s, %f, '%s')", op, key, value, tags)

	r.buffer.Append(Stats{
		Operation: Operation(strings.ToUpper(op)),
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
