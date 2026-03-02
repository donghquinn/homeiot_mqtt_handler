package configs

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
)

type MqttConfig struct {
	Host     string `envconfig:"MQTT_HOST" validate:"required"`
	Topic    string `envconfig:"MQTT_TOPIC" validate:"required"`
	Qos      int    `envconfig:"MQTT_QOS" default:"1"`
	ClientId string `envconfig:"MQTT_CLIENT_ID" validate:"required"`

	OrderMatters bool  `envconfig:"MQTT_ORDER_MATTERS" default:"false"` // 순서가 중요할 때
	Timeout      int   `envconfig:"MQTT_TIMEOUT" default:"10"`
	WriteTimeout int   `envconfig:"MQTT_WRITE_TIMEOUT" default:"1"`
	KeepAlive    int64 `envconfig:"MQTT_KEEP_ALIVE" default:"10"`
	PingTimeout  int   `envconfig:"MQTT_PING_TIMEOUT" default:"1"`

	ConnectRetry  bool `envconfig:"MQTT_CONNECT_RETRY" default:"true"`
	AutoReconnect bool `envconfig:"MQTT_AUTO_RECONNECT" default:"true"`
}

var MqttCfg MqttConfig

func ReadMqttConfig() error {
	if err := envconfig.Process("MQTT", &MqttCfg); err != nil {
		return fmt.Errorf("read mqtt config err: %v", err)
	}

	return nil
}

func (m *MqttConfig) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(m); err != nil {
		return fmt.Errorf("validate mqtt config err: %v", err)
	}
	return nil
}
