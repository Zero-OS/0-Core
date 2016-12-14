package config

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/g8os/fs/crypto"
	"github.com/g8os/fs/storage"
	"github.com/naoina/toml"
	"github.com/op/go-logging"
	"net/url"
)

var (
	log = logging.MustGetLogger("config")
)

const (
	RO = "RO"
	OL = "OL"
)

type Config struct {
	Mount   []Mount
	Backend map[string]Backend
	Stor    map[string]StorConfig
}

type Mount struct {
	Path     string
	Flist    string
	Backend  string
	Mode     string
	Stor     string `toml:",omitempty"`
	TrimBase bool
	Trim     string
}

type Backend struct {
	Name string `toml:"-"`
	Path string
	Stor string

	AydostorPushCron string `toml:",omitempty"`
	CleanupCron      string `toml:",omitempty"`
	CleanupOlderThan int    `toml:",omitempty"`

	Log string

	Encrypted bool   `toml:",omitempty"`
	UserRsa   string `toml:",omitempty"`
	StoreRsa  string `toml:",omitempty"`

	ClientKey *rsa.PrivateKey `toml:"-"`
	GlobalKey *rsa.PrivateKey `toml:"-"`
}

type StorConfig struct {
	Name string `toml:"-"`

	URL string
}

func (c *StorConfig) GetStorClient() (storage.Storage, error) {
	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http":
		fallthrough
	case "https":
		fallthrough
	case "aydo":
		return storage.NewAydoStorage(u)
	case "ipfs":
		return storage.NewIPFSStorage(u)
	default:
		return nil, fmt.Errorf("Unknown store scheme, only aydo and ipfs are supported")
	}
}

func (c *Config) GetBackend(name string) (*Backend, error) {
	if backend, ok := c.Backend[name]; ok {
		backend.Name = name
		return &backend, nil
	} else {
		return nil, fmt.Errorf("Backend '%s' not found", name)
	}
}

func (c *Config) GetStorCfg(name string) (*StorConfig, error) {
	if stor, ok := c.Stor[name]; ok {
		stor.Name = name
		return &stor, nil
	} else {
		return nil, fmt.Errorf("Stor '%s' not found", name)
	}
}

func (b *Backend) LoadRSAKeys() error {
	if b.Encrypted {
		if _, err := os.Stat(b.UserRsa); err == nil {
			content, err := ioutil.ReadFile(b.UserRsa)
			if err != nil {
				err := fmt.Errorf("Error reading rsa key at %v: %v", b.UserRsa, err)
				log.Errorf(err.Error())
				return err
			}
			b.ClientKey, err = crypto.ReadPrivateKey(content)
			if err != nil {
				err := fmt.Errorf("Error reading rsa key at %v: %v", b.UserRsa, err)
				log.Errorf(err.Error())
				return err
			}
		}

		if _, err := os.Stat(b.StoreRsa); err == nil {
			content, err := ioutil.ReadFile(b.StoreRsa)
			if err != nil {
				err := fmt.Errorf("Error reading rsa key at %v: %v", b.StoreRsa, err)
				log.Errorf(err.Error())
				return err
			}
			b.GlobalKey, err = crypto.ReadPrivateKey(content)
			if err != nil {
				err := fmt.Errorf("Error reading rsa key at %v: %v", b.StoreRsa, err)
				log.Errorf(err.Error())
				return err
			}
		}
	}
	return nil
}

func LoadConfig(path string) *Config {
	cfg := &Config{}
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("can't open config file at %s: %s\n", path, err)
	}
	err = toml.NewDecoder(f).Decode(cfg)
	if err != nil {
		log.Fatalf("can't read config file at %s: %s\n", path, err)
	}

	for name, backend := range cfg.Backend {
		if err := backend.LoadRSAKeys(); err != nil {
			log.Fatal(err)
		}

		cfg.Backend[name] = backend
	}

	return cfg
}
