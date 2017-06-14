package client

import (
	"encoding/json"
	"fmt"
	"time"

	"crypto/tls"
	"github.com/garyburd/redigo/redis"
	"github.com/pborman/uuid"
	"net"
)

const (
	CommandsQueue = "core:default"

	ResultNoTimeout      = 0
	ResultDefaultTimeout = 10
)

type redisClient struct {
	pool *redis.Pool

	info InfoManager
}

func NewClientWithPool(pool *redis.Pool) Client {
	cl := &redisClient{pool: pool}

	cl.info = &infoMgr{cl}
	return cl
}

func NewClient(address, password string) Client {
	pool := &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// the redis protocol should probably be made sett-able
			c, err := redis.Dial("tcp", address, redis.DialNetDial(func(network, address string) (net.Conn, error) {

				return tls.Dial(network, address, &tls.Config{
					InsecureSkipVerify: true,
				})
			}))

			if err != nil {
				return nil, err
			}

			if len(password) > 0 {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err := c.Do("PING"); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		// custom connection test method
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if _, err := c.Do("PING"); err != nil {
				return err
			}
			return nil
		},
	}

	return NewClientWithPool(pool)
}

func (c *redisClient) Raw(name string, args A, opts ...Option) (JobId, error) {
	cmd := &Command{
		Command:   name,
		Arguments: args,
	}

	for _, opt := range opts {
		opt.apply(cmd)
	}

	if cmd.ID == "" {
		cmd.ID = uuid.New()
	}

	db := c.pool.Get()
	defer db.Close()

	data, err := json.Marshal(cmd)
	if err != nil {
		return JobId(""), err
	}

	if _, err := db.Do("RPUSH", CommandsQueue, string(data)); err != nil {
		return JobId(""), err
	}

	flag := fmt.Sprintf("result:%v:flag", cmd.ID)
	if _, err := db.Do("BRPOPLPUSH", flag, flag, ResultDefaultTimeout); err != nil {
		return JobId(cmd.ID), fmt.Errorf("failed to queue command '%v'", cmd.ID)
	}

	return JobId(cmd.ID), nil
}

func (c *redisClient) Exists(job JobId) bool {
	db := c.pool.Get()
	defer db.Close()

	flag := fmt.Sprintf("result:%v:flag", job)

	res, err := db.Do("RPOPLPUSH", flag, flag)
	return err == nil && res != nil
}

func (c *redisClient) result(job JobId, timeout ...int) (*Result, error) {
	if !c.Exists(job) {
		return nil, fmt.Errorf("job '%v' does not exist", job)
	}
	db := c.pool.Get()
	defer db.Close()

	var reply []byte
	var err error
	q := fmt.Sprintf("result:%v", job)

	if len(timeout) > 0 {
		reply, err = redis.Bytes(db.Do("BRPOPLPUSH", q, q, timeout[0]))
	} else {
		reply, err = redis.Bytes(db.Do("RPOPLPUSH", q, q))
	}

	if err != nil {
		return nil, err
	}

	var r Result
	if err := json.Unmarshal(reply, &r); err != nil {
		return nil, err
	}

	return &r, nil
}

func (c *redisClient) Result(job JobId, timeout ...int) (*Result, error) {
	to := ResultDefaultTimeout
	if len(timeout) > 0 {
		to = timeout[0]
	}

	return c.result(job, to)
}

func (c *redisClient) ResultNonBlock(job JobId) (*Result, error) {
	return c.result(job)
}

func (c *redisClient) Info() InfoManager {
	return c.info
}
