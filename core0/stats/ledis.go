package stats

import (
	"encoding/json"
	"fmt"
	"github.com/op/go-logging"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
)

const (
	StatisticsQueueKey = "statistics:%d"
	StateKey           = "state:%s:%s"
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
	Aggregate(operation string, key string, value float64, id string, tags ...pm.Tag)
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

	pm.CmdMap["aggregator.query"] = process.NewInternalProcessFactory(redisBuffer.query)

	return redisBuffer
}

type Point struct {
	*Sample
	Key  string            `json:"key"`
	Tags map[string]string `json:"tags,omitempty"`
}

func (r *redisStatsBuffer) query(cmd *core.Command) (interface{}, error) {
	var keys [][]string
	if err := json.Unmarshal(*cmd.Arguments, &keys); err != nil {
		return nil, err
	}

	for _, metric := range keys {
		key := metric[0]
		var id string
		if len(metric) > 1 {
			id = metric[1]
		}

		ledisKey := fmt.Sprintf(StateKey, key, id)
		//TODO: use key to get stats.
		//i think if the value does not exist, just put null in the return

	}

	return nil, nil
}

func (r *redisStatsBuffer) Aggregate(op string, key string, value float64, id string, tags ...pm.Tag) {
	log.Debugf("STATS: %s(%s/%s, %f, '%s')", op, key, id, value, tags)
	lkey := fmt.Sprintf(StateKey, key, id)
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

	if len(tags) != 0 {
		state.Tags = tags
	}

	for period, sample := range state.Feed(value) {
		if sample.Start == 0 {
			//undefined sample
			continue
		}

		queue := fmt.Sprintf(StatisticsQueueKey, period)
		p := Point{
			Sample: sample,
			Key:    key,
			Tags:   make(map[string]string),
		}

		for _, tag := range state.Tags {
			p.Tags[tag.Key] = tag.Value
		}

		if id != "" {
			p.Tags["id"] = id
		}

		if data, err := json.Marshal(&p); err == nil {
			r.db.RPush([]byte(queue), data)
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
