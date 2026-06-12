package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeGoBuildCommand(parseT *testing.T, parseBinDir string) {
	parseT.Helper()
	parseScript := "@echo off\r\n" +
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
	if parseErr := os.WriteFile(filepath.Join(parseBinDir, "go.bat"), []byte(parseScript), 0644); parseErr != nil {
		parseT.Fatalf("write fake go.bat: %v", parseErr)
	}
}

func TestResolveBuildProfileAliases(parseT *testing.T) {
	parseTests := []struct {
		input         string
		wantName      string
		wantToolchain string
		wantTarget    string
		trimpath      bool
		wantFlags     string
		wantGCFlags   string
		wantOpt       string
		wantTags      string
	}{
		// Development links fast and keeps dev-only framework surfaces. The
		// debug profile keeps paths untrimmed and disables optimization for
		// source-debugging sessions. Release-shaped profiles strip fully and
		// exclude dev-only code via the production build tag.
		{input: "development", wantName: "development", wantToolchain: "go", wantTarget: "js/wasm", trimpath: false, wantFlags: "-w", wantTags: ""},
		{input: "dev", wantName: "development", wantToolchain: "go", wantTarget: "js/wasm", trimpath: false, wantFlags: "-w", wantTags: ""},
		{input: "debug", wantName: "debug", wantToolchain: "go", wantTarget: "js/wasm", trimpath: false, wantGCFlags: "all=-N -l", wantTags: ""},
		{input: "ci", wantName: "ci", wantToolchain: "go", wantTarget: "js/wasm", trimpath: true, wantFlags: "-s -w", wantTags: "production"},
		{input: "bench", wantName: "benchmark", wantToolchain: "go", wantTarget: "js/wasm", trimpath: true, wantFlags: "-s -w", wantTags: "production"},
		{input: "prod", wantName: "release", wantToolchain: "go", wantTarget: "js/wasm", trimpath: true, wantFlags: "-s -w", wantTags: "production"},
		{input: "tiny", wantName: "tinygo", wantToolchain: "tinygo", wantTarget: "wasm", trimpath: false, wantFlags: "", wantOpt: "z", wantTags: "production"},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.input, func(parseT2 *testing.T) {
			parseProfile, parseErr := resolveBuildProfile(parseTest.input)
			if parseErr != nil {
				parseT2.Fatalf("resolve build profile: %v", parseErr)
			}
			if parseProfile.Name != parseTest.wantName {
				parseT2.Fatalf("expected profile name %q, got %q", parseTest.wantName, parseProfile.Name)
			}
			if parseProfile.Toolchain != parseTest.wantToolchain {
				parseT2.Fatalf("expected toolchain %q, got %#v", parseTest.wantToolchain, parseProfile)
			}
			if parseProfile.Target != parseTest.wantTarget {
				parseT2.Fatalf("expected target %q, got %#v", parseTest.wantTarget, parseProfile)
			}
			if parseProfile.Trimpath != parseTest.trimpath {
				parseT2.Fatalf("expected trimpath %t, got %#v", parseTest.trimpath, parseProfile)
			}
			if parseProfile.Ldflags != parseTest.wantFlags {
				parseT2.Fatalf("expected ldflags %q, got %#v", parseTest.wantFlags, parseProfile)
			}
			if parseProfile.GCFlags != parseTest.wantGCFlags {
				parseT2.Fatalf("expected gcflags %q, got %#v", parseTest.wantGCFlags, parseProfile)
			}
			if parseProfile.Opt != parseTest.wantOpt {
				parseT2.Fatalf("expected opt %q, got %#v", parseTest.wantOpt, parseProfile)
			}
			if parseProfile.Tags != parseTest.wantTags {
				parseT2.Fatalf("expected tags %q, got %#v", parseTest.wantTags, parseProfile)
			}
		})
	}
	if _, parseErr2 := resolveBuildProfile("mystery"); parseErr2 == nil {
		parseT.Fatal("expected unknown build profile to fail")
	}
}

