package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveReleaseConfigPrefersScaffoldMetadata(t *testing.T) {
	tempApp := t.TempDir()
	budgetsPath := filepath.Join(tempApp, "config", "release-budgets.json")
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	metadata := `{
  "projectName": "metadata-release-app",
  "modulePath": "example.com/metadata-release-app",
  "tooling": {
    "appPath": "main.go",
	"releaseOutDir": "bin/release",
	    "releaseBinaryName": "site.wasm",
	    "releaseCompression": "none",
	    "releaseBudgetsPath": "config/release-budgets.json"
  }
}
`
	if err := os.WriteFile(filepath.Join(tempApp, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	originalGetwd := buildGetwd
	t.Cleanup(func() { buildGetwd = originalGetwd })
	buildGetwd = func() (string, error) { return tempApp, nil }

	config, err := resolveReleaseConfig(releaseConfig{})
	if err != nil {
		t.Fatalf("resolve release config: %v", err)
	}
	if config.appPath != filepath.Join(tempApp, "main.go") {
		t.Fatalf("expected metadata app path, got %#v", config)
	}
	if config.outDir != filepath.Join(tempApp, "bin", "release") {
		t.Fatalf("expected metadata release out dir, got %#v", config)
	}
	if config.binaryName != "site.wasm" {
		t.Fatalf("expected metadata release binary name, got %#v", config)
	}
	if config.budgetsPath != budgetsPath {
		t.Fatalf("expected metadata release budgets path, got %#v", config)
	}
	if !config.skipCompression {
		t.Fatalf("expected metadata release compression policy to disable compression, got %#v", config)
	}
	if config.compression != "none" {
		t.Fatalf("expected metadata release compression policy none, got %#v", config)
	}
	if config.profile != "release" {
		t.Fatalf("expected release profile default, got %#v", config)
	}
}

func TestResolveReleaseConfigUsesArtifactRootOverride(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
		t.Fatalf("write gwc-runner.json: %v", err)
	}

	originalGetwd := buildGetwd
	t.Cleanup(func() { buildGetwd = originalGetwd })
	buildGetwd = func() (string, error) { return root, nil }

	config, err := resolveReleaseConfig(releaseConfig{})
	if err != nil {
		t.Fatalf("resolve release config: %v", err)
	}
	want := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "wasm-release")
	if config.outDir != want {
		t.Fatalf("expected artifact-root release out dir %q, got %#v", want, config)
	}
}

func TestRunReleaseHandlesHelpAndInvalidFlags(t *testing.T) {
	if err := (launcher{}).runRelease([]string{"-help"}); err != nil {
		t.Fatalf("expected release help to succeed, got %v", err)
	}
	if err := (launcher{}).runRelease([]string{"-definitely-invalid"}); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid release flag error, got %v", err)
	}
}

func TestRunReleaseJSONBuildsManifestAndCompressedSidecars(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	outDir := filepath.Join(tempApp, "dist", "release")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcreleasetest\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"release", "-app", mainPath, "-root", tempApp, "-out-dir", outDir, "-binary-name", "app.wasm", "-json"}); err != nil {
		t.Fatalf("run release: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary releaseSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal release summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful release summary, got %#v", summary)
	}
	wasmArtifact, ok := summary.Artifacts["wasm"]
	if !ok || wasmArtifact.Bytes <= 0 {
		t.Fatalf("expected wasm artifact in release summary, got %#v", summary)
	}
	gzipArtifact, ok := summary.Artifacts["gzip"]
	if !ok || gzipArtifact.Bytes <= 0 {
		t.Fatalf("expected gzip artifact in release summary, got %#v", summary)
	}
	brotliArtifact, ok := summary.Artifacts["brotli"]
	if !ok || brotliArtifact.Bytes <= 0 {
		t.Fatalf("expected brotli artifact in release summary, got %#v", summary)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm")); err != nil {
		t.Fatalf("expected raw release artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm.gz")); err != nil {
		t.Fatalf("expected gzip release artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm.br")); err != nil {
		t.Fatalf("expected brotli release artifact: %v", err)
	}
	manifestPath := filepath.Join(outDir, "wasm-release-manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read release manifest: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("unmarshal release manifest: %v\n%s", err, string(manifestBytes))
	}
	if manifest["profile"] != "release" {
		t.Fatalf("expected release manifest profile, got %#v", manifest)
	}
	flags, ok := manifest["flags"].(map[string]interface{})
	if !ok || flags["brotli"] != true {
		t.Fatalf("expected release manifest to report brotli packaging, got %#v", manifest)
	}
	if flags["compressionPolicy"] != "gzip+brotli" {
		t.Fatalf("expected release manifest compression policy gzip+brotli, got %#v", manifest)
	}
}

