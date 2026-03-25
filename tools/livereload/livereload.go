package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
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
	"sync/atomic"
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

func (parseLrs *LiveReloadServer) newHTTPHandler() http.Handler {
	parseMux := http.NewServeMux()
	parseFileServer := http.FileServer(http.Dir(parseLrs.projectRoot))
	parseMux.HandleFunc("/__gwc/status", parseLrs.handleStatus)
	parseMux.HandleFunc("/__gwc/clients/disconnect", parseLrs.handleClientDisconnect)
	if parseLrs.staticDir != "" {
		parseStaticFileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(parseLrs.staticDir)))
		parseMux.Handle("/static/", parseStaticFileServer)
	}
	parseServedWASMPath := parseLrs.servedWASMPath()
	if parseServedWASMPath != "" {
		parseMux.HandleFunc(parseServedWASMPath, func(parseW http.ResponseWriter, parseR *http.Request) {
			http.ServeFile(parseW, parseR, parseLrs.outputPath)
		})
	}
	parseMux.HandleFunc("/", func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
		if parseR2.URL.Path == "/" {
			parseLrs.handleHTML(parseW2, parseR2, parseLrs.indexPath)
			return
		}
		if strings.HasSuffix(parseR2.URL.Path, ".html") {
			parseRelPath := strings.TrimPrefix(parseR2.URL.Path, "/")
			parseLrs.handleHTML(parseW2, parseR2, filepath.Join(parseLrs.projectRoot, filepath.FromSlash(parseRelPath)))
			return
		}
		parseFileServer.ServeHTTP(parseW2, parseR2)
	})
	parseMux.HandleFunc("/ws", parseLrs.handleWebSocketManaged)
	return parseMux
}

func (parseLrs *LiveReloadServer) statusURL() string {
	return "http://" + netAddr(parseLrs.host, parseLrs.port) + "/__gwc/status"
}

func (parseLrs *LiveReloadServer) websocketURL() string {
	return "ws://" + netAddr(parseLrs.host, parseLrs.port) + "/ws"
}

func (parseLrs *LiveReloadServer) currentStatus() LiveReloadStatus {
	parseLrs.mutex.Lock()
	parseLastClassification := parseLrs.lastClassification
	var parseLastBuild *BuildStatus
	var parseCurrentError *BuildStatus
	if parseLrs.lastBuildStatus != nil {
		buildCopy := *parseLrs.lastBuildStatus
		parseLastBuild = &buildCopy
		if !buildCopy.Success {
			parseErrorCopy := buildCopy
			parseCurrentError = &parseErrorCopy
		}
	}
	parseLrs.mutex.Unlock()

	parseClients := parseLrs.currentClientSessions()
	parseClientCount := len(parseClients)

	parseStatus := LiveReloadStatus{
		Mode:               "livereload-wasm",
		ListeningURL:       "http://" + netAddr(parseLrs.host, parseLrs.port),
		StatusURL:          parseLrs.statusURL(),
		WebSocketURL:       parseLrs.websocketURL(),
		ProjectRoot:        parseLrs.projectRoot,
		WatchRoot:          parseLrs.watchRoot,
		BuildDir:           parseLrs.buildDir,
		ServedWASMPath:     parseLrs.servedWASMPath(),
		HotReloadEnabled:   parseLrs.alwaysHotReload,
		HotReloadEligible:  parseLrs.alwaysHotReload || parseLastClassification.ReloadType == "hot",
		LastClassification: parseLastClassification,
		LastBuild:          parseLastBuild,
		CurrentError:       parseCurrentError,
		ClientCount:        parseClientCount,
		Clients:            parseClients,
	}
	return parseStatus
}

func (parseLrs *LiveReloadServer) currentClientSessions() []ClientSession {
	parseLrs.clientsMutex.RLock()
	defer parseLrs.clientsMutex.RUnlock()

	if len(parseLrs.clients) == 0 {
		return nil
	}

	parseSessions := make([]ClientSession, 0, len(parseLrs.clients))
	for _, parseSession := range parseLrs.clients {
		parseSessions = append(parseSessions, parseSession)
	}
	sort.Slice(parseSessions, func(parseI, parseJ int) bool {
		if parseSessions[parseI].ConnectedAt.Equal(parseSessions[parseJ].ConnectedAt) {
			return parseSessions[parseI].ID < parseSessions[parseJ].ID
		}
		return parseSessions[parseI].ConnectedAt.Before(parseSessions[parseJ].ConnectedAt)
	})
	return parseSessions
}

