package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
	"omniapi/internal/metrics"
	"omniapi/internal/router"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Configuración del upgrader para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir conexiones desde cualquier origen (CORS)
		// En producción, deberías ser más restrictivo
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Tipos de mensajes WebSocket
const (
	// Cliente → Servidor
	MessageTypeSUB   = "SUB"   // Suscripción a streams
	MessageTypeUNSUB = "UNSUB" // Cancelar suscripción
	MessageTypePING  = "PING"  // Ping/keep-alive

	// Servidor → Cliente
	MessageTypeACK    = "ACK"    // Confirmación
	MessageTypeERROR  = "ERROR"  // Error
	MessageTypePONG   = "PONG"   // Respuesta a ping
	MessageTypeDATA   = "DATA"   // Evento de datos
	MessageTypeSTATUS = "STATUS" // Evento de estado
	MessageTypeWARN   = "WARN"   // Advertencia (mensajes legacy)

	// Legacy (para compatibilidad)
	MessageTypeChat         = "chat"
	MessageTypeNotification = "notification"
	MessageTypeSystem       = "system"
)

// StreamFilter representa un filtro de stream en SUB
type StreamFilter struct {
	Kind   string  `json:"kind"`
	SiteID string  `json:"siteId"`
	CageID *string `json:"cageId,omitempty"`
	Metric *string `json:"metric,omitempty"`
}

// SubMessage mensaje de suscripción del cliente
type SubMessage struct {
	Type          string         `json:"type"` // "SUB"
	Streams       []StreamFilter `json:"streams"`
	IncludeStatus *bool          `json:"includeStatus,omitempty"`
	ThrottleMs    *int           `json:"throttleMs,omitempty"`
	NeedSnapshot  *bool          `json:"needSnapshot,omitempty"`
}

// UnsubMessage mensaje para cancelar suscripción
type UnsubMessage struct {
	Type string `json:"type"` // "UNSUB"
}

// PingMessage keep-alive
type PingMessage struct {
	Type string `json:"type"` // "PING"
}

