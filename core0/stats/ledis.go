package stats

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/op/go-logging"
	"github.com/patrickmn/go-cache"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/zero-os/0-core/base/pm"
	"sort"
	"strings"
	"time"
)

const (
	StatisticsQueueKey = "statistics:%d"
	StateKey           = "state:%s:%s"
	KeyIdSep           = ":"
	IDTag              = "id"
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

type Tags []pm.Tag

func (t Tags) Len() int {
	return len(t)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (t Tags) Less(i, j int) bool {
	return t[i].Key < t[j].Key
}

// Swap swaps the elements with indexes i and j.
func (t Tags) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

type Stats struct {
	Operation Operation `json:"operation"`
	Key       string    `json:"key"`
	Value     float64   `json:"value"`
	Tags      string    `json:"tags"`
}

type redisStatsBuffer struct {
	db    *ledis.DB
	cache *cache.Cache
}

func NewLedisStatsAggregator(db *ledis.DB) pm.StatsHandler {
	redisBuffer := &redisStatsBuffer{
		db:    db,
		cache: cache.New(1*time.Hour, 5*time.Minute),
	}

	redisBuffer.cache.OnEvicted(func(key string, _ interface{}) {
		if _, err := db.Del([]byte(key)); err != nil {
			log.Errorf("failed to evict stats key %s", key)
		}
	})

	pm.RegisterBuiltIn("aggregator.query", redisBuffer.query)

	return redisBuffer
}

type Point struct {
	*Sample
	Key  string            `json:"key"`
	Tags map[string]string `json:"tags,omitempty"`
}

func (r *redisStatsBuffer) query(cmd *pm.Command) (interface{}, error) {
	var filter struct {
		Key  string            `json:"key"`
		Tags map[string]string `json:"tags"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &filter); err != nil {
		return nil, err
	}

	result := make(map[string]*State)

	for key := range r.cache.Items() {
		parts := strings.SplitN(key, KeyIdSep, 3) //formated as `StateKey`
		metric := parts[1]
		if len(filter.Key) != 0 {
			if filter.Key != metric {
				continue
			}
		}

		data, err := r.db.Get([]byte(key))
		if err != nil {
			log.Errorf("failed to get state for metric: %s", key)
			continue
		}

		state, err := LoadState(data)
		if err != nil {
			log.Errorf("failed to load stat for %s", key)
		}

		//filter on tags
		m := true
		for k, v := range filter.Tags {
			m = false
			for _, t := range state.Tags {
				if t.Key == k && t.Value == v {
					m = true
					break
				}
			}
			if !m {
				break
			}
		}

		if !m {
			continue
		}

		//get ID if set
		for _, t := range state.Tags {
			if t.Key == IDTag {
				metric = fmt.Sprintf("%s/%s", metric, t.Value)
				break
			}
		}

		result[metric] = state
	}

	return result, nil
}

func (r *redisStatsBuffer) hash(tags []pm.Tag) string {
	sort.Sort(Tags(tags))
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v", tags))))
}

func (r *redisStatsBuffer) Stats(op string, key string, value float64, id string, tags ...pm.Tag) {
	if len(id) != 0 {
		tags = append(tags, pm.Tag{IDTag, id})
	}

	hash := r.hash(tags)
	internal := fmt.Sprintf(StateKey, key, hash)

	//touch key in cache so we know we are tracking this key
	r.cache.Set(internal, nil, cache.DefaultExpiration)

	data, err := r.db.Get([]byte(internal))
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

	if err := r.db.Set([]byte(internal), data); err != nil {
		log.Errorf("failed to save state object for %s: %s", key, err)
	}
}