func TestRunReleaseCompressionBrotliOnly(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	outDir := filepath.Join(tempApp, "dist", "release")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcreleasebrotli\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"release", "-app", mainPath, "-root", tempApp, "-out-dir", outDir, "-binary-name", "app.wasm", "-compression", "brotli", "-json"}); err != nil {
		t.Fatalf("run release: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary releaseSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal release summary: %v\n%s", err, output)
	}
	if _, ok := summary.Artifacts["brotli"]; !ok {
		t.Fatalf("expected brotli artifact in release summary, got %#v", summary)
	}
	if _, ok := summary.Artifacts["gzip"]; ok {
		t.Fatalf("expected gzip artifact to be omitted for brotli-only release, got %#v", summary)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm.br")); err != nil {
		t.Fatalf("expected brotli release artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm.gz")); !os.IsNotExist(err) {
		t.Fatalf("expected gzip artifact to be absent for brotli-only release, stat err=%v", err)
	}
	if summary.Flags["compressionPolicy"] != "brotli" {
		t.Fatalf("expected compression policy brotli, got %#v", summary)
	}
}

func TestRunReleaseRejectsConflictingCompressionFlags(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	launcher := launcher{}
	err := launcher.run([]string{"release", "-app", mainPath, "-root", tempApp, "-compression", "brotli", "-skip-compression"})
	if err == nil {
		t.Fatal("expected conflicting compression flags to fail")
	}
	if !strings.Contains(err.Error(), "use either -compression or -skip-compression") {
		t.Fatalf("expected conflicting compression flag error, got %v", err)
	}
}

func TestLoadReleaseBudgetsParsesNumericValues(t *testing.T) {
	budgetsPath := filepath.Join(t.TempDir(), "budgets.json")
	if err := os.WriteFile(budgetsPath, []byte(`{"raw_bytes":1234,"gzip_bytes":567,"brotli_bytes":345}`), 0644); err != nil {
		t.Fatalf("write budgets file: %v", err)
	}

	budgets, err := loadReleaseBudgets(budgetsPath)
	if err != nil {
		t.Fatalf("load release budgets: %v", err)
	}
	if budgets["raw_bytes"] != 1234 || budgets["gzip_bytes"] != 567 || budgets["brotli_bytes"] != 345 {
		t.Fatalf("unexpected budgets map: %#v", budgets)
	}
}

func TestLoadReleaseBudgetsRejectsInvalidFiles(t *testing.T) {
	t.Run("parse error", func(t *testing.T) {
		budgetsPath := filepath.Join(t.TempDir(), "budgets.json")
		if err := os.WriteFile(budgetsPath, []byte(`{"raw_bytes":`), 0644); err != nil {
			t.Fatalf("write invalid budgets file: %v", err)
		}
		if _, err := loadReleaseBudgets(budgetsPath); err == nil || !strings.Contains(err.Error(), "parse budgets file") {
			t.Fatalf("expected parse error, got %v", err)
		}
	})

	t.Run("non numeric", func(t *testing.T) {
		budgetsPath := filepath.Join(t.TempDir(), "budgets.json")
		if err := os.WriteFile(budgetsPath, []byte(`{"raw_bytes":"huge"}`), 0644); err != nil {
			t.Fatalf("write nonnumeric budgets file: %v", err)
		}
		if _, err := loadReleaseBudgets(budgetsPath); err == nil || !strings.Contains(err.Error(), `budget "raw_bytes" must be numeric`) {
			t.Fatalf("expected numeric validation error, got %v", err)
		}
	})
}

