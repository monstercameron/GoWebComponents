package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeGoBuildCommand(t *testing.T, binDir string) {
	t.Helper()
	script := "@echo off\r\n" +
		"if /I not \"%1\"==\"build\" exit /b 0\r\n" +
		"if /I not \"%2\"==\"-o\" exit /b 0\r\n" +
		"if /I \"%FAKE_GO_MODE%\"==\"fail-empty\" exit /b 7\r\n" +
		"if /I \"%FAKE_GO_MODE%\"==\"fail-text\" (\r\n" +
		"  echo fake build failed 1>&2\r\n" +
		"  exit /b 7\r\n" +
		")\r\n" +
		"if /I \"%FAKE_GO_MODE%\"==\"write-dir\" (\r\n" +
		"  mkdir \"%3\" >nul 2>nul\r\n" +
		"  exit /b 0\r\n" +
		")\r\n" +
		"if /I \"%FAKE_GO_MODE%\"==\"write-file\" (\r\n" +
		"  if not exist \"%~dp3\" mkdir \"%~dp3\" >nul 2>nul\r\n" +
		"  >\"%3\" echo wasm\r\n" +
		"  exit /b 0\r\n" +
		")\r\n" +
		"exit /b 0\r\n"
	if err := os.WriteFile(filepath.Join(binDir, "go.bat"), []byte(script), 0644); err != nil {
		t.Fatalf("write fake go.bat: %v", err)
	}
}

func TestResolveBuildProfileAliases(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		trimpath  bool
		wantFlags string
	}{
		{input: "development", wantName: "development", trimpath: false, wantFlags: ""},
		{input: "dev", wantName: "development", trimpath: false, wantFlags: ""},
		{input: "ci", wantName: "ci", trimpath: true, wantFlags: "-s -w"},
		{input: "bench", wantName: "benchmark", trimpath: true, wantFlags: "-s -w"},
		{input: "prod", wantName: "release", trimpath: true, wantFlags: "-s -w"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			profile, err := resolveBuildProfile(test.input)
			if err != nil {
				t.Fatalf("resolve build profile: %v", err)
			}
			if profile.Name != test.wantName {
				t.Fatalf("expected profile name %q, got %q", test.wantName, profile.Name)
			}
			if profile.Trimpath != test.trimpath {
				t.Fatalf("expected trimpath %t, got %#v", test.trimpath, profile)
			}
			if profile.Ldflags != test.wantFlags {
				t.Fatalf("expected ldflags %q, got %#v", test.wantFlags, profile)
			}
		})
	}
	if _, err := resolveBuildProfile("mystery"); err == nil {
		t.Fatal("expected unknown build profile to fail")
	}
}

func TestResolveBuildConfigDefaults(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	originalGetwd := buildGetwd
	t.Cleanup(func() { buildGetwd = originalGetwd })
	buildGetwd = func() (string, error) { return tempApp, nil }

	config, err := resolveBuildConfig(buildConfig{})
	if err != nil {
		t.Fatalf("resolve build config: %v", err)
	}
	if config.appPath != mainPath {
		t.Fatalf("expected app path %q, got %q", mainPath, config.appPath)
	}
	if config.rootPath != tempApp {
		t.Fatalf("expected root path %q, got %q", tempApp, config.rootPath)
	}
	if config.outputPath != filepath.Join(tempApp, "main.wasm") {
		t.Fatalf("expected default output under root, got %q", config.outputPath)
	}
	if config.profile != "development" {
		t.Fatalf("expected development profile, got %q", config.profile)
	}
}

func TestResolveBuildConfigPrefersScaffoldMetadata(t *testing.T) {
	tempApp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
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

	config, err := resolveBuildConfig(buildConfig{})
	if err != nil {
		t.Fatalf("resolve build config: %v", err)
	}
	if config.appPath != filepath.Join(tempApp, "main.go") {
		t.Fatalf("expected metadata app path, got %#v", config)
	}
	if config.outputPath != filepath.Join(tempApp, "out", "app.wasm") {
		t.Fatalf("expected metadata output path, got %#v", config)
	}
	if config.profile != "ci" {
		t.Fatalf("expected metadata profile ci, got %#v", config)
	}
}

