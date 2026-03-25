package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/prerender"
)

var runPrerenderCommand = func(l launcher, args []string) error {
	return l.runPrerender(args)
}

var runExportCommand = func(l launcher, args []string) error {
	return l.runPrerender(args)
}

type prerenderConfig struct {
	appPath   string
	rootPath  string
	htmlPath  string
	outDir    string
	routes    []string
	assetDirs []string
	profile   string
	skipBuild bool
	json      bool
}

type prerenderSummary struct {
	OK             bool     `json:"ok"`
	Root           string   `json:"root"`
	AppPath        string   `json:"appPath"`
	HTMLPath       string   `json:"htmlPath"`
	OutDir         string   `json:"outDir"`
	ManifestPath   string   `json:"manifestPath"`
	Routes         []string `json:"routes"`
	HTMLFiles      []string `json:"htmlFiles,omitempty"`
	BootstrapFiles []string `json:"bootstrapFiles,omitempty"`
	AssetFiles     []string `json:"assetFiles,omitempty"`
	RuntimeAsset   string   `json:"runtimeAsset,omitempty"`
	WASMArtifact   string   `json:"wasmArtifact"`
	BuildExecuted  bool     `json:"buildExecuted"`
	BuildProfile   string   `json:"buildProfile,omitempty"`
}

