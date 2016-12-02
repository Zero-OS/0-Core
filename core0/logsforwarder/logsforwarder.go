package logsforwarder

import (
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/op/go-logging"

	"github.com/g8os/core0/base/logger"
)

const (
	defaultPeriod = 10 // default forwared period
	batchSize     = 100000
)

var (
	log = logging.MustGetLogger("main")
)

// Start logs forwarder
func Start(period int) {
	if period < 1 {
		period = defaultPeriod
	}

	go func() {
		for {
			time.Sleep(time.Duration(period) * time.Second)
			if err := forward(); err != nil {
				log.Errorf("[logsforwarder]failed to forwar logs:%v", err)
			}
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

	pubConn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		return err
	}
	defer pubConn.Close()

	// forwad the logs
	for {
		// take from private redis
		b, err := redis.Bytes(privConn.Do("LPOP", logger.RedisLoggerQueue))
		if err != nil {
			if err == redis.ErrNil {
				break
			}
			log.Errorf("[logsforwarder]failed to LPOP from private redis err : `%v", err)
		}

		// send to public redis
		if err := pubConn.Send("RPUSH", logger.RedisLoggerQueue, b); err != nil {
			log.Errorf("[logsforwarder] failed to forward logs:%v", err)
		}
		if err := pubConn.Send("LTRIM", logger.RedisLoggerQueue, -1*batchSize, -1); err != nil {
			log.Errorf("[logsforwarder] failed to truncate log message to `%v` err: `%v`", batchSize, err)
		}

	}
	return nil

}
