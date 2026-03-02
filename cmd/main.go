package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"org.donghyuns.com/mqtt/listner/configs"
	"org.donghyuns.com/mqtt/listner/internal/utils"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	if err := godotenv.Load(".env"); err != nil {
		slog.Error(fmt.Sprintf("load env err: %v", err))
		return
	}

	if err := utils.SetupGlobalLogger("logs", 1000, 1000, configs.AppCfg.Env); err != nil {
		slog.Error(fmt.Sprintf("Failed to setup logger: %v", err))
		return
	}

	if err := readConfigs(); err != nil {
		slog.Error(fmt.Sprintf("read configs err: %v", err))

		return
	}

	if err := validateConfigs(); err != nil {
		slog.Error(fmt.Sprintf("validate configs err: %v", err))

		return
	}

	go func() {
		slog.Debug("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@")
		slog.Info("Start Server")
		slog.Debug("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@")

	}()

	<-quit
	slog.Info("Received Shut Down Signal")

}

func readConfigs() error {
	if err := configs.ReadAppConfig(); err != nil {
		return fmt.Errorf("read app cfg err: %v", err)
	}
	if err := configs.ReadMqttConfig(); err != nil {
		return fmt.Errorf("read mqtt cfg err: %v", err)
	}
	if err := configs.ReadPostgresCfg(); err != nil {
		return fmt.Errorf("read postgres cfg err: %v", err)
	}

	return nil
}

func validateConfigs() error {
	if err := configs.AppCfg.Validate(); err != nil {
		return fmt.Errorf("validate app configs err: %v", err)
	}

	if err := configs.MqttCfg.Validate(); err != nil {
		return fmt.Errorf("validate mqtt configs err: %v", err)
	}

	if err := configs.PostgresCfg.Validate(); err != nil {
		return fmt.Errorf("validate postgres configs err: %v", err)
	}
	return nil
}
