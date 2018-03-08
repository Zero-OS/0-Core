package logger

import (
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/apps/core0/transport"
)

var (
	Current Loggers
)

type Loggers []Logger

//message handler implementation
func (l Loggers) Message(cmd *pm.Command, msg *stream.Message) {
	l.LogRecord(&LogRecord{
		Command: cmd.ID,
		Message: msg,
	})
}

func (l Loggers) LogRecord(record *LogRecord) {
	for _, logger := range l {
		logger.LogRecord(record)
	}
}

// ConfigureLogging attachs the correct message handler on top the process manager from the configurations
func ConfigureLogging(sink *transport.Sink) {
	Current = append(Current,
		NewConsoleLogger(settings.Settings.Logging.File.Levels),
		NewLedisLogger(sink, settings.Settings.Logging.Ledis.Levels, settings.Settings.Logging.Ledis.Size),
		NewStreamLogger(sink, 0),
	)

	pm.AddHandle(Current)
}