// AckMessage confirmación de operación
type AckMessage struct {
	Type    string      `json:"type"` // "ACK"
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorMessage error de operación
type ErrorMessage struct {
	Type    string `json:"type"` // "ERROR"
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PongMessage respuesta a PING
type PongMessage struct {
	Type string `json:"type"` // "PONG"
}

// WarnMessage advertencia (mensajes legacy)
type WarnMessage struct {
	Type       string `json:"type"` // "WARN"
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// StreamInfo información del stream en eventos
type StreamInfo struct {
	Tenant string  `json:"tenant"`
	SiteID string  `json:"siteId"`
	CageID *string `json:"cageId,omitempty"`
	Kind   string  `json:"kind"`
	Metric string  `json:"metric"`
}

// DataEventMessage evento de datos
type DataEventMessage struct {
	Type    string                 `json:"type"`    // "DATA"
	Version string                 `json:"v"`       // Versión del protocolo
	TS      int64                  `json:"ts"`      // Timestamp Unix (ms)
	Stream  StreamInfo             `json:"stream"`  // Información del stream
	Payload map[string]interface{} `json:"payload"` // Datos del evento
	Flags   *struct {
		Partial *bool `json:"partial,omitempty"`
	} `json:"flags,omitempty"`
}

// StatusInfo información de estado del stream
type StatusInfo struct {
	LastSuccessTS *int64  `json:"last_success_ts,omitempty"` // Timestamp último éxito (ms)
	LastLatencyMs *int64  `json:"last_latency_ms,omitempty"` // Latencia del último request (ms)
	StalenessSec  int     `json:"staleness_s"`               // Segundos desde último éxito
	InFlight      bool    `json:"in_flight"`                 // Hay request en curso
	LastErrorTS   *int64  `json:"last_error_ts,omitempty"`   // Timestamp último error (ms)
	LastErrorMsg  *string `json:"last_error_msg,omitempty"`  // Mensaje del último error
	State         string  `json:"state"`                     // ok|partial|degraded|failing|paused
	Source        string  `json:"source"`                    // Fuente del dato
	Notes         *string `json:"notes,omitempty"`           // Notas adicionales
}

// StatusEventMessage evento de estado
type StatusEventMessage struct {
	Type    string     `json:"type"`   // "STATUS"
	Version string     `json:"v"`      // Versión del protocolo
	TS      int64      `json:"ts"`     // Timestamp Unix (ms)
	Stream  StreamInfo `json:"stream"` // Información del stream
	Status  StatusInfo `json:"status"` // Estado del stream
}

// Message estructura genérica para mensajes legacy
type Message struct {
	Type      string      `json:"type"`
	Content   string      `json:"content"`
	Data      interface{} `json:"data,omitempty"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Timestamp int64       `json:"timestamp"`
	ID        string      `json:"id"`
}

// Client representa una conexión WebSocket individual
type Client struct {
	ID       string
	TenantID primitive.ObjectID
	Conn     *websocket.Conn
	Send     chan interface{} // Canal para mensajes salientes (puede ser DataEventMessage, StatusEventMessage, etc.)
	Hub      *Hub

	// Estado de suscripción
	mu                 sync.RWMutex
	subscriptions      map[string]*ClientSubscription // key: subscription ID del router
	includeStatus      bool                           // Si incluye eventos STATUS
	throttleMs         int                            // Throttle en ms
	lastStatusByKey    map[string]*StatusEventMessage // Para keep-latest policy en STATUS
	deliveryTimes      []float64                      // Tiempos de delivery para P95
	maxDeliverySamples int
}

// ClientSubscription información de suscripción de un cliente
type ClientSubscription struct {
	RouterSubID   string
	IncludeStatus bool
	CreatedAt     time.Time
}

// HubStats estadísticas del Hub
type HubStats struct {
	ConnectionsActive      int64   `json:"connections_active"`
	WSEventsDataOutTotal   int64   `json:"ws_events_data_out_total"`
	WSEventsStatusOutTotal int64   `json:"ws_events_status_out_total"`
	WSDeliveryP95Ms        float64 `json:"ws_delivery_p95_ms"`
	MessagesSent           int64   `json:"messages_sent"`
	MessagesReceived       int64   `json:"messages_received"`
	TotalConnections       int64   `json:"total_connections"`
}

// Hub mantiene el conjunto de clientes activos y transmite mensajes
type Hub struct {
	// Router para suscripciones
	router *router.Router

	// Clientes registrados (key: client ID)
	clients map[string]*Client

	// Mutex para acceso seguro
	mu sync.RWMutex

	// Canales
	register   chan *Client
	unregister chan *Client
	broadcast  chan interface{} // Para mensajes legacy

	// Estadísticas
	stats           HubStats
	deliverySamples []float64
	maxSamples      int
}

// NewHub crea una nueva instancia de Hub
func NewHub(r *router.Router) *Hub {
	return &Hub{
		router:     r,
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan interface{}),
		maxSamples: 1000,
	}
}

// Run ejecuta el hub en un goroutine
func (h *Hub) Run() {
	// Configurar callback del router para recibir eventos
	h.router.SetEventCallback(h.onRouterEvent)

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			// Para mensajes legacy (compatibilidad)
			h.broadcastLegacy(message)
		}
	}
}

// onRouterEvent callback para eventos del router
func (h *Hub) onRouterEvent(clientID string, event *connectors.CanonicalEvent) error {
	h.mu.RLock()
	client, exists := h.clients[clientID]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	startTime := time.Now()

	// Determinar tipo de mensaje según el router
	var message interface{}
	isStatus := false

	// Verificar si es un evento STATUS (por Stream.Kind o flags)
	if event.Envelope.Stream.Kind == "status" || event.Kind[:7] == "status." {
		isStatus = true

		// Verificar si el cliente quiere eventos STATUS
		client.mu.RLock()
		includeStatus := client.includeStatus
		client.mu.RUnlock()

		if !includeStatus {
			return nil // No enviar STATUS a clientes que no lo solicitaron
		}

		// Crear mensaje STATUS
		statusMsg := h.canonicalToStatus(event)
		message = statusMsg

		// Keep-latest: reemplazar STATUS anterior para mismo stream key
		streamKey := h.makeStreamKey(&event.Envelope.Stream)
		client.mu.Lock()
		if client.lastStatusByKey == nil {
			client.lastStatusByKey = make(map[string]*StatusEventMessage)
		}
		client.lastStatusByKey[streamKey] = statusMsg
		client.mu.Unlock()

		h.mu.Lock()
		h.stats.WSEventsStatusOutTotal++
		h.mu.Unlock()
	} else {
		// Evento DATA
		dataMsg := h.canonicalToData(event)
		message = dataMsg

		h.mu.Lock()
		h.stats.WSEventsDataOutTotal++
		h.mu.Unlock()
	}

	// Enviar mensaje al cliente (con backpressure)
	select {
	case client.Send <- message:
		// Registrar tiempo de delivery
		deliveryMs := float64(time.Since(startTime).Microseconds()) / 1000.0
		h.recordDeliveryTime(deliveryMs)

		// Actualizar métricas de Prometheus
		metrics.WSDeliveryLatencyMS.Observe(deliveryMs)
		if isStatus {
			metrics.WSMessagesOutTotal.WithLabelValues(MessageTypeSTATUS).Inc()
		} else {
			metrics.WSMessagesOutTotal.WithLabelValues(MessageTypeDATA).Inc()
		}
	default:
		// Canal lleno - backpressure
		if isStatus {
			// Para STATUS, aplicar keep-latest: descartar el viejo que estaba en el canal
			// y enviar solo el más reciente
			// En este caso, ya lo guardamos en lastStatusByKey, así que no hacemos nada
			metrics.WSEventBackpressureTotal.WithLabelValues(MessageTypeSTATUS).Inc()
			log.Printf("WebSocket congestion for client %s, STATUS buffered (keep-latest)", clientID)
		} else {
			// Para DATA, intentamos enviar o descartamos
			metrics.WSEventBackpressureTotal.WithLabelValues(MessageTypeDATA).Inc()
			log.Printf("WebSocket congestion for client %s, DATA event dropped", clientID)
		}
	}

	return nil
}

// canonicalToData convierte CanonicalEvent a DataEventMessage
func (h *Hub) canonicalToData(event *connectors.CanonicalEvent) *DataEventMessage {
	stream := StreamInfo{
		Tenant: event.Envelope.Stream.TenantID.Hex(),
		SiteID: event.Envelope.Stream.SiteID,
		Kind:   string(event.Envelope.Stream.Kind),
		Metric: event.Kind,
	}
	if event.Envelope.Stream.CageID != nil {
		stream.CageID = event.Envelope.Stream.CageID
	}

	var payload map[string]interface{}
	json.Unmarshal(event.Payload, &payload)

	msg := &DataEventMessage{
		Type:    MessageTypeDATA,
		Version: "1.0",
		TS:      event.Envelope.Timestamp.UnixMilli(),
		Stream:  stream,
		Payload: payload,
	}

	// Agregar flags si hay
	if event.Envelope.Flags&connectors.EventFlagSynthetic != 0 {
		partial := true
		msg.Flags = &struct {
			Partial *bool `json:"partial,omitempty"`
		}{Partial: &partial}
	}

	return msg
}

// canonicalToStatus convierte CanonicalEvent a StatusEventMessage
func (h *Hub) canonicalToStatus(event *connectors.CanonicalEvent) *StatusEventMessage {
	stream := StreamInfo{
		Tenant: event.Envelope.Stream.TenantID.Hex(),
		SiteID: event.Envelope.Stream.SiteID,
		Kind:   string(event.Envelope.Stream.Kind),
		Metric: event.Kind,
	}
	if event.Envelope.Stream.CageID != nil {
		stream.CageID = event.Envelope.Stream.CageID
	}

	// Parsear payload como StatusInfo
	var payloadMap map[string]interface{}
	json.Unmarshal(event.Payload, &payloadMap)

	status := StatusInfo{
		State:  "unknown",
		Source: event.Envelope.Source,
	}

	// Extraer campos del payload
	if state, ok := payloadMap["state"].(string); ok {
		status.State = state
	}
	if stalenessSec, ok := payloadMap["staleness_sec"].(float64); ok {
		status.StalenessSec = int(stalenessSec)
	}
	if inFlight, ok := payloadMap["in_flight"].(bool); ok {
		status.InFlight = inFlight
	}
	if lastSuccessTS, ok := payloadMap["last_success_ts"].(float64); ok {
		ts := int64(lastSuccessTS)
		status.LastSuccessTS = &ts
	}
	if lastLatencyMs, ok := payloadMap["last_latency_ms"].(float64); ok {
		lat := int64(lastLatencyMs)
		status.LastLatencyMs = &lat
	}
	if lastErrorMsg, ok := payloadMap["last_error_msg"].(string); ok {
		status.LastErrorMsg = &lastErrorMsg
	}
	if notes, ok := payloadMap["notes"].(string); ok {
		status.Notes = &notes
	}

	return &StatusEventMessage{
		Type:    MessageTypeSTATUS,
		Version: "1.0",
		TS:      event.Envelope.Timestamp.UnixMilli(),
		Stream:  stream,
		Status:  status,
	}
}

// makeStreamKey crea una clave para identificar un stream
func (h *Hub) makeStreamKey(sk *domain.StreamKey) string {
	key := sk.TenantID.Hex() + ":" + string(sk.Kind) + ":" + sk.SiteID
	if sk.CageID != nil {
		key += ":" + *sk.CageID
	}
	return key
}

// registerClient registra un nuevo cliente
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	h.stats.TotalConnections++
	h.stats.ConnectionsActive++

	// Actualizar métrica de Prometheus
	metrics.WSConnectionsActive.Set(float64(h.stats.ConnectionsActive))
	metrics.WSConnectionsTotal.Inc()

	log.Printf("Cliente WebSocket conectado: %s - Total activos: %d",
		client.ID, h.stats.ConnectionsActive)
}

// unregisterClient desregistra un cliente
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		// Desuscribir del router
		client.mu.Lock()
		for subID := range client.subscriptions {
			h.router.Unsubscribe(subID)
		}
		client.mu.Unlock()

		delete(h.clients, client.ID)
		close(client.Send)
		h.stats.ConnectionsActive--

		// Actualizar métrica de Prometheus
		metrics.WSConnectionsActive.Set(float64(h.stats.ConnectionsActive))

		log.Printf("Cliente WebSocket desconectado: %s - Total activos: %d",
			client.ID, h.stats.ConnectionsActive)
	}
}

// broadcastLegacy envía mensajes legacy (compatibilidad)
func (h *Hub) broadcastLegacy(message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.stats.MessagesSent++

	for _, client := range h.clients {
		select {
		case client.Send <- message:
		default:
			log.Printf("Failed to send legacy message to client %s", client.ID)
		}
	}
}

// recordDeliveryTime registra tiempo de delivery para P95
func (h *Hub) recordDeliveryTime(durationMs float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.deliverySamples == nil {
		h.deliverySamples = make([]float64, 0, h.maxSamples)
	}

	h.deliverySamples = append(h.deliverySamples, durationMs)

	// Limitar tamaño
	if len(h.deliverySamples) > h.maxSamples {
		h.deliverySamples = h.deliverySamples[len(h.deliverySamples)-h.maxSamples:]
	}

	// Calcular P95
	h.stats.WSDeliveryP95Ms = h.calculateP95()
}

// calculateP95 calcula el percentil 95
func (h *Hub) calculateP95() float64 {
	if len(h.deliverySamples) == 0 {
		return 0
	}

	// Copiar y ordenar
	samples := make([]float64, len(h.deliverySamples))
	copy(samples, h.deliverySamples)

	// Bubble sort simple
	for i := 0; i < len(samples)-1; i++ {
		for j := i + 1; j < len(samples); j++ {
			if samples[j] < samples[i] {
				samples[i], samples[j] = samples[j], samples[i]
			}
		}
	}

	idx := int(float64(len(samples)) * 0.95)
	if idx >= len(samples) {
		idx = len(samples) - 1
	}

	return samples[idx]
}

// GetStats retorna las estadísticas del hub
func (h *Hub) GetStats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.stats
}

// GetClient obtiene un cliente por ID
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[clientID]
	return client, ok
}

// readPump maneja la lectura de mensajes del WebSocket
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Configurar timeouts
	c.Conn.SetReadLimit(8192) // Aumentado para mensajes más grandes
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var rawMsg map[string]interface{}
		err := c.Conn.ReadJSON(&rawMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Incrementar contador de mensajes recibidos
		c.Hub.mu.Lock()
		c.Hub.stats.MessagesReceived++
		c.Hub.mu.Unlock()

		// Procesar mensaje según tipo
		msgType, ok := rawMsg["type"].(string)
		if !ok {
			c.sendError("INVALID_MESSAGE", "Message type is required")
			continue
		}

		switch msgType {
		case MessageTypeSUB:
			c.handleSubscribe(rawMsg)
		case MessageTypeUNSUB:
			c.handleUnsubscribe(rawMsg)
		case MessageTypePING:
			c.Send <- PongMessage{Type: MessageTypePONG}
		case MessageTypeChat, MessageTypeNotification:
			// Mensajes legacy - enviar advertencia
			c.Send <- WarnMessage{
				Type:       MessageTypeWARN,
				Message:    "Legacy message type detected",
				Suggestion: "Please use SUB to subscribe to event streams",
			}
		default:
			c.sendError("UNKNOWN_TYPE", "Unknown message type: "+msgType)
		}
	}
}

// handleSubscribe maneja suscripciones
func (c *Client) handleSubscribe(rawMsg map[string]interface{}) {
	var subMsg SubMessage
	rawBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(rawBytes, &subMsg); err != nil {
		c.sendError("INVALID_SUB", "Invalid SUB message format")
		return
	}

	// Guardar configuración de suscripción
	c.mu.Lock()
	if subMsg.IncludeStatus != nil {
		c.includeStatus = *subMsg.IncludeStatus
	}
	if subMsg.ThrottleMs != nil {
		c.throttleMs = *subMsg.ThrottleMs
	}
	c.mu.Unlock()

	// Crear suscripciones en el router por cada stream
	for _, streamFilter := range subMsg.Streams {
		// Convertir StreamFilter a SubscriptionFilter del router
		filter := router.SubscriptionFilter{
			TenantID: &c.TenantID,
		}

		// Mapear Kind
		if streamFilter.Kind != "" {
			kind := domain.StreamKind(streamFilter.Kind)
			filter.Kind = &kind
		}

		// Mapear SiteID
		if streamFilter.SiteID != "" {
			filter.SiteID = &streamFilter.SiteID
		}

		// Mapear CageID
		if streamFilter.CageID != nil {
			filter.CageID = streamFilter.CageID
		}

		// Suscribir en el router
		sub, err := c.Hub.router.Subscribe(c.ID, filter)
		if err != nil {
			c.sendError("SUB_FAILED", "Failed to subscribe: "+err.Error())
			continue
		}

		// Marcar IncludeStatus en la suscripción del router
		if subMsg.IncludeStatus != nil && *subMsg.IncludeStatus {
			sub.IncludeStatus = true
		}

		// Guardar suscripción
		c.mu.Lock()
		if c.subscriptions == nil {
			c.subscriptions = make(map[string]*ClientSubscription)
		}
		c.subscriptions[sub.ID] = &ClientSubscription{
			RouterSubID:   sub.ID,
			IncludeStatus: c.includeStatus,
			CreatedAt:     time.Now(),
		}
		c.mu.Unlock()
	}

	// Enviar ACK
	c.Send <- AckMessage{
		Type:    MessageTypeACK,
		Message: "Subscribed successfully",
		Data: map[string]interface{}{
			"streams":        len(subMsg.Streams),
			"include_status": c.includeStatus,
		},
	}
}

// handleUnsubscribe maneja cancelación de suscripciones
func (c *Client) handleUnsubscribe(rawMsg map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Desuscribir todas las suscripciones
	for subID := range c.subscriptions {
		c.Hub.router.Unsubscribe(subID)
		delete(c.subscriptions, subID)
	}

	c.Send <- AckMessage{
		Type:    MessageTypeACK,
		Message: "Unsubscribed successfully",
	}
}

// sendError envía un mensaje de error al cliente
func (c *Client) sendError(code, message string) {
	c.Send <- ErrorMessage{
		Type:    MessageTypeERROR,
		Code:    code,
		Message: message,
	}
}

// writePump maneja el envío de mensajes al WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("Error escribiendo mensaje: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// generateMessageID genera un ID único para mensajes (legacy)
func generateMessageID() string {
	return time.Now().Format("20060102150405") + "-" +
		string(rune(time.Now().Nanosecond()%1000))
}
