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
	debounceTime      = 2000 * time.Millisecond // Wait 2 seconds after last keystroke
	quickDebounceTime = 500 * time.Millisecond  // Quick debounce for single file changes
	maxDebounceTime   = 5000 * time.Millisecond // Maximum wait time before forcing build
	buildCommand      = "go"
	serverPort        = ":8080"
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
	MessageTypeBuildStart     MessageType = "build_start"
	MessageTypeBuildComplete  MessageType = "build_complete"
	MessageTypeBuildError     MessageType = "build_error"
	MessageTypeReload         MessageType = "reload"
	MessageTypeHotReload      MessageType = "hot_reload"
	MessageTypeStateExport    MessageType = "state_export"
	MessageTypeStateImport    MessageType = "state_import"
	MessageTypeDebounceStatus MessageType = "debounce_status"
)

type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type BuildStatus struct {
	Success    bool   `json:"success"`
	Duration   string `json:"duration,omitempty"`
	Error      string `json:"error,omitempty"`
	ReloadType string `json:"reloadType,omitempty"` // "hot" or "full"
}

// UpdateClassification represents the type of update detected
type UpdateClassification struct {
	Type         string   `json:"type"`       // "small" or "big"
	ReloadType   string   `json:"reloadType"` // "hot" or "full"
	Reason       string   `json:"reason"`     // explanation for the classification
	ChangedFiles []string `json:"changedFiles"`
}

type LiveReloadServer struct {
	watcher            *fsnotify.Watcher
	mutex              sync.Mutex
	currentBuild       *exec.Cmd
	debounceTimer      *time.Timer
	maxDebounceTimer   *time.Timer
	projectRoot        string
	clients            map[*websocket.Conn]bool
	clientsMutex       sync.RWMutex
	httpServer         *http.Server
	changedFiles       map[string]time.Time // Track changed files for update classification
	lastClassification UpdateClassification // Store the last classification
	firstChangeTime    time.Time            // Track when the first change occurred
	changeCount        int                  // Count of changes in current batch
}

