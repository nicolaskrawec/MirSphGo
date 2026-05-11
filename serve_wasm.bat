@echo off
setlocal
echo Lancement du serveur Go pour le WASM...
set GOOS=
set GOARCH=
go run server/main.go
endlocal
