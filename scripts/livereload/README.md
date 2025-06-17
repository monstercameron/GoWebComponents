# Live Reload Server for Go WASM Development

This directory contains an enhanced live reload server for Go WASM development that provides a complete development environment with HTTP serving, WebSocket communication, and state preservation.

## Features

- 🌐 **HTTP Server**: Serves your static files and index.html with auto-injected live reload functionality
- 🔄 **Auto-rebuild**: Watches for changes in `.go` files and automatically triggers a rebuild
- 📡 **WebSocket Communication**: Real-time build status updates and reload notifications
- 🎯 **State Preservation**: Attempts to preserve WASM application state during hot reloads
- ⏱️ **Debouncing**: 1000ms debounce time to avoid excessive rebuilds during rapid file changes
- 🛑 **Process Management**: Kills running builds when new changes are detected
- 📂 **Smart Watching**: Recursively watches all directories while excluding common ignored folders (`.git`, `vendor`, `node_modules`, `bin`)
- 🎯 **Targeted**: Only watches `.go` files, ignoring temporary and backup files
- 💬 **Verbose Output**: Clear console messages showing build status and timing
- 🔌 **Client Status**: Visual status indicator in the browser showing connection and build status

## Quick Start

### Option 1: Use the convenience scripts

**Linux/macOS:**
```bash
./livereload.sh
```

**Windows (PowerShell):**
```powershell
.\livereload.ps1
```

### Option 2: Run directly

```bash
cd scripts/livereload
go run livereload.go
```

## How It Works

### Server Architecture

1. **HTTP Server**: Runs on `:8080` and serves:
   - `/` - Your `static/index.html` with injected live reload script
   - `/static/*` - Static assets from your `static/` directory
   - `/images/*` - Images from your `static/images/` directory
   - `/ws` - WebSocket endpoint for live reload communication

2. **File Watching**: Uses `fsnotify` to watch for filesystem events in Go files

3. **Build Pipeline**: 
   - Detects `.go` file changes
   - Debounces for 1000ms to avoid excessive builds
   - Kills any running build process
   - Executes WASM build: `GOOS=js GOARCH=wasm go build -o static/bin/main.wasm`
   - Notifies connected clients via WebSocket

4. **Client Communication**: WebSocket messages for:
   - `build_start` - Build process started
   - `build_complete` - Build finished (success/failure)
   - `build_error` - Build error occurred
   - `reload` - Trigger page reload

### State Management

The injected client script provides:

```javascript
window.GoLiveReload = {
    exportState(),      // Export current app state
    importState(state), // Restore app state  
    onReload(callback)  // Register reload handlers
}
```

**State includes:**
- Page scroll position
- Current URL
- Custom WASM application state (if `window.exportAppState()` is available)
- Timestamp and metadata

**Hot Reload Process:**
1. Build completes successfully
2. Client exports current state to `sessionStorage`
3. Attempts hot reload via `window.hotReloadWasm()` if available
4. Falls back to full page reload if hot reload fails
5. On page load, restores state from `sessionStorage`

## Configuration

The live reload server is configured via constants in `livereload.go`:

```go
const (
    debounceTime = 1000 * time.Millisecond  // Debounce period
    buildCommand = "go"                     // Build command
    serverPort   = ":8080"                  // HTTP server port
)

var (
    buildArgs = []string{"build", "-o", "static/bin/main.wasm"}  // Build arguments
    buildEnv  = []string{"GOOS=js", "GOARCH=wasm"}              // Environment variables
)
```

## Integration with Go WASM Applications

To enable state preservation in your Go WASM application, implement these functions:

```go
// Export current application state
//go:export exportAppState
func exportAppState() js.Value {
    state := map[string]interface{}{
        "componentStates": getComponentStates(),
        "globalState": getGlobalState(),
        // Add your application-specific state
    }
    return js.ValueOf(state)
}

// Import and restore application state  
//go:export importAppState
func importAppState(jsState js.Value) {
    // Parse and restore your application state
    restoreComponentStates(jsState.Get("componentStates"))
    restoreGlobalState(jsState.Get("globalState"))
}

// Hot reload the WASM module
//go:export hotReloadWasm
func hotReloadWasm() {
    // Perform hot reload logic
    // This could involve re-initializing components while preserving state
}
```

## Dependencies

The live reload server requires:

- Go 1.21 or later
- `github.com/fsnotify/fsnotify` (file watching)
- `github.com/gorilla/websocket` (WebSocket communication)

## Output Example

```
🔄 Live reload started. Watching for .go file changes...
📂 Watching directory: /path/to/GoWebComponents
⏱️  Debounce time: 1s
🌐 Server running on http://localhost:8080
🛑 Press Ctrl+C to stop
👀 Watching: /path/to/GoWebComponents
👀 Watching: /path/to/GoWebComponents/fiber
👀 Watching: /path/to/GoWebComponents/examples
🔨 Starting WASM build...
✅ Build completed successfully in 2.5s
🔌 WebSocket client connected (total: 1)

📝 File changed: /path/to/GoWebComponents/fiber/fiber.go
⏲️  Debouncing... will build in 1s
🔨 Starting WASM build...
✅ Build completed successfully in 1.8s
```

## Browser Features

When you open `http://localhost:8080`, you'll see:

- **Status Indicator**: Top-right corner shows connection status and build progress
- **Auto-refresh**: Page automatically reloads when builds complete
- **State Preservation**: Scroll position and app state maintained across reloads
- **Error Display**: Build errors shown in the status indicator

## Graceful Shutdown

The live reload server handles graceful shutdown:

- Press `Ctrl+C` to stop
- Automatically kills any running build processes
- Closes all WebSocket connections
- Cleans up file watchers and HTTP server

## Troubleshooting

**Build fails:**
- Check that you're in the correct directory (project root)
- Ensure your Go environment is set up correctly
- Verify that the `static/bin/` directory exists

**Server won't start:**
- Ensure port 8080 is not already in use
- Check that the `static/` directory exists with `index.html`

**WebSocket connection fails:**
- Check browser console for connection errors
- Verify firewall settings allow connections to localhost:8080

**State preservation not working:**
- Implement `exportAppState` and `importAppState` functions in your Go WASM code
- Check browser console for state export/import errors 