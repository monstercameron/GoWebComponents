package main

import (
	"encoding/json"
	"errors"
	"go/ast"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
)

func TestLiveReloadOriginValidation(parseT *testing.T) {
	parseServer := &LiveReloadServer{host: "127.0.0.1", port: "8090"}

	parseAllowed := []string{
		"http://127.0.0.1:8090",
		"http://localhost:8090",
		"http://[::1]:8090",
		"", // no Origin header (non-browser client)
	}
	for _, parseOrigin := range parseAllowed {
		parseReq := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8090/ws", nil)
		if parseOrigin != "" {
			parseReq.Header.Set("Origin", parseOrigin)
		}
		if !parseServer.originAllowed(parseReq) {
			parseT.Fatalf("expected Origin %q to be allowed", parseOrigin)
		}
	}

	parseRejected := []string{
		"https://evil.example.test",
		"http://127.0.0.1:9999", // wrong port
		"http://localhost",      // default port, not the dev port
		"http://attacker.localhost:8090",
		"null",
	}
	for _, parseOrigin := range parseRejected {
		parseReq := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8090/ws", nil)
		parseReq.Header.Set("Origin", parseOrigin)
		if parseServer.originAllowed(parseReq) {
			parseT.Fatalf("expected Origin %q to be rejected", parseOrigin)
		}
	}

	// The opt-out flag accepts any origin (tunnel/LAN dev).
	parseOpenServer := &LiveReloadServer{host: "127.0.0.1", port: "8090", allowAnyOrigin: true}
	parseReq := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8090/ws", nil)
	parseReq.Header.Set("Origin", "https://evil.example.test")
	if !parseOpenServer.originAllowed(parseReq) {
		parseT.Fatal("expected allowAnyOrigin to accept a foreign Origin")
	}
}

func TestLiveReloadForeignOriginUpgradeRejected(parseT *testing.T) {
	parseServer := &LiveReloadServer{host: "127.0.0.1", port: "8090"}
	parseServer.upgrader = websocket.Upgrader{CheckOrigin: parseServer.originAllowed}

	parseHTTP := httptest.NewServer(http.HandlerFunc(parseServer.handleWebSocket))
	defer parseHTTP.Close()

	parseWSURL := "ws" + strings.TrimPrefix(parseHTTP.URL, "http")
	parseHeader := http.Header{}
	parseHeader.Set("Origin", "https://evil.example.test")
	parseConn, parseResp, parseErr := websocket.DefaultDialer.Dial(parseWSURL, parseHeader)
	if parseErr == nil {
		parseConn.Close()
		parseT.Fatal("expected foreign-origin websocket upgrade to be rejected")
	}
	if parseResp == nil || parseResp.StatusCode != http.StatusForbidden {
		parseGotStatus := 0
		if parseResp != nil {
			parseGotStatus = parseResp.StatusCode
		}
		parseT.Fatalf("expected 403 for foreign origin, got status %d (err %v)", parseGotStatus, parseErr)
	}
}

func dialWebsocketHarness(parseT *testing.T, parseHandler func(*websocket.Conn)) (*websocket.Conn, func()) {
	parseT.Helper()
	parseHttpServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseConn, parseErr := upgrader.Upgrade(parseW, parseR, nil)
		if parseErr != nil {
			parseT.Fatalf("upgrade websocket: %v", parseErr)
		}
		parseHandler(parseConn)
	}))
	parseWsURL := "ws" + strings.TrimPrefix(parseHttpServer.URL, "http")
	parseClientConn, _, parseErr2 := websocket.DefaultDialer.Dial(parseWsURL, nil)
	if parseErr2 != nil {
		parseHttpServer.Close()
		parseT.Fatalf("dial websocket harness: %v", parseErr2)
	}
	parseCleanup := func() {
		_ = parseClientConn.Close()
		parseHttpServer.Close()
	}
	return parseClientConn, parseCleanup
}

func TestLivereloadReportHelpers(parseT *testing.T) {
	parseReport := livereloadReport(" LiveReloadServer.Start ", " /tmp/status ", " summary ", " consequence ", " next ")
	if parseReport.Code != "GWC-TOOL-LIVERELOAD" || parseReport.Headline != "tool failure in LiveReloadServer.Start" {
		parseT.Fatalf("unexpected report envelope: %+v", parseReport)
	}
	if parseReport.Path != "/tmp/status" || parseReport.Summary != "summary" || parseReport.Runtime != "consequence" || parseReport.Next != "next" {
		parseT.Fatalf("expected report helper to trim fields, got %+v", parseReport)
	}

	parseErrReport := livereloadErrReport("subject", "/tmp/file", errors.New("boom"), "broken", "fix it")
	if parseErrReport.Summary != "boom" || parseErrReport.Path != "/tmp/file" {
		parseT.Fatalf("unexpected error report: %+v", parseErrReport)
	}

	parseStatus := newBuildStatus(" compiling ", " phase summary ")
	if parseStatus.Phase != "compiling" || parseStatus.PhaseSummary != "phase summary" {
		parseT.Fatalf("unexpected build status helper output: %+v", parseStatus)
	}
}

func TestCanonicalRunnerConfigExampleMatchesLivereloadSchema(parseT *testing.T) {
	parseExamplePath := filepath.Join("..", "..", "docs", "examples", "gwc-runner.example.json")
	parseContent, parseErr := os.ReadFile(parseExamplePath)
	if parseErr != nil {
		parseT.Fatalf("read canonical runner config example: %v", parseErr)
	}

	var parseOverrides runnerconfig.Overrides
	if parseErr2 := json.Unmarshal(parseContent, &parseOverrides); parseErr2 != nil {
		parseT.Fatalf("parse canonical runner config example: %v", parseErr2)
	}
	if parseOverrides.Paths.LivereloadClientScript != "" {
		parseT.Fatalf("expected canonical example to omit livereload client script, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.WorkspaceBuildRoot != "bin" {
		parseT.Fatalf("expected workspace build root in canonical example, got %#v", parseOverrides.Paths)
	}
}

func TestPendingStateSnapshotBufferRoundTrips(parseT *testing.T) {
	parseServer := &LiveReloadServer{}
	parseServer.pendingStateSnapshot = `{"sharedCounter":1}`

	if parseGot := parseServer.takePendingStateSnapshot(); parseGot != `{"sharedCounter":1}` {
		parseT.Fatalf("expected pending snapshot to round-trip, got %q", parseGot)
	}
	if parseGot2 := parseServer.takePendingStateSnapshot(); parseGot2 != "" {
		parseT.Fatalf("expected pending snapshot to clear after take, got %q", parseGot2)
	}

	parseServer.pendingStateSnapshot = `{"sharedCounter":2}`
	parseServer.clearPendingStateSnapshot()
	if parseGot3 := parseServer.takePendingStateSnapshot(); parseGot3 != "" {
		parseT.Fatalf("expected pending snapshot to clear explicitly, got %q", parseGot3)
	}
}

func TestDebounceAndBuildQueuesFollowUpWhileBuildRunning(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		currentBuild: &exec.Cmd{Process: &os.Process{Pid: 1234}},
	}

	parseServer.debounceAndBuild()

	if !parseServer.buildQueued {
		parseT.Fatalf("expected follow-up rebuild to queue while a build is already running")
	}
	if parseServer.currentBuild == nil || parseServer.currentBuild.Process == nil || parseServer.currentBuild.Process.Pid != 1234 {
		parseT.Fatalf("expected running build to stay active, got %#v", parseServer.currentBuild)
	}
	if parseServer.debounceTimer != nil {
		parseT.Fatalf("expected no debounce timer reset while build is already running")
	}
}

func TestBuildStatusMarshalsStateSnapshot(parseT *testing.T) {
	parseStatus := BuildStatus{
		Success:       true,
		ReloadType:    "hot",
		StateSnapshot: `{"sharedCounter":1}`,
		Manifest: &ChangedComponentManifest{
			ReloadType: "hot",
			Components: []ChangedComponent{{Name: "App", QualifiedName: "example.com/test.App", File: "main.go"}},
		},
	}

	parseData, parseErr := json.Marshal(parseStatus)
	if parseErr != nil {
		parseT.Fatalf("expected build status to marshal: %v", parseErr)
	}
	var parseDecoded map[string]any
	if parseErr2 := json.Unmarshal(parseData, &parseDecoded); parseErr2 != nil {
		parseT.Fatalf("expected marshaled build status to decode: %v", parseErr2)
	}
	if _, parseOk := parseDecoded["stateSnapshot"]; !parseOk {
		parseT.Fatalf("expected marshaled build status to include stateSnapshot, got %s", parseData)
	}
	if _, parseOk2 := parseDecoded["manifest"]; !parseOk2 {
		parseT.Fatalf("expected marshaled build status to include manifest, got %s", parseData)
	}
}

func TestWebSocketMessageProtocolDefaultsAndRejectsIncompatibleVersions(parseT *testing.T) {
	parseLegacy, parseErr := normalizeWebSocketMessage(WebSocketMessage{Type: MessageTypeReload})
	if parseErr != nil {
		parseT.Fatalf("expected legacy websocket message to normalize, got %v", parseErr)
	}
	if parseLegacy.Protocol != liveReloadProtocol || parseLegacy.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected legacy websocket message to default protocol metadata, got %+v", parseLegacy)
	}

	parseCurrent, parseErr2 := normalizeWebSocketMessage(newWebSocketMessage(MessageTypeReload, nil))
	if parseErr2 != nil {
		parseT.Fatalf("expected current websocket message to normalize, got %v", parseErr2)
	}
	if parseCurrent.Protocol != liveReloadProtocol || parseCurrent.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected current websocket message to preserve protocol metadata, got %+v", parseCurrent)
	}

	if _, parseErr3 := normalizeWebSocketMessage(WebSocketMessage{Protocol: liveReloadProtocol, Version: currentLiveReloadProtocolVersion + 1, Type: MessageTypeReload}); parseErr3 == nil {
		parseT.Fatal("expected future websocket protocol version to be rejected")
	}
	if _, parseErr4 := normalizeWebSocketMessage(WebSocketMessage{Protocol: "other.protocol", Version: 1, Type: MessageTypeReload}); parseErr4 == nil {
		parseT.Fatal("expected unknown websocket protocol to be rejected")
	}
}

func TestDescribeUpdateCompatibilityPlanHotReload(parseT *testing.T) {
	parseClassification := newUpdateClassification("small", "hot", "UI changes: example components", []string{"examples/public/hot-reload/main.go"})

	if !parseClassification.Plan.PreserveState {
		parseT.Fatalf("expected hot reload classification to preserve state, got %#v", parseClassification.Plan)
	}
	if !parseClassification.Plan.RemountSubtree {
		parseT.Fatalf("expected hot reload classification to warn about subtree remounts, got %#v", parseClassification.Plan)
	}
	if !parseClassification.Plan.RestartAsync {
		parseT.Fatalf("expected hot reload classification to restart async work, got %#v", parseClassification.Plan)
	}
	if parseClassification.Plan.FullReload {
		parseT.Fatalf("expected hot reload classification not to force full reload, got %#v", parseClassification.Plan)
	}
	if !strings.Contains(parseClassification.Plan.Summary, "preserve-state hot reload") {
		parseT.Fatalf("expected hot reload summary to explain the plan, got %q", parseClassification.Plan.Summary)
	}
}

