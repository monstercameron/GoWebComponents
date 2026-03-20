package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
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
	defaultHost       = "127.0.0.1"
	defaultPort       = "8080"
)

var (
	buildEnv = []string{"GOOS=js", "GOARCH=wasm"}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin in development
		},
	}
)

type LiveReloadOptions struct {
	MainPath         string
	ProjectRoot      string
	IndexPath        string
	OutputPath       string
	Host             string
	Port             string
	AlwaysHotReload  bool
	ClientScriptPath string
}

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
	MessageTypeStateSnapshot  MessageType = "state_snapshot"
	MessageTypeDebounceStatus MessageType = "debounce_status"
	MessageTypeCurrentStatus  MessageType = "current_status"
)

type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type BuildStatus struct {
	Success       bool                      `json:"success"`
	Duration      string                    `json:"duration,omitempty"`
	Error         string                    `json:"error,omitempty"`
	ReloadType    string                    `json:"reloadType,omitempty"` // "hot" or "full"
	StateSnapshot string                    `json:"stateSnapshot,omitempty"`
	ManifestPath  string                    `json:"manifestPath,omitempty"`
	Manifest      *ChangedComponentManifest `json:"manifest,omitempty"`
}

type ChangedComponentManifest struct {
	GeneratedAt  time.Time          `json:"generatedAt"`
	ReloadType   string             `json:"reloadType"`
	Reason       string             `json:"reason,omitempty"`
	ChangedFiles []string           `json:"changedFiles,omitempty"`
	Components   []ChangedComponent `json:"components,omitempty"`
}

type ChangedComponent struct {
	Name          string `json:"name"`
	QualifiedName string `json:"qualifiedName"`
	PackageName   string `json:"packageName,omitempty"`
	PackagePath   string `json:"packagePath,omitempty"`
	File          string `json:"file"`
}

// UpdateClassification represents the type of update detected
type UpdateClassification struct {
	Type         string   `json:"type"`       // "small" or "big"
	ReloadType   string   `json:"reloadType"` // "hot" or "full"
	Reason       string   `json:"reason"`     // explanation for the classification
	ChangedFiles []string `json:"changedFiles"`
}

type LiveReloadServer struct {
	watcher              *fsnotify.Watcher
	mutex                sync.Mutex
	currentBuild         *exec.Cmd
	debounceTimer        *time.Timer
	maxDebounceTimer     *time.Timer
	projectRoot          string
	watchRoot            string
	buildDir             string
	indexPath            string
	outputPath           string
	staticDir            string
	clientScriptPath     string
	host                 string
	port                 string
	alwaysHotReload      bool
	clients              map[*websocket.Conn]bool
	clientsMutex         sync.RWMutex
	httpServer           *http.Server
	changedFiles         map[string]time.Time // Track changed files for update classification
	lastClassification   UpdateClassification // Store the last classification
	firstChangeTime      time.Time            // Track when the first change occurred
	changeCount          int                  // Count of changes in current batch
	lastBuildStatus      *BuildStatus         // Track the last build status for new clients
	pendingStateSnapshot string
	stateSnapshotMu      sync.Mutex
	modulePath           string
	manifestPath         string
}

func NewLiveReloadServer(projectRoot string) (*LiveReloadServer, error) {
	return NewLiveReloadServerWithOptions(LiveReloadOptions{ProjectRoot: projectRoot})
}

