package watcher

import (
	"github.com/g8os/fs/config"
	"github.com/g8os/fs/meta"
	"github.com/op/go-logging"
	"github.com/robfig/cron"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	log = logging.MustGetLogger("watcher")
)

type backenCleaner struct {
	backend *config.Backend
	meta    meta.MetaStore
	now     time.Time
}

func NewCleaner(meta meta.MetaStore, backend *config.Backend) cron.Job {
	return &backenCleaner{
		backend: backend,
	}
}

func (c *backenCleaner) Run() {
	log.Debugf("Cleaner is awake, checking files to clean up...")
	if err := filepath.Walk(c.backend.Path, c.walkFN); err != nil {
		log.Errorf("Failed to walk backend '%s': %s", c.backend.Path, err)
	}
}

func (c *backenCleaner) walkFN(name string, info os.FileInfo, err error) error {
	if strings.HasSuffix(name, meta.MetaSuffix) || info.IsDir() || err != nil {
		return nil
	}

	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		atime := time.Unix(sys.Atim.Unix())
		if c.now.Sub(atime) > time.Duration(c.backend.CleanupOlderThan)*time.Hour {
			if _, exists := c.meta.Get(name); !exists {
				return nil
			}

			log.Debugf("Cleaner: removing file '%s'", name)
			err := os.Remove(name)
			if err != nil {
				log.Warning("Failed to clean up file '%s'", name)
			}
		}
	}

	return nil
}
