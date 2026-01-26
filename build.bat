@echo off
echo  Building CMDO for all platforms...

REM Create build directory
if not exist build mkdir build

REM Windows builds
echo  Building for Windows...
set GOOS=windows
set GOARCH=amd64
go build -o build/cmdo-windows-amd64.exe

set GOARCH=386
go build -o build/cmdo-windows-386.exe

REM macOS builds
echo  Building for macOS...
set GOOS=darwin
set GOARCH=amd64
go build -o build/cmdo-macos-intel

set GOARCH=arm64
go build -o build/cmdo-macos-arm64

REM Linux builds
echo  Building for Linux...
set GOOS=linux
set GOARCH=amd64
go build -o build/cmdo-linux-amd64

set GOARCH=386
go build -o build/cmdo-linux-386

set GOARCH=arm64
go build -o build/cmdo-linux-arm64

echo  Build complete! Files in ./build/ directory:
dir build\