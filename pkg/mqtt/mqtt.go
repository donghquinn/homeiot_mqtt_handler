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

func NewMqttService(dbCon *postgres.PostgresService) MqttService {
	logger := slog.With("service", "mqtt")

	mqttCfg := configs.MqttCfg

	options := mqtt.NewClientOptions()

	options.AddBroker(mqttCfg.Host)
	options.SetClientID(mqttCfg.ClientId)

	options.SetOrderMatters(mqttCfg.OrderMatters)
	options.ConnectTimeout = time.Duration(mqttCfg.Timeout)
	options.WriteTimeout = time.Duration(mqttCfg.WriteTimeout)
	options.KeepAlive = mqttCfg.KeepAlive
	options.PingTimeout = time.Duration(mqttCfg.PingTimeout)

	options.ConnectRetry = mqttCfg.ConnectRetry
	options.AutoReconnect = mqttCfg.AutoReconnect

	handler := NewHandleMessageService(dbCon, logger)

	mqttService := MqttService{
		logger:  logger,
		dbCon:   dbCon,
		handler: handler,
		mqttCfg: mqttCfg,
	}

	options.DefaultPublishHandler = mqttService.onDefaultPulisherHandler
	options.OnConnect = mqttService.onConnectfunc
	options.OnConnectionLost = mqttService.onConnectionLost

	options.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
		logger.Info("attempting to reconnect")
	}

	mqttService.Client = mqtt.NewClient(options)

	return mqttService
}

func (m *MqttService) onConnectfunc(c mqtt.Client) {
	m.logger.Info("connection established")

	// Establish the subscription - doing this here means that it will happen every time a connection is established
	// (useful if opts.CleanSession is TRUE or the broker does not reliably store session data)
	t := c.Subscribe(m.mqttCfg.Topic, byte(m.mqttCfg.Qos), m.handler.handleTempMessage)
	// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
	// in other handlers does cause problems its best to just assume we should not block
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