func NewLiveReloadServerWithOptions(options LiveReloadOptions) (*LiveReloadServer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	buildDir := strings.TrimSpace(options.MainPath)
	if buildDir != "" {
		buildDir, err = resolveBuildDir(buildDir)
		if err != nil {
			return nil, err
		}
	}
	projectRoot := strings.TrimSpace(options.ProjectRoot)
	if projectRoot == "" {
		if buildDir != "" {
			projectRoot = buildDir
		} else {
			projectRoot = "."
		}
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}
	if buildDir == "" {
		buildDir = projectRoot
	}

	watchRoot := projectRoot
	if moduleRoot := resolveModuleRoot(buildDir); moduleRoot != "" {
		watchRoot = moduleRoot
	}

	indexPath := strings.TrimSpace(options.IndexPath)
	if indexPath == "" {
		indexPath = filepath.Join(projectRoot, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			indexPath = filepath.Join(projectRoot, "static", "index.html")
		}
	} else if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(projectRoot, indexPath)
	}

	outputPath := strings.TrimSpace(options.OutputPath)
	if outputPath == "" {
		outputPath = filepath.Join(buildDir, "main.wasm")
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(buildDir, outputPath)
	}

	staticDir := resolveStaticDir(projectRoot)

	clientScriptPath := strings.TrimSpace(options.ClientScriptPath)
	if clientScriptPath == "" {
		clientScriptPath = resolveClientScriptPath()
	} else if !filepath.IsAbs(clientScriptPath) {
		clientScriptPath = filepath.Join(projectRoot, clientScriptPath)
	}

	host := strings.TrimSpace(options.Host)
	if host == "" {
		host = defaultHost
	}
	port := strings.TrimSpace(options.Port)
	if port == "" {
		port = defaultPort
	}

	manifestPath := outputPath + ".hotreload-manifest.json"

	return &LiveReloadServer{
		watcher:          watcher,
		projectRoot:      projectRoot,
		watchRoot:        watchRoot,
		buildDir:         buildDir,
		indexPath:        indexPath,
		outputPath:       outputPath,
		staticDir:        staticDir,
		clientScriptPath: clientScriptPath,
		host:             host,
		port:             port,
		alwaysHotReload:  options.AlwaysHotReload,
		clients:          make(map[*websocket.Conn]bool),
		changedFiles:     make(map[string]time.Time),
		modulePath:       resolveModulePath(watchRoot),
		manifestPath:     manifestPath,
	}, nil
}

func (lrs *LiveReloadServer) newHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(lrs.projectRoot))
	if lrs.staticDir != "" {
		staticFileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(lrs.staticDir)))
		mux.Handle("/static/", staticFileServer)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			lrs.handleHTML(w, r, lrs.indexPath)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".html") {
			relPath := strings.TrimPrefix(r.URL.Path, "/")
			lrs.handleHTML(w, r, filepath.Join(lrs.projectRoot, filepath.FromSlash(relPath)))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	mux.HandleFunc("/ws", lrs.handleWebSocket)
	return mux
}

func (lrs *LiveReloadServer) Start() error {
	// Watch the module root so shared package changes rebuild example-specific servers.
	err := lrs.addWatchers(lrs.watchRoot)
	if err != nil {
		return fmt.Errorf("failed to add watchers: %w", err)
	}

	lrs.httpServer = &http.Server{
		Addr:    netAddr(lrs.host, lrs.port),
		Handler: lrs.newHTTPHandler(),
	}

	// Start HTTP server in a goroutine
	go func() {
		fmt.Printf("🌐 Live reload server starting on http://%s\n", netAddr(lrs.host, lrs.port))
		if err := lrs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ HTTP server error: %v", err)
		}
	}()

	// Handle interrupt signal for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🔄 Live reload started. Watching for .go file changes...")
	fmt.Printf("📂 Watching directory: %s\n", lrs.watchRoot)
	if lrs.watchRoot != lrs.projectRoot {
		fmt.Printf("🗂️  Serving project root: %s\n", lrs.projectRoot)
	}
	fmt.Printf("🧩 Building from: %s\n", lrs.buildDir)
	fmt.Printf("⏱️  Debounce time: %v\n", debounceTime)
	fmt.Printf("🌐 Server running on http://%s\n", netAddr(lrs.host, lrs.port))
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

func (lrs *LiveReloadServer) handleHTML(w http.ResponseWriter, r *http.Request, filePath string) {
	// Read the original html file
	htmlContent, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Read the live reload client script from external file
	scriptContent, err := os.ReadFile(lrs.clientScriptPath)
	if err != nil {
		log.Printf("❌ Could not read livereload-client.js from %s: %v", lrs.clientScriptPath, err)
		http.Error(w, "Could not read livereload client script", http.StatusInternalServerError)
		return
	}

	configScript := fmt.Sprintf("\n<script>\nwindow.__GWC_LIVERELOAD_CONFIG = Object.assign({}, window.__GWC_LIVERELOAD_CONFIG || {}, { wasmPath: %q });\n</script>", lrs.servedWASMPath())

	// Inject the live reload script before closing </body> tag
	liveReloadScript := fmt.Sprintf("%s\n<script>\n%s\n</script>", configScript, string(scriptContent))

	// Insert the script before closing </body> tag
	modifiedContent := strings.Replace(string(htmlContent), "</body>", liveReloadScript+"\n</body>", 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(modifiedContent))
}

