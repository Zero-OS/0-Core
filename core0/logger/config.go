package logger

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/g8os/core0/base/logger"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/stream"
	"github.com/g8os/core0/base/settings"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("logger")

	loggers Loggers
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
func InitLogging() {
	//apply logging handlers.
	dbLoggerConfigured := false
	for _, logcfg := range settings.Settings.Logging {
		switch strings.ToLower(logcfg.Type) {
		case "db":
			if dbLoggerConfigured {
				log.Fatalf("Only one db logger can be configured")
			}
			//sqlFactory := logger.NewSqliteFactory(logcfg.LogDir)
			os.MkdirAll(logcfg.Address, 0755)
			db, err := bolt.Open(path.Join(logcfg.Address, "logs.db"), 0644, nil)
			if err != nil {
				log.Errorf("Failed to configure db logger: %s", err)
				continue
			}
			db.MaxBatchDelay = 100 * time.Millisecond
			if err != nil {
				log.Fatalf("Failed to open logs database: %s", err)
			}

			handler, err := logger.NewDBLogger(0, db, logcfg.Levels)
			if err != nil {
				log.Fatalf("DB logger failed to initialize: %s", err)
			}

			loggers = append(loggers, handler)
			registerGetMsgsFunction(db)
			dbLoggerConfigured = true
		case "redis":
			handler := logger.NewRedisLogger(0, logcfg.Address, "", logcfg.Levels, logcfg.BatchSize)
			loggers = append(loggers, handler)
		case "console":
			handler := logger.NewConsoleLogger(0, logcfg.Levels)
			loggers = append(loggers, handler)
		default:
			log.Fatalf("Unsupported logger type: %s", logcfg.Type)
		}
	}

	pm.GetManager().AddMessageHandler(loggers.Log)
}
