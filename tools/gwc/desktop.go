package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const desktopWailsRevision = "0965e9db574e7b14a447b0ae2bf7ed36a5406462"
const desktopWailsVersion = "v3.0.0-beta.17"

type desktopConfig struct {
	context  context.Context
	timeout  time.Duration
	action   string
	root     string
	features string
	json     bool
}

type desktopSummary struct {
	OK           bool              `json:"ok"`
	Action       string            `json:"action"`
	Root         string            `json:"root"`
	Template     string            `json:"template,omitempty"`
	Artifact     string            `json:"artifact,omitempty"`
	Manifest     string            `json:"manifest,omitempty"`
	WailsVersion string            `json:"wailsVersion,omitempty"`
	Features     string            `json:"features,omitempty"`
	Signed       bool              `json:"signed"`
	Checks       []string          `json:"checks,omitempty"`
	Files        []string          `json:"files,omitempty"`
	Hashes       map[string]string `json:"hashes,omitempty"`
	ToolVersions map[string]string `json:"toolVersions,omitempty"`
}

var desktopTemplateFiles = []string{
	".gitignore", "README.md", "Taskfile.yml", "go.mod", "go.sum", "gwc-start.json",
	"assets/assets.go", "assets/src/app.css", "assets/src/bootstrap.js", "assets/src/bootstrap.web.js", "assets/src/index.html", "assets/src/tester.css",
	"cmd/desktop/main.go", "cmd/desktop/security.go", "contracts/counter.go", "contracts/smoke.go", "contracts/work.go",
	"frontend/main.go", "internal/services/counter.go", "internal/services/edit_menu.go", "internal/services/smoke.go", "internal/services/storage.go", "internal/services/storage_lock_other.go", "internal/services/storage_lock_windows.go", "tools/build/main.go",
	"contracts/api.go", "frontend/tester.go", "internal/services/api.go", "internal/services/api_menu.go",
	"internal/services/files.go",
	"WINDOWS_API_TESTER.md",
}

// runDesktopCommand runs an opt-in desktop action from the isolated template.
func (parseL launcher) runDesktopCommand(parseArgs []string) error {
	if len(parseArgs) == 0 {
		return errors.New("desktop requires an action: init, doctor, build, dev, or package")
	}
	parseAction := strings.ToLower(strings.TrimSpace(parseArgs[0]))
	parseFlags := flag.NewFlagSet("desktop "+parseAction, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stderr)
	parseRoot := parseFlags.String("root", "", "Desktop project root")
	parseJSON := parseFlags.Bool("json", false, "Emit JSON output")
	parseFeatures := parseFlags.String("features", "all", "Native feature ceiling (comma-separated, default all)")
	parseTimeout := parseFlags.Duration("timeout", 5*time.Minute, "Maximum build/prerequisite command duration")
	if parseErr := parseFlags.Parse(parseArgs[1:]); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if parseFlags.NArg() != 0 {
		return fmt.Errorf("unexpected desktop arguments: %v", parseFlags.Args())
	}
	if *parseTimeout <= 0 {
		return errors.New("desktop timeout must be positive")
	}
	parseFeaturesValue, parseFeaturesErr := normalizeDesktopFeatures(*parseFeatures)
	if parseFeaturesErr != nil {
		return parseFeaturesErr
	}
	parseContext, parseCancel := signalContext()
	defer parseCancel()
	parseConfig := desktopConfig{action: parseAction, root: strings.TrimSpace(*parseRoot), json: *parseJSON, features: parseFeaturesValue, context: parseContext, timeout: *parseTimeout}
	if parseConfig.root == "" {
		parseConfig.root, _ = os.Getwd()
	}
	parseRootAbs, parseErr := filepath.Abs(parseConfig.root)
	if parseErr != nil {
		return fmt.Errorf("resolve desktop root: %w", parseErr)
	}
	parseConfig.root = filepath.Clean(parseRootAbs)
	var parseSummary desktopSummary
	switch parseAction {
	case "init":
		parseSummary, parseErr = parseL.desktopInit(parseConfig)
	case "doctor":
		parseSummary, parseErr = parseL.desktopDoctor(parseConfig)
	case "build":
		parseSummary, parseErr = parseL.desktopBuild(parseConfig)
	case "dev":
		parseSummary, parseErr = parseL.desktopDev(parseConfig)
	case "package":
		parseSummary, parseErr = parseL.desktopPackage(parseConfig)
	default:
		return fmt.Errorf("unknown desktop action %q", parseAction)
	}
	if parseConfig.json {
		if parseSummary.Action == "" {
			parseSummary.Action = parseAction
		}
		if parseSummary.Root == "" {
			parseSummary.Root = parseConfig.root
		}
		parseSummary.OK = parseErr == nil
		if parseEncodeErr := json.NewEncoder(os.Stdout).Encode(parseSummary); parseEncodeErr != nil {
			return parseEncodeErr
		}
		return parseErr
	}
	if parseErr != nil {
		return parseErr
	}
	fmt.Printf("GWC desktop %s: %s\n", parseAction, parseConfig.root)
	return nil
}

