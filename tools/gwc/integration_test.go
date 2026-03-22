package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveScaffoldMetadataForConfigUsesConfiguredAppDirectory(t *testing.T) {
	workingDir := t.TempDir()
	projectDir := t.TempDir()
	appPath := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(appPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	metadata := `{
  "projectName": "configured-app-metadata",
  "modulePath": "example.com/configured-app-metadata",
  "tooling": {
    "appPath": "main.go"
  }
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	got, metadataDir, ok, err := resolveScaffoldMetadataForConfig(workingDir, "", appPath)
	if err != nil {
		t.Fatalf("resolve scaffold metadata: %v", err)
	}
	if !ok {
		t.Fatal("expected metadata to be discovered from the configured app directory")
	}
	if metadataDir != projectDir {
		t.Fatalf("expected metadata dir %q, got %q", projectDir, metadataDir)
	}
	if got.ProjectName != "configured-app-metadata" {
		t.Fatalf("expected project name from metadata, got %#v", got)
	}
}

func TestResolveScaffoldMetadataForConfigUsesRelativeConfiguredRoot(t *testing.T) {
	workingDir := t.TempDir()
	projectDir := filepath.Join(workingDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	metadata := `{
  "projectName": "relative-root-metadata",
  "modulePath": "example.com/relative-root-metadata"
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	got, metadataDir, ok, err := resolveScaffoldMetadataForConfig(workingDir, "project", "")
	if err != nil {
		t.Fatalf("resolve scaffold metadata: %v", err)
	}
	if !ok {
		t.Fatal("expected metadata to be discovered from the configured relative root")
	}
	if metadataDir != projectDir {
		t.Fatalf("expected metadata dir %q, got %q", projectDir, metadataDir)
	}
	if got.ProjectName != "relative-root-metadata" {
		t.Fatalf("expected project name from metadata, got %#v", got)
	}
}

func TestResolveBuildConfigExplicitFlagsOverrideMetadata(t *testing.T) {
	tempApp := t.TempDir()
	explicitRoot := filepath.Join(tempApp, "custom-root")
	explicitAppDir := filepath.Join(tempApp, "custom-app")
	explicitOut := filepath.Join(tempApp, "explicit", "bundle.wasm")
	if err := os.MkdirAll(explicitRoot, 0755); err != nil {
		t.Fatalf("mkdir explicit root: %v", err)
	}
	if err := os.MkdirAll(explicitAppDir, 0755); err != nil {
		t.Fatalf("mkdir explicit app dir: %v", err)
	}
	explicitApp := filepath.Join(explicitAppDir, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write metadata main.go: %v", err)
	}
	if err := os.WriteFile(explicitApp, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write explicit main.go: %v", err)
	}
	metadata := `{
  "projectName": "metadata-build-app",
  "modulePath": "example.com/metadata-build-app",
  "tooling": {
    "appPath": "main.go",
    "wasmPath": "out/app.wasm",
    "defaultBuildProfile": "ci"
  }
}
`
	if err := os.WriteFile(filepath.Join(tempApp, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	originalGetwd := buildGetwd
	t.Cleanup(func() { buildGetwd = originalGetwd })
	buildGetwd = func() (string, error) { return tempApp, nil }

	config, err := resolveBuildConfig(buildConfig{
		appPath:    explicitApp,
		rootPath:   explicitRoot,
		outputPath: explicitOut,
		profile:    "release",
	})
	if err != nil {
		t.Fatalf("resolve build config: %v", err)
	}
	if config.appPath != explicitApp {
		t.Fatalf("expected explicit app path %q, got %#v", explicitApp, config)
	}
	if config.rootPath != explicitRoot {
		t.Fatalf("expected explicit root path %q, got %#v", explicitRoot, config)
	}
	if config.outputPath != explicitOut {
		t.Fatalf("expected explicit output path %q, got %#v", explicitOut, config)
	}
	if config.profile != "release" {
		t.Fatalf("expected explicit profile release, got %#v", config)
	}
}

func TestResolveReleaseConfigExplicitFlagsOverrideMetadata(t *testing.T) {
	tempApp := t.TempDir()
	explicitRoot := filepath.Join(tempApp, "custom-root")
	explicitAppDir := filepath.Join(tempApp, "custom-app")
	explicitOutDir := filepath.Join(tempApp, "explicit-release")
	budgetsPath := filepath.Join(tempApp, "explicit-budgets.json")
	if err := os.MkdirAll(explicitRoot, 0755); err != nil {
		t.Fatalf("mkdir explicit root: %v", err)
	}
	if err := os.MkdirAll(explicitAppDir, 0755); err != nil {
		t.Fatalf("mkdir explicit app dir: %v", err)
	}
	explicitApp := filepath.Join(explicitAppDir, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write metadata main.go: %v", err)
	}
	if err := os.WriteFile(explicitApp, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write explicit main.go: %v", err)
	}
	if err := os.WriteFile(budgetsPath, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write explicit budgets path: %v", err)
	}
	metadata := `{
  "projectName": "metadata-release-app",
  "modulePath": "example.com/metadata-release-app",
  "tooling": {
    "appPath": "main.go",
		"releaseOutDir": "bin/release",
    "releaseBinaryName": "site.wasm",
    "releaseCompression": "none"
  }
}
`
	if err := os.WriteFile(filepath.Join(tempApp, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	originalGetwd := buildGetwd
	t.Cleanup(func() { buildGetwd = originalGetwd })
	buildGetwd = func() (string, error) { return tempApp, nil }

	config, err := resolveReleaseConfig(releaseConfig{
		appPath:        explicitApp,
		rootPath:       explicitRoot,
		outDir:         explicitOutDir,
		binaryName:     "custom.wasm",
		budgetsPath:    budgetsPath,
		profile:        "benchmark",
		compression:    "brotli",
		compressionSet: true,
	})
	if err != nil {
		t.Fatalf("resolve release config: %v", err)
	}
	if config.appPath != explicitApp {
		t.Fatalf("expected explicit app path %q, got %#v", explicitApp, config)
	}
	if config.rootPath != explicitRoot {
		t.Fatalf("expected explicit root path %q, got %#v", explicitRoot, config)
	}
	if config.outDir != explicitOutDir {
		t.Fatalf("expected explicit out dir %q, got %#v", explicitOutDir, config)
	}
	if config.binaryName != "custom.wasm" {
		t.Fatalf("expected explicit binary name custom.wasm, got %#v", config)
	}
	if config.budgetsPath != budgetsPath {
		t.Fatalf("expected explicit budgets path %q, got %#v", budgetsPath, config)
	}
	if config.profile != "benchmark" {
		t.Fatalf("expected explicit profile benchmark, got %#v", config)
	}
	if config.compression != "brotli" {
		t.Fatalf("expected explicit compression brotli, got %#v", config)
	}
}

func TestResolveDevConfigExplicitFlagsOverrideMetadata(t *testing.T) {
	tempApp := t.TempDir()
	explicitRoot := filepath.Join(tempApp, "custom-root")
	explicitAppDir := filepath.Join(tempApp, "custom-app")
	explicitHTML := filepath.Join(tempApp, "custom-index.html")
	if err := os.MkdirAll(explicitRoot, 0755); err != nil {
		t.Fatalf("mkdir explicit root: %v", err)
	}
	if err := os.MkdirAll(explicitAppDir, 0755); err != nil {
		t.Fatalf("mkdir explicit app dir: %v", err)
	}
	explicitApp := filepath.Join(explicitAppDir, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write metadata main.go: %v", err)
	}
	if err := os.WriteFile(explicitApp, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write explicit main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempApp, "index.html"), []byte("<html></html>\n"), 0644); err != nil {
		t.Fatalf("write metadata index.html: %v", err)
	}
	if err := os.WriteFile(explicitHTML, []byte("<html></html>\n"), 0644); err != nil {
		t.Fatalf("write explicit html: %v", err)
	}
	metadata := `{
  "projectName": "metadata-dev-app",
  "modulePath": "example.com/metadata-dev-app",
  "tooling": {
    "appPath": "main.go",
    "htmlPath": "index.html",
    "wasmPath": "build/app.wasm",
    "devHost": "0.0.0.0",
    "devPort": "8140"
  }
}
`
	if err := os.WriteFile(filepath.Join(tempApp, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	originalGetwd := devGetwd
	t.Cleanup(func() { devGetwd = originalGetwd })
	devGetwd = func() (string, error) { return tempApp, nil }

	launcher := launcher{}
	config, err := launcher.resolveDevConfig(devConfig{
		appPath:  explicitApp,
		rootPath: explicitRoot,
		htmlPath: explicitHTML,
		wasmPath: "custom/app.wasm",
		host:     "127.0.0.1",
		port:     "9001",
	})
	if err != nil {
		t.Fatalf("resolve dev config: %v", err)
	}
	if config.appPath != explicitApp {
		t.Fatalf("expected explicit app path %q, got %#v", explicitApp, config)
	}
	if config.rootPath != explicitRoot {
		t.Fatalf("expected explicit root path %q, got %#v", explicitRoot, config)
	}
	if config.htmlPath != explicitHTML {
		t.Fatalf("expected explicit html path %q, got %#v", explicitHTML, config)
	}
	if config.wasmPath != "custom/app.wasm" {
		t.Fatalf("expected explicit wasm path custom/app.wasm, got %#v", config)
	}
	if config.host != "127.0.0.1" || config.port != "9001" {
		t.Fatalf("expected explicit host/port override, got %#v", config)
	}
}

func TestResolveDevConfigDirectoryAppPathDefaultsAndInvalidMetadata(t *testing.T) {
	t.Run("directory app path uses detected html and default host port", func(t *testing.T) {
		root := t.TempDir()
		appDir := filepath.Join(root, "cmd", "web")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatalf("mkdir app dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDir, "index.html"), []byte("<html></html>\n"), 0644); err != nil {
			t.Fatalf("write index.html: %v", err)
		}

		originalGetwd := devGetwd
		t.Cleanup(func() { devGetwd = originalGetwd })
		devGetwd = func() (string, error) { return root, nil }

		config, err := (launcher{}).resolveDevConfig(devConfig{appPath: appDir})
		if err != nil {
			t.Fatalf("resolve dev config: %v", err)
		}
		if config.rootPath != appDir || config.htmlPath != filepath.Join(appDir, "index.html") || config.host != "127.0.0.1" || config.port != "8080" {
			t.Fatalf("expected directory app path defaults, got %#v", config)
		}
	})

	t.Run("invalid metadata bubbles parse error", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "gwc-start.json"), []byte(`{"tooling":`), 0644); err != nil {
			t.Fatalf("write invalid metadata: %v", err)
		}
		originalGetwd := devGetwd
		t.Cleanup(func() { devGetwd = originalGetwd })
		devGetwd = func() (string, error) { return root, nil }
		if _, err := (launcher{}).resolveDevConfig(devConfig{}); err == nil || !strings.Contains(err.Error(), "parse scaffold metadata") {
			t.Fatalf("expected invalid metadata error, got %v", err)
		}
	})
}

func TestResolveDevConfigBubblesGetwdAndMissingAppErrors(t *testing.T) {
	t.Run("cwd error", func(t *testing.T) {
		originalGetwd := devGetwd
		t.Cleanup(func() { devGetwd = originalGetwd })
		devGetwd = func() (string, error) { return "", errors.New("cwd failed") }

		if _, err := (launcher{}).resolveDevConfig(devConfig{}); err == nil || !strings.Contains(err.Error(), "cwd failed") {
			t.Fatalf("expected cwd error, got %v", err)
		}
	})

	t.Run("missing explicit app path", func(t *testing.T) {
		root := t.TempDir()
		originalGetwd := devGetwd
		t.Cleanup(func() { devGetwd = originalGetwd })
		devGetwd = func() (string, error) { return root, nil }

		_, err := (launcher{}).resolveDevConfig(devConfig{appPath: filepath.Join(root, "missing.go")})
		if err == nil || !strings.Contains(err.Error(), "resolve app path") {
			t.Fatalf("expected missing app path error, got %v", err)
		}
	})
}

func TestDetectAppPathFallsBackToCmdWebMain(t *testing.T) {
	tempApp := t.TempDir()
	cmdWebDir := filepath.Join(tempApp, "cmd", "web")
	if err := os.MkdirAll(cmdWebDir, 0755); err != nil {
		t.Fatalf("mkdir cmd/web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cmdWebDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write cmd/web/main.go: %v", err)
	}

	got, err := detectAppPath(tempApp)
	if err != nil {
		t.Fatalf("detect app path: %v", err)
	}
	if want := filepath.Join(tempApp, "cmd", "web", "main.go"); got != want {
		t.Fatalf("expected detected app path %q, got %q", want, got)
	}
}

func TestDetectAppPathErrorsWithoutKnownEntrypoint(t *testing.T) {
	tempApp := t.TempDir()
	if _, err := detectAppPath(tempApp); err == nil {
		t.Fatal("expected detectAppPath to fail when no supported entrypoint exists")
	}
}