func (lrs *LiveReloadServer) servedWASMPath() string {
	if lrs == nil {
		return "/main.wasm"
	}

	outputPath := strings.TrimSpace(lrs.outputPath)
	if outputPath == "" {
		return "/main.wasm"
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(lrs.buildDir, outputPath)
	}

	relPath, err := filepath.Rel(lrs.projectRoot, outputPath)
	if err == nil && relPath != "" && relPath != "." && !strings.HasPrefix(relPath, "..") && !filepath.IsAbs(relPath) {
		return "/" + filepath.ToSlash(relPath)
	}

	base := strings.TrimSpace(filepath.Base(outputPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "/main.wasm"
	}
	return "/" + filepath.ToSlash(base)
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

	// Send current build status to the new client
	lrs.sendCurrentBuildStatus(conn)

	// Remove client when done
	defer func() {
		lrs.clientsMutex.Lock()
		delete(lrs.clients, conn)
		lrs.clientsMutex.Unlock()
		fmt.Printf("🔌 WebSocket client disconnected (remaining: %d)\n", len(lrs.clients))
	}()

	// Keep connection alive and handle messages
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var message WebSocketMessage
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}
		if message.Type == MessageTypeStateSnapshot {
			if payload, ok := message.Payload.(string); ok && strings.TrimSpace(payload) != "" {
				lrs.stateSnapshotMu.Lock()
				lrs.pendingStateSnapshot = payload
				lrs.stateSnapshotMu.Unlock()
			}
		}
	}
}

func (lrs *LiveReloadServer) requestStateSnapshot() {
	lrs.broadcastMessage(MessageTypeStateExport, map[string]string{
		"reason": "hot_reload",
	})
}

func (lrs *LiveReloadServer) takePendingStateSnapshot() string {
	lrs.stateSnapshotMu.Lock()
	defer lrs.stateSnapshotMu.Unlock()
	snapshot := lrs.pendingStateSnapshot
	lrs.pendingStateSnapshot = ""
	return snapshot
}

func (lrs *LiveReloadServer) clearPendingStateSnapshot() {
	lrs.stateSnapshotMu.Lock()
	lrs.pendingStateSnapshot = ""
	lrs.stateSnapshotMu.Unlock()
}