func TestBuildCommandForDebugProfilePreservesSourceDebugFlags(parseT *testing.T) {
	parseProfile, parseErr := resolveBuildProfile("source-debug")
	if parseErr != nil {
		parseT.Fatalf("resolve debug profile: %v", parseErr)
	}

	parseCommand, parseArgs, parseEnv, parseErr := buildCommandForProfile(parseProfile, "out.wasm")
	if parseErr != nil {
		parseT.Fatalf("build command for debug profile: %v", parseErr)
	}
	if parseCommand != "go" {
		parseT.Fatalf("expected go command, got %q", parseCommand)
	}
	parseJoined := strings.Join(parseArgs, "\x00")
	for _, parseExpected := range []string{"build", "-o", "out.wasm", "-gcflags=all=-N -l", "."} {
		if !strings.Contains(parseJoined, parseExpected) {
			parseT.Fatalf("expected debug build args to contain %q, got %#v", parseExpected, parseArgs)
		}
	}
	for _, parseForbidden := range []string{"-trimpath", "-ldflags"} {
		if strings.Contains(parseJoined, parseForbidden) {
			parseT.Fatalf("debug build args should not contain %q, got %#v", parseForbidden, parseArgs)
		}
	}
	if !envContains(parseEnv, "GOOS=js") || !envContains(parseEnv, "GOARCH=wasm") {
		parseT.Fatalf("expected js/wasm build env, got %#v", parseEnv)
	}
}

func TestResolveBuildConfigDefaults(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	parseConfig, parseErr2 := resolveBuildConfig(buildConfig{})
	if parseErr2 != nil {
		parseT.Fatalf("resolve build config: %v", parseErr2)
	}
	if parseConfig.appPath != parseMainPath {
		parseT.Fatalf("expected app path %q, got %q", parseMainPath, parseConfig.appPath)
	}
	if parseConfig.rootPath != parseTempApp {
		parseT.Fatalf("expected root path %q, got %q", parseTempApp, parseConfig.rootPath)
	}
	if parseConfig.outputPath != filepath.Join(parseTempApp, "bin", "main.wasm") {
		parseT.Fatalf("expected default output under root, got %q", parseConfig.outputPath)
	}
	if parseConfig.profile != "development" {
		parseT.Fatalf("expected development profile, got %q", parseConfig.profile)
	}
}

func TestResolveBuildConfigPrefersScaffoldMetadata(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
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
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr2)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	parseConfig, parseErr3 := resolveBuildConfig(buildConfig{})
	if parseErr3 != nil {
		parseT.Fatalf("resolve build config: %v", parseErr3)
	}
	if parseConfig.appPath != filepath.Join(parseTempApp, "main.go") {
		parseT.Fatalf("expected metadata app path, got %#v", parseConfig)
	}
	if parseConfig.outputPath != filepath.Join(parseTempApp, "out", "app.wasm") {
		parseT.Fatalf("expected metadata output path, got %#v", parseConfig)
	}
	if parseConfig.profile != "ci" {
		parseT.Fatalf("expected metadata profile ci, got %#v", parseConfig)
	}
}

func TestResolveBuildConfigUsesArtifactRootOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-runner.json: %v", parseErr2)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseRoot, nil }

	parseConfig, parseErr3 := resolveBuildConfig(buildConfig{})
	if parseErr3 != nil {
		parseT.Fatalf("resolve build config: %v", parseErr3)
	}
	parseWant := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "bin", "main.wasm")
	if parseConfig.outputPath != parseWant {
		parseT.Fatalf("expected artifact-root output path %q, got %#v", parseWant, parseConfig)
	}
	if parseConfig.resolution["output"] != "gwc-runner.json paths.artifactRoot" {
		parseT.Fatalf("expected explicit runner-config output tracing, got %#v", parseConfig.resolution)
	}
}

