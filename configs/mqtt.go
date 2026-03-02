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
}

var MqttCfg MqttConfig

func (m *MqttConfig) ReadConfig() error {
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
