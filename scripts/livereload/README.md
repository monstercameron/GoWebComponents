# Live Reload for Go WASM Development

This directory contains a live reload system for Go WASM development that automatically rebuilds your WASM binary when Go files change.

## Features

- 🔄 **Auto-rebuild**: Watches for changes in `.go` files and automatically triggers a rebuild
- ⏱️ **Debouncing**: 1000ms debounce time to avoid excessive rebuilds during rapid file changes
- 🛑 **Process Management**: Kills running builds when new changes are detected
- 📂 **Smart Watching**: Recursively watches all directories while excluding common ignored folders (`.git`, `vendor`, `node_modules`)
- 🎯 **Targeted**: Only watches `.go` files, ignoring temporary and backup files
- 💬 **Verbose Output**: Clear console messages showing build status and timing

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
cd scripts
go run livereload.go
```

## How It Works

1. **File Watching**: Uses `fsnotify` to watch for filesystem events
2. **Debouncing**: When a `.go` file changes, starts a 1000ms timer
3. **Build Cancellation**: If another change occurs during the debounce period, kills any running build process and resets the timer
4. **WASM Build**: Executes the equivalent of your `build.sh` script:
   ```bash
   GOOS=js GOARCH=wasm go build -o static/bin/main.wasm
   ```
5. **Process Management**: Tracks the build process PID and can kill it if needed

## Configuration

The live reloader is configured via constants and variables in `livereload.go`:

```go
const (
    debounceTime = 1000 * time.Millisecond  // Debounce period
    buildCommand = "go"                     // Build command
)

var (
    buildArgs = []string{"build", "-o", "static/bin/main.wasm"}  // Build arguments
    buildEnv  = []string{"GOOS=js", "GOARCH=wasm"}              // Environment variables
)
```

## Dependencies

The live reloader requires:

- Go 1.21 or later
- `github.com/fsnotify/fsnotify` (automatically downloaded via `go mod tidy`)

## Output Example

```
🔄 Live reload started. Watching for .go file changes...
📂 Watching directory: /path/to/GoWebComponents
⏱️  Debounce time: 1s
🛑 Press Ctrl+C to stop
👀 Watching: /path/to/GoWebComponents
👀 Watching: /path/to/GoWebComponents/fiber
👀 Watching: /path/to/GoWebComponents/scripts
👀 Watching: /path/to/GoWebComponents/static
🔨 Starting WASM build...
✅ Build completed successfully in 2.5s

📝 File changed: /path/to/GoWebComponents/fiber/fiber.go
⏲️  Debouncing... will build in 1s
🔨 Starting WASM build...
✅ Build completed successfully in 1.8s
```

## Graceful Shutdown

The live reloader handles graceful shutdown:

- Press `Ctrl+C` to stop
- Automatically kills any running build processes
- Cleans up file watchers and timers

## Troubleshooting

**Build fails:**
- Check that you're in the correct directory (project root)
- Ensure your Go environment is set up correctly
- Verify that the `static/bin/` directory exists

**Permission errors:**
- On Unix systems, make sure the shell scripts are executable: `chmod +x livereload.sh`
- Ensure Go has write permissions to the output directory

**High CPU usage:**
- The file watcher is designed to be efficient, but very large projects with many files might cause higher CPU usage
- Consider adding more exclusion patterns in the `addWatchers` function if needed 