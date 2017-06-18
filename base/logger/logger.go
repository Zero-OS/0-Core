package logger

import (
	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base/pm/stream"
)

var (
	log = logging.MustGetLogger("logger")
)

type LogRecord struct {
	Core    uint16          `json:"core"`
	Command string          `json:"command"`
	Message *stream.Message `json:"message"`
}

// Logger interface
type Logger interface {
	LogRecord(record *LogRecord)
}

func IsLoggable(defaults []uint16, msg *stream.Message) bool {
	if len(defaults) > 0 {
		return msg.Meta.Assert(defaults...)
	}

	return true
}

// ConsoleLogger log message to the console
type ConsoleLogger struct {
	coreID   uint16
	defaults []uint16
}

// NewConsoleLogger creates a simple console logger that prints log messages to Console.
func NewConsoleLogger(coreID uint16, defaults []uint16) Logger {
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
