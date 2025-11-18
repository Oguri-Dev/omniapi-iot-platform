@echo off
REM Setup script para OmniAPI en Windows

echo 🚀 OmniAPI Setup Script
echo =======================

REM Verificar Go
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go no está instalado. Por favor instala Go 1.24.0 o superior.
    pause
    exit /b 1
)

echo ✅ Go encontrado
go version

REM Copiar archivo de entorno
if not exist .env (
    echo 📋 Creando archivo .env desde template...
    copy .env.example .env
    echo ⚠️  Por favor edita .env con tus valores reales
) else (
    echo ✅ Archivo .env ya existe
)

REM Instalar dependencias
echo 📦 Instalando dependencias Go...
go mod tidy

REM Compilar proyecto
echo 🔨 Compilando proyecto...
go build -o omniapi.exe main.go
if %errorlevel% neq 0 (
    echo ❌ Error en compilación
    pause
    exit /b 1
)
echo ✅ Compilación exitosa

REM Ejecutar tests
echo 🧪 Ejecutando tests...
go test ./...
if %errorlevel% neq 0 (
    echo ⚠️  Algunos tests fallaron
) else (
    echo ✅ Tests pasaron
)

echo.
echo 🎉 Setup completado!
echo 📖 Para ejecutar el servidor:
echo    go run main.go
echo.
echo 🌐 URLs disponibles:
echo    http://localhost:3000 - Página principal
echo    http://localhost:3000/api/health - API Health
echo    ws://localhost:3000/ws - WebSocket
echo.
pause