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

func TestNormalizeTestLanesDefaultsAndAliases(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := normalizeTestLanes(nil)
		if err != nil {
			t.Fatalf("normalize test lanes: %v", err)
		}
		want := []string{"unit", "wasm"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected default lanes %#v, got %#v", want, got)
		}
	})

	t.Run("aliases and all", func(t *testing.T) {
		got, err := normalizeTestLanes([]string{"go-native", "hydrate", "all"})
		if err != nil {
			t.Fatalf("normalize test lanes: %v", err)
		}
		want := []string{"unit", "hydration", "wasm", "browser", "release"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected normalized lanes %#v, got %#v", want, got)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		if _, err := normalizeTestLanes([]string{"mystery"}); err == nil {
			t.Fatal("expected unknown lane to fail")
		}
	})
}

func TestCollectWasmTestPackagesHydrationFilter(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha_wasm_test.go"), []byte("package main\nfunc TestAlpha(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write alpha_wasm_test.go: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "hydration"), 0755); err != nil {
		t.Fatalf("mkdir hydration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "hydration", "hydrate_wasm_test.go"), []byte("package hydration\nfunc TestHydrateFlow(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write hydrate_wasm_test.go: %v", err)
	}

	allPackages, err := collectWasmTestPackages(root, false)
	if err != nil {
		t.Fatalf("collect all wasm packages: %v", err)
	}
	if want := []string{".", "./hydration"}; !reflect.DeepEqual(allPackages, want) {
		t.Fatalf("expected all wasm packages %#v, got %#v", want, allPackages)
	}

	hydrationPackages, err := collectWasmTestPackages(root, true)
	if err != nil {
		t.Fatalf("collect hydration wasm packages: %v", err)
	}
	if want := []string{"./hydration"}; !reflect.DeepEqual(hydrationPackages, want) {
		t.Fatalf("expected hydration wasm packages %#v, got %#v", want, hydrationPackages)
	}
}

func TestRunTestJSONRunsSelectedLanes(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gwctestlanes\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main_wasm_test.go"), []byte("package main\nfunc TestWasmSmoke(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write main_wasm_test.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalRunCommand := launcherRunCommand
	originalResolveWasmExec := testResolveWasmExec
	originalExecuteRelease := testExecuteRelease
	t.Cleanup(func() {
		launcherRunCommand = originalRunCommand
		testResolveWasmExec = originalResolveWasmExec
		testExecuteRelease = originalExecuteRelease
	})

	var invocations []string
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		invocations = append(invocations, command+" "+strings.Join(args, " ")+" @ "+cwd)
		return command + " ok", nil
	}
	testResolveWasmExec = func(repoRoot string) (string, error) {
		return filepath.Join(repoRoot, "tools", "go_js_wasm_exec.bat"), nil
	}
	testExecuteRelease = func(config releaseConfig) (releaseSummary, error) {
		return releaseSummary{
			OK:           true,
			AppPath:      config.appPath,
			ProjectRoot:  config.rootPath,
			OutDir:       config.outDir,
			ManifestPath: filepath.Join(config.outDir, config.manifestName),
		}, nil
	}

	launcher := launcher{repoRoot: root}
	if err := launcher.run([]string{"test", "-root", root, "-app", mainPath, "-lane", "unit", "-lane", "wasm", "-lane", "release", "-json"}); err != nil {
		t.Fatalf("run test: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary testSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal test summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful test summary, got %#v", summary)
	}
	if want := []string{"unit", "wasm", "release"}; !reflect.DeepEqual(summary.SelectedLanes, want) {
		t.Fatalf("expected selected lanes %#v, got %#v", want, summary.SelectedLanes)
	}
	if len(summary.Lanes) != 3 {
		t.Fatalf("expected 3 lane summaries, got %#v", summary)
	}
	if len(invocations) != 2 {
		t.Fatalf("expected 2 external command invocations, got %#v", invocations)
	}
	if !strings.Contains(invocations[0], "go test ./...") {
		t.Fatalf("expected unit lane go test invocation, got %#v", invocations)
	}
	if summary.Lanes[2].Name != "release" || summary.Lanes[2].ManifestPath == "" {
		t.Fatalf("expected release lane manifest path, got %#v", summary.Lanes[2])
	}
	if summary.Lanes[1].Name != "wasm" || len(summary.Lanes[1].Packages) != 1 || summary.Lanes[1].Packages[0] != "." {
		t.Fatalf("expected wasm lane package discovery, got %#v", summary.Lanes[1])
	}
	if !strings.Contains(invocations[1], "go test -exec") {
		t.Fatalf("expected wasm lane go test -exec invocation, got %#v", invocations)
	}
	if summary.Lanes[0].Workspace != root {
		t.Fatalf("expected unit lane workspace %q, got %#v", root, summary.Lanes[0])
	}
}

func TestRunTestJSONSkipsHydrationAndBrowserWhenUnavailable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gwctestskip\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{repoRoot: root}
	if err := launcher.run([]string{"test", "-root", root, "-lane", "hydration", "-lane", "browser", "-json"}); err != nil {
		t.Fatalf("run test: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary testSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal test summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected skipped-only test summary to remain ok, got %#v", summary)
	}
	if len(summary.Lanes) != 2 {
		t.Fatalf("expected 2 lane summaries, got %#v", summary)
	}
	for _, lane := range summary.Lanes {
		if !lane.Skipped {
			t.Fatalf("expected skipped lane, got %#v", lane)
		}
	}
}