func (parseLrs *LiveReloadServer) handleStatus(parseW http.ResponseWriter, parseR *http.Request) {
	parseStatus := parseLrs.currentStatus()
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseEncoded, parseErr := json.MarshalIndent(parseStatus, "", "  ")
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, livereloadErrReport(
			"LiveReloadServer.handleStatus.marshal",
			parseR.URL.Path,
			parseErr,
			"the machine-readable dev status endpoint could not serialize the current livereload state.",
			"Inspect the status payload fields and marshalling assumptions before relying on editor or tooling integrations.",
		))
		return
	}
	_, _ = parseW.Write(parseEncoded)
}

func (parseLrs *LiveReloadServer) handleClientDisconnect(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseW.Header().Set("Allow", http.MethodPost)
		http.Error(parseW, "disconnect endpoint only supports POST", http.StatusMethodNotAllowed)
		return
	}

	parseRequest := clientDisconnectRequest{
		ClientID: strings.TrimSpace(parseR.URL.Query().Get("clientID")),
		All:      strings.EqualFold(strings.TrimSpace(parseR.URL.Query().Get("all")), "true"),
	}
	if parseR.Body != nil && parseR.ContentLength != 0 {
		defer parseR.Body.Close()
		var parseDecoded clientDisconnectRequest
		if parseErr := json.NewDecoder(parseR.Body).Decode(&parseDecoded); parseErr != nil {
			http.Error(parseW, "invalid disconnect request payload", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(parseDecoded.ClientID) != "" {
			parseRequest.ClientID = strings.TrimSpace(parseDecoded.ClientID)
		}
		parseRequest.All = parseRequest.All || parseDecoded.All
	}
	if !parseRequest.All && parseRequest.ClientID == "" {
		http.Error(parseW, "specify clientID or all=true", http.StatusBadRequest)
		return
	}

	parseDisconnectedIDs, parseRemaining := parseLrs.disconnectClients(parseRequest.ClientID, parseRequest.All)
	parseResponse := clientDisconnectResponse{
		DisconnectedIDs: parseDisconnectedIDs,
		Disconnected:    len(parseDisconnectedIDs),
		Remaining:       parseRemaining,
	}
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseEncoded, parseErr2 := json.MarshalIndent(parseResponse, "", "  ")
	if parseErr2 != nil {
		http.Error(parseW, "failed to encode disconnect response", http.StatusInternalServerError)
		return
	}
	_, _ = parseW.Write(parseEncoded)
}

func (parseLrs *LiveReloadServer) disconnectClients(parseClientID string, isDisconnectAll bool) ([]string, int) {
	parseLrs.clientsMutex.Lock()
	parseTargets := make([]*websocket.Conn, 0, len(parseLrs.clients))
	parseDisconnectedIDs := make([]string, 0, len(parseLrs.clients))
	for parseConn, parseSession := range parseLrs.clients {
		if isDisconnectAll || parseSession.ID == parseClientID {
			parseTargets = append(parseTargets, parseConn)
			parseDisconnectedIDs = append(parseDisconnectedIDs, parseSession.ID)
			delete(parseLrs.clients, parseConn)
			if !isDisconnectAll {
				break
			}
		}
	}
	parseRemaining := len(parseLrs.clients)
	parseLrs.clientsMutex.Unlock()

	sort.Strings(parseDisconnectedIDs)
	for _, parseConn2 := range parseTargets {
		_ = parseConn2.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Disconnected by gwc dashboard"), time.Now().Add(time.Second))
		_ = parseConn2.Close()
	}
	return parseDisconnectedIDs, parseRemaining
}

func (parseLrs *LiveReloadServer) nextClientSession() ClientSession {
	parseNow := time.Now().UTC()
	return ClientSession{
		ID:          fmt.Sprintf("client-%d", atomic.AddUint64(&parseLrs.nextClientID, 1)),
		ConnectedAt: parseNow,
		LastSeenAt:  parseNow,
	}
}

func (parseLrs *LiveReloadServer) markClientSeen(parseConn *websocket.Conn) {
	parseLrs.clientsMutex.Lock()
	defer parseLrs.clientsMutex.Unlock()
	parseSession, parseOk := parseLrs.clients[parseConn]
	if !parseOk {
		return
	}
	parseSession.LastSeenAt = time.Now().UTC()
	parseLrs.clients[parseConn] = parseSession
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
		fmt.Printf("🌐 Live reload server starting on http://%s\n", netAddr(parseLrs.host, parseLrs.port))
		if parseErr2 := parseLrs.httpServer.ListenAndServe(); parseErr2 != nil && parseErr2 != http.ErrServerClosed {
			emitLivereloadError("LiveReloadServer.Start.ListenAndServe", netAddr(parseLrs.host, parseLrs.port), parseErr2, "the HTTP listener stopped unexpectedly and browser clients can no longer connect.", "Inspect the bind address and listener lifetime for the livereload server.")
		}
	}()

	// Handle interrupt signal for graceful shutdown
	parseC := make(chan os.Signal, 1)
	signal.Notify(parseC, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🔄 Live reload started. Watching for .go file changes...")
	fmt.Printf("📂 Watching directory: %s\n", parseLrs.watchRoot)
	if parseLrs.watchRoot != parseLrs.projectRoot {
		fmt.Printf("🗂️  Serving project root: %s\n", parseLrs.projectRoot)
	}
	fmt.Printf("🧩 Building from: %s\n", parseLrs.buildDir)
	fmt.Printf("⏱️  Debounce time: %v\n", debounceTime)
	fmt.Printf("🌐 Server running on http://%s\n", netAddr(parseLrs.host, parseLrs.port))
	fmt.Println("🛑 Press Ctrl+C to stop")

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

			case <-parseC:
				fmt.Println("\n🛑 Shutting down live reload server...")
				parseLrs.cleanup()
				os.Exit(0)
			}
		}
	}()

	// Keep the main goroutine alive
	select {}
}

