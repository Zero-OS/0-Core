package settings

import (
	"github.com/g8os/core0/base/utils"
	"github.com/op/go-logging"
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
	Levels []int `json:"levels"`
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

	Globals   Globals              `json:"globals"`
	Extension map[string]Extension `json:"extension"`
	Logging   struct {
		File  Logger `json:"file"`
		Ledis struct {
			Logger `json:"ledis"`
			Size   int64 `json:"size"`
		}
	} `json:"logger"`

	Containers struct {
		MaxCount int `json:"max_count"`
	} `json:"containers"`
	Stats struct {
		Enabled       bool `json:"enabled"`
		FlushInterval int  `json:"flush_interval"` //in seconds
	} `json:"stats"`
}

var Settings AppSettings

func (s *AppSettings) Validate() []error {
	if s.Main.LogLevel == "" {
		s.Main.LogLevel = "info"
	}

	return nil
}

//GetSettings loads main settings from a filename
func LoadSettings(filename string) error {
	//that's the main config file, panic if can't load
	if err := utils.LoadTomlFile(filename, &Settings); err != nil {
		return err
	}

	return nil
}
