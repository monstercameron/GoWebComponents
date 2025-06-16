#!/bin/bash
set -e

echo "🔨 Building Personal Website 2025 Go compiler..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

# Create output directory
mkdir -p static/bin

# Build Go compiler to WASM
echo "📦 Compiling Go compiler to WASM..."
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o static/bin/personal-website-2025-go-compiler.wasm ./cmd/wasm-compiler

# Check if build was successful
if [ -f "static/bin/personal-website-2025-go-compiler.wasm" ]; then
    SIZE=$(stat -c%s "static/bin/personal-website-2025-go-compiler.wasm" 2>/dev/null || stat -f%z "static/bin/personal-website-2025-go-compiler.wasm")
    echo "✅ Compiler built successfully: static/bin/personal-website-2025-go-compiler.wasm ($(($SIZE / 1024))KB)"
else
    echo "❌ Build failed"
    exit 1
fi 