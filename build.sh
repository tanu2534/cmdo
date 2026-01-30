#!/bin/bash

echo " Building CMDO for all platforms..."

# Create build directory
mkdir -p build

# Windows builds
echo " Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o build/cmdo-windows-amd64.exe
GOOS=windows GOARCH=386 go build -o build/cmdo-windows-386.exe

# macOS builds
echo " Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -o build/cmdo-macos-intel
GOOS=darwin GOARCH=arm64 go build -o build/cmdo-macos-arm64

# Linux builds
echo " Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o build/cmdo-linux-amd64
GOOS=linux GOARCH=386 go build -o build/cmdo-linux-386
GOOS=linux GOARCH=arm64 go build -o build/cmdo-linux-arm64

echo "Build complete! Files in ./build/ directory:"
ls -la build/


# dekho mene git hub par 3no files dal di see

# https://github.com/tanu2534/cmdo-release/blob/main/install.sh

# https://github.com/tanu2534/cmdo-release/blob/main/cmdo-macos-arm64

