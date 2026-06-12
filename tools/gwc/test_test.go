package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeTestLanesDefaultsAndAliases(parseT *testing.T) {
	parseT.Run("defaults", func(parseT2 *testing.T) {
		parseGot, parseErr := normalizeTestLanes(nil)
		if parseErr != nil {
			parseT2.Fatalf("normalize test lanes: %v", parseErr)
		}
		parseWant := []string{"unit", "wasm"}
		if !reflect.DeepEqual(parseGot, parseWant) {
			parseT2.Fatalf("expected default lanes %#v, got %#v", parseWant, parseGot)
		}
	})

	parseT.Run("aliases and all", func(parseT3 *testing.T) {
		parseGot2, parseErr2 := normalizeTestLanes([]string{"go-native", "hydrate", "all"})
		if parseErr2 != nil {
			parseT3.Fatalf("normalize test lanes: %v", parseErr2)
		}
		parseWant2 := []string{"unit", "hydration", "race", "wasm", "browser", "perf", "release"}
		if !reflect.DeepEqual(parseGot2, parseWant2) {
			parseT3.Fatalf("expected normalized lanes %#v, got %#v", parseWant2, parseGot2)
		}
	})

	parseT.Run("race aliases", func(parseT4 *testing.T) {
		parseGot3, parseErr3 := normalizeTestLanes([]string{"race-detector", "go-race", "race"})
		if parseErr3 != nil {
			parseT4.Fatalf("normalize test lanes: %v", parseErr3)
		}
		parseWant3 := []string{"race"}
		if !reflect.DeepEqual(parseGot3, parseWant3) {
			parseT4.Fatalf("expected normalized lanes %#v, got %#v", parseWant3, parseGot3)
		}
	})

	parseT.Run("perf aliases", func(parseT4 *testing.T) {
		parseGot3, parseErr3 := normalizeTestLanes([]string{"performance", "perf-budget", "budget", "perf"})
		if parseErr3 != nil {
			parseT4.Fatalf("normalize test lanes: %v", parseErr3)
		}
		parseWant3 := []string{"perf"}
		if !reflect.DeepEqual(parseGot3, parseWant3) {
			parseT4.Fatalf("expected normalized lanes %#v, got %#v", parseWant3, parseGot3)
		}
	})

	parseT.Run("unknown", func(parseT5 *testing.T) {
		if _, parseErr3 := normalizeTestLanes([]string{"mystery"}); parseErr3 == nil {
			parseT5.Fatal("expected unknown lane to fail")
		}
	})
}

func TestCollectWasmTestPackagesHydrationFilter(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "alpha_wasm_test.go"), []byte("package main\nfunc TestAlpha(t *testing.T) {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write alpha_wasm_test.go: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseRoot, "hydration"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir hydration: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "hydration", "hydrate_wasm_test.go"), []byte("package hydration\nfunc TestHydrateFlow(t *testing.T) {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write hydrate_wasm_test.go: %v", parseErr3)
	}
	parseNestedModule := filepath.Join(parseRoot, "third_party", "nested")
	if parseErr4 := os.MkdirAll(parseNestedModule, 0755); parseErr4 != nil {
		parseT.Fatalf("mkdir nested module: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseNestedModule, "go.mod"), []byte("module example.com/nested\n\ngo 1.25.0\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write nested go.mod: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parseNestedModule, "ignored_wasm_test.go"), []byte("package nested\nfunc TestIgnored(t *testing.T) {}\n"), 0644); parseErr6 != nil {
		parseT.Fatalf("write ignored_wasm_test.go: %v", parseErr6)
	}

	parseAllPackages, parseErr7 := collectWasmTestPackages(parseRoot, false)
	if parseErr7 != nil {
		parseT.Fatalf("collect all wasm packages: %v", parseErr7)
	}
	if parseWant := []string{".", "./hydration"}; !reflect.DeepEqual(parseAllPackages, parseWant) {
		parseT.Fatalf("expected all wasm packages %#v, got %#v", parseWant, parseAllPackages)
	}

	parseHydrationPackages, parseErr8 := collectWasmTestPackages(parseRoot, true)
	if parseErr8 != nil {
		parseT.Fatalf("collect hydration wasm packages: %v", parseErr8)
	}
	if parseWant2 := []string{"./hydration"}; !reflect.DeepEqual(parseHydrationPackages, parseWant2) {
		parseT.Fatalf("expected hydration wasm packages %#v, got %#v", parseWant2, parseHydrationPackages)
	}
}