func TestResolveBuildConfigDirectoryAppPathAndInvalidMetadata(parseT *testing.T) {
	parseT.Run("directory app path uses app directory as root", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseAppDir := filepath.Join(parseRoot, "cmd", "web")
		if parseErr := os.MkdirAll(parseAppDir, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir app dir: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(filepath.Join(parseAppDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
			parseT2.Fatalf("write main.go: %v", parseErr2)
		}

		parseOriginalGetwd := buildGetwd
		parseT2.Cleanup(func() { buildGetwd = parseOriginalGetwd })
		buildGetwd = func() (string, error) { return parseRoot, nil }

		parseConfig, parseErr3 := resolveBuildConfig(buildConfig{appPath: parseAppDir, profile: "benchmark"})
		if parseErr3 != nil {
			parseT2.Fatalf("resolve build config: %v", parseErr3)
		}
		if parseConfig.rootPath != parseAppDir || parseConfig.outputPath != filepath.Join(parseAppDir, "bin", "main.wasm") || parseConfig.profile != "benchmark" {
			parseT2.Fatalf("expected directory app path defaults, got %#v", parseConfig)
		}
	})

	parseT.Run("invalid metadata bubbles parse error", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		if parseErr4 := os.WriteFile(filepath.Join(parseRoot2, "gwc-start.json"), []byte(`{"tooling":`), 0644); parseErr4 != nil {
			parseT3.Fatalf("write invalid metadata: %v", parseErr4)
		}
		parseOriginalGetwd2 := buildGetwd
		parseT3.Cleanup(func() { buildGetwd = parseOriginalGetwd2 })
		buildGetwd = func() (string, error) { return parseRoot2, nil }
		if _, parseErr5 := resolveBuildConfig(buildConfig{}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "parse scaffold metadata") {
			parseT3.Fatalf("expected invalid metadata error, got %v", parseErr5)
		}
	})
}

func TestResolveBuildConfigBubblesGetwdAndMissingAppErrors(parseT *testing.T) {
	parseT.Run("cwd error", func(parseT2 *testing.T) {
		parseOriginalGetwd := buildGetwd
		parseT2.Cleanup(func() { buildGetwd = parseOriginalGetwd })
		buildGetwd = func() (string, error) { return "", errors.New("cwd failed") }

		if _, parseErr := resolveBuildConfig(buildConfig{}); parseErr == nil || !strings.Contains(parseErr.Error(), "cwd failed") {
			parseT2.Fatalf("expected cwd error, got %v", parseErr)
		}
	})

	parseT.Run("missing explicit app path", func(parseT3 *testing.T) {
		parseRoot := parseT3.TempDir()
		parseOriginalGetwd2 := buildGetwd
		parseT3.Cleanup(func() { buildGetwd = parseOriginalGetwd2 })
		buildGetwd = func() (string, error) { return parseRoot, nil }

		_, parseErr2 := resolveBuildConfig(buildConfig{appPath: filepath.Join(parseRoot, "missing.go")})
		if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "resolve app path") {
			parseT3.Fatalf("expected missing app path error, got %v", parseErr2)
		}
	})
}

func TestRunBuildJSONBuildsWasmArtifact(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutputPath := filepath.Join(parseTempApp, "dist", "app.wasm")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcbuildtest\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"build", "-app", parseMainPath, "-root", parseTempApp, "-out", parseOutputPath, "-profile", "ci", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run build: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary buildSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal build summary: %v\n%s", parseErr5, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful build summary, got %#v", parseSummary)
	}
	if parseSummary.Profile.Name != "ci" {
		parseT.Fatalf("expected ci profile, got %#v", parseSummary)
	}
	if parseSummary.OutputPath != parseOutputPath {
		parseT.Fatalf("expected output path %q, got %q", parseOutputPath, parseSummary.OutputPath)
	}
	if parseSummary.Bytes <= 0 {
		parseT.Fatalf("expected positive artifact size, got %#v", parseSummary)
	}
	if len(parseSummary.SHA256) != 64 {
		parseT.Fatalf("expected sha256 hash, got %#v", parseSummary)
	}
	if parseInfo, parseErr6 := os.Stat(parseOutputPath); parseErr6 != nil || parseInfo.Size() <= 0 {
		parseT.Fatalf("expected built artifact at %q, stat err=%v size=%v", parseOutputPath, parseErr6, parseInfo)
	}
}

