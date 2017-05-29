package logger

import (
	"github.com/zero-os/0-core/base/logger"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
	"github.com/op/go-logging"
	"github.com/siddontang/ledisdb/ledis"
)

var (
	log = logging.MustGetLogger("logger")

	Current Loggers
)

type Loggers []logger.Logger

func (l Loggers) Log(cmd *core.Command, msg *stream.Message) {
	//default logging
	for _, logger := range l {
		logger.Log(cmd, msg)
	}
}

func (l Loggers) LogRecord(record *logger.LogRecord) {
	for _, logger := range l {
		logger.LogRecord(record)
	}
}

// ConfigureLogging attachs the correct message handler on top the process manager from the configurations
func ConfigureLogging(db *ledis.DB) {
	file := logger.NewConsoleLogger(0, settings.Settings.Logging.File.Levels)
	ledis := logger.NewLedisLogger(0, db, settings.Settings.Logging.Ledis.Levels, settings.Settings.Logging.Ledis.Size)

	Current = append(Current, file, ledis)

	pm.GetManager().AddMessageHandler(Current.Log)
}
