package websocket

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

// WSHandler maneja las conexiones WebSocket
func WSHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	// Obtener parámetros de query
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Usuario" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	userID := r.URL.Query().Get("userId")
	if userID == "" {
		userID = "user_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	// Crear nuevo cliente
	client := &Client{
		ID:       userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan Message, 256),
		Hub:      hub,
	}

	// Registrar cliente en el hub
	client.Hub.register <- client

	// Iniciar goroutines para lectura y escritura
	go client.writePump()
	go client.readPump()
}

// WSStatsHandler proporciona estadísticas del WebSocket via HTTP
func WSStatsHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	stats := hub.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Manual JSON encoding para evitar import
	jsonResp := `{
		"success": true,
		"message": "Estadísticas del WebSocket",
		"data": {
			"total_connections": ` + strconv.FormatInt(stats["total_connections"].(int64), 10) + `,
			"current_connections": ` + strconv.FormatInt(stats["current_connections"].(int64), 10) + `,
			"messages_sent": ` + strconv.FormatInt(stats["messages_sent"].(int64), 10) + `,
			"messages_received": ` + strconv.FormatInt(stats["messages_received"].(int64), 10) + `,
			"active_users": ` + strconv.Itoa(stats["active_users"].(int)) + `
		},
		"timestamp": ` + strconv.FormatInt(time.Now().Unix(), 10) + `
	}`

	w.Write([]byte(jsonResp))
}

