package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIResponse estructura estándar para respuestas de API
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// HealthHandler maneja las solicitudes de salud del servidor
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Message: "El servidor está funcionando correctamente",
		Data: map[string]interface{}{
			"status":  "healthy",
			"uptime":  "Running",
			"version": "1.0.0",
		},
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// InfoHandler proporciona información del servidor
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Message: "Información del servidor",
		Data: map[string]interface{}{
			"name":        "GoLang Web Server",
			"description": "Servidor web básico desarrollado en Go",
			"author":      "Desarrollador Go",
			"routes": []string{
				"GET /",
				"GET /api/health",
				"GET /api/info",
				"GET /api/time",
			},
		},
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// TimeHandler devuelve la hora actual del servidor
func TimeHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	response := APIResponse{
		Success: true,
		Message: "Hora actual del servidor",
		Data: map[string]interface{}{
			"current_time": now.Format(time.RFC3339),
			"unix_time":    now.Unix(),
			"formatted":    now.Format("2006-01-02 15:04:05"),
			"timezone":     now.Location().String(),
		},
		Timestamp: now.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// WSTestPageHandler sirve la página de prueba de WebSocket integrada
func WSTestPageHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="es">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>WebSocket Integration Test</title>
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
				padding: 20px;
			}
			.container {
				max-width: 1200px;
				margin: 0 auto;
				background: white;
				border-radius: 15px;
				box-shadow: 0 8px 32px rgba(0,0,0,0.2);
				overflow: hidden;
			}
			.header {
				background: #667eea;
				color: white;
				padding: 20px;
				text-align: center;
			}
			.content {
				padding: 30px;
			}
			.demo-grid {
				display: grid;
				grid-template-columns: 1fr 1fr;
				gap: 30px;
				margin-bottom: 30px;
			}
			.demo-section {
				background: #f8f9fa;
				padding: 20px;
				border-radius: 10px;
				border: 1px solid #dee2e6;
			}
			.demo-section h3 {
				color: #667eea;
				margin-bottom: 15px;
			}
			.btn {
				background: #28a745;
				color: white;
				border: none;
				padding: 10px 20px;
				border-radius: 5px;
				cursor: pointer;
				margin: 5px;
				display: inline-block;
				text-decoration: none;
			}
			.btn:hover {
				background: #218838;
			}
			.btn-primary {
				background: #007bff;
			}
			.btn-primary:hover {
				background: #0056b3;
			}
			.code-block {
				background: #2d3748;
				color: #e2e8f0;
				padding: 15px;
				border-radius: 8px;
				overflow-x: auto;
				font-family: 'Courier New', monospace;
				font-size: 14px;
				margin: 10px 0;
			}
			.endpoint-list {
				list-style: none;
			}
			.endpoint-list li {
				background: white;
				margin: 5px 0;
				padding: 10px;
				border-radius: 5px;
				border: 1px solid #dee2e6;
				font-family: monospace;
			}
			.method {
				background: #28a745;
				color: white;
				padding: 2px 8px;
				border-radius: 3px;
				font-size: 12px;
				margin-right: 10px;
			}
			.method.ws {
				background: #6f42c1;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>🚀 WebSocket Integration - Go Backend</h1>
				<p>Tu backend Go ahora soporta WebSockets para comunicación en tiempo real</p>
			</div>
			
			<div class="content">
				<div class="demo-grid">
					<div class="demo-section">
						<h3>🔗 Probar WebSocket</h3>
						<p>Abre el cliente de prueba para conectarte al WebSocket y enviar mensajes en tiempo real.</p>
						<a href="/ws/test" class="btn btn-primary" target="_blank">Abrir Cliente WebSocket</a>
						<div class="code-block">
// URL de conexión WebSocket:
ws://localhost:3000/ws?username=TuNombre&userId=123</div>
					</div>
					
					<div class="demo-section">
						<h3>📊 Estadísticas WebSocket</h3>
						<p>Ver métricas en tiempo real de las conexiones WebSocket activas.</p>
						<a href="/ws/stats" class="btn" target="_blank">Ver Estadísticas JSON</a>
						<div id="stats" style="margin-top: 10px; font-size: 14px;"></div>
					</div>
				</div>
				
				<div class="demo-section">
					<h3>🌐 Endpoints Disponibles</h3>
					<ul class="endpoint-list">
						<li><span class="method ws">WS</span> /ws - Conexión WebSocket principal</li>
						<li><span class="method">GET</span> /ws/test - Cliente de prueba WebSocket</li>
						<li><span class="method">GET</span> /ws/stats - Estadísticas de WebSocket</li>
						<li><span class="method">GET</span> /api/health - Estado del servidor</li>
						<li><span class="method">GET</span> /api/info - Información del servidor</li>
						<li><span class="method">GET</span> /api/time - Hora del servidor</li>
					</ul>
				</div>
				
				<div class="demo-section">
					<h3>📝 Ejemplo de Integración Frontend</h3>
					<p>Código JavaScript para conectar tu frontend:</p>
					<div class="code-block">
// Conectar al WebSocket
const ws = new WebSocket('ws://localhost:3000/ws?username=Usuario1');

// Manejar conexión abierta
ws.onopen = function(event) {
    console.log('Conectado al WebSocket');
};

// Recibir mensajes
ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Mensaje recibido:', message);
};

