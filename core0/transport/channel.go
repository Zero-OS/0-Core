package transport

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/utils"
	"github.com/garyburd/redigo/redis"
	"net/url"
	"strings"
)

const (
	ReturnExpire = 300
)

/*
ControllerClient represents an active agent controller connection.
*/
type channel struct {
	url   string
	redis *redis.Pool
}

/*
NewSinkClient gets a new sink connection with the given identity. Identity is used by the sink client to
introduce itself to the sink terminal.
*/
func newChannel(con string, password string) (*channel, error) {
	u, err := url.Parse(con)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "redis" {
		return nil, fmt.Errorf("expected url of format redis://<host>:<port> or redis:///unix.socket")
	}

	network := "tcp"
	address := u.Host
	if address == "" {
		network = "unix"
		address = u.Path
	}

	pool := utils.NewRedisPool(network, address, password)

	ch := &channel{
		url:   strings.TrimRight(con, "/"),
		redis: pool,
	}

	return ch, nil
}

func (client *channel) String() string {
	return client.url
}

func (cl *channel) GetNext(queue string, command *core.Command) error {
	db := cl.redis.Get()
	defer db.Close()

	payload, err := redis.ByteSlices(db.Do("BLPOP", queue, 0))
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

	db := cl.redis.Get()
	defer db.Close()

	if _, err := db.Do("EXPIRE", queue, ReturnExpire); err != nil {
		return err
	}

	return nil
}

func (cl *channel) Push(queue string, payload interface{}) error {
	db := cl.redis.Get()
	defer db.Close()

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := db.Do("RPUSH", queue, data); err != nil {
		return err
	}

	return nil
}

func (cl *channel) GetResponse(id string, timeout int) (*core.JobResult, error) {
	db := cl.redis.Get()
	defer db.Close()

	queue := fmt.Sprintf("result:%s", id)
	payload, err := redis.Bytes(db.Do("BRPOPLPUSH", queue, queue, timeout))
	if err == redis.ErrNil {
		return nil, fmt.Errorf("timeout")
	} else if err != nil {
		return nil, err
	}

	var result core.JobResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (cl *channel) Flag(id string) error {
	db := cl.redis.Get()
	defer db.Close()

	key := fmt.Sprintf("result:%s:flag", id)
	_, err := db.Do("RPUSH", key, "")
	return err
}

func (cl *channel) UnFlag(id string) error {
	db := cl.redis.Get()
	defer db.Close()

	key := fmt.Sprintf("result:%s:flag", id)
	_, err := db.Do("EXPIRE", key, ReturnExpire)
	return err
}

func (cl *channel) Flagged(id string) bool {
	db := cl.redis.Get()
	defer db.Close()

	key := fmt.Sprintf("result:%s:flag", id)
	v, _ := redis.Int(db.Do("EXISTS", key))
	return v == 1
}
