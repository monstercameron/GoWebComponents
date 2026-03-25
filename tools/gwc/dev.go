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

func (l launcher) runDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root to watch and serve")
	html := fs.String("html", "", "HTML file to serve, relative to the project root")
	index := fs.String("index", "", "Legacy alias for -html")
	wasm := fs.String("wasm", "", "WASM output path, relative to the build directory")
	output := fs.String("output", "", "Legacy alias for -wasm")
	host := fs.String("host", "", "Host to bind")
	port := fs.String("port", "", "Port to bind")
	hot := fs.Bool("hot", true, "Always use hot reload on successful rebuilds")
	tui := fs.Bool("tui", false, "Show an interactive status TUI while gwc dev runs")
	clientScript := fs.String("client-script", "", "Optional override path to a custom livereload client script")
	dryRun := fs.Bool("dry-run", false, "Resolve the dev plan and exit without starting the server")
	jsonOutput := fs.Bool("json", false, "Print the resolved dev plan as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := l.resolveDevConfig(devConfig{
		appPath:  firstNonEmpty(*app, *mainPath),
		rootPath: *root,
		htmlPath: firstNonEmpty(*html, *index),
		wasmPath: firstNonEmpty(*wasm, *output),
		host:     *host,
		port:     *port,
		hot:      *hot,
	})
	if err != nil {
		return err
	}

	forwarded := []string{"run", "./tools/livereload/livereload.go"}
	forwarded = append(forwarded, "-app", config.appPath)
	if config.rootPath != "" {
		forwarded = append(forwarded, "-root", config.rootPath)
	}
	if config.htmlPath != "" {
		forwarded = append(forwarded, "-html", config.htmlPath)
	}
	if config.wasmPath != "" {
		forwarded = append(forwarded, "-wasm", config.wasmPath)
	}
	forwarded = append(forwarded, "-host", config.host, "-port", config.port, "-hot", fmt.Sprintf("%t", config.hot))
	resolvedClientScript := strings.TrimSpace(*clientScript)
	if resolvedClientScript == "" {
		if autoClientScript, ok, resolveErr := resolveLauncherLivereloadClientScript(l.repoRoot, config.rootPath); resolveErr != nil {
			return resolveErr
		} else if ok {
			resolvedClientScript = autoClientScript
		}
	}
	if resolvedClientScript != "" {
		forwarded = append(forwarded, "-client-script", resolvedClientScript)
	}

	if *jsonOutput {
		if err := printDevPlanJSON(config); err != nil {
			return err
		}
	} else {
		printDevPlan(config)
	}
	if *dryRun {
		return nil
	}
	plan := describeDevPlan(config)
	cmd := exec.Command("go", forwarded...)
	cmd.Dir = l.repoRoot
	cmd.Env = os.Environ()
	if *tui {
		if plan.ServerMode != "livereload-wasm" || strings.TrimSpace(plan.StatusURL) == "" {
			return errors.New("dev -tui is only supported for livereload-backed js/wasm app runs")
		}
		return runDevStatusTUI(plan, plan.StatusURL, cmd)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (l launcher) resolveDevConfig(config devConfig) (devConfig, error) {
	resolved := config
	resolved.resolution = cloneResolutionTrace(config.resolution)
	resolved.host = strings.TrimSpace(resolved.host)
	resolved.port = strings.TrimSpace(resolved.port)

	cwd, err := devGetwd()
	if err != nil {
		return devConfig{}, err
	}

	metadata, metadataDir, hasMetadata, err := resolveScaffoldMetadataForConfig(cwd, resolved.rootPath, resolved.appPath)
	if err != nil {
		return devConfig{}, err
	}
	if hasMetadata {
		if strings.TrimSpace(resolved.appPath) == "" && strings.TrimSpace(metadata.Tooling.AppPath) != "" {
			resolved.appPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.AppPath))
			resolved.resolution = setResolutionSource(resolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
			resolved.resolution = setResolutionSource(resolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.htmlPath) == "" && strings.TrimSpace(metadata.Tooling.HTMLPath) != "" {
			resolved.htmlPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.HTMLPath))
			resolved.resolution = setResolutionSource(resolved.resolution, "html", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.wasmPath) == "" && strings.TrimSpace(metadata.Tooling.WASMPath) != "" {
			resolved.wasmPath = metadata.Tooling.WASMPath
			resolved.resolution = setResolutionSource(resolved.resolution, "wasm", "gwc-start.json")
		}
		if resolved.host == "" {
			resolved.host = strings.TrimSpace(metadata.Tooling.DevHost)
			resolved.resolution = setResolutionSource(resolved.resolution, "host", "gwc-start.json")
		}
		if resolved.port == "" {
			resolved.port = strings.TrimSpace(metadata.Tooling.DevPort)
			resolved.resolution = setResolutionSource(resolved.resolution, "port", "gwc-start.json")
		}
	}
	if strings.TrimSpace(config.appPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(config.rootPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "explicit flag")
	}
	if strings.TrimSpace(config.htmlPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "html", "explicit flag")
	}
	if strings.TrimSpace(config.wasmPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "wasm", "explicit flag")
	}
	if strings.TrimSpace(config.host) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "host", "explicit flag")
	}
	if strings.TrimSpace(config.port) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "port", "explicit flag")
	}
	if resolved.host == "" {
		resolved.host = "127.0.0.1"
		resolved.resolution = setResolutionSource(resolved.resolution, "host", "convention fallback")
	}
	if resolved.port == "" {
		resolved.port = "8080"
		resolved.resolution = setResolutionSource(resolved.resolution, "port", "convention fallback")
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return devConfig{}, err
		}
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "convention fallback")
	}
	resolved.appPath, err = normalizeExistingPath(cwd, resolved.appPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve app path: %w", err)
	}

	appDir := resolved.appPath
	info, err := os.Stat(resolved.appPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("inspect app path: %w", err)
	}
	if !info.IsDir() {
		appDir = filepath.Dir(resolved.appPath)
	}

	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = appDir
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "convention fallback")
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if strings.TrimSpace(resolved.htmlPath) == "" {
		resolved.htmlPath = detectHTMLPath(resolved.rootPath)
		if strings.TrimSpace(resolved.htmlPath) != "" {
			resolved.resolution = setResolutionSource(resolved.resolution, "html", "convention fallback")
		}
	}
	if strings.TrimSpace(resolved.htmlPath) != "" {
		resolved.htmlPath, err = normalizePath(cwd, resolved.htmlPath)
		if err != nil {
			return devConfig{}, fmt.Errorf("resolve html path: %w", err)
		}
	}

	if strings.TrimSpace(resolved.wasmPath) != "" {
		resolved.wasmPath = strings.TrimSpace(resolved.wasmPath)
	} else {
		resolved.resolution = setResolutionSource(resolved.resolution, "wasm", "convention fallback")
	}

	return resolved, nil
}

