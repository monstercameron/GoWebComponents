package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/andybalholm/brotli"
)

var runBuildClientFindRepoRoot = parseFindRepoRoot
var runBuildClientBuildSharedTailwind = buildSharedTailwind
var runBuildClientRemoveLegacyArtifact = parseRemoveLegacyArtifact
var runBuildClientBuildTarget = buildTarget
var runBuildClientWriteBrotliSidecar = parseWriteBrotliSidecar

func main() {
	if parseErr := runBuildClient(); parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		os.Exit(1)
	}
}

// runBuildClient orchestrates the chat wizard client build steps.
func runBuildClient() error {
	parseRepoRoot, parseErr := runBuildClientFindRepoRoot()
	if parseErr != nil {
		return parseErr
	}
	if parseErr2 := runBuildClientBuildSharedTailwind(parseRepoRoot); parseErr2 != nil {
		return parseErr2
	}

	parseLegacyArtifacts := []string{
		filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "client", "chat.wasm"),
		filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "client", "backgroundworker", "background-worker.wasm"),
	}
	for _, parseLegacyPath := range parseLegacyArtifacts {
		if parseErr3 := runBuildClientRemoveLegacyArtifact(parseLegacyPath); parseErr3 != nil {
			return parseErr3
		}
	}

	parseTargets := []struct {
		label       string
		packagePath string
		outputPath  string
	}{
		{
			label:       "chat client",
			packagePath: "./examples/server/ai-chat-wizard/client",
			outputPath:  filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "bin", "client", "app", "chat.wasm"),
		},
		{
			label:       "background worker",
			packagePath: "./examples/server/ai-chat-wizard/client/backgroundworker",
			outputPath:  filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "bin", "client", "worker", "background-worker.wasm"),
		},
	}

	for _, parseTarget := range parseTargets {
		if parseErr4 := runBuildClientBuildTarget(parseRepoRoot, parseTarget.label, parseTarget.packagePath, parseTarget.outputPath); parseErr4 != nil {
			return parseErr4
		}
		if parseErr5 := runBuildClientWriteBrotliSidecar(parseTarget.outputPath, parseTarget.outputPath+".br"); parseErr5 != nil {
			return parseErr5
		}
	}
	return nil
}

// buildSharedTailwind refreshes the shared examples Tailwind CSS before wasm compilation.
func buildSharedTailwind(parseRepoRoot string) error {
	parseCommand := exec.Command("go", "run", "./tools/gwc", "tailwind")
	parseCommand.Dir = parseRepoRoot
	parseCommand.Stdout = os.Stdout
	parseCommand.Stderr = os.Stderr
	if parseErr := parseCommand.Run(); parseErr != nil {
		return fmt.Errorf("build shared tailwind css: %w", parseErr)
	}
	return nil
}

func parseRemoveLegacyArtifact(parsePath string) error {
	if parseErr := os.Remove(parsePath); parseErr != nil {
		if os.IsNotExist(parseErr) {
			return nil
		}
		return fmt.Errorf("remove legacy artifact %s: %w", parsePath, parseErr)
	}
	fmt.Printf("removed legacy artifact: %s\n", parsePath)
	return nil
}

func parseFindRepoRoot() (string, error) {
	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		return "", fmt.Errorf("get working directory: %w", parseErr)
	}

	parseCurrentDir := parseWorkingDir
	for {
		if _, parseErr2 := os.Stat(filepath.Join(parseCurrentDir, "go.mod")); parseErr2 == nil {
			return parseCurrentDir, nil
		}
		parseParentDir := filepath.Dir(parseCurrentDir)
		if parseParentDir == parseCurrentDir {
			return "", fmt.Errorf("find repo root: go.mod not found from %s upward", parseWorkingDir)
		}
		parseCurrentDir = parseParentDir
	}
}

func buildTarget(parseRepoRoot string, parseLabel string, parsePackagePath string, parseOutputPath string) error {
	if parseErr := os.MkdirAll(filepath.Dir(parseOutputPath), 0755); parseErr != nil {
		return fmt.Errorf("prepare output directory for %s: %w", parseLabel, parseErr)
	}

	parseCommand := exec.Command("go", "build", "-o", parseOutputPath, parsePackagePath)
	parseCommand.Dir = parseRepoRoot
	parseCommand.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseCommand.Stdout = os.Stdout
	parseCommand.Stderr = os.Stderr

	if parseErr2 := parseCommand.Run(); parseErr2 != nil {
		return fmt.Errorf("build %s: %w", parseLabel, parseErr2)
	}

	parseArtifactInfo, parseErr3 := os.Stat(parseOutputPath)
	if parseErr3 != nil {
		return fmt.Errorf("stat %s artifact: %w", parseLabel, parseErr3)
	}
	fmt.Printf("built %s: %s (%d bytes)\n", parseLabel, parseOutputPath, parseArtifactInfo.Size())
	return nil
}

func parseWriteBrotliSidecar(parseSourcePath string, parseTargetPath string) error {
	parseInputBytes, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		return fmt.Errorf("read source artifact for brotli: %w", parseErr)
	}
	parseTempFile, parseErr := os.CreateTemp(filepath.Dir(parseTargetPath), filepath.Base(parseTargetPath)+".*.tmp")
	if parseErr != nil {
		return fmt.Errorf("create brotli sidecar: %w", parseErr)
	}
	parseTempPath := parseTempFile.Name()
	defer func() {
		_ = parseTempFile.Close()
		_ = os.Remove(parseTempPath)
	}()

	parseBrotliWriter := brotli.NewWriterLevel(parseTempFile, brotli.BestCompression)
	if _, parseErr2 := parseBrotliWriter.Write(parseInputBytes); parseErr2 != nil {
		parseBrotliWriter.Close()
		return fmt.Errorf("write brotli sidecar: %w", parseErr2)
	}
	if parseErr3 := parseBrotliWriter.Close(); parseErr3 != nil {
		return fmt.Errorf("finalize brotli sidecar: %w", parseErr3)
	}
	if parseErr4 := parseTempFile.Close(); parseErr4 != nil {
		return fmt.Errorf("close brotli sidecar temp file: %w", parseErr4)
	}
	if parseErr5 := os.Rename(parseTempPath, parseTargetPath); parseErr5 != nil {
		return fmt.Errorf("replace brotli sidecar: %w", parseErr5)
	}

	parseArtifactInfo, parseErr := os.Stat(parseTargetPath)
	if parseErr != nil {
		return fmt.Errorf("stat brotli sidecar: %w", parseErr)
	}
	parseCompressionRatio := 0.0
	if len(parseInputBytes) > 0 {
		parseCompressionRatio = (float64(parseArtifactInfo.Size()) / float64(len(parseInputBytes))) * 100
	}
	fmt.Printf("built brotli sidecar: %s (%d bytes, %.2f%% of raw)\n", parseTargetPath, parseArtifactInfo.Size(), parseCompressionRatio)
	return nil
}
