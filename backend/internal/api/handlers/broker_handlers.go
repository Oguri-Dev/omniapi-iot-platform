package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"omniapi/internal/broker"
	"omniapi/internal/polling"
)

// ListBrokersHandler retorna todos los brokers configurados
func ListBrokersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	manager := polling.GetEngine().GetBrokerManager()
	if manager == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker manager not available",
		})
		return
	}

	status := manager.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"data":      status,
		"timestamp": time.Now().Unix(),
	})
}

// AddBrokerHandler agrega un nuevo broker
func AddBrokerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config broker.BrokerConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Generar ID automáticamente si no viene
	if config.ID == "" {
		if config.Name != "" {
			// Generar ID basado en el nombre (slug)
			config.ID = strings.ToLower(strings.ReplaceAll(config.Name, " ", "-"))
			config.ID = strings.ReplaceAll(config.ID, "_", "-")
		} else {
			// Generar ID único basado en timestamp
			config.ID = "broker-" + time.Now().Format("20060102150405")
		}
	}

	if config.BrokerURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker URL is required",
		})
		return
	}

	manager := polling.GetEngine().GetBrokerManager()
	if manager == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker manager not available",
		})
		return
	}

	if err := manager.AddBroker(&config); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Broker added successfully",
		"data":      config,
		"timestamp": time.Now().Unix(),
	})
}

// UpdateBrokerHandler actualiza un broker existente
func UpdateBrokerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "PUT" && r.Method != "PATCH" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extraer broker ID de la URL: /api/brokers/{id}
	brokerID := strings.TrimPrefix(r.URL.Path, "/api/brokers/")
	if brokerID == "" || brokerID == r.URL.Path {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker ID is required in URL",
		})
		return
	}

	var config broker.BrokerConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Asegurar que el ID coincida
	config.ID = brokerID

	manager := polling.GetEngine().GetBrokerManager()
	if manager == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker manager not available",
		})
		return
	}

	if err := manager.UpdateBroker(&config); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Broker updated successfully",
		"data":      config,
		"timestamp": time.Now().Unix(),
	})
}

// RemoveBrokerHandler elimina un broker
func RemoveBrokerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extraer broker ID de la URL: /api/brokers/{id}
	brokerID := strings.TrimPrefix(r.URL.Path, "/api/brokers/")
	if brokerID == "" || brokerID == r.URL.Path {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker ID is required in URL",
		})
		return
	}

	manager := polling.GetEngine().GetBrokerManager()
	if manager == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker manager not available",
		})
		return
	}

	if err := manager.RemoveBroker(brokerID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Broker removed successfully",
		"timestamp": time.Now().Unix(),
	})
}

// TestBrokerConnectionHandler prueba la conexión a un broker
func TestBrokerConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config broker.BrokerConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	manager := polling.GetEngine().GetBrokerManager()
	if manager == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Broker manager not available",
		})
		return
	}

	if err := manager.TestConnection(&config); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Connection test failed: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Connection test successful",
		"timestamp": time.Now().Unix(),
	})
}

// GetTopicTemplatesHandler retorna los templates de topics disponibles
func GetTopicTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"templates": broker.DefaultTopicTemplates,
			"variables": []string{
				"{provider}",
				"{site_id}",
				"{tenant_id}",
				"{data_type}",
				"{endpoint}",
				"{instance}",
			},
		},
		"timestamp": time.Now().Unix(),
	})
}
