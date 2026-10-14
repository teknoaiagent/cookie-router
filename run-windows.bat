@echo off
title Cookie Router v2.0.0
echo ========================================================
echo   Cookie Router v2.0.0 - Web Cookie AI Gateway
echo   Dashboard: http://localhost:18899/
echo ========================================================
echo.

if not exist "config.json" (
    if exist "config.example.json" (
        copy "config.example.json" "config.json" >nul
        echo [INFO] config.json dibuat dari config.example.json
    )
)

if exist "cookie-router-windows-amd64.exe" (
    start http://localhost:18899/
    cookie-router-windows-amd64.exe
) else if exist "cookie-router.exe" (
    start http://localhost:18899/
    cookie-router.exe
) else (
    echo [ERROR] Executable cookie-router tidak ditemukan!
    pause
)