func TestRunTestJSONRunsSelectedLanes(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/gwctestlanes\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "main_wasm_test.go"), []byte("package main\nfunc TestWasmSmoke(t *testing.T) {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main_wasm_test.go: %v", parseErr3)
	}

	parseStdout, parseRestoreStdout, parseErr4 := captureExamplesStdout()
	if parseErr4 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr4)
	}
	defer parseRestoreStdout()

	parseOriginalRunCommand := launcherRunCommand
	parseOriginalResolveWasmExec := testResolveWasmExec
	parseOriginalExecuteRelease := testExecuteRelease
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
		testResolveWasmExec = parseOriginalResolveWasmExec
		testExecuteRelease = parseOriginalExecuteRelease
	})

	var parseInvocations []string
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseInvocations = append(parseInvocations, parseCommand+" "+strings.Join(parseArgs, " ")+" @ "+parseCwd)
		return parseCommand + " ok", nil
	}
	testResolveWasmExec = func(parseRepoRoot string) (string, error) {
		return filepath.Join(parseRepoRoot, "tools", "go_js_wasm_exec.bat"), nil
	}
	testExecuteRelease = func(parseConfig releaseConfig) (releaseSummary, error) {
		return releaseSummary{
			OK:           true,
			AppPath:      parseConfig.appPath,
			ProjectRoot:  parseConfig.rootPath,
			OutDir:       parseConfig.outDir,
			ManifestPath: filepath.Join(parseConfig.outDir, parseConfig.manifestName),
		}, nil
	}

	parseLauncher := launcher{repoRoot: parseRoot}
	if parseErr5 := parseLauncher.run([]string{"test", "-root", parseRoot, "-app", parseMainPath, "-lane", "unit", "-lane", "wasm", "-lane", "release", "-json"}); parseErr5 != nil {
		parseT.Fatalf("run test: %v", parseErr5)
	}

	parseOutput, parseErr4 := parseStdout()
	if parseErr4 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr4)
	}
	var parseSummary testSummary
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr6 != nil {
		parseT.Fatalf("unmarshal test summary: %v\n%s", parseErr6, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful test summary, got %#v", parseSummary)
	}
	if parseWant := []string{"unit", "wasm", "release"}; !reflect.DeepEqual(parseSummary.SelectedLanes, parseWant) {
		parseT.Fatalf("expected selected lanes %#v, got %#v", parseWant, parseSummary.SelectedLanes)
	}
	if len(parseSummary.Lanes) != 3 {
		parseT.Fatalf("expected 3 lane summaries, got %#v", parseSummary)
	}
	if len(parseInvocations) != 2 {
		parseT.Fatalf("expected 2 external command invocations, got %#v", parseInvocations)
	}
	if !strings.Contains(parseInvocations[0], "go test ./...") {
		parseT.Fatalf("expected unit lane go test invocation, got %#v", parseInvocations)
	}
	if parseSummary.Lanes[2].Name != "release" || parseSummary.Lanes[2].ManifestPath == "" {
		parseT.Fatalf("expected release lane manifest path, got %#v", parseSummary.Lanes[2])
	}
	if parseSummary.Lanes[1].Name != "wasm" || len(parseSummary.Lanes[1].Packages) != 1 || parseSummary.Lanes[1].Packages[0] != "." {
		parseT.Fatalf("expected wasm lane package discovery, got %#v", parseSummary.Lanes[1])
	}
	if !strings.Contains(parseInvocations[1], "go test -exec") {
		parseT.Fatalf("expected wasm lane go test -exec invocation, got %#v", parseInvocations)
	}
	if parseSummary.Lanes[0].Workspace != parseRoot {
		parseT.Fatalf("expected unit lane workspace %q, got %#v", parseRoot, parseSummary.Lanes[0])
	}
}

