package main

import (
	"fmt"
	"github.com/g8os/fs/config"
	"github.com/g8os/fs/files"
	"github.com/g8os/fs/meta"
	"github.com/g8os/fs/storage"
	"github.com/g8os/fs/watcher"
	"github.com/robfig/cron"
	"os"
	//"path"
	"sync"
)

const (
	FileReadBuffer = 512 * 1024 //bytes [512K]
)

//
//func watchReloadSignal(cfg *config.Config) {
//	channel := make(chan os.Signal)
//	signal.Notify(channel, syscall.SIGUSR1)
//	go func(cfg *config.Config) {
//		defer close(channel)
//		for {
//			<-channel
//			log.Info("Reloading ays mounts due to user signal")
//
//			for _, mount := range cfg.Mount {
//				//process only RO, and OL
//				backend, err := cfg.GetBackend(mount.Backend)
//				if err != nil {
//					log.Warningf("Couldn't retrive backend '%s'", backend.Name)
//				}
//				base := ""
//				if mount.TrimBase {
//					base = mount.Path
//				}
//				err = meta.Populate(backend, base, mount.Flist, mount.Trim)
//				if err != nil {
//					log.Warningf("Couldn't reload backend meta: %s", err)
//				}
//			}
//		}
//	}(cfg)
//}

func mountFS(
	mountCfg config.Mount,
	backendCfg *config.Backend,
	stor storage.Storage,
	meta meta.MetaStore,
	readOnly bool) error {

	fs, err := files.NewFS(mountCfg.Path, backendCfg, stor, meta, readOnly)
	if err != nil {
		return err
	}
	log.Info("Serving File system")
	fs.Serve()

	return nil
}

func MountOLFS(wg *sync.WaitGroup, scheduler *cron.Cron, mount config.Mount, backend *config.Backend, stor storage.Storage, opts Options) {
	//ms := meta.NewFileMetaStore(backend.Path)
	metaBackend := fmt.Sprintf("%s+meta", backend.Path)
	os.MkdirAll(metaBackend, 0755)
	ms := meta.NewFileMetaStore(metaBackend)

	if err := ms.Populate(mount.Flist, mount.Trim); err != nil {
		log.Errorf("Failed to mount overllay fs '%s': %s", mount, err)
	}

	//2- Start the cleaner worker, but never the watcher since we don't push ever to stor in OL mode
	job := watcher.NewCleaner(ms, backend)
	cron := backend.CleanupCron
	if cron == "" {
		cron = "@every 1d"
	}
	scheduler.AddJob(cron, job)

	//TODO: 3- start RWFS with overlay compatibility.
	if err := mountFS(mount, backend, stor, ms, false); err != nil {
		log.Fatal(err)
	}
	wg.Done()
}

func MountROFS(wg *sync.WaitGroup, scheduler *cron.Cron, mount config.Mount, backend *config.Backend, stor storage.Storage, opts Options) {
	//ms := meta.NewFileMetaStore(backend.Path)
	metaBackend := fmt.Sprintf("%s.meta", backend.Path)
	os.MkdirAll(metaBackend, 0755)
	ms := meta.NewFileMetaStore(metaBackend)

	if err := ms.Populate(mount.Flist, mount.Trim); err != nil {
		log.Errorf("Failed to mount overllay fs '%s': %s", mount, err)
	}

	//2- Start the cleaner worker, but never the watcher since we don't push ever to stor in OL mode
	job := watcher.NewCleaner(ms, backend)
	cron := backend.CleanupCron
	if cron == "" {
		cron = "@every 1d"
	}
	scheduler.AddJob(cron, job)

	if err := mountFS(mount, backend, stor, ms, true); err != nil {
		log.Fatal(err)
	}
	wg.Done()
}
