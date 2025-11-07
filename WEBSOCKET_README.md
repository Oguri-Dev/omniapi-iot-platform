# 🚀 Go WebSocket Backend - Documentación Completa

## 📋 Funcionalidades WebSocket Implementadas

### ✅ **Hub de WebSocket Centralizado**

- Gestión de múltiples conexiones simultáneas
- Broadcasting de mensajes a todos los clientes
- Seguimiento de usuarios conectados
- Estadísticas en tiempo real

### ✅ **Tipos de Mensajes Soportados**

- `chat` - Mensajes de chat entre usuarios
- `notification` - Notificaciones del sistema
- `system` - Mensajes del sistema
- `heartbeat` - Keep-alive del cliente
- `user_joined` - Notificación de usuario conectado
- `user_left` - Notificación de usuario desconectado
- `user_list` - Lista de usuarios conectados

### ✅ **Endpoints WebSocket**

| Endpoint     | Tipo      | Descripción                           |
| ------------ | --------- | ------------------------------------- |
| `/ws`        | WebSocket | Conexión principal WebSocket          |
| `/ws/test`   | HTTP      | Cliente de prueba integrado           |
| `/ws/stats`  | HTTP      | Estadísticas JSON del WebSocket       |
| `/websocket` | HTTP      | Página de integración y documentación |

## 🔧 **Cómo Conectar tu Frontend**

### **JavaScript Vanilla**

```javascript
// Conectar al WebSocket
const ws = new WebSocket('ws://localhost:3000/ws?username=Usuario1&userId=123')

// Manejar conexión abierta
ws.onopen = function (event) {
  console.log('Conectado al WebSocket')
}

// Recibir mensajes
ws.onmessage = function (event) {
  const message = JSON.parse(event.data)
  handleMessage(message)
}

// Enviar mensaje
const mensaje = {
  type: 'chat',
  content: 'Hola desde el frontend!',
  timestamp: Date.now(),
}
ws.send(JSON.stringify(mensaje))
```

### **React Hook**

```javascript
import { useState, useEffect, useRef } from 'react'

function useWebSocket(username) {
  const [messages, setMessages] = useState([])
  const [connected, setConnected] = useState(false)
  const ws = useRef(null)

  useEffect(() => {
    const wsUrl = `ws://localhost:3000/ws?username=${username}&userId=${Date.now()}`
    ws.current = new WebSocket(wsUrl)

    ws.current.onopen = () => setConnected(true)
    ws.current.onclose = () => setConnected(false)
    ws.current.onmessage = (event) => {
      const message = JSON.parse(event.data)
      setMessages((prev) => [...prev, message])
    }

    return () => ws.current.close()
  }, [username])

  const sendMessage = (content) => {
    if (connected && ws.current) {
      const message = {
        type: 'chat',
        content,
        timestamp: Date.now(),
      }
      ws.current.send(JSON.stringify(message))
    }
  }

  return { messages, connected, sendMessage }
}
```

### **Vue.js Composable**

```javascript
import { ref, onMounted, onUnmounted } from 'vue'

export function useWebSocket(username) {
  const messages = ref([])
  const connected = ref(false)
  let ws = null

  onMounted(() => {
    const wsUrl = `ws://localhost:3000/ws?username=${username}&userId=${Date.now()}`
    ws = new WebSocket(wsUrl)

    ws.onopen = () => (connected.value = true)
    ws.onclose = () => (connected.value = false)
    ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      messages.value.push(message)
    }
  })

  onUnmounted(() => {
    if (ws) ws.close()
  })

  const sendMessage = (content) => {
    if (connected.value && ws) {
      const message = {
        type: 'chat',
        content,
        timestamp: Date.now(),
      }
      ws.send(JSON.stringify(message))
    }
  }

  return { messages, connected, sendMessage }
}
```

## 📊 **Estructura de Mensajes**

### **Mensaje Entrante (del Frontend)**

```json
{
  "type": "chat",
  "content": "Hola, ¿cómo están?",
  "timestamp": 1699363200000
}
```

### **Mensaje Saliente (del Backend)**

```json
{
  "type": "chat",
  "content": "Hola, ¿cómo están?",
  "from": "Usuario1",
  "timestamp": 1699363200,
  "id": "20231107142000-123",
  "data": null
}
```

### **Mensaje de Sistema**

```json
{
  "type": "user_joined",
  "content": "Usuario2 se ha unido al chat",
  "from": "system",
  "timestamp": 1699363200,
  "id": "20231107142000-124",
  "data": {
    "username": "Usuario2",
    "userId": "user_456"
  }
}
```

## 🏗️ **Arquitectura del Sistema**

```
Frontend (React/Vue/Angular/Vanilla JS)
    ↓ WebSocket Connection
