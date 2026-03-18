package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"org.donghyuns.com/mqtt/listner/configs"
	"org.donghyuns.com/mqtt/listner/internal/logger"
	"org.donghyuns.com/mqtt/listner/pkg/mqtt"
	"org.donghyuns.com/mqtt/listner/pkg/postgres"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := configs.InitiateConfig()
	if err != nil {
		slog.Error(fmt.Sprintf("setting config err: %v", err))
		return
	}

	if err := logger.LogInitialize(cfg.LogConfig); err != nil {
		slog.Error(fmt.Sprintf("init logger err: %v", err))
		return
	}

	dbCon, err := connectPostgres(cfg.PostgresConfig)
	if err != nil {
		slog.Error(fmt.Sprintf("connection postgres err: %v", err))
		return
	}

	if err := dbCon.CheckConnection(); err != nil {
		slog.Error(fmt.Sprintf("ping database connection err: %v", err))
		return
	}

	mqttclient := initSubscribe(*cfg)

	if token := mqttclient.Client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error(fmt.Sprintf("connect mqtt broker err: %v", token.Error()))
		return
	}

	slog.Info("Start Server")

	<-quit
	slog.Info("Received Shut Down Signal")
	mqttclient.Client.Disconnect(1000)
	slog.Info("Server Has been Shutdown Gracefully")
}

func connectPostgres(cfg configs.PostgresConfig) (*postgres.PostgresService, error) {
	return postgres.NewPostgresConnector(cfg)
}

func initSubscribe(cfg configs.AppConfig) *mqtt.MqttService {
	dbCon, err := connectPostgres(cfg.PostgresConfig)
	if err != nil {
		slog.Error(fmt.Sprintf("connection postgres err: %v", err))
		return nil
	}
	return mqtt.NewMqttService(cfg.MqttConfig, dbCon)
}
