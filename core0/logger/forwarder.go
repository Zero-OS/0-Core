package logger

import (
	"encoding/json"
	"github.com/g8os/core0/base/logger"
	"github.com/garyburd/redigo/redis"
	"time"
)

// Start logs forwarder
func StartForwarder() {
	go func() {
		for {
			if err := forward(); err != nil {
				log.Errorf("failed to forwar logs: %v", err)
			}

			time.Sleep(2 * time.Second)
		}
	}()
}

func forward() error {
	// setup connections
	privConn, err := redis.Dial("unix", "/var/run/redis.socket")
	if err != nil {
		return err
	}

	defer privConn.Close()

	// forwad the logs
	for {
		// take from private redis
		b, err := redis.Bytes(privConn.Do("BLPOP", logger.RedisLoggerQueue, 0))
		if err != nil {
			return err
		}

		var record logger.LogRecord
		if err := json.Unmarshal(b, &record); err != nil {
			return err
		}

		loggers.LogRecord(&record)
	}

	return nil

}