func TestRunTestJSONSkipsHydrationAndBrowserWhenUnavailable(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/gwctestskip\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr2 := captureExamplesStdout()
	if parseErr2 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr2)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{repoRoot: parseRoot}
	if parseErr3 := parseLauncher.run([]string{"test", "-root", parseRoot, "-lane", "hydration", "-lane", "browser", "-json"}); parseErr3 != nil {
		parseT.Fatalf("run test: %v", parseErr3)
	}

	parseOutput, parseErr2 := parseStdout()
	if parseErr2 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr2)
	}
	var parseSummary testSummary
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr4 != nil {
		parseT.Fatalf("unmarshal test summary: %v\n%s", parseErr4, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected skipped-only test summary to remain ok, got %#v", parseSummary)
	}
	if len(parseSummary.Lanes) != 2 {
		parseT.Fatalf("expected 2 lane summaries, got %#v", parseSummary)
	}
	for _, parseLane := range parseSummary.Lanes {
		if !parseLane.Skipped {
			parseT.Fatalf("expected skipped lane, got %#v", parseLane)
		}
	}
}

func TestRunTestWasmLaneFailsWhenExecutorCannotBeResolved(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/gwctestwasmfail\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "main_wasm_test.go"), []byte("package main\nimport \"testing\"\nfunc TestWasmSmoke(t *testing.T) {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main_wasm_test.go: %v", parseErr3)
	}
	parseT.Setenv("GO_WASM_EXEC", "")

	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	parseErr4 := parseLauncher.run([]string{"test", "-root", parseRoot, "-app", parseMainPath, "-lane", "wasm"})
	if parseErr4 == nil {
		parseT.Fatal("expected wasm lane to fail without an executor")
	}
	if !strings.Contains(parseErr4.Error(), "GO_WASM_EXEC") {
		parseT.Fatalf("expected missing wasm executor error, got %v", parseErr4)
	}
}

func TestRunTestBrowserLanePropagatesCommandFailure(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir playwrightgo package: %v", parseErr)
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginalRunCommand })
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		return "playwright failed", errors.New("playwright failed")
	}

	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	parseErr2 := parseLauncher.run([]string{"test", "-root", parseRoot, "-lane", "browser"})
	if parseErr2 == nil {
		parseT.Fatal("expected browser lane failure to be returned")
	}
	if !strings.Contains(parseErr2.Error(), "playwright failed") {
		parseT.Fatalf("expected browser lane failure to be surfaced, got %v", parseErr2)
	}
}

func TestRunTestHandlesHelpAndInvalidFlags(parseT *testing.T) {
	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	if parseErr := parseLauncher.runTest([]string{"-help"}); parseErr != nil {
		parseT.Fatalf("expected test help to succeed, got %v", parseErr)
	}
	parseErr2 := parseLauncher.runTest([]string{"-definitely-invalid"})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid test flag error, got %v", parseErr2)
	}
}

func TestRunTestEnforcesEnterpriseRequiredLanes(parseT *testing.T) {
	parseRoot := parseT.TempDir()

	parseOriginalPolicy := launcherActiveEnterpriseConfig
	parseT.Cleanup(func() { launcherActiveEnterpriseConfig = parseOriginalPolicy })
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Policy: launcherEnterprisePolicy{
			RequiredTestLanes: []string{"unit", "browser"},
		},
	}

	parseErr := (launcher{repoRoot: parseRoot}).runTest([]string{"-root", parseRoot, "-lane", "unit"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "missing required test lanes: browser") {
		parseT.Fatalf("expected required-lane policy failure, got %v", parseErr)
	}
}

func TestRunTestReleaseLanePropagatesReleaseFailure(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseOriginalExecuteRelease := testExecuteRelease
	parseT.Cleanup(func() { testExecuteRelease = parseOriginalExecuteRelease })
	testExecuteRelease = func(parseConfig releaseConfig) (releaseSummary, error) {
		return releaseSummary{}, errors.New("release smoke failed")
	}

	parseLauncher := launcher{repoRoot: parseRoot}
	parseErr2 := parseLauncher.run([]string{"test", "-root", parseRoot, "-app", parseMainPath, "-lane", "release"})
	if parseErr2 == nil {
		parseT.Fatal("expected release lane failure to be returned")
	}
	if !strings.Contains(parseErr2.Error(), "release smoke failed") {
		parseT.Fatalf("expected release lane failure to be surfaced, got %v", parseErr2)
	}
}

func TestRunTestHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{}
	if parseErr := parseLauncher.run([]string{"test", "-help"}); parseErr != nil {
		parseT.Fatalf("expected test help to succeed, got %v", parseErr)
	}
}

