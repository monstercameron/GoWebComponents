package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunBuildClientOrchestratesTargets verifies the main build pipeline ordering and paths.
func TestRunBuildClientOrchestratesTargets(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseOriginalFindRepoRoot := runBuildClientFindRepoRoot
	parseOriginalBuildSharedTailwind := runBuildClientBuildSharedTailwind
	parseOriginalRemoveLegacyArtifact := runBuildClientRemoveLegacyArtifact
	parseOriginalBuildTarget := runBuildClientBuildTarget
	parseOriginalWriteBrotliSidecar := runBuildClientWriteBrotliSidecar
	parseT.Cleanup(func() {
		runBuildClientFindRepoRoot = parseOriginalFindRepoRoot
		runBuildClientBuildSharedTailwind = parseOriginalBuildSharedTailwind
		runBuildClientRemoveLegacyArtifact = parseOriginalRemoveLegacyArtifact
		runBuildClientBuildTarget = parseOriginalBuildTarget
		runBuildClientWriteBrotliSidecar = parseOriginalWriteBrotliSidecar
	})

	parseCalls := []string{}
	parseRemoved := []string{}
	parseBuilt := []string{}
	parseBrotli := []string{}

	runBuildClientFindRepoRoot = func() (string, error) {
		parseCalls = append(parseCalls, "find-root")
		return parseRepoRoot, nil
	}
	runBuildClientBuildSharedTailwind = func(parseRoot string) error {
		parseCalls = append(parseCalls, "tailwind:"+parseRoot)
		return nil
	}
	runBuildClientRemoveLegacyArtifact = func(parsePath string) error {
		parseCalls = append(parseCalls, "remove:"+filepath.Base(parsePath))
		parseRemoved = append(parseRemoved, parsePath)
		return nil
	}
	runBuildClientBuildTarget = func(parseRoot string, parseLabel string, parsePackagePath string, parseOutputPath string) error {
		parseCalls = append(parseCalls, "build:"+parseLabel)
		parseBuilt = append(parseBuilt, parsePackagePath+"->"+parseOutputPath)
		return nil
	}
	runBuildClientWriteBrotliSidecar = func(parseSourcePath string, parseTargetPath string) error {
		parseCalls = append(parseCalls, "brotli:"+filepath.Base(parseSourcePath))
		parseBrotli = append(parseBrotli, parseSourcePath+"->"+parseTargetPath)
		return nil
	}

	if parseErr := runBuildClient(); parseErr != nil {
		parseT.Fatalf("runBuildClient: %v", parseErr)
	}

	if len(parseRemoved) != 2 || len(parseBuilt) != 2 || len(parseBrotli) != 2 {
		parseT.Fatalf("unexpected orchestration counts removed=%d built=%d brotli=%d", len(parseRemoved), len(parseBuilt), len(parseBrotli))
	}
	if parseCalls[0] != "find-root" || parseCalls[1] != "tailwind:"+parseRepoRoot {
		parseT.Fatalf("unexpected initial call order %#v", parseCalls)
	}
	for _, parseNeedle := range []string{
		filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "client", "chat.wasm"),
		filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "client", "backgroundworker", "background-worker.wasm"),
		"./examples/100-ai-chat-wizard/client->" + filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "bin", "client", "app", "chat.wasm"),
		"./examples/100-ai-chat-wizard/client/backgroundworker->" + filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "bin", "client", "worker", "background-worker.wasm"),
	} {
		parseJoined := strings.Join(append(append([]string{}, parseRemoved...), parseBuilt...), "\n")
		if !strings.Contains(parseJoined, parseNeedle) {
			parseT.Fatalf("expected orchestration to include %q\n%s", parseNeedle, parseJoined)
		}
	}
}

