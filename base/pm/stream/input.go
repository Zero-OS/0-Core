package stream

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	logging "github.com/op/go-logging"
)

var (
	log             = logging.MustGetLogger("stream")
	pmMsgPattern, _ = regexp.Compile("^(\\d+)(:{2,3})(.*)$")
)

type consumerImpl struct {
	level   uint16
	handler MessageHandler

	multi *Message

	buffer *bufio.Reader
}

//Consume consumes a stream to the end, and calls the handler with the parsed stream messages
func Consume(wg *sync.WaitGroup, source io.ReadCloser, level uint16, handler MessageHandler) {
	c := &consumerImpl{
		level:   level,
		handler: handler,
		buffer:  bufio.NewReaderSize(source, 32*1024),
	}

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		if err := c.consume(); err != nil {
			log.Errorf("failed to read stream: %s", err)
		}

		source.Close()
	}()
}

func (c *consumerImpl) consume() error {
	for {
		line, err := c.buffer.ReadString('\n')
		if err == io.EOF {
			if len(line) != 0 {
				c.processLine(line)
			}
			return nil
		} else if err != nil {
			return err
		}

		c.processLine(line)
	}
}

func (c *consumerImpl) processLine(line string) {
	line = strings.TrimRight(line, "\n")

	if c.multi != nil {
		if line == ":::" {
			//last, flush mult
			c.handler(c.multi)
			c.multi = nil
			return
		}

		c.multi.Message += "\n" + line
		return
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
