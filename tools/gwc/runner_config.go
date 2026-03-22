package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const launcherOverrideEnvVar = "GWC_RUNNER_CONFIG"

type launcherOverridePaths struct {
	GeneratedProjectRoot   string `json:"generatedProjectRoot,omitempty"`
	ArtifactRoot           string `json:"artifactRoot,omitempty"`
	WASMExecJS             string `json:"wasmExecJS,omitempty"`
	GoWASMExec             string `json:"goWasmExec,omitempty"`
	BrowserWorkspace       string `json:"browserWorkspace,omitempty"`
	LivereloadWorkspace    string `json:"livereloadWorkspace,omitempty"`
	LivereloadClientScript string `json:"livereloadClientScript,omitempty"`
}

type launcherOverrides struct {
	Paths launcherOverridePaths `json:"paths,omitempty"`
}

var launcherConfigGetwd = os.Getwd

var launcherConfigUserHomeDir = os.UserHomeDir

func loadLauncherOverrides(cwd string) (launcherOverrides, string, bool, error) {
	configPath, err := resolveLauncherOverridePath(cwd)
	if err != nil {
		return launcherOverrides{}, "", false, err
	}
	if strings.TrimSpace(configPath) == "" {
		return launcherOverrides{}, "", false, nil
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		return launcherOverrides{}, "", false, fmt.Errorf("read launcher override file: %w", err)
	}
	var overrides launcherOverrides
	if err := json.Unmarshal(content, &overrides); err != nil {
		return launcherOverrides{}, "", false, fmt.Errorf("parse launcher override file: %w", err)
	}
	return overrides, configPath, true, nil
}

func loadLauncherOverridesForCurrentContext() (launcherOverrides, string, bool, error) {
	cwd, err := launcherConfigGetwd()
	if err != nil {
		cwd = ""
	}
	return loadLauncherOverrides(cwd)
}

func resolveLauncherOverridePath(cwd string) (string, error) {
	if explicit := strings.TrimSpace(os.Getenv(launcherOverrideEnvVar)); explicit != "" {
		if !filepath.IsAbs(explicit) {
			base := cwd
			if strings.TrimSpace(base) == "" {
				base = "."
			}
			resolved, err := filepath.Abs(filepath.Join(base, explicit))
			if err != nil {
				return "", fmt.Errorf("resolve %s path: %w", launcherOverrideEnvVar, err)
			}
			explicit = resolved
		}
		return explicit, nil
	}
	if found := findLauncherOverrideInParents(cwd); found != "" {
		return found, nil
	}
	homeDir, err := launcherConfigUserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		candidate := filepath.Join(homeDir, ".gwc", "runner.json")
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return "", nil
}

func findLauncherOverrideInParents(cwd string) string {
	current := strings.TrimSpace(cwd)
	if current == "" {
		return ""
	}
	resolved, err := filepath.Abs(current)
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(resolved, "gwc-runner.json")
		if fileExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(resolved)
		if parent == resolved {
			return ""
		}
		resolved = parent
	}
}

func resolveLauncherOverrideValue(configPath string, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), nil
	}
	baseDir := "."
	if strings.TrimSpace(configPath) != "" {
		baseDir = filepath.Dir(configPath)
	}
	resolved, err := filepath.Abs(filepath.Join(baseDir, raw))
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func resolveLauncherConfiguredPath(rootPath string, selector func(launcherOverridePaths) string, label string) (string, bool, error) {
	overrides, configPath, ok, err := loadLauncherOverrides(rootPath)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	overridePath, err := resolveLauncherOverrideValue(configPath, selector(overrides.Paths))
	if err != nil {
		return "", false, fmt.Errorf("resolve %s override: %w", label, err)
	}
	if strings.TrimSpace(overridePath) == "" {
		return "", false, nil
	}
	return overridePath, true, nil
}

func resolveLauncherArtifactRoot(rootPath string) (string, bool, error) {
	return resolveLauncherConfiguredPath(rootPath, func(paths launcherOverridePaths) string {
		return paths.ArtifactRoot
	}, "artifactRoot")
}

func launcherArtifactNamespace(rootPath string) string {
	cleaned := filepath.Clean(strings.TrimSpace(rootPath))
	if cleaned == "" || cleaned == "." {
		return "workspace"
	}
	base := filepath.Base(cleaned)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "workspace"
	}
	return base
}

func resolveLauncherArtifactPath(rootPath string, segments ...string) (string, bool, error) {
	artifactRoot, ok, err := resolveLauncherArtifactRoot(rootPath)
	if err != nil || !ok {
		return "", ok, err
	}
	parts := []string{artifactRoot, launcherArtifactNamespace(rootPath)}
	parts = append(parts, segments...)
	return filepath.Join(parts...), true, nil
}