func TestAssertReleaseBudgetsPassesAndFails(t *testing.T) {
	artifacts := map[string]releaseArtifactRecord{
		"wasm":   {Path: "app.wasm", Bytes: 100},
		"gzip":   {Path: "app.wasm.gz", Bytes: 50},
		"brotli": {Path: "app.wasm.br", Bytes: 40},
	}

	if err := assertReleaseBudgets(map[string]int64{"raw_bytes": 100, "gzip_bytes": 50, "brotli_bytes": 40}, artifacts); err != nil {
		t.Fatalf("expected matching budgets to pass, got %v", err)
	}
	if err := assertReleaseBudgets(map[string]int64{"raw_bytes": 99}, artifacts); err == nil || !strings.Contains(err.Error(), "artifact budget exceeded for raw wasm") {
		t.Fatalf("expected raw wasm budget failure, got %v", err)
	}
	if err := assertReleaseBudgets(map[string]int64{"gzip_bytes": 49}, artifacts); err == nil || !strings.Contains(err.Error(), "artifact budget exceeded for gzip sidecar") {
		t.Fatalf("expected gzip budget failure, got %v", err)
	}
	if err := assertReleaseBudgets(map[string]int64{"brotli_bytes": 39}, artifacts); err == nil || !strings.Contains(err.Error(), "artifact budget exceeded for brotli sidecar") {
		t.Fatalf("expected brotli budget failure, got %v", err)
	}
	if err := assertReleaseBudgets(map[string]int64{"gzip_bytes": 1}, map[string]releaseArtifactRecord{"wasm": {Path: "app.wasm", Bytes: 100}}); err != nil {
		t.Fatalf("expected missing artifact budget check to be skipped, got %v", err)
	}
}

func TestRunReleaseHelpReturnsNil(t *testing.T) {
	launcher := launcher{}
	if err := launcher.run([]string{"release", "-help"}); err != nil {
		t.Fatalf("expected release help to succeed, got %v", err)
	}
}

func TestRunReleasePrintsSummaryWithoutJSON(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	outDir := filepath.Join(tempApp, "dist", "release")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcreleasetext\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).run([]string{"release", "-app", mainPath, "-root", tempApp, "-out-dir", outDir, "-compression", "none"}); err != nil {
		t.Fatalf("run release: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	for _, expected := range []string{"GWC release", "out dir:      " + outDir, "artifact[wasm]: app.wasm"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected release output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestRunReleaseSkipCompressionProducesOnlyRawArtifact(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	outDir := filepath.Join(tempApp, "dist", "release")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcreleasenone\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"release", "-app", mainPath, "-root", tempApp, "-out-dir", outDir, "-skip-compression", "-json"}); err != nil {
		t.Fatalf("run release with skip-compression: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary releaseSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal release summary: %v\n%s", err, output)
	}
	if _, ok := summary.Artifacts["wasm"]; !ok {
		t.Fatalf("expected raw wasm artifact, got %#v", summary)
	}
	if _, ok := summary.Artifacts["gzip"]; ok {
		t.Fatalf("expected gzip artifact to be omitted, got %#v", summary)
	}
	if _, ok := summary.Artifacts["brotli"]; ok {
		t.Fatalf("expected brotli artifact to be omitted, got %#v", summary)
	}
	if summary.Flags["compressionPolicy"] != "none" || summary.Flags["compression"] != false {
		t.Fatalf("expected no-compression flags, got %#v", summary.Flags)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm.gz")); !os.IsNotExist(err) {
		t.Fatalf("expected gzip sidecar to be absent, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "app.wasm.br")); !os.IsNotExist(err) {
		t.Fatalf("expected brotli sidecar to be absent, stat err=%v", err)
	}
}

func TestResolveReleaseConfigRejectsInvalidNames(t *testing.T) {
	tempApp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	originalGetwd := buildGetwd
	t.Cleanup(func() { buildGetwd = originalGetwd })
	buildGetwd = func() (string, error) { return tempApp, nil }

	_, err := resolveReleaseConfig(releaseConfig{binaryName: "."})
	if err == nil || !strings.Contains(err.Error(), "release binary name is required") {
		t.Fatalf("expected invalid binary name error, got %v", err)
	}

	_, err = resolveReleaseConfig(releaseConfig{manifestName: "."})
	if err == nil || !strings.Contains(err.Error(), "release manifest name is required") {
		t.Fatalf("expected invalid manifest name error, got %v", err)
	}
}

func TestResolveReleaseConfigDirectoryAppPathAndInvalidMetadata(t *testing.T) {
	t.Run("directory app path uses app directory defaults", func(t *testing.T) {
		root := t.TempDir()
		appDir := filepath.Join(root, "cmd", "web")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatalf("mkdir app dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}

		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return root, nil }

		config, err := resolveReleaseConfig(releaseConfig{appPath: appDir, compression: "gzip", compressionSet: true})
		if err != nil {
			t.Fatalf("resolve release config: %v", err)
		}
		if config.rootPath != appDir || config.outDir != filepath.Join(appDir, "bin", "wasm-release") || config.binaryName != "app.wasm" {
			t.Fatalf("expected directory app path defaults, got %#v", config)
		}
	})

	t.Run("invalid metadata bubbles parse error", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "gwc-start.json"), []byte(`{"tooling":`), 0644); err != nil {
			t.Fatalf("write invalid metadata: %v", err)
		}
		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return root, nil }
		if _, err := resolveReleaseConfig(releaseConfig{}); err == nil || !strings.Contains(err.Error(), "parse scaffold metadata") {
			t.Fatalf("expected invalid metadata error, got %v", err)
		}
	})
}

