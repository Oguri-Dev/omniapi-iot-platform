# Multi-stage build para OmniAPI
FROM golang:1.24-alpine AS builder

# Instalar dependencias necesarias
RUN apk add --no-cache git ca-certificates

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de módulos primero (para aprovechar cache de Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o omniapi main.go

# Imagen final pequeña
FROM alpine:latest

# Instalar certificados SSL y timezone data
RUN apk --no-cache add ca-certificates tzdata

# Crear usuario no-root
RUN adduser -D -s /bin/sh omniapi

# Establecer directorio de trabajo
WORKDIR /app

# Copiar binario desde builder
COPY --from=builder /app/omniapi .

# Copiar archivos de configuración
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/.env.example ./.env.example

# Cambiar ownership al usuario omniapi
RUN chown -R omniapi:omniapi /app
USER omniapi

# Exponer puerto
EXPOSE 3000

# Variables de entorno por defecto
ENV ENVIRONMENT=production
ENV PORT=3000
ENV MONGODB_URI=mongodb://mongo:27017/omniapi

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/api/health || exit 1

# Comando por defecto
CMD ["./omniapi"]