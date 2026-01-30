#!/bin/bash

echo "Building CMDO for macOS ARM64..."

# Build for macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o macos-dist/cmdo-installer-temp/cmdo-macos-arm64

echo "Build complete! Binary created at macos-dist/cmdo-installer-temp/cmdo-macos-arm64"