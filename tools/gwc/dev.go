package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type devConfig struct {
	appPath    string
	rootPath   string
	htmlPath   string
	wasmPath   string
	host       string
	port       string
	hot        bool
	resolution map[string]string
}

var devGetwd = os.Getwd

func (parseL launcher) runDev(parseArgs []string) error {
	parseFs := flag.NewFlagSet("dev", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "Legacy alias for -app")
	parseRoot := parseFs.String("root", "", "Project root to watch and serve")
	parseHtml := parseFs.String("html", "", "HTML file to serve, relative to the project root")
	parseIndex := parseFs.String("index", "", "Legacy alias for -html")
	parseWasm := parseFs.String("wasm", "", "WASM output path, relative to the build directory")
	parseOutput := parseFs.String("output", "", "Legacy alias for -wasm")
	parseHost := parseFs.String("host", "", "Host to bind")
	parsePort := parseFs.String("port", "", "Port to bind")
	parseHot := parseFs.Bool("hot", true, "Always use hot reload on successful rebuilds")
	parseTui := parseFs.Bool("tui", false, "Show an interactive status TUI while gwc dev runs")
	parseClientScript := parseFs.String("client-script", "", "Optional override path to a custom livereload client script")
	parseDryRun := parseFs.Bool("dry-run", false, "Resolve the dev plan and exit without starting the server")
	parseJsonOutput := parseFs.Bool("json", false, "Print the resolved dev plan as JSON")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := parseL.resolveDevConfig(devConfig{
		appPath:  firstNonEmpty(*parseApp, *parseMainPath),
		rootPath: *parseRoot,
		htmlPath: firstNonEmpty(*parseHtml, *parseIndex),
		wasmPath: firstNonEmpty(*parseWasm, *parseOutput),
		host:     *parseHost,
		port:     *parsePort,
		hot:      *parseHot,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseForwarded, parseErr2 := buildLivereloadRunArgs()
	if parseErr2 != nil {
		return parseErr2
	}
	parseForwarded = append(parseForwarded, "-app", parseConfig.appPath)
	if parseConfig.rootPath != "" {
		parseForwarded = append(parseForwarded, "-root", parseConfig.rootPath)
	}
	if parseConfig.htmlPath != "" {
		parseForwarded = append(parseForwarded, "-html", parseConfig.htmlPath)
	}
	if parseConfig.wasmPath != "" {
		parseForwarded = append(parseForwarded, "-wasm", parseConfig.wasmPath)
	}
	parseForwarded = append(parseForwarded, "-host", parseConfig.host, "-port", parseConfig.port, "-hot", fmt.Sprintf("%t", parseConfig.hot))
	parseResolvedClientScript := strings.TrimSpace(*parseClientScript)
	if parseResolvedClientScript == "" {
		if parseAutoClientScript, parseOk, parseResolveErr := resolveLauncherLivereloadClientScript(parseConfig.rootPath); parseResolveErr != nil {
			return parseResolveErr
		} else if parseOk {
			parseResolvedClientScript = parseAutoClientScript
		}
	}
	if parseResolvedClientScript != "" {
		parseForwarded = append(parseForwarded, "-client-script", parseResolvedClientScript)
	}

	if *parseJsonOutput {
		if parseErr3 := printDevPlanJSON(parseConfig); parseErr3 != nil {
			return parseErr3
		}
	} else {
		printDevPlan(parseConfig)
	}
	if *parseDryRun {
		return nil
	}
	parsePlan := describeDevPlan(parseConfig)
	parseCmd := exec.Command("go", parseForwarded...)
	parseCmd.Dir = parseL.repoRoot
	parseCmd.Env = os.Environ()
	if *parseTui {
		if parsePlan.ServerMode != "livereload-wasm" || strings.TrimSpace(parsePlan.StatusURL) == "" {
			return errors.New("dev -tui is only supported for livereload-backed js/wasm app runs")
		}
		return runDevStatusTUI(parsePlan, parsePlan.StatusURL, parseCmd)
	}
	parseCmd.Stdin = os.Stdin
	parseCmd.Stdout = os.Stdout
	parseCmd.Stderr = os.Stderr
	return parseCmd.Run()
}

// buildLivereloadRunArgs returns the repo-root go run argument list for the
// nested livereload module, including every non-test source file it needs.
func buildLivereloadRunArgs() ([]string, error) {
	return []string{
		"run",
		"tools/livereload/client_script.go",
		"tools/livereload/livereload.go",
		"tools/livereload/livereload_http.go",
		"tools/livereload/livereload_paths.go",
		"tools/livereload/livereload_support.go",
	}, nil
}

func (parseL launcher) resolveDevConfig(parseConfig devConfig) (devConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)
	parseResolved.host = strings.TrimSpace(parseResolved.host)
	parseResolved.port = strings.TrimSpace(parseResolved.port)

	parseCwd, parseErr := devGetwd()
	if parseErr != nil {
		return devConfig{}, parseErr
	}

	parseMetadata, parseMetadataDir, hasMetadata, parseErr := resolveScaffoldMetadataForConfig(parseCwd, parseResolved.rootPath, parseResolved.appPath)
	if parseErr != nil {
		return devConfig{}, parseErr
	}
	if hasMetadata {
		if strings.TrimSpace(parseResolved.appPath) == "" && strings.TrimSpace(parseMetadata.Tooling.AppPath) != "" {
			parseResolved.appPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.AppPath))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.rootPath) == "" {
			parseResolved.rootPath = parseMetadataDir
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.htmlPath) == "" && strings.TrimSpace(parseMetadata.Tooling.HTMLPath) != "" {
			parseResolved.htmlPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.HTMLPath))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "html", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.wasmPath) == "" && strings.TrimSpace(parseMetadata.Tooling.WASMPath) != "" {
			parseResolved.wasmPath = parseMetadata.Tooling.WASMPath
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "wasm", "gwc-start.json")
		}
		if parseResolved.host == "" {
			parseResolved.host = strings.TrimSpace(parseMetadata.Tooling.DevHost)
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "host", "gwc-start.json")
		}
		if parseResolved.port == "" {
			parseResolved.port = strings.TrimSpace(parseMetadata.Tooling.DevPort)
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "port", "gwc-start.json")
		}
	}
	if strings.TrimSpace(parseConfig.appPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.rootPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.htmlPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "html", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.wasmPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "wasm", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.host) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "host", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.port) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "port", "explicit flag")
	}
	if parseResolved.host == "" {
		parseResolved.host = "127.0.0.1"
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "host", "convention fallback")
	}
	if parseResolved.port == "" {
		parseResolved.port = "8080"
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "port", "convention fallback")
	}

	if strings.TrimSpace(parseResolved.appPath) == "" {
		parseResolved.appPath, parseErr = detectAppPath(parseCwd)
		if parseErr != nil {
			return devConfig{}, parseErr
		}
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "convention fallback")
	}
	parseResolved.appPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.appPath)
	if parseErr != nil {
		return devConfig{}, fmt.Errorf("resolve app path: %w", parseErr)
	}

	parseAppDir := parseResolved.appPath
	parseInfo, parseErr := os.Stat(parseResolved.appPath)
	if parseErr != nil {
		return devConfig{}, fmt.Errorf("inspect app path: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		parseAppDir = filepath.Dir(parseResolved.appPath)
	}

	if strings.TrimSpace(parseResolved.rootPath) == "" {
		parseResolved.rootPath = parseAppDir
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "convention fallback")
	}
	parseResolved.rootPath, parseErr = normalizePath(parseCwd, parseResolved.rootPath)
	if parseErr != nil {
		return devConfig{}, fmt.Errorf("resolve root path: %w", parseErr)
	}

	if strings.TrimSpace(parseResolved.htmlPath) == "" {
		parseResolved.htmlPath = detectHTMLPath(parseResolved.rootPath)
		if strings.TrimSpace(parseResolved.htmlPath) != "" {
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "html", "convention fallback")
		}
	}
	if strings.TrimSpace(parseResolved.htmlPath) != "" {
		parseResolved.htmlPath, parseErr = normalizePath(parseCwd, parseResolved.htmlPath)
		if parseErr != nil {
			return devConfig{}, fmt.Errorf("resolve html path: %w", parseErr)
		}
	}

	if strings.TrimSpace(parseResolved.wasmPath) != "" {
		parseResolved.wasmPath = strings.TrimSpace(parseResolved.wasmPath)
	} else {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "wasm", "convention fallback")
	}

	return parseResolved, nil
}

