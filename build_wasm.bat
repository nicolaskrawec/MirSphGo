@echo off
echo Building for WebAssembly...
set GOOS=js
set GOARCH=wasm
go build -o main.wasm .
echo Done. Created main.wasm
