# Auto-creación de Usuario Administrador

## 🔐 Funcionalidad

El sistema ahora verifica automáticamente al iniciar si existe al menos un usuario administrador en la base de datos. Si no encuentra ninguno, crea uno por defecto.

## ✨ Características

- ✅ **Verificación automática** al iniciar el servidor
- ✅ **No crea duplicados** - Solo crea si no existe ningún admin
- ✅ **Logs informativos** - Muestra claramente qué acción tomó
- ✅ **Seguridad** - Contraseña hasheada con bcrypt

## 📝 Comportamiento

### Cuando existe al menos un admin:

```
🔐 Checking admin user...
✅ Admin user already exists (count: 1)
```

### Cuando NO existe ningún admin:

```
🔐 Checking admin user...
⚠️  No admin user found. Creating default admin user...
✅ Default admin user created successfully!
   ID: 691608c90b95bd1662ef27f0
   Username: admin
   Email: admin@omniapi.com
   Password: admin123
   ⚠️  IMPORTANT: Change this password after first login!
```

## 🔑 Credenciales por Defecto

Cuando se crea automáticamente:

- **Usuario:** `admin`
- **Contraseña:** `admin123`
- **Email:** `admin@omniapi.com`
- **Role:** `admin`

## ⚠️ Recomendaciones de Seguridad

1. **Cambiar la contraseña** después del primer login
2. **Crear otros usuarios admin** con credenciales únicas
3. **Eliminar el usuario default** si creaste otros admins
4. **No usar estas credenciales** en producción sin cambiarlas

## 🛠️ Implementación Técnica

La función `EnsureAdminUser()` se encuentra en:

```
services/user_service.go
```

Se ejecuta en `main.go` después de conectar a MongoDB:

```go
// Asegurar que existe un usuario administrador
fmt.Println("\n🔐 Checking admin user...")
if err := services.EnsureAdminUser(); err != nil {
    log.Printf("⚠️  Warning: Could not ensure admin user: %v", err)
}
```

## 🧪 Probar la Funcionalidad

### 1. Eliminar todos los admins de la DB (solo para pruebas):

```javascript
// En MongoDB shell o Compass
db.users.deleteMany({ role: 'admin' })
```

### 2. Reiniciar el servidor:

```bash
go run main.go
```

### 3. Verificar el output:

Deberías ver el mensaje de creación del usuario admin por defecto.

### 4. Verificar en la base de datos:

```javascript
db.users.find({ role: 'admin' })
```

## 📦 Casos de Uso

Esta funcionalidad es útil para:

- ✅ **Primera instalación** - No necesitas ejecutar scripts adicionales
- ✅ **Recuperación** - Si pierdes acceso a todos los admins
- ✅ **Desarrollo** - Siempre tendrás un admin disponible
- ✅ **Testing** - Ambiente limpio siempre tiene un admin
- ✅ **Docker/K8s** - Contenedores nuevos tienen acceso inmediato

## 🔄 Flujo de Inicio del Servidor

```
1. Cargar configuración
2. Conectar a MongoDB
   ↓
3. 🆕 Verificar/Crear Admin
   ├─ Buscar usuarios con role="admin"
   ├─ Si count > 0: ✅ No hacer nada
   └─ Si count = 0: 🔨 Crear admin por defecto
   ↓
4. Inicializar Router
5. Inicializar Requesters
6. Inicializar WebSocket
7. Servidor listo
```

## 🐛 Troubleshooting

### Error: "could not ensure admin user"

- Verifica que MongoDB esté corriendo
- Verifica los permisos de escritura en la colección `users`
- Revisa los logs para más detalles del error

### El admin no se crea

- Verifica que no exista ya un usuario con role="admin"
- Verifica que la colección `users` sea accesible
- Ejecuta manualmente: `db.users.find({ role: "admin" })`

### ¿Cómo desactivo esta funcionalidad?

Puedes comentar las líneas en `main.go`:

```go
// if err := services.EnsureAdminUser(); err != nil {
//     log.Printf("⚠️  Warning: Could not ensure admin user: %v", err)
// }
```

## 📚 Código Relacionado

- `services/user_service.go` - Función `EnsureAdminUser()`
- `main.go` - Llamada a la función
- `handlers/auth_handlers.go` - Login con usuario admin
- `scripts/create_test_user.go` - Script manual (ya no necesario)

---

**Nota:** Esta funcionalidad NO afecta usuarios existentes. Solo crea un admin si la base de datos no tiene ninguno.
