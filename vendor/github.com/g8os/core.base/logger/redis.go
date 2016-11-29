package logger

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/pm/stream"
	"github.com/g8os/core.base/utils"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	redisLoggerQueue = "core.logs"
	defaultBatchSize = 100000
)

type redisLogger struct {
	coreID    uint64
	pool      *redis.Pool
	defaults  []int
	batchSize int
}

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
	if rl.coreID == 0 {
		go rl.aggregates()
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

	if err := db.Send("RPUSH", redisLoggerQueue, bytes); err != nil {
		log.Errorf("Failed to push log message to redis: %s", err)
	}

	if err := db.Send("LTRIM", redisLoggerQueue, -1*l.batchSize, -1); err != nil {
		log.Errorf("Failed to truncate log message to `%v` err: `%v`", l.batchSize, err)
	}
}

// move logs from all coreX to core0's redis
func (l *redisLogger) aggregates() {
	for {
		time.Sleep(10 * time.Second)
		socks, err := l.getCoreXRedisSockets()
		if err != nil {
			log.Error("failed to get redis sockets list:", err)
		}
		if err := l.doAggregates(socks); err != nil {
			log.Error("redis logger failed to do aggregation:", err)
		}
	}
}

// move logs from coreX's redis to core0's redis
// TODO:
// - get the logs from coreX in bulk
// - copy the logs to core0 in bulk
func (l *redisLogger) doAggregates(socks []string) error {
	_aggregate := func(sock string) error {
		c, err := redis.Dial("unix", sock)
		if err != nil {
			return err
		}
		defer c.Close()
		for {
			b, err := redis.Bytes(c.Do("LPOP", redisLoggerQueue))
			if err != nil {
				if err != redis.ErrNil {
					log.Errorf("failed to LPOP from `%v` err : `%v", sock, err)
				}
				break
			}
			l.sendLog(b)
		}
		return nil
	}
	for _, sock := range socks {
		if err := _aggregate(sock); err != nil {
			log.Error("redisLogger failed to do aggregation to :", sock)
		}
	}
	return nil
}

// get all available redis.socket from containers directory
func (l *redisLogger) getCoreXRedisSockets() ([]string, error) {
	var socks []string

	entries, err := ioutil.ReadDir("/mnt")
	if err != nil {
		return socks, err
	}

	for _, dir := range entries {
		var id uint64
		if !dir.IsDir() {
			continue
		}
		if _, err := fmt.Sscanf(dir.Name(), "container-%d", &id); err != nil {
			continue
		}
		sockName := filepath.Join("/mnt", dir.Name(), "redis.socket")
		if _, err := os.Stat(sockName); os.IsNotExist(err) {
			continue
		}
		socks = append(socks, sockName)
	}
	return socks, nil
}
