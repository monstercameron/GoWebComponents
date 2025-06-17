package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	MessageTypeCurrentStatus  MessageType = "current_status"
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
	lastBuildStatus    *BuildStatus         // Track the last build status for new clients
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

	// Read the live reload client script from external file
	scriptPath := filepath.Join("scripts", "livereload-client.js")
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		log.Printf("❌ Could not read livereload-client.js from %s: %v", scriptPath, err)
		http.Error(w, "Could not read livereload client script", http.StatusInternalServerError)
		return
	}

	// Inject the live reload script before closing </body> tag
	liveReloadScript := fmt.Sprintf("\n<script>\n%s\n</script>", string(scriptContent))

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
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
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

	cmd := exec.Command(buildCommand, buildArgs...)
	cmd.Dir = lrs.projectRoot
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
	}

	// Check if changes are UI-related (hot reload candidates)
	hotReloadReasons := []string{}
	for _, file := range changedFiles {
		// Read file content to analyze the types of changes
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}

		contentStr := string(content)
		relPath, _ := filepath.Rel(lrs.projectRoot, file)

		// Check for element creation functions and aliases
		elementFunctions := []string{
			"H1(", "H2(", "H3(", "H4(", "H5(", "H6(",
			"Div(", "Span(", "P(", "A(", "Button(", "Input(", "Form(",
			"Table(", "Tr(", "Td(", "Th(", "Thead(", "Tbody(",
			"Ul(", "Ol(", "Li(", "Nav(", "Header(", "Footer(", "Section(",
			"Article(", "Aside(", "Main(", "Figure(", "Figcaption(",
			"Img(", "Video(", "Audio(", "Canvas(", "Svg(",
			"Select(", "Option(", "Textarea(", "Label(", "Fieldset(",
			"Legend(", "Details(", "Summary(", "Dialog(",
		}

		hasElementChanges := false
		for _, elementFunc := range elementFunctions {
			if strings.Contains(contentStr, elementFunc) {
				hasElementChanges = true
				break
			}
		}

		if hasElementChanges {
			hotReloadReasons = append(hotReloadReasons, "element creation")
		}

		// Check for HTML-like code patterns
		htmlPatterns := []string{
			"Attrs{", "\"style\":", "\"class\":", "\"id\":", "\"onclick\":",
			"\"onchange\":", "\"oninput\":", "\"onsubmit\":", "\"href\":",
			"\"src\":", "\"alt\":", "\"title\":", "\"placeholder\":",
			"\"value\":", "\"type\":", "\"disabled\":", "\"readonly\":",
		}

		hasHtmlChanges := false
		for _, pattern := range htmlPatterns {
			if strings.Contains(contentStr, pattern) {
				hasHtmlChanges = true
				break
			}
		}

		if hasHtmlChanges {
			hotReloadReasons = append(hotReloadReasons, "HTML-like attributes")
		}

		// Check for string literal changes
		if strings.Contains(contentStr, "Text(\"") ||
			strings.Contains(contentStr, "\", \"") ||
			(strings.Count(contentStr, "\"") > 10) { // Lots of strings
			hotReloadReasons = append(hotReloadReasons, "string content")
		}

		// Check if file is in examples/ directory (UI components)
		if strings.Contains(relPath, "examples/") {
			hotReloadReasons = append(hotReloadReasons, "example components")
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

	// Create the build command
	cmd := exec.Command(buildCommand, buildArgs...)

	cmd.Dir = lrs.projectRoot
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
			lrs.lastBuildStatus = &buildStatus
			lrs.broadcastMessage(MessageTypeBuildComplete, buildStatus)
		}
	} else {
		buildStatus := BuildStatus{
			Success:    true,
			Duration:   duration.String(),
			ReloadType: lrs.lastClassification.ReloadType,
		}

		fmt.Printf("✅ Build completed successfully in %v\n", duration)
		lrs.lastBuildStatus = &buildStatus
		lrs.broadcastMessage(MessageTypeBuildComplete, buildStatus)
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
