# 🚀 Setup Manual Interactivo - OmniAPI

## ✨ Características Principales

### 🎯 Lo que cambió

Antes tenías que usar credenciales por defecto (`admin/admin123`). **Ahora creas tu propio administrador** con un formulario web profesional.

### 🔐 Seguridad Mejorada

- ✅ **Sin contraseñas por defecto** - Tú defines tus credenciales
- ✅ **Validación en tiempo real** - Verifica que todo esté correcto
- ✅ **Hash bcrypt** - Contraseñas almacenadas de forma segura
- ✅ **Protección anti-duplicados** - Solo funciona cuando no hay admin

### 🎨 Interfaz Moderna

- ✅ **Diseño profesional** con degradados y animaciones
- ✅ **Responsive** - Funciona en móvil, tablet y desktop
- ✅ **Mensajes claros** - Sabes exactamente qué hacer
- ✅ **Validaciones visuales** - Errores mostrados en tiempo real

---

## 📸 Vista Previa del Flujo

### 1️⃣ Primera Visita al Sistema

Cuando abres `http://localhost:5173` por primera vez:

```
┌─────────────────────────────────────┐
│                                     │
│            🚀 (animado)             │
│                                     │
│     Configuración Inicial           │
│  Crea el primer usuario admin       │
│                                     │
├─────────────────────────────────────┤
│                                     │
│  Nombre de Usuario *                │
│  ┌───────────────────────────────┐  │
│  │                               │  │
│  └───────────────────────────────┘  │
│                                     │
│  Email *                            │
│  ┌───────────────────────────────┐  │
│  │                               │  │
│  └───────────────────────────────┘  │
│                                     │
│  Nombre Completo (opcional)         │
│  ┌───────────────────────────────┐  │
│  │                               │  │
│  └───────────────────────────────┘  │
│                                     │
│  Contraseña *                       │
│  ┌───────────────────────────────┐  │
│  │ ••••••••                      │  │
│  └───────────────────────────────┘  │
│  Mínimo 6 caracteres                │
│                                     │
│  Confirmar Contraseña *             │
│  ┌───────────────────────────────┐  │
│  │ ••••••••                      │  │
│  └───────────────────────────────┘  │
│                                     │
│  ┌───────────────────────────────┐  │
│  │  🔐 Crear Administrador       │  │
│  └───────────────────────────────┘  │
│                                     │
│  ⚠️ Importante: Este usuario        │
│  tendrá acceso completo al sistema  │
│                                     │
└─────────────────────────────────────┘
```

### 2️⃣ Después de Crear el Admin

Eres redirigido al login con un mensaje de éxito:

```
┌─────────────────────────────────────┐
│                                     │
│         🚀 OmniAPI                  │
│    Panel de Administración          │
│                                     │
├─────────────────────────────────────┤
│                                     │
│  ✅ Administrador creado           │
│     exitosamente. Ya puedes         │
│     iniciar sesión.                 │
│                                     │
├─────────────────────────────────────┤
│                                     │
│  Usuario                            │
│  ┌───────────────────────────────┐  │
│  │                               │  │
│  └───────────────────────────────┘  │
│                                     │
│  Contraseña                         │
│  ┌───────────────────────────────┐  │
│  │ ••••••••                      │  │
│  └───────────────────────────────┘  │
│                                     │
│  ┌───────────────────────────────┐  │
│  │    Iniciar Sesión             │  │
│  └───────────────────────────────┘  │
│                                     │
└─────────────────────────────────────┘
```

### 3️⃣ Dashboard (Después del Login)

```
┌──────────┬──────────────────────────────┐
│          │                              │
│  🏠 Home │  📊 Dashboard                │
│          │                              │
│ 📦 Serv. │  ┌──────┐ ┌──────┐ ┌──────┐  │
│          │  │  12  │ │  5   │ │ 453  │  │
│ 🔌 Conn. │  │ Serv.│ │ Conn.│ │ Data │  │
│          │  └──────┘ └──────┘ └──────┘  │
│ ⚙️ Config│                              │
│          │  Acciones Rápidas:           │
│ ────────  │  [+ Nuevo Servicio]         │
│          │  [🔄 Sincronizar]            │
│ admin@   │  [📊 Ver Estadísticas]       │
│ [Logout] │                              │
│          │                              │
└──────────┴──────────────────────────────┘
```

---

## 🛠️ Cómo Funciona

### Arquitectura del Sistema

