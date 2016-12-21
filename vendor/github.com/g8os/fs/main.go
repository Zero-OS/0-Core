package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"net/http"
	_ "net/http/pprof"

	"strings"

	"github.com/g8os/fs/config"
	"github.com/op/go-logging"
	"github.com/robfig/cron"
)

const (
	MetaEngineBolt = "bolt"
	MetaEngineMem  = "memory"
)

var (
	version = "0.1"
	log     = logging.MustGetLogger("main")
	boltdb  = path.Join(os.TempDir(), "aysfs.meta.db")
)

type Options struct {
	Version    bool
	Pprof      bool
	ConfigPath string
	AutoConfig bool
	LogLevel   int
	MetaEngine string
}

var progName = filepath.Base(os.Args[0])

func getCMDOptions() Options {
	opts := Options{}

	flag.BoolVar(&opts.Version, "v", false, "show version")
	flag.BoolVar(&opts.Pprof, "pprof", false, "enable net pprof")

	flag.StringVar(&opts.ConfigPath, "config", "config.toml", "path to config file")
	flag.BoolVar(&opts.AutoConfig, "auto", false, "enable auto configuration")
	flag.StringVar(&opts.ConfigPath, "c", "config.toml", "path to config file")
	flag.IntVar(&opts.LogLevel, "l", 4, "Log level (0 less verbose, to 5 most verbose) default to 4")
	flag.StringVar(&opts.MetaEngine, "meta", MetaEngineBolt, "Specify what metadata engine to use, default to 'bolt' other option is 'memory'")

	flag.Parse()
	flag.Usage = usage

	return opts
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", progName)
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", progName)
	fmt.Fprintf(os.Stderr, "\n")
	flag.PrintDefaults()
}

func configureLogging(options *Options) {
	logging.SetLevel(logging.Level(options.LogLevel), "")
	formatter := logging.MustStringFormatter("%{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)
}

func writePidFile() error {
	pid := fmt.Sprintf("%d", os.Getpid())
	return ioutil.WriteFile("/tmp/aysfs.pid", []byte(pid), 0600)
}

func main() {
	opts := getCMDOptions()
	if opts.Version {
		fmt.Println("Version: ", version)
		os.Exit(0)
	}

	configureLogging(&opts)

	if opts.Pprof {
		log.Info("starting pprof server")
		go func() {
			log.Info("%v", http.ListenAndServe(":6060", nil))

		}()
	}

	writePidFile()

	cfg := config.LoadConfig(opts.ConfigPath)

	scheduler := cron.New()
	scheduler.Start()

	wg := sync.WaitGroup{}

	for _, mount := range cfg.Mount {
		acl := strings.ToUpper(mount.Mode)

		log.Infof("Mount '%s' on %s", acl, mount.Path)

		backend, err := cfg.GetBackend(mount.Backend)
		if err != nil {
			log.Fatalf("Definition of backend %s not found in config, but required for mount %s", mount.Backend, mount.Path)
		}
		storCfg, err := cfg.GetStorCfg(backend.Stor)
		if err != nil {
			log.Fatalf("Definition of ayostor %s not found in config, but required for backend %s", backend.Stor, backend.Name)
		}
		stor, err := storCfg.GetStorClient()
		if err != nil {
			log.Fatal("Failed to initialize stor client %s: %s", storCfg.URL, err)
		}

		if acl == config.RO {
			if strings.EqualFold(mount.Flist, "") {
				log.Fatalf("RO mount point requires a PList")
			}
			wg.Add(1)
			os.MkdirAll(backend.Path, 0775)
			go MountROFS(&wg, scheduler, mount, backend, stor, opts)
		} else if acl == config.OL {
			if strings.EqualFold(mount.Flist, "") {
				log.Fatalf("OL mount point requires a PList")
			}

			wg.Add(1)
			os.MkdirAll(backend.Path, 0775)
			go MountOLFS(&wg, scheduler, mount, backend, stor, opts)
		} else {
			log.Fatalf("Unknown ACL mode '%s' only (RW, RO, OL) are supported", mount.Mode)
		}
	}

	//watchReloadSignal(cfg)

	wg.Wait()
}
