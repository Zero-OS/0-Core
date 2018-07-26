package stream

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	logging "github.com/op/go-logging"
)

const (
	ioBufferSize      = 32 * 1024
	messageBufferSize = 10 * 1024
)

var (
	log             = logging.MustGetLogger("stream")
	pmMsgPattern, _ = regexp.Compile("^(\\d+)(:{2,3})(.*)$")

	errNotHeader = fmt.Errorf("not header")
)

type consumerImpl struct {
	level   uint16
	handler MessageHandler

	multi *Message

	buffer *bufio.Reader
}

type header struct {
	level     uint16
	multiline bool
	length    int
}

//Consume consumes a stream to the end, and calls the handler with the parsed stream messages
func Consume(wg *sync.WaitGroup, source io.ReadCloser, level uint16, handler MessageHandler) {
	c := &consumerImpl{
		level:   level,
		handler: handler,
		buffer:  bufio.NewReaderSize(source, ioBufferSize),
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

func (c *consumerImpl) isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func (c *consumerImpl) parseHead(head []byte) (*header, error) {
	//parse the header without regex
	//valid pattern is
	// `dd?:::?`
	if len(head) < 3 {
		return nil, errNotHeader //noway this is a header
	}

	skip := bytes.Index(head, []byte("::"))

	if skip > 2 || skip <= 0 {
		return nil, errNotHeader
	}

	level, _ := strconv.ParseUint(string(head[0:skip]), 10, 16)
	h := header{
		level: uint16(level),
	}

	if bytes.HasPrefix(head[skip:], []byte(":::")) {
		//multi line
		h.multiline = true
		h.length = skip + 3
	} else if bytes.HasPrefix(head[skip:], []byte("::")) {
		h.length = skip + 2
	} else {
		return nil, errNotHeader
	}

	return &h, nil
}

func (c *consumerImpl) readLine(out io.Writer) error {
	line, err := c.buffer.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return err
	}
	_, err = out.Write(line)
	return err
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
