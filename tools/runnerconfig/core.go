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

func (parseFs FS) withDefaults() FS {
	if parseFs.Getwd == nil {
		parseFs.Getwd = os.Getwd
	}
	if parseFs.UserHomeDir == nil {
		parseFs.UserHomeDir = os.UserHomeDir
	}
	if parseFs.ReadFile == nil {
		parseFs.ReadFile = os.ReadFile
	}
	if parseFs.Stat == nil {
		parseFs.Stat = os.Stat
	}
	return parseFs
}

func (parseFs FS) pathExists(parsePath string) bool {
	parseFs = parseFs.withDefaults()
	if parsePath == "" {
		return false
	}
	_, parseErr := parseFs.Stat(parsePath)
	return parseErr == nil
}

// Load reads and parses the runner override file found from cwd.
func Load(parseCwd string, parseFs FS) (Overrides, string, bool, error) {
	parseFs = parseFs.withDefaults()
	parseConfigPath, parseErr := ResolveConfigPath(parseCwd, parseFs)
	if parseErr != nil {
		return Overrides{}, "", false, parseErr
	}
	if strings.TrimSpace(parseConfigPath) == "" {
		return Overrides{}, "", false, nil
	}
	parseContent, parseErr := parseFs.ReadFile(parseConfigPath)
	if parseErr != nil {
		return Overrides{}, "", false, fmt.Errorf("read launcher override file: %w", parseErr)
	}
	var parseOverrides Overrides
	if parseErr2 := json.Unmarshal(parseContent, &parseOverrides); parseErr2 != nil {
		return Overrides{}, "", false, fmt.Errorf("parse launcher override file: %w", parseErr2)
	}
	return parseOverrides, parseConfigPath, true, nil
}

// ResolveConfigPath determines the absolute path to the runner config file starting from cwd.
func ResolveConfigPath(parseCwd string, parseFs FS) (string, error) {
	parseFs = parseFs.withDefaults()
	if parseExplicit := strings.TrimSpace(os.Getenv(OverrideEnvVar)); parseExplicit != "" {
		if !filepath.IsAbs(parseExplicit) {
			parseBase := parseCwd
			if strings.TrimSpace(parseBase) == "" {
				parseBase = "."
			}
			parseResolved, parseErr := filepath.Abs(filepath.Join(parseBase, parseExplicit))
			if parseErr != nil {
				return "", fmt.Errorf("resolve %s path: %w", OverrideEnvVar, parseErr)
			}
			parseExplicit = parseResolved
		}
		return parseExplicit, nil
	}
	if parseFound := LocateConfigInParents(parseCwd, parseFs); parseFound != "" {
		return parseFound, nil
	}
	parseHomeDir, parseErr2 := parseFs.UserHomeDir()
	if parseErr2 == nil && strings.TrimSpace(parseHomeDir) != "" {
		parseCandidate := filepath.Join(parseHomeDir, ".gwc", "runner.json")
		if parseFs.pathExists(parseCandidate) {
			return parseCandidate, nil
		}
	}
	return "", nil
}

// LocateConfigInParents walks up the directory tree from cwd looking for a gwc-runner.json file.
func LocateConfigInParents(parseCwd string, parseFs FS) string {
	parseFs = parseFs.withDefaults()
	parseCurrent := strings.TrimSpace(parseCwd)
	if parseCurrent == "" {
		return ""
	}
	parseResolved, parseErr := filepath.Abs(parseCurrent)
	if parseErr != nil {
		return ""
	}
	for {
		parseCandidate := filepath.Join(parseResolved, "gwc-runner.json")
		if parseFs.pathExists(parseCandidate) {
			return parseCandidate
		}
		parseParent := filepath.Dir(parseResolved)
		if parseParent == parseResolved {
			return ""
		}
		parseResolved = parseParent
	}
}

