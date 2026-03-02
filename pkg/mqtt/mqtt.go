package mqtt

import (
	"fmt"
	"log/slog"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"org.donghyuns.com/mqtt/listner/configs"
	"org.donghyuns.com/mqtt/listner/pkg/postgres"
)

type MqttService struct {
	Client  mqtt.Client
	logger  *slog.Logger
	dbCon   *postgres.PostgresService
	handler HandleMessageService
	mqttCfg configs.MqttConfig
}

func NewMqttService(dbCon *postgres.PostgresService) *MqttService {
	logger := slog.With("service", "mqtt")

	mqttCfg := configs.MqttCfg

	options := mqtt.NewClientOptions()

	options.AddBroker(mqttCfg.Host)
	options.SetClientID(mqttCfg.ClientId)

	options.SetOrderMatters(mqttCfg.OrderMatters)
	options.ConnectTimeout = time.Duration(mqttCfg.Timeout) * time.Second
	options.WriteTimeout = time.Duration(mqttCfg.WriteTimeout) * time.Second
	options.KeepAlive = mqttCfg.KeepAlive
	options.PingTimeout = time.Duration(mqttCfg.PingTimeout) * time.Second

	options.ConnectRetry = mqttCfg.ConnectRetry
	options.AutoReconnect = mqttCfg.AutoReconnect

	handler := NewHandleMessageService(dbCon, logger)

	mqttService := &MqttService{
		logger:  logger,
		dbCon:   dbCon,
		handler: handler,
		mqttCfg: mqttCfg,
	}

	options.DefaultPublishHandler = mqttService.onDefaultPulisherHandler
	options.OnConnect = mqttService.onConnectfunc
	options.OnConnectionLost = mqttService.onConnectionLost

	options.OnReconnecting = mqttService.onReconnecting

	mqttService.Client = mqtt.NewClient(options)

	return mqttService
}

func (m *MqttService) onConnectfunc(c mqtt.Client) {
	m.logger.Info("connection established")

	t := c.Subscribe(m.mqttCfg.Topic, byte(m.mqttCfg.Qos), m.handler.handleTempMessage)

	go func() {
		_ = t.Wait() // Can also use '<-t.Done()' in releases > 1.2.0
		if t.Error() != nil {
			m.logger.Error(fmt.Sprintf("SUBSCRIBING ERROR: %s\n", t.Error()))
		}
	}()
}

func (m *MqttService) onConnectionLost(cl mqtt.Client, err error) {
	m.logger.Warn(fmt.Sprintf("connection lost: %v", err))
}

func (m *MqttService) onDefaultPulisherHandler(_ mqtt.Client, msg mqtt.Message) {
	m.logger.Info(fmt.Sprintf("UNEXPECTED MESSAGE: %s", msg))
}

func (m *MqttService) onReconnecting(mqtt.Client, *mqtt.ClientOptions) {
	m.logger.Info("attempting to reconnect")
}
