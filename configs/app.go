package configs

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Env string `envconfig:"APP_ENV" default:"development"`
}

var AppCfg AppConfig

func ReadAppConfig() error {
	if err := envconfig.Process("APP", &AppCfg); err != nil {
		return fmt.Errorf("read app config err: %v", err)
	}
	return nil
}

func (a *AppConfig) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(a); err != nil {
		return fmt.Errorf("validate app config err: %v", err)
	}
	return nil
}