// runPrerender executes static export through a first-class launcher command.
func (parseL launcher) runPrerender(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("prerender", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseApp := parseFlags.String("app", "", "Path to the app main.go file or app directory")
	parseMain := parseFlags.String("main", "", "Legacy alias for -app")
	parseRoot := parseFlags.String("root", "", "Project root used for static export")
	parseHTML := parseFlags.String("html", "", "HTML shell path relative to -root")
	parseIndex := parseFlags.String("index", "", "Legacy alias for -html")
	parseOut := parseFlags.String("out", "", "Output directory for static export artifacts")
	parseOutput := parseFlags.String("output", "", "Legacy alias for -out")
	parseProfile := parseFlags.String("profile", "release", "Build profile used when exporting wasm artifacts")
	parseSkipBuild := parseFlags.Bool("skip-build", false, "Skip the build step and reuse an existing <out-dir>/main.wasm artifact")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON summary")
	var parseRoutes stringListFlag
	var parseAssetDirs stringListFlag
	parseFlags.Var(&parseRoutes, "route", "Route path to prerender; repeatable, defaults to /")
	parseFlags.Var(&parseAssetDirs, "asset-dir", "Directory under -root to copy into the export output; repeatable")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := parseL.prerenderConfig(prerenderConfig{
		appPath:   firstNonEmpty(*parseApp, *parseMain),
		rootPath:  *parseRoot,
		htmlPath:  firstNonEmpty(*parseHTML, *parseIndex),
		outDir:    firstNonEmpty(*parseOut, *parseOutput),
		routes:    parseRoutes.Values(),
		assetDirs: parseAssetDirs.Values(),
		profile:   *parseProfile,
		skipBuild: *parseSkipBuild,
		json:      *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	applySummary, applyErr := parseL.applyPrerenderExport(parseConfig)
	if applyErr != nil {
		return applyErr
	}
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(applySummary)
	}
	renderPrerenderSummary(applySummary)
	return nil
}

// parsePrerenderConfig resolves and validates static export configuration.
func (parseL launcher) prerenderConfig(parseConfig prerenderConfig) (prerenderConfig, error) {
	parseRootPath, parseErr := parseLifecycleRootPath(parseConfig.rootPath)
	if parseErr != nil {
		return prerenderConfig{}, parseErr
	}
	parseAppPath := strings.TrimSpace(parseConfig.appPath)
	if parseAppPath == "" {
		parseDetectedApp, parseDetectErr := detectAppPath(parseRootPath)
		if parseDetectErr != nil {
			return prerenderConfig{}, fmt.Errorf("detect app path for prerender: %w", parseDetectErr)
		}
		parseAppPath = parseDetectedApp
	}
	parseAppPath, parseErr = normalizeExistingPath(parseRootPath, parseAppPath)
	if parseErr != nil {
		return prerenderConfig{}, fmt.Errorf("resolve prerender app path: %w", parseErr)
	}

	parseHTMLPath := strings.TrimSpace(parseConfig.htmlPath)
	if parseHTMLPath == "" {
		parseHTMLPath = detectHTMLPath(parseRootPath)
	}
	if strings.TrimSpace(parseHTMLPath) == "" {
		return prerenderConfig{}, errors.New("prerender could not detect an html shell; pass -html")
	}
	parseHTMLPath, parseErr = normalizeExistingPath(parseRootPath, parseHTMLPath)
	if parseErr != nil {
		return prerenderConfig{}, fmt.Errorf("resolve prerender html path: %w", parseErr)
	}

	parseOutDir := strings.TrimSpace(parseConfig.outDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Join(parseRootPath, "bin", "static-export")
	}
	parseOutDir, parseErr = normalizePath(parseRootPath, parseOutDir)
	if parseErr != nil {
		return prerenderConfig{}, fmt.Errorf("resolve prerender out path: %w", parseErr)
	}

	parseRoutes, parseErr := parsePrerenderRoutes(parseConfig.routes)
	if parseErr != nil {
		return prerenderConfig{}, parseErr
	}
	parseAssetDirs, parseErr := parsePrerenderAssetDirs(parseRootPath, parseConfig.assetDirs)
	if parseErr != nil {
		return prerenderConfig{}, parseErr
	}
	parseProfile := strings.TrimSpace(parseConfig.profile)
	if parseProfile == "" {
		parseProfile = "release"
	}
	return prerenderConfig{
		appPath:   parseAppPath,
		rootPath:  parseRootPath,
		htmlPath:  parseHTMLPath,
		outDir:    parseOutDir,
		routes:    parseRoutes,
		assetDirs: parseAssetDirs,
		profile:   parseProfile,
		skipBuild: parseConfig.skipBuild,
		json:      parseConfig.json,
	}, nil
}

// parsePrerenderRoutes normalizes route values for static export.
func parsePrerenderRoutes(parseRoutes []string) ([]string, error) {
	if len(parseRoutes) == 0 {
		return []string{"/"}, nil
	}
	parseRouteSet := map[string]struct{}{}
	for _, parseRoute := range parseRoutes {
		parseRoute = strings.TrimSpace(parseRoute)
		if parseRoute == "" {
			continue
		}
		if !strings.HasPrefix(parseRoute, "/") {
			return nil, fmt.Errorf("prerender routes must start with '/': %q", parseRoute)
		}
		if parseRoute != "/" {
			parseRoute = strings.TrimRight(parseRoute, "/")
		}
		parseRouteSet[parseRoute] = struct{}{}
	}
	if len(parseRouteSet) == 0 {
		return []string{"/"}, nil
	}
	parseNormalized := make([]string, 0, len(parseRouteSet))
	for parseRoute := range parseRouteSet {
		parseNormalized = append(parseNormalized, parseRoute)
	}
	sort.Strings(parseNormalized)
	return parseNormalized, nil
}

// parsePrerenderAssetDirs resolves asset copy directories under the project root.
func parsePrerenderAssetDirs(parseRootPath string, parseAssetDirs []string) ([]string, error) {
	parseNormalized := []string{}
	parseSeen := map[string]struct{}{}
	for _, parseAssetDir := range parseAssetDirs {
		parseAssetDir = strings.TrimSpace(parseAssetDir)
		if parseAssetDir == "" {
			continue
		}
		parseAssetPath, parseErr := normalizeExistingPath(parseRootPath, parseAssetDir)
		if parseErr != nil {
			return nil, fmt.Errorf("resolve asset directory %q: %w", parseAssetDir, parseErr)
		}
		parseInfo, parseErr := os.Stat(parseAssetPath)
		if parseErr != nil {
			return nil, parseErr
		}
		if !parseInfo.IsDir() {
			return nil, fmt.Errorf("asset-dir is not a directory: %s", parseAssetPath)
		}
		if _, parseExists := parseSeen[parseAssetPath]; parseExists {
			continue
		}
		parseSeen[parseAssetPath] = struct{}{}
		parseNormalized = append(parseNormalized, parseAssetPath)
	}
	sort.Strings(parseNormalized)
	return parseNormalized, nil
}

// applyPrerenderExport builds static files and route HTML output.
func (parseL launcher) applyPrerenderExport(applyConfig prerenderConfig) (prerenderSummary, error) {
	if applyErr := os.MkdirAll(applyConfig.outDir, 0755); applyErr != nil {
		return prerenderSummary{}, fmt.Errorf("create prerender output directory: %w", applyErr)
	}
	applyWASMPath := filepath.Join(applyConfig.outDir, "main.wasm")
	isApplyBuildExecuted := false
	if applyConfig.skipBuild {
		if !fileExists(applyWASMPath) {
			return prerenderSummary{}, fmt.Errorf("skip-build requested but %s does not exist", applyWASMPath)
		}
	} else {
		applyBuildConfig, applyResolveErr := resolveBuildConfig(buildConfig{
			appPath:    applyConfig.appPath,
			rootPath:   applyConfig.rootPath,
			outputPath: applyWASMPath,
			profile:    applyConfig.profile,
		})
		if applyResolveErr != nil {
			return prerenderSummary{}, applyResolveErr
		}
		if _, applyBuildErr := executeBuild(applyBuildConfig); applyBuildErr != nil {
			return prerenderSummary{}, applyBuildErr
		}
		isApplyBuildExecuted = true
	}

	applyRuntimeAssetPath, applyRuntimeErr := applyLifecycleRuntimeAsset(applyConfig.outDir)
	if applyRuntimeErr != nil {
		return prerenderSummary{}, applyRuntimeErr
	}

	applyHTMLBytes, applyReadErr := os.ReadFile(applyConfig.htmlPath)
	if applyReadErr != nil {
		return prerenderSummary{}, fmt.Errorf("read prerender html shell: %w", applyReadErr)
	}
	applyRoutes := make([]prerender.Route, 0, len(applyConfig.routes))
	applyTemplateHTML := string(applyHTMLBytes)
	for _, applyRoute := range applyConfig.routes {
		applyPath := applyRoute
		applyRoutes = append(applyRoutes, prerender.Route{
			Path: applyPath,
			Build: func(buildTarget prerender.Target) (prerender.RouteOutput, error) {
				return prerender.RouteOutput{HTML: applyTemplateHTML}, nil
			},
		})
	}
	applyExportSummary, applyExportErr := prerender.Export(applyConfig.outDir, applyRoutes)
	if applyExportErr != nil {
		return prerenderSummary{}, applyExportErr
	}

	applyAssetFiles, applyAssetErr := applyPrerenderAssets(applyConfig.rootPath, applyConfig.outDir, applyConfig.assetDirs)
	if applyAssetErr != nil {
		return prerenderSummary{}, applyAssetErr
	}

	applySummary := prerenderSummary{
		OK:             true,
		Root:           applyConfig.rootPath,
		AppPath:        applyConfig.appPath,
		HTMLPath:       applyConfig.htmlPath,
		OutDir:         applyConfig.outDir,
		Routes:         append([]string(nil), applyConfig.routes...),
		HTMLFiles:      applyExportSummary.HTMLFiles,
		BootstrapFiles: applyExportSummary.BootstrapFiles,
		AssetFiles:     applyAssetFiles,
		RuntimeAsset:   applyRuntimeAssetPath,
		WASMArtifact:   applyWASMPath,
		BuildExecuted:  isApplyBuildExecuted,
		BuildProfile:   applyConfig.profile,
	}
	applyManifestPath, applyManifestErr := applyPrerenderManifest(applyConfig.outDir, applySummary)
	if applyManifestErr != nil {
		return prerenderSummary{}, applyManifestErr
	}
	applySummary.ManifestPath = applyManifestPath
	return applySummary, nil
}

// applyPrerenderAssets copies selected asset directories into the output directory.
func applyPrerenderAssets(applyRootPath string, applyOutDir string, applyAssetDirs []string) ([]string, error) {
	applyCopied := []string{}
	for _, applyAssetDir := range applyAssetDirs {
		applyWalkErr := filepath.WalkDir(applyAssetDir, func(applyPath string, applyEntry fs.DirEntry, applyEntryErr error) error {
			if applyEntryErr != nil {
				return applyEntryErr
			}
			if applyEntry.IsDir() {
				return nil
			}
			applyRelativePath, applyRelativeErr := filepath.Rel(applyRootPath, applyPath)
			if applyRelativeErr != nil {
				return applyRelativeErr
			}
			applyTargetPath := filepath.Join(applyOutDir, applyRelativePath)
			if applyMkdirErr := os.MkdirAll(filepath.Dir(applyTargetPath), 0755); applyMkdirErr != nil {
				return applyMkdirErr
			}
			applyBytes, applyReadErr := os.ReadFile(applyPath)
			if applyReadErr != nil {
				return applyReadErr
			}
			if applyWriteErr := os.WriteFile(applyTargetPath, applyBytes, 0644); applyWriteErr != nil {
				return applyWriteErr
			}
			applyCopied = append(applyCopied, filepath.ToSlash(applyTargetPath))
			return nil
		})
		if applyWalkErr != nil {
			return nil, fmt.Errorf("copy asset directory %s: %w", applyAssetDir, applyWalkErr)
		}
	}
	sort.Strings(applyCopied)
	return applyCopied, nil
}

// applyPrerenderManifest writes a manifest for exported static artifacts.
func applyPrerenderManifest(applyOutDir string, applySummary prerenderSummary) (string, error) {
	applyManifestPath := filepath.Join(applyOutDir, "static-export-manifest.json")
	applyBytes, applyErr := json.MarshalIndent(applySummary, "", "  ")
	if applyErr != nil {
		return "", fmt.Errorf("encode prerender manifest: %w", applyErr)
	}
	applyBytes = append(applyBytes, '\n')
	if applyErr := os.WriteFile(applyManifestPath, applyBytes, 0644); applyErr != nil {
		return "", fmt.Errorf("write prerender manifest: %w", applyErr)
	}
	return applyManifestPath, nil
}

// renderPrerenderSummary prints a readable summary for static export runs.
func renderPrerenderSummary(renderSummary prerenderSummary) {
	fmt.Println("GWC prerender")
	fmt.Printf("  root:          %s\n", renderSummary.Root)
	fmt.Printf("  out:           %s\n", renderSummary.OutDir)
	fmt.Printf("  app:           %s\n", renderSummary.AppPath)
	fmt.Printf("  html shell:    %s\n", renderSummary.HTMLPath)
	fmt.Printf("  wasm:          %s\n", renderSummary.WASMArtifact)
	fmt.Printf("  wasm_exec.js:  %s\n", renderSummary.RuntimeAsset)
	fmt.Printf("  routes:        %s\n", strings.Join(renderSummary.Routes, ", "))
	fmt.Printf("  html files:    %d\n", len(renderSummary.HTMLFiles))
	fmt.Printf("  assets copied: %d\n", len(renderSummary.AssetFiles))
	fmt.Printf("  manifest:      %s\n", renderSummary.ManifestPath)
	if renderSummary.BuildExecuted {
		fmt.Printf("  build:         executed (%s profile)\n", renderSummary.BuildProfile)
	} else {
		fmt.Println("  build:         skipped")
	}
}