func TestResolveReleaseConfigRejectsInvalidMetadataCompressionAndErrors(t *testing.T) {
	t.Run("invalid metadata compression policy", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		metadata := `{
  "projectName": "metadata-release-app",
  "modulePath": "example.com/metadata-release-app",
  "tooling": {
    "appPath": "main.go",
    "releaseCompression": "mystery"
  }
}
`
		if err := os.WriteFile(filepath.Join(root, "gwc-start.json"), []byte(metadata), 0644); err != nil {
			t.Fatalf("write gwc-start.json: %v", err)
		}
		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return root, nil }

		if _, err := resolveReleaseConfig(releaseConfig{}); err == nil || !strings.Contains(err.Error(), "unknown release compression policy") {
			t.Fatalf("expected metadata compression error, got %v", err)
		}
	})

	t.Run("cwd error", func(t *testing.T) {
		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return "", errors.New("cwd failed") }

		if _, err := resolveReleaseConfig(releaseConfig{}); err == nil || !strings.Contains(err.Error(), "cwd failed") {
			t.Fatalf("expected cwd error, got %v", err)
		}
	})

	t.Run("missing explicit app path", func(t *testing.T) {
		root := t.TempDir()
		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return root, nil }

		_, err := resolveReleaseConfig(releaseConfig{appPath: filepath.Join(root, "missing.go")})
		if err == nil || !strings.Contains(err.Error(), "resolve app path") {
			t.Fatalf("expected missing app path error, got %v", err)
		}
	})
}

func TestNormalizeReleaseCompressionPolicyAliases(t *testing.T) {
	tests := map[string]string{
		"":            "gzip",
		"gzip":        "gzip",
		"br":          "brotli",
		"both":        "gzip+brotli",
		"off":         "none",
		"disabled":    "none",
		"brotli+gzip": "gzip+brotli",
	}
	for input, want := range tests {
		got, err := normalizeReleaseCompressionPolicy(input)
		if err != nil {
			t.Fatalf("normalize compression %q: %v", input, err)
		}
		if got != want {
			t.Fatalf("expected compression %q -> %q, got %q", input, want, got)
		}
	}
	if _, err := normalizeReleaseCompressionPolicy("mystery"); err == nil {
		t.Fatal("expected unknown compression policy to fail")
	}
}

