@echo off
setlocal
echo Building for WebAssembly...
set GOOS=js
set GOARCH=wasm
go build -o dist/main.wasm .
echo Done. Created dist/main.wasm
endlocal
