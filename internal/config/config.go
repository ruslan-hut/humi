package config

import (
	"fmt"
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the service configuration, loaded from YAML with environment overrides.
type Config struct {
	Env    string `yaml:"env" env-default:"local" env-required:"true"`
	Listen struct {
		BindIP string `yaml:"bind_ip" env-default:"127.0.0.1"`
		Port   string `yaml:"port" env-default:"9820"`
	} `yaml:"listen"`
	Database struct {
		Path string `yaml:"path" env-default:"humi.db"`
	} `yaml:"database"`
	Ingest struct {
		MaxBatch     int `yaml:"max_batch" env-default:"32"`
		MaxAgeS      int `yaml:"max_age_s" env-default:"86400"`
		RatePerMin   int `yaml:"rate_per_min" env-default:"30"`
		OfflineRatio int `yaml:"offline_ratio" env-default:"3"` // last_seen older than ratio*interval means offline
	} `yaml:"ingest"`
	Telegram struct {
		Enabled bool   `yaml:"enabled" env-default:"false"`
		ApiKey  string `yaml:"api_key" env-default:""`
		ChatID  string `yaml:"chat_id" env-default:""`
	} `yaml:"telegram"`
	Web struct {
		Dir string `yaml:"dir" env-default:""` // static Angular build, empty disables serving
	} `yaml:"web"`
}

var instance *Config
var once sync.Once

// MustLoad reads the configuration file and terminates the process if it cannot.
func MustLoad(path string) *Config {
	var err error
	once.Do(func() {
		instance = &Config{}
		if err = cleanenv.ReadConfig(path, instance); err != nil {
			desc, _ := cleanenv.GetDescription(instance, nil)
			err = fmt.Errorf("%s; %s", err, desc)
			instance = nil
			log.Fatal(err)
		}
	})
	return instance
}