// Enviar mensaje
const mensaje = {
    type: 'chat',
    content: 'Hola desde el frontend!',
    timestamp: Date.now()
};
ws.send(JSON.stringify(mensaje));
					</div>
				</div>
				
				<div class="demo-section">
					<h3>🔧 Tipos de Mensajes Soportados</h3>
					<ul style="margin-left: 20px;">
						<li><strong>chat</strong> - Mensajes de chat entre usuarios</li>
						<li><strong>notification</strong> - Notificaciones del sistema</li>
						<li><strong>system</strong> - Mensajes del sistema</li>
						<li><strong>heartbeat</strong> - Keep-alive del cliente</li>
						<li><strong>user_joined</strong> - Usuario se conectó</li>
						<li><strong>user_left</strong> - Usuario se desconectó</li>
						<li><strong>user_list</strong> - Lista de usuarios conectados</li>
					</ul>
				</div>
			</div>
		</div>
		
		<script>
			// Cargar estadísticas cada 5 segundos
			function loadStats() {
				fetch('/ws/stats')
					.then(response => response.json())
					.then(data => {
						if (data && data.data) {
							const stats = data.data;
							document.getElementById('stats').innerHTML = 
								'🔗 Conexiones activas: ' + stats.current_connections + '<br>' +
								'📨 Mensajes enviados: ' + stats.messages_sent + '<br>' +
								'📬 Mensajes recibidos: ' + stats.messages_received;
						}
					})
					.catch(err => {
						document.getElementById('stats').innerHTML = 'Error cargando estadísticas';
					});
			}
			
			// Cargar estadísticas al inicio y cada 5 segundos
			loadStats();
			setInterval(loadStats, 5000);
		</script>
	</body>
	</html>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// HomeHandler maneja la página principal
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="es">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Proyecto Go - Dashboard</title>
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
				color: #333;
			}
			.container {
				max-width: 1200px;
				margin: 0 auto;
				padding: 20px;
			}
			.header {
				text-align: center;
				color: white;
				margin-bottom: 40px;
			}
			.header h1 {
				font-size: 3rem;
				margin-bottom: 10px;
			}
			.header p {
				font-size: 1.2rem;
				opacity: 0.9;
			}
			.dashboard {
				display: grid;
				grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
				gap: 20px;
				margin-bottom: 40px;
			}
			.card {
				background: white;
				border-radius: 15px;
				padding: 25px;
				box-shadow: 0 8px 32px rgba(0,0,0,0.1);
				transition: transform 0.3s ease;
			}
			.card:hover {
				transform: translateY(-5px);
			}
			.card h3 {
				color: #667eea;
				margin-bottom: 15px;
				font-size: 1.3rem;
			}
			.endpoints {
				background: white;
				border-radius: 15px;
				padding: 25px;
				box-shadow: 0 8px 32px rgba(0,0,0,0.1);
			}
			.endpoint {
				display: flex;
				align-items: center;
				justify-content: space-between;
				padding: 12px 0;
				border-bottom: 1px solid #eee;
			}
			.endpoint:last-child {
				border-bottom: none;
			}
			.method {
				background: #28a745;
				color: white;
				padding: 4px 12px;
				border-radius: 20px;
				font-size: 0.8rem;
				font-weight: bold;
			}
			.url {
				flex-grow: 1;
				margin: 0 15px;
				font-family: monospace;
				color: #666;
			}
			.test-btn {
				background: #007bff;
				color: white;
				border: none;
				padding: 6px 12px;
				border-radius: 5px;
				cursor: pointer;
				font-size: 0.8rem;
			}
			.test-btn:hover {
				background: #0056b3;
			}
			.footer {
				text-align: center;
				color: white;
				margin-top: 40px;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>🚀 Go Web Server</h1>
				<p>Servidor web moderno desarrollado en Go</p>
			</div>

			<div class="dashboard">
				<div class="card">
					<h3>📊 Estado del Servidor</h3>
					<p><strong>Estado:</strong> <span style="color: #28a745;">✅ Activo</span></p>
					<p><strong>Puerto:</strong> 8080</p>
					<p><strong>Versión:</strong> 1.0.0</p>
					<p><strong>Uptime:</strong> <span id="uptime">Calculando...</span></p>
				</div>

				<div class="card">
					<h3>🔧 Características</h3>
					<ul>
						<li>✅ API REST funcional</li>
						<li>✅ Configuración por variables de entorno</li>
						<li>✅ Estructura modular</li>
						<li>✅ Respuestas JSON estandarizadas</li>
					</ul>
				</div>

				<div class="card">
					<h3>📈 Métricas</h3>
					<p><strong>Endpoints disponibles:</strong> 4</p>
					<p><strong>Tiempo de respuesta:</strong> ~2ms</p>
					<p><strong>Memoria utilizada:</strong> Bajo</p>
				</div>
			</div>

			<div class="endpoints">
				<h3>🌐 Endpoints Disponibles</h3>
				
				<div class="endpoint">
					<span class="method">GET</span>
					<span class="url">/</span>
					<button class="test-btn" onclick="window.location.reload()">Probar</button>
				</div>

				<div class="endpoint">
					<span class="method">GET</span>
					<span class="url">/api/health</span>
					<button class="test-btn" onclick="testEndpoint('/api/health')">Probar</button>
				</div>

				<div class="endpoint">
					<span class="method">GET</span>
					<span class="url">/api/info</span>
					<button class="test-btn" onclick="testEndpoint('/api/info')">Probar</button>
				</div>

				<div class="endpoint">
					<span class="method">GET</span>
					<span class="url">/api/time</span>
					<button class="test-btn" onclick="testEndpoint('/api/time')">Probar</button>
				</div>
			</div>

			<div class="footer">
				<p>Desarrollado con ❤️ usando Go | Proyecto creado automáticamente</p>
			</div>
		</div>

		<script>
			function testEndpoint(url) {
				window.open(url, '_blank');
			}

			// Actualizar uptime cada segundo
			let startTime = Date.now();
			setInterval(() => {
				let uptime = Math.floor((Date.now() - startTime) / 1000);
				let minutes = Math.floor(uptime / 60);
				let seconds = uptime % 60;
				document.getElementById('uptime').textContent = minutes + 'm ' + seconds + 's';
			}, 1000);
		</script>
	</body>
	</html>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}
