package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	MessageTypeChat         = "chat"
	MessageTypeNotification = "notification"
	MessageTypeSystem       = "system"
	MessageTypeHeartbeat    = "heartbeat"
	MessageTypeUserJoined   = "user_joined"
	MessageTypeUserLeft     = "user_left"
	MessageTypeUserList     = "user_list"
)

// Message estructura para mensajes WebSocket
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
	Username string
	Conn     *websocket.Conn
	Send     chan Message
	Hub      *Hub
}

// Hub mantiene el conjunto de clientes activos y transmite mensajes
type Hub struct {
	// Clientes registrados
	clients map[*Client]bool

	// Mensajes entrantes de los clientes
	broadcast chan Message

	// Registrar solicitudes de los clientes
	register chan *Client

	// Cancelar registro de solicitudes de clientes
	unregister chan *Client

	// Mutex para acceso seguro a clients map
	mutex sync.RWMutex

	// Estadísticas
	stats struct {
		TotalConnections   int64
		CurrentConnections int64
		MessagesSent       int64
		MessagesReceived   int64
	}
}

// NewHub crea una nueva instancia de Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run ejecuta el hub en un goroutine
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registra un nuevo cliente
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true
	h.stats.TotalConnections++
	h.stats.CurrentConnections++

	log.Printf("Cliente conectado: %s (%s) - Total: %d",
		client.Username, client.ID, h.stats.CurrentConnections)

	// Notificar a otros clientes
	userJoinedMsg := Message{
		Type:      MessageTypeUserJoined,
		Content:   client.Username + " se ha unido al chat",
		From:      "system",
		Timestamp: time.Now().Unix(),
		ID:        generateMessageID(),
		Data: map[string]interface{}{
			"username": client.Username,
			"userId":   client.ID,
		},
	}

	// Enviar lista de usuarios al nuevo cliente
	userList := h.getUserList()
	userListMsg := Message{
		Type:      MessageTypeUserList,
		Content:   "Lista de usuarios conectados",
		From:      "system",
		Timestamp: time.Now().Unix(),
		ID:        generateMessageID(),
		Data:      userList,
	}

	select {
	case client.Send <- userListMsg:
	default:
		close(client.Send)
		delete(h.clients, client)
	}

	// Broadcast user joined message a otros clientes
	for c := range h.clients {
		if c != client {
			select {
			case c.Send <- userJoinedMsg:
			default:
				close(c.Send)
				delete(h.clients, c)
			}
		}
	}
}

// unregisterClient desregistra un cliente
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.Send)
		h.stats.CurrentConnections--

		log.Printf("Cliente desconectado: %s (%s) - Total: %d",
			client.Username, client.ID, h.stats.CurrentConnections)

		// Notificar a otros clientes
		userLeftMsg := Message{
			Type:      MessageTypeUserLeft,
			Content:   client.Username + " ha salido del chat",
			From:      "system",
			Timestamp: time.Now().Unix(),
			ID:        generateMessageID(),
			Data: map[string]interface{}{
				"username": client.Username,
				"userId":   client.ID,
			},
		}

		for c := range h.clients {
			select {
			case c.Send <- userLeftMsg:
			default:
				close(c.Send)
				delete(h.clients, c)
			}
		}
	}
}

// broadcastMessage envía un mensaje a todos los clientes
func (h *Hub) broadcastMessage(message Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	h.stats.MessagesSent++

	for client := range h.clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.clients, client)
		}
	}
}

// getUserList obtiene la lista de usuarios conectados
func (h *Hub) getUserList() []map[string]interface{} {
	var users []map[string]interface{}

	for client := range h.clients {
		users = append(users, map[string]interface{}{
			"id":       client.ID,
			"username": client.Username,
			"status":   "online",
		})
	}

	return users
}

// GetStats retorna las estadísticas del hub
func (h *Hub) GetStats() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return map[string]interface{}{
		"total_connections":   h.stats.TotalConnections,
		"current_connections": h.stats.CurrentConnections,
		"messages_sent":       h.stats.MessagesSent,
		"messages_received":   h.stats.MessagesReceived,
		"active_users":        len(h.clients),
	}
}

// SendToUser envía un mensaje a un usuario específico
func (h *Hub) SendToUser(userID string, message Message) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.ID == userID {
			select {
			case client.Send <- message:
				return true
			default:
				return false
			}
		}
	}
	return false
}

// BroadcastSystemMessage envía un mensaje del sistema a todos
func (h *Hub) BroadcastSystemMessage(content string, data interface{}) {
	message := Message{
		Type:      MessageTypeSystem,
		Content:   content,
		From:      "system",
		Timestamp: time.Now().Unix(),
		ID:        generateMessageID(),
		Data:      data,
	}

	h.broadcast <- message
}

// readPump maneja la lectura de mensajes del WebSocket
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Configurar timeouts
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message Message
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Agregar metadata al mensaje
		message.From = c.Username
		message.Timestamp = time.Now().Unix()
		message.ID = generateMessageID()

		// Incrementar contador de mensajes recibidos
		c.Hub.mutex.Lock()
		c.Hub.stats.MessagesReceived++
		c.Hub.mutex.Unlock()

		// Enviar mensaje al hub para broadcast
		c.Hub.broadcast <- message
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

// generateMessageID genera un ID único para mensajes
func generateMessageID() string {
	return time.Now().Format("20060102150405") + "-" +
		string(rune(time.Now().Nanosecond()%1000))
}
