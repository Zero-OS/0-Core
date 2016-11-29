package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/pm/stream"
	"github.com/g8os/core.base/utils"
	"github.com/op/go-logging"
	"net/http"
	"time"
)

var (
	log = logging.MustGetLogger("logger")
)

/*
Logger interface
*/
type Logger interface {
	Log(*core.Command, *stream.Message)
}

/*
DBLogger implements a logger that stores the message in a bold database.
*/
type DBLogger struct {
	db       *bolt.DB
	defaults []int
}

/*
NewDBLogger creates a new Database logger, it stores the logged message in database
factory: is the DB connection factory
defaults: default log levels to store in db if is not specificed by the logged message.
*/
func NewDBLogger(db *bolt.DB, defaults []int) (Logger, error) {
	tx, err := db.Begin(true)

	defer tx.Rollback()

	if err != nil {
		return nil, err
	}

	if _, err := tx.CreateBucketIfNotExists([]byte("logs")); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &DBLogger{
		db:       db,
		defaults: defaults,
	}, nil
}

//Log message
func (logger *DBLogger) Log(cmd *core.Command, msg *stream.Message) {
	levels := logger.defaults
	msgLevels := cmd.LogLevels

	if len(msgLevels) > 0 {
		levels = msgLevels
	}

	if len(levels) > 0 && !utils.In(levels, msg.Level) {
		return
	}

	go logger.db.Batch(func(tx *bolt.Tx) error {
		logs := tx.Bucket([]byte("logs"))
		jobBucket, err := logs.CreateBucketIfNotExists([]byte(cmd.ID))
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		value, err := json.Marshal(msg)
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		key := []byte(fmt.Sprintf("%020d-%03d", msg.Epoch, msg.Level))
		return jobBucket.Put(key, value)
	})
}

/*
ACLogger buffers the messages, then send it to the agent controller in bulks
*/
type ACLogger struct {
	endpoints map[string]*http.Client
	buffer    utils.Buffer
	defaults  []int
}

/*
NewACLogger creates a new AC logger. AC logger buffers log messages into bulks and batch send it to the given end points over HTTP (POST)
endpoints: list of URLs that the AC logger will post the batches to
bufsize: Max number of messages to keep before sending the data to the end points
flushInt: Max time to wait before sending data to the end points. So either a full buffer or flushInt can force flushing
  the messages
defaults: default log levels to store in db if is not specificed by the logged message.
*/
func NewACLogger(endpoints map[string]*http.Client, bufsize int, flushInt time.Duration, defaults []int) Logger {
	logger := &ACLogger{
		endpoints: endpoints,
		defaults:  defaults,
	}

	logger.buffer = utils.NewBuffer(bufsize, flushInt, logger.send)

	return logger
}

//Log message
func (logger *ACLogger) Log(cmd *core.Command, msg *stream.Message) {
	levels := logger.defaults
	msgLevels := cmd.LogLevels

	if len(msgLevels) > 0 {
		levels = msgLevels
	}

	if len(levels) > 0 && !utils.In(levels, msg.Level) {
		return
	}

	logger.buffer.Append(msg)
}

func (logger *ACLogger) send(objs []interface{}) {
	if len(objs) == 0 {
		//objs can be of length zero, when flushed on timer while
		//no messages are ready.
		return
	}

	msgs, err := json.Marshal(objs)
	if err != nil {
		log.Errorf("Failed to serialize the logs")
		return
	}

	reader := bytes.NewReader(msgs)
	for endpoint, client := range logger.endpoints {
		resp, err := client.Post(endpoint, "application/json", reader)
		if err != nil {
			log.Errorf("Failed to send log batch to controller '%s': %s", endpoint, err)
			continue
		}
		defer resp.Body.Close()
	}
}

/*
ConsoleLogger log message to the console
*/
type ConsoleLogger struct {
	defaults []int
}

//NewConsoleLogger creates a simple console logger that prints log messages to Console.
func NewConsoleLogger(defaults []int) Logger {
	return &ConsoleLogger{
		defaults: defaults,
	}
}

//Log messages
func (logger *ConsoleLogger) Log(cmd *core.Command, msg *stream.Message) {
	if len(logger.defaults) > 0 && !utils.In(logger.defaults, msg.Level) {
		return
	}

	log.Infof("%s %s", cmd, msg)
}
