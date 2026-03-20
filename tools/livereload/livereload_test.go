package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestBuildStatusMarshalsStateSnapshot(t *testing.T) {
	status := BuildStatus{
		Success:       true,
		ReloadType:    "hot",
		StateSnapshot: `{"sharedCounter":1}`,
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