func TestRunTestPrintsSummaryWithoutJSON(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := (launcher{repoRoot: parseRoot}).run([]string{"test", "-root", parseRoot, "-lane", "hydration"}); parseErr2 != nil {
		parseT.Fatalf("run test: %v", parseErr2)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseExpected := range []string{"GWC test", "lanes:        hydration", "[skipped] hydration"} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected test output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestResolveTestConfigDefaultsAndInvalidApp(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOriginalGetwd := testGetwd
	parseT.Cleanup(func() { testGetwd = parseOriginalGetwd })
	testGetwd = func() (string, error) { return parseRoot, nil }

	parseConfig, parseErr := resolveTestConfig(testConfig{})
	if parseErr != nil {
		parseT.Fatalf("resolve default test config: %v", parseErr)
	}
	if parseConfig.rootPath != parseRoot {
		parseT.Fatalf("expected default root path %q, got %#v", parseRoot, parseConfig)
	}
	if parseWant := []string{"unit", "wasm"}; !reflect.DeepEqual(parseConfig.lanes, parseWant) {
		parseT.Fatalf("expected default lanes %#v, got %#v", parseWant, parseConfig.lanes)
	}

	_, parseErr = resolveTestConfig(testConfig{appPath: filepath.Join(parseRoot, "missing.go")})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "resolve app path") {
		parseT.Fatalf("expected invalid app path error, got %v", parseErr)
	}
}

func TestResolveTestConfigBubblesGetwdError(parseT *testing.T) {
	parseOriginalGetwd := testGetwd
	parseT.Cleanup(func() { testGetwd = parseOriginalGetwd })
	testGetwd = func() (string, error) { return "", errors.New("cwd failed") }

	if _, parseErr := resolveTestConfig(testConfig{}); parseErr == nil || !strings.Contains(parseErr.Error(), "cwd failed") {
		parseT.Fatalf("expected cwd error, got %v", parseErr)
	}
}

func TestRunUnitTestLaneIncludesNestedLivereloadWorkspace(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseNestedRoot := filepath.Join(parseRepoRoot, "tools", "livereload")
	if parseErr := os.MkdirAll(parseNestedRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir nested root: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseNestedRoot, "go.mod"), []byte("module example.com/livereload\n\ngo 1.25.0\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write nested go.mod: %v", parseErr2)
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginalRunCommand })

	var parseInvocations []string
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseInvocations = append(parseInvocations, parseCwd)
		return filepath.Base(parseCwd) + " ok", nil
	}

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseSummary, parseErr3 := parseLauncher.runUnitTestLane(parseRepoRoot)
	if parseErr3 != nil {
		parseT.Fatalf("run unit test lane: %v", parseErr3)
	}
	if len(parseInvocations) != 2 {
		parseT.Fatalf("expected root and nested invocations, got %#v", parseInvocations)
	}
	if parseInvocations[0] != parseRepoRoot || parseInvocations[1] != parseNestedRoot {
		parseT.Fatalf("expected invocation order [%q %q], got %#v", parseRepoRoot, parseNestedRoot, parseInvocations)
	}
	if parseSummary.Summary != "Native Go tests passed, including the nested livereload workspace." {
		parseT.Fatalf("expected nested unit test summary, got %#v", parseSummary)
	}
	if !strings.Contains(parseSummary.Output, filepath.Base(parseRepoRoot)+" ok") || !strings.Contains(parseSummary.Output, "livereload ok") {
		parseT.Fatalf("expected combined unit outputs, got %#v", parseSummary)
	}
}

func TestRunUnitTestLanePropagatesNestedFailure(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseNestedRoot := filepath.Join(parseRepoRoot, "tools", "livereload")
	if parseErr := os.MkdirAll(parseNestedRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir nested root: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseNestedRoot, "go.mod"), []byte("module example.com/livereload\n\ngo 1.25.0\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write nested go.mod: %v", parseErr2)
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginalRunCommand })

	parseCallCount := 0
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseCallCount++
		if parseCallCount == 2 {
			return "nested failed", errors.New("nested failed")
		}
		return "root ok", nil
	}

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	_, parseErr3 := parseLauncher.runUnitTestLane(parseRepoRoot)
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "nested failed") {
		parseT.Fatalf("expected nested unit lane failure, got %v", parseErr3)
	}
}