func detectAppPath(cwd string) (string, error) {
	directMain := filepath.Join(cwd, "main.go")
	if fileExists(directMain) {
		return directMain, nil
	}
	cmdWebMain := filepath.Join(cwd, "cmd", "web", "main.go")
	if fileExists(cmdWebMain) {
		return cmdWebMain, nil
	}
	return "", errors.New("dev could not detect an app entrypoint; pass -app or run from a directory with main.go or cmd/web/main.go")
}

func detectHTMLPath(root string) string {
	indexPath := filepath.Join(root, "index.html")
	if fileExists(indexPath) {
		return indexPath
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".html") {
			return filepath.Join(root, entry.Name())
		}
	}
	return ""
}

func resolveScaffoldMetadataForConfig(cwd string, configuredRoot string, configuredApp string) (scaffoldMetadata, string, bool, error) {
	candidates := []string{}
	for _, candidate := range []string{configuredRoot, configuredApp, cwd} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		resolved := candidate
		if !filepath.IsAbs(resolved) {
			absolute, err := normalizePath(cwd, resolved)
			if err != nil {
				return scaffoldMetadata{}, "", false, err
			}
			resolved = absolute
		}
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			resolved = filepath.Dir(resolved)
		}
		alreadyIncluded := false
		for _, existing := range candidates {
			if existing == resolved {
				alreadyIncluded = true
				break
			}
		}
		if !alreadyIncluded {
			candidates = append(candidates, resolved)
		}
	}
	for _, candidate := range candidates {
		metadata, ok, err := loadScaffoldMetadata(candidate)
		if err != nil {
			return scaffoldMetadata{}, "", false, err
		}
		if ok {
			return metadata, candidate, true, nil
		}
	}
	return scaffoldMetadata{}, "", false, nil
}

func loadScaffoldMetadata(dir string) (scaffoldMetadata, bool, error) {
	metadataPath := filepath.Join(strings.TrimSpace(dir), "gwc-start.json")
	if !fileExists(metadataPath) {
		return scaffoldMetadata{}, false, nil
	}
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		return scaffoldMetadata{}, false, fmt.Errorf("read scaffold metadata: %w", err)
	}
	var metadata scaffoldMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return scaffoldMetadata{}, false, fmt.Errorf("parse scaffold metadata: %w", err)
	}
	if err := normalizeScaffoldMetadata(&metadata); err != nil {
		return scaffoldMetadata{}, false, err
	}
	return metadata, true, nil
}

