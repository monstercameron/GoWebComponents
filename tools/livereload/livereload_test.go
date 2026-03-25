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
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
)

func dialWebsocketHarness(t *testing.T, handler func(*websocket.Conn)) (*websocket.Conn, func()) {
	t.Helper()
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade websocket: %v", err)
		}
		handler(conn)
	}))
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		httpServer.Close()
		t.Fatalf("dial websocket harness: %v", err)
	}
	cleanup := func() {
		_ = clientConn.Close()
		httpServer.Close()
	}
	return clientConn, cleanup
}

func TestLivereloadReportHelpers(t *testing.T) {
	report := livereloadReport(" LiveReloadServer.Start ", " /tmp/status ", " summary ", " consequence ", " next ")
	if report.Code != "GWC-TOOL-LIVERELOAD" || report.Headline != "tool failure in LiveReloadServer.Start" {
		t.Fatalf("unexpected report envelope: %+v", report)
	}
	if report.Path != "/tmp/status" || report.Summary != "summary" || report.Runtime != "consequence" || report.Next != "next" {
		t.Fatalf("expected report helper to trim fields, got %+v", report)
	}

	errReport := livereloadErrReport("subject", "/tmp/file", errors.New("boom"), "broken", "fix it")
	if errReport.Summary != "boom" || errReport.Path != "/tmp/file" {
		t.Fatalf("unexpected error report: %+v", errReport)
	}

	status := newBuildStatus(" compiling ", " phase summary ")
	if status.Phase != "compiling" || status.PhaseSummary != "phase summary" {
		t.Fatalf("unexpected build status helper output: %+v", status)
	}
}

func TestCanonicalRunnerConfigExampleMatchesLivereloadSchema(t *testing.T) {
	examplePath := filepath.Join("..", "..", "docs", "examples", "gwc-runner.example.json")
	content, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read canonical runner config example: %v", err)
	}

	var overrides runnerconfig.Overrides
	if err := json.Unmarshal(content, &overrides); err != nil {
		t.Fatalf("parse canonical runner config example: %v", err)
	}
	if overrides.Paths.LivereloadClientScript != "" {
		t.Fatalf("expected canonical example to omit livereload client script, got %#v", overrides.Paths)
	}
	if overrides.Paths.WorkspaceBuildRoot != "bin" {
		t.Fatalf("expected workspace build root in canonical example, got %#v", overrides.Paths)
	}
}

func TestPendingStateSnapshotBufferRoundTrips(t *testing.T) {
	server := &LiveReloadServer{}
	server.pendingStateSnapshot = `{"sharedCounter":1}`

	if got := server.takePendingStateSnapshot(); got != `{"sharedCounter":1}` {
		t.Fatalf("expected pending snapshot to round-trip, got %q", got)
	}
	if got := server.takePendingStateSnapshot(); got != "" {
		t.Fatalf("expected pending snapshot to clear after take, got %q", got)
	}

	server.pendingStateSnapshot = `{"sharedCounter":2}`
	server.clearPendingStateSnapshot()
	if got := server.takePendingStateSnapshot(); got != "" {
		t.Fatalf("expected pending snapshot to clear explicitly, got %q", got)
	}
}

func TestDebounceAndBuildQueuesFollowUpWhileBuildRunning(t *testing.T) {
	server := &LiveReloadServer{
		currentBuild: &exec.Cmd{Process: &os.Process{Pid: 1234}},
	}

	server.debounceAndBuild()

	if !server.buildQueued {
		t.Fatalf("expected follow-up rebuild to queue while a build is already running")
	}
	if server.currentBuild == nil || server.currentBuild.Process == nil || server.currentBuild.Process.Pid != 1234 {
		t.Fatalf("expected running build to stay active, got %#v", server.currentBuild)
	}
	if server.debounceTimer != nil {
		t.Fatalf("expected no debounce timer reset while build is already running")
	}
}

func TestBuildStatusMarshalsStateSnapshot(t *testing.T) {
	status := BuildStatus{
		Success:       true,
		ReloadType:    "hot",
		StateSnapshot: `{"sharedCounter":1}`,
		Manifest: &ChangedComponentManifest{
			ReloadType: "hot",
			Components: []ChangedComponent{{Name: "App", QualifiedName: "example.com/test.App", File: "main.go"}},
		},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("expected build status to marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("expected marshaled build status to decode: %v", err)
	}
	if _, ok := decoded["stateSnapshot"]; !ok {
		t.Fatalf("expected marshaled build status to include stateSnapshot, got %s", data)
	}
	if _, ok := decoded["manifest"]; !ok {
		t.Fatalf("expected marshaled build status to include manifest, got %s", data)
	}
}

func TestDescribeUpdateCompatibilityPlanHotReload(t *testing.T) {
	classification := newUpdateClassification("small", "hot", "UI changes: example components", []string{"examples/98-hot-reload/main.go"})

	if !classification.Plan.PreserveState {
		t.Fatalf("expected hot reload classification to preserve state, got %#v", classification.Plan)
	}
	if !classification.Plan.RemountSubtree {
		t.Fatalf("expected hot reload classification to warn about subtree remounts, got %#v", classification.Plan)
	}
	if !classification.Plan.RestartAsync {
		t.Fatalf("expected hot reload classification to restart async work, got %#v", classification.Plan)
	}
	if classification.Plan.FullReload {
		t.Fatalf("expected hot reload classification not to force full reload, got %#v", classification.Plan)
	}
	if !strings.Contains(classification.Plan.Summary, "preserve-state hot reload") {
		t.Fatalf("expected hot reload summary to explain the plan, got %q", classification.Plan.Summary)
	}
}

func TestDescribeUpdateCompatibilityPlanFullReload(t *testing.T) {
	classification := newUpdateClassification("big", "full", "Main function or entry point changed", []string{"main.go"})

	if classification.Plan.PreserveState {
		t.Fatalf("expected full reload classification not to preserve state, got %#v", classification.Plan)
	}
	if classification.Plan.RemountSubtree {
		t.Fatalf("expected full reload classification not to advertise subtree remounts, got %#v", classification.Plan)
	}
	if classification.Plan.RestartAsync {
		t.Fatalf("expected full reload classification not to advertise async restart-only behavior, got %#v", classification.Plan)
	}
	if !classification.Plan.FullReload {
		t.Fatalf("expected full reload classification to force full reload, got %#v", classification.Plan)
	}
	if !strings.Contains(classification.Plan.Summary, "planned full reload") {
		t.Fatalf("expected full reload summary to explain the plan, got %q", classification.Plan.Summary)
	}
}

func TestDescribeUpdateCompatibilityPlanEdgeCases(t *testing.T) {
	hotNoFiles := describeUpdateCompatibilityPlan(UpdateClassification{ReloadType: "hot"})
	if !hotNoFiles.PreserveState || hotNoFiles.RemountSubtree || hotNoFiles.RestartAsync {
		t.Fatalf("unexpected hot plan without changed files: %+v", hotNoFiles)
	}
	if !strings.Contains(hotNoFiles.Summary, "compatible local state is preserved") {
		t.Fatalf("unexpected hot-no-files summary: %q", hotNoFiles.Summary)
	}

	unknown := describeUpdateCompatibilityPlan(UpdateClassification{ReloadType: "mystery"})
	if unknown != (UpdateCompatibilityPlan{}) {
		t.Fatalf("expected empty compatibility plan for unknown reload type, got %+v", unknown)
	}
}

func TestBuildChangedComponentManifestExtractsTopLevelComponents(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod fixture: %v", err)
	}
	appDir := filepath.Join(workspaceDir, "examples", "98-hot-reload")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	mainPath := filepath.Join(appDir, "main.go")
	content := `package main

import "github.com/monstercameron/GoWebComponents/ui"

func App() ui.Node { return nil }
func helper() int { return 1 }
var Banner = func() ui.Node { return nil }
`
	if err := os.WriteFile(mainPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write main.go fixture: %v", err)
	}

	server := &LiveReloadServer{
		watchRoot:    workspaceDir,
		modulePath:   "example.com/test",
		manifestPath: filepath.Join(workspaceDir, "main.wasm.hotreload-manifest.json"),
	}

	manifest, err := server.buildChangedComponentManifest(UpdateClassification{
		ReloadType:   "hot",
		Reason:       "UI changes",
		ChangedFiles: []string{mainPath},
	})
	if err != nil {
		t.Fatalf("expected manifest build to succeed, got %v", err)
	}
	if len(manifest.Components) != 2 {
		t.Fatalf("expected 2 changed components, got %#v", manifest.Components)
	}
	if manifest.Components[0].QualifiedName != "example.com/test/examples/98-hot-reload.App" {
		t.Fatalf("unexpected first component: %#v", manifest.Components[0])
	}
	if manifest.Components[1].QualifiedName != "example.com/test/examples/98-hot-reload.Banner" {
		t.Fatalf("unexpected second component: %#v", manifest.Components[1])
	}
}