func TestRunBuildTinyGoProfileUsesTinyGoCommand(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutputPath := filepath.Join(parseTempApp, "dist", "app.wasm")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwctinygotest\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseOriginalLookPath := buildLookPath
	parseOriginalRunCommand := buildRunCommand
	parseT.Cleanup(func() {
		buildLookPath = parseOriginalLookPath
		buildRunCommand = parseOriginalRunCommand
	})

	buildLookPath = func(parseFile string) (string, error) {
		if parseFile != "tinygo" {
			return "", errors.New("unexpected tool lookup: " + parseFile)
		}
		return filepath.Join(parseT.TempDir(), "tinygo.exe"), nil
	}
	var parseGotCommand string
	var parseGotArgs []string
	var parseGotCwd string
	buildRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseGotCommand = parseCommand
		parseGotArgs = append([]string{}, parseArgs...)
		parseGotCwd = parseCwd
		if parseCommand != "tinygo" {
			return "", errors.New("unexpected command: " + parseCommand)
		}
		if parseErr := os.MkdirAll(filepath.Dir(parseOutputPath), 0755); parseErr != nil {
			return "", parseErr
		}
		if parseErr := os.WriteFile(parseOutputPath, []byte("tiny wasm"), 0644); parseErr != nil {
			return "", parseErr
		}
		return "", nil
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"build", "-app", parseMainPath, "-root", parseTempApp, "-out", parseOutputPath, "-profile", "tinygo", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run tinygo build: %v", parseErr4)
	}
	parseWantArgs := []string{"build", "-target=wasm", "-o", parseOutputPath, "-opt=z", "-tags", "production", "."}
	if parseGotCommand != "tinygo" || strings.Join(parseGotArgs, "\x00") != strings.Join(parseWantArgs, "\x00") {
		parseT.Fatalf("expected tinygo args %#v, got command=%q args=%#v", parseWantArgs, parseGotCommand, parseGotArgs)
	}
	if parseGotCwd != parseTempApp {
		parseT.Fatalf("expected tinygo cwd %q, got %q", parseTempApp, parseGotCwd)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary buildSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal tinygo build summary: %v\n%s", parseErr5, parseOutput)
	}
	if parseSummary.Profile.Name != "tinygo" || parseSummary.Profile.Toolchain != "tinygo" || parseSummary.Profile.Target != "wasm" || parseSummary.Profile.Opt != "z" {
		parseT.Fatalf("expected tinygo profile metadata, got %#v", parseSummary.Profile)
	}
	if parseSummary.OutputPath != parseOutputPath || parseSummary.Bytes <= 0 || len(parseSummary.SHA256) != 64 {
		parseT.Fatalf("expected tinygo artifact metadata, got %#v", parseSummary)
	}
}

func TestRunBuildTinyGoProfileRequiresTinyGoOnPath(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseOriginalLookPath := buildLookPath
	parseT.Cleanup(func() { buildLookPath = parseOriginalLookPath })
	buildLookPath = func(parseFile string) (string, error) {
		if parseFile != "tinygo" {
			return "", errors.New("unexpected tool lookup: " + parseFile)
		}
		return "", errors.New("not found")
	}

	parseLauncher := launcher{}
	parseErr2 := parseLauncher.run([]string{"build", "-app", parseMainPath, "-root", parseTempApp, "-profile", "tinygo"})
	if parseErr2 == nil {
		parseT.Fatal("expected missing TinyGo to fail")
	}
	if !strings.Contains(parseErr2.Error(), "tinygo build profile requires TinyGo on PATH") {
		parseT.Fatalf("expected actionable TinyGo error, got %v", parseErr2)
	}
}

func TestRunBuildRejectsUnknownProfile(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseLauncher := launcher{}
	parseErr2 := parseLauncher.run([]string{"build", "-app", parseMainPath, "-root", parseTempApp, "-profile", "mystery"})
	if parseErr2 == nil {
		parseT.Fatal("expected unknown build profile to fail")
	}
	if !strings.Contains(parseErr2.Error(), "unknown build profile") {
		parseT.Fatalf("expected unknown build profile error, got %v", parseErr2)
	}
}

func TestRunBuildHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{}
	if parseErr := parseLauncher.run([]string{"build", "-help"}); parseErr != nil {
		parseT.Fatalf("expected build help to succeed, got %v", parseErr)
	}
}

func TestRunBuildHandlesInvalidFlags(parseT *testing.T) {
	if parseErr := (launcher{}).runBuild([]string{"-definitely-invalid"}); parseErr == nil || !strings.Contains(parseErr.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid build flag error, got %v", parseErr)
	}
}

func TestRunBuildPrintsSummaryWithoutJSON(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutputPath := filepath.Join(parseTempApp, "dist", "app.wasm")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcbuildtext\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).run([]string{"build", "-app", parseMainPath, "-root", parseTempApp, "-out", parseOutputPath, "-profile", "ci"}); parseErr4 != nil {
		parseT.Fatalf("run build: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	for _, parseExpected := range []string{"GWC build", "profile:      ci", "output:       " + parseOutputPath} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected build output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestRunBuildSupportsLegacyAliases(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutputPath := filepath.Join(parseTempApp, "dist", "alias.wasm")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcbuildalias\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"build", "-main", parseMainPath, "-root", parseTempApp, "-output", parseOutputPath, "-json"}); parseErr4 != nil {
		parseT.Fatalf("run build with legacy aliases: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary buildSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal build summary: %v\n%s", parseErr5, parseOutput)
	}
	if parseSummary.OutputPath != parseOutputPath {
		parseT.Fatalf("expected output alias path %q, got %#v", parseOutputPath, parseSummary)
	}
}

func TestRunBuildPropagatesGoBuildFailure(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcbuildfail\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() { this does not compile }\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseLauncher := launcher{}
	parseErr3 := parseLauncher.run([]string{"build", "-app", parseMainPath, "-root", parseTempApp})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "go build failed") {
		parseT.Fatalf("expected build failure to be surfaced, got %v", parseErr3)
	}
}

