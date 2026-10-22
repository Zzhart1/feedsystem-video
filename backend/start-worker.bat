@echo off
cd /d D:\feedsystem_video\backend
echo === Building Worker ===
go build -o worker.exe ./cmd/worker
if errorlevel 1 (echo Build failed & pause & exit /b 1)
echo === Starting Worker ===
worker.exe
pause