func TestWriteChangedComponentManifestPersistsJSON(t *testing.T) {
	workspaceDir := t.TempDir()
	manifestPath := filepath.Join(workspaceDir, "out", "main.wasm.hotreload-manifest.json")
	server := &LiveReloadServer{manifestPath: manifestPath}
	manifest := &ChangedComponentManifest{
		ReloadType:   "hot",
		ChangedFiles: []string{"examples/98-hot-reload/main.go"},
		Components:   []ChangedComponent{{Name: "App", QualifiedName: "example.com/test/examples/98-hot-reload.App", File: "examples/98-hot-reload/main.go"}},
	}

	if err := server.writeChangedComponentManifest(manifest); err != nil {
		t.Fatalf("expected manifest write to succeed, got %v", err)
	}
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("expected manifest file to exist, got %v", err)
	}
	if !strings.Contains(string(content), "example.com/test/examples/98-hot-reload.App") {
		t.Fatalf("expected manifest file to contain component identity, got %s", content)
	}
}

func TestCurrentClientSessionsSortsAndMarkClientSeenUpdatesTimestamp(t *testing.T) {
	connA := &websocket.Conn{}
	connB := &websocket.Conn{}
	earlier := time.Now().UTC().Add(-2 * time.Minute)
	later := earlier.Add(1 * time.Minute)
	server := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{
			connB: {ID: "client-b", ConnectedAt: later, LastSeenAt: later},
			connA: {ID: "client-a", ConnectedAt: earlier, LastSeenAt: earlier},
		},
	}

	sessions := server.currentClientSessions()
	if len(sessions) != 2 || sessions[0].ID != "client-a" || sessions[1].ID != "client-b" {
		t.Fatalf("expected sessions sorted by ConnectedAt then ID, got %+v", sessions)
	}

	server.markClientSeen(connA)
	updated := server.clients[connA]
	if !updated.LastSeenAt.After(earlier) {
		t.Fatalf("expected markClientSeen to advance LastSeenAt, got %+v", updated)
	}

	server.markClientSeen(&websocket.Conn{})
}

func TestHandleClientDisconnectValidation(t *testing.T) {
	server := &LiveReloadServer{clients: map[*websocket.Conn]ClientSession{}}
	handler := server.newHTTPHandler()

	getReq := httptest.NewRequest(http.MethodGet, "/__gwc/clients/disconnect", nil)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected method-not-allowed for GET disconnect, got %d", getResp.Code)
	}

	missingReq := httptest.NewRequest(http.MethodPost, "/__gwc/clients/disconnect", nil)
	missingResp := httptest.NewRecorder()
	handler.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for missing target, got %d", missingResp.Code)
	}

	invalidJSONReq := httptest.NewRequest(http.MethodPost, "/__gwc/clients/disconnect", strings.NewReader("{"))
	invalidJSONReq.Header.Set("Content-Type", "application/json")
	invalidJSONResp := httptest.NewRecorder()
	handler.ServeHTTP(invalidJSONResp, invalidJSONReq)
	if invalidJSONResp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid JSON, got %d", invalidJSONResp.Code)
	}
}

func TestResolveBuildDirUsesMainGoParent(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to create main.go fixture: %v", err)
	}

	got, err := resolveBuildDir(mainPath)
	if err != nil {
		t.Fatalf("expected build dir resolution to succeed, got %v", err)
	}
	if got != dir {
		t.Fatalf("expected build dir %q, got %q", dir, got)
	}
}

func TestResolveBuildDirUsesDirectoryInput(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveBuildDir(dir)
	if err != nil {
		t.Fatalf("expected build dir resolution to succeed, got %v", err)
	}
	if got != dir {
		t.Fatalf("expected build dir %q, got %q", dir, got)
	}
}

func TestResolveModuleRootFindsNearestGoMod(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod fixture: %v", err)
	}
	exampleDir := filepath.Join(workspaceDir, "examples", "98-hot-reload")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatalf("failed to create example dir: %v", err)
	}

	if got := resolveModuleRoot(exampleDir); got != workspaceDir {
		t.Fatalf("expected module root %q, got %q", workspaceDir, got)
	}
}

func TestNewLiveReloadServerUsesModuleRootForWatching(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod fixture: %v", err)
	}
	exampleDir := filepath.Join(workspaceDir, "examples", "98-hot-reload")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatalf("failed to create example dir: %v", err)
	}
	mainPath := filepath.Join(exampleDir, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write main.go fixture: %v", err)
	}

	server, err := NewLiveReloadServerWithOptions(LiveReloadOptions{
		MainPath:    mainPath,
		ProjectRoot: exampleDir,
	})
	if err != nil {
		t.Fatalf("expected server creation to succeed, got %v", err)
	}
	defer server.cleanup()

	if server.projectRoot != exampleDir {
		t.Fatalf("expected project root %q, got %q", exampleDir, server.projectRoot)
	}
	if server.watchRoot != workspaceDir {
		t.Fatalf("expected watch root %q, got %q", workspaceDir, server.watchRoot)
	}
	if server.buildDir != exampleDir {
		t.Fatalf("expected build dir %q, got %q", exampleDir, server.buildDir)
	}
	wantOutputPath := filepath.Join(workspaceDir, "bin", "examples", "98-hot-reload", "main.wasm")
	if server.outputPath != wantOutputPath {
		t.Fatalf("expected output path %q, got %q", wantOutputPath, server.outputPath)
	}
}

