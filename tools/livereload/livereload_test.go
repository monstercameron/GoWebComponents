package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
)

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
	if overrides.Paths.LivereloadClientScript != "tools/livereload/scripts/livereload-client.js" {
		t.Fatalf("expected livereload client script in canonical example, got %#v", overrides.Paths)
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
		clients:            map[*websocket.Conn]bool{},
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

func TestResolveClientScriptPathFindsRepoRelativeScriptFromCwd(t *testing.T) {
	workspaceDir := t.TempDir()
	scriptPath := filepath.Join(workspaceDir, "tools", "livereload", "scripts", "livereload-client.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("failed to create script dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("console.log('ok');"), 0o644); err != nil {
		t.Fatalf("failed to write script fixture: %v", err)
	}

	previousGetwd := livereloadConfigGetwd
	previousExecutable := livereloadExecutablePath
	livereloadConfigGetwd = func() (string, error) { return workspaceDir, nil }
	livereloadExecutablePath = func() (string, error) { return "", os.ErrNotExist }
	defer func() {
		livereloadConfigGetwd = previousGetwd
		livereloadExecutablePath = previousExecutable
	}()

	if got := resolveClientScriptPath(); got != scriptPath {
		t.Fatalf("expected repo-relative client script %q, got %q", scriptPath, got)
	}
}

func TestResolveClientScriptPathFindsRepoRelativeScriptFromBuiltBinary(t *testing.T) {
	workspaceDir := t.TempDir()
	exePath := filepath.Join(workspaceDir, "bin", "tools", "livereload", "livereload.exe")
	scriptPath := filepath.Join(workspaceDir, "tools", "livereload", "scripts", "livereload-client.js")
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatalf("failed to create executable dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("failed to create script dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("console.log('ok');"), 0o644); err != nil {
		t.Fatalf("failed to write script fixture: %v", err)
	}

	previousGetwd := livereloadConfigGetwd
	previousExecutable := livereloadExecutablePath
	livereloadConfigGetwd = func() (string, error) { return filepath.Join(workspaceDir, "examples", "98-hot-reload"), nil }
	livereloadExecutablePath = func() (string, error) { return exePath, nil }
	defer func() {
		livereloadConfigGetwd = previousGetwd
		livereloadExecutablePath = previousExecutable
	}()

	if got := resolveClientScriptPath(); got != scriptPath {
		t.Fatalf("expected built-binary client script %q, got %q", scriptPath, got)
	}
}