func TestRunTestWasmLaneFailsWhenExecutorCannotBeResolved(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gwctestwasmfail\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main_wasm_test.go"), []byte("package main\nimport \"testing\"\nfunc TestWasmSmoke(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write main_wasm_test.go: %v", err)
	}
	t.Setenv("GO_WASM_EXEC", "")

	launcher := launcher{repoRoot: t.TempDir()}
	err := launcher.run([]string{"test", "-root", root, "-app", mainPath, "-lane", "wasm"})
	if err == nil {
		t.Fatal("expected wasm lane to fail without an executor")
	}
	if !strings.Contains(err.Error(), "GO_WASM_EXEC") {
		t.Fatalf("expected missing wasm executor error, got %v", err)
	}
}

func TestRunTestBrowserLanePropagatesCommandFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "playwright.config.ts"), []byte("export default {};\n"), 0644); err != nil {
		t.Fatalf("write playwright config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "playwrightgo"), 0755); err != nil {
		t.Fatalf("mkdir playwrightgo package: %v", err)
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() { launcherRunCommand = originalRunCommand })
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		return "playwright failed", errors.New("playwright failed")
	}

	launcher := launcher{repoRoot: t.TempDir()}
	err := launcher.run([]string{"test", "-root", root, "-lane", "browser"})
	if err == nil {
		t.Fatal("expected browser lane failure to be returned")
	}
	if !strings.Contains(err.Error(), "playwright failed") {
		t.Fatalf("expected browser lane failure to be surfaced, got %v", err)
	}
}

func TestRunTestHandlesHelpAndInvalidFlags(t *testing.T) {
	launcher := launcher{repoRoot: t.TempDir()}
	if err := launcher.runTest([]string{"-help"}); err != nil {
		t.Fatalf("expected test help to succeed, got %v", err)
	}
	err := launcher.runTest([]string{"-definitely-invalid"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid test flag error, got %v", err)
	}
}

func TestRunTestEnforcesEnterpriseRequiredLanes(t *testing.T) {
	root := t.TempDir()

	originalPolicy := launcherActiveEnterpriseConfig
	t.Cleanup(func() { launcherActiveEnterpriseConfig = originalPolicy })
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Policy: launcherEnterprisePolicy{
			RequiredTestLanes: []string{"unit", "browser"},
		},
	}

	err := (launcher{repoRoot: root}).runTest([]string{"-root", root, "-lane", "unit"})
	if err == nil || !strings.Contains(err.Error(), "missing required test lanes: browser") {
		t.Fatalf("expected required-lane policy failure, got %v", err)
	}
}

func TestRunTestReleaseLanePropagatesReleaseFailure(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	originalExecuteRelease := testExecuteRelease
	t.Cleanup(func() { testExecuteRelease = originalExecuteRelease })
	testExecuteRelease = func(config releaseConfig) (releaseSummary, error) {
		return releaseSummary{}, errors.New("release smoke failed")
	}

	launcher := launcher{repoRoot: root}
	err := launcher.run([]string{"test", "-root", root, "-app", mainPath, "-lane", "release"})
	if err == nil {
		t.Fatal("expected release lane failure to be returned")
	}
	if !strings.Contains(err.Error(), "release smoke failed") {
		t.Fatalf("expected release lane failure to be surfaced, got %v", err)
	}
}

