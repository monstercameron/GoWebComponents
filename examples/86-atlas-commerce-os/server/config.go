package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	workingDir, err := os.Getwd()
	if err != nil {
		return config{}, fmt.Errorf("get working directory: %w", err)
	}
	repoRoot, err := findRepoRoot(workingDir)
	if err != nil {
		return config{}, err
	}
	exampleRoot := filepath.Join(repoRoot, "examples", "86-atlas-commerce-os")
	staticDir := filepath.Join(repoRoot, "examples", "static")
	addr := strings.TrimSpace(os.Getenv("ATLAS_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:8096"
	}
	return config{
		Addr:              addr,
		RepoRoot:          repoRoot,
		ExampleRoot:       exampleRoot,
		FallbackSchema:    filepath.Join(exampleRoot, "server", "data", "schema.sql"),
		MigrationsDir:     filepath.Join(exampleRoot, "server", "data", "migrations"),
		SQLitePath:        filepath.Join(exampleRoot, "server", "data", "atlas-commerce-os.db"),
		StaticDir:         staticDir,
		TailwindCSS:       filepath.Join(staticDir, "css", "tailwind.css"),
		WASMExecJS:        filepath.Join(staticDir, "script", "wasm_exec.js"),
		ExampleLoggerJS:   filepath.Join(staticDir, "script", "example-logger.js"),
		AtlasWASM:         filepath.Join(staticDir, "bin", "atlas-commerce-os.wasm"),
		DefaultPublicHost: "http://127.0.0.1:8096",
	}, nil
}

func findRepoRoot(start string) (string, error) {
	current := start
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not locate repo root from %s", start)
		}
		current = parent
	}
}