func TestNewHTTPHandlerServesParentStaticAssetsInSingleExampleMode(t *testing.T) {
	workspaceDir := t.TempDir()
	projectRoot := filepath.Join(workspaceDir, "98-hot-reload")
	if err := os.MkdirAll(filepath.Join(projectRoot), 0o755); err != nil {
		t.Fatalf("failed to create project root: %v", err)
	}
	cssPath := filepath.Join(workspaceDir, "static", "css", "tailwind.css")
	if err := os.MkdirAll(filepath.Dir(cssPath), 0o755); err != nil {
		t.Fatalf("failed to create parent static dir: %v", err)
	}
	const cssBody = "body { color: red; }"
	if err := os.WriteFile(cssPath, []byte(cssBody), 0o644); err != nil {
		t.Fatalf("failed to write static asset: %v", err)
	}

	server := &LiveReloadServer{
		projectRoot: projectRoot,
		staticDir:   resolveStaticDir(projectRoot),
	}

	req := httptest.NewRequest("GET", "/static/css/tailwind.css", nil)
	recorder := httptest.NewRecorder()
	server.newHTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("expected 200 serving parent static asset, got %d with body %q", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); body != cssBody {
		t.Fatalf("expected CSS body %q, got %q", cssBody, body)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/css") {
		t.Fatalf("expected CSS content type, got %q", contentType)
	}
	if server.staticDir != filepath.Join(workspaceDir, "static") {
		t.Fatalf("expected parent static dir to be selected, got %q", server.staticDir)
	}
}

func TestNewHTTPHandlerPrefersProjectStaticAssets(t *testing.T) {
	workspaceDir := t.TempDir()
	projectRoot := filepath.Join(workspaceDir, "98-hot-reload")
	parentCSSPath := filepath.Join(workspaceDir, "static", "css", "app.css")
	projectCSSPath := filepath.Join(projectRoot, "static", "css", "app.css")
	if err := os.MkdirAll(filepath.Dir(parentCSSPath), 0o755); err != nil {
		t.Fatalf("failed to create parent static dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(projectCSSPath), 0o755); err != nil {
		t.Fatalf("failed to create project static dir: %v", err)
	}
	if err := os.WriteFile(parentCSSPath, []byte("body { color: red; }"), 0o644); err != nil {
		t.Fatalf("failed to write parent static asset: %v", err)
	}
	const projectCSSBody = "body { color: blue; }"
	if err := os.WriteFile(projectCSSPath, []byte(projectCSSBody), 0o644); err != nil {
		t.Fatalf("failed to write project static asset: %v", err)
	}

	server := &LiveReloadServer{
		projectRoot: projectRoot,
		staticDir:   resolveStaticDir(projectRoot),
	}

	req := httptest.NewRequest("GET", "/static/css/app.css", nil)
	recorder := httptest.NewRecorder()
	server.newHTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("expected 200 serving project static asset, got %d with body %q", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); body != projectCSSBody {
		t.Fatalf("expected project CSS body %q, got %q", projectCSSBody, body)
	}
	if server.staticDir != filepath.Join(projectRoot, "static") {
		t.Fatalf("expected project static dir to be selected, got %q", server.staticDir)
	}
}

func TestNewHTTPHandlerReturnsNotFoundWithoutStaticDir(t *testing.T) {
	projectRoot := t.TempDir()
	server := &LiveReloadServer{
		projectRoot: projectRoot,
		staticDir:   resolveStaticDir(projectRoot),
	}

	req := httptest.NewRequest("GET", "/static/css/missing.css", nil)
	recorder := httptest.NewRecorder()
	server.newHTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != 404 {
		t.Fatalf("expected 404 when no static dir is available, got %d with body %q", recorder.Code, recorder.Body.String())
	}
	if server.staticDir != "" {
		t.Fatalf("expected empty static dir when none exist, got %q", server.staticDir)
	}
}

func TestNewHTTPHandlerServesStatusEndpoint(t *testing.T) {
	projectRoot := t.TempDir()
	server := &LiveReloadServer{
		projectRoot:        projectRoot,
		watchRoot:          projectRoot,
		buildDir:           projectRoot,
		outputPath:         filepath.Join(projectRoot, "main.wasm"),
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

	req := httptest.NewRequest("GET", "/__gwc/status", nil)
	recorder := httptest.NewRecorder()
	server.newHTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("expected 200 serving status endpoint, got %d with body %q", recorder.Code, recorder.Body.String())
	}

	var payload LiveReloadStatus
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON status payload, got %v", err)
	}
	if payload.Mode != "livereload-wasm" {
		t.Fatalf("expected mode livereload-wasm, got %#v", payload)
	}
	if payload.StatusURL != "http://127.0.0.1:8099/__gwc/status" {
		t.Fatalf("expected status URL to be reported, got %#v", payload)
	}
	if payload.WebSocketURL != "ws://127.0.0.1:8099/ws" {
		t.Fatalf("expected websocket URL to be reported, got %#v", payload)
	}
	if !payload.HotReloadEligible || !payload.HotReloadEnabled {
		t.Fatalf("expected hot reload to be enabled and eligible, got %#v", payload)
	}
	if payload.LastBuild == nil || payload.LastBuild.Phase != "serving_output" {
		t.Fatalf("expected last build phase in status payload, got %#v", payload)
	}
}

func TestNewHTTPHandlerServesResolvedWasmOutsideProjectRoot(t *testing.T) {
	workspaceDir := t.TempDir()
	projectRoot := filepath.Join(workspaceDir, "examples", "98-hot-reload")
	outputPath := filepath.Join(workspaceDir, "bin", "examples", "98-hot-reload", "main.wasm")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("failed to create project root: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("failed to create wasm output dir: %v", err)
	}
	const wasmBody = "wasm-bytes"
	if err := os.WriteFile(outputPath, []byte(wasmBody), 0o644); err != nil {
		t.Fatalf("failed to write wasm output: %v", err)
	}

	server := &LiveReloadServer{
		projectRoot: projectRoot,
		outputPath:  outputPath,
	}

	req := httptest.NewRequest("GET", "/main.wasm", nil)
	recorder := httptest.NewRecorder()
	server.newHTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("expected 200 serving wasm output, got %d with body %q", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); body != wasmBody {
		t.Fatalf("expected wasm body %q, got %q", wasmBody, body)
	}
}

func TestNewHTTPHandlerReportsAndDisconnectsClientSessions(t *testing.T) {
	projectRoot := t.TempDir()
	server := &LiveReloadServer{
		projectRoot: projectRoot,
		watchRoot:   projectRoot,
		buildDir:    projectRoot,
		outputPath:  filepath.Join(projectRoot, "main.wasm"),
		host:        "127.0.0.1",
		port:        "8099",
		clients:     map[*websocket.Conn]ClientSession{},
	}

	testServer := httptest.NewServer(server.newHTTPHandler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
	headers := http.Header{}
	headers.Set("User-Agent", "gwc-dashboard-test")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	statusResp, err := http.Get(testServer.URL + "/__gwc/status")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	defer statusResp.Body.Close()
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %s", statusResp.Status)
	}

	var status LiveReloadStatus
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatalf("decode status payload: %v", err)
	}
	if status.ClientCount != 1 || len(status.Clients) != 1 {
		t.Fatalf("expected one connected client in status, got %+v", status)
	}
	if status.Clients[0].ID == "" || status.Clients[0].UserAgent != "gwc-dashboard-test" {
		t.Fatalf("unexpected client metadata: %+v", status.Clients[0])
	}

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/__gwc/clients/disconnect", strings.NewReader(`{"all":true}`))
	if err != nil {
		t.Fatalf("build disconnect request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	disconnectResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post disconnect: %v", err)
	}
	defer disconnectResp.Body.Close()
	if disconnectResp.StatusCode != http.StatusOK {
		t.Fatalf("expected disconnect 200, got %s", disconnectResp.Status)
	}

	var response clientDisconnectResponse
	if err := json.NewDecoder(disconnectResp.Body).Decode(&response); err != nil {
		t.Fatalf("decode disconnect response: %v", err)
	}
	if response.Disconnected != 1 || response.Remaining != 0 || len(response.DisconnectedIDs) != 1 {
		t.Fatalf("unexpected disconnect response: %+v", response)
	}

	if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected websocket connection to close after disconnect")
	}
}

