@echo off
cd /d D:\feedsystem_video\backend
echo === Building API ===
go build -o feedsystem.exe ./cmd
if errorlevel 1 (echo Build failed & pause & exit /b 1)
echo === Starting API on :8080 ===
feedsystem.exe
pause
