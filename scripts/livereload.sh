#!/bin/bash

# Live reload script for Go WASM development
# This script watches for .go file changes and automatically rebuilds the WASM binary

echo "🚀 Starting Go WASM Live Reload..."
echo "📦 Building from project root..."

# Navigate to the scripts directory if not already there
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/livereload"

# Run the live reloader
go run livereload.go 