func TestResolveClientScriptPathReturnsEmptyWithoutConfiguredOverride(t *testing.T) {
	workspaceDir := t.TempDir()

	previousGetwd := livereloadConfigGetwd
	defer func() {
		livereloadConfigGetwd = previousGetwd
	}()
	livereloadConfigGetwd = func() (string, error) { return workspaceDir, nil }

	if got := resolveClientScriptPath(); got != "" {
		t.Fatalf("expected no default client script override, got %q", got)
	}
}

func TestResolveConfiguredClientScriptPathAndRunnerConfigPath(t *testing.T) {
	workspaceDir := t.TempDir()
	configPath := filepath.Join(workspaceDir, "gwc-runner.json")
	scriptPath := filepath.Join(workspaceDir, "vendor", "livereload-client.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("console.log('ok');"), 0o644); err != nil {
		t.Fatalf("write script file: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"paths":{"livereloadClientScript":"vendor/livereload-client.js"}}`), 0o644); err != nil {
		t.Fatalf("write runner config: %v", err)
	}

	previousGetwd := livereloadConfigGetwd
	defer func() {
		livereloadConfigGetwd = previousGetwd
	}()
	livereloadConfigGetwd = func() (string, error) { return workspaceDir, nil }

	if got := resolveConfiguredClientScriptPath(); got != scriptPath {
		t.Fatalf("expected configured client script %q, got %q", scriptPath, got)
	}
	if got := resolveLivereloadRunnerConfigPath(); got != configPath {
		t.Fatalf("expected runner config path %q, got %q", configPath, got)
	}

	if err := os.Remove(scriptPath); err != nil {
		t.Fatalf("remove script file: %v", err)
	}
	if got := resolveConfiguredClientScriptPath(); got != "" {
		t.Fatalf("expected missing configured client script to resolve empty, got %q", got)
	}
}

func TestFirstExistingPathNetAddrAndServedWASMPathHelpers(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "missing.js")
	second := filepath.Join(dir, "exists.js")
	if err := os.WriteFile(second, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write second candidate: %v", err)
	}
	if got := firstExistingPath("", first, second); got != second {
		t.Fatalf("expected first existing path %q, got %q", second, got)
	}
	if got := firstExistingPath("", first); got != "" {
		t.Fatalf("expected empty result when no candidates exist, got %q", got)
	}

	if got := netAddr("", ""); got != "127.0.0.1:8080" {
		t.Fatalf("expected default net addr, got %q", got)
	}
	if got := netAddr("0.0.0.0", "9010"); got != "0.0.0.0:9010" {
		t.Fatalf("expected explicit net addr, got %q", got)
	}

	projectRoot := filepath.Join(dir, "project")
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("mkdir build dir: %v", err)
	}

	server := &LiveReloadServer{projectRoot: projectRoot, buildDir: buildDir}
	if got := server.servedWASMPath(); got != "/main.wasm" {
		t.Fatalf("expected default served wasm path, got %q", got)
	}

	server.outputPath = "dist/app.wasm"
	if got := server.servedWASMPath(); got != "/app.wasm" {
		t.Fatalf("expected basename fallback for wasm outside project root, got %q", got)
	}

	server.buildDir = projectRoot
	server.outputPath = filepath.Join(projectRoot, "dist", "app.wasm")
	if got := server.servedWASMPath(); got != "/dist/app.wasm" {
		t.Fatalf("expected project-relative served wasm path, got %q", got)
	}
}

func TestResolveModuleRootStaticDirAndCleanupHelpers(t *testing.T) {
	if got := resolveModuleRoot(""); got != "" {
		t.Fatalf("expected empty module root for blank input, got %q", got)
	}
	if got := resolveModuleRoot(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Fatalf("expected empty module root for missing path, got %q", got)
	}

	root := t.TempDir()
	if got := resolveStaticDir(root); got != "" {
		t.Fatalf("expected no static dir, got %q", got)
	}

	server := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{},
	}
	server.cleanup()
}

func TestHandleWebSocketManagedTracksSessionsAndSnapshots(t *testing.T) {
	projectRoot := t.TempDir()
	server := &LiveReloadServer{
		projectRoot:     projectRoot,
		watchRoot:       projectRoot,
		buildDir:        projectRoot,
		outputPath:      filepath.Join(projectRoot, "main.wasm"),
		clients:         map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{Success: true, Phase: "serving_output"},
	}
	httpServer := httptest.NewServer(server.newHTTPHandler())
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	headers := http.Header{}
	headers.Set("User-Agent", "gwc-handle-websocket-test")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial managed websocket: %v", err)
	}
	defer clientConn.Close()

	var currentStatus WebSocketMessage
	if err := clientConn.ReadJSON(&currentStatus); err != nil {
		t.Fatalf("read initial current_status message: %v", err)
	}
	if currentStatus.Type != MessageTypeCurrentStatus {
		t.Fatalf("expected current_status message, got %+v", currentStatus)
	}
	if len(server.currentClientSessions()) != 1 {
		t.Fatalf("expected one tracked websocket client, got %+v", server.currentClientSessions())
	}

	if err := clientConn.WriteJSON(WebSocketMessage{
		Type:      MessageTypeStateSnapshot,
		Payload:   "snapshot-1",
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatalf("write state snapshot message: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	receivedSnapshot := false
	for time.Now().Before(deadline) {
		if got := server.takePendingStateSnapshot(); got == "snapshot-1" {
			receivedSnapshot = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !receivedSnapshot {
		t.Fatal("expected managed websocket to store a pending snapshot payload")
	}

	_ = clientConn.Close()
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(server.currentClientSessions()) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected websocket client cleanup after close, got %+v", server.currentClientSessions())
}

func TestSendCurrentBuildStatusAndBroadcastMessageDeliverPayloads(t *testing.T) {
	server := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{
			Success:    false,
			ReloadType: "full",
			Phase:      "blocked_on_error",
			Error:      "compile failed",
		},
	}

	clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
		defer conn.Close()
		server.clients[conn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		server.sendCurrentBuildStatus(conn)
		server.broadcastMessage(MessageTypeReload, map[string]string{"reason": "template"})
		time.Sleep(100 * time.Millisecond)
	})
	defer cleanup()

	var statusMsg WebSocketMessage
	if err := clientConn.ReadJSON(&statusMsg); err != nil {
		t.Fatalf("read current status message: %v", err)
	}
	if statusMsg.Type != MessageTypeCurrentStatus {
		t.Fatalf("expected current status message, got %+v", statusMsg)
	}

	var reloadMsg WebSocketMessage
	if err := clientConn.ReadJSON(&reloadMsg); err != nil {
		t.Fatalf("read broadcast reload message: %v", err)
	}
	if reloadMsg.Type != MessageTypeReload {
		t.Fatalf("expected reload message, got %+v", reloadMsg)
	}
}

func TestCheckCurrentBuildStateSuccessAndFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		buildDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte("module example.com/checkstate\n\ngo 1.26.0\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(filepath.Join(buildDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}

		server := &LiveReloadServer{
			buildDir:   buildDir,
			outputPath: filepath.Join(buildDir, "bin", "main.exe"),
		}
		clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
			defer conn.Close()
			server.checkCurrentBuildState(conn)
		})
		defer cleanup()

		var msg WebSocketMessage
		if err := clientConn.ReadJSON(&msg); err != nil {
			t.Fatalf("read success build-state message: %v", err)
		}
		if msg.Type != MessageTypeCurrentStatus {
			t.Fatalf("expected current status message, got %+v", msg)
		}
		if server.lastBuildStatus == nil || !server.lastBuildStatus.Success || server.lastBuildStatus.Phase != "serving_output" {
			t.Fatalf("unexpected success build status: %+v", server.lastBuildStatus)
		}
	})

	t.Run("failure", func(t *testing.T) {
		buildDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte("module example.com/checkstatefail\n\ngo 1.26.0\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(filepath.Join(buildDir, "main.go"), []byte("package main\nfunc main() { nope }\n"), 0o644); err != nil {
			t.Fatalf("write bad main.go: %v", err)
		}

		server := &LiveReloadServer{
			buildDir:   buildDir,
			outputPath: filepath.Join(buildDir, "bin", "main.exe"),
		}
		clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
			defer conn.Close()
			server.checkCurrentBuildState(conn)
		})
		defer cleanup()

		var msg WebSocketMessage
		if err := clientConn.ReadJSON(&msg); err != nil {
			t.Fatalf("read failure build-state message: %v", err)
		}
		if msg.Type != MessageTypeCurrentStatus {
			t.Fatalf("expected current status message, got %+v", msg)
		}
		if server.lastBuildStatus == nil || server.lastBuildStatus.Success || server.lastBuildStatus.Phase != "blocked_on_error" {
			t.Fatalf("unexpected failure build status: %+v", server.lastBuildStatus)
		}
		if !strings.Contains(server.lastBuildStatus.Error, "undefined") && !strings.Contains(server.lastBuildStatus.Error, "nope") {
			t.Fatalf("expected compile failure details, got %+v", server.lastBuildStatus)
		}
	})
}

