# 🍃 MongoDB Integration - Go Backend

## 📋 **MongoDB Setup Completado**

### ✅ **Base de Datos Configurada**

- **Nombre:** `omniapi`
- **Conexión:** `mongodb://localhost:27017` (configurable)
- **Driver:** MongoDB Go Driver oficial
- **Timeout:** 10 segundos (configurable)

### 🏗️ **Arquitectura Implementada**

```
📁 database/
  └── mongodb.go          # Conexión y utilidades MongoDB

📁 models/
  └── models.go           # Modelos de datos y estructuras

📁 services/
  ├── user_service.go     # CRUD completo de usuarios
  └── message_service.go  # CRUD completo de mensajes

📁 handlers/
  └── mongodb_handlers.go # Endpoints HTTP para API REST
```

## 🗄️ **Modelos de Datos**

### **User (usuarios)**

```json
{
  "_id": "ObjectId",
  "username": "string",
  "email": "string",
  "password": "string (hash)",
  "full_name": "string",
  "avatar": "string (URL)",
  "status": "active|inactive|banned|deleted",
  "role": "admin|user|moderator",
  "created_at": "datetime",
  "updated_at": "datetime",
  "last_login": "datetime",
  "metadata": {}
}
```

### **Message (mensajes)**

```json
{
  "_id": "ObjectId",
  "type": "chat|system|notification",
  "content": "string",
  "from_user": "ObjectId",
  "to_user": "ObjectId (opcional)",
  "channel": "general|private|custom",
  "created_at": "datetime",
  "read_by": ["ObjectId"],
  "metadata": {}
}
```

### **Otros Modelos**

- `Session` - Sesiones de usuario
- `APILog` - Logs de peticiones API
- `WSConnection` - Conexiones WebSocket activas
- `Setting` - Configuraciones del sistema

## 🌐 **API Endpoints MongoDB**

### **👥 Users API**

| Método | Endpoint                   | Descripción                    |
| ------ | -------------------------- | ------------------------------ |
| GET    | `/api/users`               | Lista paginada de usuarios     |
| POST   | `/api/users/create`        | Crear nuevo usuario            |
| GET    | `/api/users/get?id=xxx`    | Obtener usuario por ID         |
| PUT    | `/api/users/update?id=xxx` | Actualizar usuario             |
| DELETE | `/api/users/delete?id=xxx` | Eliminar usuario (soft delete) |

### **💬 Messages API**

| Método | Endpoint               | Descripción                |
| ------ | ---------------------- | -------------------------- |
| GET    | `/api/messages`        | Lista paginada de mensajes |
| POST   | `/api/messages/create` | Crear nuevo mensaje        |

### **📊 Database API**

| Método | Endpoint              | Descripción                      |
| ------ | --------------------- | -------------------------------- |
| GET    | `/api/database/stats` | Estadísticas de la base de datos |

## 🔧 **Ejemplos de Uso**

### **Crear Usuario**

```bash
curl -X POST http://localhost:3000/api/users/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "juan123",
    "email": "juan@email.com",
    "full_name": "Juan Pérez",
    "password": "mi_password_hash"
  }'
```

### **Obtener Usuarios (Paginado)**

```bash
curl "http://localhost:3000/api/users?page=1&per_page=10&status=active"
```

### **Crear Mensaje**

```bash
curl -X POST http://localhost:3000/api/messages/create \
  -H "Content-Type: application/json" \
  -d '{
    "type": "chat",
    "content": "Hola, ¿cómo están todos?",
    "from_user": "673c123456789abcdef12345",
    "channel": "general"
  }'
```

### **Obtener Mensajes por Canal**

```bash
curl "http://localhost:3000/api/messages?channel=general&page=1&per_page=20"
```

## 📋 **Respuestas API Estandarizadas**

### **Éxito**

```json
{
  "success": true,
  "message": "Operación exitosa",
  "data": {
    /* datos */
  },
  "timestamp": 1699363200
}
```

### **Éxito con Paginación**

```json
{
  "success": true,
  "message": "Datos obtenidos",
  "data": [
    /* array de datos */
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  },
  "timestamp": 1699363200
}
```

### **Error**

```json
{
  "success": false,
  "message": "Descripción del error",
  "errors": [
    {
      "field": "username",
      "message": "Username es requerido"
    }
  ],
  "timestamp": 1699363200
}
```

## 🛠️ **Servicios Implementados**

### **UserService**

- ✅ `Create(user)` - Crear usuario
- ✅ `GetByID(id)` - Obtener por ID
- ✅ `GetByUsername(username)` - Obtener por username
- ✅ `GetByEmail(email)` - Obtener por email
- ✅ `Update(id, updates)` - Actualizar campos
- ✅ `Delete(id)` - Soft delete
- ✅ `List(page, perPage, filter)` - Lista paginada
- ✅ `ExistsByUsername(username)` - Verificar existencia
- ✅ `ExistsByEmail(email)` - Verificar existencia
- ✅ `UpdateLastLogin(id)` - Actualizar último login
- ✅ `GetActiveUsers()` - Usuarios activos
- ✅ `SearchUsers(query)` - Búsqueda por texto

### **MessageService**

- ✅ `Create(message)` - Crear mensaje
- ✅ `GetByID(id)` - Obtener por ID
- ✅ `GetByChannel(channel)` - Mensajes por canal
- ✅ `GetRecentMessages(limit)` - Mensajes recientes
- ✅ `GetUserMessages(userID)` - Mensajes de usuario
- ✅ `GetPrivateMessages(user1, user2)` - Mensajes privados
- ✅ `MarkAsRead(messageID, userID)` - Marcar como leído
- ✅ `Delete(id)` - Eliminar mensaje
- ✅ `GetMessageStats()` - Estadísticas de mensajes

## ⚙️ **Configuración**

### **Variables de Entorno (.env)**

```env
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=omniapi
MONGODB_TIMEOUT=10s
```

### **Conexión Automática**

- ✅ Conexión automática al iniciar servidor
- ✅ Health check de MongoDB
- ✅ Reconnección automática
- ✅ Cierre graceful con SIGTERM
- ✅ Logs detallados de conexión

## 📊 **Características Avanzadas**

### **Paginación Inteligente**

- Parámetros: `page`, `per_page`
- Límites: máximo 100 elementos por página
- Información completa de navegación

### **Filtros Dinámicos**

- Filtros por campos específicos
- Búsqueda por texto con regex
- Exclusión de registros eliminados

### **Validaciones**

- Validación de ObjectIDs
- Verificación de campos requeridos
- Prevención de duplicados (username/email)

### **Optimizaciones**

- Timeouts configurables
- Conexión pool automático
- Índices de base de datos (preparado)

## 🔮 **Próximas Mejoras**

- [ ] Autenticación JWT completa
- [ ] Middleware de autorización por roles
- [ ] Índices de MongoDB optimizados
- [ ] Rate limiting por usuario
- [ ] Logs de auditoría completos
- [ ] Backup y restore automático
- [ ] Métricas de performance
- [ ] Cache con Redis

## 📝 **Testing**

```bash
# Probar conexión a MongoDB
curl http://localhost:3000/api/database/stats

# Crear usuario de prueba
curl -X POST http://localhost:3000/api/users/create \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@test.com","full_name":"Usuario Test"}'

# Ver usuarios
curl http://localhost:3000/api/users
```

---

**🎉 ¡Tu backend Go ahora tiene integración completa con MongoDB!**

La base de datos `omniapi` está lista para recibir datos de cualquier frontend que necesites conectar. Todos los endpoints están documentados y funcionando.
