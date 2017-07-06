package logger

import (
	"github.com/siddontang/ledisdb/ledis"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
)

var (
	Current Loggers
)

type Loggers []Logger

func (l Loggers) log(cmd *core.Command, msg *stream.Message) {
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
func ConfigureLogging(db *ledis.DB) {
	Current = append(Current,
		NewConsoleLogger(settings.Settings.Logging.File.Levels),
		NewLedisLogger(db, settings.Settings.Logging.Ledis.Levels, settings.Settings.Logging.Ledis.Size),
		NewStreamLogger(db, 0),
	)

	pm.GetManager().AddMessageHandler(Current.log)
}