// desktopInit creates a fresh contributor-linked desktop project.
func (parseL launcher) desktopInit(parseConfig desktopConfig) (desktopSummary, error) {
	if _, parseErr := os.Lstat(parseConfig.root); parseErr == nil {
		return desktopSummary{}, fmt.Errorf("desktop init refuses existing root %s", parseConfig.root)
	} else if !os.IsNotExist(parseErr) {
		return desktopSummary{}, parseErr
	}
	parseSource := filepath.Join(parseL.repoRoot, "examples", "desktop", "wails-counter")
	if parseErr := desktopRejectSymlinkPath(filepath.Dir(parseConfig.root), parseConfig.root); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseConfig.root), 0o755); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := os.Mkdir(parseConfig.root, 0o755); parseErr != nil {
		return desktopSummary{}, fmt.Errorf("create desktop root: %w", parseErr)
	}
	parseSummary := desktopSummary{OK: true, Action: "init", Root: parseConfig.root, Template: "contributor-linked examples/desktop/wails-counter"}
	for _, parseRel := range desktopTemplateFiles {
		parseSourcePath := filepath.Join(parseSource, filepath.FromSlash(parseRel))
		parseTargetPath := filepath.Join(parseConfig.root, filepath.FromSlash(parseRel))
		if parseErr := desktopCopyRegularFile(parseSourcePath, parseTargetPath); parseErr != nil {
			return desktopSummary{}, parseErr
		}
		parseSummary.Files = append(parseSummary.Files, filepath.ToSlash(parseRel))
	}
	parseGoModPath := filepath.Join(parseConfig.root, "go.mod")
	parseGoModBytes, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseGoMod := strings.ReplaceAll(string(parseGoModBytes), "replace github.com/monstercameron/GoWebComponents/v6 => ../../..", "replace github.com/monstercameron/GoWebComponents/v6 => "+strconv.Quote(filepath.ToSlash(parseL.repoRoot)))
	parseGoMod = strings.ReplaceAll(parseGoMod, "replace github.com/monstercameron/GoWebComponents/v6/desktop/wails => ../../../desktop/wails", "replace github.com/monstercameron/GoWebComponents/v6/desktop/wails => "+strconv.Quote(filepath.ToSlash(filepath.Join(parseL.repoRoot, "desktop", "wails"))))
	parseGoMod = strings.ReplaceAll(parseGoMod, "replace github.com/wailsapp/wails/v3 => ../../../third_party/wails/v3", "replace github.com/wailsapp/wails/v3 => "+strconv.Quote(filepath.ToSlash(filepath.Join(parseL.repoRoot, "third_party", "wails", "v3"))))
	if parseErr := os.WriteFile(parseGoModPath, []byte(parseGoMod), 0o644); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseReadmePath := filepath.Join(parseConfig.root, "README.md")
	parseReadmeBytes, parseReadmeErr := os.ReadFile(parseReadmePath)
	if parseReadmeErr != nil {
		return desktopSummary{}, parseReadmeErr
	}
	parseReadme := strings.ReplaceAll(string(parseReadmeBytes), "../../../third_party/wails/v3", filepath.ToSlash(filepath.Join(parseL.repoRoot, "third_party", "wails", "v3")))
	parseReadme = strings.ReplaceAll(parseReadme, "../../..", filepath.ToSlash(parseL.repoRoot))
	parseReadme = strings.ReplaceAll(parseReadme, "The checked-in .gitkeep lets native tests compile the embed package\nbefore building; launching the host still requires a complete build.", "This scaffold starts without generated assets. Run the build before native tests or launching the host.")
	if parseReadmeErr = os.WriteFile(parseReadmePath, []byte(parseReadme), 0o644); parseReadmeErr != nil {
		return desktopSummary{}, parseReadmeErr
	}
	parseMetadata := scaffoldMetadata{SchemaVersion: currentScaffoldMetadataSchemaVersion, ProjectName: "gwc-wails-counter", ModulePath: "example.com/gwc-wails-counter", Description: "Contributor-linked Wails counter template.", Ownership: scaffoldOwnershipMetadata{ProjectOwnership: "framework-coupled", FrameworkSourceMode: "local-replace"}, Desktop: &scaffoldDesktopMetadata{Version: 1, NativeEntry: "cmd/desktop/main.go", FrontendEntry: "frontend/main.go", AssetsDir: "assets", OutputPath: "bin/wails-counter.exe", WailsVersion: "v3.0.0-beta.17"}}
	if parseErr := normalizeScaffoldMetadata(&parseMetadata); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseMetadataBytes, parseErr := json.MarshalIndent(parseMetadata, "", "  ")
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := os.WriteFile(filepath.Join(parseConfig.root, "gwc-start.json"), append(parseMetadataBytes, '\n'), 0o644); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	return parseSummary, nil
}