func (parseLrs *LiveReloadServer) handleHTML(parseW http.ResponseWriter, parseR *http.Request, parseFilePath string) {
	// Read the original html file
	parseHtmlContent, parseErr := os.ReadFile(parseFilePath)
	if parseErr != nil {
		http.NotFound(parseW, parseR)
		return
	}

	parseScriptContent, parseErr := livereloadClientScriptBytes(parseLrs.clientScriptPath)
	if parseErr != nil {
		parseScriptPath := strings.TrimSpace(parseLrs.clientScriptPath)
		if parseScriptPath == "" {
			parseScriptPath = "embedded launcher client script"
		}
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, livereloadErrReport(
			"LiveReloadServer.handleHTML.clientScript",
			parseScriptPath,
			parseErr,
			"the HTML response could not inject the livereload client script, so browser reload coordination is unavailable for this request.",
			"Verify the configured livereload client override path or restore the embedded launcher client asset before serving HTML through this tool.",
		))
		return
	}

	parseConfigScript := fmt.Sprintf("\n<script>\nwindow.__GWC_LIVERELOAD_CONFIG = Object.assign({}, window.__GWC_LIVERELOAD_CONFIG || {}, { wasmPath: %q });\n</script>", parseLrs.servedWASMPath())

	// Inject the live reload script before closing </body> tag
	parseLiveReloadScript := fmt.Sprintf("%s\n<script>\n%s\n</script>", parseConfigScript, string(parseScriptContent))

	// Insert the script before closing </body> tag
	parseModifiedContent := strings.Replace(string(parseHtmlContent), "</body>", parseLiveReloadScript+"\n</body>", 1)

	parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	parseW.Write([]byte(parseModifiedContent))
}

func (parseLrs *LiveReloadServer) servedWASMPath() string {
	if parseLrs == nil {
		return "/main.wasm"
	}

	parseOutputPath := strings.TrimSpace(parseLrs.outputPath)
	if parseOutputPath == "" {
		return "/main.wasm"
	}
	if !filepath.IsAbs(parseOutputPath) {
		parseOutputPath = filepath.Join(parseLrs.buildDir, parseOutputPath)
	}

	parseRelPath, parseErr := filepath.Rel(parseLrs.projectRoot, parseOutputPath)
	if parseErr == nil && parseRelPath != "" && parseRelPath != "." && !strings.HasPrefix(parseRelPath, "..") && !filepath.IsAbs(parseRelPath) {
		return "/" + filepath.ToSlash(parseRelPath)
	}

	parseBase := strings.TrimSpace(filepath.Base(parseOutputPath))
	if parseBase == "" || parseBase == "." || parseBase == string(filepath.Separator) {
		return "/main.wasm"
	}
	return "/" + filepath.ToSlash(parseBase)
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

	fmt.Printf("🔌 WebSocket client connected (total: %d)\n", len(parseLrs.clients))

	// Send current build status to the new client
	parseLrs.sendCurrentBuildStatus(parseConn)

	// Remove client when done
	defer func() {
		parseLrs.clientsMutex.Lock()
		delete(parseLrs.clients, parseConn)
		parseLrs.clientsMutex.Unlock()
		fmt.Printf("🔌 WebSocket client disconnected (remaining: %d)\n", len(parseLrs.clients))
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
				parseLrs.stateSnapshotMu.Lock()
				parseLrs.pendingStateSnapshot = parsePayload
				parseLrs.stateSnapshotMu.Unlock()
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
				parseLrs.stateSnapshotMu.Lock()
				parseLrs.pendingStateSnapshot = parsePayload
				parseLrs.stateSnapshotMu.Unlock()
			}
		}
	}
}