func TestRunTestHelpReturnsNil(t *testing.T) {
	launcher := launcher{}
	if err := launcher.run([]string{"test", "-help"}); err != nil {
		t.Fatalf("expected test help to succeed, got %v", err)
	}
}

func TestRunTestPrintsSummaryWithoutJSON(t *testing.T) {
	root := t.TempDir()
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{repoRoot: root}).run([]string{"test", "-root", root, "-lane", "hydration"}); err != nil {
		t.Fatalf("run test: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	for _, expected := range []string{"GWC test", "lanes:        hydration", "[skipped] hydration"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected test output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestResolveTestConfigDefaultsAndInvalidApp(t *testing.T) {
	root := t.TempDir()
	originalGetwd := testGetwd
	t.Cleanup(func() { testGetwd = originalGetwd })
	testGetwd = func() (string, error) { return root, nil }

	config, err := resolveTestConfig(testConfig{})
	if err != nil {
		t.Fatalf("resolve default test config: %v", err)
	}
	if config.rootPath != root {
		t.Fatalf("expected default root path %q, got %#v", root, config)
	}
	if want := []string{"unit", "wasm"}; !reflect.DeepEqual(config.lanes, want) {
		t.Fatalf("expected default lanes %#v, got %#v", want, config.lanes)
	}

	_, err = resolveTestConfig(testConfig{appPath: filepath.Join(root, "missing.go")})
	if err == nil || !strings.Contains(err.Error(), "resolve app path") {
		t.Fatalf("expected invalid app path error, got %v", err)
	}
}

func TestResolveTestConfigBubblesGetwdError(t *testing.T) {
	originalGetwd := testGetwd
	t.Cleanup(func() { testGetwd = originalGetwd })
	testGetwd = func() (string, error) { return "", errors.New("cwd failed") }

	if _, err := resolveTestConfig(testConfig{}); err == nil || !strings.Contains(err.Error(), "cwd failed") {
		t.Fatalf("expected cwd error, got %v", err)
	}
}

func TestRunUnitTestLaneIncludesNestedLivereloadWorkspace(t *testing.T) {
	repoRoot := t.TempDir()
	nestedRoot := filepath.Join(repoRoot, "tools", "livereload")
	if err := os.MkdirAll(nestedRoot, 0755); err != nil {
		t.Fatalf("mkdir nested root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedRoot, "go.mod"), []byte("module example.com/livereload\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write nested go.mod: %v", err)
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() { launcherRunCommand = originalRunCommand })

	var invocations []string
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		invocations = append(invocations, cwd)
		return filepath.Base(cwd) + " ok", nil
	}

	launcher := launcher{repoRoot: repoRoot}
	summary, err := launcher.runUnitTestLane(repoRoot)
	if err != nil {
		t.Fatalf("run unit test lane: %v", err)
	}
	if len(invocations) != 2 {
		t.Fatalf("expected root and nested invocations, got %#v", invocations)
	}
	if invocations[0] != repoRoot || invocations[1] != nestedRoot {
		t.Fatalf("expected invocation order [%q %q], got %#v", repoRoot, nestedRoot, invocations)
	}
	if summary.Summary != "Native Go tests passed, including the nested livereload workspace." {
		t.Fatalf("expected nested unit test summary, got %#v", summary)
	}
	if !strings.Contains(summary.Output, filepath.Base(repoRoot)+" ok") || !strings.Contains(summary.Output, "livereload ok") {
		t.Fatalf("expected combined unit outputs, got %#v", summary)
	}
}

func TestRunUnitTestLanePropagatesNestedFailure(t *testing.T) {
	repoRoot := t.TempDir()
	nestedRoot := filepath.Join(repoRoot, "tools", "livereload")
	if err := os.MkdirAll(nestedRoot, 0755); err != nil {
		t.Fatalf("mkdir nested root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedRoot, "go.mod"), []byte("module example.com/livereload\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write nested go.mod: %v", err)
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() { launcherRunCommand = originalRunCommand })

	callCount := 0
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		callCount++
		if callCount == 2 {
			return "nested failed", errors.New("nested failed")
		}
		return "root ok", nil
	}

	launcher := launcher{repoRoot: repoRoot}
	_, err := launcher.runUnitTestLane(repoRoot)
	if err == nil || !strings.Contains(err.Error(), "nested failed") {
		t.Fatalf("expected nested unit lane failure, got %v", err)
	}
}

func TestExecuteTestLaneRejectsUnknownLane(t *testing.T) {
	launcher := launcher{}
	_, err := launcher.executeTestLane(testConfig{}, "mystery")
	if err == nil || !strings.Contains(err.Error(), `unknown test lane "mystery"`) {
		t.Fatalf("expected unknown lane error, got %v", err)
	}
}

func TestRunWasmTestLaneSkipsWhenNoPackagesMatch(t *testing.T) {
	root := t.TempDir()
	launcher := launcher{repoRoot: root}

	summary, err := launcher.runWasmTestLane(root, false)
	if err != nil {
		t.Fatalf("run wasm test lane: %v", err)
	}
	if !summary.Skipped || summary.Name != "wasm" {
		t.Fatalf("expected skipped wasm lane summary, got %#v", summary)
	}

	summary, err = launcher.runWasmTestLane(root, true)
	if err != nil {
		t.Fatalf("run hydration test lane: %v", err)
	}
	if !summary.Skipped || summary.Name != "hydration" {
		t.Fatalf("expected skipped hydration lane summary, got %#v", summary)
	}
}

func TestCollectWasmTestPackagesAndWasmLaneFailureBranches(t *testing.T) {
	t.Run("collect packages returns walk error for missing root", func(t *testing.T) {
		missingRoot := filepath.Join(t.TempDir(), "missing-root")

		if _, err := collectWasmTestPackages(missingRoot, false); err == nil || !strings.Contains(err.Error(), "collect js/wasm test packages") {
			t.Fatalf("expected package collection error, got %v", err)
		}
	})

	t.Run("hydration lane success", func(t *testing.T) {
		root := t.TempDir()
		hydrationDir := filepath.Join(root, "hydration")
		if err := os.MkdirAll(hydrationDir, 0755); err != nil {
			t.Fatalf("mkdir hydration dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(hydrationDir, "hydrate_wasm_test.go"), []byte("package hydration\nfunc TestHydrationFlow(t *testing.T) {}\n"), 0644); err != nil {
			t.Fatalf("write hydrate_wasm_test.go: %v", err)
		}

		originalRunCommand := launcherRunCommand
		originalResolveWasmExec := testResolveWasmExec
		t.Cleanup(func() {
			launcherRunCommand = originalRunCommand
			testResolveWasmExec = originalResolveWasmExec
		})

		var gotArgs []string
		launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
			gotArgs = append([]string(nil), args...)
			if cwd != root {
				t.Fatalf("expected root cwd %q, got %q", root, cwd)
			}
			return "wasm ok", nil
		}
		testResolveWasmExec = func(repoRoot string) (string, error) {
			return filepath.Join(repoRoot, "tools", "go_js_wasm_exec.bat"), nil
		}

		summary, err := (launcher{repoRoot: root}).runWasmTestLane(root, true)
		if err != nil {
			t.Fatalf("run hydration lane: %v", err)
		}
		if summary.Name != "hydration" || summary.Skipped || !summary.OK {
			t.Fatalf("expected successful hydration summary, got %#v", summary)
		}
		if want := []string{"./hydration"}; !reflect.DeepEqual(summary.Packages, want) {
			t.Fatalf("expected hydration packages %#v, got %#v", want, summary.Packages)
		}
		if len(gotArgs) < 4 || gotArgs[0] != "test" || gotArgs[1] != "-exec" || gotArgs[3] != "./hydration" {
			t.Fatalf("expected go test -exec args for hydration lane, got %#v", gotArgs)
		}
	})

	t.Run("wasm lane propagates command failure", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "main_wasm_test.go"), []byte("package main\nfunc TestWasmSmoke(t *testing.T) {}\n"), 0644); err != nil {
			t.Fatalf("write main_wasm_test.go: %v", err)
		}

		originalRunCommand := launcherRunCommand
		originalResolveWasmExec := testResolveWasmExec
		t.Cleanup(func() {
			launcherRunCommand = originalRunCommand
			testResolveWasmExec = originalResolveWasmExec
		})
		launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
			return "wasm failed", errors.New("wasm failed")
		}
		testResolveWasmExec = func(repoRoot string) (string, error) {
			return filepath.Join(repoRoot, "tools", "go_js_wasm_exec.bat"), nil
		}

		_, err := (launcher{repoRoot: root}).runWasmTestLane(root, false)
		if err == nil || !strings.Contains(err.Error(), "wasm failed") {
			t.Fatalf("expected wasm lane failure, got %v", err)
		}
	})
}