func TestReleaseArtifactAndSidecarHelpers(t *testing.T) {
	root := t.TempDir()
	wasmPath := filepath.Join(root, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte("wasm-bytes"), 0644); err != nil {
		t.Fatalf("write wasm artifact: %v", err)
	}
	record, err := releaseArtifactRecordForPath(root, wasmPath)
	if err != nil {
		t.Fatalf("release artifact record: %v", err)
	}
	if record.Path != "app.wasm" || record.Bytes <= 0 || len(record.SHA256) != 64 {
		t.Fatalf("unexpected release artifact record: %#v", record)
	}

	gzipPath := wasmPath + ".gz"
	if err := writeGzipSidecar(wasmPath, gzipPath); err != nil {
		t.Fatalf("write gzip sidecar: %v", err)
	}
	if info, err := os.Stat(gzipPath); err != nil || info.Size() <= 0 {
		t.Fatalf("expected gzip sidecar to exist, stat err=%v info=%v", err, info)
	}

	brotliPath := wasmPath + ".br"
	if err := writeBrotliSidecar(wasmPath, brotliPath); err != nil {
		t.Fatalf("write brotli sidecar: %v", err)
	}
	if info, err := os.Stat(brotliPath); err != nil || info.Size() <= 0 {
		t.Fatalf("expected brotli sidecar to exist, stat err=%v info=%v", err, info)
	}

	if _, err := releaseArtifactRecordForPath(root, filepath.Join(root, "missing.wasm")); err == nil || !strings.Contains(err.Error(), "read release artifact") {
		t.Fatalf("expected missing artifact read error, got %v", err)
	}
	if err := writeGzipSidecar(filepath.Join(root, "missing.wasm"), filepath.Join(root, "missing.gz")); err == nil || !strings.Contains(err.Error(), "read source artifact for gzip") {
		t.Fatalf("expected gzip read error, got %v", err)
	}
	if err := writeBrotliSidecar(filepath.Join(root, "missing.wasm"), filepath.Join(root, "missing.br")); err == nil || !strings.Contains(err.Error(), "read source artifact for brotli") {
		t.Fatalf("expected brotli read error, got %v", err)
	}
}

func TestReleaseSidecarCreateErrorsAndGzipOnlyRelease(t *testing.T) {
	root := t.TempDir()
	wasmPath := filepath.Join(root, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte("wasm-bytes"), 0644); err != nil {
		t.Fatalf("write wasm artifact: %v", err)
	}
	if err := writeGzipSidecar(wasmPath, filepath.Join(root, "missing-dir", "app.wasm.gz")); err == nil || !strings.Contains(err.Error(), "create gzip sidecar") {
		t.Fatalf("expected gzip sidecar create error, got %v", err)
	}
	if err := writeBrotliSidecar(wasmPath, filepath.Join(root, "missing-dir", "app.wasm.br")); err == nil || !strings.Contains(err.Error(), "create brotli sidecar") {
		t.Fatalf("expected brotli sidecar create error, got %v", err)
	}

	appRoot := t.TempDir()
	mainPath := filepath.Join(appRoot, "main.go")
	if err := os.WriteFile(filepath.Join(appRoot, "go.mod"), []byte("module example.com/gwcreleasegzip\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	summary, err := executeRelease(releaseConfig{appPath: mainPath, rootPath: appRoot, outDir: filepath.Join(appRoot, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", profile: "release", compression: "gzip"})
	if err != nil {
		t.Fatalf("execute gzip-only release: %v", err)
	}
	if _, ok := summary.Artifacts["gzip"]; !ok {
		t.Fatalf("expected gzip artifact, got %#v", summary)
	}
	if _, ok := summary.Artifacts["brotli"]; ok {
		t.Fatalf("expected brotli artifact to be omitted, got %#v", summary)
	}
	if summary.Flags["compressionPolicy"] != "gzip" {
		t.Fatalf("expected gzip compression policy, got %#v", summary.Flags)
	}
}