func TestDescribeUpdateCompatibilityPlanFullReload(parseT *testing.T) {
	parseClassification := newUpdateClassification("big", "full", "Main function or entry point changed", []string{"main.go"})

	if parseClassification.Plan.PreserveState {
		parseT.Fatalf("expected full reload classification not to preserve state, got %#v", parseClassification.Plan)
	}
	if parseClassification.Plan.RemountSubtree {
		parseT.Fatalf("expected full reload classification not to advertise subtree remounts, got %#v", parseClassification.Plan)
	}
	if parseClassification.Plan.RestartAsync {
		parseT.Fatalf("expected full reload classification not to advertise async restart-only behavior, got %#v", parseClassification.Plan)
	}
	if !parseClassification.Plan.FullReload {
		parseT.Fatalf("expected full reload classification to force full reload, got %#v", parseClassification.Plan)
	}
	if !strings.Contains(parseClassification.Plan.Summary, "planned full reload") {
		parseT.Fatalf("expected full reload summary to explain the plan, got %q", parseClassification.Plan.Summary)
	}
}

func TestDescribeUpdateCompatibilityPlanEdgeCases(parseT *testing.T) {
	parseHotNoFiles := describeUpdateCompatibilityPlan(UpdateClassification{ReloadType: "hot"})
	if !parseHotNoFiles.PreserveState || parseHotNoFiles.RemountSubtree || parseHotNoFiles.RestartAsync {
		parseT.Fatalf("unexpected hot plan without changed files: %+v", parseHotNoFiles)
	}
	if !strings.Contains(parseHotNoFiles.Summary, "compatible local state is preserved") {
		parseT.Fatalf("unexpected hot-no-files summary: %q", parseHotNoFiles.Summary)
	}

	parseUnknown := describeUpdateCompatibilityPlan(UpdateClassification{ReloadType: "mystery"})
	if parseUnknown != (UpdateCompatibilityPlan{}) {
		parseT.Fatalf("expected empty compatibility plan for unknown reload type, got %+v", parseUnknown)
	}
}

func TestBuildChangedComponentManifestExtractsTopLevelComponents(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseWorkspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); parseErr != nil {
		parseT.Fatalf("failed to write go.mod fixture: %v", parseErr)
	}
	parseAppDir := filepath.Join(parseWorkspaceDir, "examples", "public", "hot-reload")
	if parseErr2 := os.MkdirAll(parseAppDir, 0o755); parseErr2 != nil {
		parseT.Fatalf("failed to create app dir: %v", parseErr2)
	}
	parseMainPath := filepath.Join(parseAppDir, "main.go")
	parseContent := `package main

import "github.com/monstercameron/GoWebComponents/ui"

func App() ui.Node { return nil }
func helper() int { return 1 }
var Banner = func() ui.Node { return nil }
`
	if parseErr3 := os.WriteFile(parseMainPath, []byte(parseContent), 0o644); parseErr3 != nil {
		parseT.Fatalf("failed to write main.go fixture: %v", parseErr3)
	}

	parseServer := &LiveReloadServer{
		watchRoot:    parseWorkspaceDir,
		modulePath:   "example.com/test",
		manifestPath: filepath.Join(parseWorkspaceDir, "main.wasm.hotreload-manifest.json"),
	}

	parseManifest, parseErr4 := parseServer.buildChangedComponentManifest(UpdateClassification{
		ReloadType:   "hot",
		Reason:       "UI changes",
		ChangedFiles: []string{parseMainPath},
	})
	if parseErr4 != nil {
		parseT.Fatalf("expected manifest build to succeed, got %v", parseErr4)
	}
	if len(parseManifest.Components) != 2 {
		parseT.Fatalf("expected 2 changed components, got %#v", parseManifest.Components)
	}
	if parseManifest.Components[0].QualifiedName != "example.com/test/examples/public/hot-reload.App" {
		parseT.Fatalf("unexpected first component: %#v", parseManifest.Components[0])
	}
	if parseManifest.Components[1].QualifiedName != "example.com/test/examples/public/hot-reload.Banner" {
		parseT.Fatalf("unexpected second component: %#v", parseManifest.Components[1])
	}
}

func TestWriteChangedComponentManifestPersistsJSON(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	parseManifestPath := filepath.Join(parseWorkspaceDir, "out", "main.wasm.hotreload-manifest.json")
	parseServer := &LiveReloadServer{manifestPath: parseManifestPath}
	parseManifest := &ChangedComponentManifest{
		ReloadType:   "hot",
		ChangedFiles: []string{"examples/public/hot-reload/main.go"},
		Components:   []ChangedComponent{{Name: "App", QualifiedName: "example.com/test/examples/public/hot-reload.App", File: "examples/public/hot-reload/main.go"}},
	}

	if parseErr := parseServer.writeChangedComponentManifest(parseManifest); parseErr != nil {
		parseT.Fatalf("expected manifest write to succeed, got %v", parseErr)
	}
	parseContent, parseErr2 := os.ReadFile(parseManifestPath)
	if parseErr2 != nil {
		parseT.Fatalf("expected manifest file to exist, got %v", parseErr2)
	}
	if !strings.Contains(string(parseContent), "example.com/test/examples/public/hot-reload.App") {
		parseT.Fatalf("expected manifest file to contain component identity, got %s", parseContent)
	}
}

func TestCurrentClientSessionsSortsAndMarkClientSeenUpdatesTimestamp(parseT *testing.T) {
	parseConnA := &websocket.Conn{}
	parseConnB := &websocket.Conn{}
	parseEarlier := time.Now().UTC().Add(-2 * time.Minute)
	parseLater := parseEarlier.Add(1 * time.Minute)
	parseServer := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{
			parseConnB: {ID: "client-b", ConnectedAt: parseLater, LastSeenAt: parseLater},
			parseConnA: {ID: "client-a", ConnectedAt: parseEarlier, LastSeenAt: parseEarlier},
		},
	}

	parseSessions := parseServer.currentClientSessions()
	if len(parseSessions) != 2 || parseSessions[0].ID != "client-a" || parseSessions[1].ID != "client-b" {
		parseT.Fatalf("expected sessions sorted by ConnectedAt then ID, got %+v", parseSessions)
	}

	parseServer.markClientSeen(parseConnA)
	parseUpdated := parseServer.clients[parseConnA]
	if !parseUpdated.LastSeenAt.After(parseEarlier) {
		parseT.Fatalf("expected markClientSeen to advance LastSeenAt, got %+v", parseUpdated)
	}

	parseServer.markClientSeen(&websocket.Conn{})
}

func TestHandleClientDisconnectValidation(parseT *testing.T) {
	parseServer := &LiveReloadServer{clients: map[*websocket.Conn]ClientSession{}}
	parseHandler := parseServer.newHTTPHandler()

	getReq := httptest.NewRequest(http.MethodGet, "/__gwc/clients/disconnect", nil)
	getResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusMethodNotAllowed {
		parseT.Fatalf("expected method-not-allowed for GET disconnect, got %d", getResp.Code)
	}

	parseMissingReq := httptest.NewRequest(http.MethodPost, "/__gwc/clients/disconnect", nil)
	parseMissingResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseMissingResp, parseMissingReq)
	if parseMissingResp.Code != http.StatusBadRequest {
		parseT.Fatalf("expected bad request for missing target, got %d", parseMissingResp.Code)
	}

	parseInvalidJSONReq := httptest.NewRequest(http.MethodPost, "/__gwc/clients/disconnect", strings.NewReader("{"))
	parseInvalidJSONReq.Header.Set("Content-Type", "application/json")
	parseInvalidJSONResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseInvalidJSONResp, parseInvalidJSONReq)
	if parseInvalidJSONResp.Code != http.StatusBadRequest {
		parseT.Fatalf("expected bad request for invalid JSON, got %d", parseInvalidJSONResp.Code)
	}
}

func TestResolveBuildDirUsesMainGoParent(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseMainPath := filepath.Join(parseDir, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\n"), 0o644); parseErr != nil {
		parseT.Fatalf("failed to create main.go fixture: %v", parseErr)
	}

	parseGot, parseErr2 := resolveBuildDir(parseMainPath)
	if parseErr2 != nil {
		parseT.Fatalf("expected build dir resolution to succeed, got %v", parseErr2)
	}
	if parseGot != parseDir {
		parseT.Fatalf("expected build dir %q, got %q", parseDir, parseGot)
	}
}

func TestResolveBuildDirUsesDirectoryInput(parseT *testing.T) {
	parseDir := parseT.TempDir()

	parseGot, parseErr := resolveBuildDir(parseDir)
	if parseErr != nil {
		parseT.Fatalf("expected build dir resolution to succeed, got %v", parseErr)
	}
	if parseGot != parseDir {
		parseT.Fatalf("expected build dir %q, got %q", parseDir, parseGot)
	}
}

func TestResolveModuleRootFindsNearestGoMod(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseWorkspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); parseErr != nil {
		parseT.Fatalf("failed to write go.mod fixture: %v", parseErr)
	}
	parseExampleDir := filepath.Join(parseWorkspaceDir, "examples", "public", "hot-reload")
	if parseErr2 := os.MkdirAll(parseExampleDir, 0o755); parseErr2 != nil {
		parseT.Fatalf("failed to create example dir: %v", parseErr2)
	}

	if parseGot := resolveModuleRoot(parseExampleDir); parseGot != parseWorkspaceDir {
		parseT.Fatalf("expected module root %q, got %q", parseWorkspaceDir, parseGot)
	}
}

func TestNewLiveReloadServerUsesModuleRootForWatching(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseWorkspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); parseErr != nil {
		parseT.Fatalf("failed to write go.mod fixture: %v", parseErr)
	}
	parseExampleDir := filepath.Join(parseWorkspaceDir, "examples", "public", "hot-reload")
	if parseErr2 := os.MkdirAll(parseExampleDir, 0o755); parseErr2 != nil {
		parseT.Fatalf("failed to create example dir: %v", parseErr2)
	}
	parseMainPath := filepath.Join(parseExampleDir, "main.go")
	if parseErr3 := os.WriteFile(parseMainPath, []byte("package main\n"), 0o644); parseErr3 != nil {
		parseT.Fatalf("failed to write main.go fixture: %v", parseErr3)
	}

	parseServer, parseErr4 := NewLiveReloadServerWithOptions(LiveReloadOptions{
		MainPath:    parseMainPath,
		ProjectRoot: parseExampleDir,
	})
	if parseErr4 != nil {
		parseT.Fatalf("expected server creation to succeed, got %v", parseErr4)
	}
	defer parseServer.cleanup()

	if parseServer.projectRoot != parseExampleDir {
		parseT.Fatalf("expected project root %q, got %q", parseExampleDir, parseServer.projectRoot)
	}
	if parseServer.watchRoot != parseWorkspaceDir {
		parseT.Fatalf("expected watch root %q, got %q", parseWorkspaceDir, parseServer.watchRoot)
	}
	if parseServer.buildDir != parseExampleDir {
		parseT.Fatalf("expected build dir %q, got %q", parseExampleDir, parseServer.buildDir)
	}
	parseWantOutputPath := filepath.Join(parseWorkspaceDir, "bin", "examples", "public", "hot-reload", "main.wasm")
	if parseServer.outputPath != parseWantOutputPath {
		parseT.Fatalf("expected output path %q, got %q", parseWantOutputPath, parseServer.outputPath)
	}
}

