package transport

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/zero-os/0-core/base/pm"
)

const (
	SinkQueue = "core:default"
	DBIndex   = 0
)

type Sink struct {
	ch   *channel
	pool *redis.Pool
}

type SinkConfig struct {
	Port int
}

func (c *SinkConfig) Local() string {
	return fmt.Sprintf("127.0.0.1:%d", c.Port)
}

func NewSink(c SinkConfig) (*Sink, error) {
	pool := newPool()
	sink := &Sink{
		pool: newPool(),
		ch:   newChannel(pool),
	}

	pm.AddHandle(sink)

	return sink, nil
}

//RPush pushes values to the right
func (sink *Sink) RPush(key string, args ...[]byte) (int64, error) {
	conn := sink.pool.Get()
	defer conn.Close()
	input := make([]interface{}, 0, len(args)+1)
	input = append(input, key)
	for _, arg := range args {
		input = append(input, arg)
	}

	return redis.Int64(conn.Do("RPUSH", input...))
}

//LTrim trims a list
func (sink *Sink) LTrim(key string, start, stop int64) error {
	conn := sink.pool.Get()
	defer conn.Close()

	_, err := conn.Do("LTRIM", key, start, stop)
	return err
}

//Get gets value from key
func (sink *Sink) Get(key string) ([]byte, error) {
	conn := sink.pool.Get()
	defer conn.Close()
	result, err := redis.Bytes(conn.Do("GET", key))
	//we do the next, because this is how ledis used
	//to behave
	if err == redis.ErrNil {
		return nil, nil
	}
	return result, err
}

//Set sets a value to a key
func (sink *Sink) Set(key string, value []byte) error {
	conn := sink.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, value)
	return err
}

//Del delets keys
func (sink *Sink) Del(keys ...string) (int64, error) {
	conn := sink.pool.Get()
	defer conn.Close()
	input := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		input = append(input, key)
	}

	return redis.Int64(conn.Do("DEL", input...))
}

//LExpire sets TTL on a list
func (sink *Sink) LExpire(key string, duration int64) (int64, error) {
	conn := sink.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("EXPIRE", key, duration))
}

//Result handler implementation
func (sink *Sink) Result(cmd *pm.Command, result *pm.JobResult) {
	if err := sink.Forward(result); err != nil {
		log.Errorf("failed to forward result: %s", cmd.ID)
	}
}

func (sink *Sink) process() {

	for {
		var command pm.Command
		err := sink.ch.GetNext(SinkQueue, &command)
		if err == redis.ErrNil {
			continue
		} else if err != nil {
			log.Errorf("Failed to get next command from (%s): %s", SinkQueue, err)
			<-time.After(200 * time.Millisecond)
			continue
		}

		if command.ID == "" {
			log.Warningf("receiving a command with no ID, dropping")
			continue
		}

		if sink.ch.Flagged(command.ID) {
			log.Errorf("received a command with a duplicate ID(%v), dropping", command.ID)
			continue
		}

		sink.ch.Flag(command.ID)
		log.Debugf("Starting command %s", &command)

		if !sink.allowedCommnad(&command) {
			sink.respondUnknwonCommand(&command)
			continue
		}
		_, err = pm.Run(&command)

		if err == pm.UnknownCommandErr {
			sink.respondUnknwonCommand(&command)
		} else if err != nil {
			log.Errorf("Unknown error while processing command (%s): %s", command, err)
		}
	}
}

func (sink *Sink) respondUnknwonCommand(cmd *pm.Command) {
	result := pm.NewJobResult(cmd)
	result.State = pm.StateUnknownCmd
	sink.Forward(result)
}

//Forward forwards job result
func (sink *Sink) Forward(result *pm.JobResult) error {
	sink.ch.UnFlag(result.ID)
	return sink.ch.Respond(result)
}

//Flag marks a job ID as running
func (sink *Sink) Flag(id string) error {
	return sink.ch.Flag(id)
}

//Start sink
func (sink *Sink) Start() {
	go sink.process()
}

//GetResult gets a result of a job if it exists
func (sink *Sink) GetResult(job string, timeout int) (*pm.JobResult, error) {
	if sink.ch.Flagged(job) {
		return sink.ch.GetResponse(job, timeout)
	}

	return nil, fmt.Errorf("unknown job id '%s' (may be it has expired)", job)
}
