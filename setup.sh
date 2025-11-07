#!/bin/bash
# Setup script para OmniAPI

echo "🚀 OmniAPI Setup Script"
echo "======================="

# Verificar Go
if ! command -v go &> /dev/null; then
    echo "❌ Go no está instalado. Por favor instala Go 1.24.0 o superior."
    exit 1
fi

echo "✅ Go encontrado: $(go version)"

# Verificar MongoDB
if ! command -v mongod &> /dev/null; then
    echo "⚠️  MongoDB no encontrado. Asegúrate de tener MongoDB corriendo en localhost:27017"
else
    echo "✅ MongoDB encontrado"
fi

# Copiar archivo de entorno
if [ ! -f .env ]; then
    echo "📋 Creando archivo .env desde template..."
    cp .env.example .env
    echo "⚠️  Por favor edita .env con tus valores reales"
else
    echo "✅ Archivo .env ya existe"
fi

# Instalar dependencias
echo "📦 Instalando dependencias Go..."
go mod tidy

# Compilar proyecto
echo "🔨 Compilando proyecto..."
if go build -o omniapi.exe main.go; then
    echo "✅ Compilación exitosa"
else
    echo "❌ Error en compilación"
    exit 1
fi

# Ejecutar tests
echo "🧪 Ejecutando tests..."
if go test ./...; then
    echo "✅ Tests pasaron"
else
    echo "⚠️  Algunos tests fallaron"
fi

echo ""
echo "🎉 Setup completado!"
echo "📖 Para ejecutar el servidor:"
echo "   go run main.go"
echo ""
echo "🌐 URLs disponibles:"
echo "   http://localhost:3000 - Página principal"
echo "   http://localhost:3000/api/health - API Health"
echo "   ws://localhost:3000/ws - WebSocket"