func (lrs *LiveReloadServer) sendCurrentBuildStatus(conn *websocket.Conn) {
	// Check if we have a previous build status to send
	if lrs.lastBuildStatus != nil {
		message := WebSocketMessage{
			Type:      MessageTypeCurrentStatus,
			Payload:   *lrs.lastBuildStatus,
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			log.Printf("❌ Failed to marshal current build status: %v", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("⚠️  Failed to send current build status to client: %v", err)
		} else {
			statusText := "success"
			if !lrs.lastBuildStatus.Success {
				statusText = "failed"
			}
			fmt.Printf("📤 Sent current build status (%s) to new client\n", statusText)
		}
	} else {
		// Try to check current build state by attempting a quick build check
		go lrs.checkCurrentBuildState(conn)
	}
}

func (lrs *LiveReloadServer) checkCurrentBuildState(conn *websocket.Conn) {
	// Do a quick build check to see if the current code compiles
	fmt.Println("🔍 Checking current build state for new client...")

	cmd := exec.Command(buildCommand, "build", "-o", lrs.outputPath)
	cmd.Dir = lrs.buildDir
	cmd.Env = append(os.Environ(), buildEnv...)

	// Capture stderr for error reporting
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	var buildStatus BuildStatus
	if err != nil {
		// Build failed - get the error
		buildError := strings.TrimSpace(stderr.String())
		if buildError == "" {
			buildError = err.Error()
		}

		buildStatus = BuildStatus{
			Success:    false,
			Error:      buildError,
			ReloadType: "none",
		}
		fmt.Printf("❌ Current build state: FAILED - %s\n", buildError)
	} else {
		buildStatus = BuildStatus{
			Success:    true,
			ReloadType: "none", // This is just a status check, not a real build
		}
		fmt.Println("✅ Current build state: OK")
	}

	// Store this as the current build status
	lrs.lastBuildStatus = &buildStatus

	// Send to the specific client
	message := WebSocketMessage{
		Type:      MessageTypeCurrentStatus,
		Payload:   buildStatus,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal build state check: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("⚠️  Failed to send build state to client: %v", err)
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

	if lrs.alwaysHotReload {
		reason := "App dev server hot reload"
		if len(changedFiles) > 0 {
			reason = "Hot reload for app changes"
		}
		return UpdateClassification{
			Type:         "small",
			ReloadType:   "hot",
			Reason:       reason,
			ChangedFiles: changedFiles,
		}
	}

	if len(changedFiles) == 0 {
		return UpdateClassification{
			Type:         "small",
			ReloadType:   "hot",
			Reason:       "No files changed",
			ChangedFiles: changedFiles,
		}
	}

	// Analyze the changed files to determine update type
	var hotReloadReasons []string
	for _, file := range changedFiles {
		relPath, _ := filepath.Rel(lrs.projectRoot, file)

		// Always full reload for critical system files
		if strings.Contains(relPath, "main.go") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "Main function or entry point changed",
				ChangedFiles: changedFiles,
			}
		}

		if strings.Contains(relPath, "fiber/fiber.go") ||
			strings.Contains(relPath, "fiber/hooks.go") ||
			strings.Contains(relPath, "fiber/types.go") ||
			strings.Contains(relPath, "fiber/state_management.go") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "Core fiber system changed",
				ChangedFiles: changedFiles,
			}
		}

		if strings.Contains(relPath, "go.mod") || strings.Contains(relPath, "go.sum") {
			return UpdateClassification{
				Type:         "big",
				ReloadType:   "full",
				Reason:       "Package dependencies changed",
				ChangedFiles: changedFiles,
			}
		}

		// Check if file is in examples/ directory (UI components)
		if strings.Contains(relPath, "examples/") {
			hotReloadReasons = append(hotReloadReasons, "example components")
		}
		// Check if file is in website/ directory (UI components)
		if strings.Contains(relPath, "website/") {
			hotReloadReasons = append(hotReloadReasons, "website components")
		}
	}

	// Remove duplicates from hotReloadReasons
	uniqueReasons := make(map[string]bool)
	var finalReasons []string
	for _, reason := range hotReloadReasons {
		if !uniqueReasons[reason] {
			uniqueReasons[reason] = true
			finalReasons = append(finalReasons, reason)
		}
	}

	// If we detected UI-related changes, use hot reload
	if len(finalReasons) > 0 {
		return UpdateClassification{
			Type:         "small",
			ReloadType:   "hot",
			Reason:       "UI changes: " + strings.Join(finalReasons, ", "),
			ChangedFiles: changedFiles,
		}
	}

	// Default to full reload for logic changes
	return UpdateClassification{
		Type:         "big",
		ReloadType:   "full",
		Reason:       "Logic changes detected, using full reload for safety",
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
	if classification.ReloadType == "hot" {
		lrs.clearPendingStateSnapshot()
		lrs.requestStateSnapshot()
	}

	// Create the build command
	cmd := exec.Command(buildCommand, "build", "-o", lrs.outputPath)

	cmd.Dir = lrs.buildDir
	cmd.Env = append(os.Environ(), buildEnv...)

	// Capture stdout and stderr for error reporting
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	lrs.currentBuild = cmd
	fmt.Printf("🏗️  Build process started (PID: will be available after start)\n")

	// Start the build process (non-blocking)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start build: %v\n", err)
		lrs.currentBuild = nil
		lrs.clearPendingStateSnapshot()
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
			// Get the actual build error output
			buildError := strings.TrimSpace(stderr.String())
			if buildError == "" {
				buildError = err.Error()
			}

			buildStatus := BuildStatus{
				Success:    false,
				Error:      buildError,
				ReloadType: "none",
			}

			fmt.Printf("❌ Build failed after %v: %v\n", duration, err)
			lrs.clearPendingStateSnapshot()
			lrs.lastBuildStatus = &buildStatus
			lrs.broadcastMessage(MessageTypeBuildComplete, buildStatus)
		}
	} else {
		manifest, manifestErr := lrs.buildChangedComponentManifest(lrs.lastClassification)
		if manifestErr != nil {
			log.Printf("⚠️  Failed to build changed-component manifest: %v", manifestErr)
		}
		if manifest != nil {
			if err := lrs.writeChangedComponentManifest(manifest); err != nil {
				log.Printf("⚠️  Failed to write changed-component manifest: %v", err)
			}
		}

		buildStatus := BuildStatus{
			Success:      true,
			Duration:     duration.String(),
			ReloadType:   lrs.lastClassification.ReloadType,
			ManifestPath: lrs.manifestPath,
			Manifest:     manifest,
		}
		if buildStatus.ReloadType == "hot" {
			buildStatus.StateSnapshot = lrs.takePendingStateSnapshot()
		} else {
			lrs.clearPendingStateSnapshot()
		}

		fmt.Printf("✅ Build completed successfully in %v\n", duration)
		lrs.lastBuildStatus = &buildStatus
		lrs.broadcastMessage(MessageTypeBuildComplete, buildStatus)
	}

	lrs.currentBuild = nil
}