func NewLiveReloadServer(projectRoot string) (*LiveReloadServer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &LiveReloadServer{
		watcher:      watcher,
		projectRoot:  projectRoot,
		clients:      make(map[*websocket.Conn]bool),
		changedFiles: make(map[string]time.Time),
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
                const classification = message.payload?.classification;
                if (classification) {
                    console.log('🔍 Update classification:', classification.type, '(' + classification.reloadType + ') -', classification.reason);
                    showStatus('Building (' + classification.reloadType + ' reload)...', 'building');
                } else {
                    showStatus('Building...', 'building');
                }
                break;
                
            case 'build_complete':
                if (message.payload && message.payload.success) {
                    const reloadType = message.payload.reloadType || 'full';
                    console.log('✅ Build successful, reload type:', reloadType);
                    
                    if (reloadType === 'hot') {
                        showStatus('Hot reloading...', 'success');
                        performHotReload();
                    } else {
                        showStatus('Full page reloading...', 'success');
                        performFullReload();
                    }
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
                
            case 'hot_reload':
                performHotReload();
                break;
                
            case 'debounce_status':
                handleDebounceStatus(message.payload);
                break;
        }
    }
    
    function performHotReload() {
        console.log('🔥 Attempting hot reload...');
        
        // Export and store current state before reload
        const currentState = window.GoLiveReload.exportState();
        if (currentState) {
            window.GoLiveReload.storeState(currentState);
            console.log('💾 State saved for hot reload');
        }
        
        // Try hot reload with WASM module replacement
        setTimeout(() => {
            try {
                if (window.hotReloadWasm && typeof window.hotReloadWasm === 'function') {
                    console.log('🔥 Calling WASM hot reload function');
                    window.hotReloadWasm();
                } else {
                    console.log('⚠️ WASM hot reload not available, trying manual reload');
                    // Try to reload just the WASM module
                    reloadWasmModule();
                }
            } catch (e) {
                console.warn('🚨 Hot reload failed, falling back to full page reload:', e);
                performFullReload();
            }
        }, 100);
    }
    
    function performFullReload() {
        console.log('🔄 Performing full page reload...');
        
        // Export and store current state before reload
        const currentState = window.GoLiveReload.exportState();
        if (currentState) {
            window.GoLiveReload.storeState(currentState);
            console.log('💾 State saved for full reload');
        }
        
        // Full page reload
        setTimeout(() => {
            location.reload();
        }, 100);
    }
    
    function reloadWasmModule() {
        console.log('🔄 Attempting WASM module reload...');
        
        // Try to find and reload the WASM script
        const wasmScript = document.querySelector('script[src*="wasm_exec.js"]');
        if (wasmScript) {
            // Create a new script element
            const newScript = document.createElement('script');
            newScript.src = wasmScript.src + '?t=' + Date.now();
            newScript.onload = function() {
                console.log('✅ WASM script reloaded');
                // Try to reinitialize the Go WASM
                if (window.Go) {
                    const go = new Go();
                    WebAssembly.instantiateStreaming(fetch('/bin/main.wasm?t=' + Date.now()), go.importObject)
                        .then((result) => {
                            console.log('✅ WASM module reloaded');
                            go.run(result.instance);
                        })
                        .catch((e) => {
                            console.warn('🚨 WASM module reload failed:', e);
                            performFullReload();
                        });
                } else {
                    console.warn('⚠️ Go WASM runtime not available');
                    performFullReload();
                }
            };
            newScript.onerror = function() {
                console.warn('🚨 WASM script reload failed');
                performFullReload();
            };
            
            // Replace the old script
            wasmScript.parentNode.replaceChild(newScript, wasmScript);
        } else {
            console.warn('⚠️ WASM script not found, falling back to full reload');
            performFullReload();
        }
    }
    
    function handleDebounceStatus(payload) {
        if (!payload) return;
        
        const { changeCount, timeSinceFirstChange, waitTime, maxWaitTime } = payload;
        const remainingTime = Math.max(0, waitTime);
        const totalElapsed = timeSinceFirstChange;
        
        let message = 'Waiting for changes... (' + changeCount + ' change' + (changeCount > 1 ? 's' : '') + ')';
        if (remainingTime > 0) {
            message += ' ' + (remainingTime / 1000).toFixed(1) + 's';
        }
        
        // Show progress if we're approaching max wait time
        if (totalElapsed > maxWaitTime * 0.5) {
            const progress = Math.min(100, (totalElapsed / maxWaitTime) * 100);
            message += ' [' + progress.toFixed(0) + '%]';
        }
        
        showStatus(message, 'waiting', false); // Don't auto-remove
    }
    
    function showStatus(message, type, autoRemove = true) {
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
            info: '#3B82F6',
            waiting: '#8B5CF6'
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
            boxShadow: '0 2px 4px rgba(0,0,0,0.2)',
            transition: 'all 0.3s ease'
        });
        
        document.body.appendChild(status);
        
        // Auto-remove success messages or when specified
        if (autoRemove && (type === 'success' || type === 'waiting')) {
            setTimeout(() => {
                if (status.parentNode) {
                    status.remove();
                }
            }, type === 'waiting' ? 1000 : 3000);
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

		// Track the changed file
		lrs.mutex.Lock()
		lrs.changedFiles[event.Name] = time.Now()
		lrs.mutex.Unlock()

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

	now := time.Now()

	// Track if this is the first change in a batch
	if lrs.firstChangeTime.IsZero() {
		lrs.firstChangeTime = now
		lrs.changeCount = 0
	}

	lrs.changeCount++

	// Calculate smart debounce time based on change pattern
	var smartDebounceTime time.Duration
	timeSinceFirstChange := now.Sub(lrs.firstChangeTime)

	if lrs.changeCount == 1 {
		// First change - use longer debounce to give user time to continue typing
		smartDebounceTime = debounceTime
		fmt.Printf("⏱️  First change detected, waiting %v for more changes...\n", smartDebounceTime)
	} else if lrs.changeCount <= 3 && timeSinceFirstChange < 10*time.Second {
		// Multiple quick changes - user is actively typing, extend wait
		smartDebounceTime = debounceTime
		fmt.Printf("⏱️  Change #%d detected, extending wait %v (user actively typing)...\n", lrs.changeCount, smartDebounceTime)
	} else {
		// Many changes or been waiting a while - use shorter debounce
		smartDebounceTime = quickDebounceTime
		fmt.Printf("⏱️  Change #%d detected, using quick debounce %v...\n", lrs.changeCount, smartDebounceTime)
	}

	// Don't wait longer than maxDebounceTime total
	if timeSinceFirstChange > maxDebounceTime-smartDebounceTime {
		smartDebounceTime = maxDebounceTime - timeSinceFirstChange
		if smartDebounceTime <= 0 {
			fmt.Printf("⏰ Maximum debounce time reached, building immediately\n")
			lrs.resetDebounceState()
			go lrs.triggerBuild()
			return
		}
		fmt.Printf("⏰ Approaching max debounce time, will build in %v\n", smartDebounceTime)
	}

	// Reset the debounce timer
	if lrs.debounceTimer != nil {
		lrs.debounceTimer.Stop()
	}

	// Reset max debounce timer if this is the first change
	if lrs.changeCount == 1 {
		if lrs.maxDebounceTimer != nil {
			lrs.maxDebounceTimer.Stop()
		}
		lrs.maxDebounceTimer = time.AfterFunc(maxDebounceTime, func() {
			fmt.Printf("⏰ Maximum debounce time (%v) reached, forcing build\n", maxDebounceTime)
			lrs.mutex.Lock()
			lrs.resetDebounceState()
			lrs.mutex.Unlock()
			lrs.triggerBuild()
		})
	}

	lrs.debounceTimer = time.AfterFunc(smartDebounceTime, func() {
		lrs.mutex.Lock()
		lrs.resetDebounceState()
		lrs.mutex.Unlock()
		lrs.triggerBuild()
	})

	// Notify clients about debouncing status
	lrs.broadcastMessage(MessageTypeDebounceStatus, map[string]interface{}{
		"changeCount":          lrs.changeCount,
		"timeSinceFirstChange": timeSinceFirstChange.Milliseconds(),
		"waitTime":             smartDebounceTime.Milliseconds(),
		"maxWaitTime":          maxDebounceTime.Milliseconds(),
	})
}