// WSTestHandler proporciona una página de prueba del WebSocket
func WSTestHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="es">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>WebSocket Test - Go Backend</title>
		<style>
			* {
				margin: 0;
				padding: 0;
				box-sizing: border-box;
			}
			body {
				font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
				background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
				min-height: 100vh;
				display: flex;
				align-items: center;
				justify-content: center;
			}
			.container {
				background: white;
				border-radius: 15px;
				box-shadow: 0 8px 32px rgba(0,0,0,0.2);
				width: 90%;
				max-width: 800px;
				height: 600px;
				display: flex;
				flex-direction: column;
				overflow: hidden;
			}
			.header {
				background: #667eea;
				color: white;
				padding: 20px;
				text-align: center;
			}
			.status {
				padding: 10px 20px;
				background: #f8f9fa;
				border-bottom: 1px solid #dee2e6;
				font-size: 14px;
			}
			.status.connected {
				background: #d4edda;
				color: #155724;
			}
			.status.disconnected {
				background: #f8d7da;
				color: #721c24;
			}
			.chat-area {
				flex: 1;
				padding: 20px;
				overflow-y: auto;
				background: #f8f9fa;
			}
			.message {
				margin-bottom: 10px;
				padding: 10px;
				border-radius: 8px;
				max-width: 70%;
			}
			.message.own {
				background: #007bff;
				color: white;
				margin-left: auto;
			}
			.message.other {
				background: white;
				border: 1px solid #dee2e6;
			}
			.message.system {
				background: #ffeaa7;
				color: #6c757d;
				text-align: center;
				max-width: 100%;
				font-style: italic;
			}
			.input-area {
				padding: 20px;
				border-top: 1px solid #dee2e6;
				display: flex;
				gap: 10px;
			}
			.username-input, .message-input {
				padding: 10px;
				border: 1px solid #ced4da;
				border-radius: 5px;
			}
			.username-input {
				flex: 0 0 150px;
			}
			.message-input {
				flex: 1;
			}
			button {
				padding: 10px 20px;
				background: #28a745;
				color: white;
				border: none;
				border-radius: 5px;
				cursor: pointer;
			}
			button:hover {
				background: #218838;
			}
			button:disabled {
				background: #6c757d;
				cursor: not-allowed;
			}
			.connect-btn {
				background: #007bff;
			}
			.connect-btn:hover {
				background: #0056b3;
			}
			.stats {
				font-size: 12px;
				color: #6c757d;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>🚀 WebSocket Test Client</h1>
				<p>Prueba la conexión WebSocket con tu backend Go</p>
			</div>
			
			<div id="status" class="status disconnected">
				❌ Desconectado - Ingresa tu nombre y presiona Conectar
			</div>
			
			<div class="chat-area" id="chatArea">
				<div class="message system">
					Bienvenido al chat WebSocket. Conectate para comenzar.
				</div>
			</div>
			
			<div class="input-area">
				<input type="text" id="usernameInput" class="username-input" placeholder="Tu nombre" value="Usuario1">
				<input type="text" id="messageInput" class="message-input" placeholder="Escribe un mensaje..." disabled>
				<button id="connectBtn" class="connect-btn" onclick="toggleConnection()">Conectar</button>
				<button id="sendBtn" onclick="sendMessage()" disabled>Enviar</button>
			</div>
		</div>

		<script>
			let ws = null;
			let connected = false;
			let username = 'Usuario1';

			function toggleConnection() {
				if (connected) {
					disconnect();
				} else {
					connect();
				}
			}

			function connect() {
				username = document.getElementById('usernameInput').value || 'Usuario1';
				const userId = 'user_' + Date.now();
				
				// Usar el puerto actual de la página
				const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
				const wsUrl = protocol + '//' + window.location.host + '/ws?username=' + 
							  encodeURIComponent(username) + '&userId=' + userId;
				
				ws = new WebSocket(wsUrl);
				
				ws.onopen = function(event) {
					connected = true;
					updateStatus('✅ Conectado como: ' + username, 'connected');
					document.getElementById('connectBtn').textContent = 'Desconectar';
					document.getElementById('connectBtn').className = 'connect-btn';
					document.getElementById('messageInput').disabled = false;
					document.getElementById('sendBtn').disabled = false;
					document.getElementById('usernameInput').disabled = true;
					
					addSystemMessage('Conectado al servidor WebSocket');
				};
				
				ws.onmessage = function(event) {
					const message = JSON.parse(event.data);
					handleMessage(message);
				};
				
				ws.onclose = function(event) {
					connected = false;
					updateStatus('❌ Desconectado', 'disconnected');
					document.getElementById('connectBtn').textContent = 'Conectar';
					document.getElementById('connectBtn').className = 'connect-btn';
					document.getElementById('messageInput').disabled = true;
					document.getElementById('sendBtn').disabled = true;
					document.getElementById('usernameInput').disabled = false;
					
					addSystemMessage('Desconectado del servidor');
				};
				
				ws.onerror = function(error) {
					console.error('WebSocket error:', error);
					addSystemMessage('Error de conexión');
				};
			}

			function disconnect() {
				if (ws) {
					ws.close();
				}
			}

			function sendMessage() {
				const messageInput = document.getElementById('messageInput');
				const content = messageInput.value.trim();
				
				if (content && connected) {
					const message = {
						type: 'chat',
						content: content,
						timestamp: Date.now()
					};
					
					ws.send(JSON.stringify(message));
					messageInput.value = '';
					
					// Mostrar mensaje propio
					addMessage(content, username, true);
				}
			}

			function handleMessage(message) {
				switch(message.type) {
					case 'chat':
						if (message.from !== username) {
							addMessage(message.content, message.from, false);
						}
						break;
					case 'system':
					case 'user_joined':
					case 'user_left':
						addSystemMessage(message.content);
						break;
					case 'user_list':
						console.log('Usuarios conectados:', message.data);
						break;
					default:
						console.log('Mensaje recibido:', message);
				}
			}

			function addMessage(content, from, isOwn) {
				const chatArea = document.getElementById('chatArea');
				const messageDiv = document.createElement('div');
				messageDiv.className = 'message ' + (isOwn ? 'own' : 'other');
				
				const time = new Date().toLocaleTimeString();
				messageDiv.innerHTML = '<strong>' + from + '</strong> (' + time + ')<br>' + content;
				
				chatArea.appendChild(messageDiv);
				chatArea.scrollTop = chatArea.scrollHeight;
			}

			function addSystemMessage(content) {
				const chatArea = document.getElementById('chatArea');
				const messageDiv = document.createElement('div');
				messageDiv.className = 'message system';
				messageDiv.textContent = content;
				
				chatArea.appendChild(messageDiv);
				chatArea.scrollTop = chatArea.scrollHeight;
			}

			function updateStatus(text, className) {
				const status = document.getElementById('status');
				status.textContent = text;
				status.className = 'status ' + className;
			}

			// Enviar mensaje con Enter
			document.getElementById('messageInput').addEventListener('keypress', function(e) {
				if (e.key === 'Enter') {
					sendMessage();
				}
			});

			// Conectar con Enter en username
			document.getElementById('usernameInput').addEventListener('keypress', function(e) {
				if (e.key === 'Enter' && !connected) {
					connect();
				}
			});
		</script>
	</body>
	</html>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