func (lrs *LiveReloadServer) buildChangedComponentManifest(classification UpdateClassification) (*ChangedComponentManifest, error) {
	manifest := &ChangedComponentManifest{
		GeneratedAt:  time.Now(),
		ReloadType:   classification.ReloadType,
		Reason:       classification.Reason,
		ChangedFiles: make([]string, 0, len(classification.ChangedFiles)),
	}

	componentsByQualifiedName := make(map[string]ChangedComponent)
	for _, file := range classification.ChangedFiles {
		relFile := file
		if relative, err := filepath.Rel(lrs.watchRoot, file); err == nil {
			relFile = filepath.ToSlash(relative)
		}
		manifest.ChangedFiles = append(manifest.ChangedFiles, relFile)

		components, err := lrs.extractChangedComponents(file)
		if err != nil {
			return nil, err
		}
		for _, component := range components {
			componentsByQualifiedName[component.QualifiedName] = component
		}
	}

	if len(componentsByQualifiedName) > 0 {
		qualifiedNames := make([]string, 0, len(componentsByQualifiedName))
		for qualifiedName := range componentsByQualifiedName {
			qualifiedNames = append(qualifiedNames, qualifiedName)
		}
		sort.Strings(qualifiedNames)
		manifest.Components = make([]ChangedComponent, 0, len(qualifiedNames))
		for _, qualifiedName := range qualifiedNames {
			manifest.Components = append(manifest.Components, componentsByQualifiedName[qualifiedName])
		}
	}

	return manifest, nil
}

func (lrs *LiveReloadServer) extractChangedComponents(filePath string) ([]ChangedComponent, error) {
	if strings.TrimSpace(filePath) == "" || !strings.HasSuffix(filePath, ".go") {
		return nil, nil
	}

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	relFile := filePath
	if relative, err := filepath.Rel(lrs.watchRoot, filePath); err == nil {
		relFile = filepath.ToSlash(relative)
	}
	packagePath := resolvePackagePath(lrs.modulePath, lrs.watchRoot, filePath)
	packageName := ""
	if parsed.Name != nil {
		packageName = parsed.Name.Name
	}

	var components []ChangedComponent
	for _, decl := range parsed.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			if typed.Name == nil || typed.Recv != nil || !returnsComponentNode(typed.Type) {
				continue
			}
			components = append(components, ChangedComponent{
				Name:          typed.Name.Name,
				QualifiedName: qualifyComponentName(packagePath, typed.Name.Name),
				PackageName:   packageName,
				PackagePath:   packagePath,
				File:          relFile,
			})
		case *ast.GenDecl:
			if typed.Tok != token.VAR {
				continue
			}
			for _, spec := range typed.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range valueSpec.Names {
					if name == nil || !isComponentValueSpec(valueSpec, index) {
						continue
					}
					components = append(components, ChangedComponent{
						Name:          name.Name,
						QualifiedName: qualifyComponentName(packagePath, name.Name),
						PackageName:   packageName,
						PackagePath:   packagePath,
						File:          relFile,
					})
				}
			}
		}
	}

	return components, nil
}

func isComponentValueSpec(spec *ast.ValueSpec, index int) bool {
	if spec == nil {
		return false
	}
	if funcType, ok := spec.Type.(*ast.FuncType); ok {
		return returnsComponentNode(funcType)
	}
	if index >= len(spec.Values) {
		return false
	}
	funcLiteral, ok := spec.Values[index].(*ast.FuncLit)
	if !ok {
		return false
	}
	return returnsComponentNode(funcLiteral.Type)
}

func returnsComponentNode(funcType *ast.FuncType) bool {
	if funcType == nil || funcType.Results == nil || len(funcType.Results.List) != 1 {
		return false
	}
	return isComponentResultExpr(funcType.Results.List[0].Type)
}

func isComponentResultExpr(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name == "Node" || typed.Name == "Element"
	case *ast.SelectorExpr:
		packageIdent, ok := typed.X.(*ast.Ident)
		if !ok {
			return false
		}
		return (packageIdent.Name == "ui" && (typed.Sel.Name == "Node" || typed.Sel.Name == "Element")) ||
			(packageIdent.Name == "runtime" && typed.Sel.Name == "Element")
	case *ast.StarExpr:
		return isRuntimeElementExpr(typed.X)
	default:
		return false
	}
}

