package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveScaffoldMetadataForConfigUsesConfiguredAppDirectory(parseT *testing.T) {
	parseWorkingDir := parseT.TempDir()
	parseProjectDir := parseT.TempDir()
	parseAppPath := filepath.Join(parseProjectDir, "main.go")
	if parseErr := os.WriteFile(parseAppPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	parseMetadata := `{
  "projectName": "configured-app-metadata",
  "modulePath": "example.com/configured-app-metadata",
  "tooling": {
    "appPath": "main.go"
  }
}
`
	if parseErr2 := os.WriteFile(filepath.Join(parseProjectDir, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr2)
	}

	parseGot, parseMetadataDir, parseOk, parseErr3 := resolveScaffoldMetadataForConfig(parseWorkingDir, "", parseAppPath)
	if parseErr3 != nil {
		parseT.Fatalf("resolve scaffold metadata: %v", parseErr3)
	}
	if !parseOk {
		parseT.Fatal("expected metadata to be discovered from the configured app directory")
	}
	if parseMetadataDir != parseProjectDir {
		parseT.Fatalf("expected metadata dir %q, got %q", parseProjectDir, parseMetadataDir)
	}
	if parseGot.ProjectName != "configured-app-metadata" {
		parseT.Fatalf("expected project name from metadata, got %#v", parseGot)
	}
}

func TestResolveScaffoldMetadataForConfigUsesRelativeConfiguredRoot(parseT *testing.T) {
	parseWorkingDir := parseT.TempDir()
	parseProjectDir := filepath.Join(parseWorkingDir, "project")
	if parseErr := os.MkdirAll(parseProjectDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir project dir: %v", parseErr)
	}
	parseMetadata := `{
  "projectName": "relative-root-metadata",
  "modulePath": "example.com/relative-root-metadata"
}
`
	if parseErr2 := os.WriteFile(filepath.Join(parseProjectDir, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr2)
	}

	parseGot, parseMetadataDir, parseOk, parseErr3 := resolveScaffoldMetadataForConfig(parseWorkingDir, "project", "")
	if parseErr3 != nil {
		parseT.Fatalf("resolve scaffold metadata: %v", parseErr3)
	}
	if !parseOk {
		parseT.Fatal("expected metadata to be discovered from the configured relative root")
	}
	if parseMetadataDir != parseProjectDir {
		parseT.Fatalf("expected metadata dir %q, got %q", parseProjectDir, parseMetadataDir)
	}
	if parseGot.ProjectName != "relative-root-metadata" {
		parseT.Fatalf("expected project name from metadata, got %#v", parseGot)
	}
}

func TestResolveBuildConfigExplicitFlagsOverrideMetadata(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseExplicitRoot := filepath.Join(parseTempApp, "custom-root")
	parseExplicitAppDir := filepath.Join(parseTempApp, "custom-app")
	parseExplicitOut := filepath.Join(parseTempApp, "explicit", "bundle.wasm")
	if parseErr := os.MkdirAll(parseExplicitRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir explicit root: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseExplicitAppDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir explicit app dir: %v", parseErr2)
	}
	parseExplicitApp := filepath.Join(parseExplicitAppDir, "main.go")
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write metadata main.go: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseExplicitApp, []byte("package main\nfunc main() {}\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write explicit main.go: %v", parseErr4)
	}
	parseMetadata := `{
  "projectName": "metadata-build-app",
  "modulePath": "example.com/metadata-build-app",
  "tooling": {
    "appPath": "main.go",
    "wasmPath": "out/app.wasm",
    "defaultBuildProfile": "ci"
  }
}
`
	if parseErr5 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr5 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr5)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	parseConfig, parseErr6 := resolveBuildConfig(buildConfig{
		appPath:    parseExplicitApp,
		rootPath:   parseExplicitRoot,
		outputPath: parseExplicitOut,
		profile:    "release",
	})
	if parseErr6 != nil {
		parseT.Fatalf("resolve build config: %v", parseErr6)
	}
	if parseConfig.appPath != parseExplicitApp {
		parseT.Fatalf("expected explicit app path %q, got %#v", parseExplicitApp, parseConfig)
	}
	if parseConfig.rootPath != parseExplicitRoot {
		parseT.Fatalf("expected explicit root path %q, got %#v", parseExplicitRoot, parseConfig)
	}
	if parseConfig.outputPath != parseExplicitOut {
		parseT.Fatalf("expected explicit output path %q, got %#v", parseExplicitOut, parseConfig)
	}
	if parseConfig.profile != "release" {
		parseT.Fatalf("expected explicit profile release, got %#v", parseConfig)
	}
}