func TestGoRaceDetectorSupportMatrix(parseT *testing.T) {
	parseCases := []struct {
		goos      string
		goarch    string
		supported bool
	}{
		{goos: "linux", goarch: "amd64", supported: true},
		{goos: "linux", goarch: "arm64", supported: true},
		{goos: "darwin", goarch: "arm64", supported: true},
		{goos: "windows", goarch: "amd64", supported: true},
		{goos: "windows", goarch: "arm64", supported: false},
		{goos: "js", goarch: "wasm", supported: false},
	}
	for _, parseCase := range parseCases {
		parseGot := isGoRaceDetectorSupported(parseCase.goos, parseCase.goarch)
		if parseGot != parseCase.supported {
			parseT.Fatalf("isGoRaceDetectorSupported(%q, %q) = %v, want %v", parseCase.goos, parseCase.goarch, parseGot, parseCase.supported)
		}
	}
}

func TestRunRaceTestLaneSkipsWhenUnsupported(parseT *testing.T) {
	parseOriginalSupported := testRaceDetectorSupported
	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		testRaceDetectorSupported = parseOriginalSupported
		launcherRunCommand = parseOriginalRunCommand
	})
	testRaceDetectorSupported = func() bool { return false }
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseT.Fatalf("race lane should skip before invoking %s %#v", parseCommand, parseArgs)
		return "", nil
	}

	parseRoot := parseT.TempDir()
	parseSummary, parseErr := (launcher{}).runRaceTestLane(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("run race lane: %v", parseErr)
	}
	if !parseSummary.OK || !parseSummary.Skipped || parseSummary.Name != "race" {
		parseT.Fatalf("expected skipped successful race summary, got %#v", parseSummary)
	}
	if !strings.Contains(parseSummary.Summary, "Go race detector is not supported") {
		parseT.Fatalf("expected unsupported summary, got %#v", parseSummary)
	}
}

func TestRunRaceTestLaneInvokesRaceDetector(parseT *testing.T) {
	parseOriginalSupported := testRaceDetectorSupported
	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		testRaceDetectorSupported = parseOriginalSupported
		launcherRunCommand = parseOriginalRunCommand
	})
	testRaceDetectorSupported = func() bool { return true }

	parseRoot := parseT.TempDir()
	var parseGotCommand string
	var parseGotArgs []string
	var parseGotCwd string
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseGotCommand = parseCommand
		parseGotArgs = append([]string(nil), parseArgs...)
		parseGotCwd = parseCwd
		for _, parseEntry := range parseEnv {
			if strings.HasPrefix(parseEntry, "GOOS=") || strings.HasPrefix(parseEntry, "GOARCH=") {
				parseT.Fatalf("expected native env to clear GOOS/GOARCH, got %q", parseEntry)
			}
		}
		return "race ok", nil
	}

	parseSummary, parseErr := (launcher{}).runRaceTestLane(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("run race lane: %v", parseErr)
	}
	if parseGotCommand != "go" || !reflect.DeepEqual(parseGotArgs, []string{"test", "-race", "./..."}) || parseGotCwd != parseRoot {
		parseT.Fatalf("expected go test -race invocation, got command=%q args=%#v cwd=%q", parseGotCommand, parseGotArgs, parseGotCwd)
	}
	if parseSummary.Name != "race" || !parseSummary.OK || parseSummary.Command != "go test -race ./..." || parseSummary.Output != "race ok" {
		parseT.Fatalf("expected successful race summary, got %#v", parseSummary)
	}
}

func TestExecuteTestLaneRejectsUnknownLane(parseT *testing.T) {
	parseLauncher := launcher{}
	_, parseErr := parseLauncher.executeTestLane(testConfig{}, "mystery")
	if parseErr == nil || !strings.Contains(parseErr.Error(), `unknown test lane "mystery"`) {
		parseT.Fatalf("expected unknown lane error, got %v", parseErr)
	}
}

func TestRunWasmTestLaneSkipsWhenNoPackagesMatch(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseLauncher := launcher{repoRoot: parseRoot}

	parseSummary, parseErr := parseLauncher.runWasmTestLane(parseRoot, false)
	if parseErr != nil {
		parseT.Fatalf("run wasm test lane: %v", parseErr)
	}
	if !parseSummary.Skipped || parseSummary.Name != "wasm" {
		parseT.Fatalf("expected skipped wasm lane summary, got %#v", parseSummary)
	}

	parseSummary, parseErr = parseLauncher.runWasmTestLane(parseRoot, true)
	if parseErr != nil {
		parseT.Fatalf("run hydration test lane: %v", parseErr)
	}
	if !parseSummary.Skipped || parseSummary.Name != "hydration" {
		parseT.Fatalf("expected skipped hydration lane summary, got %#v", parseSummary)
	}
}