// resetDebounceState resets the debouncing state after a build
func (lrs *LiveReloadServer) resetDebounceState() {
	lrs.firstChangeTime = time.Time{}
	lrs.changeCount = 0
	if lrs.maxDebounceTimer != nil {
		lrs.maxDebounceTimer.Stop()
		lrs.maxDebounceTimer = nil
	}
}

// classifyUpdate determines whether the changes require a hot reload or full page reload
func (lrs *LiveReloadServer) classifyUpdate() UpdateClassification {
	lrs.mutex.Lock()
	defer lrs.mutex.Unlock()

	var changedFiles []string
	for file := range lrs.changedFiles {
		changedFiles = append(changedFiles, file)
	}

	// Clear the changed files after classification
	lrs.changedFiles = make(map[string]time.Time)

	if len(changedFiles) == 0 {
		return UpdateClassification{
			Type:         "small",
			ReloadType:   "hot",
			Reason:       "No files changed",
			ChangedFiles: changedFiles,
		}
	}

	// Analyze the changed files to determine update type
	for _, file := range changedFiles {
		relPath, _ := filepath.Rel(lrs.projectRoot, file)

		// Big updates (require full page reload):

		// 1. Main function changes
		if strings.Contains(relPath, "main.go") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "Main function or entry point changed",
				ChangedFiles: changedFiles,
			}
		}

		// 2. Core fiber system changes
		if strings.Contains(relPath, "fiber/fiber.go") ||
			strings.Contains(relPath, "fiber/hooks.go") ||
			strings.Contains(relPath, "fiber/types.go") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "Core fiber system changed",
				ChangedFiles: changedFiles,
			}
		}

		// 3. State management changes
		if strings.Contains(relPath, "fiber/state_management.go") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "State management system changed",
				ChangedFiles: changedFiles,
			}
		}

		// 4. New files or package structure changes
		if strings.Contains(relPath, "go.mod") || strings.Contains(relPath, "go.sum") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "Package dependencies changed",
				ChangedFiles: changedFiles,
			}
		}
	}

	// Small updates (can use hot reload):
	// - Component changes in examples/
	// - Utility functions
	// - Non-core fiber files

	smallUpdateReasons := []string{}
	for _, file := range changedFiles {
		relPath, _ := filepath.Rel(lrs.projectRoot, file)

		if strings.Contains(relPath, "examples/") {
			smallUpdateReasons = append(smallUpdateReasons, "example components")
		} else if strings.Contains(relPath, "fiber/utils.go") ||
			strings.Contains(relPath, "fiber/dom.go") ||
			strings.Contains(relPath, "fiber/events.go") ||
			strings.Contains(relPath, "fiber/fetch.go") {
			smallUpdateReasons = append(smallUpdateReasons, "utility functions")
		}
	}

	if len(smallUpdateReasons) > 0 {
		return UpdateClassification{
			Type:         "small",
			ReloadType:   "hot",
			Reason:       "Small changes: " + strings.Join(smallUpdateReasons, ", "),
			ChangedFiles: changedFiles,
		}
	}

	// Default to full reload for safety
	return UpdateClassification{
		Type:         "big",
		ReloadType:   "full",
		Reason:       "Unknown changes, defaulting to full reload for safety",
		ChangedFiles: changedFiles,
	}
}

func (lrs *LiveReloadServer) triggerBuild() {
	// Classify the update before building
	classification := lrs.classifyUpdate()
	fmt.Printf("🔍 Update classification: %s (%s) - %s\n",
		classification.Type, classification.ReloadType, classification.Reason)

	lrs.mutex.Lock()
	defer lrs.mutex.Unlock()

	// Store the classification for use after build completion
	lrs.lastClassification = classification

	fmt.Println("🔨 Starting WASM build...")
	startTime := time.Now()

	// Notify clients that build started with classification info
	lrs.broadcastMessage(MessageTypeBuildStart, map[string]interface{}{
		"classification": classification,
	})

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
			Success:    true,
			Duration:   duration.String(),
			ReloadType: lrs.lastClassification.ReloadType,
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

	if lrs.maxDebounceTimer != nil {
		lrs.maxDebounceTimer.Stop()
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