// desktopCopyRegularFile copies one allowlisted regular template file.
func desktopCopyRegularFile(parseSource string, parseTarget string) error {
	parseInfo, parseErr := os.Lstat(parseSource)
	if parseErr != nil {
		return fmt.Errorf("desktop template source %s: %w", parseSource, parseErr)
	}
	if !parseInfo.Mode().IsRegular() {
		return fmt.Errorf("desktop template source is not a regular file: %s", parseSource)
	}
	parseBytes, parseErr := os.ReadFile(parseSource)
	if parseErr != nil {
		return parseErr
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseTarget), 0o755); parseErr != nil {
		return parseErr
	}
	return desktopWriteExclusive(parseTarget, parseBytes)
}

// desktopDoctor checks Windows, WebView2, pinned Wails, and module prerequisites.
func (parseL launcher) desktopDoctor(parseConfig desktopConfig) (desktopSummary, error) {
	parseSummary := desktopSummary{OK: true, Action: "doctor", Root: parseConfig.root}
	if runtime.GOOS != "windows" {
		return parseSummary, errors.New("desktop Wails target currently supports Windows only")
	}
	parseChecks := []string{}
	parseChecks = append(parseChecks, "windows")
	if parseOutput, parseErr := desktopExecute(parseConfig, "go", []string{"version"}, parseConfig.root, desktopNativeEnv()); parseErr != nil || !strings.Contains(parseOutput, "go1.") {
		return parseSummary, fmt.Errorf("go prerequisite failed: %v", parseErr)
	}
	parseChecks = append(parseChecks, "go")
	parseWebViewOutput, parseWebViewErr := desktopExecute(parseConfig, "reg", []string{"query", `HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`, "/v", "pv"}, "", desktopNativeEnv())
	if parseWebViewErr != nil || !strings.Contains(parseWebViewOutput, "pv") {
		return parseSummary, errors.New("Microsoft WebView2 runtime was not found")
	}
	parseChecks = append(parseChecks, "webview2")
	if _, parseErr := os.Stat(filepath.Join(parseL.repoRoot, "third_party", "wails", "v3", "go.mod")); parseErr != nil {
		return parseSummary, fmt.Errorf("pinned Wails source unavailable: %w", parseErr)
	}
	parseChecks = append(parseChecks, "wails-source")
	if _, parseErr := os.Stat(filepath.Join(parseConfig.root, "go.mod")); parseErr != nil {
		return parseSummary, fmt.Errorf("desktop go.mod unavailable: %w", parseErr)
	}
	parseChecks = append(parseChecks, "module")
	if parseErr := parseL.desktopVerifyPin(parseConfig); parseErr != nil {
		return parseSummary, parseErr
	}
	// Resolve the service dependency closure without requiring generated assets.
	// Listing every module also resolves unrelated local-only framework tooling.
	if parseOutput, parseErr := desktopExecute(parseConfig, "go", []string{"list", "-deps", "./internal/services"}, parseConfig.root, desktopNativeEnv()); parseErr != nil {
		return parseSummary, fmt.Errorf("desktop module resolution failed: %w (%s)", parseErr, parseOutput)
	}
	parseChecks = append(parseChecks, "wails-pin", "module-resolution")
	parseSummary.Checks = parseChecks
	return parseSummary, nil
}