func TestCollectWasmTestPackagesAndWasmLaneFailureBranches(parseT *testing.T) {
	parseT.Run("collect packages returns walk error for missing root", func(parseT2 *testing.T) {
		parseMissingRoot := filepath.Join(parseT2.TempDir(), "missing-root")

		if _, parseErr := collectWasmTestPackages(parseMissingRoot, false); parseErr == nil || !strings.Contains(parseErr.Error(), "collect js/wasm test packages") {
			parseT2.Fatalf("expected package collection error, got %v", parseErr)
		}
	})

	parseT.Run("hydration lane success", func(parseT3 *testing.T) {
		parseRoot := parseT3.TempDir()
		parseHydrationDir := filepath.Join(parseRoot, "hydration")
		if parseErr2 := os.MkdirAll(parseHydrationDir, 0755); parseErr2 != nil {
			parseT3.Fatalf("mkdir hydration dir: %v", parseErr2)
		}
		if parseErr3 := os.WriteFile(filepath.Join(parseHydrationDir, "hydrate_wasm_test.go"), []byte("package hydration\nfunc TestHydrationFlow(t *testing.T) {}\n"), 0644); parseErr3 != nil {
			parseT3.Fatalf("write hydrate_wasm_test.go: %v", parseErr3)
		}

		parseOriginalRunCommand := launcherRunCommand
		parseOriginalResolveWasmExec := testResolveWasmExec
		parseT3.Cleanup(func() {
			launcherRunCommand = parseOriginalRunCommand
			testResolveWasmExec = parseOriginalResolveWasmExec
		})

		var parseGotArgs []string
		launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			parseGotArgs = append([]string(nil), parseArgs...)
			if parseCwd != parseRoot {
				parseT3.Fatalf("expected root cwd %q, got %q", parseRoot, parseCwd)
			}
			return "wasm ok", nil
		}
		testResolveWasmExec = func(parseRepoRoot string) (string, error) {
			return filepath.Join(parseRepoRoot, "tools", "go_js_wasm_exec.bat"), nil
		}

		parseSummary, parseErr4 := (launcher{repoRoot: parseRoot}).runWasmTestLane(parseRoot, true)
		if parseErr4 != nil {
			parseT3.Fatalf("run hydration lane: %v", parseErr4)
		}
		if parseSummary.Name != "hydration" || parseSummary.Skipped || !parseSummary.OK {
			parseT3.Fatalf("expected successful hydration summary, got %#v", parseSummary)
		}
		if parseWant := []string{"./hydration"}; !reflect.DeepEqual(parseSummary.Packages, parseWant) {
			parseT3.Fatalf("expected hydration packages %#v, got %#v", parseWant, parseSummary.Packages)
		}
		if len(parseGotArgs) < 4 || parseGotArgs[0] != "test" || parseGotArgs[1] != "-exec" || parseGotArgs[3] != "./hydration" {
			parseT3.Fatalf("expected go test -exec args for hydration lane, got %#v", parseGotArgs)
		}
	})

	parseT.Run("wasm lane propagates command failure", func(parseT4 *testing.T) {
		parseRoot2 := parseT4.TempDir()
		if parseErr5 := os.WriteFile(filepath.Join(parseRoot2, "main_wasm_test.go"), []byte("package main\nfunc TestWasmSmoke(t *testing.T) {}\n"), 0644); parseErr5 != nil {
			parseT4.Fatalf("write main_wasm_test.go: %v", parseErr5)
		}

		parseOriginalRunCommand2 := launcherRunCommand
		parseOriginalResolveWasmExec2 := testResolveWasmExec
		parseT4.Cleanup(func() {
			launcherRunCommand = parseOriginalRunCommand2
			testResolveWasmExec = parseOriginalResolveWasmExec2
		})
		launcherRunCommand = func(parseCommand2 string, parseArgs2 []string, parseCwd2 string, parseEnv2 []string) (string, error) {
			return "wasm failed", errors.New("wasm failed")
		}
		testResolveWasmExec = func(parseRepoRoot2 string) (string, error) {
			return filepath.Join(parseRepoRoot2, "tools", "go_js_wasm_exec.bat"), nil
		}

		_, parseErr6 := (launcher{repoRoot: parseRoot2}).runWasmTestLane(parseRoot2, false)
		if parseErr6 == nil || !strings.Contains(parseErr6.Error(), "wasm failed") {
			parseT4.Fatalf("expected wasm lane failure, got %v", parseErr6)
		}
	})
}

