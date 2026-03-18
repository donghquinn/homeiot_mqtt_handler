package configs

type PostgresConfig struct {
	// PostgreSQL
	DbHost   string `env:"POSTGRES_HOST" validate:"required"`
	DbPort   string `env:"POSTGRES_PORT" envDefault:"5432"`
	DbName   string `env:"POSTGRES_NAME" validate:"required"`
	DbUser   string `env:"POSTGRES_USER" validate:"required"`
	DbPasswd string `env:"POSTGRES_PASSWD" validate:"required"`
}
