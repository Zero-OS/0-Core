package transport

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/garyburd/redigo/redis"
	"github.com/siddontang/ledisdb/ledis"
	"time"
)

const (
	ReturnExpire = 300
)

/*
ControllerClient represents an active agent controller connection.
*/
type channel struct {
	db *ledis.DB
}

/*
NewSinkClient gets a new sink connection with the given identity. Identity is used by the sink client to
introduce itself to the sink terminal.
*/
func newChannel(db *ledis.DB) *channel {
	ch := &channel{
		db: db,
	}

	return ch
}

func (client *channel) String() string {
	return "ledis"
}

func (cl *channel) GetNext(queue string, command *core.Command) error {
	payload, err := redis.ByteSlices(cl.db.BLPop([][]byte{[]byte(queue)}, 0))
	if err != nil {
		return err
	}

	return json.Unmarshal(payload[1], command)
}

func (cl *channel) Respond(result *core.JobResult) error {
	if result.ID == "" {
		return fmt.Errorf("result with no ID, not pushing results back...")
	}

	queue := fmt.Sprintf("result:%s", result.ID)

	if err := cl.Push(queue, result); err != nil {
		return err
	}

	if _, err := cl.db.Expire([]byte(queue), ReturnExpire); err != nil {
		return err
	}

	return nil
}

func (cl *channel) Push(queue string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := cl.db.RPush([]byte(queue), data); err != nil {
		return err
	}

	return nil
}

func (cl *channel) cycle(queue string, timeout int) ([]byte, error) {
	db := cl.db
	payload, err := redis.ByteSlices(db.BRPop([][]byte{[]byte(queue)}, time.Duration(timeout)*time.Second))
	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, fmt.Errorf("timeout")
	}

	data := payload[1]
	if _, err := db.LPush([]byte(queue), data); err != nil {
		return nil, err
	}

	return data, nil
}

func (cl *channel) GetResponse(id string, timeout int) (*core.JobResult, error) {
	queue := fmt.Sprintf("result:%s", id)
	payload, err := cl.cycle(queue, timeout)
	if err != nil {
		return nil, err
	}

	var result core.JobResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (cl *channel) Flag(id string) error {
	key := fmt.Sprintf("result:%s:flag", id)
	_, err := cl.db.RPush([]byte(key), []byte(""))
	return err
}

func (cl *channel) UnFlag(id string) error {
	key := fmt.Sprintf("result:%s:flag", id)
	_, err := cl.db.Expire([]byte(key), ReturnExpire)
	return err
}

func (cl *channel) Flagged(id string) bool {
	key := fmt.Sprintf("result:%s:flag", id)
	v, _ := cl.db.LKeyExists([]byte(key))
	return v == 1
}