func TestResolveBuildConfigDirectoryAppPathAndInvalidMetadata(t *testing.T) {
	t.Run("directory app path uses app directory as root", func(t *testing.T) {
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

		config, err := resolveBuildConfig(buildConfig{appPath: appDir, profile: "benchmark"})
		if err != nil {
			t.Fatalf("resolve build config: %v", err)
		}
		if config.rootPath != appDir || config.outputPath != filepath.Join(appDir, "main.wasm") || config.profile != "benchmark" {
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
		if _, err := resolveBuildConfig(buildConfig{}); err == nil || !strings.Contains(err.Error(), "parse scaffold metadata") {
			t.Fatalf("expected invalid metadata error, got %v", err)
		}
	})
}

func TestResolveBuildConfigBubblesGetwdAndMissingAppErrors(t *testing.T) {
	t.Run("cwd error", func(t *testing.T) {
		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return "", errors.New("cwd failed") }

		if _, err := resolveBuildConfig(buildConfig{}); err == nil || !strings.Contains(err.Error(), "cwd failed") {
			t.Fatalf("expected cwd error, got %v", err)
		}
	})

	t.Run("missing explicit app path", func(t *testing.T) {
		root := t.TempDir()
		originalGetwd := buildGetwd
		t.Cleanup(func() { buildGetwd = originalGetwd })
		buildGetwd = func() (string, error) { return root, nil }

		_, err := resolveBuildConfig(buildConfig{appPath: filepath.Join(root, "missing.go")})
		if err == nil || !strings.Contains(err.Error(), "resolve app path") {
			t.Fatalf("expected missing app path error, got %v", err)
		}
	})
}

func TestRunBuildJSONBuildsWasmArtifact(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	outputPath := filepath.Join(tempApp, "dist", "app.wasm")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcbuildtest\n\ngo 1.25.0\n"), 0644); err != nil {
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
	if err := launcher.run([]string{"build", "-app", mainPath, "-root", tempApp, "-out", outputPath, "-profile", "ci", "-json"}); err != nil {
		t.Fatalf("run build: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary buildSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal build summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful build summary, got %#v", summary)
	}
	if summary.Profile.Name != "ci" {
		t.Fatalf("expected ci profile, got %#v", summary)
	}
	if summary.OutputPath != outputPath {
		t.Fatalf("expected output path %q, got %q", outputPath, summary.OutputPath)
	}
	if summary.Bytes <= 0 {
		t.Fatalf("expected positive artifact size, got %#v", summary)
	}
	if len(summary.SHA256) != 64 {
		t.Fatalf("expected sha256 hash, got %#v", summary)
	}
	if info, err := os.Stat(outputPath); err != nil || info.Size() <= 0 {
		t.Fatalf("expected built artifact at %q, stat err=%v size=%v", outputPath, err, info)
	}
}

func TestRunBuildRejectsUnknownProfile(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	launcher := launcher{}
	err := launcher.run([]string{"build", "-app", mainPath, "-root", tempApp, "-profile", "mystery"})
	if err == nil {
		t.Fatal("expected unknown build profile to fail")
	}
	if !strings.Contains(err.Error(), "unknown build profile") {
		t.Fatalf("expected unknown build profile error, got %v", err)
	}
}

func TestRunBuildHelpReturnsNil(t *testing.T) {
	launcher := launcher{}
	if err := launcher.run([]string{"build", "-help"}); err != nil {
		t.Fatalf("expected build help to succeed, got %v", err)
	}
}

func TestRunBuildHandlesInvalidFlags(t *testing.T) {
	if err := (launcher{}).runBuild([]string{"-definitely-invalid"}); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid build flag error, got %v", err)
	}
}

