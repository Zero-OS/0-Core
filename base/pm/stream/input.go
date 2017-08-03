package stream

import (
	"bytes"
	"github.com/op/go-logging"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	log             = logging.MustGetLogger("stream")
	pmMsgPattern, _ = regexp.Compile("^(\\d+)(:{2,3})(.*)$")
)

type Consumer interface {
	Write(p []byte) (n int, err error)
}

type consumerImpl struct {
	level   uint16
	handler MessageHandler

	last  []byte
	multi *Message
}

func NewConsumer(wg *sync.WaitGroup, source io.ReadCloser, level uint16, handler MessageHandler) Consumer {
	c := &consumerImpl{
		level:   level,
		handler: handler,
	}

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		io.Copy(c, source)
		source.Close()
	}()

	return c
}

func (c *consumerImpl) Write(p []byte) (n int, err error) {
	n = len(p)
	if len(c.last) > 0 {
		p = append(c.last, p...)
	}

	reader := bytes.NewBuffer(p)
	for {
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			//reached end of current chunk. we need to wait until we
			//get more data.
			c.last = []byte(line)
			return n, nil
		} else if err != nil {
			return 0, err
		}

		line = strings.TrimRight(line, "\n")
		if c.multi != nil {
			if line == ":::" {
				//last, flush mult
				c.handler(c.multi)
				c.multi = nil
				return n, nil
			} else {
				c.multi.Message += "\n" + line
			}

			continue
		}

		matches := pmMsgPattern.FindStringSubmatch(line)

		if matches == nil {
			//use default level.
			c.handler(&Message{
				Meta:    NewMeta(c.level),
				Message: line,
			})
		} else {
			l, _ := strconv.ParseUint(matches[1], 10, 16)
			level := uint16(l)
			message := matches[3]

			if matches[2] == ":::" {
				c.multi = &Message{
					Meta:    NewMeta(level),
					Message: message,
				}
			} else {
				//single line message
				c.handler(&Message{
					Meta:    NewMeta(level),
					Message: message,
				})
			}
		}
	}
}
