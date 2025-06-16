#!/bin/bash
set -e

echo "🚀 Deploying Personal Website 2025..."

# Get the script directory to ensure relative paths work
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "📁 Project root: $PROJECT_ROOT"
cd "$PROJECT_ROOT"

# Build all components
echo "Step 1: Building Go compiler..."
./scripts/personal_website_2025/build-go-compiler.sh

echo "Step 2: Building runtime..."
./scripts/personal_website_2025/build-runtime.sh

echo "Step 3: Validating deployment..."
REQUIRED_FILES=(
    "static/personal_website_2025.html"
    "static/bin/personal-website-2025-go-compiler.wasm"
    "static/bin/main.wasm"
    "static/script/wasm_exec.js"
)

MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        MISSING_FILES+=("$file")
    fi
done

if [ ${#MISSING_FILES[@]} -eq 0 ]; then
    echo "✅ Personal Website 2025 deployed successfully!"
    echo "🌐 Open: static/personal_website_2025.html"
    echo "📁 Compiler: static/bin/personal-website-2025-go-compiler.wasm"
    echo "📁 Runtime: static/bin/main.wasm"
else
    echo "❌ Deployment incomplete. Missing files:"
    for file in "${MISSING_FILES[@]}"; do
        echo "   - $file"
    done
    exit 1
fi 