func detectAppPath(parseCwd string) (string, error) {
	parseDirectMain := filepath.Join(parseCwd, "main.go")
	if fileExists(parseDirectMain) {
		return parseDirectMain, nil
	}
	parseCmdWebMain := filepath.Join(parseCwd, "cmd", "web", "main.go")
	if fileExists(parseCmdWebMain) {
		return parseCmdWebMain, nil
	}
	return "", errors.New("dev could not detect an app entrypoint; pass -app or run from a directory with main.go or cmd/web/main.go")
}

func detectHTMLPath(parseRoot string) string {
	parseIndexPath := filepath.Join(parseRoot, "index.html")
	if fileExists(parseIndexPath) {
		return parseIndexPath
	}

	parseEntries, parseErr := os.ReadDir(parseRoot)
	if parseErr != nil {
		return ""
	}
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() {
			continue
		}
		parseName := strings.ToLower(parseEntry.Name())
		if strings.HasSuffix(parseName, ".html") {
			return filepath.Join(parseRoot, parseEntry.Name())
		}
	}
	return ""
}

func resolveScaffoldMetadataForConfig(parseCwd string, parseConfiguredRoot string, parseConfiguredApp string) (scaffoldMetadata, string, bool, error) {
	parseCandidates := []string{}
	for _, parseCandidate := range []string{parseConfiguredRoot, parseConfiguredApp, parseCwd} {
		parseCandidate = strings.TrimSpace(parseCandidate)
		if parseCandidate == "" {
			continue
		}
		parseResolved := parseCandidate
		if !filepath.IsAbs(parseResolved) {
			parseAbsolute, parseErr := normalizePath(parseCwd, parseResolved)
			if parseErr != nil {
				return scaffoldMetadata{}, "", false, parseErr
			}
			parseResolved = parseAbsolute
		}
		if parseInfo, parseErr2 := os.Stat(parseResolved); parseErr2 == nil && !parseInfo.IsDir() {
			parseResolved = filepath.Dir(parseResolved)
		}
		isParseAlreadyIncluded := false
		for _, parseExisting := range parseCandidates {
			if parseExisting == parseResolved {
				isParseAlreadyIncluded = true
				break
			}
		}
		if !isParseAlreadyIncluded {
			parseCandidates = append(parseCandidates, parseResolved)
		}
	}
	for _, parseCandidate2 := range parseCandidates {
		parseMetadata, parseOk, parseErr3 := loadScaffoldMetadata(parseCandidate2)
		if parseErr3 != nil {
			return scaffoldMetadata{}, "", false, parseErr3
		}
		if parseOk {
			return parseMetadata, parseCandidate2, true, nil
		}
	}
	return scaffoldMetadata{}, "", false, nil
}

