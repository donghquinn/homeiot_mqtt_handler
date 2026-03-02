package configs

type MqttConfig struct {
	Topic    string `envconfig:"MQTT_TOPIC" validate:"required"`
	Qos      int    `envconfig:"MQTT_QOS" default:"1"`
	ClientId string `envconfig:"MQTT_CLIENT_ID" validate:"required"`
}