// TestRunBuildClientPropagatesStepErrors verifies build pipeline failures stop at the failing step.
func TestRunBuildClientPropagatesStepErrors(parseT *testing.T) {
	parseOriginalFindRepoRoot := runBuildClientFindRepoRoot
	parseOriginalBuildSharedTailwind := runBuildClientBuildSharedTailwind
	parseOriginalRemoveLegacyArtifact := runBuildClientRemoveLegacyArtifact
	parseOriginalBuildTarget := runBuildClientBuildTarget
	parseOriginalWriteBrotliSidecar := runBuildClientWriteBrotliSidecar
	parseT.Cleanup(func() {
		runBuildClientFindRepoRoot = parseOriginalFindRepoRoot
		runBuildClientBuildSharedTailwind = parseOriginalBuildSharedTailwind
		runBuildClientRemoveLegacyArtifact = parseOriginalRemoveLegacyArtifact
		runBuildClientBuildTarget = parseOriginalBuildTarget
		runBuildClientWriteBrotliSidecar = parseOriginalWriteBrotliSidecar
	})

	parseRepoRoot := parseT.TempDir()
	runBuildClientFindRepoRoot = func() (string, error) { return parseRepoRoot, nil }
	runBuildClientBuildSharedTailwind = func(string) error { return nil }
	runBuildClientRemoveLegacyArtifact = func(string) error { return nil }
	runBuildClientBuildTarget = func(string, string, string, string) error { return nil }
	runBuildClientWriteBrotliSidecar = func(string, string) error { return nil }

	parseCases := []struct {
		name       string
		buildSetup func()
		want       string
	}{
		{
			name: "root",
			buildSetup: func() {
				runBuildClientFindRepoRoot = func() (string, error) { return "", errors.New("root failure") }
			},
			want: "root failure",
		},
		{
			name: "tailwind",
			buildSetup: func() {
				runBuildClientFindRepoRoot = func() (string, error) { return parseRepoRoot, nil }
				runBuildClientBuildSharedTailwind = func(string) error { return errors.New("tailwind failure") }
			},
			want: "tailwind failure",
		},
		{
			name: "remove",
			buildSetup: func() {
				runBuildClientBuildSharedTailwind = func(string) error { return nil }
				runBuildClientRemoveLegacyArtifact = func(string) error { return errors.New("remove failure") }
			},
			want: "remove failure",
		},
		{
			name: "build",
			buildSetup: func() {
				runBuildClientRemoveLegacyArtifact = func(string) error { return nil }
				runBuildClientBuildTarget = func(string, string, string, string) error { return errors.New("build failure") }
			},
			want: "build failure",
		},
		{
			name: "brotli",
			buildSetup: func() {
				runBuildClientBuildTarget = func(string, string, string, string) error { return nil }
				runBuildClientWriteBrotliSidecar = func(string, string) error { return errors.New("brotli failure") }
			},
			want: "brotli failure",
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			runBuildClientFindRepoRoot = func() (string, error) { return parseRepoRoot, nil }
			runBuildClientBuildSharedTailwind = func(string) error { return nil }
			runBuildClientRemoveLegacyArtifact = func(string) error { return nil }
			runBuildClientBuildTarget = func(string, string, string, string) error { return nil }
			runBuildClientWriteBrotliSidecar = func(string, string) error { return nil }
			parseCase.buildSetup()

			parseErr := runBuildClient()
			if parseErr == nil || !strings.Contains(parseErr.Error(), parseCase.want) {
				parseT.Fatalf("expected %q, got %v", parseCase.want, parseErr)
			}
		})
	}
}

// TestBuildSharedTailwindUsesGoCommand verifies the Tailwind helper command and failure path.
func TestBuildSharedTailwindUsesGoCommand(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseShimDir := filepath.Join(parseRepoRoot, "shim")
	if parseErr := os.MkdirAll(parseShimDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(shim): %v", parseErr)
	}
	parseShimPath := filepath.Join(parseShimDir, "go.cmd")
	parseOriginalPath := os.Getenv("PATH")
	parseT.Cleanup(func() { _ = os.Setenv("PATH", parseOriginalPath) })
	if parseErr2 := os.Setenv("PATH", parseShimDir+string(os.PathListSeparator)+parseOriginalPath); parseErr2 != nil {
		parseT.Fatalf("Setenv(PATH): %v", parseErr2)
	}

	parseArgsPath := filepath.Join(parseRepoRoot, "tailwind-args.txt")
	parseSuccessShim := "@echo off\r\nsetlocal\r\necho %* > \"" + parseArgsPath + "\"\r\nexit /b 0\r\n"
	if parseErr3 := os.WriteFile(parseShimPath, []byte(parseSuccessShim), 0o644); parseErr3 != nil {
		parseT.Fatalf("WriteFile(success shim): %v", parseErr3)
	}

	if parseErr4 := buildSharedTailwind(parseRepoRoot); parseErr4 != nil {
		parseT.Fatalf("buildSharedTailwind(success): %v", parseErr4)
	}
	parseArgsBytes, parseErr5 := os.ReadFile(parseArgsPath)
	if parseErr5 != nil {
		parseT.Fatalf("ReadFile(args): %v", parseErr5)
	}
	if parseGot := string(parseArgsBytes); !strings.Contains(parseGot, "run ./tools/gwc tailwind") {
		parseT.Fatalf("expected go shim args, got %q", parseGot)
	}

	if parseErr6 := os.WriteFile(parseShimPath, []byte("@echo off\r\nexit /b 9\r\n"), 0o644); parseErr6 != nil {
		parseT.Fatalf("WriteFile(failure shim): %v", parseErr6)
	}
	if parseErr7 := buildSharedTailwind(parseRepoRoot); parseErr7 == nil || !strings.Contains(parseErr7.Error(), "build shared tailwind css") {
		parseT.Fatalf("expected tailwind build error, got %v", parseErr7)
	}
}
