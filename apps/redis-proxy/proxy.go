package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/tidwall/redcon"
	"github.com/zero-os/0-core/apps/redis-proxy/assets"
)

func newPool(socket string) *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("unix", socket)
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
func Proxy(listen, redis, organization string) error {
	authMethod := func(_ string) bool {
		return true
	}

	doAuth := false

	if organization != "" {
		var err error
		authMethod, err = AuthMethod(organization, string(assets.MustAsset("text/itsyouonline.pub")))
		if err != nil {
			return err
		}
		doAuth = true
	}

	p := redisProxy{
		pool:       newPool(redis),
		authMethod: authMethod,
		doAuth:     doAuth,
	}

	tlsConfig, err := generateCRT()
	if err != nil {
		return err
	}

	return redcon.ListenAndServeTLS(
		listen,
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
	}

	write := func(conn redcon.Conn, result interface{}) {
		switch result := result.(type) {
		case error:
			conn.WriteError(result.Error())
		case int64:
			conn.WriteInt64(result)
		case string:
			conn.WriteString(result)
		case []byte:
			conn.WriteBulk(result)
		default:
			conn.WriteError(fmt.Sprintf("unhandled return type: %T(%v)", result, result))
		}
	}
	switch result := result.(type) {
	case []interface{}:
		conn.WriteArray(len(result))
		for _, elm := range result {
			write(conn, elm)
		}
	default:
		write(conn, result)
	}
}

func (r *redisProxy) handler(conn redcon.Conn, cmd redcon.Command) {
	command := strings.ToLower(string(cmd.Args[0]))
	if command == "auth" {
		r.auth(conn, cmd)
		return
	}

	// translation for compatability with ledis
	switch command {
	// the next 2 cases are for compatibility with
	// older client that worked against ledis
	case "lkeyexists":
		cmd.Args[0] = []byte("exists")
	case "lttl":
		cmd.Args[0] = []byte("ttl")
	}

	r.proxy(conn, cmd)
}

func (r *redisProxy) accept(conn redcon.Conn) bool {
	return true
}

func (r *redisProxy) closed(conn redcon.Conn, err error) {

}
