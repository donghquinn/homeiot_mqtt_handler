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
	handler HandleMessageService
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

	options.DefaultPublishHandler = func(_ mqtt.Client, msg mqtt.Message) {
		fmt.Printf("UNEXPECTED MESSAGE: %s\n", msg)
	}

	options.OnConnectionLost = func(cl mqtt.Client, err error) {
		logger.Warn(fmt.Sprintf("connection lost: %v", err))
	}

	handler := NewHandleMessageService(dbCon, logger)

	options.OnConnect = func(c mqtt.Client) {
		logger.Info("connection established")

		// Establish the subscription - doing this here means that it will happen every time a connection is established
		// (useful if opts.CleanSession is TRUE or the broker does not reliably store session data)
		t := c.Subscribe(mqttCfg.Topic, byte(mqttCfg.Qos), handler.handleTempMessage)
		// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
		// in other handlers does cause problems its best to just assume we should not block
		go func() {
			_ = t.Wait() // Can also use '<-t.Done()' in releases > 1.2.0
			if t.Error() != nil {
				logger.Error(fmt.Sprintf("SUBSCRIBING ERROR: %s\n", t.Error()))
			}
		}()
	}

	options.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
		logger.Info("attempting to reconnect")
	}

	return MqttService{
		Client: mqtt.NewClient(options),
		logger: logger,
	}
}