func TestRunBuildPrintsSummaryWithoutJSON(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	outputPath := filepath.Join(tempApp, "dist", "app.wasm")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcbuildtext\n\ngo 1.25.0\n"), 0644); err != nil {
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

	if err := (launcher{}).run([]string{"build", "-app", mainPath, "-root", tempApp, "-out", outputPath, "-profile", "ci"}); err != nil {
		t.Fatalf("run build: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	for _, expected := range []string{"GWC build", "profile:      ci", "output:       " + outputPath} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected build output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestRunBuildSupportsLegacyAliases(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	outputPath := filepath.Join(tempApp, "dist", "alias.wasm")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcbuildalias\n\ngo 1.25.0\n"), 0644); err != nil {
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
	if err := launcher.run([]string{"build", "-main", mainPath, "-root", tempApp, "-output", outputPath, "-json"}); err != nil {
		t.Fatalf("run build with legacy aliases: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary buildSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal build summary: %v\n%s", err, output)
	}
	if summary.OutputPath != outputPath {
		t.Fatalf("expected output alias path %q, got %#v", outputPath, summary)
	}
}

func TestRunBuildPropagatesGoBuildFailure(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcbuildfail\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() { this does not compile }\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	launcher := launcher{}
	err := launcher.run([]string{"build", "-app", mainPath, "-root", tempApp})
	if err == nil || !strings.Contains(err.Error(), "go build failed") {
		t.Fatalf("expected build failure to be surfaced, got %v", err)
	}
}

func TestExecuteBuildDirectBranches(t *testing.T) {
	t.Run("create build output directory failure", func(t *testing.T) {
		root := t.TempDir()
		mainPath := filepath.Join(root, "main.go")
		if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("write main.go: %v", err)
		}
		parentFile := filepath.Join(root, "occupied")
		if err := os.WriteFile(parentFile, []byte("occupied"), 0644); err != nil {
			t.Fatalf("write parent file: %v", err)
		}

		_, err := executeBuild(buildConfig{appPath: mainPath, rootPath: root, outputPath: filepath.Join(parentFile, "app.wasm"), profile: "development"})
		if err == nil || !strings.Contains(err.Error(), "create build output directory") {
			t.Fatalf("expected output directory creation error, got %v", err)
		}
	})

	t.Run("go build blank output failure", func(t *testing.T) {
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

		_, err := executeBuild(buildConfig{appPath: mainPath, rootPath: root, outputPath: filepath.Join(root, "dist", "app.wasm"), profile: "release"})
		if err == nil || !strings.Contains(err.Error(), "go build failed: exit status") {
			t.Fatalf("expected blank-output build failure, got %v", err)
		}
	})

	t.Run("read built artifact failure after fake build", func(t *testing.T) {
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

		_, err := executeBuild(buildConfig{appPath: mainPath, rootPath: root, outputPath: filepath.Join(root, "dist", "app.wasm"), profile: "development"})
		if err == nil || !strings.Contains(err.Error(), "read built wasm artifact") {
			t.Fatalf("expected built artifact read failure, got %v", err)
		}
	})

	t.Run("directory app path succeeds", func(t *testing.T) {
		root := t.TempDir()
		appDir := filepath.Join(root, "cmd", "web")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatalf("mkdir app dir: %v", err)
		}
		binDir := filepath.Join(t.TempDir(), "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatalf("mkdir fake bin: %v", err)
		}
		writeFakeGoBuildCommand(t, binDir)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("FAKE_GO_MODE", "write-file")

		summary, err := executeBuild(buildConfig{appPath: appDir, rootPath: root, outputPath: filepath.Join(root, "dist", "app.wasm"), profile: "ci"})
		if err != nil {
			t.Fatalf("execute build with directory app path: %v", err)
		}
		if summary.PackageDir != appDir || summary.OutputPath != filepath.Join(root, "dist", "app.wasm") || summary.Profile.Name != "ci" {
			t.Fatalf("expected successful directory build summary, got %#v", summary)
		}
		if summary.Bytes <= 0 || len(summary.SHA256) != 64 {
			t.Fatalf("expected artifact metadata, got %#v", summary)
		}
	})
}
