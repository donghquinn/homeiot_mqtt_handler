package configs

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
)

type PostgresConfig struct {
	// PostgreSQL
	DbHost   string `envconfig:"POSTGRES_HOST" validate:"required"`
	DbPort   string `envconfig:"POSTGRES_PORT" default:"5432"`
	DbName   string `envconfig:"POSTGRES_NAME" validate:"required"`
	DbUser   string `envconfig:"POSTGRES_USER" validate:"required"`
	DbPasswd string `envconfig:"POSTGRES_PASSWD" validate:"required"`
}

var PostgresCfg PostgresConfig

func ReadPostgresCfg() error {
	if err := envconfig.Process("POSTGRES", &PostgresCfg); err != nil {
		return fmt.Errorf("read postgres cfg err: %v", err)
	}
	return nil
}

func (g *PostgresConfig) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(g); err != nil {
		return fmt.Errorf("validate postgres cfg err: %v", err)
	}
	return nil
}