func loadScaffoldMetadata(parseDir string) (scaffoldMetadata, bool, error) {
	parseMetadataPath := filepath.Join(strings.TrimSpace(parseDir), "gwc-start.json")
	if !fileExists(parseMetadataPath) {
		return scaffoldMetadata{}, false, nil
	}
	parseContent, parseErr := os.ReadFile(parseMetadataPath)
	if parseErr != nil {
		return scaffoldMetadata{}, false, fmt.Errorf("read scaffold metadata: %w", parseErr)
	}
	var parseMetadata scaffoldMetadata
	if parseErr2 := json.Unmarshal(parseContent, &parseMetadata); parseErr2 != nil {
		return scaffoldMetadata{}, false, fmt.Errorf("parse scaffold metadata: %w", parseErr2)
	}
	if parseErr3 := normalizeScaffoldMetadata(&parseMetadata); parseErr3 != nil {
		return scaffoldMetadata{}, false, parseErr3
	}
	return parseMetadata, true, nil
}

func normalizeScaffoldMetadata(parseMetadata *scaffoldMetadata) error {
	if parseMetadata == nil {
		return nil
	}
	switch parseMetadata.SchemaVersion {
	case 0:
		parseMetadata.SchemaVersion = currentScaffoldMetadataSchemaVersion
		if strings.TrimSpace(parseMetadata.Ownership.ProjectOwnership) == "" {
			parseMetadata.Ownership.ProjectOwnership = "standalone"
		}
		if strings.TrimSpace(parseMetadata.Ownership.FrameworkSourceMode) == "" {
			parseMetadata.Ownership.FrameworkSourceMode = "module-proxy"
		}
		return nil
	case currentScaffoldMetadataSchemaVersion:
		if strings.TrimSpace(parseMetadata.Ownership.ProjectOwnership) == "" {
			parseMetadata.Ownership.ProjectOwnership = "standalone"
		}
		if strings.TrimSpace(parseMetadata.Ownership.FrameworkSourceMode) == "" {
			parseMetadata.Ownership.FrameworkSourceMode = "module-proxy"
		}
		return nil
	default:
		return fmt.Errorf("unsupported scaffold metadata schema version %d", parseMetadata.SchemaVersion)
	}
}

func normalizeExistingPath(parseBase string, parseTarget string) (string, error) {
	parseResolved, parseErr := normalizePath(parseBase, parseTarget)
	if parseErr != nil {
		return "", parseErr
	}
	if _, parseErr2 := os.Stat(parseResolved); parseErr2 != nil {
		return "", parseErr2
	}
	return parseResolved, nil
}