func TestNewHTTPHandlerServesParentStaticAssetsInSingleExampleMode(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseWorkspaceDir, "98-hot-reload")
	if parseErr := os.MkdirAll(filepath.Join(parseProjectRoot), 0o755); parseErr != nil {
		parseT.Fatalf("failed to create project root: %v", parseErr)
	}
	parseCssPath := filepath.Join(parseWorkspaceDir, "static", "css", "tailwind.css")
	if parseErr2 := os.MkdirAll(filepath.Dir(parseCssPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("failed to create parent static dir: %v", parseErr2)
	}
	const cssBody = "body { color: red; }"
	if parseErr3 := os.WriteFile(parseCssPath, []byte(cssBody), 0o644); parseErr3 != nil {
		parseT.Fatalf("failed to write static asset: %v", parseErr3)
	}

	parseServer := &LiveReloadServer{
		projectRoot: parseProjectRoot,
		staticDir:   resolveStaticDir(parseProjectRoot),
	}

	parseReq := httptest.NewRequest("GET", "/static/css/tailwind.css", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.newHTTPHandler().ServeHTTP(parseRecorder, parseReq)

	if parseRecorder.Code != 200 {
		parseT.Fatalf("expected 200 serving parent static asset, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseBody := parseRecorder.Body.String(); parseBody != cssBody {
		parseT.Fatalf("expected CSS body %q, got %q", cssBody, parseBody)
	}
	if parseContentType := parseRecorder.Header().Get("Content-Type"); !strings.Contains(parseContentType, "text/css") {
		parseT.Fatalf("expected CSS content type, got %q", parseContentType)
	}
	if parseServer.staticDir != filepath.Join(parseWorkspaceDir, "static") {
		parseT.Fatalf("expected parent static dir to be selected, got %q", parseServer.staticDir)
	}
}

func TestNewHTTPHandlerPrefersProjectStaticAssets(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseWorkspaceDir, "98-hot-reload")
	parseParentCSSPath := filepath.Join(parseWorkspaceDir, "static", "css", "app.css")
	parseProjectCSSPath := filepath.Join(parseProjectRoot, "static", "css", "app.css")
	if parseErr := os.MkdirAll(filepath.Dir(parseParentCSSPath), 0o755); parseErr != nil {
		parseT.Fatalf("failed to create parent static dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parseProjectCSSPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("failed to create project static dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseParentCSSPath, []byte("body { color: red; }"), 0o644); parseErr3 != nil {
		parseT.Fatalf("failed to write parent static asset: %v", parseErr3)
	}
	const projectCSSBody = "body { color: blue; }"
	if parseErr4 := os.WriteFile(parseProjectCSSPath, []byte(projectCSSBody), 0o644); parseErr4 != nil {
		parseT.Fatalf("failed to write project static asset: %v", parseErr4)
	}

	parseServer := &LiveReloadServer{
		projectRoot: parseProjectRoot,
		staticDir:   resolveStaticDir(parseProjectRoot),
	}

	parseReq := httptest.NewRequest("GET", "/static/css/app.css", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.newHTTPHandler().ServeHTTP(parseRecorder, parseReq)

	if parseRecorder.Code != 200 {
		parseT.Fatalf("expected 200 serving project static asset, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseBody := parseRecorder.Body.String(); parseBody != projectCSSBody {
		parseT.Fatalf("expected project CSS body %q, got %q", projectCSSBody, parseBody)
	}
	if parseServer.staticDir != filepath.Join(parseProjectRoot, "static") {
		parseT.Fatalf("expected project static dir to be selected, got %q", parseServer.staticDir)
	}
}

func TestNewHTTPHandlerReturnsNotFoundWithoutStaticDir(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseServer := &LiveReloadServer{
		projectRoot: parseProjectRoot,
		staticDir:   resolveStaticDir(parseProjectRoot),
	}

	parseReq := httptest.NewRequest("GET", "/static/css/missing.css", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.newHTTPHandler().ServeHTTP(parseRecorder, parseReq)

	if parseRecorder.Code != 404 {
		parseT.Fatalf("expected 404 when no static dir is available, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseServer.staticDir != "" {
		parseT.Fatalf("expected empty static dir when none exist, got %q", parseServer.staticDir)
	}
}

func TestNewHTTPHandlerServesStatusEndpoint(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseServer := &LiveReloadServer{
		projectRoot:        parseProjectRoot,
		watchRoot:          parseProjectRoot,
		buildDir:           parseProjectRoot,
		outputPath:         filepath.Join(parseProjectRoot, "main.wasm"),
		host:               "127.0.0.1",
		port:               "8099",
		clients:            map[*websocket.Conn]ClientSession{},
		alwaysHotReload:    true,
		lastClassification: newUpdateClassification("small", "hot", "UI changes: example components", []string{"main.go"}),
		lastBuildStatus: &BuildStatus{
			Success:      true,
			ReloadType:   "hot",
			Phase:        "serving_output",
			PhaseSummary: "serving the latest successful output",
			StaleOutput:  false,
		},
	}

	parseReq := httptest.NewRequest("GET", "/__gwc/status", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.newHTTPHandler().ServeHTTP(parseRecorder, parseReq)

	if parseRecorder.Code != 200 {
		parseT.Fatalf("expected 200 serving status endpoint, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}

	var parsePayload LiveReloadStatus
	if parseErr := json.Unmarshal(parseRecorder.Body.Bytes(), &parsePayload); parseErr != nil {
		parseT.Fatalf("expected JSON status payload, got %v", parseErr)
	}
	if parsePayload.Mode != "livereload-wasm" {
		parseT.Fatalf("expected mode livereload-wasm, got %#v", parsePayload)
	}
	if parsePayload.StatusURL != "http://127.0.0.1:8099/__gwc/status" {
		parseT.Fatalf("expected status URL to be reported, got %#v", parsePayload)
	}
	if parsePayload.WebSocketURL != "ws://127.0.0.1:8099/ws" {
		parseT.Fatalf("expected websocket URL to be reported, got %#v", parsePayload)
	}
	if !parsePayload.HotReloadEligible || !parsePayload.HotReloadEnabled {
		parseT.Fatalf("expected hot reload to be enabled and eligible, got %#v", parsePayload)
	}
	if parsePayload.LastBuild == nil || parsePayload.LastBuild.Phase != "serving_output" {
		parseT.Fatalf("expected last build phase in status payload, got %#v", parsePayload)
	}
}

func TestNewHTTPHandlerServesResolvedWasmOutsideProjectRoot(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseWorkspaceDir, "examples", "public", "hot-reload")
	parseOutputPath := filepath.Join(parseWorkspaceDir, "bin", "examples", "public", "hot-reload", "main.wasm")
	if parseErr := os.MkdirAll(parseProjectRoot, 0o755); parseErr != nil {
		parseT.Fatalf("failed to create project root: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parseOutputPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("failed to create wasm output dir: %v", parseErr2)
	}
	const wasmBody = "wasm-bytes"
	if parseErr3 := os.WriteFile(parseOutputPath, []byte(wasmBody), 0o644); parseErr3 != nil {
		parseT.Fatalf("failed to write wasm output: %v", parseErr3)
	}

	parseServer := &LiveReloadServer{
		projectRoot: parseProjectRoot,
		outputPath:  parseOutputPath,
	}

	parseReq := httptest.NewRequest("GET", "/main.wasm", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.newHTTPHandler().ServeHTTP(parseRecorder, parseReq)

	if parseRecorder.Code != 200 {
		parseT.Fatalf("expected 200 serving wasm output, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseBody := parseRecorder.Body.String(); parseBody != wasmBody {
		parseT.Fatalf("expected wasm body %q, got %q", wasmBody, parseBody)
	}
}

func TestNewHTTPHandlerReportsAndDisconnectsClientSessions(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseServer := &LiveReloadServer{
		projectRoot: parseProjectRoot,
		watchRoot:   parseProjectRoot,
		buildDir:    parseProjectRoot,
		outputPath:  filepath.Join(parseProjectRoot, "main.wasm"),
		host:        "127.0.0.1",
		port:        "8099",
		clients:     map[*websocket.Conn]ClientSession{},
	}

	parseTestServer := httptest.NewServer(parseServer.newHTTPHandler())
	defer parseTestServer.Close()

	parseWsURL := "ws" + strings.TrimPrefix(parseTestServer.URL, "http") + "/ws"
	parseHeaders := http.Header{}
	parseHeaders.Set("User-Agent", "gwc-dashboard-test")
	parseConn, _, parseErr := websocket.DefaultDialer.Dial(parseWsURL, parseHeaders)
	if parseErr != nil {
		parseT.Fatalf("dial websocket: %v", parseErr)
	}
	defer parseConn.Close()

	parseStatusResp, parseErr := http.Get(parseTestServer.URL + "/__gwc/status")
	if parseErr != nil {
		parseT.Fatalf("get status: %v", parseErr)
	}
	defer parseStatusResp.Body.Close()
	if parseStatusResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected status 200, got %s", parseStatusResp.Status)
	}

	var parseStatus LiveReloadStatus
	if parseErr2 := json.NewDecoder(parseStatusResp.Body).Decode(&parseStatus); parseErr2 != nil {
		parseT.Fatalf("decode status payload: %v", parseErr2)
	}
	if parseStatus.ClientCount != 1 || len(parseStatus.Clients) != 1 {
		parseT.Fatalf("expected one connected client in status, got %+v", parseStatus)
	}
	if parseStatus.Clients[0].ID == "" || parseStatus.Clients[0].UserAgent != "gwc-dashboard-test" {
		parseT.Fatalf("unexpected client metadata: %+v", parseStatus.Clients[0])
	}

	parseReq, parseErr := http.NewRequest(http.MethodPost, parseTestServer.URL+"/__gwc/clients/disconnect", strings.NewReader(`{"all":true}`))
	if parseErr != nil {
		parseT.Fatalf("build disconnect request: %v", parseErr)
	}
	parseReq.Header.Set("Content-Type", "application/json")
	parseDisconnectResp, parseErr := http.DefaultClient.Do(parseReq)
	if parseErr != nil {
		parseT.Fatalf("post disconnect: %v", parseErr)
	}
	defer parseDisconnectResp.Body.Close()
	if parseDisconnectResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected disconnect 200, got %s", parseDisconnectResp.Status)
	}

	var parseResponse clientDisconnectResponse
	if parseErr3 := json.NewDecoder(parseDisconnectResp.Body).Decode(&parseResponse); parseErr3 != nil {
		parseT.Fatalf("decode disconnect response: %v", parseErr3)
	}
	if parseResponse.Disconnected != 1 || parseResponse.Remaining != 0 || len(parseResponse.DisconnectedIDs) != 1 {
		parseT.Fatalf("unexpected disconnect response: %+v", parseResponse)
	}

	if parseErr4 := parseConn.SetReadDeadline(time.Now().Add(1 * time.Second)); parseErr4 != nil {
		parseT.Fatalf("set read deadline: %v", parseErr4)
	}
	if _, _, parseErr5 := parseConn.ReadMessage(); parseErr5 == nil {
		parseT.Fatal("expected websocket connection to close after disconnect")
	}
}

func TestResolveClientScriptPathReturnsEmptyWithoutConfiguredOverride(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()

	parsePreviousGetwd := livereloadConfigGetwd
	defer func() {
		livereloadConfigGetwd = parsePreviousGetwd
	}()
	livereloadConfigGetwd = func() (string, error) { return parseWorkspaceDir, nil }

	if parseGot := resolveClientScriptPath(); parseGot != "" {
		parseT.Fatalf("expected no default client script override, got %q", parseGot)
	}
}

func TestResolveConfiguredClientScriptPathAndRunnerConfigPath(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	parseConfigPath := filepath.Join(parseWorkspaceDir, "gwc-runner.json")
	parseScriptPath := filepath.Join(parseWorkspaceDir, "vendor", "livereload-client.js")
	if parseErr := os.MkdirAll(filepath.Dir(parseScriptPath), 0o755); parseErr != nil {
		parseT.Fatalf("mkdir script dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseScriptPath, []byte("console.log('ok');"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write script file: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseConfigPath, []byte(`{"paths":{"livereloadClientScript":"vendor/livereload-client.js"}}`), 0o644); parseErr3 != nil {
		parseT.Fatalf("write runner config: %v", parseErr3)
	}

	parsePreviousGetwd := livereloadConfigGetwd
	defer func() {
		livereloadConfigGetwd = parsePreviousGetwd
	}()
	livereloadConfigGetwd = func() (string, error) { return parseWorkspaceDir, nil }

	if parseGot := resolveConfiguredClientScriptPath(); parseGot != parseScriptPath {
		parseT.Fatalf("expected configured client script %q, got %q", parseScriptPath, parseGot)
	}
	if parseGot2 := resolveLivereloadRunnerConfigPath(); parseGot2 != parseConfigPath {
		parseT.Fatalf("expected runner config path %q, got %q", parseConfigPath, parseGot2)
	}

	if parseErr4 := os.Remove(parseScriptPath); parseErr4 != nil {
		parseT.Fatalf("remove script file: %v", parseErr4)
	}
	if parseGot3 := resolveConfiguredClientScriptPath(); parseGot3 != "" {
		parseT.Fatalf("expected missing configured client script to resolve empty, got %q", parseGot3)
	}
}

func TestFirstExistingPathNetAddrAndServedWASMPathHelpers(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseFirst := filepath.Join(parseDir, "missing.js")
	parseSecond := filepath.Join(parseDir, "exists.js")
	if parseErr := os.WriteFile(parseSecond, []byte("ok"), 0o644); parseErr != nil {
		parseT.Fatalf("write second candidate: %v", parseErr)
	}
	if parseGot := firstExistingPath("", parseFirst, parseSecond); parseGot != parseSecond {
		parseT.Fatalf("expected first existing path %q, got %q", parseSecond, parseGot)
	}
	if parseGot2 := firstExistingPath("", parseFirst); parseGot2 != "" {
		parseT.Fatalf("expected empty result when no candidates exist, got %q", parseGot2)
	}

	if parseGot3 := netAddr("", ""); parseGot3 != "127.0.0.1:8080" {
		parseT.Fatalf("expected default net addr, got %q", parseGot3)
	}
	if parseGot4 := netAddr("0.0.0.0", "9010"); parseGot4 != "0.0.0.0:9010" {
		parseT.Fatalf("expected explicit net addr, got %q", parseGot4)
	}

	parseProjectRoot := filepath.Join(parseDir, "project")
	buildDir := filepath.Join(parseDir, "build")
	if parseErr2 := os.MkdirAll(parseProjectRoot, 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir project root: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(buildDir, 0o755); parseErr3 != nil {
		parseT.Fatalf("mkdir build dir: %v", parseErr3)
	}

	parseServer := &LiveReloadServer{projectRoot: parseProjectRoot, buildDir: buildDir}
	if parseGot5 := parseServer.servedWASMPath(); parseGot5 != "/main.wasm" {
		parseT.Fatalf("expected default served wasm path, got %q", parseGot5)
	}

	parseServer.outputPath = "dist/app.wasm"
	if parseGot6 := parseServer.servedWASMPath(); parseGot6 != "/app.wasm" {
		parseT.Fatalf("expected basename fallback for wasm outside project root, got %q", parseGot6)
	}

	parseServer.buildDir = parseProjectRoot
	parseServer.outputPath = filepath.Join(parseProjectRoot, "dist", "app.wasm")
	if parseGot7 := parseServer.servedWASMPath(); parseGot7 != "/dist/app.wasm" {
		parseT.Fatalf("expected project-relative served wasm path, got %q", parseGot7)
	}
}

func TestResolveModuleRootStaticDirAndCleanupHelpers(parseT *testing.T) {
	if parseGot := resolveModuleRoot(""); parseGot != "" {
		parseT.Fatalf("expected empty module root for blank input, got %q", parseGot)
	}
	if parseGot2 := resolveModuleRoot(filepath.Join(parseT.TempDir(), "missing")); parseGot2 != "" {
		parseT.Fatalf("expected empty module root for missing path, got %q", parseGot2)
	}

	parseRoot := parseT.TempDir()
	if parseGot3 := resolveStaticDir(parseRoot); parseGot3 != "" {
		parseT.Fatalf("expected no static dir, got %q", parseGot3)
	}

	parseServer := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{},
	}
	parseServer.cleanup()
}

func TestHandleWebSocketManagedTracksSessionsAndSnapshots(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseServer := &LiveReloadServer{
		projectRoot:     parseProjectRoot,
		watchRoot:       parseProjectRoot,
		buildDir:        parseProjectRoot,
		outputPath:      filepath.Join(parseProjectRoot, "main.wasm"),
		clients:         map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{Success: true, Phase: "serving_output"},
	}
	parseHttpServer := httptest.NewServer(parseServer.newHTTPHandler())
	defer parseHttpServer.Close()

	parseWsURL := "ws" + strings.TrimPrefix(parseHttpServer.URL, "http") + "/ws"
	parseHeaders := http.Header{}
	parseHeaders.Set("User-Agent", "gwc-handle-websocket-test")
	parseClientConn, _, parseErr := websocket.DefaultDialer.Dial(parseWsURL, parseHeaders)
	if parseErr != nil {
		parseT.Fatalf("dial managed websocket: %v", parseErr)
	}
	defer parseClientConn.Close()

	var parseCurrentStatus WebSocketMessage
	if parseErr2 := parseClientConn.ReadJSON(&parseCurrentStatus); parseErr2 != nil {
		parseT.Fatalf("read initial current_status message: %v", parseErr2)
	}
	if parseCurrentStatus.Type != MessageTypeCurrentStatus {
		parseT.Fatalf("expected current_status message, got %+v", parseCurrentStatus)
	}
	if parseCurrentStatus.Protocol != liveReloadProtocol || parseCurrentStatus.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected versioned current_status message, got %+v", parseCurrentStatus)
	}
	if len(parseServer.currentClientSessions()) != 1 {
		parseT.Fatalf("expected one tracked websocket client, got %+v", parseServer.currentClientSessions())
	}

	if parseErr3 := parseClientConn.WriteJSON(WebSocketMessage{
		Type:      MessageTypeStateSnapshot,
		Payload:   "snapshot-1",
		Timestamp: time.Now(),
	}); parseErr3 != nil {
		parseT.Fatalf("write state snapshot message: %v", parseErr3)
	}

	parseDeadline := time.Now().Add(2 * time.Second)
	isParseReceivedSnapshot := false
	for time.Now().Before(parseDeadline) {
		if parseGot := parseServer.takePendingStateSnapshot(); parseGot == "snapshot-1" {
			isParseReceivedSnapshot = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !isParseReceivedSnapshot {
		parseT.Fatal("expected managed websocket to store a pending snapshot payload")
	}

	_ = parseClientConn.Close()
	parseDeadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		if len(parseServer.currentClientSessions()) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	parseT.Fatalf("expected websocket client cleanup after close, got %+v", parseServer.currentClientSessions())
}

func TestHandleWebSocketManagedRejectsFutureProtocolSnapshot(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseServer := &LiveReloadServer{
		projectRoot:     parseProjectRoot,
		watchRoot:       parseProjectRoot,
		buildDir:        parseProjectRoot,
		outputPath:      filepath.Join(parseProjectRoot, "main.wasm"),
		clients:         map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{Success: true, Phase: "serving_output"},
	}
	parseHttpServer := httptest.NewServer(parseServer.newHTTPHandler())
	defer parseHttpServer.Close()

	parseWsURL := "ws" + strings.TrimPrefix(parseHttpServer.URL, "http") + "/ws"
	parseClientConn, _, parseErr := websocket.DefaultDialer.Dial(parseWsURL, nil)
	if parseErr != nil {
		parseT.Fatalf("dial managed websocket: %v", parseErr)
	}
	defer parseClientConn.Close()

	var parseCurrentStatus WebSocketMessage
	if parseErr2 := parseClientConn.ReadJSON(&parseCurrentStatus); parseErr2 != nil {
		parseT.Fatalf("read initial current_status message: %v", parseErr2)
	}

	if parseErr3 := parseClientConn.WriteJSON(WebSocketMessage{
		Protocol:  liveReloadProtocol,
		Version:   currentLiveReloadProtocolVersion + 1,
		Type:      MessageTypeStateSnapshot,
		Payload:   "future-snapshot",
		Timestamp: time.Now(),
	}); parseErr3 != nil {
		parseT.Fatalf("write future state snapshot message: %v", parseErr3)
	}

	time.Sleep(150 * time.Millisecond)
	if parseGot := parseServer.takePendingStateSnapshot(); parseGot != "" {
		parseT.Fatalf("expected future protocol snapshot to be ignored, got %q", parseGot)
	}
}

func TestSendCurrentBuildStatusAndBroadcastMessageDeliverPayloads(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{
			Success:    false,
			ReloadType: "full",
			Phase:      "blocked_on_error",
			Error:      "compile failed",
		},
	}

	parseClientConn, parseCleanup := dialWebsocketHarness(parseT, func(parseConn *websocket.Conn) {
		defer parseConn.Close()
		parseServer.clients[parseConn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		parseServer.sendCurrentBuildStatus(parseConn)
		parseServer.broadcastMessage(MessageTypeReload, map[string]string{"reason": "template"})
		time.Sleep(100 * time.Millisecond)
	})
	defer parseCleanup()

	var parseStatusMsg WebSocketMessage
	if parseErr := parseClientConn.ReadJSON(&parseStatusMsg); parseErr != nil {
		parseT.Fatalf("read current status message: %v", parseErr)
	}
	if parseStatusMsg.Type != MessageTypeCurrentStatus {
		parseT.Fatalf("expected current status message, got %+v", parseStatusMsg)
	}
	if parseStatusMsg.Protocol != liveReloadProtocol || parseStatusMsg.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected versioned current status message, got %+v", parseStatusMsg)
	}

	var parseReloadMsg WebSocketMessage
	if parseErr2 := parseClientConn.ReadJSON(&parseReloadMsg); parseErr2 != nil {
		parseT.Fatalf("read broadcast reload message: %v", parseErr2)
	}
	if parseReloadMsg.Type != MessageTypeReload {
		parseT.Fatalf("expected reload message, got %+v", parseReloadMsg)
	}
	if parseReloadMsg.Protocol != liveReloadProtocol || parseReloadMsg.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected versioned reload message, got %+v", parseReloadMsg)
	}
}

func TestCheckCurrentBuildStateSuccessAndFailure(parseT *testing.T) {
	parseT.Run("success", func(parseT2 *testing.T) {
		buildDir := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte("module example.com/checkstate\n\ngo 1.26.0\n"), 0o644); parseErr != nil {
			parseT2.Fatalf("write go.mod: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(filepath.Join(buildDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); parseErr2 != nil {
			parseT2.Fatalf("write main.go: %v", parseErr2)
		}

		parseServer := &LiveReloadServer{
			buildDir:   buildDir,
			outputPath: filepath.Join(buildDir, "bin", "main.exe"),
		}
		parseClientConn, parseCleanup := dialWebsocketHarness(parseT2, func(parseConn *websocket.Conn) {
			defer parseConn.Close()
			parseServer.checkCurrentBuildState(parseConn)
		})
		defer parseCleanup()

		var parseMsg WebSocketMessage
		if parseErr3 := parseClientConn.ReadJSON(&parseMsg); parseErr3 != nil {
			parseT2.Fatalf("read success build-state message: %v", parseErr3)
		}
		if parseMsg.Type != MessageTypeCurrentStatus {
			parseT2.Fatalf("expected current status message, got %+v", parseMsg)
		}
		if parseServer.lastBuildStatus == nil || !parseServer.lastBuildStatus.Success || parseServer.lastBuildStatus.Phase != "serving_output" {
			parseT2.Fatalf("unexpected success build status: %+v", parseServer.lastBuildStatus)
		}
	})

	parseT.Run("failure", func(parseT3 *testing.T) {
		buildDir := parseT3.TempDir()
		if parseErr4 := os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte("module example.com/checkstatefail\n\ngo 1.26.0\n"), 0o644); parseErr4 != nil {
			parseT3.Fatalf("write go.mod: %v", parseErr4)
		}
		if parseErr5 := os.WriteFile(filepath.Join(buildDir, "main.go"), []byte("package main\nfunc main() { nope }\n"), 0o644); parseErr5 != nil {
			parseT3.Fatalf("write bad main.go: %v", parseErr5)
		}

		parseServer2 := &LiveReloadServer{
			buildDir:   buildDir,
			outputPath: filepath.Join(buildDir, "bin", "main.exe"),
		}
		parseClientConn2, parseCleanup2 := dialWebsocketHarness(parseT3, func(parseConn2 *websocket.Conn) {
			defer parseConn2.Close()
			parseServer2.checkCurrentBuildState(parseConn2)
		})
		defer parseCleanup2()

		var parseMsg2 WebSocketMessage
		if parseErr6 := parseClientConn2.ReadJSON(&parseMsg2); parseErr6 != nil {
			parseT3.Fatalf("read failure build-state message: %v", parseErr6)
		}
		if parseMsg2.Type != MessageTypeCurrentStatus {
			parseT3.Fatalf("expected current status message, got %+v", parseMsg2)
		}
		if parseServer2.lastBuildStatus == nil || parseServer2.lastBuildStatus.Success || parseServer2.lastBuildStatus.Phase != "blocked_on_error" {
			parseT3.Fatalf("unexpected failure build status: %+v", parseServer2.lastBuildStatus)
		}
		if !strings.Contains(parseServer2.lastBuildStatus.Error, "undefined") && !strings.Contains(parseServer2.lastBuildStatus.Error, "nope") {
			parseT3.Fatalf("expected compile failure details, got %+v", parseServer2.lastBuildStatus)
		}
	})
}

func TestHandleHTMLInjectsClientScriptAndWasmConfig(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseIndexPath := filepath.Join(parseProjectRoot, "index.html")
	parseScriptPath := filepath.Join(parseProjectRoot, "scripts", "custom-livereload-client.txt")
	parseOutputPath := filepath.Join(parseProjectRoot, "dist", "main.wasm")
	if parseErr := os.MkdirAll(filepath.Dir(parseScriptPath), 0o755); parseErr != nil {
		parseT.Fatalf("mkdir script dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parseOutputPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir output dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseIndexPath, []byte("<html><body><h1>App</h1></body></html>"), 0o644); parseErr3 != nil {
		parseT.Fatalf("write index fixture: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseScriptPath, []byte("console.log('livereload');"), 0o644); parseErr4 != nil {
		parseT.Fatalf("write client script fixture: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseOutputPath, []byte("wasm"), 0o644); parseErr5 != nil {
		parseT.Fatalf("write wasm fixture: %v", parseErr5)
	}

	parseServer := &LiveReloadServer{
		projectRoot:      parseProjectRoot,
		buildDir:         parseProjectRoot,
		outputPath:       parseOutputPath,
		clientScriptPath: parseScriptPath,
	}

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.handleHTML(parseRecorder, parseReq, parseIndexPath)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected HTML handler success, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
	parseBody := parseRecorder.Body.String()
	if !strings.Contains(parseBody, "window.__GWC_LIVERELOAD_CONFIG") ||
		!strings.Contains(parseBody, `wasmPath: "/dist/main.wasm"`) ||
		!strings.Contains(parseBody, `projectRoot: `+strconv.Quote(filepath.ToSlash(parseProjectRoot))) {
		parseT.Fatalf("expected HTML injection to include wasm config, got %q", parseBody)
	}
	if !strings.Contains(parseBody, "console.log('livereload');") {
		parseT.Fatalf("expected HTML injection to include livereload client script, got %q", parseBody)
	}
}

func TestHandleHTMLInjectsEmbeddedClientScriptByDefault(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseIndexPath := filepath.Join(parseProjectRoot, "index.html")
	parseOutputPath := filepath.Join(parseProjectRoot, "dist", "main.wasm")
	if parseErr := os.MkdirAll(filepath.Dir(parseOutputPath), 0o755); parseErr != nil {
		parseT.Fatalf("mkdir output dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseIndexPath, []byte("<html><body><h1>App</h1></body></html>"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write index fixture: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseOutputPath, []byte("wasm"), 0o644); parseErr3 != nil {
		parseT.Fatalf("write wasm fixture: %v", parseErr3)
	}

	parseServer := &LiveReloadServer{
		projectRoot: parseProjectRoot,
		buildDir:    parseProjectRoot,
		outputPath:  parseOutputPath,
	}

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.handleHTML(parseRecorder, parseReq, parseIndexPath)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected HTML handler success, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
	parseBody := parseRecorder.Body.String()
	if !strings.Contains(parseBody, "Live Reload Client Script") {
		parseT.Fatalf("expected HTML injection to include embedded livereload client script, got %q", parseBody)
	}
	if !strings.Contains(parseBody, liveReloadProtocol) {
		parseT.Fatalf("expected embedded livereload client script to include protocol metadata, got %q", parseBody)
	}
	for _, parseExpected := range []string{
		"installRuntimePanicConsoleBridge",
		"gwc-runtime-error-overlay",
		"gwc:runtime-panic",
	} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected embedded client script to include runtime overlay bridge %q, got %q", parseExpected, parseBody)
		}
	}
}

func TestHandleHTMLMissingClientScriptReturnsServerError(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseIndexPath := filepath.Join(parseProjectRoot, "index.html")
	if parseErr := os.WriteFile(parseIndexPath, []byte("<html><body>App</body></html>"), 0o644); parseErr != nil {
		parseT.Fatalf("write index fixture: %v", parseErr)
	}

	parseServer := &LiveReloadServer{
		projectRoot:      parseProjectRoot,
		buildDir:         parseProjectRoot,
		outputPath:       filepath.Join(parseProjectRoot, "main.wasm"),
		clientScriptPath: filepath.Join(parseProjectRoot, "missing-client.js"),
	}

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRecorder := httptest.NewRecorder()
	parseServer.handleHTML(parseRecorder, parseReq, parseIndexPath)

	if parseRecorder.Code != http.StatusInternalServerError {
		parseT.Fatalf("expected server error for missing client script, got %d with body %q", parseRecorder.Code, parseRecorder.Body.String())
	}
}

func TestRequestStateSnapshotBroadcastsStateExport(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{},
	}

	parseClientConn, parseCleanup := dialWebsocketHarness(parseT, func(parseConn *websocket.Conn) {
		defer parseConn.Close()
		parseServer.clients[parseConn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		parseServer.requestStateSnapshot()
		time.Sleep(100 * time.Millisecond)
	})
	defer parseCleanup()

	var parseMsg WebSocketMessage
	if parseErr := parseClientConn.ReadJSON(&parseMsg); parseErr != nil {
		parseT.Fatalf("read state export message: %v", parseErr)
	}
	if parseMsg.Type != MessageTypeStateExport {
		parseT.Fatalf("expected state export message, got %+v", parseMsg)
	}
	if parseMsg.Protocol != liveReloadProtocol || parseMsg.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected versioned state export message, got %+v", parseMsg)
	}
}

func TestHandleWebSocketTracksSessionsAndSnapshots(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		clients:         map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{Success: true, Phase: "serving_output"},
	}
	parseHttpServer := httptest.NewServer(http.HandlerFunc(parseServer.handleWebSocket))
	defer parseHttpServer.Close()

	parseWsURL := "ws" + strings.TrimPrefix(parseHttpServer.URL, "http")
	parseHeaders := http.Header{}
	parseHeaders.Set("User-Agent", "gwc-legacy-websocket-test")
	parseClientConn, _, parseErr := websocket.DefaultDialer.Dial(parseWsURL, parseHeaders)
	if parseErr != nil {
		parseT.Fatalf("dial legacy websocket: %v", parseErr)
	}
	defer parseClientConn.Close()

	var parseCurrentStatus WebSocketMessage
	if parseErr2 := parseClientConn.ReadJSON(&parseCurrentStatus); parseErr2 != nil {
		parseT.Fatalf("read initial current status from legacy websocket: %v", parseErr2)
	}
	if parseCurrentStatus.Type != MessageTypeCurrentStatus {
		parseT.Fatalf("expected current status message, got %+v", parseCurrentStatus)
	}
	if parseCurrentStatus.Protocol != liveReloadProtocol || parseCurrentStatus.Version != currentLiveReloadProtocolVersion {
		parseT.Fatalf("expected versioned current status message, got %+v", parseCurrentStatus)
	}
	if len(parseServer.currentClientSessions()) != 1 {
		parseT.Fatalf("expected one tracked legacy websocket client, got %+v", parseServer.currentClientSessions())
	}

	if parseErr3 := parseClientConn.WriteJSON(WebSocketMessage{
		Type:      MessageTypeStateSnapshot,
		Payload:   "legacy-snapshot",
		Timestamp: time.Now(),
	}); parseErr3 != nil {
		parseT.Fatalf("write legacy snapshot message: %v", parseErr3)
	}

	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		if parseGot := parseServer.takePendingStateSnapshot(); parseGot == "legacy-snapshot" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if parseGot2 := parseServer.takePendingStateSnapshot(); parseGot2 != "" {
		parseT.Fatalf("expected pending snapshot buffer to be drained, got %q", parseGot2)
	}

	_ = parseClientConn.Close()
	parseDeadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		if len(parseServer.currentClientSessions()) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	parseT.Fatalf("expected legacy websocket client cleanup after close, got %+v", parseServer.currentClientSessions())
}

func TestNewLiveReloadServerWithOptionsResolvesRelativePathsAndDefaults(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseWorkspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	parseProjectRoot := filepath.Join(parseWorkspaceDir, "app")
	parseIndexRel := filepath.Join("custom", "index.html")
	parseScriptRel := filepath.Join("scripts", "client.js")
	if parseErr2 := os.MkdirAll(filepath.Join(parseProjectRoot, "custom"), 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir custom dir: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Join(parseProjectRoot, "scripts"), 0o755); parseErr3 != nil {
		parseT.Fatalf("mkdir script dir: %v", parseErr3)
	}
	if parseErr4 := os.MkdirAll(filepath.Join(parseProjectRoot, "static"), 0o755); parseErr4 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseProjectRoot, parseIndexRel), []byte("<html></html>"), 0o644); parseErr5 != nil {
		parseT.Fatalf("write custom index: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parseProjectRoot, parseScriptRel), []byte("console.log('client');"), 0o644); parseErr6 != nil {
		parseT.Fatalf("write client script: %v", parseErr6)
	}

	parseServer, parseErr7 := NewLiveReloadServerWithOptions(LiveReloadOptions{
		ProjectRoot:      parseProjectRoot,
		IndexPath:        parseIndexRel,
		OutputPath:       filepath.Join("dist", "app.wasm"),
		ClientScriptPath: parseScriptRel,
		AlwaysHotReload:  true,
	})
	if parseErr7 != nil {
		parseT.Fatalf("new live reload server: %v", parseErr7)
	}
	defer parseServer.cleanup()

	if parseServer.projectRoot != parseProjectRoot {
		parseT.Fatalf("expected project root %q, got %q", parseProjectRoot, parseServer.projectRoot)
	}
	if parseServer.buildDir != parseProjectRoot {
		parseT.Fatalf("expected build dir %q, got %q", parseProjectRoot, parseServer.buildDir)
	}
	if parseServer.watchRoot != parseWorkspaceDir {
		parseT.Fatalf("expected watch root %q, got %q", parseWorkspaceDir, parseServer.watchRoot)
	}
	if parseServer.indexPath != filepath.Join(parseProjectRoot, parseIndexRel) {
		parseT.Fatalf("expected resolved index path, got %q", parseServer.indexPath)
	}
	if parseServer.outputPath != filepath.Join(parseProjectRoot, "dist", "app.wasm") {
		parseT.Fatalf("expected resolved output path, got %q", parseServer.outputPath)
	}
	if parseServer.clientScriptPath != filepath.Join(parseProjectRoot, parseScriptRel) {
		parseT.Fatalf("expected resolved client script path, got %q", parseServer.clientScriptPath)
	}
	if parseServer.host != defaultHost || parseServer.port != defaultPort {
		parseT.Fatalf("expected default host/port %s:%s, got %s:%s", defaultHost, defaultPort, parseServer.host, parseServer.port)
	}
	if !parseServer.alwaysHotReload {
		parseT.Fatal("expected always hot reload flag to persist")
	}
	if parseServer.staticDir != filepath.Join(parseProjectRoot, "static") {
		parseT.Fatalf("expected project static dir, got %q", parseServer.staticDir)
	}
	if parseServer.manifestPath != filepath.Join(parseProjectRoot, "dist", "app.wasm.hotreload-manifest.json") {
		parseT.Fatalf("unexpected manifest path %q", parseServer.manifestPath)
	}
}

func TestNewLiveReloadServerWithOptionsFallsBackToStaticIndexAndWorkspaceBuildRoot(parseT *testing.T) {
	parseWorkspaceDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseWorkspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseWorkspaceDir, "gwc-runner.json"), []byte(`{"paths":{"workspaceBuildRoot":"out"}}`), 0o644); parseErr2 != nil {
		parseT.Fatalf("write runner config: %v", parseErr2)
	}
	parseProjectRoot := filepath.Join(parseWorkspaceDir, "examples", "demo")
	parseStaticIndex := filepath.Join(parseProjectRoot, "static", "index.html")
	if parseErr3 := os.MkdirAll(filepath.Dir(parseStaticIndex), 0o755); parseErr3 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseStaticIndex, []byte("<html>static</html>"), 0o644); parseErr4 != nil {
		parseT.Fatalf("write static index: %v", parseErr4)
	}

	parseServer, parseErr5 := NewLiveReloadServerWithOptions(LiveReloadOptions{
		ProjectRoot: parseProjectRoot,
	})
	if parseErr5 != nil {
		parseT.Fatalf("new live reload server: %v", parseErr5)
	}
	defer parseServer.cleanup()

	if parseServer.indexPath != parseStaticIndex {
		parseT.Fatalf("expected static index fallback %q, got %q", parseStaticIndex, parseServer.indexPath)
	}
	if parseServer.outputPath != filepath.Join(parseWorkspaceDir, "out", "examples", "demo", "main.wasm") {
		parseT.Fatalf("expected workspace build root output path, got %q", parseServer.outputPath)
	}
}

func TestCurrentStatusReturnsCopiesAndCurrentErrorForFailedBuild(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseConn := &websocket.Conn{}
	parseConnectedAt := time.Now().UTC().Add(-time.Minute)
	parseServer := &LiveReloadServer{
		projectRoot:        parseProjectRoot,
		watchRoot:          parseProjectRoot,
		buildDir:           parseProjectRoot,
		outputPath:         filepath.Join(parseProjectRoot, "main.wasm"),
		host:               "127.0.0.1",
		port:               "9090",
		alwaysHotReload:    false,
		lastClassification: UpdateClassification{ReloadType: "hot", Reason: "ui change"},
		lastBuildStatus: &BuildStatus{
			Success: false,
			Error:   "compile failed",
			Phase:   "blocked_on_error",
		},
		clients: map[*websocket.Conn]ClientSession{
			parseConn: {ID: "client-1", ConnectedAt: parseConnectedAt, LastSeenAt: parseConnectedAt},
		},
	}

	parseStatus := parseServer.currentStatus()
	if parseStatus.Mode != "livereload-wasm" || parseStatus.ListeningURL != "http://127.0.0.1:9090" {
		parseT.Fatalf("unexpected status envelope: %+v", parseStatus)
	}
	if !parseStatus.HotReloadEligible || parseStatus.HotReloadEnabled {
		parseT.Fatalf("unexpected hot reload flags: %+v", parseStatus)
	}
	if parseStatus.CurrentError == nil || parseStatus.CurrentError.Error != "compile failed" {
		parseT.Fatalf("expected current error copy for failed build, got %+v", parseStatus.CurrentError)
	}
	if parseStatus.LastBuild == nil || parseStatus.LastBuild.Error != "compile failed" {
		parseT.Fatalf("expected last build copy, got %+v", parseStatus.LastBuild)
	}
	if parseStatus.ClientCount != 1 || len(parseStatus.Clients) != 1 {
		parseT.Fatalf("expected client snapshot in status, got %+v", parseStatus)
	}

	parseServer.lastBuildStatus.Error = "mutated"
	parseServer.clients[parseConn] = ClientSession{ID: "client-2", ConnectedAt: parseConnectedAt, LastSeenAt: parseConnectedAt}
	if parseStatus.LastBuild.Error != "compile failed" || parseStatus.CurrentError.Error != "compile failed" || parseStatus.Clients[0].ID != "client-1" {
		parseT.Fatalf("expected status snapshot copies to be isolated, got %+v", parseStatus)
	}
}

func TestAddWatchersAddsDirectoriesAndSkipsIgnoredRoots(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseWatcher, parseErr := fsnotify.NewWatcher()
	if parseErr != nil {
		parseT.Fatalf("new watcher: %v", parseErr)
	}
	defer parseWatcher.Close()

	parseDirs := []string{
		filepath.Join(parseRoot, "app"),
		filepath.Join(parseRoot, "app", "nested"),
		filepath.Join(parseRoot, ".git"),
		filepath.Join(parseRoot, "vendor"),
		filepath.Join(parseRoot, "node_modules"),
		filepath.Join(parseRoot, "bin"),
	}
	for _, parseDir := range parseDirs {
		if parseErr2 := os.MkdirAll(parseDir, 0o755); parseErr2 != nil {
			parseT.Fatalf("mkdir %q: %v", parseDir, parseErr2)
		}
	}

	parseServer := &LiveReloadServer{watcher: parseWatcher}
	if parseErr3 := parseServer.addWatchers(parseRoot); parseErr3 != nil {
		parseT.Fatalf("add watchers: %v", parseErr3)
	}

	parseWatched := parseWatcher.WatchList()
	if !slices.Contains(parseWatched, parseRoot) || !slices.Contains(parseWatched, filepath.Join(parseRoot, "app")) || !slices.Contains(parseWatched, filepath.Join(parseRoot, "app", "nested")) {
		parseT.Fatalf("expected real directories to be watched, got %+v", parseWatched)
	}
	if slices.Contains(parseWatched, filepath.Join(parseRoot, ".git")) || slices.Contains(parseWatched, filepath.Join(parseRoot, "vendor")) || slices.Contains(parseWatched, filepath.Join(parseRoot, "node_modules")) || slices.Contains(parseWatched, filepath.Join(parseRoot, "bin")) {
		parseT.Fatalf("expected ignored directories to be skipped, got %+v", parseWatched)
	}
}

func TestAddWatchersReturnsWalkErrorForMissingRoot(parseT *testing.T) {
	parseWatcher, parseErr := fsnotify.NewWatcher()
	if parseErr != nil {
		parseT.Fatalf("new watcher: %v", parseErr)
	}
	defer parseWatcher.Close()

	parseServer := &LiveReloadServer{watcher: parseWatcher}
	if parseErr2 := parseServer.addWatchers(filepath.Join(parseT.TempDir(), "missing")); parseErr2 == nil {
		parseT.Fatal("expected missing root to return a walk error")
	}
}

func TestHandleFileEventTracksGoChangesAndSkipsIgnoredFiles(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		clients:      map[*websocket.Conn]ClientSession{},
		changedFiles: make(map[string]time.Time),
	}

	parseServer.handleFileEvent(fsnotify.Event{Name: "styles.css", Op: fsnotify.Write})
	parseServer.handleFileEvent(fsnotify.Event{Name: "main.go.tmp", Op: fsnotify.Write})
	parseServer.handleFileEvent(fsnotify.Event{Name: "main.go~", Op: fsnotify.Write})
	if len(parseServer.changedFiles) != 0 {
		parseT.Fatalf("expected ignored files not to be tracked, got %+v", parseServer.changedFiles)
	}

	parseServer.handleFileEvent(fsnotify.Event{Name: "main.go", Op: fsnotify.Write})
	if len(parseServer.changedFiles) != 1 {
		parseT.Fatalf("expected go write to be tracked, got %+v", parseServer.changedFiles)
	}
	if _, parseOk := parseServer.changedFiles["main.go"]; !parseOk {
		parseT.Fatalf("expected main.go change to be tracked, got %+v", parseServer.changedFiles)
	}
	if parseServer.changeCount != 1 || parseServer.debounceTimer == nil || parseServer.maxDebounceTimer == nil {
		parseT.Fatalf("expected debounce state to initialize, got changeCount=%d debounce=%v maxDebounce=%v", parseServer.changeCount, parseServer.debounceTimer, parseServer.maxDebounceTimer)
	}
	parseServer.resetDebounceState()
	if parseServer.debounceTimer != nil {
		parseServer.debounceTimer.Stop()
		parseServer.debounceTimer = nil
	}
}

func TestClassifyUpdateBranches(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()

	parseT.Run("always hot reload", func(parseT2 *testing.T) {
		parseServer := &LiveReloadServer{
			projectRoot:     parseProjectRoot,
			alwaysHotReload: true,
			changedFiles:    map[string]time.Time{filepath.Join(parseProjectRoot, "pkg", "thing.go"): time.Now()},
		}
		parseClassification := parseServer.classifyUpdate()
		if parseClassification.ReloadType != "hot" || !strings.Contains(parseClassification.Reason, "Hot reload") {
			parseT2.Fatalf("expected always-hot classification, got %+v", parseClassification)
		}
		if len(parseServer.changedFiles) != 0 {
			parseT2.Fatalf("expected changed files to clear after classification, got %+v", parseServer.changedFiles)
		}
	})

	parseT.Run("main go triggers full reload", func(parseT3 *testing.T) {
		parseServer2 := &LiveReloadServer{
			projectRoot:  parseProjectRoot,
			changedFiles: map[string]time.Time{filepath.Join(parseProjectRoot, "main.go"): time.Now()},
		}
		parseClassification2 := parseServer2.classifyUpdate()
		if parseClassification2.ReloadType != "full" || !strings.Contains(parseClassification2.Reason, "entry point") {
			parseT3.Fatalf("expected entry-point full reload, got %+v", parseClassification2)
		}
	})

	parseT.Run("ui directories dedupe to hot reload", func(parseT4 *testing.T) {
		parseServer3 := &LiveReloadServer{
			projectRoot: parseProjectRoot,
			changedFiles: map[string]time.Time{
				filepath.Join(parseProjectRoot, "examples", "a.go"): time.Now(),
				filepath.Join(parseProjectRoot, "website", "b.go"):  time.Now(),
				filepath.Join(parseProjectRoot, "examples", "c.go"): time.Now(),
			},
		}
		parseClassification3 := parseServer3.classifyUpdate()
		if parseClassification3.ReloadType != "hot" || !strings.Contains(parseClassification3.Reason, "example components") || !strings.Contains(parseClassification3.Reason, "website components") {
			parseT4.Fatalf("expected ui-directory hot reload, got %+v", parseClassification3)
		}
	})

	parseT.Run("app logic changes default to hot reload", func(parseT5 *testing.T) {
		parseServer4 := &LiveReloadServer{
			projectRoot:  parseProjectRoot,
			changedFiles: map[string]time.Time{filepath.Join(parseProjectRoot, "pkg", "logic.go"): time.Now()},
		}
		parseClassification4 := parseServer4.classifyUpdate()
		if parseClassification4.ReloadType != "hot" || !strings.Contains(parseClassification4.Reason, "app components") {
			parseT5.Fatalf("expected app-logic hot reload (state snapshot + selective remount handle safety), got %+v", parseClassification4)
		}
	})

	parseT.Run("core runtime changes force full reload", func(parseT6 *testing.T) {
		parseServer5 := &LiveReloadServer{
			projectRoot:  parseProjectRoot,
			changedFiles: map[string]time.Time{filepath.Join(parseProjectRoot, "internal", "runtime", "reconciler.go"): time.Now()},
		}
		parseClassification5 := parseServer5.classifyUpdate()
		if parseClassification5.ReloadType != "full" || !strings.Contains(parseClassification5.Reason, "Core runtime") {
			parseT6.Fatalf("expected core-runtime full reload, got %+v", parseClassification5)
		}
	})
}

func TestHandleFileEventAssetAndArtifactBranches(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()

	parseT.Run("build artifact paths are ignored", func(parseT2 *testing.T) {
		parseServer := &LiveReloadServer{projectRoot: parseProjectRoot, outputPath: filepath.Join(parseProjectRoot, "bin", "main.wasm")}
		if !parseServer.isBuildArtifactPath(filepath.Join(parseProjectRoot, "bin", "styles.css")) {
			parseT2.Fatal("expected bin/ css to be classified as build artifact")
		}
		if parseServer.isBuildArtifactPath(filepath.Join(parseProjectRoot, "assets", "styles.css")) {
			parseT2.Fatal("expected source css to not be classified as build artifact")
		}
	})

	parseT.Run("css change queues asset swap without build", func(parseT3 *testing.T) {
		parseServer2 := &LiveReloadServer{
			projectRoot:  parseProjectRoot,
			changedFiles: map[string]time.Time{},
			clients:      map[*websocket.Conn]ClientSession{},
		}
		parseServer2.queueAssetSwap(filepath.Join(parseProjectRoot, "assets", "styles.css"))
		parseServer2.mutex.Lock()
		parsePending := len(parseServer2.changedAssets)
		parseTimerSet := parseServer2.assetDebounceTimer != nil
		parseServer2.mutex.Unlock()
		if parsePending != 1 || !parseTimerSet {
			parseT3.Fatalf("expected one pending asset and an armed debounce timer, got pending=%d timer=%v", parsePending, parseTimerSet)
		}
		// The asset path must never enter the Go-build changed-files set.
		if len(parseServer2.changedFiles) != 0 {
			parseT3.Fatalf("expected css change to bypass the build pipeline, got %+v", parseServer2.changedFiles)
		}
	})
}

func TestComponentSignatureHelpers(parseT *testing.T) {
	if isComponentValueSpec(nil, 0) {
		parseT.Fatal("nil value spec should not be treated as a component")
	}

	parseComponentType := &ast.FuncType{
		Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "ui"},
			Sel: &ast.Ident{Name: "Node"},
		}}}},
	}
	parseNonComponentType := &ast.FuncType{
		Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "error"}}}},
	}

	if !returnsComponentNode(parseComponentType) {
		parseT.Fatal("expected ui.Node function type to be treated as a component")
	}
	if returnsComponentNode(parseNonComponentType) {
		parseT.Fatal("expected non-component function type not to be treated as a component")
	}

	parseTypedSpec := &ast.ValueSpec{Type: parseComponentType}
	if !isComponentValueSpec(parseTypedSpec, 0) {
		parseT.Fatal("expected typed function value spec to be treated as a component")
	}

	parseLiteralSpec := &ast.ValueSpec{Values: []ast.Expr{
		&ast.FuncLit{Type: parseComponentType},
		&ast.FuncLit{Type: parseNonComponentType},
	}}
	if !isComponentValueSpec(parseLiteralSpec, 0) {
		parseT.Fatal("expected component func literal to be treated as a component")
	}
	if isComponentValueSpec(parseLiteralSpec, 1) {
		parseT.Fatal("expected non-component func literal not to be treated as a component")
	}
	if isComponentValueSpec(parseLiteralSpec, 2) {
		parseT.Fatal("expected out-of-range value spec index to be false")
	}
}