// ResolveValue resolves a raw config path value relative to the config file location.
func ResolveValue(parseConfigPath string, parseRaw string) (string, error) {
	parseRaw = strings.TrimSpace(parseRaw)
	if parseRaw == "" {
		return "", nil
	}
	if filepath.IsAbs(parseRaw) {
		return filepath.Clean(parseRaw), nil
	}
	parseBaseDir := "."
	if strings.TrimSpace(parseConfigPath) != "" {
		parseBaseDir = filepath.Dir(parseConfigPath)
	}
	parseResolved, parseErr := filepath.Abs(filepath.Join(parseBaseDir, parseRaw))
	if parseErr != nil {
		return "", parseErr
	}
	return parseResolved, nil
}

// ResolveConfiguredPath resolves a path from the runner config using the given selector.
func ResolveConfiguredPath(parseCwd string, parseSelector func(Paths) string, parseLabel string, parseFs FS) (string, bool, error) {
	parseOverrides, parseConfigPath, parseOk, parseErr := Load(parseCwd, parseFs)
	if parseErr != nil {
		return "", false, parseErr
	}
	if !parseOk {
		return "", false, nil
	}
	parseOverridePath, parseErr := ResolveValue(parseConfigPath, parseSelector(parseOverrides.Paths))
	if parseErr != nil {
		return "", false, fmt.Errorf("resolve %s override: %w", parseLabel, parseErr)
	}
	if strings.TrimSpace(parseOverridePath) == "" {
		return "", false, nil
	}
	return parseOverridePath, true, nil
}

// ResolveArtifactRoot resolves the configured artifact root directory for the workspace.
func ResolveArtifactRoot(parseCwd string, parseFs FS) (string, bool, error) {
	return ResolveConfiguredPath(parseCwd, func(parsePaths Paths) string {
		return parsePaths.ArtifactRoot
	}, "artifactRoot", parseFs)
}

// ResolveWorkspaceBuildRoot resolves the build output root for the workspace, defaulting to bin/.
func ResolveWorkspaceBuildRoot(parseRootPath string, parseFs FS) (string, error) {
	parseConfigured, parseOk, parseErr := ResolveConfiguredPath(parseRootPath, func(parsePaths Paths) string {
		return parsePaths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", parseFs)
	if parseErr != nil {
		return "", parseErr
	}
	if parseOk {
		return parseConfigured, nil
	}
	parseCleaned := filepath.Clean(strings.TrimSpace(parseRootPath))
	if parseCleaned == "" || parseCleaned == "." {
		parseCleaned = "bin"
		if filepath.IsAbs(parseCleaned) {
			return parseCleaned, nil
		}
		parseResolved, parseResolveErr := filepath.Abs(parseCleaned)
		if parseResolveErr != nil {
			return "", parseResolveErr
		}
		return parseResolved, nil
	}
	return filepath.Join(parseCleaned, "bin"), nil
}

// ResolveWorkspaceBuildPath resolves a path within the workspace build root.
func ResolveWorkspaceBuildPath(parseRootPath string, parseFs FS, parseSegments ...string) (string, error) {
	buildRoot, parseErr := ResolveWorkspaceBuildRoot(parseRootPath, parseFs)
	if parseErr != nil {
		return "", parseErr
	}
	parseParts := []string{buildRoot}
	parseParts = append(parseParts, parseSegments...)
	return filepath.Join(parseParts...), nil
}

// GetArtifactNamespace returns the artifact namespace derived from the workspace root path.
func GetArtifactNamespace(parseRootPath string) string {
	parseCleaned := filepath.Clean(strings.TrimSpace(parseRootPath))
	if parseCleaned == "" || parseCleaned == "." {
		return "workspace"
	}
	parseBase := filepath.Base(parseCleaned)
	if parseBase == "" || parseBase == "." || parseBase == string(filepath.Separator) {
		return "workspace"
	}
	return parseBase
}

// ResolveArtifactPath resolves the path to an artifact within the configured artifact root.
func ResolveArtifactPath(parseRootPath string, parseFs FS, parseSegments ...string) (string, bool, error) {
	parseArtifactRoot, parseOk, parseErr := ResolveArtifactRoot(parseRootPath, parseFs)
	if parseErr != nil || !parseOk {
		return "", parseOk, parseErr
	}
	parseParts := []string{parseArtifactRoot, GetArtifactNamespace(parseRootPath)}
	parseParts = append(parseParts, parseSegments...)
	return filepath.Join(parseParts...), true, nil
}
