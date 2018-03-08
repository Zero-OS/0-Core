package transport

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

func newPool() *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("unix", "/var/run/redis.sock")
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxActive: 20,
		Wait:      true,
	}
}