func TestComponentResultExpressionHelpers(parseT *testing.T) {
	parseCases := []struct {
		name string
		expr ast.Expr
		want bool
	}{
		{name: "node ident", expr: &ast.Ident{Name: "Node"}, want: true},
		{name: "element ident", expr: &ast.Ident{Name: "Element"}, want: true},
		{name: "ui node", expr: &ast.SelectorExpr{X: &ast.Ident{Name: "ui"}, Sel: &ast.Ident{Name: "Node"}}, want: true},
		{name: "ui element", expr: &ast.SelectorExpr{X: &ast.Ident{Name: "ui"}, Sel: &ast.Ident{Name: "Element"}}, want: true},
		{name: "runtime element", expr: &ast.SelectorExpr{X: &ast.Ident{Name: "runtime"}, Sel: &ast.Ident{Name: "Element"}}, want: true},
		{name: "pointer runtime element", expr: &ast.StarExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "runtime"}, Sel: &ast.Ident{Name: "Element"}}}, want: true},
		{name: "pointer ui element", expr: &ast.StarExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "ui"}, Sel: &ast.Ident{Name: "Element"}}}, want: true},
		{name: "pointer ident element", expr: &ast.StarExpr{X: &ast.Ident{Name: "Element"}}, want: true},
		{name: "other selector", expr: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Node"}}, want: false},
		{name: "string ident", expr: &ast.Ident{Name: "string"}, want: false},
	}

	for _, parseTc := range parseCases {
		if parseGot := isComponentResultExpr(parseTc.expr); parseGot != parseTc.want {
			parseT.Fatalf("%s: isComponentResultExpr() = %v, want %v", parseTc.name, parseGot, parseTc.want)
		}
	}

	if !isRuntimeElementExpr(&ast.Ident{Name: "Element"}) {
		parseT.Fatal("expected ident Element to be runtime element")
	}
	if isRuntimeElementExpr(&ast.Ident{Name: "Node"}) {
		parseT.Fatal("expected ident Node not to be runtime element")
	}
	if !isRuntimeElementExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "ui"}, Sel: &ast.Ident{Name: "Element"}}) {
		parseT.Fatal("expected ui.Element to be runtime element")
	}
	if isRuntimeElementExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Element"}}) {
		parseT.Fatal("expected pkg.Element not to be runtime element")
	}
}