func TestRunBrowserTestLaneSuccessAndSkipPaths(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseWorkspace := filepath.Join(parseRoot, "test")
	if parseErr := os.MkdirAll(parseWorkspace, 0755); parseErr != nil {
		parseT.Fatalf("mkdir browser workspace: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseWorkspace, "playwrightgo"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir playwrightgo package: %v", parseErr2)
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginalRunCommand })
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		parseExpectedArgs := []string{"test", "-tags", "playwrightgo", "./test/playwrightgo", "-run", "TestMainSuite", "-v"}
		if !reflect.DeepEqual(parseArgs, parseExpectedArgs) {
			parseT.Fatalf("expected args %#v, got %#v", parseExpectedArgs, parseArgs)
		}
		if parseCwd != parseRoot {
			parseT.Fatalf("expected workspace cwd %q, got %q", parseRoot, parseCwd)
		}
		return "playwright ok", nil
	}

	parseTestLauncher := launcher{repoRoot: parseRoot}
	parseSummary, parseErr3 := parseTestLauncher.runBrowserTestLane(parseRoot)
	if parseErr3 != nil {
		parseT.Fatalf("run browser test lane: %v", parseErr3)
	}
	if parseSummary.Skipped || !parseSummary.OK || parseSummary.Command == "" || !strings.Contains(parseSummary.Output, "playwright ok") {
		parseT.Fatalf("expected successful browser summary, got %#v", parseSummary)
	}

	parseSummary, parseErr3 = (launcher{repoRoot: parseT.TempDir()}).runBrowserTestLane(parseT.TempDir())
	if parseErr3 != nil {
		parseT.Fatalf("run browser skipped lane: %v", parseErr3)
	}
	if !parseSummary.Skipped || parseSummary.Workspace == "" {
		parseT.Fatalf("expected skipped browser lane summary, got %#v", parseSummary)
	}
}

func TestRunBrowserTestLaneFailsForInvalidOverride(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"browserWorkspace":"missing-browser"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write gwc-runner.json: %v", parseErr)
	}

	_, parseErr2 := (launcher{repoRoot: parseRepoRoot}).runBrowserTestLane(parseRoot)
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "configured browserWorkspace path does not exist") {
		parseT.Fatalf("expected invalid browserWorkspace override to fail, got %v", parseErr2)
	}
}

func TestRunReleaseTestLaneSuccess(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseOriginalExecuteRelease := testExecuteRelease
	parseT.Cleanup(func() { testExecuteRelease = parseOriginalExecuteRelease })
	testExecuteRelease = func(parseConfig releaseConfig) (releaseSummary, error) {
		return releaseSummary{
			OK:           true,
			AppPath:      parseConfig.appPath,
			ProjectRoot:  parseConfig.rootPath,
			OutDir:       parseConfig.outDir,
			ManifestPath: filepath.Join(parseConfig.outDir, parseConfig.manifestName),
		}, nil
	}

	parseLauncher := launcher{repoRoot: parseRoot}
	parseSummary, parseErr2 := parseLauncher.runReleaseTestLane(testConfig{appPath: parseMainPath, rootPath: parseRoot})
	if parseErr2 != nil {
		parseT.Fatalf("run release test lane: %v", parseErr2)
	}
	if parseSummary.Name != "release" || !parseSummary.OK || parseSummary.ManifestPath == "" || parseSummary.OutDir == "" {
		parseT.Fatalf("expected successful release test lane summary, got %#v", parseSummary)
	}
}

func TestRunReleaseTestLaneRejectsInvalidApp(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	_, parseErr := (launcher{repoRoot: parseRoot}).runReleaseTestLane(testConfig{appPath: filepath.Join(parseRoot, "missing.go"), rootPath: parseRoot})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "resolve app path") {
		parseT.Fatalf("expected invalid release lane app path error, got %v", parseErr)
	}
}
