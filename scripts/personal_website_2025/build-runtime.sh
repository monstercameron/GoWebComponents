#!/bin/bash
set -e

echo "🔨 Building Personal Website 2025 runtime..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

# Create output directory
mkdir -p static/bin

# Build main GoWebComponents runtime
echo "📦 Compiling main runtime to WASM..."
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o static/bin/main.wasm ./

# Check if build was successful
if [ -f "static/bin/main.wasm" ]; then
    SIZE=$(stat -c%s "static/bin/main.wasm" 2>/dev/null || stat -f%z "static/bin/main.wasm")
    echo "✅ Runtime built successfully: static/bin/main.wasm ($(($SIZE / 1024))KB)"
else
    echo "❌ Runtime build failed"
    exit 1
fi 