func normalizeScaffoldMetadata(metadata *scaffoldMetadata) error {
	if metadata == nil {
		return nil
	}
	switch metadata.SchemaVersion {
	case 0:
		metadata.SchemaVersion = currentScaffoldMetadataSchemaVersion
		if strings.TrimSpace(metadata.Ownership.ProjectOwnership) == "" {
			metadata.Ownership.ProjectOwnership = "standalone"
		}
		if strings.TrimSpace(metadata.Ownership.FrameworkSourceMode) == "" {
			metadata.Ownership.FrameworkSourceMode = "module-proxy"
		}
		return nil
	case currentScaffoldMetadataSchemaVersion:
		if strings.TrimSpace(metadata.Ownership.ProjectOwnership) == "" {
			metadata.Ownership.ProjectOwnership = "standalone"
		}
		if strings.TrimSpace(metadata.Ownership.FrameworkSourceMode) == "" {
			metadata.Ownership.FrameworkSourceMode = "module-proxy"
		}
		return nil
	default:
		return fmt.Errorf("unsupported scaffold metadata schema version %d", metadata.SchemaVersion)
	}
}

func normalizeExistingPath(base string, target string) (string, error) {
	resolved, err := normalizePath(base, target)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func normalizePath(base string, target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target), nil
	}
	return filepath.Abs(filepath.Join(base, target))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func printDevPlan(config devConfig) {
	plan := describeDevPlan(config)
	fmt.Println("GWC dev plan")
	fmt.Printf("  project root:  %s\n", plan.ProjectRoot)
	fmt.Printf("  app mode:      %s\n", plan.AppMode)
	fmt.Printf("  server mode:   %s\n", plan.ServerMode)
	fmt.Printf("  app:           %s\n", config.appPath)
	fmt.Printf("  root:          %s\n", config.rootPath)
	if config.htmlPath != "" {
		fmt.Printf("  html:          %s\n", config.htmlPath)
	} else {
		fmt.Println("  html:          <auto-detect skipped>")
	}
	if config.wasmPath != "" {
		fmt.Printf("  wasm:          %s\n", config.wasmPath)
	} else {
		fmt.Println("  wasm:          <livereload default>")
	}
	fmt.Printf("  host:          %s\n", config.host)
	fmt.Printf("  port:          %s\n", config.port)
	fmt.Printf("  hot:           %t\n", config.hot)
	fmt.Printf("  listening URL: %s\n", plan.ListeningURL)
	if plan.ServerMode == "livereload-wasm" {
		fmt.Printf("  status URL:    %s\n", plan.StatusURL)
	}
	printResolutionTrace(config.resolution, []string{"app", "root", "html", "wasm", "host", "port"}, "  ")
}

func printDevPlanJSON(config devConfig) error {
	plan := describeDevPlan(config)
	payload := map[string]interface{}{
		"app":          config.appPath,
		"root":         config.rootPath,
		"projectRoot":  plan.ProjectRoot,
		"appMode":      plan.AppMode,
		"serverMode":   plan.ServerMode,
		"html":         config.htmlPath,
		"wasm":         config.wasmPath,
		"host":         config.host,
		"port":         config.port,
		"hot":          config.hot,
		"listeningURL": plan.ListeningURL,
		"statusURL":    plan.StatusURL,
		"resolution":   config.resolution,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

type devPlanSummary struct {
	ProjectRoot  string
	AppMode      string
	ServerMode   string
	ListeningURL string
	StatusURL    string
}

func describeDevPlan(config devConfig) devPlanSummary {
	projectRoot := strings.TrimSpace(config.rootPath)
	if projectRoot == "" {
		projectRoot = filepath.Dir(strings.TrimSpace(config.appPath))
	}
	appMode := "client-only-wasm"
	serverMode := "livereload-wasm"
	if strings.Contains(strings.ToLower(filepath.ToSlash(strings.TrimSpace(config.appPath))), "/cmd/web/main.go") {
		appMode = "server-app"
		serverMode = "server-entrypoint"
	}
	statusURL := ""
	if serverMode == "livereload-wasm" {
		statusURL = "http://" + joinHostPort(config.host, config.port) + "/__gwc/status"
	}
	return devPlanSummary{
		ProjectRoot:  projectRoot,
		AppMode:      appMode,
		ServerMode:   serverMode,
		ListeningURL: "http://" + joinHostPort(config.host, config.port),
		StatusURL:    statusURL,
	}
}
