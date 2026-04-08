package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(parseT *testing.T) {
	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	parseRepoRoot, parseErr := findRepoRoot(parseWorkingDir)
	if parseErr != nil {
		parseT.Fatalf("findRepoRoot current dir: %v", parseErr)
	}
	if _, parseErr2 := os.Stat(filepath.Join(parseRepoRoot, "go.mod")); parseErr2 != nil {
		parseT.Fatalf("expected go.mod at repo root %q: %v", parseRepoRoot, parseErr2)
	}

	parseNested := filepath.Join(parseRepoRoot, "examples", "server", "atlas-commerce-os", "server", "data")
	parseFromNested, parseErr := findRepoRoot(parseNested)
	if parseErr != nil {
		parseT.Fatalf("findRepoRoot nested dir: %v", parseErr)
	}
	if parseFromNested != parseRepoRoot {
		parseT.Fatalf("findRepoRoot nested = %q, want %q", parseFromNested, parseRepoRoot)
	}

	parseMissingRoot := parseT.TempDir()
	if _, parseErr3 := findRepoRoot(parseMissingRoot); parseErr3 == nil {
		parseT.Fatal("expected findRepoRoot to fail outside a repo tree")
	}
}

func TestLoadConfigDefaultsAndEnvOverride(parseT *testing.T) {
	parseOriginalAddr, parseHadAddr := os.LookupEnv("ATLAS_ADDR")
	parseT.Cleanup(func() {
		if parseHadAddr {
			_ = os.Setenv("ATLAS_ADDR", parseOriginalAddr)
		} else {
			_ = os.Unsetenv("ATLAS_ADDR")
		}
	})

	_ = os.Unsetenv("ATLAS_ADDR")
	parseCfg, parseErr := loadConfig()
	if parseErr != nil {
		parseT.Fatalf("loadConfig default: %v", parseErr)
	}
	if parseCfg.Addr != "127.0.0.1:8096" {
		parseT.Fatalf("default Addr = %q, want 127.0.0.1:8096", parseCfg.Addr)
	}
	if parseCfg.DefaultPublicHost != "http://127.0.0.1:8096" {
		parseT.Fatalf("DefaultPublicHost = %q, want http://127.0.0.1:8096", parseCfg.DefaultPublicHost)
	}
	if filepath.Base(parseCfg.RepoRoot) != "GoWebComponents" {
		parseT.Fatalf("unexpected RepoRoot: %q", parseCfg.RepoRoot)
	}
	if parseCfg.ExampleRoot != filepath.Join(parseCfg.RepoRoot, "examples", "server", "atlas-commerce-os") {
		parseT.Fatalf("unexpected ExampleRoot: %q", parseCfg.ExampleRoot)
	}
	if parseCfg.FallbackSchema != filepath.Join(parseCfg.ExampleRoot, "server", "data", "schema.sql") {
		parseT.Fatalf("unexpected FallbackSchema: %q", parseCfg.FallbackSchema)
	}
	if parseCfg.MigrationsDir != filepath.Join(parseCfg.ExampleRoot, "server", "data", "migrations") {
		parseT.Fatalf("unexpected MigrationsDir: %q", parseCfg.MigrationsDir)
	}
	if parseCfg.SQLitePath != filepath.Join(parseCfg.ExampleRoot, "server", "data", "atlas-commerce-os.db") {
		parseT.Fatalf("unexpected SQLitePath: %q", parseCfg.SQLitePath)
	}
	if parseCfg.StaticDir != filepath.Join(parseCfg.RepoRoot, "examples", "static") {
		parseT.Fatalf("unexpected StaticDir: %q", parseCfg.StaticDir)
	}
	if parseCfg.TailwindCSS != filepath.Join(parseCfg.StaticDir, "css", "tailwind.css") {
		parseT.Fatalf("unexpected TailwindCSS: %q", parseCfg.TailwindCSS)
	}
	if parseCfg.WASMExecJS != filepath.Join(parseCfg.StaticDir, "script", "wasm_exec.js") {
		parseT.Fatalf("unexpected WASMExecJS: %q", parseCfg.WASMExecJS)
	}
	if parseCfg.ExampleLoggerJS != filepath.Join(parseCfg.StaticDir, "script", "example-logger.js") {
		parseT.Fatalf("unexpected ExampleLoggerJS: %q", parseCfg.ExampleLoggerJS)
	}
	if filepath.Base(parseCfg.AtlasWASM) != "atlas-commerce-os.wasm" {
		parseT.Fatalf("unexpected AtlasWASM path: %q", parseCfg.AtlasWASM)
	}

	if parseErr2 := os.Setenv("ATLAS_ADDR", " 127.0.0.1:9001 "); parseErr2 != nil {
		parseT.Fatalf("Setenv ATLAS_ADDR: %v", parseErr2)
	}
	parseCfg, parseErr = loadConfig()
	if parseErr != nil {
		parseT.Fatalf("loadConfig env override: %v", parseErr)
	}
	if parseCfg.Addr != "127.0.0.1:9001" {
		parseT.Fatalf("overridden Addr = %q, want 127.0.0.1:9001", parseCfg.Addr)
	}
}