func TestReturnsComponentNodeCornerCases(parseT *testing.T) {
	if returnsComponentNode(nil) {
		parseT.Fatal("nil func type should not return a component node")
	}
	if returnsComponentNode(&ast.FuncType{}) {
		parseT.Fatal("func type without results should not return a component node")
	}
	if returnsComponentNode(&ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{
		{Type: &ast.Ident{Name: "Node"}},
		{Type: &ast.Ident{Name: "error"}},
	}}}) {
		parseT.Fatal("multi-result func type should not be treated as a component")
	}
}

func TestPackageAndModulePathHelpers(parseT *testing.T) {
	if parseGot := qualifyComponentName("", "App"); parseGot != "App" {
		parseT.Fatalf("expected unqualified component name, got %q", parseGot)
	}
	if parseGot2 := qualifyComponentName("example.com/test", "App"); parseGot2 != "example.com/test.App" {
		parseT.Fatalf("expected qualified component name, got %q", parseGot2)
	}

	parseWatchRoot := filepath.Join("workspace", "repo")
	parseFilePath := filepath.Join(parseWatchRoot, "examples", "demo", "main.go")
	if parseGot3 := resolvePackagePath("example.com/test", parseWatchRoot, parseFilePath); parseGot3 != "example.com/test/examples/demo" {
		parseT.Fatalf("expected resolved package path, got %q", parseGot3)
	}
	if parseGot4 := resolvePackagePath("", parseWatchRoot, parseFilePath); parseGot4 != "examples/demo" {
		parseT.Fatalf("expected relative package path without module, got %q", parseGot4)
	}
	if parseGot5 := resolvePackagePath("example.com/test", parseWatchRoot, filepath.Join(parseWatchRoot, "main.go")); parseGot5 != "example.com/test" {
		parseT.Fatalf("expected module root package path, got %q", parseGot5)
	}

	parseRoot := parseT.TempDir()
	parseGoModPath := filepath.Join(parseRoot, "go.mod")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/test\n\ngo 1.26.0\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseGot6 := resolveModulePath(parseRoot); parseGot6 != "example.com/test" {
		parseT.Fatalf("expected module path from go.mod, got %q", parseGot6)
	}
	if parseErr2 := os.WriteFile(parseGoModPath, []byte("go 1.26.0\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("rewrite go.mod without module: %v", parseErr2)
	}
	if parseGot7 := resolveModulePath(parseRoot); parseGot7 != "" {
		parseT.Fatalf("expected empty module path without module directive, got %q", parseGot7)
	}
}

func TestWriteChangedComponentManifestNoopCases(parseT *testing.T) {
	parseServer := &LiveReloadServer{}
	if parseErr := parseServer.writeChangedComponentManifest(nil); parseErr != nil {
		parseT.Fatalf("expected nil manifest to be ignored, got %v", parseErr)
	}

	parseServer.manifestPath = ""
	if parseErr2 := parseServer.writeChangedComponentManifest(&ChangedComponentManifest{}); parseErr2 != nil {
		parseT.Fatalf("expected empty manifest path to be ignored, got %v", parseErr2)
	}
}

func TestWriteChangedComponentManifestDirectoryError(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBlockingFile := filepath.Join(parseRoot, "blocking-file")
	if parseErr := os.WriteFile(parseBlockingFile, []byte("block"), 0o644); parseErr != nil {
		parseT.Fatalf("write blocking file: %v", parseErr)
	}

	parseServer := &LiveReloadServer{
		manifestPath: filepath.Join(parseBlockingFile, "manifest.json"),
	}
	parseErr2 := parseServer.writeChangedComponentManifest(&ChangedComponentManifest{ReloadType: "hot"})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "create manifest dir") {
		parseT.Fatalf("expected manifest directory creation error, got %v", parseErr2)
	}
}