func TestExecuteReleaseErrorPaths(t *testing.T) {
	originalReleaseExecuteBuild := releaseExecuteBuild
	originalArtifactRecord := releaseArtifactRecordForPathFunc
	originalWriteGzip := releaseWriteGzipSidecar
	originalWriteBrotli := releaseWriteBrotliSidecar
	originalMarshalIndent := releaseMarshalIndent
	t.Cleanup(func() {
		releaseExecuteBuild = originalReleaseExecuteBuild
		releaseArtifactRecordForPathFunc = originalArtifactRecord
		releaseWriteGzipSidecar = originalWriteGzip
		releaseWriteBrotliSidecar = originalWriteBrotli
		releaseMarshalIndent = originalMarshalIndent
	})

	t.Run("outdir is file", func(t *testing.T) {
		root := t.TempDir()
		outPath := filepath.Join(root, "release-file")
		if err := os.WriteFile(outPath, []byte("occupied"), 0644); err != nil {
			t.Fatalf("write occupied out path: %v", err)
		}
		_, err := executeRelease(releaseConfig{outDir: outPath})
		if err == nil || !strings.Contains(err.Error(), "create release output directory") {
			t.Fatalf("expected out dir creation error, got %v", err)
		}
	})

	t.Run("invalid budgets file", func(t *testing.T) {
		root := t.TempDir()
		mainPath := filepath.Join(root, "main.go")
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gwcreleaseinvalidbudget\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		budgetsPath := filepath.Join(root, "budgets.json")
		if err := os.WriteFile(budgetsPath, []byte(`{"raw_bytes":`), 0644); err != nil {
			t.Fatalf("write budgets: %v", err)
		}
		_, err := executeRelease(releaseConfig{appPath: mainPath, rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", profile: "release", compression: "none", skipCompression: true, budgetsPath: budgetsPath})
		if err == nil || !strings.Contains(err.Error(), "parse budgets file") {
			t.Fatalf("expected invalid budgets error, got %v", err)
		}
	})

	t.Run("artifact budgets exceeded", func(t *testing.T) {
		root := t.TempDir()
		mainPath := filepath.Join(root, "main.go")
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gwcreleasebudgetfail\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		budgetsPath := filepath.Join(root, "budgets.json")
		if err := os.WriteFile(budgetsPath, []byte(`{"raw_bytes":1}`), 0644); err != nil {
			t.Fatalf("write budgets: %v", err)
		}
		_, err := executeRelease(releaseConfig{appPath: mainPath, rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", profile: "release", compression: "none", skipCompression: true, budgetsPath: budgetsPath})
		if err == nil || !strings.Contains(err.Error(), "artifact budget exceeded for raw wasm") {
			t.Fatalf("expected artifact budget failure, got %v", err)
		}
	})

	t.Run("manifest write failure", func(t *testing.T) {
		root := t.TempDir()
		mainPath := filepath.Join(root, "main.go")
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gwcreleasemanifestfail\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		_, err := executeRelease(releaseConfig{appPath: mainPath, rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: ".", profile: "release", compression: "none", skipCompression: true})
		if err == nil || !strings.Contains(err.Error(), "write release manifest") {
			t.Fatalf("expected manifest write failure, got %v", err)
		}
	})

	t.Run("build failure", func(t *testing.T) {
		root := t.TempDir()
		mainPath := filepath.Join(root, "main.go")
		if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		binDir := filepath.Join(t.TempDir(), "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatalf("mkdir fake bin: %v", err)
		}
		writeFakeGoBuildCommand(t, binDir)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("FAKE_GO_MODE", "fail-empty")

		_, err := executeRelease(releaseConfig{appPath: mainPath, rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "go build failed") {
			t.Fatalf("expected build failure, got %v", err)
		}
	})

	t.Run("propagates built artifact read failure from executeBuild", func(t *testing.T) {
		root := t.TempDir()
		mainPath := filepath.Join(root, "main.go")
		if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		binDir := filepath.Join(t.TempDir(), "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatalf("mkdir fake bin: %v", err)
		}
		writeFakeGoBuildCommand(t, binDir)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("FAKE_GO_MODE", "write-dir")

		_, err := executeRelease(releaseConfig{appPath: mainPath, rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "read built wasm artifact") {
			t.Fatalf("expected executeBuild artifact read failure, got %v", err)
		}
	})

	t.Run("raw artifact helper failure", func(t *testing.T) {
		root := t.TempDir()
		releaseExecuteBuild = func(config buildConfig) (buildSummary, error) {
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: config.appPath, ProjectRoot: config.rootPath, PackageDir: root}, nil
		}
		releaseArtifactRecordForPathFunc = func(baseDir string, artifactPath string) (releaseArtifactRecord, error) {
			return releaseArtifactRecord{}, errors.New("artifact failed")
		}
		defer func() {
			releaseExecuteBuild = originalReleaseExecuteBuild
			releaseArtifactRecordForPathFunc = originalArtifactRecord
		}()

		_, err := executeRelease(releaseConfig{appPath: filepath.Join(root, "main.go"), rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "artifact failed") {
			t.Fatalf("expected raw artifact helper failure, got %v", err)
		}
	})

	t.Run("gzip sidecar helper failure", func(t *testing.T) {
		root := t.TempDir()
		releaseExecuteBuild = func(config buildConfig) (buildSummary, error) {
			artifactPath := filepath.Join(config.rootPath, "dist", "app.wasm")
			if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
				return buildSummary{}, err
			}
			if err := os.WriteFile(artifactPath, []byte("wasm"), 0644); err != nil {
				return buildSummary{}, err
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: config.appPath, ProjectRoot: config.rootPath, PackageDir: root}, nil
		}
		releaseWriteGzipSidecar = func(sourcePath string, targetPath string) error { return errors.New("gzip failed") }
		defer func() {
			releaseExecuteBuild = originalReleaseExecuteBuild
			releaseWriteGzipSidecar = originalWriteGzip
		}()

		_, err := executeRelease(releaseConfig{appPath: filepath.Join(root, "main.go"), rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "gzip", profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "gzip failed") {
			t.Fatalf("expected gzip sidecar helper failure, got %v", err)
		}
	})

	t.Run("brotli artifact helper and manifest encoding failures", func(t *testing.T) {
		root := t.TempDir()
		releaseExecuteBuild = func(config buildConfig) (buildSummary, error) {
			artifactPath := filepath.Join(config.rootPath, "dist", "app.wasm")
			if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
				return buildSummary{}, err
			}
			if err := os.WriteFile(artifactPath, []byte("wasm"), 0644); err != nil {
				return buildSummary{}, err
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: config.appPath, ProjectRoot: config.rootPath, PackageDir: root}, nil
		}
		callCount := 0
		releaseArtifactRecordForPathFunc = func(baseDir string, artifactPath string) (releaseArtifactRecord, error) {
			callCount++
			if callCount == 2 {
				return releaseArtifactRecord{}, errors.New("brotli artifact failed")
			}
			return releaseArtifactRecord{Path: filepath.Base(artifactPath), Bytes: 4, SHA256: strings.Repeat("a", 64)}, nil
		}
		defer func() {
			releaseExecuteBuild = originalReleaseExecuteBuild
			releaseArtifactRecordForPathFunc = originalArtifactRecord
			releaseMarshalIndent = originalMarshalIndent
		}()

		_, err := executeRelease(releaseConfig{appPath: filepath.Join(root, "main.go"), rootPath: root, outDir: filepath.Join(root, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "brotli", profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "brotli artifact failed") {
			t.Fatalf("expected brotli artifact helper failure, got %v", err)
		}

		releaseArtifactRecordForPathFunc = func(baseDir string, artifactPath string) (releaseArtifactRecord, error) {
			return releaseArtifactRecord{Path: filepath.Base(artifactPath), Bytes: 4, SHA256: strings.Repeat("a", 64)}, nil
		}
		releaseMarshalIndent = func(v interface{}, prefix string, indent string) ([]byte, error) {
			return nil, errors.New("marshal failed")
		}
		_, err = executeRelease(releaseConfig{appPath: filepath.Join(root, "main.go"), rootPath: root, outDir: filepath.Join(root, "dist-2"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "encode release manifest") {
			t.Fatalf("expected manifest encoding failure, got %v", err)
		}
	})
}