func TestHandleHTMLInjectsClientScriptAndWasmConfig(t *testing.T) {
	projectRoot := t.TempDir()
	indexPath := filepath.Join(projectRoot, "index.html")
	scriptPath := filepath.Join(projectRoot, "scripts", "custom-livereload-client.txt")
	outputPath := filepath.Join(projectRoot, "dist", "main.wasm")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("<html><body><h1>App</h1></body></html>"), 0o644); err != nil {
		t.Fatalf("write index fixture: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("console.log('livereload');"), 0o644); err != nil {
		t.Fatalf("write client script fixture: %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write wasm fixture: %v", err)
	}

	server := &LiveReloadServer{
		projectRoot:      projectRoot,
		buildDir:         projectRoot,
		outputPath:       outputPath,
		clientScriptPath: scriptPath,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	server.handleHTML(recorder, req, indexPath)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTML handler success, got %d with body %q", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "window.__GWC_LIVERELOAD_CONFIG") || !strings.Contains(body, `wasmPath: "/dist/main.wasm"`) {
		t.Fatalf("expected HTML injection to include wasm config, got %q", body)
	}
	if !strings.Contains(body, "console.log('livereload');") {
		t.Fatalf("expected HTML injection to include livereload client script, got %q", body)
	}
}

func TestHandleHTMLInjectsEmbeddedClientScriptByDefault(t *testing.T) {
	projectRoot := t.TempDir()
	indexPath := filepath.Join(projectRoot, "index.html")
	outputPath := filepath.Join(projectRoot, "dist", "main.wasm")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("<html><body><h1>App</h1></body></html>"), 0o644); err != nil {
		t.Fatalf("write index fixture: %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write wasm fixture: %v", err)
	}

	server := &LiveReloadServer{
		projectRoot: projectRoot,
		buildDir:    projectRoot,
		outputPath:  outputPath,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	server.handleHTML(recorder, req, indexPath)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTML handler success, got %d with body %q", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Live Reload Client Script") {
		t.Fatalf("expected HTML injection to include embedded livereload client script, got %q", body)
	}
}

func TestHandleHTMLMissingClientScriptReturnsServerError(t *testing.T) {
	projectRoot := t.TempDir()
	indexPath := filepath.Join(projectRoot, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>App</body></html>"), 0o644); err != nil {
		t.Fatalf("write index fixture: %v", err)
	}

	server := &LiveReloadServer{
		projectRoot:      projectRoot,
		buildDir:         projectRoot,
		outputPath:       filepath.Join(projectRoot, "main.wasm"),
		clientScriptPath: filepath.Join(projectRoot, "missing-client.js"),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	server.handleHTML(recorder, req, indexPath)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected server error for missing client script, got %d with body %q", recorder.Code, recorder.Body.String())
	}
}

func TestRequestStateSnapshotBroadcastsStateExport(t *testing.T) {
	server := &LiveReloadServer{
		clients: map[*websocket.Conn]ClientSession{},
	}

	clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
		defer conn.Close()
		server.clients[conn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		server.requestStateSnapshot()
		time.Sleep(100 * time.Millisecond)
	})
	defer cleanup()

	var msg WebSocketMessage
	if err := clientConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read state export message: %v", err)
	}
	if msg.Type != MessageTypeStateExport {
		t.Fatalf("expected state export message, got %+v", msg)
	}
}

