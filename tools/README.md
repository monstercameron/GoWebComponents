# Tools

**Location:** `/tools`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/
├── render/
├── router/
├── fetch/
├── internal/
├── examples/
├── test/
├── tools/            ← YOU ARE HERE
│   ├── build.ps1
│   ├── build.sh
│   ├── livereload.ps1
│   ├── livereload.sh
│   ├── pages.sh
│   ├── serve.ps1
│   └── livereload/
│       └── livereload.go
└── utils/
```

## Overview

Development tools and build scripts for GoWebComponents projects.

## Build Scripts

### `build.sh` / `build.ps1`

Compiles Go code to WebAssembly.

**Linux/macOS:**

```bash
./tools/build.sh
```

**Windows (PowerShell):**

```powershell
.\tools\build.ps1
```

**Output:**

- Compiles `main.go` to `static/bin/main.wasm`
- Copies `wasm_exec.js` from Go installation
- Shows build time and file size

**Usage in Projects:**

```bash
# From project root
./tools/build.sh

# Custom source file
./tools/build.sh app.go

# Custom output
./tools/build.sh main.go custom/output.wasm
```

---

### `serve.ps1`

Simple HTTP server for testing WASM applications.

**Windows (PowerShell):**

```powershell
.\tools\serve.ps1
```

**Features:**

- Serves files from `static/` directory
- Runs on port 8080
- Proper MIME types for `.wasm` files
- CORS headers enabled

**Requirements:**

- Python 3 with http.server module

---

### `livereload.sh` / `livereload.ps1`

Development server with automatic rebuild and hot reload.

**Linux/macOS:**

```bash
./tools/livereload.sh
```

**Windows (PowerShell):**

```powershell
.\tools\livereload.ps1
```

**Features:**

- Watches `.go` files for changes
- Auto-rebuilds WASM on file save
- WebSocket-based hot reload
- State preservation across reloads
- Build status notifications
- Debouncing (2s delay after last change)

**Server Output:**

```
🔄 Live reload started. Watching for .go file changes...
📂 Watching directory: /your-project
🌐 Server running on http://localhost:8080

👀 Watching: /your-project/components
👀 Watching: /your-project/pages
🔨 Starting WASM build...
✅ Build completed successfully in 1.2s
```

**Browser Features:**

- Connection status indicator
- Build progress display
- Auto-refresh on successful build
- Error reporting in console

**Configuration:**

Edit `livereload/livereload.go` to customize:

```go
const (
    debounceTime      = 2000 * time.Millisecond // Wait time after file change
    serverPort        = ":8080"                 // HTTP server port
)
```

---

### `pages.sh`

GitHub Pages deployment script.

**Linux/macOS:**

```bash
./tools/pages.sh
```

**What it does:**

1. Builds WASM binary
2. Creates `gh-pages` branch
3. Copies static files
4. Commits and pushes to GitHub

**Requirements:**

- Git repository
- GitHub remote configured
- Write access to repository

**Usage:**

```bash
# First time setup
git remote add origin https://github.com/user/repo.git

# Deploy
./tools/pages.sh
```

---

## Live Reload System

### Architecture

```
┌─────────────────┐
│   File Watcher  │
│   (Go runtime)  │
└────────┬────────┘
         │ File changed
         ▼
┌─────────────────┐
│  Build System   │
│ (go build WASM) │
└────────┬────────┘
         │ Build complete
         ▼
┌─────────────────┐
│  WebSocket Hub  │
│ (Notify clients)│
└────────┬────────┘
         │ Reload signal
         ▼
┌─────────────────┐
│     Browser     │
│ (Hot reload UI) │
└─────────────────┘
```

### Implementation

**`livereload/livereload.go`** - Main server:

- HTTP server for static files
- WebSocket server for reload notifications
- File system watcher
- Build orchestration
- Debouncing logic

**WebSocket Protocol:**

Client → Server:

```json
{ "type": "ping" }
```

Server → Client:

```json
{"type": "building", "message": "Building WASM..."}
{"type": "success", "message": "Build completed in 1.2s"}
{"type": "error", "message": "Build failed: syntax error"}
{"type": "reload"}
```

**Client-Side (Auto-injected):**

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === "reload") {
    window.location.reload();
  }
};
```

### Advanced Features

**State Preservation:**

Export/import functions in your Go app:

```go
//go:export exportAppState
func exportAppState() js.Value {
    return js.ValueOf(map[string]interface{}{
        "count": currentCount,
        "user":  currentUser,
    })
}

//go:export importAppState
func importAppState(state js.Value) {
    currentCount = state.Get("count").Int()
    currentUser = state.Get("user").String()
}
```

**Custom Reload Logic:**

```go
//go:export hotReloadWasm
func hotReloadWasm() {
    // Re-initialize without full page reload
    reinitializeComponents()
}
```

---

## Tool Scripts

### Creating New Tool

1. Create script in `/tools/`
2. Make executable (Linux/macOS): `chmod +x tool.sh`
3. Add documentation to this README

**Template:**

```bash
#!/bin/bash
# tools/mytool.sh
# Description: What this tool does

set -e  # Exit on error

echo "Running my tool..."

# Tool logic here

echo "✅ Done!"
```

---

## Related

- **[/examples](../examples/)** - Uses build scripts for examples
- **[/test](../test/)** - Uses test server
- **[Main README](../)** - Project overview

## Contributing

When adding new tools:

1. Support both Windows (PowerShell) and Unix (bash) if possible
2. Include error handling
3. Provide clear output messages
4. Document in this README
5. Add usage examples
