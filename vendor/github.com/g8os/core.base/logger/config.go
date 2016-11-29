package logger

import (
	"github.com/boltdb/bolt"
	"github.com/g8os/core.base/pm"
	"github.com/g8os/core.base/settings"
	"os"
	"path"
	"strings"
	"time"
)

/*
ConfigureLogging attached the correct message handler on top the process manager from the configurations
*/
func ConfigureLogging(coreID uint64) {
	//apply logging handlers.
	dbLoggerConfigured := false
	mgr := pm.GetManager()
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

			handler, err := NewDBLogger(db, logcfg.Levels)
			if err != nil {
				log.Fatalf("DB logger failed to initialize: %s", err)
			}
			mgr.AddMessageHandler(handler.Log)
			registerGetMsgsFunction(db)

			dbLoggerConfigured = true
		case "redis":
			handler := NewRedisLogger(coreID, logcfg.Address, "", logcfg.Levels, logcfg.BatchSize)
			mgr.AddMessageHandler(handler.Log)
		case "console":
			handler := NewConsoleLogger(logcfg.Levels)
			mgr.AddMessageHandler(handler.Log)
		default:
			log.Fatalf("Unsupported logger type: %s", logcfg.Type)
		}
	}
}
