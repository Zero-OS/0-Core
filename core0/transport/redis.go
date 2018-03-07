package transport

import (
	"fmt"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/tidwall/redcon"
	"github.com/zero-os/0-core/core0/assets"
	"github.com/zero-os/0-core/core0/options"
)

func newPool() *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("unix", "/tmp/redis.sock")
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxActive: 20,
		Wait:      true,
	}
}

type redisProxy struct {
	pool       *redis.Pool
	authMethod func(string) bool
	doAuth     bool
}

//liste to core0 port and
func Listen() error {
	authMethod := func(_ string) bool {
		return true
	}

	doAuth := false

	if orgs, ok := options.Options.Kernel.Get("organization"); ok {
		org := orgs[len(orgs)-1]
		var err error
		authMethod, err = AuthMethod(org, string(assets.MustAsset("text/itsyouonline.pub")))
		if err != nil {
			return err
		}
		doAuth = true
	}

	p := redisProxy{
		pool:       newPool(),
		authMethod: authMethod,
		doAuth:     doAuth,
	}

	tlsConfig, err := generateCRT()
	if err != nil {
		return err
	}

	return redcon.ListenAndServeTLS(
		":6379",
		p.handler,
		p.accept,
		p.closed,
		tlsConfig,
	)
}

func (r *redisProxy) auth(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("invalid number of arguments")
		return
	}

	password := string(cmd.Args[1])

	if r.authMethod(password) {
		conn.SetContext(true)
		conn.WriteString("OK")
	} else {
		conn.WriteError("invalid jwt")
	}
}

func (r *redisProxy) proxy(conn redcon.Conn, cmd redcon.Command) {
	// is authorized ?
	if ctx := conn.Context(); r.doAuth && ctx == nil {
		//ctx was not set, hence he either didn't call auth or not authorized
		conn.WriteError("permission denied, please call AUTH first with a valid JWT")
		return
	}

	//proxy to underlying redis
	local := r.pool.Get()
	defer local.Close()

	args := make([]interface{}, 0, len(cmd.Args)-1)

	for _, arg := range cmd.Args[1:] {
		args = append(args, arg)
	}

	result, err := local.Do(string(cmd.Args[0]), args...)

	if err != nil {
		conn.WriteError(err.Error())
		return
	} else if result == nil {
		conn.WriteNull()
		return
	} else if result, err := redis.Int64(result, err); err == nil {
		conn.WriteInt64(result)
		return
	} else if result, err := redis.String(result, err); err == nil {
		if result == "OK" || result == "PONG" {
			conn.WriteString(result)
		} else {
			conn.WriteBulkString(result)
		}
		return
	} else if result, err := redis.Bytes(result, err); err == nil {
		conn.WriteBulk(result)
		return
	} else if result, err := redis.Strings(result, err); err == nil {
		conn.WriteArray(len(result))
		for _, r := range result {
			conn.WriteBulkString(r)
		}
		return
	}
	conn.WriteError(fmt.Sprintf("unhandled return type: %T(%v)", result, result))
}

func (r *redisProxy) handler(conn redcon.Conn, cmd redcon.Command) {
	switch strings.ToLower(string(cmd.Args[0])) {
	case "auth":
		r.auth(conn, cmd)
	default:
		r.proxy(conn, cmd)
	}
}

func (r *redisProxy) accept(conn redcon.Conn) bool {
	return true
}

func (r *redisProxy) closed(conn redcon.Conn, err error) {

}
