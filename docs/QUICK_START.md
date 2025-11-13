# 🚀 Guía Rápida de Inicio - Setup Manual

## ⚡ Inicio en 3 Pasos

### 1️⃣ Inicia el Backend

```bash
cd C:\Users\Andres\Documents\VsCodework\omniapi-iot-platform-git
go run main.go
```

✅ **Verás esto:**

```
🔐 Checking admin user...
✅ Admin user exists
```

O si no hay admin:

```
🔐 Checking admin user...
⚠️  No admin user found. Please complete setup via /api/auth/setup
```

### 2️⃣ Inicia el Frontend

```bash
cd C:\Users\Andres\Documents\VsCodework\omniapi-front
npm run dev
```

✅ **Verás esto:**

```
  VITE v5.x.x  ready in xxx ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: use --host to expose
```

### 3️⃣ Abre el Navegador

```
http://localhost:5173
```

---

## 🎬 Escenarios de Uso

### Escenario A: Primera Vez (Sin Admin)

```
┌─────────────────────────────────────────────────┐
│ 1. Abres http://localhost:5173                  │
│    └→ Sistema verifica si hay admin             │
│                                                  │
│ 2. NO hay admin                                 │
│    └→ Redirige automáticamente a /setup        │
│                                                  │
│ 3. Ves el formulario de setup                   │
│    ┌──────────────────────────────────────┐     │
│    │  🚀 Configuración Inicial            │     │
│    │                                      │     │
│    │  Username: [___________________]     │     │
│    │  Email:    [___________________]     │     │
│    │  Name:     [___________________]     │     │
│    │  Password: [___________________]     │     │
│    │  Confirm:  [___________________]     │     │
│    │                                      │     │
│    │  [🔐 Crear Administrador]            │     │
│    └──────────────────────────────────────┘     │
│                                                  │
│ 4. Completas el formulario                      │
│    └→ Click "Crear Administrador"               │
│                                                  │
│ 5. Sistema crea el admin                        │
│    └→ Redirige a /login con mensaje verde      │
│                                                  │
│ 6. Login con tus credenciales                   │
│    └→ Acceso al dashboard ✅                    │
└─────────────────────────────────────────────────┘
```

### Escenario B: Admin Ya Existe

```
┌─────────────────────────────────────────────────┐
│ 1. Abres http://localhost:5173                  │
│    └→ Sistema verifica si hay admin             │
│                                                  │
│ 2. SÍ hay admin                                 │
│    └→ Muestra login normal                      │
│                                                  │
│ 3. Ves el formulario de login                   │
│    ┌──────────────────────────────────────┐     │
│    │  🚀 OmniAPI                          │     │
│    │  Panel de Administración             │     │
│    │                                      │     │
│    │  Usuario:   [___________________]    │     │
│    │  Password:  [___________________]    │     │
│    │                                      │     │
│    │  [Iniciar Sesión]                    │     │
│    └──────────────────────────────────────┘     │
│                                                  │
│ 4. Login con credenciales existentes            │
│    └→ Acceso al dashboard ✅                    │
└─────────────────────────────────────────────────┘
```

---

## 🧪 Cómo Probar el Setup desde Cero

### Método 1: Script Batch (Recomendado para Windows)

```bash
cd C:\Users\Andres\Documents\VsCodework\omniapi-iot-platform-git\scripts
reset_admin.bat
```

**Menú interactivo:**

```
==========================================
 OmniAPI - Reset Admin User Script
==========================================

Opciones:
[1] Ver estado actual (contar admins)
[2] Eliminar TODOS los usuarios admin
[3] Salir

Selecciona una opcion (1-3): 2
```

### Método 2: MongoDB Compass (Visual)

```
1. Abrir MongoDB Compass
   └→ Conectar a: mongodb://localhost:27017

2. Seleccionar base de datos
   └→ Click en "omniapi"

3. Seleccionar colección
   └→ Click en "users"

4. Filtrar admins
   └→ Filtro: { "role": "admin" }

5. Eliminar
   └→ Seleccionar todos → Delete
```

### Método 3: MongoDB Shell

```bash
mongosh omniapi
```

```javascript
// Ver admins actuales
db.users.find({ role: 'admin' })

// Eliminar todos los admins
db.users.deleteMany({ role: 'admin' })

// Verificar que no queden
db.users.countDocuments({ role: 'admin' })
// Debe retornar: 0

// Salir
exit
```

---

## 📋 Checklist de Verificación

### Antes de Probar

- [ ] MongoDB está corriendo
- [ ] Backend está corriendo en puerto 3000
- [ ] Frontend está corriendo en puerto 5173
- [ ] No hay errores en consola del backend
- [ ] No hay errores en consola del frontend

### Durante el Setup

- [ ] Formulario de setup aparece automáticamente
- [ ] Todos los campos son editables
- [ ] Validaciones funcionan (campos vacíos, email, passwords)
- [ ] Botón cambia a "Creando..." con spinner
- [ ] Redirección a /login ocurre automáticamente
- [ ] Mensaje de éxito verde aparece en login

### Después del Setup

- [ ] Login funciona con credenciales creadas
- [ ] Dashboard se muestra correctamente
- [ ] Usuario logueado aparece en sidebar
- [ ] Logout funciona
- [ ] Al volver a /setup redirige a login (ya hay admin)

---

## 🐛 Troubleshooting Rápido

### "No veo el formulario de setup"

**Causa:** Ya hay un admin en la base de datos

**Solución:**