Backend Go Server
    ├── WebSocket Hub (Gorilla WebSocket)
    ├── Client Manager
    ├── Message Broadcasting
    └── Statistics Tracking
```

## 🧪 **Pruebas y Testing**

### **1. Cliente de Prueba Integrado**

- Accede a: `http://localhost:3000/ws/test`
- Prueba mensajes en tiempo real
- Multiple usuarios simultáneos

### **2. Estadísticas en Tiempo Real**

- Endpoint: `http://localhost:3000/ws/stats`
- Métricas de conexiones activas
- Contador de mensajes

### **3. Página de Integración**

- Accede a: `http://localhost:3000/websocket`
- Documentación completa
- Ejemplos de código

## 🔒 **Configuración de Seguridad**

### **CORS (Cross-Origin Resource Sharing)**

```go
// En websocket/hub.go - línea 15
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        // PERMITIR SOLO DOMINIOS ESPECÍFICOS EN PRODUCCIÓN
        return true  // Cambiar por validación específica
    },
}
```

### **Autenticación (Opcional)**

```go
// Ejemplo de validación de token
func WSHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    if !validateToken(token) {
        http.Error(w, "Token inválido", http.StatusUnauthorized)
        return
    }
    // ... resto del código
}
```

## 📈 **Monitoreo y Métricas**

### **Estadísticas Disponibles**

- `total_connections` - Total de conexiones desde el inicio
- `current_connections` - Conexiones activas actuales
- `messages_sent` - Total de mensajes enviados
- `messages_received` - Total de mensajes recibidos
- `active_users` - Número de usuarios únicos conectados

### **Logs del Sistema**

```
Cliente conectado: Usuario1 (user_123) - Total: 1
Cliente desconectado: Usuario1 (user_123) - Total: 0
WebSocket error: websocket: close 1006 (abnormal closure)
```

## 🚀 **Casos de Uso Comunes**

### **1. Chat en Tiempo Real**

- Múltiples usuarios
- Mensajes instantáneos
- Notificaciones de entrada/salida

### **2. Notificaciones Push**

- Alertas del sistema
- Updates de estado
- Mensajes administrativos

### **3. Colaboración en Tiempo Real**

- Editores colaborativos
- Actualizaciones de documentos
- Sincronización de estado

### **4. Gaming/Aplicaciones Interactivas**

- Estados de juego
- Movimientos de jugadores
- Actualizaciones de score

## ⚡ **Optimizaciones y Best Practices**

### **1. Gestión de Memoria**

- Channels con buffer limitado
- Cleanup automático de conexiones
- Timeouts configurables

### **2. Performance**

- Goroutines para cada cliente
- Broadcasting eficiente
- Heartbeat para keep-alive

### **3. Escalabilidad**

- Hub centralizado
- Estadísticas thread-safe
- Múltiples instancias (con Redis en futuro)

## 🔧 **Próximas Mejoras Sugeridas**

- [ ] Persistencia de mensajes (Database)
- [ ] Salas/Canales específicos
- [ ] Autenticación JWT
- [ ] Rate limiting
- [ ] Integración con Redis para múltiples instancias
- [ ] Métricas con Prometheus
- [ ] SSL/TLS para WebSocket Secure (WSS)

---

**¡Tu backend Go ahora soporta WebSockets completamente funcionales!** 🎉