func TestNewLiveReloadServerWrapperUsesProjectRoot(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseIndexPath := filepath.Join(parseProjectRoot, "static", "index.html")
	if parseErr := os.MkdirAll(filepath.Dir(parseIndexPath), 0o755); parseErr != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseIndexPath, []byte("<html></html>"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write static index: %v", parseErr2)
	}

	parseServer, parseErr3 := NewLiveReloadServer(parseProjectRoot)
	if parseErr3 != nil {
		parseT.Fatalf("new live reload server: %v", parseErr3)
	}
	defer parseServer.cleanup()

	if parseServer.projectRoot != parseProjectRoot {
		parseT.Fatalf("expected wrapper project root %q, got %q", parseProjectRoot, parseServer.projectRoot)
	}
	if parseServer.buildDir != parseProjectRoot {
		parseT.Fatalf("expected wrapper build dir %q, got %q", parseProjectRoot, parseServer.buildDir)
	}
	if parseServer.indexPath != parseIndexPath {
		parseT.Fatalf("expected wrapper to fall back to static index %q, got %q", parseIndexPath, parseServer.indexPath)
	}
}

func TestResolveBuildDirMissingPathReturnsError(parseT *testing.T) {
	if _, parseErr := resolveBuildDir(filepath.Join(parseT.TempDir(), "missing", "main.go")); parseErr == nil {
		parseT.Fatal("expected missing build entry path to return an error")
	}
}

