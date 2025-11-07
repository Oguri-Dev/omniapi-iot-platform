package adapters

import (
	"omniapi/adapters/dummy"
	"omniapi/internal/connectors"
	"omniapi/internal/connectors/adapters/mqttfeed"
	"omniapi/internal/connectors/adapters/restclimate"
)

// RegisterAllAdapters registra todos los adaptadores disponibles
func RegisterAllAdapters() error {
	// Dummy adapter para testing
	if err := connectors.RegisterConnector(dummy.Registration); err != nil {
		return err
	}

	// MQTT feed adapter para datos de alimentación vía MQTT
	if err := connectors.RegisterConnector(mqttfeed.Registration); err != nil {
		return err
	}

	// REST climate adapter para datos climáticos vía HTTP polling
	if err := connectors.RegisterConnector(restclimate.Registration); err != nil {
		return err
	}

	return nil
}