func TestRunBrowserTestLaneSuccessAndSkipPaths(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "test")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir browser workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "playwright.config.ts"), []byte("export default {};\n"), 0644); err != nil {
		t.Fatalf("write playwright config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "playwrightgo"), 0755); err != nil {
		t.Fatalf("mkdir playwrightgo package: %v", err)
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() { launcherRunCommand = originalRunCommand })
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("expected go command, got %q", command)
		}
		expectedArgs := []string{"test", "-tags", "playwrightgo", "./playwrightgo", "-run", "TestMainSuite", "-v"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("expected args %#v, got %#v", expectedArgs, args)
		}
		if cwd != workspace {
			t.Fatalf("expected workspace cwd %q, got %q", workspace, cwd)
		}
		return "playwright ok", nil
	}

	testLauncher := launcher{repoRoot: root}
	summary, err := testLauncher.runBrowserTestLane(root)
	if err != nil {
		t.Fatalf("run browser test lane: %v", err)
	}
	if summary.Skipped || !summary.OK || summary.Command == "" || !strings.Contains(summary.Output, "playwright ok") {
		t.Fatalf("expected successful browser summary, got %#v", summary)
	}

	summary, err = (launcher{repoRoot: t.TempDir()}).runBrowserTestLane(t.TempDir())
	if err != nil {
		t.Fatalf("run browser skipped lane: %v", err)
	}
	if !summary.Skipped || summary.Workspace == "" {
		t.Fatalf("expected skipped browser lane summary, got %#v", summary)
	}
}