func (parseLrs *LiveReloadServer) requestStateSnapshot() {
	parseLrs.broadcastMessage(MessageTypeStateExport, map[string]string{
		"reason": "hot_reload",
	})
}

func (parseLrs *LiveReloadServer) takePendingStateSnapshot() string {
	parseLrs.stateSnapshotMu.Lock()
	defer parseLrs.stateSnapshotMu.Unlock()
	parseSnapshot := parseLrs.pendingStateSnapshot
	parseLrs.pendingStateSnapshot = ""
	return parseSnapshot
}

func (parseLrs *LiveReloadServer) clearPendingStateSnapshot() {
	parseLrs.stateSnapshotMu.Lock()
	parseLrs.pendingStateSnapshot = ""
	parseLrs.stateSnapshotMu.Unlock()
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
			fmt.Printf("📤 Sent current build status (%s) to new client\n", parseStatusText)
		}
	} else {
		// Try to check current build state by attempting a quick build check
		go parseLrs.checkCurrentBuildState(parseConn)
	}
}

func (parseLrs *LiveReloadServer) checkCurrentBuildState(parseConn *websocket.Conn) {
	// Do a quick build check to see if the current code compiles
	fmt.Println("🔍 Checking current build state for new client...")
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
		fmt.Println("✅ Current build state: OK")
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
				fmt.Printf("👀 Watching: %s\n", parsePath)
			}
		}

		return nil
	})
}

func (parseLrs *LiveReloadServer) handleFileEvent(parseEvent fsnotify.Event) {
	// Only handle .go files
	if !strings.HasSuffix(parseEvent.Name, ".go") {
		return
	}

	// Skip temporary files and test files
	if strings.Contains(parseEvent.Name, ".tmp") || strings.Contains(parseEvent.Name, "~") {
		return
	}

	// Only handle write and create events
	if parseEvent.Op&fsnotify.Write == fsnotify.Write || parseEvent.Op&fsnotify.Create == fsnotify.Create {
		fmt.Printf("📝 File changed: %s\n", parseEvent.Name)

		// Track the changed file
		parseLrs.mutex.Lock()
		parseLrs.changedFiles[parseEvent.Name] = time.Now()
		parseLrs.mutex.Unlock()

		parseLrs.debounceAndBuild()
	}
}

