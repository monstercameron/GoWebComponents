package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot, err := findRepoRoot(workingDir)
	if err != nil {
		t.Fatalf("findRepoRoot current dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		t.Fatalf("expected go.mod at repo root %q: %v", repoRoot, err)
	}

	nested := filepath.Join(repoRoot, "examples", "86-atlas-commerce-os", "server", "data")
	fromNested, err := findRepoRoot(nested)
	if err != nil {
		t.Fatalf("findRepoRoot nested dir: %v", err)
	}
	if fromNested != repoRoot {
		t.Fatalf("findRepoRoot nested = %q, want %q", fromNested, repoRoot)
	}

	missingRoot := t.TempDir()
	if _, err := findRepoRoot(missingRoot); err == nil {
		t.Fatal("expected findRepoRoot to fail outside a repo tree")
	}
}

func TestLoadConfigDefaultsAndEnvOverride(t *testing.T) {
	originalAddr, hadAddr := os.LookupEnv("ATLAS_ADDR")
	t.Cleanup(func() {
		if hadAddr {
			_ = os.Setenv("ATLAS_ADDR", originalAddr)
		} else {
			_ = os.Unsetenv("ATLAS_ADDR")
		}
	})

	_ = os.Unsetenv("ATLAS_ADDR")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig default: %v", err)
	}
	if cfg.Addr != "127.0.0.1:8096" {
		t.Fatalf("default Addr = %q, want 127.0.0.1:8096", cfg.Addr)
	}
	if cfg.DefaultPublicHost != "http://127.0.0.1:8096" {
		t.Fatalf("DefaultPublicHost = %q, want http://127.0.0.1:8096", cfg.DefaultPublicHost)
	}
	if filepath.Base(cfg.RepoRoot) != "GoWebComponents" {
		t.Fatalf("unexpected RepoRoot: %q", cfg.RepoRoot)
	}
	if cfg.ExampleRoot != filepath.Join(cfg.RepoRoot, "examples", "86-atlas-commerce-os") {
		t.Fatalf("unexpected ExampleRoot: %q", cfg.ExampleRoot)
	}
	if cfg.FallbackSchema != filepath.Join(cfg.ExampleRoot, "server", "data", "schema.sql") {
		t.Fatalf("unexpected FallbackSchema: %q", cfg.FallbackSchema)
	}
	if cfg.MigrationsDir != filepath.Join(cfg.ExampleRoot, "server", "data", "migrations") {
		t.Fatalf("unexpected MigrationsDir: %q", cfg.MigrationsDir)
	}
	if cfg.SQLitePath != filepath.Join(cfg.ExampleRoot, "server", "data", "atlas-commerce-os.db") {
		t.Fatalf("unexpected SQLitePath: %q", cfg.SQLitePath)
	}
	if cfg.StaticDir != filepath.Join(cfg.RepoRoot, "examples", "static") {
		t.Fatalf("unexpected StaticDir: %q", cfg.StaticDir)
	}
	if cfg.TailwindCSS != filepath.Join(cfg.StaticDir, "css", "tailwind.css") {
		t.Fatalf("unexpected TailwindCSS: %q", cfg.TailwindCSS)
	}
	if cfg.WASMExecJS != filepath.Join(cfg.StaticDir, "script", "wasm_exec.js") {
		t.Fatalf("unexpected WASMExecJS: %q", cfg.WASMExecJS)
	}
	if cfg.ExampleLoggerJS != filepath.Join(cfg.StaticDir, "script", "example-logger.js") {
		t.Fatalf("unexpected ExampleLoggerJS: %q", cfg.ExampleLoggerJS)
	}
	if filepath.Base(cfg.AtlasWASM) != "atlas-commerce-os.wasm" {
		t.Fatalf("unexpected AtlasWASM path: %q", cfg.AtlasWASM)
	}

	if err := os.Setenv("ATLAS_ADDR", " 127.0.0.1:9001 "); err != nil {
		t.Fatalf("Setenv ATLAS_ADDR: %v", err)
	}
	cfg, err = loadConfig()
	if err != nil {
		t.Fatalf("loadConfig env override: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9001" {
		t.Fatalf("overridden Addr = %q, want 127.0.0.1:9001", cfg.Addr)
	}
}