func TestRunBrowserTestLaneFailsForInvalidOverride(t *testing.T) {
	repoRoot := t.TempDir()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"browserWorkspace":"missing-browser"}}`), 0644); err != nil {
		t.Fatalf("write gwc-runner.json: %v", err)
	}

	_, err := (launcher{repoRoot: repoRoot}).runBrowserTestLane(root)
	if err == nil || !strings.Contains(err.Error(), "configured browserWorkspace does not contain a package.json file") {
		t.Fatalf("expected invalid browserWorkspace override to fail, got %v", err)
	}
}

func TestRunReleaseTestLaneSuccess(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	originalExecuteRelease := testExecuteRelease
	t.Cleanup(func() { testExecuteRelease = originalExecuteRelease })
	testExecuteRelease = func(config releaseConfig) (releaseSummary, error) {
		return releaseSummary{
			OK:           true,
			AppPath:      config.appPath,
			ProjectRoot:  config.rootPath,
			OutDir:       config.outDir,
			ManifestPath: filepath.Join(config.outDir, config.manifestName),
		}, nil
	}

	launcher := launcher{repoRoot: root}
	summary, err := launcher.runReleaseTestLane(testConfig{appPath: mainPath, rootPath: root})
	if err != nil {
		t.Fatalf("run release test lane: %v", err)
	}
	if summary.Name != "release" || !summary.OK || summary.ManifestPath == "" || summary.OutDir == "" {
		t.Fatalf("expected successful release test lane summary, got %#v", summary)
	}
}

func TestRunReleaseTestLaneRejectsInvalidApp(t *testing.T) {
	root := t.TempDir()
	_, err := (launcher{repoRoot: root}).runReleaseTestLane(testConfig{appPath: filepath.Join(root, "missing.go"), rootPath: root})
	if err == nil || !strings.Contains(err.Error(), "resolve app path") {
		t.Fatalf("expected invalid release lane app path error, got %v", err)
	}
}
