package stats

import (
	"encoding/json"
	"fmt"
	"github.com/op/go-logging"
	"github.com/siddontang/ledisdb/ledis"
)

const (
	StatisticsQueueKey = "statistics:%d"
	StateKey           = "state:%s"
)

var (
	log     = logging.MustGetLogger("stats")
	Periods = []int64{300, 3600} //5 min, 1 hour
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
	db *ledis.DB
}

func NewLedisStatsAggregator(db *ledis.DB) Aggregator {
	redisBuffer := &redisStatsBuffer{
		db: db,
	}

	return redisBuffer
}

type Point struct {
	*Sample
	Key string
}

func (r *redisStatsBuffer) Aggregate(op string, key string, value float64, tags string) {
	log.Debugf("STATS: %s(%s, %f, '%s')", op, key, value, tags)
	lkey := fmt.Sprintf(StateKey, key)
	data, err := r.db.Get([]byte(lkey))
	if err != nil {
		log.Errorf("failed to get value for %s: %s", key, err)
		return
	}

	var state *State
	if data == nil {
		state = NewState(Operation(op), Periods...)
	} else if state, err = LoadState(data); err != nil {
		log.Errorf("failed to load state object for %s: %s", key, err)
		return
	}

	state.Tags = tags

	for period, sample := range state.Feed(value) {
		key := fmt.Sprintf(StatisticsQueueKey, period)
		p := Point{
			Sample: sample,
			Key:    key,
		}

		if data, err := json.Marshal(&p); err == nil {
			r.db.RPush([]byte(key), data)
		} else {
			log.Errorf("statistics point marshal error: %s", err)
		}
	}

	data, err = json.Marshal(state)
	if err != nil {
		log.Errorf("failed to marshal state object for %s: %s", key, err)
		return
	}

	if err := r.db.Set([]byte(lkey), data); err != nil {
		log.Errorf("failed to save state object for %s: %s", key, err)
	}
}
