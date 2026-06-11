package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/diagnostics"
	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
)

const (
	debounceTime      = 2000 * time.Millisecond // Wait 2 seconds after last keystroke
	quickDebounceTime = 500 * time.Millisecond  // Quick debounce for single file changes
	maxDebounceTime   = 5000 * time.Millisecond // Maximum wait time before forcing build
	buildCommand      = "go"
	defaultHost       = "127.0.0.1"
	defaultPort       = "8080"
	livereloadDocs    = "ACTIONABLE_ERRORS.md#gwc-tool-livereload"
)

var (
	//go:embed scripts/livereload-client.txt
	embeddedLivereloadClientScript string

	buildEnv = []string{"GOOS=js", "GOARCH=wasm"}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin in development
		},
	}
	livereloadConfigGetwd       = os.Getwd
	livereloadConfigUserHomeDir = os.UserHomeDir
	livereloadExecutablePath    = os.Executable
)

func livereloadClientScriptBytes(parsePath string) ([]byte, error) {
	parseOverridePath := strings.TrimSpace(parsePath)
	if parseOverridePath != "" {
		parseContent, parseErr := os.ReadFile(parseOverridePath)
		if parseErr != nil {
			return nil, parseErr
		}
		return parseContent, nil
	}

	if strings.TrimSpace(embeddedLivereloadClientScript) == "" {
		return nil, fmt.Errorf("embedded livereload client script is empty")
	}

	return []byte(embeddedLivereloadClientScript), nil
}

func terminateLivereloadProcessTree(parseCmd *exec.Cmd) {
	if parseCmd == nil || parseCmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parseCmd.Process.Pid)).Run()
		return
	}
	_ = parseCmd.Process.Kill()
}

func livereloadRunnerConfigFS() runnerconfig.FS {
	return runnerconfig.FS{
		Getwd:       livereloadConfigGetwd,
		UserHomeDir: livereloadConfigUserHomeDir,
		ReadFile:    os.ReadFile,
		Stat:        os.Stat,
	}
}

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
	// MessageTypeAssetSwap tells clients to hot-swap changed static assets
	// (stylesheets) in place without a wasm rebuild or page reload.
	MessageTypeAssetSwap MessageType = "asset_swap"
	// MessageTypeWatcherDegraded warns clients that file watching is degraded
	// and some source changes may no longer trigger rebuilds.
	MessageTypeWatcherDegraded MessageType = "watcher_degraded"
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
	Phase         string                    `json:"phase,omitempty"`
	PhaseSummary  string                    `json:"phaseSummary,omitempty"`
	StaleOutput   bool                      `json:"staleOutput,omitempty"`
	StateSnapshot string                    `json:"stateSnapshot,omitempty"`
	ManifestPath  string                    `json:"manifestPath,omitempty"`
	Manifest      *ChangedComponentManifest `json:"manifest,omitempty"`
	// Timings carries per-stage pipeline durations in milliseconds (classifyMs,
	// compileMs, totalMs) plus artifactBytes, so clients can show a precise
	// save-to-paint breakdown and teams can budget dev-loop latency.
	Timings map[string]int64 `json:"timings,omitempty"`
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

type LiveReloadStatus struct {
	Mode               string               `json:"mode"`
	ListeningURL       string               `json:"listeningURL"`
	StatusURL          string               `json:"statusURL"`
	WebSocketURL       string               `json:"websocketURL"`
	ProjectRoot        string               `json:"projectRoot"`
	WatchRoot          string               `json:"watchRoot"`
	BuildDir           string               `json:"buildDir"`
	ServedWASMPath     string               `json:"servedWasmPath"`
	HotReloadEnabled   bool                 `json:"hotReloadEnabled"`
	HotReloadEligible  bool                 `json:"hotReloadEligible"`
	LastClassification UpdateClassification `json:"lastClassification,omitempty"`
	LastBuild          *BuildStatus         `json:"lastBuild,omitempty"`
	CurrentError       *BuildStatus         `json:"currentError,omitempty"`
	ClientCount        int                  `json:"clientCount"`
	Clients            []ClientSession      `json:"clients,omitempty"`
}

