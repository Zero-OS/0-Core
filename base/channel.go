package core

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/settings"
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
func newChannel(cfg *settings.Channel) (*channel, error) {
	u, err := url.Parse(cfg.URL)
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

	pool := utils.NewRedisPool(network, address, cfg.Password)

	ch := &channel{
		url:   strings.TrimRight(cfg.URL, "/"),
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

	db := cl.redis.Get()
	defer db.Close()

	queue := fmt.Sprintf("result:%s", result.ID)

	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}

	if _, err := db.Do("RPUSH", queue, payload); err != nil {
		return err
	}
	if _, err := db.Do("EXPIRE", queue, ReturnExpire); err != nil {
		return err
	}

	return nil
}

func (cl *channel) GetResponse(id string, timeout int) (*core.JobResult, error) {
	db := cl.redis.Get()
	defer db.Close()

	queue := fmt.Sprintf("result:%s", id)
	payload, err := redis.Bytes(db.Do("BRPOPLPUSH", queue, queue, timeout))
	if err != nil {
		return nil, err
	}

	var result core.JobResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