func TestExecuteBuildDirectBranches(parseT *testing.T) {
	parseT.Run("create build output directory failure", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseMainPath := filepath.Join(parseRoot, "main.go")
		if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write main.go: %v", parseErr)
		}
		parseParentFile := filepath.Join(parseRoot, "occupied")
		if parseErr2 := os.WriteFile(parseParentFile, []byte("occupied"), 0644); parseErr2 != nil {
			parseT2.Fatalf("write parent file: %v", parseErr2)
		}

		_, parseErr3 := executeBuild(buildConfig{appPath: parseMainPath, rootPath: parseRoot, outputPath: filepath.Join(parseParentFile, "app.wasm"), profile: "development"})
		if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "create build output directory") {
			parseT2.Fatalf("expected output directory creation error, got %v", parseErr3)
		}
	})

	parseT.Run("go build blank output failure", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		parseMainPath2 := filepath.Join(parseRoot2, "main.go")
		if parseErr4 := os.WriteFile(parseMainPath2, []byte("package main\nfunc main() {}\n"), 0644); parseErr4 != nil {
			parseT3.Fatalf("write main.go: %v", parseErr4)
		}
		parseBinDir := filepath.Join(parseT3.TempDir(), "bin")
		if parseErr5 := os.MkdirAll(parseBinDir, 0755); parseErr5 != nil {
			parseT3.Fatalf("mkdir fake bin: %v", parseErr5)
		}
		writeFakeGoBuildCommand(parseT3, parseBinDir)
		parseT3.Setenv("PATH", parseBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		parseT3.Setenv("FAKE_GO_MODE", "fail-empty")

		_, parseErr6 := executeBuild(buildConfig{appPath: parseMainPath2, rootPath: parseRoot2, outputPath: filepath.Join(parseRoot2, "dist", "app.wasm"), profile: "release"})
		if parseErr6 == nil || !strings.Contains(parseErr6.Error(), "go build failed: exit status") {
			parseT3.Fatalf("expected blank-output build failure, got %v", parseErr6)
		}
	})

	parseT.Run("read built artifact failure after fake build", func(parseT4 *testing.T) {
		parseRoot3 := parseT4.TempDir()
		parseMainPath3 := filepath.Join(parseRoot3, "main.go")
		if parseErr7 := os.WriteFile(parseMainPath3, []byte("package main\nfunc main() {}\n"), 0644); parseErr7 != nil {
			parseT4.Fatalf("write main.go: %v", parseErr7)
		}
		parseBinDir2 := filepath.Join(parseT4.TempDir(), "bin")
		if parseErr8 := os.MkdirAll(parseBinDir2, 0755); parseErr8 != nil {
			parseT4.Fatalf("mkdir fake bin: %v", parseErr8)
		}
		writeFakeGoBuildCommand(parseT4, parseBinDir2)
		parseT4.Setenv("PATH", parseBinDir2+string(os.PathListSeparator)+os.Getenv("PATH"))
		parseT4.Setenv("FAKE_GO_MODE", "write-dir")

		_, parseErr9 := executeBuild(buildConfig{appPath: parseMainPath3, rootPath: parseRoot3, outputPath: filepath.Join(parseRoot3, "dist", "app.wasm"), profile: "development"})
		if parseErr9 == nil || !strings.Contains(parseErr9.Error(), "read built wasm artifact") {
			parseT4.Fatalf("expected built artifact read failure, got %v", parseErr9)
		}
	})

	parseT.Run("directory app path succeeds", func(parseT5 *testing.T) {
		parseRoot4 := parseT5.TempDir()
		parseAppDir := filepath.Join(parseRoot4, "cmd", "web")
		if parseErr10 := os.MkdirAll(parseAppDir, 0755); parseErr10 != nil {
			parseT5.Fatalf("mkdir app dir: %v", parseErr10)
		}
		parseBinDir3 := filepath.Join(parseT5.TempDir(), "bin")
		if parseErr11 := os.MkdirAll(parseBinDir3, 0755); parseErr11 != nil {
			parseT5.Fatalf("mkdir fake bin: %v", parseErr11)
		}
		writeFakeGoBuildCommand(parseT5, parseBinDir3)
		parseT5.Setenv("PATH", parseBinDir3+string(os.PathListSeparator)+os.Getenv("PATH"))
		parseT5.Setenv("FAKE_GO_MODE", "write-file")

		parseSummary, parseErr12 := executeBuild(buildConfig{appPath: parseAppDir, rootPath: parseRoot4, outputPath: filepath.Join(parseRoot4, "dist", "app.wasm"), profile: "ci"})
		if parseErr12 != nil {
			parseT5.Fatalf("execute build with directory app path: %v", parseErr12)
		}
		if parseSummary.PackageDir != parseAppDir || parseSummary.OutputPath != filepath.Join(parseRoot4, "dist", "app.wasm") || parseSummary.Profile.Name != "ci" {
			parseT5.Fatalf("expected successful directory build summary, got %#v", parseSummary)
		}
		if parseSummary.Bytes <= 0 || len(parseSummary.SHA256) != 64 {
			parseT5.Fatalf("expected artifact metadata, got %#v", parseSummary)
		}
	})
}

// TestDetectHeavyWASMImports verifies the size-hygiene warnings fire for known
// heavy stdlib packages in a real wasm dependency graph and stay silent for
// packages that are not linked.
func TestDetectHeavyWASMImports(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseWarnings := detectHeavyWASMImports(filepath.Join(parseRepoRoot, "test", "testapp"))

	parseHasRegexp := false
	for _, parseWarning := range parseWarnings {
		if strings.Contains(parseWarning, "net/http is linked") {
			parseT.Fatalf("net/http must not be linked into wasm test app (diagnostics http split regressed): %v", parseWarnings)
		}
		if strings.Contains(parseWarning, "regexp is linked") {
			parseHasRegexp = true
		}
	}
	// runtime2's diagnostic redaction legitimately uses regexp today; the
	// warning documents the cost. If that dependency is ever removed, this
	// assertion should flip to require no warnings at all.
	if !parseHasRegexp {
		parseT.Fatalf("expected regexp size warning for test app, got %v", parseWarnings)
	}
}
