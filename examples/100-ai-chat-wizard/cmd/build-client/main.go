package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/andybalholm/brotli"
)

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	legacyArtifacts := []string{
		filepath.Join(repoRoot, "examples", "100-ai-chat-wizard", "client", "chat.wasm"),
		filepath.Join(repoRoot, "examples", "100-ai-chat-wizard", "client", "backgroundworker", "background-worker.wasm"),
	}
	for _, legacyPath := range legacyArtifacts {
		if err := removeLegacyArtifact(legacyPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	targets := []struct {
		label       string
		packagePath string
		outputPath  string
	}{
		{
			label:       "chat client",
			packagePath: "./examples/100-ai-chat-wizard/client",
			outputPath:  filepath.Join(repoRoot, "examples", "100-ai-chat-wizard", "bin", "client", "app", "chat.wasm"),
		},
		{
			label:       "background worker",
			packagePath: "./examples/100-ai-chat-wizard/client/backgroundworker",
			outputPath:  filepath.Join(repoRoot, "examples", "100-ai-chat-wizard", "bin", "client", "worker", "background-worker.wasm"),
		},
	}

	for _, target := range targets {
		if err := buildTarget(repoRoot, target.label, target.packagePath, target.outputPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := writeBrotliSidecar(target.outputPath, target.outputPath+".br"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func removeLegacyArtifact(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove legacy artifact %s: %w", path, err)
	}
	fmt.Printf("removed legacy artifact: %s\n", path)
	return nil
}

func findRepoRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	currentDir := workingDir
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("find repo root: go.mod not found from %s upward", workingDir)
		}
		currentDir = parentDir
	}
}

func buildTarget(repoRoot string, label string, packagePath string, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("prepare output directory for %s: %w", label, err)
	}

	command := exec.Command("go", "build", "-o", outputPath, packagePath)
	command.Dir = repoRoot
	command.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("build %s: %w", label, err)
	}

	artifactInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("stat %s artifact: %w", label, err)
	}
	fmt.Printf("built %s: %s (%d bytes)\n", label, outputPath, artifactInfo.Size())
	return nil
}

func writeBrotliSidecar(sourcePath string, targetPath string) error {
	inputBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source artifact for brotli: %w", err)
	}
	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), filepath.Base(targetPath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create brotli sidecar: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	brotliWriter := brotli.NewWriterLevel(tempFile, brotli.BestCompression)
	if _, err := brotliWriter.Write(inputBytes); err != nil {
		brotliWriter.Close()
		return fmt.Errorf("write brotli sidecar: %w", err)
	}
	if err := brotliWriter.Close(); err != nil {
		return fmt.Errorf("finalize brotli sidecar: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close brotli sidecar temp file: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("replace brotli sidecar: %w", err)
	}

	artifactInfo, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("stat brotli sidecar: %w", err)
	}
	compressionRatio := 0.0
	if len(inputBytes) > 0 {
		compressionRatio = (float64(artifactInfo.Size()) / float64(len(inputBytes))) * 100
	}
	fmt.Printf("built brotli sidecar: %s (%d bytes, %.2f%% of raw)\n", targetPath, artifactInfo.Size(), compressionRatio)
	return nil
}