func TestResolveReleaseConfigExplicitFlagsOverrideMetadata(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseExplicitRoot := filepath.Join(parseTempApp, "custom-root")
	parseExplicitAppDir := filepath.Join(parseTempApp, "custom-app")
	parseExplicitOutDir := filepath.Join(parseTempApp, "explicit-release")
	parseBudgetsPath := filepath.Join(parseTempApp, "explicit-budgets.json")
	if parseErr := os.MkdirAll(parseExplicitRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir explicit root: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseExplicitAppDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir explicit app dir: %v", parseErr2)
	}
	parseExplicitApp := filepath.Join(parseExplicitAppDir, "main.go")
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write metadata main.go: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseExplicitApp, []byte("package main\nfunc main() {}\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write explicit main.go: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseBudgetsPath, []byte("{}\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write explicit budgets path: %v", parseErr5)
	}
	parseMetadata := `{
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
	if parseErr6 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr6 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr6)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	parseConfig, parseErr7 := resolveReleaseConfig(releaseConfig{
		appPath:        parseExplicitApp,
		rootPath:       parseExplicitRoot,
		outDir:         parseExplicitOutDir,
		binaryName:     "custom.wasm",
		budgetsPath:    parseBudgetsPath,
		profile:        "benchmark",
		compression:    "brotli",
		compressionSet: true,
	})
	if parseErr7 != nil {
		parseT.Fatalf("resolve release config: %v", parseErr7)
	}
	if parseConfig.appPath != parseExplicitApp {
		parseT.Fatalf("expected explicit app path %q, got %#v", parseExplicitApp, parseConfig)
	}
	if parseConfig.rootPath != parseExplicitRoot {
		parseT.Fatalf("expected explicit root path %q, got %#v", parseExplicitRoot, parseConfig)
	}
	if parseConfig.outDir != parseExplicitOutDir {
		parseT.Fatalf("expected explicit out dir %q, got %#v", parseExplicitOutDir, parseConfig)
	}
	if parseConfig.binaryName != "custom.wasm" {
		parseT.Fatalf("expected explicit binary name custom.wasm, got %#v", parseConfig)
	}
	if parseConfig.budgetsPath != parseBudgetsPath {
		parseT.Fatalf("expected explicit budgets path %q, got %#v", parseBudgetsPath, parseConfig)
	}
	if parseConfig.profile != "benchmark" {
		parseT.Fatalf("expected explicit profile benchmark, got %#v", parseConfig)
	}
	if parseConfig.compression != "brotli" {
		parseT.Fatalf("expected explicit compression brotli, got %#v", parseConfig)
	}
}

func TestResolveDevConfigExplicitFlagsOverrideMetadata(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseExplicitRoot := filepath.Join(parseTempApp, "custom-root")
	parseExplicitAppDir := filepath.Join(parseTempApp, "custom-app")
	parseExplicitHTML := filepath.Join(parseTempApp, "custom-index.html")
	if parseErr := os.MkdirAll(parseExplicitRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir explicit root: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseExplicitAppDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir explicit app dir: %v", parseErr2)
	}
	parseExplicitApp := filepath.Join(parseExplicitAppDir, "main.go")
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write metadata main.go: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseExplicitApp, []byte("package main\nfunc main() {}\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write explicit main.go: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<html></html>\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write metadata index.html: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(parseExplicitHTML, []byte("<html></html>\n"), 0644); parseErr6 != nil {
		parseT.Fatalf("write explicit html: %v", parseErr6)
	}
	parseMetadata := `{
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
	if parseErr7 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr7 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr7)
	}

	parseOriginalGetwd := devGetwd
	parseT.Cleanup(func() { devGetwd = parseOriginalGetwd })
	devGetwd = func() (string, error) { return parseTempApp, nil }

	parseLauncher := launcher{}
	parseConfig, parseErr8 := parseLauncher.resolveDevConfig(devConfig{
		appPath:  parseExplicitApp,
		rootPath: parseExplicitRoot,
		htmlPath: parseExplicitHTML,
		wasmPath: "custom/app.wasm",
		host:     "127.0.0.1",
		port:     "9001",
	})
	if parseErr8 != nil {
		parseT.Fatalf("resolve dev config: %v", parseErr8)
	}
	if parseConfig.appPath != parseExplicitApp {
		parseT.Fatalf("expected explicit app path %q, got %#v", parseExplicitApp, parseConfig)
	}
	if parseConfig.rootPath != parseExplicitRoot {
		parseT.Fatalf("expected explicit root path %q, got %#v", parseExplicitRoot, parseConfig)
	}
	if parseConfig.htmlPath != parseExplicitHTML {
		parseT.Fatalf("expected explicit html path %q, got %#v", parseExplicitHTML, parseConfig)
	}
	if parseConfig.wasmPath != "custom/app.wasm" {
		parseT.Fatalf("expected explicit wasm path custom/app.wasm, got %#v", parseConfig)
	}
	if parseConfig.host != "127.0.0.1" || parseConfig.port != "9001" {
		parseT.Fatalf("expected explicit host/port override, got %#v", parseConfig)
	}
}

