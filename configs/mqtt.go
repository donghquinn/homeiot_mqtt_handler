package configs

type MqttConfig struct {
	Host     string `env:"MQTT_HOST" validate:"required"`
	Topic    string `env:"MQTT_TOPIC" validate:"required"`
	Qos      int    `env:"MQTT_QOS" envDefault:"1"`
	ClientId string `env:"MQTT_CLIENT_ID" validate:"required"`

	OrderMatters bool  `env:"MQTT_ORDER_MATTERS" envDefault:"false"` // 순서가 중요할 때
	Timeout      int   `env:"MQTT_TIMEOUT" envDefault:"10"`
	WriteTimeout int   `env:"MQTT_WRITE_TIMEOUT" envDefault:"1"`
	KeepAlive    int64 `env:"MQTT_KEEP_ALIVE" envDefault:"10"`
	PingTimeout  int   `env:"MQTT_PING_TIMEOUT" envDefault:"1"`

	ConnectRetry  bool `env:"MQTT_CONNECT_RETRY" envDefault:"true"`
	AutoReconnect bool `env:"MQTT_AUTO_RECONNECT" envDefault:"true"`
}
