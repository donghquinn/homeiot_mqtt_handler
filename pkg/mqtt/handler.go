package mqtt

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/donghquinn/gqbd"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"org.donghyuns.com/mqtt/listner/internal/constants"
	"org.donghyuns.com/mqtt/listner/pkg/postgres"
)

type HandleMessageService struct {
	logger *slog.Logger
	dbCon  *postgres.PostgresService
}

func NewHandleMessageService(dbCon *postgres.PostgresService, logger *slog.Logger) HandleMessageService {
	return HandleMessageService{
		logger: logger,
		dbCon:  dbCon,
	}
}

func (h *HandleMessageService) handleTempMessage(_ mqtt.Client, msg mqtt.Message) {
	// We extract the count and write that out first to simplify checking for missing values
	var message TempAndHumid
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		h.logger.Error(fmt.Sprintf("Message could not be parsed (%s): %s", msg.Payload(), err))
		return
	}

	h.logger.Debug(fmt.Sprintf("received message: %+v", message))
	if err := h.insertNewTempData(message); err != nil {
		h.logger.Error(fmt.Sprintf("saving temp data err: %v", err))
		return
	}

	h.logger.Debug("saving temp data successfully")
}

func (h *HandleMessageService) insertNewTempData(message TempAndHumid) error {
	data := map[string]any{
		"temperature": message.Temperature,
		"humidity":    message.Humidity,
	}

	query, args, err := gqbd.BuildInsert(gqbd.PostgreSQL, string(constants.TempHumidTable)).Values(data).Build()
	if err != nil {
		return fmt.Errorf("build query string inserting temp and humid data err: %v", err)
	}

	defer h.dbCon.Client.Close()
	if _, err := h.dbCon.Client.Exec(query, args...); err != nil {
		return fmt.Errorf("query inserting temp and humid data err: %v", err)
	}

	return nil
}
