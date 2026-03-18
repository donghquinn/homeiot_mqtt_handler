package configs

import (
	"fmt"

	"github.com/caarlos0/env/v8"
)

type AppConfig struct {
	GlobalConfig
	LogConfig
	MqttConfig
	PostgresConfig
}

type GlobalConfig struct {
	Env string `env:"APP_ENV" envDefault:"development"`
}

func InitiateConfig() (*AppConfig, error) {
	cfg := AppConfig{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read env : %v", err)
	}
	return &cfg, nil
}
