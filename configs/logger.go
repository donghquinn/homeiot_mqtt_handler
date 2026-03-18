package configs

type LogConfig struct {
	Level      string `env:"AS_LOG_LEVEL" envDefault:"debug"`
	OutputType string `env:"AS_LOG_OUTPUT_TYPE" envDefault:"both"`
	MaxSize    int    `env:"AS_LOG_MAX_SIZE" envDefault:"31457280"`
	Path       string `env:"AS_LOG_PATH" envDefault:"/home/node/logs/saju.log"`
}
