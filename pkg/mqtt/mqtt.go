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

func NewMqttService(cfg configs.MqttConfig, dbCon *postgres.PostgresService) *MqttService {
	logger := slog.With("service", "mqtt")

	options := mqtt.NewClientOptions()

	options.AddBroker(cfg.Host)
	options.SetClientID(cfg.ClientId)

	options.SetOrderMatters(cfg.OrderMatters)
	options.ConnectTimeout = time.Duration(cfg.Timeout) * time.Second
	options.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	options.KeepAlive = cfg.KeepAlive
	options.PingTimeout = time.Duration(cfg.PingTimeout) * time.Second

	options.ConnectRetry = cfg.ConnectRetry
	options.AutoReconnect = cfg.AutoReconnect

	handler := NewHandleMessageService(dbCon, logger)

	mqttService := &MqttService{
		logger:  logger,
		dbCon:   dbCon,
		handler: handler,
		mqttCfg: cfg,
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
