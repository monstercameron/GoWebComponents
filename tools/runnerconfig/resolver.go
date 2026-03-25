package runnerconfig

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ResolveValue(configPath string, raw string) (string, error) {
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

func ResolveConfiguredPath(cwd string, selector func(Paths) string, label string, fs FS) (string, bool, error) {
	overrides, configPath, ok, err := Load(cwd, fs)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	overridePath, err := ResolveValue(configPath, selector(overrides.Paths))
	if err != nil {
		return "", false, fmt.Errorf("resolve %s override: %w", label, err)
	}
	if strings.TrimSpace(overridePath) == "" {
		return "", false, nil
	}
	return overridePath, true, nil
}

func ResolveArtifactRoot(cwd string, fs FS) (string, bool, error) {
	return ResolveConfiguredPath(cwd, func(paths Paths) string {
		return paths.ArtifactRoot
	}, "artifactRoot", fs)
}

func ArtifactNamespace(rootPath string) string {
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

func ResolveArtifactPath(rootPath string, fs FS, segments ...string) (string, bool, error) {
	artifactRoot, ok, err := ResolveArtifactRoot(rootPath, fs)
	if err != nil || !ok {
		return "", ok, err
	}
	parts := []string{artifactRoot, ArtifactNamespace(rootPath)}
	parts = append(parts, segments...)
	return filepath.Join(parts...), true, nil
}
