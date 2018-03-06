package transport

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/zero-os/0-core/base/pm"
)

const (
	//expires in 300 seconds (5min)
	ReturnExpire = 300
)

/*
ControllerClient represents an active agent controller connection.
*/
type channel struct {
	pool *redis.Pool
}

/*
NewSinkClient gets a new sink connection with the given identity. Identity is used by the sink client to
introduce itself to the sink terminal.
*/
func newChannel(pool *redis.Pool) *channel {
	ch := &channel{
		pool: pool,
	}

	return ch
}

func (cl *channel) String() string {
	return "redis"
}

//GetNext gets the next available command
func (cl *channel) GetNext(queue string, command *pm.Command) error {
	conn := cl.pool.Get()
	defer conn.Close()

	payload, err := redis.ByteSlices(conn.Do("BLPOP", queue, 10))
	if err != nil {
		return err
	}

	if payload == nil || len(payload) < 2 {
		return redis.ErrNil
	}

	return json.Unmarshal(payload[1], command)
}

func (cl *channel) Respond(result *pm.JobResult) error {
	if result.ID == "" {
		return fmt.Errorf("result with no ID, not pushing results back")
	}

	queue := fmt.Sprintf("result:%s", result.ID)

	if err := cl.Push(queue, result); err != nil {
		return err
	}

	conn := cl.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("EXPIRE", queue, ReturnExpire); err != nil {
		return err
	}

	return nil
}

func (cl *channel) Push(queue string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	conn := cl.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("RPUSH", queue, data); err != nil {
		return err
	}

	return nil
}

func (cl *channel) cycle(queue string, timeout int) ([]byte, error) {
	conn := cl.pool.Get()
	defer conn.Close()

	payload, err := redis.ByteSlices(conn.Do("BRPOPLPUSH", queue, queue, timeout))
	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, fmt.Errorf("timeout")
	}

	data := payload[1]
	return data, nil
}

func (cl *channel) GetResponse(id string, timeout int) (*pm.JobResult, error) {
	queue := fmt.Sprintf("result:%s", id)
	payload, err := cl.cycle(queue, timeout)
	if err != nil {
		return nil, err
	}

	var result pm.JobResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (cl *channel) Flag(id string) error {
	conn := cl.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("result:%s:flag", id)
	_, err := conn.Do("RPUSH", key, "")
	return err
}

func (cl *channel) UnFlag(id string) error {
	conn := cl.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("result:%s:flag", id)
	_, err := conn.Do("EXPIRE", key, ReturnExpire)
	return err
}

func (cl *channel) Flagged(id string) bool {
	conn := cl.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("result:%s:flag", id)
	v, _ := redis.Int(conn.Do("EXISTS", key))
	return v == 1
}