```bash
# Opción 1: Usar el script
scripts\reset_admin.bat

# Opción 2: MongoDB Shell
mongosh omniapi --eval "db.users.deleteMany({role: 'admin'})"
```

### "Error: port 3000 already in use"

**Causa:** Otro proceso usando el puerto

**Solución:**

```bash
# Ver qué proceso usa el puerto
netstat -ano | findstr :3000

# Matar el proceso (reemplaza PID con el número que ves)
taskkill /F /PID [PID]
```

### "MongoDB no conecta"

**Causa:** MongoDB no está corriendo

**Solución:**

```bash
# Windows (como servicio)
net start MongoDB

# Windows (manual)
mongod --dbpath C:\data\db

# Linux/Mac
sudo systemctl start mongod
```

### "Frontend no inicia"

**Causa:** Dependencias no instaladas

**Solución:**

```bash
cd omniapi-front
npm install
npm run dev
```

### "Formulario no valida"

**Causa:** JavaScript no cargado o error en consola

**Solución:**

```
1. Abrir DevTools (F12)
2. Ver pestaña Console
3. Verificar errores
4. Hacer hard refresh (Ctrl+Shift+R)
```

---

## 💡 Tips y Mejores Prácticas

### Credenciales Recomendadas

✅ **Buenas prácticas:**

```
Username: admin
Email: admin@tuempresa.com
Password: TuPassword2024!
```

❌ **Evitar:**

```
Username: test, admin123, root
Password: 123456, password, admin
Email: test@test.com, admin@admin.com
```

### Seguridad

- 🔒 Usa contraseñas de al menos 8-12 caracteres
- 🔒 Combina mayúsculas, minúsculas, números y símbolos
- 🔒 No uses información personal (nombre, fecha nacimiento)
- 🔒 Guarda las credenciales en lugar seguro
- 🔒 Considera usar un gestor de contraseñas

### Desarrollo vs Producción

**Desarrollo:**

```
Username: admin_dev
Password: Dev2024!
Email: dev@localhost
```

**Producción:**

```
Username: [tu_nombre_real]
Password: [password_fuerte_único]
Email: [tu_email_real]
```

---

## 📊 Estados del Sistema

### Backend Logs

**Sin admin:**

```bash
🔐 Checking admin user...
⚠️  No admin user found. Please complete setup via /api/auth/setup
📡 Initializing Router...
✅ Router started successfully
...
🎯 Server listening on port 3000
```

**Con admin:**

```bash
🔐 Checking admin user...
✅ Admin user exists
📡 Initializing Router...
✅ Router started successfully
...
🎯 Server listening on port 3000
```

### Frontend DevTools

**Request a /api/auth/setup/check:**

```json
{
  "success": true,
  "message": "Setup status",
  "data": {
    "needsSetup": true // false si ya hay admin
  },
  "timestamp": 1699876543
}
```

**Request a /api/auth/setup (POST):**

```json
{
  "success": true,
  "message": "Administrador creado exitosamente",
  "data": {
    "id": "655f1c2e8c4b2a1234567890",
    "username": "admin",
    "email": "admin@omniapi.com",
    "role": "admin"
  },
  "timestamp": 1699876543
}
```

---

## 🎯 Flujo Visual Completo

```
┌──────────────┐
│  Usuario     │
│  abre app    │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│  AuthContext         │
│  verifica setup      │
└──────┬───────────────┘
       │
       ├─────────────┬─────────────┐
       │             │             │
   ¿Hay admin?   NO  │        SÍ   │
       │             │             │
       ▼             ▼             ▼
┌─────────────┐ ┌──────────┐ ┌──────────┐
│ needsSetup  │ │ Redirige │ │ Muestra  │
│   = true    │ │ a /setup │ │  /login  │
└─────────────┘ └────┬─────┘ └────┬─────┘
                     │            │
                     ▼            │
              ┌─────────────┐    │
              │ Formulario  │    │
              │   Setup     │    │
              └──────┬──────┘    │
                     │            │
                     ▼            │
              ┌─────────────┐    │
              │ Usuario     │    │
              │ completa    │    │
              └──────┬──────┘    │
                     │            │
                     ▼            │
              ┌─────────────┐    │
              │ POST /setup │    │
              └──────┬──────┘    │
                     │            │
                     ▼            │
              ┌─────────────┐    │
              │ Admin       │    │
              │ creado ✅   │    │
              └──────┬──────┘    │
                     │            │
                     ▼            │
              ┌─────────────┐    │
              │ Redirige a  │◄───┘
              │   /login    │
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │ Usuario     │
              │ hace login  │
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │ Dashboard   │
              │   ✅        │
              └─────────────┘
```

---

## 📞 Recursos de Ayuda

### Documentación

- 📖 [MANUAL_SETUP.md](./MANUAL_SETUP.md) - Guía completa
- 📖 [SETUP_CHANGES_SUMMARY.md](./SETUP_CHANGES_SUMMARY.md) - Cambios detallados
- 📖 [TASK_COMPLETED.md](./TASK_COMPLETED.md) - Resumen de tarea
- 📖 [../SETUP_GUIDE.md](../SETUP_GUIDE.md) - Guía general

### Scripts

- 🔧 [reset_admin.bat](../scripts/reset_admin.bat) - Resetear admin

### Endpoints

- 🌐 Frontend: http://localhost:5173
- 🌐 Backend: http://localhost:3000
- 🌐 Setup Check: http://localhost:3000/api/auth/setup/check

---

**✨ ¡Disfruta tu nuevo sistema de setup manual!**

**Versión:** 2.0  
**Última actualización:** Noviembre 13, 2025
