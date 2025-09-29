package config

import "github.com/kelseyhightower/envconfig"

type Configuration struct {
	Logger
}

type Logger struct {
	Level       string `envconfig:"RT_LOG_LEVEL" default:"debug"`
	Path        string `envconfig:"RT_LOG_PATH" default:"./logs/access.log"`
	PrintStdOut bool   `envconfig:"RT_LOG_STDOUT" default:"true"`
}

func LoadConfiguration() (*Configuration, error) {
	var cfg Configuration
	if err := envconfig.Process("rt", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
