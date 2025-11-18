# Setup Manual del Primer Administrador

## 🎯 Descripción

El sistema ahora requiere **configuración manual** del primer usuario administrador a través de un formulario web interactivo, en lugar de crear uno automáticamente.

## ✨ Características

- ✅ **Formulario interactivo** - Interfaz web amigable para crear el primer admin
- ✅ **Campos personalizables** - Define tu propio username, email, password y nombre completo
- ✅ **Validaciones en tiempo real** - Verifica que los datos sean válidos antes de enviar
- ✅ **Redirección automática** - Detecta si el sistema necesita setup y redirige automáticamente
- ✅ **Seguridad** - El endpoint solo funciona cuando NO existe ningún admin

## 🔄 Flujo de Funcionamiento

### 1. Primera vez que accedes al sistema

```
Usuario visita → http://localhost:5173
       ↓
Sistema verifica → GET /api/auth/setup/check
       ↓
Si needsSetup=true → Redirige a /setup
       ↓
Usuario completa formulario
       ↓
POST /api/auth/setup → Crea primer admin
       ↓
Redirige a /login con mensaje de éxito
       ↓
Usuario inicia sesión con las credenciales creadas
```

### 2. Si ya existe un administrador

```
Usuario visita → http://localhost:5173
       ↓
Sistema verifica → GET /api/auth/setup/check
       ↓
Si needsSetup=false → Muestra página de login normal
```

## 📝 Endpoints Backend

### GET `/api/auth/setup/check`

Verifica si el sistema necesita configuración inicial.

**Request:**

```bash
GET http://localhost:3000/api/auth/setup/check
```

**Response:**

```json
{
  "success": true,
  "message": "Setup status",
  "data": {
    "needsSetup": true // true si no hay admin, false si ya existe
  },
  "timestamp": 1699876543
}
```

### POST `/api/auth/setup`

Crea el primer usuario administrador. **Solo funciona si no existe ningún admin.**

**Request:**

```bash
POST http://localhost:3000/api/auth/setup
Content-Type: application/json

{
  "username": "admin",
  "email": "admin@omniapi.com",
  "password": "mi_password_seguro",
  "fullName": "Administrador Principal"
}
```

**Response (éxito):**

```json
{
  "success": true,
  "message": "Administrador creado exitosamente",
  "data": {
    "id": "655f1c2e8c4b2a1234567890",
    "username": "admin",
    "email": "admin@omniapi.com",
    "fullName": "Administrador Principal",
    "role": "admin",
    "status": "active"
  },
  "timestamp": 1699876543
}
```

**Response (error - ya existe admin):**

```json
{
  "success": false,
  "message": "Error creando administrador: ya existe un usuario administrador en el sistema",
  "timestamp": 1699876543
}
```

## 🖼️ Interfaz de Usuario

### Página de Setup (`/setup`)

- **Icono animado** 🚀 con efecto de rebote
- **Campos del formulario:**

  - Username (requerido)
  - Email (requerido)
  - Nombre Completo (opcional)
  - Contraseña (requerido, mínimo 6 caracteres)
  - Confirmar Contraseña (requerido)

- **Validaciones:**
  - ✅ Campos obligatorios no vacíos
  - ✅ Email con formato válido
  - ✅ Contraseña mínimo 6 caracteres
  - ✅ Contraseñas coinciden
  - ✅ Mensaje de advertencia sobre seguridad

### Página de Login (`/login`)

- Muestra mensaje de éxito cuando vienes desde `/setup`
- Permite iniciar sesión con las credenciales creadas

## 🧪 Cómo Probar el Flujo Completo

### Opción 1: Con MongoDB Compass (Recomendado)

1. **Abrir MongoDB Compass** y conectar a `mongodb://localhost:27017`
2. **Seleccionar la base de datos** `omniapi`
3. **Ir a la colección** `users`
4. **Eliminar todos los documentos** con `{role: "admin"}`
5. **Ir al navegador** → `http://localhost:5173`
6. **Deberías ver** el formulario de setup automáticamente
7. **Completar el formulario** con tus datos
8. **Click en "Crear Administrador"**
9. **Serás redirigido** a `/login` con mensaje de éxito
10. **Iniciar sesión** con las credenciales que creaste

### Opción 2: Con MongoDB Shell

```bash
# Conectar a la base de datos
mongosh omniapi

# Eliminar usuarios admin
db.users.deleteMany({ role: "admin" })

# Verificar
db.users.find({ role: "admin" }).count()  // Debe retornar 0

# Salir
exit
```

Luego abrir el navegador en `http://localhost:5173`

### Opción 3: Con script Go (si MongoDB Shell no está disponible)

Crear archivo `scripts/reset_admin.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	collection := client.Database("omniapi").Collection("users")
	result, err := collection.DeleteMany(ctx, bson.M{"role": "admin"})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Deleted %d admin user(s)\n", result.DeletedCount)
}
```

Ejecutar:

```bash
cd scripts
go run reset_admin.go
```

## 🔍 Verificar Estado del Sistema

### Backend Log

Cuando inicias el backend, verás uno de estos mensajes:

**Si hay admin:**

```
🔐 Checking admin user...
✅ Admin user exists
```

**Si NO hay admin:**

```
🔐 Checking admin user...
⚠️  No admin user found. Please complete setup via /api/auth/setup
```

### Frontend Behavior

1. **Abrir DevTools** (F12) en el navegador
2. **Ir a** `http://localhost:5173`
3. **Ver Network tab** → Buscar llamada a `/api/auth/setup/check`
4. **Response:**
   - `needsSetup: true` → Muestra formulario de setup
   - `needsSetup: false` → Muestra login normal

## 🛡️ Seguridad

### Protecciones Implementadas

1. **Endpoint protegido:** `/api/auth/setup` solo funciona si NO existe ningún admin
2. **Validación backend:** Verifica que no haya usuarios admin antes de crear
3. **Contraseña hasheada:** Usa bcrypt para almacenar passwords de forma segura
4. **Validación frontend:** Verifica formato de email, longitud de password, etc.
5. **Sin autenticación:** El endpoint de setup NO requiere token (es público, pero solo funciona una vez)

### Posibles Ataques y Mitigaciones

❌ **Ataque:** Alguien intenta crear admin cuando ya existe uno
✅ **Mitigación:** El backend retorna error "ya existe un usuario administrador"

❌ **Ataque:** Alguien intenta acceder a setup después de crear admin
✅ **Mitigación:** El sistema verifica y redirige al login

❌ **Ataque:** Fuerza bruta al endpoint de setup
✅ **Mitigación:** Una vez creado el admin, el endpoint deja de funcionar

## 📊 Comparación con Versión Anterior

| Aspecto           | Versión Anterior            | Versión Actual                  |
| ----------------- | --------------------------- | ------------------------------- |
| Creación admin    | Automática (admin/admin123) | Manual vía formulario web       |
| Credenciales      | Fijas y conocidas           | Personalizadas por el usuario   |
| Seguridad inicial | Baja (password por defecto) | Alta (usuario define password)  |
| Experiencia UX    | Script manual o auto        | Formulario web intuitivo        |
| Flexibilidad      | Ninguna                     | Total (username, email, nombre) |

## 🎨 Archivos Modificados/Creados

### Backend

- ✅ `services/user_service.go`
  - Renombrado: `EnsureAdminUser()` → `CheckAdminExists()`
  - Agregado: `CreateFirstAdmin()` función
- ✅ `handlers/auth_handlers.go`
  - Agregado: `CheckSetupHandler()` - GET /api/auth/setup/check
  - Agregado: `SetupHandler()` - POST /api/auth/setup
- ✅ `main.go`
  - Modificado: Llamada a `CheckAdminExists()` en lugar de `EnsureAdminUser()`
  - Agregado: Rutas para setup endpoints

### Frontend

- ✅ `src/pages/Setup.tsx` - Página de configuración inicial
- ✅ `src/styles/Setup.css` - Estilos del formulario de setup
- ✅ `src/services/setup.service.ts` - Servicio para verificar estado de setup
- ✅ `src/contexts/AuthContext.tsx` - Verificación automática de setup
- ✅ `src/pages/Login.tsx` - Mensaje de éxito desde setup
- ✅ `src/styles/Login.css` - Estilos para mensaje de éxito
- ✅ `src/App.tsx` - Ruta `/setup` agregada

## 📖 Casos de Uso

### Caso 1: Primera Instalación

```
Usuario instala OmniAPI → Inicia backend y frontend
→ Sistema detecta que no hay admin
→ Usuario es redirigido a /setup
→ Completa formulario con sus datos
→ Sistema crea admin y redirige a login
→ Usuario inicia sesión
→ ✅ Listo para usar
```

### Caso 2: Reinstalación/Reset

```
Admin quiere resetear el sistema
→ Elimina todos los usuarios admin de MongoDB
→ Reinicia el navegador
→ Sistema detecta que no hay admin
→ Muestra formulario de setup nuevamente
→ Se puede crear un nuevo admin
```

### Caso 3: Sistema ya configurado

```
Usuario visita la aplicación
→ Sistema detecta que ya hay admin
→ Muestra login normal
→ Usuario inicia sesión con credenciales existentes
```

## ⚠️ Notas Importantes

1. **No más credenciales por defecto** - El sistema NO crea usuario admin/admin123
2. **Setup solo una vez** - El formulario solo funciona cuando no hay admin
3. **Guarda tus credenciales** - No hay recuperación automática (por ahora)
4. **Primer usuario = Admin total** - Tendrá acceso completo al sistema
5. **Contraseña segura recomendada** - Mínimo 6 caracteres, pero usa más en producción

## 🚀 Próximas Mejoras (Futuro)

- [ ] Recuperación de contraseña por email
- [ ] Autenticación de dos factores (2FA)
- [ ] Límite de intentos de setup
- [ ] Logging de intentos de acceso a setup
- [ ] Opción de "Setup Mode" en configuración
- [ ] Wizard multi-paso para configuración inicial completa

---

**Autor:** OmniAPI Team  
**Versión:** 2.0  
**Fecha:** Noviembre 2025