// desktopBuild runs the isolated helper with explicit native target settings.
func (parseL launcher) desktopBuild(parseConfig desktopConfig) (desktopSummary, error) {
	parseFeatureInput := firstNonEmpty(parseConfig.features, "all")
	parseFeatures, parseFeaturesErr := normalizeDesktopFeatures(parseFeatureInput)
	if parseFeaturesErr != nil {
		return desktopSummary{}, parseFeaturesErr
	}
	parseConfig.features = parseFeatures
	parseMetadata, parseErr := desktopLoadMetadata(parseConfig.root)
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := desktopValidateCanonicalTarget(parseMetadata.Desktop); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseArtifact := filepath.Join(parseConfig.root, filepath.FromSlash(parseMetadata.Desktop.OutputPath))
	if parseErr := desktopRejectSymlinkPath(parseConfig.root, parseArtifact); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := desktopRejectOutputTree(parseConfig.root, filepath.Join(parseConfig.root, "assets", "dist")); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := parseL.desktopVerifyPin(parseConfig); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseOutput, parseErr := desktopExecute(parseConfig, "go", []string{"run", "./tools/build", "-features", parseConfig.features}, parseConfig.root, desktopNativeEnv())
	if parseErr != nil {
		return desktopSummary{}, fmt.Errorf("desktop build: %w\n%s", parseErr, parseOutput)
	}
	if parseInfo, parseStatErr := os.Lstat(parseArtifact); parseStatErr != nil || !parseInfo.Mode().IsRegular() {
		return desktopSummary{}, fmt.Errorf("desktop build produced no regular artifact %s: %v", parseArtifact, parseStatErr)
	}
	_ = parseOutput
	return desktopSummary{OK: true, Action: "build", Root: parseConfig.root, Artifact: artifactRelativePath(parseConfig.root, parseArtifact), Features: parseConfig.features}, nil
}

// desktopWebBuild prepares the isolated browser assets without touching desktop outputs.
func (parseL launcher) desktopWebBuild(parseConfig desktopConfig) (desktopSummary, error) {
	parseMetadata, parseFound, parseMetadataErr := loadScaffoldMetadata(parseConfig.root)
	if parseMetadataErr != nil {
		return desktopSummary{}, parseMetadataErr
	}
	if !parseFound || parseMetadata.Desktop == nil {
		return desktopSummary{}, errors.New("desktop metadata is required; run gwc desktop init")
	}
	if parseErr := desktopValidateCanonicalTarget(parseMetadata.Desktop); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseWebRoot := filepath.Join(parseConfig.root, "assets", "web")
	if parseErr := desktopRejectSymlinkPath(parseConfig.root, parseWebRoot); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	// Validate every helper destination before any write, not just the containing directory.
	for _, parseName := range []string{"index.html", "bootstrap.js", "app.css", "tester.css", "wasm_exec.js", "app.wasm"} {
		if parseErr := desktopRejectSymlinkPath(parseConfig.root, filepath.Join(parseWebRoot, parseName)); parseErr != nil {
			return desktopSummary{}, parseErr
		}
	}
	parseOutput, parseErr := desktopExecute(parseConfig, "go", []string{"run", "./tools/build", "-target", "web"}, parseConfig.root, desktopNativeEnv())
	if parseErr != nil {
		return desktopSummary{}, fmt.Errorf("desktop web build: %w\n%s", parseErr, parseOutput)
	}
	parseArtifact := filepath.Join(parseConfig.root, "assets", "web", "app.wasm")
	if parseInfo, parseStatErr := os.Lstat(parseArtifact); parseStatErr != nil || !parseInfo.Mode().IsRegular() {
		return desktopSummary{}, fmt.Errorf("desktop web build produced no regular artifact %s: %v", parseArtifact, parseStatErr)
	}
	return desktopSummary{OK: true, Action: "build", Root: parseConfig.root, Artifact: artifactRelativePath(parseConfig.root, parseArtifact)}, nil
}