func TestResolveModuleRootFromFileAndServedWASMDefaultCases(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseGoModPath := filepath.Join(parseRoot, "go.mod")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/test\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	parseMainPath := filepath.Join(parseRoot, "cmd", "demo", "main.go")
	if parseErr2 := os.MkdirAll(filepath.Dir(parseMainPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir main dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseMainPath, []byte("package main\n"), 0o644); parseErr3 != nil {
		parseT.Fatalf("write main.go: %v", parseErr3)
	}

	if parseGot := resolveModuleRoot(parseMainPath); parseGot != parseRoot {
		parseT.Fatalf("expected file-based module root %q, got %q", parseRoot, parseGot)
	}
	if parseGot2 := (*LiveReloadServer)(nil).servedWASMPath(); parseGot2 != "/main.wasm" {
		parseT.Fatalf("expected nil server to serve default wasm path, got %q", parseGot2)
	}

	parseServer := &LiveReloadServer{projectRoot: parseRoot, buildDir: parseRoot}
	if parseGot3 := parseServer.servedWASMPath(); parseGot3 != "/main.wasm" {
		parseT.Fatalf("expected empty output path to serve default wasm path, got %q", parseGot3)
	}
}

func TestResolveClientScriptPathAndRunnerConfigPathHandleGetwdFailure(parseT *testing.T) {
	parsePreviousGetwd := livereloadConfigGetwd
	defer func() {
		livereloadConfigGetwd = parsePreviousGetwd
	}()
	livereloadConfigGetwd = func() (string, error) { return "", errors.New("boom") }
	if parseGot := resolveClientScriptPath(); parseGot != "" {
		parseT.Fatalf("expected empty client script path on getwd failure, got %q", parseGot)
	}
	if parseGot2 := resolveConfiguredClientScriptPath(); parseGot2 != "" {
		parseT.Fatalf("expected empty configured client script path on getwd failure, got %q", parseGot2)
	}
	if parseGot3 := resolveLivereloadRunnerConfigPath(); parseGot3 != "" {
		parseT.Fatalf("expected empty runner config path on getwd failure, got %q", parseGot3)
	}
}

func TestBroadcastMessageHandlesMarshalAndWriteErrors(parseT *testing.T) {
	parseT.Run("marshal error", func(parseT2 *testing.T) {
		parseServer := &LiveReloadServer{clients: map[*websocket.Conn]ClientSession{}}
		parseServer.broadcastMessage(MessageTypeReload, map[string]any{"bad": make(chan int)})
	})

	parseT.Run("write error", func(parseT3 *testing.T) {
		parseServer2 := &LiveReloadServer{
			clients: map[*websocket.Conn]ClientSession{},
		}
		parseClientConn, parseCleanup := dialWebsocketHarness(parseT3, func(parseConn *websocket.Conn) {
			parseServer2.clients[parseConn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
			_ = parseConn.Close()
			parseServer2.broadcastMessage(MessageTypeReload, map[string]string{"reason": "closed"})
		})
		defer parseCleanup()
		_ = parseClientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _, _ = parseClientConn.ReadMessage()
	})
}

func TestSendCurrentBuildStatusHandlesClosedConnection(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		lastBuildStatus: &BuildStatus{
			Success: false,
			Error:   "compile failed",
		},
	}
	parseClientConn, parseCleanup := dialWebsocketHarness(parseT, func(parseConn *websocket.Conn) {
		_ = parseConn.Close()
		parseServer.sendCurrentBuildStatus(parseConn)
	})
	defer parseCleanup()
	_ = parseClientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, _ = parseClientConn.ReadMessage()
}

func TestCleanupClosesWatcherAndWebsocketClients(parseT *testing.T) {
	parseWatcher, parseErr := fsnotify.NewWatcher()
	if parseErr != nil {
		parseT.Fatalf("new watcher: %v", parseErr)
	}
	parseServer := &LiveReloadServer{
		watcher:          parseWatcher,
		httpServer:       &http.Server{},
		clients:          map[*websocket.Conn]ClientSession{},
		debounceTimer:    time.NewTimer(time.Minute),
		maxDebounceTimer: time.NewTimer(time.Minute),
	}
	parseClientConn, parseCleanup := dialWebsocketHarness(parseT, func(parseConn *websocket.Conn) {
		parseServer.clients[parseConn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		parseServer.cleanup()
	})
	defer parseCleanup()

	if parseErr2 := parseWatcher.Add(parseT.TempDir()); parseErr2 == nil {
		parseT.Fatal("expected closed watcher to reject new watches after cleanup")
	}
	if parseErr3 := parseClientConn.SetReadDeadline(time.Now().Add(time.Second)); parseErr3 != nil {
		parseT.Fatalf("set read deadline: %v", parseErr3)
	}
	if _, _, parseErr4 := parseClientConn.ReadMessage(); parseErr4 == nil {
		parseT.Fatal("expected websocket client to close during cleanup")
	}
}

func TestDebounceAndBuildUsesQuickDebounceForLaterChanges(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		clients:         map[*websocket.Conn]ClientSession{},
		firstChangeTime: time.Now().Add(-2 * time.Second),
		changeCount:     3,
	}
	parseClientConn, parseCleanup := dialWebsocketHarness(parseT, func(parseConn *websocket.Conn) {
		parseServer.clients[parseConn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		parseServer.debounceAndBuild()
	})
	defer parseCleanup()
	defer func() {
		if parseServer.debounceTimer != nil {
			parseServer.debounceTimer.Stop()
		}
		if parseServer.maxDebounceTimer != nil {
			parseServer.maxDebounceTimer.Stop()
		}
	}()

	var parseMsg WebSocketMessage
	if parseErr := parseClientConn.ReadJSON(&parseMsg); parseErr != nil {
		parseT.Fatalf("read debounce status message: %v", parseErr)
	}
	if parseMsg.Type != MessageTypeDebounceStatus {
		parseT.Fatalf("expected debounce status message, got %+v", parseMsg)
	}
	parsePayload, parseOk := parseMsg.Payload.(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected debounce payload map, got %#v", parseMsg.Payload)
	}
	if parsePayload["changeCount"] != float64(4) {
		parseT.Fatalf("expected change count 4, got %#v", parsePayload)
	}
	if parsePayload["waitTime"] != float64(quickDebounceTime.Milliseconds()) {
		parseT.Fatalf("expected quick debounce wait %dms, got %#v", quickDebounceTime.Milliseconds(), parsePayload)
	}
}

func TestDebounceAndBuildTruncatesWaitNearMaxWindow(parseT *testing.T) {
	parseServer := &LiveReloadServer{
		clients:         map[*websocket.Conn]ClientSession{},
		firstChangeTime: time.Now().Add(-(maxDebounceTime - 200*time.Millisecond)),
		changeCount:     1,
	}
	parseClientConn, parseCleanup := dialWebsocketHarness(parseT, func(parseConn *websocket.Conn) {
		parseServer.clients[parseConn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		parseServer.debounceAndBuild()
	})
	defer parseCleanup()
	defer func() {
		if parseServer.debounceTimer != nil {
			parseServer.debounceTimer.Stop()
		}
		if parseServer.maxDebounceTimer != nil {
			parseServer.maxDebounceTimer.Stop()
		}
	}()

	var parseMsg WebSocketMessage
	if parseErr := parseClientConn.ReadJSON(&parseMsg); parseErr != nil {
		parseT.Fatalf("read debounce status message: %v", parseErr)
	}
	parsePayload, parseOk := parseMsg.Payload.(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected debounce payload map, got %#v", parseMsg.Payload)
	}
	parseWaitTime, parseOk := parsePayload["waitTime"].(float64)
	if !parseOk {
		parseT.Fatalf("expected numeric wait time, got %#v", parsePayload)
	}
	if parseWaitTime <= 0 || parseWaitTime > 250 {
		parseT.Fatalf("expected truncated wait time near 200ms, got %#v", parsePayload)
	}
	if parsePayload["changeCount"] != float64(2) {
		parseT.Fatalf("expected change count 2, got %#v", parsePayload)
	}
}
