package runnerconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveConfigPath(cwd string, fs FS) (string, error) {
	fs = fs.withDefaults()
	if explicit := strings.TrimSpace(os.Getenv(OverrideEnvVar)); explicit != "" {
		if !filepath.IsAbs(explicit) {
			base := cwd
			if strings.TrimSpace(base) == "" {
				base = "."
			}
			resolved, err := filepath.Abs(filepath.Join(base, explicit))
			if err != nil {
				return "", fmt.Errorf("resolve %s path: %w", OverrideEnvVar, err)
			}
			explicit = resolved
		}
		return explicit, nil
	}
	if found := FindConfigInParents(cwd, fs); found != "" {
		return found, nil
	}
	homeDir, err := fs.UserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		candidate := filepath.Join(homeDir, ".gwc", "runner.json")
		if fs.pathExists(candidate) {
			return candidate, nil
		}
	}
	return "", nil
}

func FindConfigInParents(cwd string, fs FS) string {
	fs = fs.withDefaults()
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
		if fs.pathExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(resolved)
		if parent == resolved {
			return ""
		}
		resolved = parent
	}
}

func ResolveWorkspaceBuildRoot(rootPath string, fs FS) (string, error) {
	configured, ok, err := ResolveConfiguredPath(rootPath, func(paths Paths) string {
		return paths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", fs)
	if err != nil {
		return "", err
	}
	if ok {
		return configured, nil
	}
	cleaned := filepath.Clean(strings.TrimSpace(rootPath))
	if cleaned == "" || cleaned == "." {
		cleaned = "bin"
		if filepath.IsAbs(cleaned) {
			return cleaned, nil
		}
		resolved, resolveErr := filepath.Abs(cleaned)
		if resolveErr != nil {
			return "", resolveErr
		}
		return resolved, nil
	}
	return filepath.Join(cleaned, "bin"), nil
}

func ResolveWorkspaceBuildPath(rootPath string, fs FS, segments ...string) (string, error) {
	buildRoot, err := ResolveWorkspaceBuildRoot(rootPath, fs)
	if err != nil {
		return "", err
	}
	parts := []string{buildRoot}
	parts = append(parts, segments...)
	return filepath.Join(parts...), nil
}
