package watcher

import (
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	logging "log"
	"time"
)

type TLogger interface {
	Log(path, hash string)
}

type tlogger struct {
	logger *logging.Logger
	writer *lumberjack.Logger
}

func NewTLogger(name string) TLogger {
	writer := &lumberjack.Logger{
		Filename:   name,
		MaxSize:    5, //Megabytes
		MaxBackups: 3,
		MaxAge:     3, //Days
		LocalTime:  true,
	}
	logger := logging.New(writer, "", 0)

	go func(t *time.Ticker) {
		for _ = range t.C {
			writer.Rotate()
		}
	}(time.NewTicker(24 * time.Hour))

	return &tlogger{
		logger: logger,
		writer: writer,
	}
}

func (l *tlogger) Log(path, hash string) {
	l.logger.Println(fmt.Sprintf("%s|%s|%d", path, hash, time.Now().Unix()))
}