// desktopPackage smokes and archives a canonical desktop build with hashes.
func (parseL launcher) desktopPackage(parseConfig desktopConfig) (desktopSummary, error) {
	parseZipPath := filepath.Join(parseConfig.root, "bin", "wails-counter.zip")
	parseManifestPath := filepath.Join(parseConfig.root, "bin", "wails-counter.manifest.json")
	for _, parseTarget := range []string{parseZipPath, parseManifestPath} {
		if parseErr := desktopRejectSymlinkPath(parseConfig.root, parseTarget); parseErr != nil {
			return desktopSummary{}, parseErr
		}
		if _, parseErr := os.Lstat(parseTarget); !os.IsNotExist(parseErr) {
			if parseErr != nil {
				return desktopSummary{}, parseErr
			}
			return desktopSummary{}, fmt.Errorf("desktop package refuses to overwrite %s", parseTarget)
		}
	}
	parseBuildSummary, parseErr := parseL.desktopBuild(parseConfig)
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseArtifact := filepath.Join(parseConfig.root, parseBuildSummary.Artifact)
	if parseErr := desktopSmokeProbe(parseConfig, parseArtifact, ""); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseGoVersion, parseErr := desktopExecute(parseConfig, "go", []string{"version"}, parseConfig.root, desktopNativeEnv())
	if parseErr != nil {
		return desktopSummary{}, fmt.Errorf("record Go tool version: %w", parseErr)
	}
	parseWailsRevision, parseErr := desktopExecute(parseConfig, "git", []string{"-C", filepath.Join(parseL.repoRoot, "third_party", "wails"), "rev-parse", "HEAD"}, "", desktopNativeEnv())
	if parseErr != nil {
		return desktopSummary{}, fmt.Errorf("record Wails revision: %w", parseErr)
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseZipPath), 0o755); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	parseZipFile, parseErr := os.OpenFile(parseZipPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	isComplete := false
	defer func() {
		if !isComplete {
			_ = os.Remove(parseZipPath)
		}
	}()
	parseZip := zip.NewWriter(parseZipFile)
	parseData, parseErr := os.ReadFile(parseArtifact)
	if parseErr == nil {
		parseWriter, parseWriteErr := parseZip.Create(filepath.Base(parseArtifact))
		if parseWriteErr != nil {
			parseErr = parseWriteErr
		} else {
			_, parseErr = parseWriter.Write(parseData)
		}
	}
	if parseCloseErr := parseZip.Close(); parseErr == nil {
		parseErr = parseCloseErr
	}
	if parseCloseErr := parseZipFile.Close(); parseErr == nil {
		parseErr = parseCloseErr
	}
	if parseErr != nil {
		return desktopSummary{}, fmt.Errorf("write desktop package: %w", parseErr)
	}
	parseHash := sha256.Sum256(parseData)
	parseZipData, parseReadZipErr := os.ReadFile(parseZipPath)
	if parseReadZipErr != nil {
		return desktopSummary{}, parseReadZipErr
	}
	parseZipHash := sha256.Sum256(parseZipData)
	parseSummary := desktopSummary{OK: true, Action: "package", Root: parseConfig.root, Artifact: artifactRelativePath(parseConfig.root, parseZipPath), Manifest: artifactRelativePath(parseConfig.root, parseManifestPath), WailsVersion: desktopWailsVersion, Features: parseConfig.features, Signed: false, Hashes: map[string]string{filepath.Base(parseArtifact): hex.EncodeToString(parseHash[:]), filepath.Base(parseZipPath): hex.EncodeToString(parseZipHash[:])}, ToolVersions: map[string]string{"go": strings.TrimSpace(parseGoVersion), "wailsRevision": strings.TrimSpace(parseWailsRevision)}}
	parseManifest, parseErr := json.MarshalIndent(parseSummary, "", "  ")
	if parseErr != nil {
		return desktopSummary{}, parseErr
	}
	if parseErr := desktopWriteExclusive(parseManifestPath, append(parseManifest, '\n')); parseErr != nil {
		return desktopSummary{}, parseErr
	}
	isComplete = true
	return parseSummary, nil
}