type ClientSession struct {
	ID          string    `json:"id"`
	RemoteAddr  string    `json:"remoteAddr,omitempty"`
	UserAgent   string    `json:"userAgent,omitempty"`
	ConnectedAt time.Time `json:"connectedAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

type clientDisconnectRequest struct {
	ClientID string `json:"clientID,omitempty"`
	All      bool   `json:"all,omitempty"`
}

type clientDisconnectResponse struct {
	DisconnectedIDs []string `json:"disconnectedIDs,omitempty"`
	Disconnected    int      `json:"disconnected"`
	Remaining       int      `json:"remaining"`
}

type UpdateCompatibilityPlan struct {
	Summary        string `json:"summary,omitempty"`
	PreserveState  bool   `json:"preserveState,omitempty"`
	RemountSubtree bool   `json:"remountSubtree,omitempty"`
	RestartAsync   bool   `json:"restartAsync,omitempty"`
	FullReload     bool   `json:"fullReload,omitempty"`
}

// UpdateClassification represents the type of update detected
type UpdateClassification struct {
	Type         string                  `json:"type"`       // "small" or "big"
	ReloadType   string                  `json:"reloadType"` // "hot" or "full"
	Reason       string                  `json:"reason"`     // explanation for the classification
	ChangedFiles []string                `json:"changedFiles"`
	Plan         UpdateCompatibilityPlan `json:"plan,omitempty"`
}

type LiveReloadServer struct {
	watcher              *fsnotify.Watcher
	mutex                sync.Mutex
	currentBuild         *exec.Cmd
	buildQueued          bool
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
	clients              map[*websocket.Conn]ClientSession
	clientsMutex         sync.RWMutex
	nextClientID         uint64
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
	changedAssets        map[string]time.Time // Changed static assets (css) pending an asset_swap broadcast
	assetDebounceTimer   *time.Timer
	wasmGzipMu           sync.Mutex // guards the gzip artifact cache below
	wasmGzipCache        []byte     // gzip-compressed wasm artifact, keyed by mtime
	wasmGzipModTime      time.Time
}

func livereloadReport(parseSubject string, parsePath string, parseSummary string, parseConsequence string, parseNext string) diagnostics.Report {
	return diagnostics.NewReport(diagnostics.Options{
		Summary:  strings.TrimSpace(parseSummary),
		Code:     "GWC-TOOL-LIVERELOAD",
		Headline: "tool failure in " + strings.TrimSpace(parseSubject),
		Path:     strings.TrimSpace(parsePath),
		Runtime:  strings.TrimSpace(parseConsequence),
		Next:     strings.TrimSpace(parseNext),
		Docs:     livereloadDocs,
	})
}

func livereloadErrReport(parseSubject string, parsePath string, parseErr error, parseConsequence string, parseNext string) diagnostics.Report {
	return livereloadReport(parseSubject, parsePath, parseErr.Error(), parseConsequence, parseNext)
}

func emitLivereloadError(parseSubject string, parsePath string, parseErr error, parseConsequence string, parseNext string) {
	diagnostics.Emit(livereloadErrReport(parseSubject, parsePath, parseErr, parseConsequence, parseNext))
}

func newUpdateClassification(parseUpdateType string, parseReloadType string, parseReason string, parseChangedFiles []string) UpdateClassification {
	parseClassification := UpdateClassification{
		Type:         parseUpdateType,
		ReloadType:   parseReloadType,
		Reason:       parseReason,
		ChangedFiles: parseChangedFiles,
	}
	parseClassification.Plan = describeUpdateCompatibilityPlan(parseClassification)
	return parseClassification
}

func describeUpdateCompatibilityPlan(parseClassification UpdateClassification) UpdateCompatibilityPlan {
	switch parseClassification.ReloadType {
	case "hot":
		parsePlan := UpdateCompatibilityPlan{
			Summary:        "planned preserve-state hot reload; compatible local state is preserved, changed subtrees may remount, async and router work restart",
			PreserveState:  true,
			RemountSubtree: true,
			RestartAsync:   true,
		}
		if len(parseClassification.ChangedFiles) == 0 {
			parsePlan.Summary = "planned preserve-state hot reload; compatible local state is preserved"
			parsePlan.RemountSubtree = false
			parsePlan.RestartAsync = false
		}
		return parsePlan
	case "full":
		return UpdateCompatibilityPlan{
			Summary:    "planned full reload; preserved local state will be discarded",
			FullReload: true,
		}
	default:
		return UpdateCompatibilityPlan{}
	}
}

func newBuildStatus(parsePhase string, parsePhaseSummary string) BuildStatus {
	return BuildStatus{
		Phase:        strings.TrimSpace(parsePhase),
		PhaseSummary: strings.TrimSpace(parsePhaseSummary),
	}
}

func fatalLivereloadStartup(parseSubject string, parsePath string, parseErr error, parseNext string) {
	emitLivereloadError(parseSubject, parsePath, parseErr, "the live reload tool did not finish startup, so no build or websocket loop is running.", parseNext)
	os.Exit(1)
}

// NewLiveReloadServer creates a LiveReloadServer rooted at projectRoot with default options.
func NewLiveReloadServer(parseProjectRoot string) (*LiveReloadServer, error) {
	return NewLiveReloadServerWithOptions(LiveReloadOptions{ProjectRoot: parseProjectRoot})
}

// NewLiveReloadServerWithOptions creates a LiveReloadServer from the given options.
func NewLiveReloadServerWithOptions(parseOptions LiveReloadOptions) (*LiveReloadServer, error) {
	parseWatcher, parseErr := fsnotify.NewWatcher()
	if parseErr != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", parseErr)
	}

	buildDir := strings.TrimSpace(parseOptions.MainPath)
	if buildDir != "" {
		buildDir, parseErr = resolveBuildDir(buildDir)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	parseProjectRoot := strings.TrimSpace(parseOptions.ProjectRoot)
	if parseProjectRoot == "" {
		if buildDir != "" {
			parseProjectRoot = buildDir
		} else {
			parseProjectRoot = "."
		}
	}
	parseProjectRoot, parseErr = filepath.Abs(parseProjectRoot)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", parseErr)
	}
	if buildDir == "" {
		buildDir = parseProjectRoot
	}

	parseWatchRoot := parseProjectRoot
	if parseModuleRoot := resolveModuleRoot(buildDir); parseModuleRoot != "" {
		parseWatchRoot = parseModuleRoot
	}

	parseIndexPath := strings.TrimSpace(parseOptions.IndexPath)
	if parseIndexPath == "" {
		parseIndexPath = filepath.Join(parseProjectRoot, "index.html")
		if _, parseErr2 := os.Stat(parseIndexPath); parseErr2 != nil {
			parseIndexPath = filepath.Join(parseProjectRoot, "static", "index.html")
		}
	} else if !filepath.IsAbs(parseIndexPath) {
		parseIndexPath = filepath.Join(parseProjectRoot, parseIndexPath)
	}

	parseOutputPath := strings.TrimSpace(parseOptions.OutputPath)
	if parseOutputPath == "" {
		parseWorkspaceBuildRoot, parseResolveErr := runnerconfig.ResolveWorkspaceBuildRoot(parseWatchRoot, livereloadRunnerConfigFS())
		if parseResolveErr != nil {
			return nil, fmt.Errorf("resolve workspace build root: %w", parseResolveErr)
		}
		parseRelProjectPath, parseRelErr := filepath.Rel(parseWatchRoot, parseProjectRoot)
		if parseRelErr != nil || strings.HasPrefix(parseRelProjectPath, "..") {
			parseRelProjectPath = filepath.Base(parseProjectRoot)
		}
		parseOutputPath = filepath.Join(parseWorkspaceBuildRoot, parseRelProjectPath, "main.wasm")
	} else if !filepath.IsAbs(parseOutputPath) {
		parseOutputPath = filepath.Join(buildDir, parseOutputPath)
	}

	parseStaticDir := resolveStaticDir(parseProjectRoot)

	parseClientScriptPath := strings.TrimSpace(parseOptions.ClientScriptPath)
	if parseClientScriptPath == "" {
		parseClientScriptPath = resolveClientScriptPath()
	} else if !filepath.IsAbs(parseClientScriptPath) {
		parseClientScriptPath = filepath.Join(parseProjectRoot, parseClientScriptPath)
	}

	parseHost := strings.TrimSpace(parseOptions.Host)
	if parseHost == "" {
		parseHost = defaultHost
	}
	parsePort := strings.TrimSpace(parseOptions.Port)
	if parsePort == "" {
		parsePort = defaultPort
	}

	parseManifestPath := parseOutputPath + ".hotreload-manifest.json"

	return &LiveReloadServer{
		watcher:          parseWatcher,
		projectRoot:      parseProjectRoot,
		watchRoot:        parseWatchRoot,
		buildDir:         buildDir,
		indexPath:        parseIndexPath,
		outputPath:       parseOutputPath,
		staticDir:        parseStaticDir,
		clientScriptPath: parseClientScriptPath,
		host:             parseHost,
		port:             parsePort,
		alwaysHotReload:  parseOptions.AlwaysHotReload,
		clients:          make(map[*websocket.Conn]ClientSession),
		changedFiles:     make(map[string]time.Time),
		modulePath:       resolveModulePath(parseWatchRoot),
		manifestPath:     parseManifestPath,
	}, nil
}

func (parseLrs *LiveReloadServer) Start() error {
	// Watch the module root so shared package changes rebuild example-specific servers.
	parseErr := parseLrs.addWatchers(parseLrs.watchRoot)
	if parseErr != nil {
		return fmt.Errorf("failed to add watchers: %w", parseErr)
	}

	parseLrs.httpServer = &http.Server{
		Addr:    netAddr(parseLrs.host, parseLrs.port),
		Handler: parseLrs.newHTTPHandler(),
	}

	// Start HTTP server in a goroutine
	go func() {
		fmt.Printf("Ã°Å¸Å’Â Live reload server starting on http://%s\n", netAddr(parseLrs.host, parseLrs.port))
		if parseErr2 := parseLrs.httpServer.ListenAndServe(); parseErr2 != nil && parseErr2 != http.ErrServerClosed {
			emitLivereloadError("LiveReloadServer.Start.ListenAndServe", netAddr(parseLrs.host, parseLrs.port), parseErr2, "the HTTP listener stopped unexpectedly and browser clients can no longer connect.", "Inspect the bind address and listener lifetime for the livereload server.")
		}
	}()

	// Handle interrupt signal for graceful shutdown
	parseC := make(chan os.Signal, 1)
	signal.Notify(parseC, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Ã°Å¸â€â€ž Live reload started. Watching for .go file changes...")
	fmt.Printf("Ã°Å¸â€œâ€š Watching directory: %s\n", parseLrs.watchRoot)
	if parseLrs.watchRoot != parseLrs.projectRoot {
		fmt.Printf("Ã°Å¸â€”â€šÃ¯Â¸Â  Serving project root: %s\n", parseLrs.projectRoot)
	}
	fmt.Printf("Ã°Å¸Â§Â© Building from: %s\n", parseLrs.buildDir)
	fmt.Printf("Ã¢ÂÂ±Ã¯Â¸Â  Debounce time: %v\n", debounceTime)
	fmt.Printf("Ã°Å¸Å’Â Server running on http://%s\n", netAddr(parseLrs.host, parseLrs.port))
	fmt.Println("Ã°Å¸â€ºâ€˜ Press Ctrl+C to stop")

	// Trigger initial build
	parseLrs.triggerBuild()

	go func() {
		for {
			select {
			case parseEvent, parseOk := <-parseLrs.watcher.Events:
				if !parseOk {
					return
				}
				parseLrs.handleFileEvent(parseEvent)

			case parseErr3, parseOk2 := <-parseLrs.watcher.Errors:
				if !parseOk2 {
					return
				}
				emitLivereloadError("LiveReloadServer.Start.watcher", parseLrs.watchRoot, parseErr3, "file watching degraded and future source changes may not trigger rebuilds.", "Inspect filesystem watcher limits and the watched root for this livereload session.")
				// Surface degradation to connected browsers, not just stdout:
				// a dev who saves and sees nothing must not have to guess why.
				parseLrs.broadcastMessage(MessageTypeWatcherDegraded, map[string]string{
					"error":     parseErr3.Error(),
					"watchRoot": parseLrs.watchRoot,
				})

			case <-parseC:
				fmt.Println("\nÃ°Å¸â€ºâ€˜ Shutting down live reload server...")
				parseLrs.cleanup()
				os.Exit(0)
			}
		}
	}()

	// Keep the main goroutine alive
	select {}
}

func (parseLrs *LiveReloadServer) handleWebSocket(parseW http.ResponseWriter, parseR *http.Request) {
	parseConn, parseErr := upgrader.Upgrade(parseW, parseR, nil)
	if parseErr != nil {
		emitLivereloadError("LiveReloadServer.handleWebSocket.upgrade", parseR.URL.Path, parseErr, "the browser could not establish the livereload websocket, so it will miss build notifications.", "Inspect the websocket endpoint, browser connection state, and any local proxy interference.")
		return
	}
	defer parseConn.Close()

	// Add client to the list
	parseLrs.clientsMutex.Lock()
	parseSession := parseLrs.nextClientSession()
	parseSession.RemoteAddr = strings.TrimSpace(parseR.RemoteAddr)
	parseSession.UserAgent = strings.TrimSpace(parseR.UserAgent())
	parseLrs.clients[parseConn] = parseSession
	parseLrs.clientsMutex.Unlock()

	fmt.Printf("Ã°Å¸â€Å’ WebSocket client connected (total: %d)\n", len(parseLrs.clients))

	// Send current build status to the new client
	parseLrs.sendCurrentBuildStatus(parseConn)

	// Remove client when done
	defer func() {
		parseLrs.clientsMutex.Lock()
		delete(parseLrs.clients, parseConn)
		parseLrs.clientsMutex.Unlock()
		fmt.Printf("Ã°Å¸â€Å’ WebSocket client disconnected (remaining: %d)\n", len(parseLrs.clients))
	}()

	// Keep connection alive and handle messages
	for {
		_, parseData, parseErr2 := parseConn.ReadMessage()
		if parseErr2 != nil {
			break
		}
		parseLrs.markClientSeen(parseConn)

		var parseMessage WebSocketMessage
		if parseErr3 := json.Unmarshal(parseData, &parseMessage); parseErr3 != nil {
			continue
		}
		if parseMessage.Type == MessageTypeStateSnapshot {
			if parsePayload, parseOk := parseMessage.Payload.(string); parseOk && strings.TrimSpace(parsePayload) != "" {
				parseLrs.storePendingStateSnapshot(parsePayload)
			}
		}
	}
}

func (parseLrs *LiveReloadServer) handleWebSocketManaged(parseW http.ResponseWriter, parseR *http.Request) {
	parseConn, parseErr := upgrader.Upgrade(parseW, parseR, nil)
	if parseErr != nil {
		emitLivereloadError("LiveReloadServer.handleWebSocketManaged.upgrade", parseR.URL.Path, parseErr, "the browser could not establish the livereload websocket, so it will miss build notifications.", "Inspect the websocket endpoint, browser connection state, and any local proxy interference.")
		return
	}
	defer parseConn.Close()

	parseSession := parseLrs.nextClientSession()
	parseSession.RemoteAddr = strings.TrimSpace(parseR.RemoteAddr)
	parseSession.UserAgent = strings.TrimSpace(parseR.UserAgent())
	parseLrs.clientsMutex.Lock()
	parseLrs.clients[parseConn] = parseSession
	parseClientCount := len(parseLrs.clients)
	parseLrs.clientsMutex.Unlock()

	fmt.Printf("websocket client connected (%s, total: %d)\n", parseSession.ID, parseClientCount)
	parseLrs.sendCurrentBuildStatus(parseConn)

	defer func() {
		parseLrs.clientsMutex.Lock()
		delete(parseLrs.clients, parseConn)
		parseRemaining := len(parseLrs.clients)
		parseLrs.clientsMutex.Unlock()
		fmt.Printf("websocket client disconnected (%s, remaining: %d)\n", parseSession.ID, parseRemaining)
	}()

	for {
		_, parseData, parseErr2 := parseConn.ReadMessage()
		if parseErr2 != nil {
			break
		}
		parseLrs.markClientSeen(parseConn)

		var parseMessage WebSocketMessage
		if parseErr3 := json.Unmarshal(parseData, &parseMessage); parseErr3 != nil {
			continue
		}
		if parseMessage.Type == MessageTypeStateSnapshot {
			if parsePayload, parseOk := parseMessage.Payload.(string); parseOk && strings.TrimSpace(parsePayload) != "" {
				parseLrs.storePendingStateSnapshot(parsePayload)
			}
		}
	}
}

func (parseLrs *LiveReloadServer) sendCurrentBuildStatus(parseConn *websocket.Conn) {
	// Check if we have a previous build status to send
	if parseLrs.lastBuildStatus != nil {
		parseMessage := WebSocketMessage{
			Type:      MessageTypeCurrentStatus,
			Payload:   *parseLrs.lastBuildStatus,
			Timestamp: time.Now(),
		}

		parseData, parseErr := json.Marshal(parseMessage)
		if parseErr != nil {
			emitLivereloadError("LiveReloadServer.sendCurrentBuildStatus.marshal", "current_status", parseErr, "the current build status could not be serialized, so the new websocket client received no initial status.", "Inspect the build status payload for unsupported values before marshalling.")
			return
		}

		if parseErr2 := parseConn.WriteMessage(websocket.TextMessage, parseData); parseErr2 != nil {
			emitLivereloadError("LiveReloadServer.sendCurrentBuildStatus.write", "current_status", parseErr2, "the new websocket client did not receive the current build status and may show stale state.", "Inspect websocket connectivity and client lifecycle during status delivery.")
		} else {
			parseStatusText := "success"
			if !parseLrs.lastBuildStatus.Success {
				parseStatusText = "failed"
			}
			fmt.Printf("Ã°Å¸â€œÂ¤ Sent current build status (%s) to new client\n", parseStatusText)
		}
	} else {
		// Try to check current build state by attempting a quick build check
		go parseLrs.checkCurrentBuildState(parseConn)
	}
}

func (parseLrs *LiveReloadServer) checkCurrentBuildState(parseConn *websocket.Conn) {
	// Do a quick build check to see if the current code compiles
	fmt.Println("Ã°Å¸â€Â Checking current build state for new client...")
	if parseErr := os.MkdirAll(filepath.Dir(parseLrs.outputPath), 0o755); parseErr != nil {
		emitLivereloadError("LiveReloadServer.checkCurrentBuildState.mkdir", filepath.Dir(parseLrs.outputPath), parseErr, "the livereload output directory could not be created before the status check build.", "Inspect the configured build root and directory permissions for the livereload artifact path.")
		return
	}

	parseLrs.lastBuildStatus = &BuildStatus{
		Success:      false,
		ReloadType:   "none",
		Phase:        "checking_current_state",
		PhaseSummary: "checking current build state for a newly connected client",
		StaleOutput:  false,
	}

	parseCmd := exec.Command(buildCommand, "build", "-o", parseLrs.outputPath)
	parseCmd.Dir = parseLrs.buildDir
	parseCmd.Env = append(os.Environ(), buildEnv...)

	// Capture stderr for error reporting
	var parseStderr bytes.Buffer
	parseCmd.Stderr = &parseStderr

	parseErr2 := parseCmd.Run()

	var buildStatus BuildStatus
	if parseErr2 != nil {
		// Build failed - get the error
		buildError := strings.TrimSpace(parseStderr.String())
		if buildError == "" {
			buildError = parseErr2.Error()
		}

		buildStatus = BuildStatus{
			Success:      false,
			Error:        buildError,
			ReloadType:   "none",
			Phase:        "blocked_on_error",
			PhaseSummary: "blocked on a build error; the last good output is all the dev server can still serve",
			StaleOutput:  true,
		}
		diagnostics.Emit(livereloadReport(
			"LiveReloadServer.checkCurrentBuildState",
			parseLrs.buildDir,
			buildError,
			"the current app does not compile, so newly connected clients are informed that the dev server is in a failed build state.",
			"Inspect the current build stderr and fix the compile error before relying on hot reload state.",
		))
	} else {
		buildStatus = BuildStatus{
			Success:      true,
			ReloadType:   "none", // This is just a status check, not a real build
			Phase:        "serving_output",
			PhaseSummary: "serving the latest successful output",
			StaleOutput:  false,
		}
		fmt.Println("Ã¢Å“â€¦ Current build state: OK")
	}

	// Store this as the current build status
	parseLrs.lastBuildStatus = &buildStatus

	// Send to the specific client
	parseMessage := WebSocketMessage{
		Type:      MessageTypeCurrentStatus,
		Payload:   buildStatus,
		Timestamp: time.Now(),
	}

	parseData, parseErr2 := json.Marshal(parseMessage)
	if parseErr2 != nil {
		emitLivereloadError("LiveReloadServer.checkCurrentBuildState.marshal", "current_status", parseErr2, "the build-state check result could not be serialized, so the client received no current-status payload.", "Inspect the build-state payload for unsupported values before marshalling.")
		return
	}

	if parseErr3 := parseConn.WriteMessage(websocket.TextMessage, parseData); parseErr3 != nil {
		emitLivereloadError("LiveReloadServer.checkCurrentBuildState.write", "current_status", parseErr3, "the websocket client did not receive the build-state check result and may show stale information.", "Inspect websocket connectivity and client lifecycle during status delivery.")
	}
}

func (parseLrs *LiveReloadServer) broadcastMessage(parseMsgType MessageType, parsePayload interface{}) {
	parseMessage := WebSocketMessage{
		Type:      parseMsgType,
		Payload:   parsePayload,
		Timestamp: time.Now(),
	}

	parseData, parseErr := json.Marshal(parseMessage)
	if parseErr != nil {
		emitLivereloadError("LiveReloadServer.broadcastMessage.marshal", string(parseMsgType), parseErr, "the livereload event was not serialized, so connected clients will miss this update.", "Inspect the websocket payload for unsupported values before marshalling.")
		return
	}

	parseLrs.clientsMutex.RLock()
	defer parseLrs.clientsMutex.RUnlock()

	for parseConn := range parseLrs.clients {
		if parseErr2 := parseConn.WriteMessage(websocket.TextMessage, parseData); parseErr2 != nil {
			emitLivereloadError("LiveReloadServer.broadcastMessage.write", string(parseMsgType), parseErr2, "one websocket client did not receive the livereload event and may drift out of sync.", "Inspect websocket connectivity and client lifecycle for the failing connection.")
		}
	}
}

func (parseLrs *LiveReloadServer) addWatchers(parseRoot string) error {
	return filepath.Walk(parseRoot, func(parsePath string, parseInfo os.FileInfo, parseErr2 error) error {
		if parseErr2 != nil {
			return parseErr2
		}

		// Skip hidden directories, vendor, and node_modules
		if parseInfo.IsDir() {
			parseName := parseInfo.Name()
			if strings.HasPrefix(parseName, ".") || parseName == "vendor" || parseName == "node_modules" || parseName == "bin" {
				return filepath.SkipDir
			}

			// Add the directory to the watcher
			parseErr := parseLrs.watcher.Add(parsePath)
			if parseErr != nil {
				emitLivereloadError("LiveReloadServer.addWatchers", parsePath, parseErr, "changes under this directory will not trigger rebuilds because the watcher could not attach.", "Inspect filesystem watcher limits, permissions, and directory availability for this path.")
			} else {
				fmt.Printf("Ã°Å¸â€˜â‚¬ Watching: %s\n", parsePath)
			}
		}

		return nil
	})
}

func (parseLrs *LiveReloadServer) handleFileEvent(parseEvent fsnotify.Event) {
	// Skip temporary files and editor backup files
	if strings.Contains(parseEvent.Name, ".tmp") || strings.Contains(parseEvent.Name, "~") {
		return
	}

	// Only handle write and create events
	if parseEvent.Op&fsnotify.Write != fsnotify.Write && parseEvent.Op&fsnotify.Create != fsnotify.Create {
		return
	}

	// Newly created directories must be attached to the watcher, or changes
	// under them will silently never trigger rebuilds.
	if parseEvent.Op&fsnotify.Create == fsnotify.Create {
		if parseInfo, parseStatErr := os.Stat(parseEvent.Name); parseStatErr == nil && parseInfo.IsDir() {
			parseName := parseInfo.Name()
			if !strings.HasPrefix(parseName, ".") && parseName != "vendor" && parseName != "node_modules" && parseName != "bin" {
				if parseAddErr := parseLrs.addWatchers(parseEvent.Name); parseAddErr != nil {
					emitLivereloadError("LiveReloadServer.handleFileEvent.addWatchers", parseEvent.Name, parseAddErr, "changes under this new directory will not trigger rebuilds.", "Inspect filesystem watcher limits and permissions for this path.")
				}
			}
			return
		}
	}

	switch {
	case strings.HasSuffix(parseEvent.Name, ".go"):
		fmt.Printf("Ã°Å¸â€œÂ File changed: %s\n", parseEvent.Name)
		parseLrs.mutex.Lock()
		parseLrs.changedFiles[parseEvent.Name] = time.Now()
		parseLrs.mutex.Unlock()
		parseLrs.debounceAndBuild()
	case strings.HasSuffix(parseEvent.Name, ".css"):
		// Stylesheets hot-swap in place: no wasm rebuild, no page reload.
		if parseLrs.isBuildArtifactPath(parseEvent.Name) {
			return
		}
		fmt.Printf("Ã°Å¸Å½Â¨ Stylesheet changed: %s\n", parseEvent.Name)
		parseLrs.queueAssetSwap(parseEvent.Name)
	case strings.HasSuffix(parseEvent.Name, ".html") || strings.HasSuffix(parseEvent.Name, ".js"):
		// Markup/static script changes need a page reload but never a rebuild.
		if parseLrs.isBuildArtifactPath(parseEvent.Name) {
			return
		}
		fmt.Printf("Ã°Å¸â€œâ€ž Static file changed, reloading page: %s\n", parseEvent.Name)
		parseLrs.broadcastMessage(MessageTypeReload, map[string]string{
			"reason": "static file changed: " + filepath.Base(parseEvent.Name),
		})
	}
}

// isBuildArtifactPath reports whether a path points at generated build output
// (which changes on every rebuild and must not feed back into the watcher).
func (parseLrs *LiveReloadServer) isBuildArtifactPath(parsePath string) bool {
	parseNormalized := filepath.ToSlash(parsePath)
	if strings.Contains(parseNormalized, "/bin/") || strings.Contains(parseNormalized, "/dist/") {
		return true
	}
	if parseLrs.outputPath != "" {
		parseOutputDir := filepath.ToSlash(filepath.Dir(parseLrs.outputPath))
		if parseOutputDir != "." && strings.HasPrefix(parseNormalized, parseOutputDir+"/") {
			return true
		}
	}
	return false
}

// queueAssetSwap batches changed stylesheets briefly, then broadcasts a single
// asset_swap message so clients refresh the matching <link> tags in place.
func (parseLrs *LiveReloadServer) queueAssetSwap(parsePath string) {
	parseLrs.mutex.Lock()
	if parseLrs.changedAssets == nil {
		parseLrs.changedAssets = make(map[string]time.Time)
	}
	parseLrs.changedAssets[parsePath] = time.Now()
	if parseLrs.assetDebounceTimer != nil {
		parseLrs.assetDebounceTimer.Stop()
	}
	parseLrs.assetDebounceTimer = time.AfterFunc(100*time.Millisecond, func() {
		parseLrs.mutex.Lock()
		parseAssets := parseLrs.changedAssets
		parseLrs.changedAssets = nil
		parseLrs.assetDebounceTimer = nil
		parseLrs.mutex.Unlock()

		parsePaths := make([]string, 0, len(parseAssets))
		for parseAsset := range parseAssets {
			parseRel, parseRelErr := filepath.Rel(parseLrs.projectRoot, parseAsset)
			if parseRelErr != nil || strings.HasPrefix(parseRel, "..") {
				parseRel = filepath.Base(parseAsset)
			}
			parsePaths = append(parsePaths, "/"+filepath.ToSlash(parseRel))
		}
		sort.Strings(parsePaths)
		fmt.Printf("Ã°Å¸Å½Â¨ Hot-swapping %d stylesheet(s): %s\n", len(parsePaths), strings.Join(parsePaths, ", "))
		parseLrs.broadcastMessage(MessageTypeAssetSwap, map[string]interface{}{
			"assets": parsePaths,
			"kind":   "css",
		})
	})
	parseLrs.mutex.Unlock()
}

func (parseLrs *LiveReloadServer) debounceAndBuild() {
	parseLrs.mutex.Lock()
	defer parseLrs.mutex.Unlock()

	if parseLrs.currentBuild != nil && parseLrs.currentBuild.Process != nil {
		if !parseLrs.buildQueued {
			fmt.Printf("Ã¢ÂÂ­Ã¯Â¸Â  Build already running (PID: %d); queueing one follow-up rebuild\n", parseLrs.currentBuild.Process.Pid)
		}
		parseLrs.buildQueued = true
		parseLrs.broadcastMessage(MessageTypeDebounceStatus, map[string]interface{}{
			"changeCount":          parseLrs.changeCount,
			"timeSinceFirstChange": int64(0),
			"waitTime":             int64(0),
			"maxWaitTime":          maxDebounceTime.Milliseconds(),
			"queuedBehindBuild":    true,
		})
		return
	}

	parseNow := time.Now()

	// Track if this is the first change in a batch
	if parseLrs.firstChangeTime.IsZero() {
		parseLrs.firstChangeTime = parseNow
		parseLrs.changeCount = 0
	}

	parseLrs.changeCount++

	// Calculate smart debounce time based on change pattern
	var parseSmartDebounceTime time.Duration
	parseTimeSinceFirstChange := parseNow.Sub(parseLrs.firstChangeTime)

	if parseLrs.changeCount == 1 {
		// First change - use longer debounce to give user time to continue typing
		parseSmartDebounceTime = debounceTime
		fmt.Printf("Ã¢ÂÂ±Ã¯Â¸Â  First change detected, waiting %v for more changes...\n", parseSmartDebounceTime)
	} else if parseLrs.changeCount <= 3 && parseTimeSinceFirstChange < 10*time.Second {
		// Multiple quick changes - user is actively typing, extend wait
		parseSmartDebounceTime = debounceTime
		fmt.Printf("Ã¢ÂÂ±Ã¯Â¸Â  Change #%d detected, extending wait %v (user actively typing)...\n", parseLrs.changeCount, parseSmartDebounceTime)
	} else {
		// Many changes or been waiting a while - use shorter debounce
		parseSmartDebounceTime = quickDebounceTime
		fmt.Printf("Ã¢ÂÂ±Ã¯Â¸Â  Change #%d detected, using quick debounce %v...\n", parseLrs.changeCount, parseSmartDebounceTime)
	}

	// Don't wait longer than maxDebounceTime total
	if parseTimeSinceFirstChange > maxDebounceTime-parseSmartDebounceTime {
		parseSmartDebounceTime = maxDebounceTime - parseTimeSinceFirstChange
		if parseSmartDebounceTime <= 0 {
			fmt.Printf("Ã¢ÂÂ° Maximum debounce time reached, building immediately\n")
			parseLrs.resetDebounceState()
			go parseLrs.triggerBuild()
			return
		}
		fmt.Printf("Ã¢ÂÂ° Approaching max debounce time, will build in %v\n", parseSmartDebounceTime)
	}

	// Reset the debounce timer
	if parseLrs.debounceTimer != nil {
		parseLrs.debounceTimer.Stop()
	}

	// Reset max debounce timer if this is the first change
	if parseLrs.changeCount == 1 {
		if parseLrs.maxDebounceTimer != nil {
			parseLrs.maxDebounceTimer.Stop()
		}
		parseLrs.maxDebounceTimer = time.AfterFunc(maxDebounceTime, func() {
			fmt.Printf("Ã¢ÂÂ° Maximum debounce time (%v) reached, forcing build\n", maxDebounceTime)
			parseLrs.mutex.Lock()
			parseLrs.resetDebounceState()
			parseLrs.mutex.Unlock()
			parseLrs.triggerBuild()
		})
	}

	parseLrs.debounceTimer = time.AfterFunc(parseSmartDebounceTime, func() {
		parseLrs.mutex.Lock()
		parseLrs.resetDebounceState()
		parseLrs.mutex.Unlock()
		parseLrs.triggerBuild()
	})

	// Notify clients about debouncing status
	parseLrs.broadcastMessage(MessageTypeDebounceStatus, map[string]interface{}{
		"changeCount":          parseLrs.changeCount,
		"timeSinceFirstChange": parseTimeSinceFirstChange.Milliseconds(),
		"waitTime":             parseSmartDebounceTime.Milliseconds(),
		"maxWaitTime":          maxDebounceTime.Milliseconds(),
	})
}

func (parseLrs *LiveReloadServer) triggerBuild() {
	// Classify the update before building
	parseClassifyStart := time.Now()
	parseClassification := parseLrs.classifyUpdate()
	parseClassifyMs := time.Since(parseClassifyStart).Milliseconds()
	fmt.Printf("Ã°Å¸â€Â Update classification: %s (%s) - %s\n",
		parseClassification.Type, parseClassification.ReloadType, parseClassification.Reason)

	fmt.Println("Ã°Å¸â€Â¨ Starting WASM build...")
	parseStartTime := time.Now()

	buildStatus := BuildStatus{
		Success:      false,
		ReloadType:   parseClassification.ReloadType,
		Phase:        "compiling",
		PhaseSummary: "compiling a new wasm artifact while the previous output remains live",
		StaleOutput:  true,
	}
	parseLrs.mutex.Lock()
	parseLrs.lastClassification = parseClassification
	parseLrs.lastBuildStatus = &buildStatus
	parseLrs.mutex.Unlock()

	// Notify clients that build started with classification info
	parseLrs.broadcastMessage(MessageTypeBuildStart, map[string]interface{}{
		"classification": parseClassification,
		"status":         buildStatus,
	})
	if parseClassification.ReloadType == "hot" {
		parseLrs.clearPendingStateSnapshot()
		parseLrs.requestStateSnapshot()
	}

	// Create the build command
	if parseErr := os.MkdirAll(filepath.Dir(parseLrs.outputPath), 0o755); parseErr != nil {
		emitLivereloadError("LiveReloadServer.triggerBuild.mkdir", filepath.Dir(parseLrs.outputPath), parseErr, "the livereload output directory could not be created before rebuilding.", "Inspect the configured build root and directory permissions for the livereload artifact path.")
		parseLrs.clearPendingStateSnapshot()
		parseLrs.lastBuildStatus = &BuildStatus{
			Success:      false,
			Error:        fmt.Sprintf("Failed to prepare build output directory: %v", parseErr),
			ReloadType:   "none",
			Phase:        "blocked_on_error",
			PhaseSummary: "blocked on a build error; the last good output is all the dev server can still serve",
			StaleOutput:  true,
		}
		parseLrs.broadcastMessage(MessageTypeBuildError, *parseLrs.lastBuildStatus)
		return
	}
	// Dev builds skip DWARF generation (-ldflags=-w): the linker dominates
	// rebuild latency and debug info is not consumed by browser dev sessions.
	parseCmd := exec.Command(buildCommand, "build", "-ldflags=-w", "-o", parseLrs.outputPath)

	parseCmd.Dir = parseLrs.buildDir
	parseCmd.Env = append(os.Environ(), buildEnv...)

	// Capture stdout and stderr for error reporting
	var parseStdout, parseStderr bytes.Buffer
	parseCmd.Stdout = io.MultiWriter(os.Stdout, &parseStdout)
	parseCmd.Stderr = io.MultiWriter(os.Stderr, &parseStderr)

	fmt.Printf("Ã°Å¸Ââ€”Ã¯Â¸Â  Build process started (PID: will be available after start)\n")

	// Start the build process (non-blocking)
	parseErr2 := parseCmd.Start()
	if parseErr2 != nil {
		emitLivereloadError("LiveReloadServer.triggerBuild.start", parseLrs.buildDir, parseErr2, "the rebuild never started, so connected clients remain on the previous artifact state.", "Inspect the build command, working directory, and output path for the livereload session.")
		parseLrs.clearPendingStateSnapshot()
		parseFailedStatus := &BuildStatus{
			Success:      false,
			Error:        fmt.Sprintf("Failed to start build: %v", parseErr2),
			ReloadType:   "none",
			Phase:        "blocked_on_error",
			PhaseSummary: "blocked on a build error; the last good output is all the dev server can still serve",
			StaleOutput:  true,
		}
		parseLrs.mutex.Lock()
		parseLrs.currentBuild = nil
		parseLrs.lastBuildStatus = parseFailedStatus
		parseLrs.mutex.Unlock()
		parseLrs.broadcastMessage(MessageTypeBuildError, *parseFailedStatus)
		return
	}
	parseLrs.mutex.Lock()
	parseLrs.currentBuild = parseCmd
	parseLrs.mutex.Unlock()

	fmt.Printf("Ã°Å¸Ââ€”Ã¯Â¸Â  Build process running (PID: %d)\n", parseCmd.Process.Pid)

	// Wait for the build to complete
	parseErr2 = parseCmd.Wait()

	parseDuration := time.Since(parseStartTime)

	// Check if the process was killed vs completed naturally
	if parseErr2 != nil {
		if parseCmd.ProcessState != nil && parseCmd.ProcessState.String() == "signal: killed" {
			fmt.Printf("Ã¢ÂÂ¹Ã¯Â¸Â  Build was cancelled after %v\n", parseDuration)
		} else {
			// Get the actual build error output
			buildError := strings.TrimSpace(parseStderr.String())
			if buildError == "" {
				buildError = parseErr2.Error()
			}

			buildStatus := BuildStatus{
				Success:      false,
				Error:        buildError,
				ReloadType:   "none",
				Phase:        "blocked_on_error",
				PhaseSummary: "blocked on a build error; the last good output is all the dev server can still serve",
				StaleOutput:  true,
			}

			diagnostics.Emit(livereloadReport(
				"LiveReloadServer.triggerBuild.wait",
				parseLrs.buildDir,
				buildError,
				"the rebuild failed, so no new wasm artifact or reload event was produced for connected clients.",
				"Inspect the captured build stderr and fix the compile error before relying on the next livereload cycle.",
			))
			parseLrs.clearPendingStateSnapshot()
			parseLrs.mutex.Lock()
			parseLrs.lastBuildStatus = &buildStatus
			parseLrs.mutex.Unlock()
			parseLrs.broadcastMessage(MessageTypeBuildComplete, buildStatus)
		}
	} else {
		parseManifest, parseManifestErr := parseLrs.buildChangedComponentManifest(parseClassification)
		if parseManifestErr != nil {
			emitLivereloadError("LiveReloadServer.triggerBuild.manifest", parseLrs.manifestPath, parseManifestErr, "hot-reload metadata was not generated, so the next client update may fall back to less precise behavior.", "Inspect component-manifest generation for unsupported files or parser failures.")
		}
		if parseManifest != nil {
			if parseErr3 := parseLrs.writeChangedComponentManifest(parseManifest); parseErr3 != nil {
				emitLivereloadError("LiveReloadServer.triggerBuild.writeManifest", parseLrs.manifestPath, parseErr3, "the changed-component manifest was not written, so downstream tooling cannot consume precise hot-reload metadata.", "Inspect manifest path permissions and filesystem availability before writing the hot-reload manifest.")
			}
		}

		parseTimings := map[string]int64{
			"classifyMs": parseClassifyMs,
			"compileMs":  parseDuration.Milliseconds(),
			"totalMs":    time.Since(parseClassifyStart).Milliseconds(),
		}
		if parseArtifactInfo, parseStatErr := os.Stat(parseLrs.outputPath); parseStatErr == nil {
			parseTimings["artifactBytes"] = parseArtifactInfo.Size()
			// Pre-compress now so the browser's reload fetch is served from
			// cache immediately, and the transfer size is known up front.
			if parseGzipBytes := parseLrs.precompressWASMArtifact(parseArtifactInfo.ModTime()); parseGzipBytes > 0 {
				parseTimings["artifactGzipBytes"] = parseGzipBytes
				fmt.Printf("📦 Artifact: %.2f MB raw / %.2f MB gzip transfer\n",
					float64(parseArtifactInfo.Size())/1024/1024, float64(parseGzipBytes)/1024/1024)
			}
		}

		buildStatus := BuildStatus{
			Success:      true,
			Duration:     parseDuration.String(),
			ReloadType:   parseClassification.ReloadType,
			Phase:        "waiting_for_reload",
			PhaseSummary: "build finished; waiting for the browser to load the fresh artifact",
			StaleOutput:  true,
			ManifestPath: parseLrs.manifestPath,
			Manifest:     parseManifest,
			Timings:      parseTimings,
		}
		if buildStatus.ReloadType == "hot" {
			buildStatus.StateSnapshot = parseLrs.takePendingStateSnapshot()
		} else {
			parseLrs.clearPendingStateSnapshot()
		}

		fmt.Printf("Ã¢Å“â€¦ Build completed successfully in %v\n", parseDuration)
		parseLrs.mutex.Lock()
		parseLrs.lastBuildStatus = &buildStatus
		parseLrs.mutex.Unlock()
		parseLrs.broadcastMessage(MessageTypeBuildComplete, buildStatus)
	}

	parseLrs.mutex.Lock()
	parseQueuedRebuild := parseLrs.buildQueued
	parseLrs.buildQueued = false
	parseLrs.currentBuild = nil
	parseLrs.mutex.Unlock()

	if parseQueuedRebuild {
		fmt.Println("Ã°Å¸â€Â Running queued rebuild for changes that landed during the previous compile")
		go parseLrs.triggerBuild()
	}
}

func (parseLrs *LiveReloadServer) extractChangedComponents(parseFilePath string) ([]ChangedComponent, error) {
	if strings.TrimSpace(parseFilePath) == "" || !strings.HasSuffix(parseFilePath, ".go") {
		return nil, nil
	}

	parseFset := token.NewFileSet()
	parseParsed, parseErr := parser.ParseFile(parseFset, parseFilePath, nil, 0)
	if parseErr != nil {
		return nil, fmt.Errorf("parse %s: %w", parseFilePath, parseErr)
	}

	parseRelFile := parseFilePath
	if parseRelative, parseErr2 := filepath.Rel(parseLrs.watchRoot, parseFilePath); parseErr2 == nil {
		parseRelFile = filepath.ToSlash(parseRelative)
	}
	parsePackagePath := resolvePackagePath(parseLrs.modulePath, parseLrs.watchRoot, parseFilePath)
	parsePackageName := ""
	if parseParsed.Name != nil {
		parsePackageName = parseParsed.Name.Name
	}

	var parseComponents []ChangedComponent
	for _, parseDecl := range parseParsed.Decls {
		switch parseTyped := parseDecl.(type) {
		case *ast.FuncDecl:
			if parseTyped.Name == nil || parseTyped.Recv != nil || !returnsComponentNode(parseTyped.Type) {
				continue
			}
			parseComponents = append(parseComponents, ChangedComponent{
				Name:          parseTyped.Name.Name,
				QualifiedName: qualifyComponentName(parsePackagePath, parseTyped.Name.Name),
				PackageName:   parsePackageName,
				PackagePath:   parsePackagePath,
				File:          parseRelFile,
			})
		case *ast.GenDecl:
			if parseTyped.Tok != token.VAR {
				continue
			}
			for _, parseSpec := range parseTyped.Specs {
				parseValueSpec, parseOk := parseSpec.(*ast.ValueSpec)
				if !parseOk {
					continue
				}
				for parseIndex, parseName := range parseValueSpec.Names {
					if parseName == nil || !isComponentValueSpec(parseValueSpec, parseIndex) {
						continue
					}
					parseComponents = append(parseComponents, ChangedComponent{
						Name:          parseName.Name,
						QualifiedName: qualifyComponentName(parsePackagePath, parseName.Name),
						PackageName:   parsePackageName,
						PackagePath:   parsePackagePath,
						File:          parseRelFile,
					})
				}
			}
		}
	}

	return parseComponents, nil
}

func (parseLrs *LiveReloadServer) cleanup() {
	parseLrs.mutex.Lock()
	defer parseLrs.mutex.Unlock()

	if parseLrs.debounceTimer != nil {
		parseLrs.debounceTimer.Stop()
	}

	if parseLrs.maxDebounceTimer != nil {
		parseLrs.maxDebounceTimer.Stop()
	}

	if parseLrs.currentBuild != nil && parseLrs.currentBuild.Process != nil {
		fmt.Println("Ã°Å¸â€ºâ€˜ Killing running build process...")
		terminateLivereloadProcessTree(parseLrs.currentBuild)
		_ = parseLrs.currentBuild.Wait()
	}

	if parseLrs.watcher != nil {
		parseLrs.watcher.Close()
	}

	if parseLrs.httpServer != nil {
		parseLrs.httpServer.Close()
	}

	// Close all WebSocket connections
	parseLrs.clientsMutex.Lock()
	for parseConn := range parseLrs.clients {
		parseConn.Close()
	}
	parseLrs.clientsMutex.Unlock()
}
