#!/bin/bash

# Live reload server for Go WASM development
# This script starts an HTTP server with WebSocket live reload functionality
# Features:
# - Serves static files and index.html with injected live reload script
# - Watches for .go file changes and automatically rebuilds the WASM binary
# - WebSocket communication for real-time build status and hot reload
# - State preservation during hot reloads

echo "🚀 Starting Go WASM Live Reload Server..."
echo "📦 Building from project root..."
echo "🌐 Server will be available at http://localhost:8080"

# Store original directory to restore on exit
ORIGINAL_DIR=$(pwd)

# Function to restore directory on exit
cleanup() {
    echo "🔄 Restoring original directory..."
    cd "$ORIGINAL_DIR"
}

# Set trap to restore directory on script exit or interruption
trap cleanup EXIT INT TERM

# Navigate to the scripts directory if not already there
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/livereload"

# Always build the binary on startup
BINARY_PATH="./livereload"
echo "🔨 Building live reload server..."
go build -o livereload .
if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi
echo "✅ Build completed!"

# Run the live reload server
echo "🚀 Starting live reload server..."
$BINARY_PATH 