```
Frontend (React)                Backend (Go)
─────────────                   ─────────────

1. Usuario visita /
   │
   ├─→ GET /api/auth/setup/check
   │                              │
   │                              ├─→ CheckAdminExists()
   │                              │   └─→ MongoDB: count admins
   │                              │
   │   ←─── { needsSetup: true } ─┤
   │
   ├─→ Redirige a /setup
   │
   │
2. Usuario completa formulario
   │
   ├─→ POST /api/auth/setup
   │   {
   │     username: "mi_admin",
   │     email: "admin@empresa.com",
   │     password: "Password123!",
   │     fullName: "Admin"
   │   }
   │                              │
   │                              ├─→ CreateFirstAdmin()
   │                              │   ├─→ Verifica no hay admin
   │                              │   ├─→ Hash password (bcrypt)
   │                              │   └─→ MongoDB: insert user
   │                              │
   │   ←─── { success: true } ────┤
   │
   ├─→ Redirige a /login
   │
   │
3. Usuario hace login
   │
   ├─→ POST /api/auth/login
   │   {
   │     username: "mi_admin",
   │     password: "Password123!"
   │   }
   │                              │
   │                              ├─→ LoginHandler()
   │                              │   ├─→ Busca usuario
   │                              │   ├─→ Verifica password
   │                              │   ├─→ Genera token
   │                              │   └─→ Crea sesión
   │                              │
   │   ←─── { token, user } ──────┤
   │
   └─→ Acceso al Dashboard ✅
```

---

## 🔧 Instalación y Uso

### Requisitos

- Node.js 18+
- Go 1.24+
- MongoDB 4.4+

### Instalación

**1. Backend:**

```bash
cd omniapi-iot-platform-git
go run main.go
```

**2. Frontend:**

```bash
cd omniapi-front
npm install
npm run dev
```

### Primer Uso

1. Abre `http://localhost:5173`
2. Serás redirigido automáticamente a `/setup`
3. Completa el formulario con tus datos
4. Click "Crear Administrador"
5. Inicia sesión con tus credenciales
6. ¡Listo! 🎉

---

## 📋 Checklist de Setup

### Backend ✅

- [x] CheckAdminExists() implementado
- [x] CreateFirstAdmin() implementado
- [x] Endpoint GET /api/auth/setup/check
- [x] Endpoint POST /api/auth/setup
- [x] Validaciones de seguridad
- [x] Hash bcrypt de contraseñas

### Frontend ✅

- [x] Página Setup.tsx creada
- [x] Estilos Setup.css
- [x] Servicio setup.service.ts
- [x] AuthContext actualizado
- [x] Ruta /setup agregada
- [x] Validaciones de formulario
- [x] Mensaje de éxito en login

---

## 🐛 Troubleshooting

### El formulario no aparece

**Problema:** Abro `http://localhost:5173` pero no veo el formulario de setup.

**Solución:**

1. Verifica que el backend esté corriendo en puerto 3000
2. Abre DevTools (F12) → Network tab
3. Busca la llamada a `/api/auth/setup/check`
4. Si dice `needsSetup: false`, significa que ya hay un admin en la DB

### Error "ya existe un usuario administrador"

**Problema:** Al intentar crear admin, dice que ya existe uno.

**Solución:**
Esto es correcto. Solo puedes crear un admin cuando NO existe ninguno.

Para resetear y probar de nuevo:

1. Abre MongoDB Compass
2. Conecta a `mongodb://localhost:27017`
3. Base de datos: `omniapi`
4. Colección: `users`
5. Elimina documentos con `{ role: "admin" }`
6. Recarga el navegador

### El backend no inicia en puerto 3000

**Problema:** Error "port 3000 already in use"

**Solución:**

```bash
# Windows
netstat -ano | findstr :3000
taskkill /F /PID [PID_NUMBER]

# Linux/Mac
lsof -i :3000
kill -9 [PID]
```

---

## 📚 Documentación Adicional

- [MANUAL_SETUP.md](./MANUAL_SETUP.md) - Guía completa de setup manual
- [SETUP_CHANGES_SUMMARY.md](./SETUP_CHANGES_SUMMARY.md) - Resumen de todos los cambios
- [SETUP_GUIDE.md](../SETUP_GUIDE.md) - Guía general del sistema

---

## 🎯 Características Destacadas

### Validaciones Implementadas

✅ **Username:**

- Campo requerido
- Sin espacios al inicio/final

✅ **Email:**

- Campo requerido
- Debe contener @
- Formato de email válido

✅ **Password:**

- Campo requerido
- Mínimo 6 caracteres
- Confirmación debe coincidir

✅ **Seguridad:**

- Endpoint solo funciona sin admin existente
- Hash bcrypt automático
- Sin exposición de contraseñas

### Estados del Sistema

🟢 **needsSetup: true** → Muestra formulario de setup
🔴 **needsSetup: false** → Muestra login normal

---

## 💡 Tips

1. **Usa contraseñas seguras** - Mínimo 8 caracteres con mayúsculas, minúsculas y números
2. **Guarda tus credenciales** - Por ahora no hay recuperación de contraseña
3. **Email real** - Útil para futuras notificaciones
4. **Nombre completo** - Ayuda a identificar al usuario en logs

---

## 🚀 Próximas Mejoras

- [ ] Recuperación de contraseña por email
- [ ] Validación de fuerza de contraseña
- [ ] Autenticación de dos factores (2FA)
- [ ] Wizard multi-paso
- [ ] Modo oscuro/claro
- [ ] Soporte multi-idioma

---

**Creado con ❤️ por el equipo de OmniAPI**

**Versión:** 2.0  
**Última actualización:** Noviembre 13, 2025