func TestHandleWebSocketTracksSessionsAndSnapshots(t *testing.T) {
	server := &LiveReloadServer{
		clients:         map[*websocket.Conn]ClientSession{},
		lastBuildStatus: &BuildStatus{Success: true, Phase: "serving_output"},
	}
	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	headers := http.Header{}
	headers.Set("User-Agent", "gwc-legacy-websocket-test")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial legacy websocket: %v", err)
	}
	defer clientConn.Close()

	var currentStatus WebSocketMessage
	if err := clientConn.ReadJSON(&currentStatus); err != nil {
		t.Fatalf("read initial current status from legacy websocket: %v", err)
	}
	if currentStatus.Type != MessageTypeCurrentStatus {
		t.Fatalf("expected current status message, got %+v", currentStatus)
	}
	if len(server.currentClientSessions()) != 1 {
		t.Fatalf("expected one tracked legacy websocket client, got %+v", server.currentClientSessions())
	}

	if err := clientConn.WriteJSON(WebSocketMessage{
		Type:      MessageTypeStateSnapshot,
		Payload:   "legacy-snapshot",
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatalf("write legacy snapshot message: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := server.takePendingStateSnapshot(); got == "legacy-snapshot" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := server.takePendingStateSnapshot(); got != "" {
		t.Fatalf("expected pending snapshot buffer to be drained, got %q", got)
	}

	_ = clientConn.Close()
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(server.currentClientSessions()) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected legacy websocket client cleanup after close, got %+v", server.currentClientSessions())
}

func TestNewLiveReloadServerWithOptionsResolvesRelativePathsAndDefaults(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	projectRoot := filepath.Join(workspaceDir, "app")
	indexRel := filepath.Join("custom", "index.html")
	scriptRel := filepath.Join("scripts", "client.js")
	if err := os.MkdirAll(filepath.Join(projectRoot, "custom"), 0o755); err != nil {
		t.Fatalf("mkdir custom dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "static"), 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, indexRel), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write custom index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, scriptRel), []byte("console.log('client');"), 0o644); err != nil {
		t.Fatalf("write client script: %v", err)
	}

	server, err := NewLiveReloadServerWithOptions(LiveReloadOptions{
		ProjectRoot:      projectRoot,
		IndexPath:        indexRel,
		OutputPath:       filepath.Join("dist", "app.wasm"),
		ClientScriptPath: scriptRel,
		AlwaysHotReload:  true,
	})
	if err != nil {
		t.Fatalf("new live reload server: %v", err)
	}
	defer server.cleanup()

	if server.projectRoot != projectRoot {
		t.Fatalf("expected project root %q, got %q", projectRoot, server.projectRoot)
	}
	if server.buildDir != projectRoot {
		t.Fatalf("expected build dir %q, got %q", projectRoot, server.buildDir)
	}
	if server.watchRoot != workspaceDir {
		t.Fatalf("expected watch root %q, got %q", workspaceDir, server.watchRoot)
	}
	if server.indexPath != filepath.Join(projectRoot, indexRel) {
		t.Fatalf("expected resolved index path, got %q", server.indexPath)
	}
	if server.outputPath != filepath.Join(projectRoot, "dist", "app.wasm") {
		t.Fatalf("expected resolved output path, got %q", server.outputPath)
	}
	if server.clientScriptPath != filepath.Join(projectRoot, scriptRel) {
		t.Fatalf("expected resolved client script path, got %q", server.clientScriptPath)
	}
	if server.host != defaultHost || server.port != defaultPort {
		t.Fatalf("expected default host/port %s:%s, got %s:%s", defaultHost, defaultPort, server.host, server.port)
	}
	if !server.alwaysHotReload {
		t.Fatal("expected always hot reload flag to persist")
	}
	if server.staticDir != filepath.Join(projectRoot, "static") {
		t.Fatalf("expected project static dir, got %q", server.staticDir)
	}
	if server.manifestPath != filepath.Join(projectRoot, "dist", "app.wasm.hotreload-manifest.json") {
		t.Fatalf("unexpected manifest path %q", server.manifestPath)
	}
}

func TestNewLiveReloadServerWithOptionsFallsBackToStaticIndexAndWorkspaceBuildRoot(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "gwc-runner.json"), []byte(`{"paths":{"workspaceBuildRoot":"out"}}`), 0o644); err != nil {
		t.Fatalf("write runner config: %v", err)
	}
	projectRoot := filepath.Join(workspaceDir, "examples", "demo")
	staticIndex := filepath.Join(projectRoot, "static", "index.html")
	if err := os.MkdirAll(filepath.Dir(staticIndex), 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(staticIndex, []byte("<html>static</html>"), 0o644); err != nil {
		t.Fatalf("write static index: %v", err)
	}

	server, err := NewLiveReloadServerWithOptions(LiveReloadOptions{
		ProjectRoot: projectRoot,
	})
	if err != nil {
		t.Fatalf("new live reload server: %v", err)
	}
	defer server.cleanup()

	if server.indexPath != staticIndex {
		t.Fatalf("expected static index fallback %q, got %q", staticIndex, server.indexPath)
	}
	if server.outputPath != filepath.Join(workspaceDir, "out", "examples", "demo", "main.wasm") {
		t.Fatalf("expected workspace build root output path, got %q", server.outputPath)
	}
}

func TestCurrentStatusReturnsCopiesAndCurrentErrorForFailedBuild(t *testing.T) {
	projectRoot := t.TempDir()
	conn := &websocket.Conn{}
	connectedAt := time.Now().UTC().Add(-time.Minute)
	server := &LiveReloadServer{
		projectRoot:        projectRoot,
		watchRoot:          projectRoot,
		buildDir:           projectRoot,
		outputPath:         filepath.Join(projectRoot, "main.wasm"),
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
			conn: {ID: "client-1", ConnectedAt: connectedAt, LastSeenAt: connectedAt},
		},
	}

	status := server.currentStatus()
	if status.Mode != "livereload-wasm" || status.ListeningURL != "http://127.0.0.1:9090" {
		t.Fatalf("unexpected status envelope: %+v", status)
	}
	if !status.HotReloadEligible || status.HotReloadEnabled {
		t.Fatalf("unexpected hot reload flags: %+v", status)
	}
	if status.CurrentError == nil || status.CurrentError.Error != "compile failed" {
		t.Fatalf("expected current error copy for failed build, got %+v", status.CurrentError)
	}
	if status.LastBuild == nil || status.LastBuild.Error != "compile failed" {
		t.Fatalf("expected last build copy, got %+v", status.LastBuild)
	}
	if status.ClientCount != 1 || len(status.Clients) != 1 {
		t.Fatalf("expected client snapshot in status, got %+v", status)
	}

	server.lastBuildStatus.Error = "mutated"
	server.clients[conn] = ClientSession{ID: "client-2", ConnectedAt: connectedAt, LastSeenAt: connectedAt}
	if status.LastBuild.Error != "compile failed" || status.CurrentError.Error != "compile failed" || status.Clients[0].ID != "client-1" {
		t.Fatalf("expected status snapshot copies to be isolated, got %+v", status)
	}
}

func TestAddWatchersAddsDirectoriesAndSkipsIgnoredRoots(t *testing.T) {
	root := t.TempDir()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	defer watcher.Close()

	dirs := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "app", "nested"),
		filepath.Join(root, ".git"),
		filepath.Join(root, "vendor"),
		filepath.Join(root, "node_modules"),
		filepath.Join(root, "bin"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
	}

	server := &LiveReloadServer{watcher: watcher}
	if err := server.addWatchers(root); err != nil {
		t.Fatalf("add watchers: %v", err)
	}

	watched := watcher.WatchList()
	if !slices.Contains(watched, root) || !slices.Contains(watched, filepath.Join(root, "app")) || !slices.Contains(watched, filepath.Join(root, "app", "nested")) {
		t.Fatalf("expected real directories to be watched, got %+v", watched)
	}
	if slices.Contains(watched, filepath.Join(root, ".git")) || slices.Contains(watched, filepath.Join(root, "vendor")) || slices.Contains(watched, filepath.Join(root, "node_modules")) || slices.Contains(watched, filepath.Join(root, "bin")) {
		t.Fatalf("expected ignored directories to be skipped, got %+v", watched)
	}
}

func TestAddWatchersReturnsWalkErrorForMissingRoot(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	defer watcher.Close()

	server := &LiveReloadServer{watcher: watcher}
	if err := server.addWatchers(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected missing root to return a walk error")
	}
}

func TestHandleFileEventTracksGoChangesAndSkipsIgnoredFiles(t *testing.T) {
	server := &LiveReloadServer{
		clients:      map[*websocket.Conn]ClientSession{},
		changedFiles: make(map[string]time.Time),
	}

	server.handleFileEvent(fsnotify.Event{Name: "styles.css", Op: fsnotify.Write})
	server.handleFileEvent(fsnotify.Event{Name: "main.go.tmp", Op: fsnotify.Write})
	server.handleFileEvent(fsnotify.Event{Name: "main.go~", Op: fsnotify.Write})
	if len(server.changedFiles) != 0 {
		t.Fatalf("expected ignored files not to be tracked, got %+v", server.changedFiles)
	}

	server.handleFileEvent(fsnotify.Event{Name: "main.go", Op: fsnotify.Write})
	if len(server.changedFiles) != 1 {
		t.Fatalf("expected go write to be tracked, got %+v", server.changedFiles)
	}
	if _, ok := server.changedFiles["main.go"]; !ok {
		t.Fatalf("expected main.go change to be tracked, got %+v", server.changedFiles)
	}
	if server.changeCount != 1 || server.debounceTimer == nil || server.maxDebounceTimer == nil {
		t.Fatalf("expected debounce state to initialize, got changeCount=%d debounce=%v maxDebounce=%v", server.changeCount, server.debounceTimer, server.maxDebounceTimer)
	}
	server.resetDebounceState()
	if server.debounceTimer != nil {
		server.debounceTimer.Stop()
		server.debounceTimer = nil
	}
}

