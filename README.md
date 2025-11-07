# OmniAPI 🚀

Un sistema IoT avanzado desarrollado en Go con arquitectura de conectores, validación de esquemas, WebSockets y API REST completa para integración multi-tenant de datos agrícolas y acuícolas.

## 🎯 Características Principales

### 🏗️ Arquitectura de Conectores
- ✅ Framework de conectores extensible con catálogo global
- ✅ Conectores MQTT Feed para datos de alimentación
- ✅ Conectores REST Climate para datos climáticos
- ✅ Conector dummy para testing y demostración
- ✅ Sistema de mappings configurable proveedor→canónico

### 🔍 Validación y Esquemas
- ✅ Validación automática con JSON Schema
- ✅ Esquemas versionados (feeding.v1, climate.v1, biometric.v1)
- ✅ Backward compatibility y evolución de esquemas
- ✅ API de validación REST

### 🌐 API y WebSockets
- ✅ API REST completa con MongoDB
- ✅ WebSockets en tiempo real para streaming de datos
- ✅ Endpoints de salud y monitoreo
- ✅ Sistema multi-tenant con control de acceso

### ⚙️ Configuración y Deployment
- ✅ Configuración YAML multi-archivo
- ✅ Gestión de secretos con variables de entorno
- ✅ Hot-reload de configuración
- ✅ Docker ready y production ready

## 🛠️ Requisitos Previos

- Go 1.24.0 o superior
- MongoDB 4.4 o superior
- VS Code con extensión de Go (recomendado)

## 🚀 Instalación y Ejecución

### 1. Clonar o descargar el proyecto

```bash
# Si usas git
git clone <tu-repositorio>
cd omniapi
```

### 2. Instalar dependencias

```bash
go mod tidy
```

### 3. Compilar el proyecto

```bash
go build .
```

### 4. Ejecutar la aplicación

```bash
# Opción 1: Ejecutar directamente con go
go run main.go

# Opción 2: Ejecutar el binario compilado
./omniapi.exe     # En Windows
./omniapi         # En Linux/Mac
```

### 5. Acceder a la aplicación

- **Página principal**: http://localhost:8080
- **API de salud**: http://localhost:8080/api/health

## 📁 Estructura del Proyecto

```
omniapi/
├── .github/
│   └── copilot-instructions.md    # Instrucciones para GitHub Copilot
├── main.go                        # Punto de entrada de la aplicación
├── go.mod                         # Definición del módulo Go
├── go.sum                         # Checksums de dependencias
├── omniapi.exe                    # Binario compilado (Windows)
└── README.md                      # Este archivo
```

## 🔧 Desarrollo

### Agregar nuevas rutas

Para agregar una nueva ruta, modifica el archivo `main.go`:

```go
func main() {
    // Agregar nueva ruta
    http.HandleFunc("/nueva-ruta", nuevoHandler)

    // ... resto del código
}

func nuevoHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "¡Nueva funcionalidad!")
}
```

### Ejecutar en modo desarrollo

Para desarrollo con recarga automática, puedes usar:

```bash
# Instalar air para hot reload (opcional)
go install github.com/cosmtrek/air@latest

# Ejecutar con hot reload
air
```

## 🧪 Testing

Para agregar tests, crea archivos `*_test.go`:

```bash
# Ejecutar tests
go test ./...

# Ejecutar tests con cobertura
go test -cover ./...
```

## 🐳 Docker (Opcional)

Crear un `Dockerfile` para containerización:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

## 📊 API Endpoints

| Método | Endpoint    | Descripción                   |
| ------ | ----------- | ----------------------------- |
| GET    | /           | Página principal con interfaz |
| GET    | /api/health | Estado del servidor (JSON)    |

### Ejemplo de respuesta `/api/health`:

```json
{
  "status": "ok",
  "message": "El servidor está funcionando correctamente",
  "timestamp": "1699363200",
  "version": "1.0.0"
}
```

## 🔄 Próximos Pasos

- [ ] Agregar base de datos (PostgreSQL/MySQL)
- [ ] Implementar autenticación JWT
- [ ] Crear middleware de logging
- [ ] Agregar tests unitarios
- [ ] Implementar métricas con Prometheus
- [ ] Documentación API con Swagger
- [ ] Configuración con variables de entorno

## 📝 Comandos Útiles

```bash
# Formatear código
go fmt ./...

# Verificar problemas
go vet ./...

# Instalar dependencias
go mod tidy

# Actualizar dependencias
go get -u ./...

# Ver información del módulo
go list -m all
```

## 🤝 Contribución

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/nueva-funcionalidad`)
3. Commit tus cambios (`git commit -am 'Agregar nueva funcionalidad'`)
4. Push a la rama (`git push origin feature/nueva-funcionalidad`)
5. Crear un Pull Request

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 👨‍💻 Autor

Desarrollado con ❤️ usando Go y las mejores prácticas de desarrollo.