func normalizePath(parseBase string, parseTarget string) (string, error) {
	parseTarget = strings.TrimSpace(parseTarget)
	if parseTarget == "" {
		return "", nil
	}
	if filepath.IsAbs(parseTarget) {
		return filepath.Clean(parseTarget), nil
	}
	return filepath.Abs(filepath.Join(parseBase, parseTarget))
}

func fileExists(parsePath string) bool {
	parseInfo, parseErr := os.Stat(parsePath)
	if parseErr != nil {
		return false
	}
	return !parseInfo.IsDir()
}

func firstNonEmpty(parseValues ...string) string {
	for _, parseValue := range parseValues {
		if strings.TrimSpace(parseValue) != "" {
			return strings.TrimSpace(parseValue)
		}
	}
	return ""
}

func printDevPlan(parseConfig devConfig) {
	parsePlan := describeDevPlan(parseConfig)
	fmt.Println("GWC dev plan")
	fmt.Printf("  project root:  %s\n", parsePlan.ProjectRoot)
	fmt.Printf("  app mode:      %s\n", parsePlan.AppMode)
	fmt.Printf("  server mode:   %s\n", parsePlan.ServerMode)
	fmt.Printf("  app:           %s\n", parseConfig.appPath)
	fmt.Printf("  root:          %s\n", parseConfig.rootPath)
	if parseConfig.htmlPath != "" {
		fmt.Printf("  html:          %s\n", parseConfig.htmlPath)
	} else {
		fmt.Println("  html:          <auto-detect skipped>")
	}
	if parseConfig.wasmPath != "" {
		fmt.Printf("  wasm:          %s\n", parseConfig.wasmPath)
	} else {
		fmt.Println("  wasm:          <livereload default>")
	}
	fmt.Printf("  host:          %s\n", parseConfig.host)
	fmt.Printf("  port:          %s\n", parseConfig.port)
	fmt.Printf("  hot:           %t\n", parseConfig.hot)
	fmt.Printf("  listening URL: %s\n", parsePlan.ListeningURL)
	if parsePlan.ServerMode == "livereload-wasm" {
		fmt.Printf("  status URL:    %s\n", parsePlan.StatusURL)
	}
	printResolutionTrace(parseConfig.resolution, []string{"app", "root", "html", "wasm", "host", "port"}, "  ")
}

func printDevPlanJSON(parseConfig devConfig) error {
	parsePlan := describeDevPlan(parseConfig)
	parsePayload := map[string]interface{}{
		"app":          parseConfig.appPath,
		"root":         parseConfig.rootPath,
		"projectRoot":  parsePlan.ProjectRoot,
		"appMode":      parsePlan.AppMode,
		"serverMode":   parsePlan.ServerMode,
		"html":         parseConfig.htmlPath,
		"wasm":         parseConfig.wasmPath,
		"host":         parseConfig.host,
		"port":         parseConfig.port,
		"hot":          parseConfig.hot,
		"listeningURL": parsePlan.ListeningURL,
		"statusURL":    parsePlan.StatusURL,
		"resolution":   parseConfig.resolution,
	}
	parseEncoder := json.NewEncoder(os.Stdout)
	parseEncoder.SetIndent("", "  ")
	return parseEncoder.Encode(parsePayload)
}

type devPlanSummary struct {
	ProjectRoot  string
	AppMode      string
	ServerMode   string
	ListeningURL string
	StatusURL    string
}

func describeDevPlan(parseConfig devConfig) devPlanSummary {
	parseProjectRoot := strings.TrimSpace(parseConfig.rootPath)
	if parseProjectRoot == "" {
		parseProjectRoot = filepath.Dir(strings.TrimSpace(parseConfig.appPath))
	}
	parseAppMode := "client-only-wasm"
	parseServerMode := "livereload-wasm"
	parseNormalizedAppPath := strings.ToLower(filepath.ToSlash(strings.TrimSpace(parseConfig.appPath)))
	if strings.HasSuffix(parseNormalizedAppPath, "/cmd/web/main.go") ||
		parseNormalizedAppPath == "cmd/web/main.go" {
		parseAppMode = "server-app"
		parseServerMode = "server-entrypoint"
	}
	parseStatusURL := ""
	if parseServerMode == "livereload-wasm" {
		parseStatusURL = "http://" + joinHostPort(parseConfig.host, parseConfig.port) + "/__gwc/status"
	}
	return devPlanSummary{
		ProjectRoot:  parseProjectRoot,
		AppMode:      parseAppMode,
		ServerMode:   parseServerMode,
		ListeningURL: "http://" + joinHostPort(parseConfig.host, parseConfig.port),
		StatusURL:    parseStatusURL,
	}
}
