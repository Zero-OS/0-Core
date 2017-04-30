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
	Type string `json:"type"`
	//list of controlles base URLs
	Controllers []string `json:"controllers"`
	//Process which levels
	Levels []int `json:"levels"`

	//Log address (for loggers that needs it)
	Address string `json:"address"`
	//Flush interval (for loggers that needs it)
	FlushInt int `json:"flush_int"`
	//Flush batch size (for loggers that needs it)
	BatchSize int `json:"batch_size"`
}

//Extension cmd config
type Extension struct {
	//binary to execute
	Binary string `json:"binary"`
	//script search path
	Cwd string `json:"cwd"`
	//(optional) Env variables
	Env map[string]string `json:"env"`

	Args []string `json:"args"`

	key string `json:"key"`
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
	URL      string `json:"url"`
	Password string `json:"password"`
}

type Globals map[string]string

func (g Globals) Get(key string, def ...string) string {
	v, ok := g[key]
	if !ok && len(def) == 1 {
		return def[0]
	}

	return v
}

//Settings main agent settings
type AppSettings struct {
	Main struct {
		MaxJobs  int      `json:"max_jobs"`
		Include  []string `json:"include"`
		Network  string   `json:"network"`
		LogLevel string   `json:"log_level"`
	} `json:"main"`

	Globals   Globals               `json:"globals"`
	Sink      map[string]SinkConfig `json:"sink"`
	Extension map[string]Extension  `json:"extension"`
	Logging   map[string]Logger     `json:"logger"`

	Stats struct {
		//Interval is deprecated
		Interval int `json:"interval"`
		Redis    struct {
			Enabled       bool   `json:"enabled"`
			FlushInterval int    `json:"flush_interval"` //in seconds
			Address       string `json:"address"`
		} `json:"redis"`
	} `json:"stats"`
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
