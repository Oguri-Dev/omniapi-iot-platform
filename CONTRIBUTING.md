# Contributing to OmniAPI 🤝

¡Gracias por tu interés en contribuir a OmniAPI! Este documento te guiará a través del proceso.

## 🎯 Cómo Contribuir

### 🐛 Reportar Bugs
1. Verifica que el bug no haya sido reportado anteriormente
2. Abre un issue con el template de bug report
3. Incluye información detallada sobre el entorno y pasos para reproducir

### ✨ Sugerir Features
1. Abre un issue con el template de feature request
2. Describe claramente el caso de uso y beneficios
3. Discute la implementación propuesta

### 🔄 Pull Requests
1. Fork el repositorio
2. Crea una rama desde `main`: `git checkout -b feature/nueva-funcionalidad`
3. Hace tus cambios siguiendo las convenciones del proyecto
4. Agrega tests para nuevas funcionalidades
5. Asegúrate que todos los tests pasen: `go test ./...`
6. Haz commit con mensajes descriptivos
7. Push tu rama y abre un Pull Request

## 🏗️ Desarrollo Local

### Prerequisites
- Go 1.24.0+
- MongoDB 4.4+
- Docker (opcional)

### Setup
```bash
git clone https://github.com/TM-Opera-O/omniapi-iot-platform.git
cd omniapi-iot-platform
cp .env.example .env  # Configurar variables
go mod tidy
go run main.go
```

### Testing
```bash
# Todos los tests
go test ./...

# Tests con coverage
go test -cover ./...

# Tests específicos
go test ./internal/connectors/...
```

## 📋 Convenciones

### Commits
Usamos [Conventional Commits](https://www.conventionalcommits.org/):
```
feat: agregar conector LoRaWAN
fix: corregir validación de schema climate
docs: actualizar README con nuevos endpoints
test: agregar tests para MQTT connector
```

### Código Go
- Seguir [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Usar `gofmt` y `golint`
- Documentar funciones públicas
- Máximo 100 caracteres por línea

### Conectores
Al crear nuevos conectores:
1. Implementar la interfaz `Connector`
2. Agregar factory function
3. Registrar en `adapters/register.go`
4. Incluir tests unitarios
5. Documentar configuración y capabilities

### API Endpoints
- Seguir RESTful conventions
- Validar inputs con structs
- Manejar errores consistentemente
- Documentar en README

## 🏷️ Labels

- `bug`: Problemas/errores
- `enhancement`: Nuevas funcionalidades
- `documentation`: Mejoras en docs
- `good first issue`: Issues para nuevos contributors
- `help wanted`: Necesita ayuda de la comunidad
- `connector`: Relacionado con conectores
- `api`: Endpoints REST
- `websocket`: Funcionalidad WebSocket
- `schema`: Validación de esquemas

## 📞 Contacto

- Issues: Usa GitHub Issues para reportes y discusiones
- Discusiones: GitHub Discussions para preguntas generales
- Seguridad: Para vulnerabilidades, contacta privadamente

## 📜 Código de Conducta

Este proyecto sigue el [Contributor Covenant](https://www.contributor-covenant.org/). Al participar, se espera que mantengas este código.

¡Gracias por hacer OmniAPI mejor! 🚀