package logger

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/utils"
	"github.com/op/go-logging"
)

var (
	log      = logging.MustGetLogger("logger")
	Disabled = []int{stream.LevelInvalid}
)

type LogRecord struct {
	Core    uint16          `json:"core"`
	Command string          `json:"command"`
	Message *stream.Message `json:"message"`
}

// Logger interface
type Logger interface {
	Log(*core.Command, *stream.Message)
	LogRecord(record *LogRecord)
}

func IsLoggableCmd(cmd *core.Command, msg *stream.Message) bool {
	if len(cmd.LogLevels) > 0 {
		return utils.In(cmd.LogLevels, msg.Level)
	}

	return true
}

func IsLoggable(defaults []int, msg *stream.Message) bool {
	if len(defaults) > 0 {
		return utils.In(defaults, msg.Level)
	}

	return true
}

// DBLogger implements a logger that stores the message in a bold database.
type DBLogger struct {
	coreID   uint16
	db       *bolt.DB
	defaults []int
}

// NewDBLogger creates a new Database logger, it stores the logged message in database
// factory: is the DB connection factory
// defaults: default log levels to store in db if is not specificed by the logged message.
func NewDBLogger(coreID uint16, db *bolt.DB, defaults []int) (Logger, error) {
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
		coreID:   coreID,
		db:       db,
		defaults: defaults,
	}, nil
}

func (logger *DBLogger) Log(cmd *core.Command, msg *stream.Message) {
	if !IsLoggableCmd(cmd, msg) {
		return
	}

	logger.LogRecord(&LogRecord{
		Core:    logger.coreID,
		Command: cmd.ID,
		Message: msg,
	})
}

// Log message
func (logger *DBLogger) LogRecord(record *LogRecord) {
	if !IsLoggable(logger.defaults, record.Message) {
		return
	}

	go logger.db.Batch(func(tx *bolt.Tx) error {
		logs := tx.Bucket([]byte("logs"))
		jobBucket, err := logs.CreateBucketIfNotExists([]byte(record.Command))
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		value, err := json.Marshal(record.Message)
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		key := []byte(fmt.Sprintf("%020d-%03d", record.Message.Epoch, record.Message.Level))
		return jobBucket.Put(key, value)
	})
}

// ConsoleLogger log message to the console
type ConsoleLogger struct {
	coreID   uint16
	defaults []int
}

// NewConsoleLogger creates a simple console logger that prints log messages to Console.
func NewConsoleLogger(coreID uint16, defaults []int) Logger {
	return &ConsoleLogger{
		coreID:   coreID,
		defaults: defaults,
	}
}

func (logger *ConsoleLogger) LogRecord(record *LogRecord) {
	if !IsLoggable(logger.defaults, record.Message) {
		return
	}
	log.Infof("[%d]%s %s", record.Core, record.Command, record.Message)
}

// Log messages
func (logger *ConsoleLogger) Log(cmd *core.Command, msg *stream.Message) {
	if !IsLoggableCmd(cmd, msg) {
		return
	}

	logger.LogRecord(&LogRecord{
		Core:    logger.coreID,
		Command: cmd.ID,
		Message: msg,
	})

}
