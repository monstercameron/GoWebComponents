package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
)

type config struct {
	Addr              string
	RepoRoot          string
	ExampleRoot       string
	FallbackSchema    string
	MigrationsDir     string
	SQLitePath        string
	StaticDir         string
	TailwindCSS       string
	WASMExecJS        string
	ExampleLoggerJS   string
	AtlasWASM         string
	DefaultPublicHost string
}

func loadConfig() (config, error) {
	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		return config{}, fmt.Errorf("get working directory: %w", parseErr)
	}
	parseRepoRoot, parseErr := findRepoRoot(parseWorkingDir)
	if parseErr != nil {
		return config{}, parseErr
	}
	parseExampleRoot := filepath.Join(parseRepoRoot, "examples", "86-atlas-commerce-os")
	parseStaticDir := filepath.Join(parseRepoRoot, "examples", "static")
	parseExamplesWasmDir, parseErr := runnerconfig.ResolveWorkspaceBuildPath(parseRepoRoot, runnerconfig.FS{}, "examples")
	if parseErr != nil {
		return config{}, fmt.Errorf("resolve examples build root: %w", parseErr)
	}
	parseAddr := strings.TrimSpace(os.Getenv("ATLAS_ADDR"))
	if parseAddr == "" {
		parseAddr = "127.0.0.1:8096"
	}
	return config{
		Addr:              parseAddr,
		RepoRoot:          parseRepoRoot,
		ExampleRoot:       parseExampleRoot,
		FallbackSchema:    filepath.Join(parseExampleRoot, "server", "data", "schema.sql"),
		MigrationsDir:     filepath.Join(parseExampleRoot, "server", "data", "migrations"),
		SQLitePath:        filepath.Join(parseExampleRoot, "server", "data", "atlas-commerce-os.db"),
		StaticDir:         parseStaticDir,
		TailwindCSS:       filepath.Join(parseStaticDir, "css", "tailwind.css"),
		WASMExecJS:        filepath.Join(parseStaticDir, "script", "wasm_exec.js"),
		ExampleLoggerJS:   filepath.Join(parseStaticDir, "script", "example-logger.js"),
		AtlasWASM:         filepath.Join(parseExamplesWasmDir, "atlas-commerce-os.wasm"),
		DefaultPublicHost: "http://127.0.0.1:8096",
	}, nil
}

func findRepoRoot(parseStart string) (string, error) {
	parseCurrent := parseStart
	for {
		if _, parseErr := os.Stat(filepath.Join(parseCurrent, "go.mod")); parseErr == nil {
			return parseCurrent, nil
		}
		parseParent := filepath.Dir(parseCurrent)
		if parseParent == parseCurrent {
			return "", fmt.Errorf("could not locate repo root from %s", parseStart)
		}
		parseCurrent = parseParent
	}
}
