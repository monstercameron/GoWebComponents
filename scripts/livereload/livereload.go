package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	debounceTime = 1000 * time.Millisecond
	buildCommand = "go"
	serverPort   = ":8080"
)

var (
	buildArgs = []string{"build", "-o", "static/bin/main.wasm"}
	buildEnv  = []string{"GOOS=js", "GOARCH=wasm"}
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin in development
		},
	}
)

// WebSocket message types
type MessageType string

const (
	MessageTypeBuildStart    MessageType = "build_start"
	MessageTypeBuildComplete MessageType = "build_complete"
	MessageTypeBuildError    MessageType = "build_error"
	MessageTypeReload        MessageType = "reload"
	MessageTypeStateExport   MessageType = "state_export"
	MessageTypeStateImport   MessageType = "state_import"
)

type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type BuildStatus struct {
	Success  bool   `json:"success"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

type LiveReloadServer struct {
	watcher       *fsnotify.Watcher
	mutex         sync.Mutex
	currentBuild  *exec.Cmd
	debounceTimer *time.Timer
	projectRoot   string
	clients       map[*websocket.Conn]bool
	clientsMutex  sync.RWMutex
	httpServer    *http.Server
}

func NewLiveReloadServer(projectRoot string) (*LiveReloadServer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &LiveReloadServer{
		watcher:     watcher,
		projectRoot: projectRoot,
		clients:     make(map[*websocket.Conn]bool),
	}, nil
}

func (lrs *LiveReloadServer) Start() error {
	// Add the project root and subdirectories to the watcher
	err := lrs.addWatchers(lrs.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to add watchers: %w", err)
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", lrs.handleIndex)
	mux.HandleFunc("/ws", lrs.handleWebSocket)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(lrs.projectRoot, "static")))))
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(filepath.Join(lrs.projectRoot, "static/images")))))
	mux.Handle("/script/", http.StripPrefix("/script/", http.FileServer(http.Dir(filepath.Join(lrs.projectRoot, "static/script")))))
	mux.Handle("/bin/", http.StripPrefix("/bin/", http.FileServer(http.Dir(filepath.Join(lrs.projectRoot, "static/bin")))))

	lrs.httpServer = &http.Server{
		Addr:    serverPort,
		Handler: mux,
	}

	// Start HTTP server in a goroutine
	go func() {
		fmt.Printf("🌐 Live reload server starting on http://localhost%s\n", serverPort)
		if err := lrs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ HTTP server error: %v", err)
		}
	}()

	// Handle interrupt signal for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🔄 Live reload started. Watching for .go file changes...")
	fmt.Printf("📂 Watching directory: %s\n", lrs.projectRoot)
	fmt.Printf("⏱️  Debounce time: %v\n", debounceTime)
	fmt.Printf("🌐 Server running on http://localhost%s\n", serverPort)
	fmt.Println("🛑 Press Ctrl+C to stop")

	// Trigger initial build
	lrs.triggerBuild()

	go func() {
		for {
			select {
			case event, ok := <-lrs.watcher.Events:
				if !ok {
					return
				}
				lrs.handleFileEvent(event)

			case err, ok := <-lrs.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("❌ Watcher error: %v", err)

			case <-c:
				fmt.Println("\n🛑 Shutting down live reload server...")
				lrs.cleanup()
				os.Exit(0)
			}
		}
	}()

	// Keep the main goroutine alive
	select {}
}

func (lrs *LiveReloadServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(lrs.projectRoot, "static", "index.html")

	// Read the original index.html
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		http.Error(w, "Could not read index.html", http.StatusInternalServerError)
		return
	}

	// Inject the live reload script before closing </body> tag
	liveReloadScript := `
<script>
// Live Reload Client Script
(function() {
    'use strict';
    
    let ws;
    let reconnectTimer = null;
    const reconnectDelay = 5000; // 5 seconds
    
    // State storage in memory
    let storedState = null;
    
    // State management
    window.GoLiveReload = {
        exportState: function() {
            // Try to export WASM app state if available
            if (window.exportAppState && typeof window.exportAppState === 'function') {
                try {
                    const wasmState = window.exportAppState();
                    console.log('🔄 GoLiveReload: Exported WASM state:', wasmState);
                    return wasmState;
                } catch (e) {
                    console.warn('🚨 Failed to export WASM state:', e);
                }
            }
            
            return null;
        },
        
        importState: function(state) {
            if (!state) return;
            
            // Try to import WASM app state if available
            if (window.importAppState && typeof window.importAppState === 'function') {
                try {
                    console.log('🔄 GoLiveReload: Importing WASM state:', state);
                    window.importAppState(state);
                } catch (e) {
                    console.warn('🚨 Failed to import WASM state:', e);
                }
            }
        },
        
        storeState: function(state) {
            storedState = state;
            console.log('💾 GoLiveReload: State stored in memory:', state);
        },
        
        getStoredState: function() {
            console.log('📥 GoLiveReload: Retrieved stored state:', storedState);
            return storedState;
        },
        
        clearStoredState: function() {
            console.log('🧹 GoLiveReload: Cleared stored state');
            storedState = null;
        },
        
        onReload: function(callback) {
            document.addEventListener('beforeunload', callback);
        }
    };
    
    function connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = protocol + '//' + window.location.host + '/ws';
        
        ws = new WebSocket(wsUrl);
        
        ws.onopen = function() {
            console.log('🔄 Live reload connected');
            showStatus('Connected', 'success');
            
            // Clear any existing reconnect timer
            if (reconnectTimer) {
                clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
        };
        
        ws.onmessage = function(event) {
            try {
                const message = JSON.parse(event.data);
                handleMessage(message);
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }
        };
        
        ws.onclose = function() {
            console.log('🔄 Live reload disconnected');
            showStatus('Disconnected', 'error');
            
            // Always try to reconnect every 5 seconds
            if (!reconnectTimer) {
                reconnectTimer = setTimeout(function() {
                    reconnectTimer = null;
                    connect();
                }, reconnectDelay);
            }
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
        };
    }
    
    function handleMessage(message) {
        console.log('📨 Live reload message:', message.type);
        
        switch (message.type) {
            case 'build_start':
                showStatus('Building...', 'building');
                break;
                
                         case 'build_complete':
                 if (message.payload && message.payload.success) {
                     showStatus('Build successful', 'success');
                     
                     // Export and store current state before reload
                     const currentState = window.GoLiveReload.exportState();
                     if (currentState) {
                         window.GoLiveReload.storeState(currentState);
                     }
                     
                     // Hot reload the WASM module
                     setTimeout(() => {
                         hotReloadWasm();
                     }, 500);
                 } else {
                     showStatus('Build failed', 'error');
                 }
                 break;
                
            case 'build_error':
                showStatus('Build error: ' + (message.payload || 'Unknown error'), 'error');
                break;
                
            case 'reload':
                location.reload();
                break;
        }
    }
    
    function hotReloadWasm() {
        // Try hot reload first, fallback to full page reload
        try {
            if (window.hotReloadWasm && typeof window.hotReloadWasm === 'function') {
                window.hotReloadWasm();
            } else {
                // Fallback to full page reload
                location.reload();
            }
        } catch (e) {
            console.warn('Hot reload failed, falling back to full page reload:', e);
            location.reload();
        }
    }
    
    function showStatus(message, type) {
        // Remove existing status
        const existing = document.getElementById('livereload-status');
        if (existing) {
            existing.remove();
        }
        
        // Create status element
        const status = document.createElement('div');
        status.id = 'livereload-status';
        status.textContent = message;
        
        const colors = {
            success: '#10B981',
            error: '#EF4444',
            building: '#F59E0B',
            info: '#3B82F6'
        };
        
        Object.assign(status.style, {
            position: 'fixed',
            top: '10px',
            right: '10px',
            padding: '8px 16px',
            backgroundColor: colors[type] || colors.info,
            color: 'white',
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '12px',
            zIndex: '10000',
            boxShadow: '0 2px 4px rgba(0,0,0,0.2)'
        });
        
        document.body.appendChild(status);
        
        // Auto-remove success messages
        if (type === 'success') {
            setTimeout(() => {
                if (status.parentNode) {
                    status.remove();
                }
            }, 3000);
        }
    }
    
         // Restore state on page load
     document.addEventListener('DOMContentLoaded', function() {
         // Delay state restoration to ensure WASM is loaded
         setTimeout(() => {
             const savedState = window.GoLiveReload.getStoredState();
             if (savedState) {
                 try {
                     console.log('🔄 Restoring state after page load...');
                     window.GoLiveReload.importState(savedState);
                     window.GoLiveReload.clearStoredState();
                 } catch (e) {
                     console.warn('🚨 Failed to restore state:', e);
                 }
             }
         }, 1000);
     });
    
    // Connect on load
    connect();
})();
</script>`

	// Insert the script before closing </body> tag
	modifiedContent := strings.Replace(string(indexContent), "</body>", liveReloadScript+"\n</body>", 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(modifiedContent))
}

func (lrs *LiveReloadServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Add client to the list
	lrs.clientsMutex.Lock()
	lrs.clients[conn] = true
	lrs.clientsMutex.Unlock()

	fmt.Printf("🔌 WebSocket client connected (total: %d)\n", len(lrs.clients))

	// Remove client when done
	defer func() {
		lrs.clientsMutex.Lock()
		delete(lrs.clients, conn)
		lrs.clientsMutex.Unlock()
		fmt.Printf("🔌 WebSocket client disconnected (remaining: %d)\n", len(lrs.clients))
	}()

	// Keep connection alive and handle messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (lrs *LiveReloadServer) broadcastMessage(msgType MessageType, payload interface{}) {
	message := WebSocketMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal WebSocket message: %v", err)
		return
	}

	lrs.clientsMutex.RLock()
	defer lrs.clientsMutex.RUnlock()

	for conn := range lrs.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("⚠️  Failed to send message to client: %v", err)
		}
	}
}

func (lrs *LiveReloadServer) addWatchers(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories, vendor, and node_modules
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "bin" {
				return filepath.SkipDir
			}

			// Add the directory to the watcher
			err := lrs.watcher.Add(path)
			if err != nil {
				log.Printf("⚠️  Failed to watch directory %s: %v", path, err)
			} else {
				fmt.Printf("👀 Watching: %s\n", path)
			}
		}

		return nil
	})
}

func (lrs *LiveReloadServer) handleFileEvent(event fsnotify.Event) {
	// Only handle .go files
	if !strings.HasSuffix(event.Name, ".go") {
		return
	}

	// Skip temporary files and test files
	if strings.Contains(event.Name, ".tmp") || strings.Contains(event.Name, "~") {
		return
	}

	// Only handle write and create events
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		fmt.Printf("📝 File changed: %s\n", event.Name)
		lrs.debounceAndBuild()
	}
}

func (lrs *LiveReloadServer) debounceAndBuild() {
	lrs.mutex.Lock()
	defer lrs.mutex.Unlock()

	// Kill current build if running
	if lrs.currentBuild != nil && lrs.currentBuild.Process != nil {
		fmt.Printf("⏹️  Killing current build process (PID: %d)...\n", lrs.currentBuild.Process.Pid)
		err := lrs.currentBuild.Process.Kill()
		if err != nil {
			log.Printf("❌ Failed to kill current build: %v", err)
		} else {
			fmt.Println("✅ Previous build process killed successfully")
		}
		lrs.currentBuild = nil
	}

	// Reset the debounce timer
	if lrs.debounceTimer != nil {
		lrs.debounceTimer.Stop()
		fmt.Println("⏱️  Previous debounce timer cancelled")
	}

	lrs.debounceTimer = time.AfterFunc(debounceTime, func() {
		lrs.triggerBuild()
	})

	fmt.Printf("⏲️  Debouncing... will build in %v\n", debounceTime)
}

func (lrs *LiveReloadServer) triggerBuild() {
	lrs.mutex.Lock()
	defer lrs.mutex.Unlock()

	fmt.Println("🔨 Starting WASM build...")
	startTime := time.Now()

	// Notify clients that build started
	lrs.broadcastMessage(MessageTypeBuildStart, nil)

	// Create the build command
	cmd := exec.Command(buildCommand, buildArgs...)

	cmd.Dir = lrs.projectRoot
	cmd.Env = append(os.Environ(), buildEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	lrs.currentBuild = cmd
	fmt.Printf("🏗️  Build process started (PID: will be available after start)\n")

	// Start the build process (non-blocking)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start build: %v\n", err)
		lrs.currentBuild = nil
		lrs.broadcastMessage(MessageTypeBuildError, fmt.Sprintf("Failed to start build: %v", err))
		return
	}

	fmt.Printf("🏗️  Build process running (PID: %d)\n", cmd.Process.Pid)

	// Wait for the build to complete
	err = cmd.Wait()

	duration := time.Since(startTime)

	// Check if the process was killed vs completed naturally
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.String() == "signal: killed" {
			fmt.Printf("⏹️  Build was cancelled after %v\n", duration)
		} else {
			fmt.Printf("❌ Build failed after %v: %v\n", duration, err)
			lrs.broadcastMessage(MessageTypeBuildComplete, BuildStatus{
				Success: false,
				Error:   err.Error(),
			})
		}
	} else {
		fmt.Printf("✅ Build completed successfully in %v\n", duration)
		lrs.broadcastMessage(MessageTypeBuildComplete, BuildStatus{
			Success:  true,
			Duration: duration.String(),
		})
	}

	lrs.currentBuild = nil
}

func (lrs *LiveReloadServer) cleanup() {
	lrs.mutex.Lock()
	defer lrs.mutex.Unlock()

	if lrs.debounceTimer != nil {
		lrs.debounceTimer.Stop()
	}

	if lrs.currentBuild != nil && lrs.currentBuild.Process != nil {
		fmt.Println("🛑 Killing running build process...")
		lrs.currentBuild.Process.Kill()
		lrs.currentBuild.Wait()
	}

	if lrs.watcher != nil {
		lrs.watcher.Close()
	}

	if lrs.httpServer != nil {
		lrs.httpServer.Close()
	}

	// Close all WebSocket connections
	lrs.clientsMutex.Lock()
	for conn := range lrs.clients {
		conn.Close()
	}
	lrs.clientsMutex.Unlock()
}

func main() {
	// Get the project root (parent directory of scripts)
	scriptDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Failed to get current directory: %v", err)
	}

	// If we're in the livereload directory, go up two levels to reach project root
	if filepath.Base(scriptDir) == "livereload" {
		scriptDir = filepath.Dir(filepath.Dir(scriptDir))
	} else if filepath.Base(scriptDir) == "scripts" {
		// If we're in the scripts directory, go up one level
		scriptDir = filepath.Dir(scriptDir)
	}

	server, err := NewLiveReloadServer(scriptDir)
	if err != nil {
		log.Fatalf("❌ Failed to create live reload server: %v", err)
	}

	defer server.cleanup()

	err = server.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start live reload server: %v", err)
	}
}