func isRuntimeElementExpr(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name == "Element"
	case *ast.SelectorExpr:
		packageIdent, ok := typed.X.(*ast.Ident)
		if !ok {
			return false
		}
		return (packageIdent.Name == "ui" || packageIdent.Name == "runtime") && typed.Sel.Name == "Element"
	default:
		return false
	}
}

func qualifyComponentName(packagePath string, name string) string {
	if strings.TrimSpace(packagePath) == "" {
		return name
	}
	return packagePath + "." + name
}

func resolvePackagePath(modulePath string, watchRoot string, filePath string) string {
	directory := filepath.Dir(filePath)
	relDirectory, err := filepath.Rel(watchRoot, directory)
	if err != nil || relDirectory == "." {
		return modulePath
	}
	relDirectory = filepath.ToSlash(relDirectory)
	if strings.TrimSpace(modulePath) == "" {
		return relDirectory
	}
	return modulePath + "/" + relDirectory
}

func resolveModulePath(root string) string {
	goModPath := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
		}
	}
	return ""
}

func (lrs *LiveReloadServer) writeChangedComponentManifest(manifest *ChangedComponentManifest) error {
	if manifest == nil || strings.TrimSpace(lrs.manifestPath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(lrs.manifestPath), 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(lrs.manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
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

func resolveBuildDir(entryPath string) (string, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(entryPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve entry path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to inspect entry path %s: %w", absPath, err)
	}
	if info.IsDir() {
		return absPath, nil
	}
	return filepath.Dir(absPath), nil
}

func resolveModuleRoot(startPath string) string {
	current := strings.TrimSpace(startPath)
	if current == "" {
		return ""
	}

	absPath, err := filepath.Abs(current)
	if err != nil {
		return ""
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	for {
		if _, err := os.Stat(filepath.Join(absPath, "go.mod")); err == nil {
			return absPath
		}
		parent := filepath.Dir(absPath)
		if parent == absPath {
			return ""
		}
		absPath = parent
	}
}

func resolveStaticDir(projectRoot string) string {
	candidates := []string{
		filepath.Join(projectRoot, "static"),
		filepath.Join(filepath.Dir(projectRoot), "static"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

func resolveClientScriptPath() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		candidate := filepath.Join(filepath.Dir(exe), "scripts", "livereload-client.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "scripts", "livereload-client.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func netAddr(host, port string) string {
	if strings.TrimSpace(host) == "" {
		host = defaultHost
	}
	if strings.TrimSpace(port) == "" {
		port = defaultPort
	}
	return net.JoinHostPort(host, port)
}

func main() {
	appPath := flag.String("app", "", "Path to the app main.go file or the app directory")
	mainPath := flag.String("main", "", "Legacy alias for -app")
	rootPath := flag.String("root", "", "Project root to watch and serve")
	htmlPath := flag.String("html", "", "HTML file to serve, relative to the project root")
	indexPath := flag.String("index", "", "Legacy alias for -html")
	wasmPath := flag.String("wasm", "", "WASM output path, relative to the build directory")
	outputPath := flag.String("output", "", "Legacy alias for -wasm")
	host := flag.String("host", defaultHost, "Host to bind")
	port := flag.String("port", defaultPort, "Port to bind")
	hot := flag.Bool("hot", true, "Always use hot reload on successful rebuilds")
	clientScriptPath := flag.String("client-script", "", "Path to livereload-client.js")
	flag.Parse()

	selectedAppPath := strings.TrimSpace(*appPath)
	if selectedAppPath == "" {
		selectedAppPath = strings.TrimSpace(*mainPath)
	}
	selectedHTMLPath := strings.TrimSpace(*htmlPath)
	if selectedHTMLPath == "" {
		selectedHTMLPath = strings.TrimSpace(*indexPath)
	}
	selectedWASMPath := strings.TrimSpace(*wasmPath)
	if selectedWASMPath == "" {
		selectedWASMPath = strings.TrimSpace(*outputPath)
	}

	server, err := NewLiveReloadServerWithOptions(LiveReloadOptions{
		MainPath:         selectedAppPath,
		ProjectRoot:      *rootPath,
		IndexPath:        selectedHTMLPath,
		OutputPath:       selectedWASMPath,
		Host:             *host,
		Port:             *port,
		AlwaysHotReload:  *hot,
		ClientScriptPath: *clientScriptPath,
	})
	if err != nil {
		log.Fatalf("❌ Failed to create live reload server: %v", err)
	}

	defer server.cleanup()

	err = server.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start live reload server: %v", err)
	}
}
