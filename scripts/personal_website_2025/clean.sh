#!/bin/bash
set -e

echo "🧹 Cleaning Personal Website 2025 build artifacts..."

# Get the script directory to ensure relative paths work
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "📁 Project root: $PROJECT_ROOT"
cd "$PROJECT_ROOT"

# Remove WASM files
echo "🗑️  Removing WASM binaries..."
rm -f static/bin/personal-website-2025-*.wasm

# Remove JavaScript files (if any are generated)
echo "🗑️  Removing generated JavaScript files..."
rm -f static/script/personal-website-2025-*.js

# Remove HTML file (if regenerated)
echo "🗑️  Removing HTML file..."
rm -f static/personal_website_2025.html

# List what was removed
REMOVED_FILES=()
if [ ! -f "static/bin/personal-website-2025-go-compiler.wasm" ]; then
    REMOVED_FILES+=("personal-website-2025-go-compiler.wasm")
fi

if [ ${#REMOVED_FILES[@]} -gt 0 ]; then
    echo "✅ Cleaned the following files:"
    for file in "${REMOVED_FILES[@]}"; do
        echo "   - $file"
    done
else
    echo "✅ No artifacts found to clean"
fi

echo "✅ Clean complete." 