// desktopNativeEnv returns an explicit Windows native build environment.
func desktopNativeEnv() []string {
	parseEnv := make([]string, 0, len(os.Environ()))
	for _, parseEntry := range os.Environ() {
		parseKey, _, _ := strings.Cut(parseEntry, "=")
		switch strings.ToUpper(parseKey) {
		case "GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS":
			continue
		default:
			parseEnv = append(parseEnv, parseEntry)
		}
	}
	return append(parseEnv, "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=-mod=mod")
}

// desktopSmoke runs the host's actual WebView smoke report.
func (parseL launcher) desktopSmoke(parseRoot string, parseArtifact string) error {
	return desktopSmokeProbe(desktopConfig{root: parseRoot}, parseArtifact, "")
}

// desktopRejectSymlinkPath rejects output paths that could escape the project.
func desktopRejectSymlinkPath(parseRoot string, parseTarget string) error {
	parseRoot = filepath.Clean(parseRoot)
	parseTarget = filepath.Clean(parseTarget)
	parseRelative, parseErr := filepath.Rel(parseRoot, parseTarget)
	if parseErr != nil || parseRelative == ".." || strings.HasPrefix(parseRelative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("desktop path escapes root: %s", parseTarget)
	}
	// Inspect the root and every ancestor too, including junctions outside the relative suffix.
	for parseParent := parseRoot; ; parseParent = filepath.Dir(parseParent) {
		if parseInfo, parseErr := os.Lstat(parseParent); parseErr == nil {
			if parseInfo.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("desktop path refuses symlink root: %s", parseParent)
			}
		} else if !os.IsNotExist(parseErr) {
			return parseErr
		}
		if filepath.Dir(parseParent) == parseParent {
			break
		}
	}
	parseCurrent := parseRoot
	for _, parsePart := range strings.Split(parseRelative, string(filepath.Separator)) {
		if parsePart == "." || parsePart == "" {
			continue
		}
		parseCurrent = filepath.Join(parseCurrent, parsePart)
		parseInfo, parseStatErr := os.Lstat(parseCurrent)
		if parseStatErr != nil {
			if os.IsNotExist(parseStatErr) {
				continue
			}
			return parseStatErr
		}
		if parseInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("desktop path refuses symlink escape: %s", parseCurrent)
		}
	}
	return nil
}

// desktopLoadMetadata loads and validates the required desktop target metadata.
func desktopLoadMetadata(parseRoot string) (scaffoldMetadata, error) {
	parseMetadata, parseFound, parseErr := loadScaffoldMetadata(parseRoot)
	if parseErr != nil {
		return scaffoldMetadata{}, parseErr
	}
	if !parseFound || parseMetadata.Desktop == nil {
		return scaffoldMetadata{}, errors.New("desktop metadata is required; run gwc desktop init")
	}
	return parseMetadata, nil
}

// desktopValidateCanonicalTarget limits build/package to the proven template.
func desktopValidateCanonicalTarget(parseMetadata *scaffoldDesktopMetadata) error {
	parseWant := scaffoldDesktopMetadata{Version: 1, NativeEntry: "cmd/desktop/main.go", FrontendEntry: "frontend/main.go", AssetsDir: "assets", OutputPath: "bin/wails-counter.exe", WailsVersion: "v3.0.0-beta.17"}
	if parseMetadata == nil || *parseMetadata != parseWant {
		return errors.New("desktop metadata target is unsupported; only the contributor-linked wails-counter template is supported")
	}
	return nil
}

// artifactRelativePath renders an artifact path relative to its project root.
func artifactRelativePath(parseRoot string, parsePath string) string {
	parseRelative, parseErr := filepath.Rel(parseRoot, parsePath)
	if parseErr != nil {
		return filepath.ToSlash(parsePath)
	}
	return filepath.ToSlash(parseRelative)
}
