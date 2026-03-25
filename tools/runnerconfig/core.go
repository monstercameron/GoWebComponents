package runnerconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const OverrideEnvVar = "GWC_RUNNER_CONFIG"

type Paths struct {
	GeneratedProjectRoot   string `json:"generatedProjectRoot,omitempty"`
	ArtifactRoot           string `json:"artifactRoot,omitempty"`
	WorkspaceBuildRoot     string `json:"workspaceBuildRoot,omitempty"`
	WASMExecJS             string `json:"wasmExecJS,omitempty"`
	GoWASMExec             string `json:"goWasmExec,omitempty"`
	BrowserWorkspace       string `json:"browserWorkspace,omitempty"`
	LivereloadWorkspace    string `json:"livereloadWorkspace,omitempty"`
	LivereloadClientScript string `json:"livereloadClientScript,omitempty"`
}

type Overrides struct {
	Paths Paths `json:"paths,omitempty"`
}

type FS struct {
	Getwd       func() (string, error)
	UserHomeDir func() (string, error)
	ReadFile    func(string) ([]byte, error)
	Stat        func(string) (os.FileInfo, error)
}

func (fs FS) withDefaults() FS {
	if fs.Getwd == nil {
		fs.Getwd = os.Getwd
	}
	if fs.UserHomeDir == nil {
		fs.UserHomeDir = os.UserHomeDir
	}
	if fs.ReadFile == nil {
		fs.ReadFile = os.ReadFile
	}
	if fs.Stat == nil {
		fs.Stat = os.Stat
	}
	return fs
}

func (fs FS) pathExists(path string) bool {
	fs = fs.withDefaults()
	if path == "" {
		return false
	}
	_, err := fs.Stat(path)
	return err == nil
}

// Load reads and parses the runner override file found from cwd.
func Load(cwd string, fs FS) (Overrides, string, bool, error) {
	fs = fs.withDefaults()
	configPath, err := ResolveConfigPath(cwd, fs)
	if err != nil {
		return Overrides{}, "", false, err
	}
	if strings.TrimSpace(configPath) == "" {
		return Overrides{}, "", false, nil
	}
	content, err := fs.ReadFile(configPath)
	if err != nil {
		return Overrides{}, "", false, fmt.Errorf("read launcher override file: %w", err)
	}
	var overrides Overrides
	if err := json.Unmarshal(content, &overrides); err != nil {
		return Overrides{}, "", false, fmt.Errorf("parse launcher override file: %w", err)
	}
	return overrides, configPath, true, nil
}

// ResolveConfigPath determines the absolute path to the runner config file starting from cwd.
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
	if found := LocateConfigInParents(cwd, fs); found != "" {
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

// LocateConfigInParents walks up the directory tree from cwd looking for a gwc-runner.json file.
func LocateConfigInParents(cwd string, fs FS) string {
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

// ResolveValue resolves a raw config path value relative to the config file location.
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

// ResolveConfiguredPath resolves a path from the runner config using the given selector.
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

// ResolveArtifactRoot resolves the configured artifact root directory for the workspace.
func ResolveArtifactRoot(cwd string, fs FS) (string, bool, error) {
	return ResolveConfiguredPath(cwd, func(paths Paths) string {
		return paths.ArtifactRoot
	}, "artifactRoot", fs)
}

// ResolveWorkspaceBuildRoot resolves the build output root for the workspace, defaulting to bin/.
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

// ResolveWorkspaceBuildPath resolves a path within the workspace build root.
func ResolveWorkspaceBuildPath(rootPath string, fs FS, segments ...string) (string, error) {
	buildRoot, err := ResolveWorkspaceBuildRoot(rootPath, fs)
	if err != nil {
		return "", err
	}
	parts := []string{buildRoot}
	parts = append(parts, segments...)
	return filepath.Join(parts...), nil
}

// GetArtifactNamespace returns the artifact namespace derived from the workspace root path.
func GetArtifactNamespace(rootPath string) string {
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

// ResolveArtifactPath resolves the path to an artifact within the configured artifact root.
func ResolveArtifactPath(rootPath string, fs FS, segments ...string) (string, bool, error) {
	artifactRoot, ok, err := ResolveArtifactRoot(rootPath, fs)
	if err != nil || !ok {
		return "", ok, err
	}
	parts := []string{artifactRoot, GetArtifactNamespace(rootPath)}
	parts = append(parts, segments...)
	return filepath.Join(parts...), true, nil
}
