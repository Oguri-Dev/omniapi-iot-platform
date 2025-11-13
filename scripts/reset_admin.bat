@echo off
echo ==========================================
echo  OmniAPI - Reset Admin User Script
echo ==========================================
echo.
echo Este script te ayudara a resetear el sistema
echo para probar el flujo de setup manual.
echo.

:menu
echo.
echo Opciones:
echo [1] Ver estado actual (contar admins)
echo [2] Eliminar TODOS los usuarios admin
echo [3] Salir
echo.
set /p option="Selecciona una opcion (1-3): "

if "%option%"=="1" goto count
if "%option%"=="2" goto delete
if "%option%"=="3" goto end
echo Opcion invalida. Intenta de nuevo.
goto menu

:count
echo.
echo Verificando usuarios admin en MongoDB...
echo.
mongosh omniapi --quiet --eval "db.users.countDocuments({role: 'admin', status: 'active'})"
if errorlevel 1 (
    echo.
    echo ERROR: No se pudo conectar a MongoDB.
    echo Asegurate de que MongoDB este corriendo en localhost:27017
    echo.
    pause
    goto menu
)
echo.
echo Usuarios con rol 'admin' y status 'active'
pause
goto menu

:delete
echo.
echo ==========================================
echo  ADVERTENCIA - OPERACION DESTRUCTIVA
echo ==========================================
echo.
echo Esto eliminara TODOS los usuarios con rol 'admin'
echo de la base de datos 'omniapi'.
echo.
echo Despues de esto, podras probar el flujo de setup
echo manual abriendo http://localhost:5173
echo.
set /p confirm="Estas seguro? (S/N): "
if /i not "%confirm%"=="S" (
    echo Operacion cancelada.
    pause
    goto menu
)

echo.
echo Eliminando usuarios admin...
echo.
mongosh omniapi --quiet --eval "const result = db.users.deleteMany({role: 'admin'}); print('Eliminados: ' + result.deletedCount + ' usuarios admin')"
if errorlevel 1 (
    echo.
    echo ERROR: No se pudo ejecutar la operacion.
    echo.
    pause
    goto menu
)

echo.
echo ==========================================
echo  Setup Completado!
echo ==========================================
echo.
echo Ahora puedes:
echo 1. Abrir tu navegador en http://localhost:5173
echo 2. Seras redirigido automaticamente a /setup
echo 3. Completa el formulario con tus datos
echo 4. Inicia sesion con las credenciales creadas
echo.
echo El backend mostrara:
echo "  No admin user found. Please complete setup via /api/auth/setup"
echo.
pause
goto menu

:end
echo.
echo Saliendo...
exit /b