func TestResolveDevConfigDirectoryAppPathDefaultsAndInvalidMetadata(parseT *testing.T) {
	parseT.Run("directory app path uses detected html and default host port", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseAppDir := filepath.Join(parseRoot, "cmd", "web")
		if parseErr := os.MkdirAll(parseAppDir, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir app dir: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(filepath.Join(parseAppDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
			parseT2.Fatalf("write main.go: %v", parseErr2)
		}
		if parseErr3 := os.WriteFile(filepath.Join(parseAppDir, "index.html"), []byte("<html></html>\n"), 0644); parseErr3 != nil {
			parseT2.Fatalf("write index.html: %v", parseErr3)
		}

		parseOriginalGetwd := devGetwd
		parseT2.Cleanup(func() { devGetwd = parseOriginalGetwd })
		devGetwd = func() (string, error) { return parseRoot, nil }

		parseConfig, parseErr4 := (launcher{}).resolveDevConfig(devConfig{appPath: parseAppDir})
		if parseErr4 != nil {
			parseT2.Fatalf("resolve dev config: %v", parseErr4)
		}
		if parseConfig.rootPath != parseAppDir || parseConfig.htmlPath != filepath.Join(parseAppDir, "index.html") || parseConfig.host != "127.0.0.1" || parseConfig.port != "8080" {
			parseT2.Fatalf("expected directory app path defaults, got %#v", parseConfig)
		}
	})

	parseT.Run("invalid metadata bubbles parse error", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		if parseErr5 := os.WriteFile(filepath.Join(parseRoot2, "gwc-start.json"), []byte(`{"tooling":`), 0644); parseErr5 != nil {
			parseT3.Fatalf("write invalid metadata: %v", parseErr5)
		}
		parseOriginalGetwd2 := devGetwd
		parseT3.Cleanup(func() { devGetwd = parseOriginalGetwd2 })
		devGetwd = func() (string, error) { return parseRoot2, nil }
		if _, parseErr6 := (launcher{}).resolveDevConfig(devConfig{}); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "parse scaffold metadata") {
			parseT3.Fatalf("expected invalid metadata error, got %v", parseErr6)
		}
	})
}

func TestResolveDevConfigBubblesGetwdAndMissingAppErrors(parseT *testing.T) {
	parseT.Run("cwd error", func(parseT2 *testing.T) {
		parseOriginalGetwd := devGetwd
		parseT2.Cleanup(func() { devGetwd = parseOriginalGetwd })
		devGetwd = func() (string, error) { return "", errors.New("cwd failed") }

		if _, parseErr := (launcher{}).resolveDevConfig(devConfig{}); parseErr == nil || !strings.Contains(parseErr.Error(), "cwd failed") {
			parseT2.Fatalf("expected cwd error, got %v", parseErr)
		}
	})

	parseT.Run("missing explicit app path", func(parseT3 *testing.T) {
		parseRoot := parseT3.TempDir()
		parseOriginalGetwd2 := devGetwd
		parseT3.Cleanup(func() { devGetwd = parseOriginalGetwd2 })
		devGetwd = func() (string, error) { return parseRoot, nil }

		_, parseErr2 := (launcher{}).resolveDevConfig(devConfig{appPath: filepath.Join(parseRoot, "missing.go")})
		if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "resolve app path") {
			parseT3.Fatalf("expected missing app path error, got %v", parseErr2)
		}
	})
}

func TestDetectAppPathFallsBackToCmdWebMain(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseCmdWebDir := filepath.Join(parseTempApp, "cmd", "web")
	if parseErr := os.MkdirAll(parseCmdWebDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir cmd/web: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseCmdWebDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write cmd/web/main.go: %v", parseErr2)
	}

	parseGot, parseErr3 := detectAppPath(parseTempApp)
	if parseErr3 != nil {
		parseT.Fatalf("detect app path: %v", parseErr3)
	}
	if parseWant := filepath.Join(parseTempApp, "cmd", "web", "main.go"); parseGot != parseWant {
		parseT.Fatalf("expected detected app path %q, got %q", parseWant, parseGot)
	}
}

func TestDetectAppPathErrorsWithoutKnownEntrypoint(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	if _, parseErr := detectAppPath(parseTempApp); parseErr == nil {
		parseT.Fatal("expected detectAppPath to fail when no supported entrypoint exists")
	}
}
