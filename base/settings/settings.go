package settings

import (
	"fmt"
	"github.com/g8os/core0/base/utils"
	"github.com/op/go-logging"
	"net/url"
	"strings"
)

var (
	log = logging.MustGetLogger("settings")
)

const (
	//ConfigSuffix config file ext
	ConfigSuffix = ".toml"
)

//Logger settings
type Logger struct {
	//logger type, now only 'db' and 'ac' are supported
	Type string
	//list of controlles base URLs
	Controllers []string
	//Process which levels
	Levels []int

	//Log address (for loggers that needs it)
	Address string
	//Flush interval (for loggers that needs it)
	FlushInt int
	//Flush batch size (for loggers that needs it)
	BatchSize int
}

//Extension cmd config
type Extension struct {
	//binary to execute
	Binary string
	//script search path
	Cwd string
	//(optional) Env variables
	Env map[string]string

	Args []string

	key string
}

func (e *Extension) Key() string {
	return e.key
}

//Security certificate path
type Security struct {
	CertificateAuthority string
	ClientCertificate    string
	ClientCertificateKey string
}

//Controller url and certificates
type SinkConfig struct {
	URL      string
	Password string
}

//Settings main agent settings
type AppSettings struct {
	Main struct {
		MaxJobs  int
		Include  string
		Network  string
		LogLevel string
	}

	Sink map[string]SinkConfig

	Extension map[string]Extension

	Logging map[string]Logger

	Stats struct {
		Interval int
		Redis    struct {
			Enabled       bool
			FlushInterval int
			Address       string
		}
	}

	Channel struct {
		Cmds []string
	}
}

var Settings AppSettings

func (s *AppSettings) Validate() []error {
	if s.Main.LogLevel == "" {
		s.Main.LogLevel = "info"
	}

	errors := make([]error, 0)
	for name, con := range s.Sink {
		if u, err := url.Parse(con.URL); err != nil {
			verr := fmt.Errorf("[sink.%s] `url`: %s", name, err)
			errors = append(errors, verr)
		} else if !utils.InString([]string{"redis"}, strings.ToLower(u.Scheme)) {
			verr := fmt.Errorf("[sink.%s] `url` has unknown schema (%s), only redis is allowed atm", name, u.Scheme)
			errors = append(errors, verr)
		}
	}

	return errors
}

//GetSettings loads main settings from a filename
func LoadSettings(filename string) error {
	//that's the main config file, panic if can't load
	if err := utils.LoadTomlFile(filename, &Settings); err != nil {
		return err
	}

	return nil
}