func TestClassifyUpdateBranches(t *testing.T) {
	projectRoot := t.TempDir()

	t.Run("always hot reload", func(t *testing.T) {
		server := &LiveReloadServer{
			projectRoot:     projectRoot,
			alwaysHotReload: true,
			changedFiles:    map[string]time.Time{filepath.Join(projectRoot, "pkg", "thing.go"): time.Now()},
		}
		classification := server.classifyUpdate()
		if classification.ReloadType != "hot" || !strings.Contains(classification.Reason, "Hot reload") {
			t.Fatalf("expected always-hot classification, got %+v", classification)
		}
		if len(server.changedFiles) != 0 {
			t.Fatalf("expected changed files to clear after classification, got %+v", server.changedFiles)
		}
	})

	t.Run("main go triggers full reload", func(t *testing.T) {
		server := &LiveReloadServer{
			projectRoot:  projectRoot,
			changedFiles: map[string]time.Time{filepath.Join(projectRoot, "main.go"): time.Now()},
		}
		classification := server.classifyUpdate()
		if classification.ReloadType != "full" || !strings.Contains(classification.Reason, "entry point") {
			t.Fatalf("expected entry-point full reload, got %+v", classification)
		}
	})

	t.Run("ui directories dedupe to hot reload", func(t *testing.T) {
		server := &LiveReloadServer{
			projectRoot: projectRoot,
			changedFiles: map[string]time.Time{
				filepath.Join(projectRoot, "examples", "a.go"): time.Now(),
				filepath.Join(projectRoot, "website", "b.go"):  time.Now(),
				filepath.Join(projectRoot, "examples", "c.go"): time.Now(),
			},
		}
		classification := server.classifyUpdate()
		if classification.ReloadType != "hot" || !strings.Contains(classification.Reason, "example components") || !strings.Contains(classification.Reason, "website components") {
			t.Fatalf("expected ui-directory hot reload, got %+v", classification)
		}
	})

	t.Run("logic changes default to full reload", func(t *testing.T) {
		server := &LiveReloadServer{
			projectRoot:  projectRoot,
			changedFiles: map[string]time.Time{filepath.Join(projectRoot, "pkg", "logic.go"): time.Now()},
		}
		classification := server.classifyUpdate()
		if classification.ReloadType != "full" || !strings.Contains(classification.Reason, "Logic changes") {
			t.Fatalf("expected logic-change full reload, got %+v", classification)
		}
	})
}

func TestComponentSignatureHelpers(t *testing.T) {
	if isComponentValueSpec(nil, 0) {
		t.Fatal("nil value spec should not be treated as a component")
	}

	componentType := &ast.FuncType{
		Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "ui"},
			Sel: &ast.Ident{Name: "Node"},
		}}}},
	}
	nonComponentType := &ast.FuncType{
		Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "error"}}}},
	}

	if !returnsComponentNode(componentType) {
		t.Fatal("expected ui.Node function type to be treated as a component")
	}
	if returnsComponentNode(nonComponentType) {
		t.Fatal("expected non-component function type not to be treated as a component")
	}

	typedSpec := &ast.ValueSpec{Type: componentType}
	if !isComponentValueSpec(typedSpec, 0) {
		t.Fatal("expected typed function value spec to be treated as a component")
	}

	literalSpec := &ast.ValueSpec{Values: []ast.Expr{
		&ast.FuncLit{Type: componentType},
		&ast.FuncLit{Type: nonComponentType},
	}}
	if !isComponentValueSpec(literalSpec, 0) {
		t.Fatal("expected component func literal to be treated as a component")
	}
	if isComponentValueSpec(literalSpec, 1) {
		t.Fatal("expected non-component func literal not to be treated as a component")
	}
	if isComponentValueSpec(literalSpec, 2) {
		t.Fatal("expected out-of-range value spec index to be false")
	}
}

func TestComponentResultExpressionHelpers(t *testing.T) {
	cases := []struct {
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

	for _, tc := range cases {
		if got := isComponentResultExpr(tc.expr); got != tc.want {
			t.Fatalf("%s: isComponentResultExpr() = %v, want %v", tc.name, got, tc.want)
		}
	}

	if !isRuntimeElementExpr(&ast.Ident{Name: "Element"}) {
		t.Fatal("expected ident Element to be runtime element")
	}
	if isRuntimeElementExpr(&ast.Ident{Name: "Node"}) {
		t.Fatal("expected ident Node not to be runtime element")
	}
	if !isRuntimeElementExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "ui"}, Sel: &ast.Ident{Name: "Element"}}) {
		t.Fatal("expected ui.Element to be runtime element")
	}
	if isRuntimeElementExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Element"}}) {
		t.Fatal("expected pkg.Element not to be runtime element")
	}
}

func TestReturnsComponentNodeCornerCases(t *testing.T) {
	if returnsComponentNode(nil) {
		t.Fatal("nil func type should not return a component node")
	}
	if returnsComponentNode(&ast.FuncType{}) {
		t.Fatal("func type without results should not return a component node")
	}
	if returnsComponentNode(&ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{
		{Type: &ast.Ident{Name: "Node"}},
		{Type: &ast.Ident{Name: "error"}},
	}}}) {
		t.Fatal("multi-result func type should not be treated as a component")
	}
}

func TestPackageAndModulePathHelpers(t *testing.T) {
	if got := qualifyComponentName("", "App"); got != "App" {
		t.Fatalf("expected unqualified component name, got %q", got)
	}
	if got := qualifyComponentName("example.com/test", "App"); got != "example.com/test.App" {
		t.Fatalf("expected qualified component name, got %q", got)
	}

	watchRoot := filepath.Join("workspace", "repo")
	filePath := filepath.Join(watchRoot, "examples", "demo", "main.go")
	if got := resolvePackagePath("example.com/test", watchRoot, filePath); got != "example.com/test/examples/demo" {
		t.Fatalf("expected resolved package path, got %q", got)
	}
	if got := resolvePackagePath("", watchRoot, filePath); got != "examples/demo" {
		t.Fatalf("expected relative package path without module, got %q", got)
	}
	if got := resolvePackagePath("example.com/test", watchRoot, filepath.Join(watchRoot, "main.go")); got != "example.com/test" {
		t.Fatalf("expected module root package path, got %q", got)
	}

	root := t.TempDir()
	goModPath := filepath.Join(root, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module example.com/test\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if got := resolveModulePath(root); got != "example.com/test" {
		t.Fatalf("expected module path from go.mod, got %q", got)
	}
	if err := os.WriteFile(goModPath, []byte("go 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("rewrite go.mod without module: %v", err)
	}
	if got := resolveModulePath(root); got != "" {
		t.Fatalf("expected empty module path without module directive, got %q", got)
	}
}

func TestWriteChangedComponentManifestNoopCases(t *testing.T) {
	server := &LiveReloadServer{}
	if err := server.writeChangedComponentManifest(nil); err != nil {
		t.Fatalf("expected nil manifest to be ignored, got %v", err)
	}

	server.manifestPath = ""
	if err := server.writeChangedComponentManifest(&ChangedComponentManifest{}); err != nil {
		t.Fatalf("expected empty manifest path to be ignored, got %v", err)
	}
}

func TestWriteChangedComponentManifestDirectoryError(t *testing.T) {
	root := t.TempDir()
	blockingFile := filepath.Join(root, "blocking-file")
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	server := &LiveReloadServer{
		manifestPath: filepath.Join(blockingFile, "manifest.json"),
	}
	err := server.writeChangedComponentManifest(&ChangedComponentManifest{ReloadType: "hot"})
	if err == nil || !strings.Contains(err.Error(), "create manifest dir") {
		t.Fatalf("expected manifest directory creation error, got %v", err)
	}
}

func TestNewLiveReloadServerWrapperUsesProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	indexPath := filepath.Join(projectRoot, "static", "index.html")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write static index: %v", err)
	}

	server, err := NewLiveReloadServer(projectRoot)
	if err != nil {
		t.Fatalf("new live reload server: %v", err)
	}
	defer server.cleanup()

	if server.projectRoot != projectRoot {
		t.Fatalf("expected wrapper project root %q, got %q", projectRoot, server.projectRoot)
	}
	if server.buildDir != projectRoot {
		t.Fatalf("expected wrapper build dir %q, got %q", projectRoot, server.buildDir)
	}
	if server.indexPath != indexPath {
		t.Fatalf("expected wrapper to fall back to static index %q, got %q", indexPath, server.indexPath)
	}
}