func (parseLrs *LiveReloadServer) debounceAndBuild() {
	parseLrs.mutex.Lock()
	defer parseLrs.mutex.Unlock()

	if parseLrs.currentBuild != nil && parseLrs.currentBuild.Process != nil {
		if !parseLrs.buildQueued {
			fmt.Printf("⏭️  Build already running (PID: %d); queueing one follow-up rebuild\n", parseLrs.currentBuild.Process.Pid)
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
		fmt.Printf("⏱️  First change detected, waiting %v for more changes...\n", parseSmartDebounceTime)
	} else if parseLrs.changeCount <= 3 && parseTimeSinceFirstChange < 10*time.Second {
		// Multiple quick changes - user is actively typing, extend wait
		parseSmartDebounceTime = debounceTime
		fmt.Printf("⏱️  Change #%d detected, extending wait %v (user actively typing)...\n", parseLrs.changeCount, parseSmartDebounceTime)
	} else {
		// Many changes or been waiting a while - use shorter debounce
		parseSmartDebounceTime = quickDebounceTime
		fmt.Printf("⏱️  Change #%d detected, using quick debounce %v...\n", parseLrs.changeCount, parseSmartDebounceTime)
	}

	// Don't wait longer than maxDebounceTime total
	if parseTimeSinceFirstChange > maxDebounceTime-parseSmartDebounceTime {
		parseSmartDebounceTime = maxDebounceTime - parseTimeSinceFirstChange
		if parseSmartDebounceTime <= 0 {
			fmt.Printf("⏰ Maximum debounce time reached, building immediately\n")
			parseLrs.resetDebounceState()
			go parseLrs.triggerBuild()
			return
		}
		fmt.Printf("⏰ Approaching max debounce time, will build in %v\n", parseSmartDebounceTime)
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
			fmt.Printf("⏰ Maximum debounce time (%v) reached, forcing build\n", maxDebounceTime)
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

// resetDebounceState resets the debouncing state after a build
func (parseLrs *LiveReloadServer) resetDebounceState() {
	parseLrs.firstChangeTime = time.Time{}
	parseLrs.changeCount = 0
	if parseLrs.maxDebounceTimer != nil {
		parseLrs.maxDebounceTimer.Stop()
		parseLrs.maxDebounceTimer = nil
	}
}

// classifyUpdate determines whether the changes require a hot reload or full page reload
func (parseLrs *LiveReloadServer) classifyUpdate() UpdateClassification {
	parseLrs.mutex.Lock()
	defer parseLrs.mutex.Unlock()

	var parseChangedFiles []string
	for parseFile := range parseLrs.changedFiles {
		parseChangedFiles = append(parseChangedFiles, parseFile)
	}

	// Clear the changed files after classification
	parseLrs.changedFiles = make(map[string]time.Time)

	if parseLrs.alwaysHotReload {
		parseReason := "App dev server hot reload"
		if len(parseChangedFiles) > 0 {
			parseReason = "Hot reload for app changes"
		}
		return newUpdateClassification("small", "hot", parseReason, parseChangedFiles)
	}

	if len(parseChangedFiles) == 0 {
		return newUpdateClassification("small", "hot", "No files changed", parseChangedFiles)
	}

	// Analyze the changed files to determine update type
	var parseHotReloadReasons []string
	for _, parseFile2 := range parseChangedFiles {
		parseRelPath, _ := filepath.Rel(parseLrs.projectRoot, parseFile2)
		parseRelPath = filepath.ToSlash(parseRelPath)

		// Always full reload for critical system files
		if strings.Contains(parseRelPath, "main.go") {
			return newUpdateClassification("big", "full", "Main function or entry point changed", parseChangedFiles)
		}

		if strings.Contains(parseRelPath, "fiber/fiber.go") ||
			strings.Contains(parseRelPath, "fiber/hooks.go") ||
			strings.Contains(parseRelPath, "fiber/types.go") ||
			strings.Contains(parseRelPath, "fiber/state_management.go") {
			return newUpdateClassification("big", "full", "Core fiber system changed", parseChangedFiles)
		}

		if strings.Contains(parseRelPath, "go.mod") || strings.Contains(parseRelPath, "go.sum") {
			return newUpdateClassification("big", "full", "Package dependencies changed", parseChangedFiles)
		}

		// Check if file is in examples/ directory (UI components)
		if strings.Contains(parseRelPath, "examples/") {
			parseHotReloadReasons = append(parseHotReloadReasons, "example components")
		}
		// Check if file is in website/ directory (UI components)
		if strings.Contains(parseRelPath, "website/") {
			parseHotReloadReasons = append(parseHotReloadReasons, "website components")
		}
	}

	// Remove duplicates from hotReloadReasons
	parseUniqueReasons := make(map[string]bool)
	var parseFinalReasons []string
	for _, parseReason2 := range parseHotReloadReasons {
		if !parseUniqueReasons[parseReason2] {
			parseUniqueReasons[parseReason2] = true
			parseFinalReasons = append(parseFinalReasons, parseReason2)
		}
	}

	// If we detected UI-related changes, use hot reload
	if len(parseFinalReasons) > 0 {
		return newUpdateClassification("small", "hot", "UI changes: "+strings.Join(parseFinalReasons, ", "), parseChangedFiles)
	}

	// Default to full reload for logic changes
	return newUpdateClassification("big", "full", "Logic changes detected, using full reload for safety", parseChangedFiles)
}

func (parseLrs *LiveReloadServer) triggerBuild() {
	// Classify the update before building
	parseClassification := parseLrs.classifyUpdate()
	fmt.Printf("🔍 Update classification: %s (%s) - %s\n",
		parseClassification.Type, parseClassification.ReloadType, parseClassification.Reason)

	fmt.Println("🔨 Starting WASM build...")
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
	parseCmd := exec.Command(buildCommand, "build", "-o", parseLrs.outputPath)

	parseCmd.Dir = parseLrs.buildDir
	parseCmd.Env = append(os.Environ(), buildEnv...)

	// Capture stdout and stderr for error reporting
	var parseStdout, parseStderr bytes.Buffer
	parseCmd.Stdout = io.MultiWriter(os.Stdout, &parseStdout)
	parseCmd.Stderr = io.MultiWriter(os.Stderr, &parseStderr)

	fmt.Printf("🏗️  Build process started (PID: will be available after start)\n")

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

	fmt.Printf("🏗️  Build process running (PID: %d)\n", parseCmd.Process.Pid)

	// Wait for the build to complete
	parseErr2 = parseCmd.Wait()

	parseDuration := time.Since(parseStartTime)

	// Check if the process was killed vs completed naturally
	if parseErr2 != nil {
		if parseCmd.ProcessState != nil && parseCmd.ProcessState.String() == "signal: killed" {
			fmt.Printf("⏹️  Build was cancelled after %v\n", parseDuration)
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

		buildStatus := BuildStatus{
			Success:      true,
			Duration:     parseDuration.String(),
			ReloadType:   parseClassification.ReloadType,
			Phase:        "waiting_for_reload",
			PhaseSummary: "build finished; waiting for the browser to load the fresh artifact",
			StaleOutput:  true,
			ManifestPath: parseLrs.manifestPath,
			Manifest:     parseManifest,
		}
		if buildStatus.ReloadType == "hot" {
			buildStatus.StateSnapshot = parseLrs.takePendingStateSnapshot()
		} else {
			parseLrs.clearPendingStateSnapshot()
		}

		fmt.Printf("✅ Build completed successfully in %v\n", parseDuration)
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
		fmt.Println("🔁 Running queued rebuild for changes that landed during the previous compile")
		go parseLrs.triggerBuild()
	}
}

func (parseLrs *LiveReloadServer) buildChangedComponentManifest(parseClassification UpdateClassification) (*ChangedComponentManifest, error) {
	parseManifest := &ChangedComponentManifest{
		GeneratedAt:  time.Now(),
		ReloadType:   parseClassification.ReloadType,
		Reason:       parseClassification.Reason,
		ChangedFiles: make([]string, 0, len(parseClassification.ChangedFiles)),
	}

	parseComponentsByQualifiedName := make(map[string]ChangedComponent)
	for _, parseFile := range parseClassification.ChangedFiles {
		parseRelFile := parseFile
		if parseRelative, parseErr := filepath.Rel(parseLrs.watchRoot, parseFile); parseErr == nil {
			parseRelFile = filepath.ToSlash(parseRelative)
		}
		parseManifest.ChangedFiles = append(parseManifest.ChangedFiles, parseRelFile)

		parseComponents, parseErr2 := parseLrs.extractChangedComponents(parseFile)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		for _, parseComponent := range parseComponents {
			parseComponentsByQualifiedName[parseComponent.QualifiedName] = parseComponent
		}
	}

	if len(parseComponentsByQualifiedName) > 0 {
		parseQualifiedNames := make([]string, 0, len(parseComponentsByQualifiedName))
		for parseQualifiedName := range parseComponentsByQualifiedName {
			parseQualifiedNames = append(parseQualifiedNames, parseQualifiedName)
		}
		sort.Strings(parseQualifiedNames)
		parseManifest.Components = make([]ChangedComponent, 0, len(parseQualifiedNames))
		for _, parseQualifiedName2 := range parseQualifiedNames {
			parseManifest.Components = append(parseManifest.Components, parseComponentsByQualifiedName[parseQualifiedName2])
		}
	}

	return parseManifest, nil
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

func isComponentValueSpec(parseSpec *ast.ValueSpec, parseIndex int) bool {
	if parseSpec == nil {
		return false
	}
	if parseFuncType, parseOk := parseSpec.Type.(*ast.FuncType); parseOk {
		return returnsComponentNode(parseFuncType)
	}
	if parseIndex >= len(parseSpec.Values) {
		return false
	}
	parseFuncLiteral, parseOk2 := parseSpec.Values[parseIndex].(*ast.FuncLit)
	if !parseOk2 {
		return false
	}
	return returnsComponentNode(parseFuncLiteral.Type)
}

func returnsComponentNode(parseFuncType *ast.FuncType) bool {
	if parseFuncType == nil || parseFuncType.Results == nil || len(parseFuncType.Results.List) != 1 {
		return false
	}
	return isComponentResultExpr(parseFuncType.Results.List[0].Type)
}

func isComponentResultExpr(parseExpr ast.Expr) bool {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name == "Node" || parseTyped.Name == "Element"
	case *ast.SelectorExpr:
		parsePackageIdent, parseOk := parseTyped.X.(*ast.Ident)
		if !parseOk {
			return false
		}
		return (parsePackageIdent.Name == "ui" && (parseTyped.Sel.Name == "Node" || parseTyped.Sel.Name == "Element")) ||
			(parsePackageIdent.Name == "runtime" && parseTyped.Sel.Name == "Element")
	case *ast.StarExpr:
		return isRuntimeElementExpr(parseTyped.X)
	default:
		return false
	}
}

func isRuntimeElementExpr(parseExpr ast.Expr) bool {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name == "Element"
	case *ast.SelectorExpr:
		parsePackageIdent, parseOk := parseTyped.X.(*ast.Ident)
		if !parseOk {
			return false
		}
		return (parsePackageIdent.Name == "ui" || parsePackageIdent.Name == "runtime") && parseTyped.Sel.Name == "Element"
	default:
		return false
	}
}

func qualifyComponentName(parsePackagePath string, parseName string) string {
	if strings.TrimSpace(parsePackagePath) == "" {
		return parseName
	}
	return parsePackagePath + "." + parseName
}

func resolvePackagePath(parseModulePath string, parseWatchRoot string, parseFilePath string) string {
	parseDirectory := filepath.Dir(parseFilePath)
	parseRelDirectory, parseErr := filepath.Rel(parseWatchRoot, parseDirectory)
	if parseErr != nil || parseRelDirectory == "." {
		return parseModulePath
	}
	parseRelDirectory = filepath.ToSlash(parseRelDirectory)
	if strings.TrimSpace(parseModulePath) == "" {
		return parseRelDirectory
	}
	return parseModulePath + "/" + parseRelDirectory
}

func resolveModulePath(parseRoot string) string {
	parseGoModPath := filepath.Join(parseRoot, "go.mod")
	parseContent, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		return ""
	}
	for _, parseLine := range strings.Split(string(parseContent), "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseTrimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(parseTrimmed, "module "))
		}
	}
	return ""
}

func (parseLrs *LiveReloadServer) writeChangedComponentManifest(parseManifest *ChangedComponentManifest) error {
	if parseManifest == nil || strings.TrimSpace(parseLrs.manifestPath) == "" {
		return nil
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseLrs.manifestPath), 0o755); parseErr != nil {
		return fmt.Errorf("create manifest dir: %w", parseErr)
	}
	parseData, parseErr2 := json.MarshalIndent(parseManifest, "", "  ")
	if parseErr2 != nil {
		return fmt.Errorf("marshal manifest: %w", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseLrs.manifestPath, parseData, 0o644); parseErr3 != nil {
		return fmt.Errorf("write manifest: %w", parseErr3)
	}
	return nil
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
		fmt.Println("🛑 Killing running build process...")
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

func resolveBuildDir(parseEntryPath string) (string, error) {
	parseAbsPath, parseErr := filepath.Abs(strings.TrimSpace(parseEntryPath))
	if parseErr != nil {
		return "", fmt.Errorf("failed to resolve entry path: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsPath)
	if parseErr != nil {
		return "", fmt.Errorf("failed to inspect entry path %s: %w", parseAbsPath, parseErr)
	}
	if parseInfo.IsDir() {
		return parseAbsPath, nil
	}
	return filepath.Dir(parseAbsPath), nil
}

func resolveModuleRoot(parseStartPath string) string {
	parseCurrent := strings.TrimSpace(parseStartPath)
	if parseCurrent == "" {
		return ""
	}

	parseAbsPath, parseErr := filepath.Abs(parseCurrent)
	if parseErr != nil {
		return ""
	}

	parseInfo, parseErr := os.Stat(parseAbsPath)
	if parseErr != nil {
		return ""
	}
	if !parseInfo.IsDir() {
		parseAbsPath = filepath.Dir(parseAbsPath)
	}

	for {
		if _, parseErr2 := os.Stat(filepath.Join(parseAbsPath, "go.mod")); parseErr2 == nil {
			return parseAbsPath
		}
		parseParent := filepath.Dir(parseAbsPath)
		if parseParent == parseAbsPath {
			return ""
		}
		parseAbsPath = parseParent
	}
}

func resolveStaticDir(parseProjectRoot string) string {
	parseCandidates := []string{
		filepath.Join(parseProjectRoot, "static"),
		filepath.Join(filepath.Dir(parseProjectRoot), "static"),
	}

	for _, parseCandidate := range parseCandidates {
		if parseInfo, parseErr := os.Stat(parseCandidate); parseErr == nil && parseInfo.IsDir() {
			return parseCandidate
		}
	}

	return ""
}

func resolveClientScriptPath() string {
	return resolveConfiguredClientScriptPath()
}

func firstExistingPath(parseCandidates ...string) string {
	for _, parseCandidate := range parseCandidates {
		if strings.TrimSpace(parseCandidate) == "" {
			continue
		}
		parseCleanCandidate := filepath.Clean(parseCandidate)
		if _, parseErr := os.Stat(parseCleanCandidate); parseErr == nil {
			return parseCleanCandidate
		}
	}
	return ""
}

func resolveConfiguredClientScriptPath() string {
	parseCwd, parseErr := livereloadConfigGetwd()
	if parseErr != nil {
		parseCwd = ""
	}
	parseResolved, parseOk, parseErr := runnerconfig.ResolveConfiguredPath(parseCwd, func(parsePaths runnerconfig.Paths) string {
		return parsePaths.LivereloadClientScript
	}, "livereloadClientScript", livereloadRunnerConfigFS())
	if parseErr != nil || !parseOk {
		return ""
	}
	if _, parseErr2 := os.Stat(parseResolved); parseErr2 == nil {
		return parseResolved
	}
	return ""
}

func resolveLivereloadRunnerConfigPath() string {
	parseCwd, parseErr := livereloadConfigGetwd()
	if parseErr != nil {
		parseCwd = ""
	}
	parseConfigPath, parseErr := runnerconfig.ResolveConfigPath(parseCwd, livereloadRunnerConfigFS())
	if parseErr != nil {
		return ""
	}
	return parseConfigPath
}

func netAddr(parseHost, parsePort string) string {
	if strings.TrimSpace(parseHost) == "" {
		parseHost = defaultHost
	}
	if strings.TrimSpace(parsePort) == "" {
		parsePort = defaultPort
	}
	return net.JoinHostPort(parseHost, parsePort)
}

func main() {
	parseAppPath := flag.String("app", "", "Path to the app main.go file or the app directory")
	parseMainPath := flag.String("main", "", "Legacy alias for -app")
	parseRootPath := flag.String("root", "", "Project root to watch and serve")
	parseHtmlPath := flag.String("html", "", "HTML file to serve, relative to the project root")
	parseIndexPath := flag.String("index", "", "Legacy alias for -html")
	parseWasmPath := flag.String("wasm", "", "WASM output path, relative to the build directory")
	parseOutputPath := flag.String("output", "", "Legacy alias for -wasm")
	parseHost := flag.String("host", defaultHost, "Host to bind")
	parsePort := flag.String("port", defaultPort, "Port to bind")
	parseHot := flag.Bool("hot", true, "Always use hot reload on successful rebuilds")
	parseClientScriptPath := flag.String("client-script", "", "Optional override path to a custom livereload client script")
	flag.Parse()

	parseSelectedAppPath := strings.TrimSpace(*parseAppPath)
	if parseSelectedAppPath == "" {
		parseSelectedAppPath = strings.TrimSpace(*parseMainPath)
	}
	parseSelectedHTMLPath := strings.TrimSpace(*parseHtmlPath)
	if parseSelectedHTMLPath == "" {
		parseSelectedHTMLPath = strings.TrimSpace(*parseIndexPath)
	}
	parseSelectedWASMPath := strings.TrimSpace(*parseWasmPath)
	if parseSelectedWASMPath == "" {
		parseSelectedWASMPath = strings.TrimSpace(*parseOutputPath)
	}

	parseServer, parseErr := NewLiveReloadServerWithOptions(LiveReloadOptions{
		MainPath:         parseSelectedAppPath,
		ProjectRoot:      *parseRootPath,
		IndexPath:        parseSelectedHTMLPath,
		OutputPath:       parseSelectedWASMPath,
		Host:             *parseHost,
		Port:             *parsePort,
		AlwaysHotReload:  *parseHot,
		ClientScriptPath: *parseClientScriptPath,
	})
	if parseErr != nil {
		fatalLivereloadStartup("main.NewLiveReloadServerWithOptions", strings.TrimSpace(*parseRootPath), parseErr, "Inspect the selected app, project root, HTML path, and client-script arguments before starting livereload again.")
	}

	defer parseServer.cleanup()

	parseErr = parseServer.Start()
	if parseErr != nil {
		fatalLivereloadStartup("main.Start", netAddr(*parseHost, *parsePort), parseErr, "Inspect watcher initialization and HTTP startup errors before restarting the livereload tool.")
	}
}
