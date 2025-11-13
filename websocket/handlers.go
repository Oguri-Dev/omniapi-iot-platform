package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WSHandler maneja las conexiones WebSocket
func WSHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	// Obtener parÃ¡metros de query
	tenantIDStr := r.URL.Query().Get("tenantId")
	if tenantIDStr == "" {
		conn.WriteJSON(ErrorMessage{
			Type:    MessageTypeERROR,
			Code:    "MISSING_TENANT",
			Message: "tenantId query parameter is required",
		})
		conn.Close()
		return
	}

	// Parsear TenantID
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		conn.WriteJSON(ErrorMessage{
			Type:    MessageTypeERROR,
			Code:    "INVALID_TENANT",
			Message: "Invalid tenantId format",
		})
		conn.Close()
		return
	}

	// Generar client ID
	clientID := r.URL.Query().Get("clientId")
	if clientID == "" {
		clientID = "ws_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	// Registrar cliente en el router (con capabilities vacÃ­as por defecto)
	hub.router.RegisterClient(clientID, tenantID, nil, nil, nil)

	// Crear nuevo cliente
	client := &Client{
		ID:                 clientID,
		TenantID:           tenantID,
		Conn:               conn,
		Send:               make(chan interface{}, 256),
		Hub:                hub,
		subscriptions:      make(map[string]*ClientSubscription),
		includeStatus:      false,
		lastStatusByKey:    make(map[string]*StatusEventMessage),
		maxDeliverySamples: 1000,
	}

	// Registrar cliente en el hub
	client.Hub.register <- client

	// Enviar mensaje de bienvenida
	client.Send <- AckMessage{
		Type:    MessageTypeACK,
		Message: "Connected to WebSocket server",
		Data: map[string]interface{}{
			"clientId": clientID,
			"tenantId": tenantIDStr,
			"protocol": "omniapi-ws-v1",
		},
	}

	// Iniciar goroutines para lectura y escritura
	go client.writePump()
	go client.readPump()
}

// WSStatsHandler proporciona estadÃ­sticas del WebSocket via HTTP
func WSStatsHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	stats := hub.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"success": true,
		"message": "WebSocket Statistics",
		"data": map[string]interface{}{
			"connections_active":         stats.ConnectionsActive,
			"total_connections":          stats.TotalConnections,
			"ws_events_data_out_total":   stats.WSEventsDataOutTotal,
			"ws_events_status_out_total": stats.WSEventsStatusOutTotal,
			"ws_delivery_p95_ms":         stats.WSDeliveryP95Ms,
			"messages_sent":              stats.MessagesSent,
			"messages_received":          stats.MessagesReceived,
		},
		"timestamp": time.Now().Unix(),
	}

	jsonBytes, _ := json.Marshal(response)
	w.Write(jsonBytes)
}

// WSTestHandler proporciona una pÃ¡gina de prueba del WebSocket
func WSTestHandler(w http.ResponseWriter, r *http.Request) {
	// Leer archivo HTML
	htmlBytes, err := os.ReadFile("websocket/test_client.html")
	if err != nil {
		// Si no existe el archivo, generar HTML simple
		html := `<!DOCTYPE html>
<html><head><title>WebSocket Test</title></head>
<body><h1>WebSocket Test Client</h1>
<p>File websocket/test_client.html not found. Please ensure it exists.</p>
<p>Expected path: websocket/test_client.html</p>
</body></html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlBytes)
}