func TestResolveBuildDirMissingPathReturnsError(t *testing.T) {
	if _, err := resolveBuildDir(filepath.Join(t.TempDir(), "missing", "main.go")); err == nil {
		t.Fatal("expected missing build entry path to return an error")
	}
}

func TestResolveModuleRootFromFileAndServedWASMDefaultCases(t *testing.T) {
	root := t.TempDir()
	goModPath := filepath.Join(root, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	mainPath := filepath.Join(root, "cmd", "demo", "main.go")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatalf("mkdir main dir: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	if got := resolveModuleRoot(mainPath); got != root {
		t.Fatalf("expected file-based module root %q, got %q", root, got)
	}
	if got := (*LiveReloadServer)(nil).servedWASMPath(); got != "/main.wasm" {
		t.Fatalf("expected nil server to serve default wasm path, got %q", got)
	}

	server := &LiveReloadServer{projectRoot: root, buildDir: root}
	if got := server.servedWASMPath(); got != "/main.wasm" {
		t.Fatalf("expected empty output path to serve default wasm path, got %q", got)
	}
}

func TestResolveClientScriptPathAndRunnerConfigPathHandleGetwdFailure(t *testing.T) {
	previousGetwd := livereloadConfigGetwd
	defer func() {
		livereloadConfigGetwd = previousGetwd
	}()
	livereloadConfigGetwd = func() (string, error) { return "", errors.New("boom") }
	if got := resolveClientScriptPath(); got != "" {
		t.Fatalf("expected empty client script path on getwd failure, got %q", got)
	}
	if got := resolveConfiguredClientScriptPath(); got != "" {
		t.Fatalf("expected empty configured client script path on getwd failure, got %q", got)
	}
	if got := resolveLivereloadRunnerConfigPath(); got != "" {
		t.Fatalf("expected empty runner config path on getwd failure, got %q", got)
	}
}

func TestBroadcastMessageHandlesMarshalAndWriteErrors(t *testing.T) {
	t.Run("marshal error", func(t *testing.T) {
		server := &LiveReloadServer{clients: map[*websocket.Conn]ClientSession{}}
		server.broadcastMessage(MessageTypeReload, map[string]interface{}{"bad": make(chan int)})
	})

	t.Run("write error", func(t *testing.T) {
		server := &LiveReloadServer{
			clients: map[*websocket.Conn]ClientSession{},
		}
		clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
			server.clients[conn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
			_ = conn.Close()
			server.broadcastMessage(MessageTypeReload, map[string]string{"reason": "closed"})
		})
		defer cleanup()
		_ = clientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _, _ = clientConn.ReadMessage()
	})
}

func TestSendCurrentBuildStatusHandlesClosedConnection(t *testing.T) {
	server := &LiveReloadServer{
		lastBuildStatus: &BuildStatus{
			Success: false,
			Error:   "compile failed",
		},
	}
	clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
		_ = conn.Close()
		server.sendCurrentBuildStatus(conn)
	})
	defer cleanup()
	_ = clientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, _ = clientConn.ReadMessage()
}

func TestCleanupClosesWatcherAndWebsocketClients(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	server := &LiveReloadServer{
		watcher:          watcher,
		httpServer:       &http.Server{},
		clients:          map[*websocket.Conn]ClientSession{},
		debounceTimer:    time.NewTimer(time.Minute),
		maxDebounceTimer: time.NewTimer(time.Minute),
	}
	clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
		server.clients[conn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		server.cleanup()
	})
	defer cleanup()

	if err := watcher.Add(t.TempDir()); err == nil {
		t.Fatal("expected closed watcher to reject new watches after cleanup")
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, _, err := clientConn.ReadMessage(); err == nil {
		t.Fatal("expected websocket client to close during cleanup")
	}
}

func TestDebounceAndBuildUsesQuickDebounceForLaterChanges(t *testing.T) {
	server := &LiveReloadServer{
		clients:         map[*websocket.Conn]ClientSession{},
		firstChangeTime: time.Now().Add(-2 * time.Second),
		changeCount:     3,
	}
	clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
		server.clients[conn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		server.debounceAndBuild()
	})
	defer cleanup()
	defer func() {
		if server.debounceTimer != nil {
			server.debounceTimer.Stop()
		}
		if server.maxDebounceTimer != nil {
			server.maxDebounceTimer.Stop()
		}
	}()

	var msg WebSocketMessage
	if err := clientConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read debounce status message: %v", err)
	}
	if msg.Type != MessageTypeDebounceStatus {
		t.Fatalf("expected debounce status message, got %+v", msg)
	}
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected debounce payload map, got %#v", msg.Payload)
	}
	if payload["changeCount"] != float64(4) {
		t.Fatalf("expected change count 4, got %#v", payload)
	}
	if payload["waitTime"] != float64(quickDebounceTime.Milliseconds()) {
		t.Fatalf("expected quick debounce wait %dms, got %#v", quickDebounceTime.Milliseconds(), payload)
	}
}

func TestDebounceAndBuildTruncatesWaitNearMaxWindow(t *testing.T) {
	server := &LiveReloadServer{
		clients:         map[*websocket.Conn]ClientSession{},
		firstChangeTime: time.Now().Add(-(maxDebounceTime - 200*time.Millisecond)),
		changeCount:     1,
	}
	clientConn, cleanup := dialWebsocketHarness(t, func(conn *websocket.Conn) {
		server.clients[conn] = ClientSession{ID: "client-1", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}
		server.debounceAndBuild()
	})
	defer cleanup()
	defer func() {
		if server.debounceTimer != nil {
			server.debounceTimer.Stop()
		}
		if server.maxDebounceTimer != nil {
			server.maxDebounceTimer.Stop()
		}
	}()

	var msg WebSocketMessage
	if err := clientConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read debounce status message: %v", err)
	}
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected debounce payload map, got %#v", msg.Payload)
	}
	waitTime, ok := payload["waitTime"].(float64)
	if !ok {
		t.Fatalf("expected numeric wait time, got %#v", payload)
	}
	if waitTime <= 0 || waitTime > 250 {
		t.Fatalf("expected truncated wait time near 200ms, got %#v", payload)
	}
	if payload["changeCount"] != float64(2) {
		t.Fatalf("expected change count 2, got %#v", payload)
	}
}
