package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (parseL launcher) runRelease(parseArgs []string) error {
	parseFs := flag.NewFlagSet("release", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "(deprecated) alias for -app; use -app")
	parseRoot := parseFs.String("root", "", "Project root used for output resolution")
	parseOutDir := parseFs.String("out-dir", "", "Release output directory")
	parseBinaryName := parseFs.String("binary-name", "", "Primary wasm artifact filename")
	parseManifestName := parseFs.String("manifest-name", "wasm-release-manifest.json", "Release manifest filename")
	parseBudgetsPath := parseFs.String("budgets", "", "Optional path to a JSON budgets file")
	parseCompareManifest := parseFs.String("compare-manifest", "", "Optional baseline release manifest to diff against")
	parseProfile := parseFs.String("profile", "release", "Release build profile: development, debug, ci, benchmark, release, or tinygo")
	parseCompression := parseFs.String("compression", "", "Compression sidecars: none, gzip, brotli, or gzip+brotli")
	parsePostLinkOpt := parseFs.String("post-link-opt", "", "Optional post-link optimization: none or wasm-opt")
	parseSizeAttribution := parseFs.String("size-attribution", "", "Optional size attribution: none or packages")
	parseStartupMeasure := parseFs.String("startup-measure", "", "Optional startup measurement: none or browser")
	parseStartupTimeoutMs := parseFs.Int("startup-timeout-ms", 30000, "Startup measurement timeout in milliseconds")
	parseValidateSmoke := parseFs.Bool("validate-smoke", false, "Run post-build release smoke validation")
	parseSkipCompression := parseFs.Bool("skip-compression", false, "Skip gzip sidecar generation")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parsePluginResults, parseErr2 := runLauncherPluginsForCapability("release_validator", "release", parseArgs, parseL.repoRoot, launcherActiveEnterpriseSources)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr3 := enforcePluginChecks(parsePluginResults, "release_validator"); parseErr3 != nil {
		return parseErr3
	}

	isParseSkipCompressionSet := false
	isParseCompressionSet := false
	parseFs.Visit(func(parseFlag *flag.Flag) {
		if parseFlag.Name == "skip-compression" {
			isParseSkipCompressionSet = true
		}
		if parseFlag.Name == "compression" {
			isParseCompressionSet = true
		}
	})
	if isParseSkipCompressionSet && isParseCompressionSet {
		return errors.New("use either -compression or -skip-compression, not both")
	}
	parseCompressionPolicy := *parseCompression
	if isParseSkipCompressionSet {
		parseCompressionPolicy = "none"
	}

	parseConfig, parseErr2 := resolveReleaseConfig(releaseConfig{
		appPath:          firstNonEmpty(*parseApp, *parseMainPath),
		rootPath:         *parseRoot,
		outDir:           *parseOutDir,
		binaryName:       *parseBinaryName,
		manifestName:     *parseManifestName,
		budgetsPath:      *parseBudgetsPath,
		compareManifest:  *parseCompareManifest,
		profile:          *parseProfile,
		compression:      parseCompressionPolicy,
		postLinkOpt:      *parsePostLinkOpt,
		sizeAttribution:  *parseSizeAttribution,
		startupMeasure:   *parseStartupMeasure,
		startupTimeoutMs: *parseStartupTimeoutMs,
		validateSmoke:    *parseValidateSmoke,
		compressionSet:   isParseCompressionSet || isParseSkipCompressionSet,
		skipCompression:  *parseSkipCompression,
		skipCompressSet:  isParseSkipCompressionSet,
		json:             *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr4 := enforceEnterpriseGoToolchainPolicy(parseConfig.rootPath, launcherActiveEnterpriseConfig.Policy); parseErr4 != nil {
		return parseErr4
	}
	if parseErr5 := enforceEnterpriseReleasePolicy(parseConfig, launcherActiveEnterpriseConfig.Policy); parseErr5 != nil {
		return parseErr5
	}

	parseSummary, parseErr2 := executeRelease(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printReleaseSummary(parseSummary)
	return nil
}

func (parseL launcher) runBuild(parseArgs []string) error {
	parseFs := flag.NewFlagSet("build", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "(deprecated) alias for -app; use -app")
	parseRoot := parseFs.String("root", "", "Project root used for output resolution")
	parseOut := parseFs.String("out", "", "WASM output path")
	parseOutput := parseFs.String("output", "", "(deprecated) alias for -out; use -out")
	parseProfile := parseFs.String("profile", "", "Build profile: development, debug, ci, benchmark, release, or tinygo")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	parseNoDoctor := parseFs.Bool("no-doctor", false, "Do not auto-run gwc doctor diagnosis when the build fails")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveBuildConfig(buildConfig{
		appPath:    firstNonEmpty(*parseApp, *parseMainPath),
		rootPath:   *parseRoot,
		outputPath: firstNonEmpty(*parseOut, *parseOutput),
		profile:    *parseProfile,
		json:       *parseJsonOutput,
	})
	if parseErr2 != nil {
		if !*parseNoDoctor && !*parseJsonOutput {
			parseL.diagnoseEnvironmentOnFailure()
		}
		return parseErr2
	}

	parseSummary, parseErr2 := buildExecuteBuild(parseConfig)
	if parseErr2 != nil {
		if !*parseNoDoctor && !parseConfig.json {
			parseL.diagnoseEnvironmentOnFailure()
		}
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printBuildSummary(parseSummary)
	return nil
}

func resolveBuildConfig(parseConfig buildConfig) (buildConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)
	isParseExplicitOutputPath := strings.TrimSpace(parseConfig.outputPath) != ""
	parseCwd, parseErr := buildGetwd()
	if parseErr != nil {
		return buildConfig{}, parseErr
	}

	parseMetadata, parseMetadataDir, hasMetadata, parseErr := resolveScaffoldMetadataForConfig(parseCwd, parseResolved.rootPath, parseResolved.appPath)
	if parseErr != nil {
		return buildConfig{}, parseErr
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
		if strings.TrimSpace(parseResolved.outputPath) == "" && strings.TrimSpace(parseMetadata.Tooling.WASMPath) != "" {
			parseResolved.outputPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.WASMPath))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.profile) == "" && strings.TrimSpace(parseMetadata.Tooling.DefaultBuildProfile) != "" {
			parseResolved.profile = strings.TrimSpace(parseMetadata.Tooling.DefaultBuildProfile)
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "gwc-start.json")
		}
	}
	if strings.TrimSpace(parseConfig.appPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.rootPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	if isParseExplicitOutputPath {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.profile) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "explicit flag")
	}

	if strings.TrimSpace(parseResolved.appPath) == "" {
		parseResolved.appPath, parseErr = detectAppPath(parseCwd)
		if parseErr != nil {
			return buildConfig{}, parseErr
		}
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "convention fallback")
	}
	parseResolved.appPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.appPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("resolve app path: %w", parseErr)
	}

	parseAppDir := parseResolved.appPath
	parseInfo, parseErr := os.Stat(parseResolved.appPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("inspect app path: %w", parseErr)
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
		return buildConfig{}, fmt.Errorf("resolve root path: %w", parseErr)
	}

	if strings.TrimSpace(parseResolved.outputPath) == "" {
		parseDefaultOutputPath, parseSource, parseErr2 := resolveLauncherDefaultBuildOutput(parseResolved.rootPath)
		if parseErr2 != nil {
			return buildConfig{}, parseErr2
		}
		parseResolved.outputPath = parseDefaultOutputPath
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", parseSource)
	}
	parseResolved.outputPath, parseErr = normalizePath(parseCwd, parseResolved.outputPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("resolve output path: %w", parseErr)
	}

	parseProfile, parseErr := resolveBuildProfile(strings.TrimSpace(parseResolved.profile))
	if parseErr != nil {
		return buildConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.profile) == "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "convention fallback")
	}
	parseResolved.profile = parseProfile.Name
	return parseResolved, nil
}

func resolveBuildProfile(parseProfile string) (buildProfile, error) {
	switch strings.TrimSpace(strings.ToLower(parseProfile)) {
	case "", "development", "dev":
		// -w keeps the inner loop smaller and faster. Use the debug profile
		// when preserving untrimmed paths and disabling optimization matters.
		return buildProfile{Name: "development", Toolchain: "go", Target: "js/wasm", Trimpath: false, Ldflags: "-w"}, nil
	case "debug", "dbg", "source-debug", "sourcedebug":
		return buildProfile{Name: "debug", Toolchain: "go", Target: "js/wasm", Trimpath: false, GCFlags: "all=-N -l"}, nil
	case "ci", "verification", "verify":
		return buildProfile{Name: "ci", Toolchain: "go", Target: "js/wasm", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false", Tags: "production"}, nil
	case "benchmark", "bench":
		return buildProfile{Name: "benchmark", Toolchain: "go", Target: "js/wasm", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false", Tags: "production"}, nil
	case "release", "production", "prod":
		return buildProfile{Name: "release", Toolchain: "go", Target: "js/wasm", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false", Tags: "production"}, nil
	case "tinygo", "tiny", "tinygo-release":
		return buildProfile{Name: "tinygo", Toolchain: "tinygo", Target: "wasm", Opt: "z", Tags: "production"}, nil
	default:
		return buildProfile{}, fmt.Errorf("unknown build profile %q", parseProfile)
	}
}

func resolveReleaseConfig(parseConfig releaseConfig) (releaseConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)
	isParseExplicitOutDir := strings.TrimSpace(parseConfig.outDir) != ""
	parseCwd, parseErr := buildGetwd()
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}

	parseMetadata, parseMetadataDir, hasMetadata, parseErr := resolveScaffoldMetadataForConfig(parseCwd, parseResolved.rootPath, parseResolved.appPath)
	if parseErr != nil {
		return releaseConfig{}, parseErr
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
		if strings.TrimSpace(parseResolved.outDir) == "" && strings.TrimSpace(parseMetadata.Tooling.ReleaseOutDir) != "" {
			parseResolved.outDir = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.ReleaseOutDir))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.binaryName) == "" && strings.TrimSpace(parseMetadata.Tooling.ReleaseBinaryName) != "" {
			parseResolved.binaryName = strings.TrimSpace(parseMetadata.Tooling.ReleaseBinaryName)
		}
		if strings.TrimSpace(parseResolved.budgetsPath) == "" && strings.TrimSpace(parseMetadata.Tooling.ReleaseBudgetsPath) != "" {
			parseResolved.budgetsPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.ReleaseBudgetsPath))
		}
		if !parseResolved.compressionSet {
			parseCompressionPolicy, parseErr2 := normalizeReleaseCompressionPolicy(parseMetadata.Tooling.ReleaseCompression)
			if parseErr2 != nil {
				return releaseConfig{}, parseErr2
			}
			parseResolved.compression = parseCompressionPolicy
		}
	}
	if strings.TrimSpace(parseConfig.appPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.rootPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	if isParseExplicitOutDir {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.profile) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "explicit flag")
	}

	if strings.TrimSpace(parseResolved.appPath) == "" {
		parseResolved.appPath, parseErr = detectAppPath(parseCwd)
		if parseErr != nil {
			return releaseConfig{}, parseErr
		}
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "convention fallback")
	}
	parseResolved.appPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.appPath)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("resolve app path: %w", parseErr)
	}

	parseAppDir := parseResolved.appPath
	parseInfo, parseErr := os.Stat(parseResolved.appPath)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("inspect app path: %w", parseErr)
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
		return releaseConfig{}, fmt.Errorf("resolve root path: %w", parseErr)
	}

	if strings.TrimSpace(parseResolved.outDir) == "" {
		parseDefaultOutDir, parseSource, parseErr3 := resolveLauncherDefaultReleaseOutDir(parseResolved.rootPath)
		if parseErr3 != nil {
			return releaseConfig{}, parseErr3
		}
		parseResolved.outDir = parseDefaultOutDir
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", parseSource)
	}
	parseResolved.outDir, parseErr = normalizePath(parseCwd, parseResolved.outDir)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("resolve release output directory: %w", parseErr)
	}

	parseResolved.binaryName = filepath.Base(strings.TrimSpace(firstNonEmpty(parseResolved.binaryName, defaultScaffoldReleaseBinaryName())))
	if parseResolved.binaryName == "." || parseResolved.binaryName == string(filepath.Separator) || parseResolved.binaryName == "" {
		return releaseConfig{}, errors.New("release binary name is required")
	}
	parseResolved.manifestName = filepath.Base(strings.TrimSpace(firstNonEmpty(parseResolved.manifestName, "wasm-release-manifest.json")))
	if parseResolved.manifestName == "." || parseResolved.manifestName == string(filepath.Separator) || parseResolved.manifestName == "" {
		return releaseConfig{}, errors.New("release manifest name is required")
	}
	if strings.TrimSpace(parseResolved.budgetsPath) != "" {
		parseResolved.budgetsPath, parseErr = normalizePath(parseCwd, parseResolved.budgetsPath)
		if parseErr != nil {
			return releaseConfig{}, fmt.Errorf("resolve budgets path: %w", parseErr)
		}
	}
	if strings.TrimSpace(parseResolved.compareManifest) != "" {
		parseResolved.compareManifest, parseErr = normalizeExistingPath(parseCwd, parseResolved.compareManifest)
		if parseErr != nil {
			return releaseConfig{}, fmt.Errorf("resolve compare manifest path: %w", parseErr)
		}
	}
	parseCompressionPolicy2, parseErr := normalizeReleaseCompressionPolicy(firstNonEmpty(parseResolved.compression, defaultScaffoldReleaseCompression()))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.compression = parseCompressionPolicy2
	parseResolved.skipCompression = parseCompressionPolicy2 == "none"
	parsePostLinkOpt, parseErr := normalizeReleasePostLinkOptimization(firstNonEmpty(parseResolved.postLinkOpt, "none"))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.postLinkOpt = parsePostLinkOpt
	parseSizeAttribution, parseErr := normalizeReleaseSizeAttributionMode(firstNonEmpty(parseResolved.sizeAttribution, "none"))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.sizeAttribution = parseSizeAttribution
	parseStartupMeasure, parseErr := normalizeReleaseStartupMeasureMode(firstNonEmpty(parseResolved.startupMeasure, "none"))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.startupMeasure = parseStartupMeasure
	if parseResolved.startupTimeoutMs <= 0 {
		parseResolved.startupTimeoutMs = 30000
	}

	parseProfile, parseErr := resolveBuildProfile(strings.TrimSpace(firstNonEmpty(parseResolved.profile, "release")))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.profile) == "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "convention fallback")
	}
	parseResolved.profile = parseProfile.Name
	return parseResolved, nil
}

func normalizeReleaseCompressionPolicy(parsePolicy string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parsePolicy)) {
	case "", "gzip":
		return "gzip", nil
	case "brotli", "br":
		return "brotli", nil
	case "gzip+brotli", "brotli+gzip", "both", "all":
		return "gzip+brotli", nil
	case "none", "off", "disabled":
		return "none", nil
	default:
		return "", fmt.Errorf("unknown release compression policy %q", parsePolicy)
	}
}

func normalizeReleasePostLinkOptimization(parseMode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "wasm-opt", "size", "optimize":
		return "wasm-opt", nil
	default:
		return "", fmt.Errorf("unknown release post-link optimization %q", parseMode)
	}
}

func normalizeReleaseSizeAttributionMode(parseMode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "packages", "package", "per-package":
		return "packages", nil
	default:
		return "", fmt.Errorf("unknown release size attribution mode %q", parseMode)
	}
}

func normalizeReleaseStartupMeasureMode(parseMode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "browser", "playwright":
		return "browser", nil
	default:
		return "", fmt.Errorf("unknown release startup measurement mode %q", parseMode)
	}
}

func executeBuild(parseConfig buildConfig) (buildSummary, error) {
	parseProfile, parseErr := resolveBuildProfile(parseConfig.profile)
	if parseErr != nil {
		return buildSummary{}, parseErr
	}
	parsePackageDir := parseConfig.appPath
	if parseInfo, parseErr2 := os.Stat(parseConfig.appPath); parseErr2 == nil && !parseInfo.IsDir() {
		parsePackageDir = filepath.Dir(parseConfig.appPath)
	}
	if parseErr3 := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr3 != nil {
		return buildSummary{}, fmt.Errorf("create build output directory: %w", parseErr3)
	}

	parseCommand, buildArgs, parseEnv, parseErr := buildCommandForProfile(parseProfile, parseConfig.outputPath)
	if parseErr != nil {
		return buildSummary{}, parseErr
	}
	parseOutput, parseErr := buildRunCommand(parseCommand, buildArgs, parsePackageDir, parseEnv)
	if parseErr != nil {
		parseTrimmed := strings.TrimSpace(parseOutput)
		if parseTrimmed == "" {
			return buildSummary{}, fmt.Errorf("%s build failed: %w", parseCommand, parseErr)
		}
		return buildSummary{}, fmt.Errorf("%s build failed: %s", parseCommand, parseTrimmed)
	}

	parseArtifactBytes, parseErr := os.ReadFile(parseConfig.outputPath)
	if parseErr != nil {
		return buildSummary{}, fmt.Errorf("read built wasm artifact: %w", parseErr)
	}
	parseArtifactInfo, parseErr := os.Stat(parseConfig.outputPath)
	if parseErr != nil {
		return buildSummary{}, fmt.Errorf("inspect built wasm artifact: %w", parseErr)
	}
	parseHash := sha256.Sum256(parseArtifactBytes)
	return buildSummary{
		OK:           true,
		Profile:      parseProfile,
		AppPath:      parseConfig.appPath,
		ProjectRoot:  parseConfig.rootPath,
		PackageDir:   parsePackageDir,
		OutputPath:   parseConfig.outputPath,
		Bytes:        parseArtifactInfo.Size(),
		SHA256:       fmt.Sprintf("%x", parseHash[:]),
		Resolution:   cloneResolutionTrace(parseConfig.resolution),
		SizeWarnings: detectHeavyWASMImports(parsePackageDir),
	}, nil
}

func buildCommandForProfile(parseProfile buildProfile, parseOutputPath string) (string, []string, []string, error) {
	parseToolchain := strings.TrimSpace(strings.ToLower(firstNonEmpty(parseProfile.Toolchain, "go")))
	switch parseToolchain {
	case "go":
		buildArgs := []string{"build", "-o", parseOutputPath}
		if parseProfile.Trimpath {
			buildArgs = append(buildArgs, "-trimpath")
		}
		if strings.TrimSpace(parseProfile.Ldflags) != "" {
			buildArgs = append(buildArgs, "-ldflags="+parseProfile.Ldflags)
		}
		if strings.TrimSpace(parseProfile.GCFlags) != "" {
			buildArgs = append(buildArgs, "-gcflags="+parseProfile.GCFlags)
		}
		if strings.TrimSpace(parseProfile.BuildVCS) != "" {
			buildArgs = append(buildArgs, "-buildvcs="+parseProfile.BuildVCS)
		}
		if strings.TrimSpace(parseProfile.Tags) != "" {
			buildArgs = append(buildArgs, "-tags", parseProfile.Tags)
		}
		buildArgs = append(buildArgs, ".")
		return "go", buildArgs, buildWasmGoEnv(), nil
	case "tinygo":
		if _, parseErr := buildLookPath("tinygo"); parseErr != nil {
			return "", nil, nil, fmt.Errorf("tinygo build profile requires TinyGo on PATH; install TinyGo or choose -profile release: %w", parseErr)
		}
		parseTarget := strings.TrimSpace(firstNonEmpty(parseProfile.Target, "wasm"))
		buildArgs := []string{"build", "-target=" + parseTarget, "-o", parseOutputPath}
		if strings.TrimSpace(parseProfile.Opt) != "" {
			buildArgs = append(buildArgs, "-opt="+strings.TrimSpace(parseProfile.Opt))
		}
		if strings.TrimSpace(parseProfile.Tags) != "" {
			buildArgs = append(buildArgs, "-tags", parseProfile.Tags)
		}
		buildArgs = append(buildArgs, ".")
		return "tinygo", buildArgs, os.Environ(), nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported build toolchain %q", parseProfile.Toolchain)
	}
}

// heavyWASMImports maps stdlib packages that disproportionately inflate wasm
// artifacts to actionable guidance.  One careless import anywhere in an app's
// dependency chain taxes every build forever, so every gwc build surfaces them.
var heavyWASMImports = map[string]string{
	"net/http":      "~1 MB (drags crypto/tls + x509 + DNS); browser apps should use the framework fetch package or js interop instead",
	"regexp":        "~290 kB; for simple patterns a hand-rolled match avoids linking the regexp engine",
	"encoding/xml":  "~300 kB; prefer encoding/json or a manual parser in wasm builds",
	"text/template": "~250 kB (plus reflect pressure); prefer direct string building in wasm builds",
	"html/template": "~400 kB; prefer the framework's html package for markup in wasm builds",
	"database/sql":  "driver-dependent but heavy; databases belong on the server side of a wasm app",
}

// detectHeavyWASMImports inspects the app's js/wasm dependency graph and
// returns a warning per known-heavy stdlib package found.  Failures are
// silent (nil): this is advisory tooling and must never break a build.
func detectHeavyWASMImports(parsePackageDir string) []string {
	parseCmd := exec.Command("go", "list", "-deps", ".")
	parseCmd.Dir = parsePackageDir
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseOutput, parseErr := parseCmd.Output()
	if parseErr != nil {
		return nil
	}
	parseDeps := make(map[string]bool)
	for parseLine := range strings.SplitSeq(string(parseOutput), "\n") {
		parseDeps[strings.TrimSpace(parseLine)] = true
	}
	parseHeavy := make([]string, 0, len(heavyWASMImports))
	for parsePkg := range heavyWASMImports {
		if parseDeps[parsePkg] {
			parseHeavy = append(parseHeavy, parsePkg)
		}
	}
	sort.Strings(parseHeavy)
	parseWarnings := make([]string, 0, len(parseHeavy))
	for _, parsePkg := range parseHeavy {
		parseWarnings = append(parseWarnings, fmt.Sprintf("wasm size: %s is linked — %s", parsePkg, heavyWASMImports[parsePkg]))
	}
	return parseWarnings
}

func executeRelease(parseConfig releaseConfig) (releaseSummary, error) {
	if parseErr := os.MkdirAll(parseConfig.outDir, 0755); parseErr != nil {
		return releaseSummary{}, fmt.Errorf("create release output directory: %w", parseErr)
	}
	parseWasmPath := filepath.Join(parseConfig.outDir, parseConfig.binaryName)
	buildSummary, parseErr2 := releaseExecuteBuild(buildConfig{
		appPath:    parseConfig.appPath,
		rootPath:   parseConfig.rootPath,
		outputPath: parseWasmPath,
		profile:    parseConfig.profile,
	})
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	parseArtifacts := map[string]releaseArtifactRecord{}
	parseOptimizer, parseErr2 := releaseApplyPostLinkOptimization(parseConfig.postLinkOpt, parseWasmPath)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	parseArtifacts["wasm"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, parseWasmPath)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	parseAttribution, parseErr2 := releaseWriteSizeAttribution(parseConfig.sizeAttribution, buildSummary.PackageDir, parseConfig.outDir)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseAttribution != nil {
		parseArtifacts["size_attribution"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseAttribution.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	isParseEmitGzip := parseConfig.compression == "gzip" || parseConfig.compression == "gzip+brotli"
	isParseEmitBrotli := parseConfig.compression == "brotli" || parseConfig.compression == "gzip+brotli"
	if isParseEmitGzip {
		parseGzipPath := parseWasmPath + ".gz"
		if parseErr3 := releaseWriteGzipSidecar(parseWasmPath, parseGzipPath); parseErr3 != nil {
			return releaseSummary{}, parseErr3
		}
		parseArtifacts["gzip"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, parseGzipPath)
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	if isParseEmitBrotli {
		parseBrotliPath := parseWasmPath + ".br"
		if parseErr4 := releaseWriteBrotliSidecar(parseWasmPath, parseBrotliPath); parseErr4 != nil {
			return releaseSummary{}, parseErr4
		}
		parseArtifacts["brotli"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, parseBrotliPath)
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	if strings.TrimSpace(parseConfig.budgetsPath) != "" {
		parseBudgets, parseErr5 := loadReleaseBudgets(parseConfig.budgetsPath)
		if parseErr5 != nil {
			return releaseSummary{}, parseErr5
		}
		if parseErr6 := assertReleaseBudgets(parseBudgets, parseArtifacts); parseErr6 != nil {
			return releaseSummary{}, parseErr6
		}
	}
	parseStartupConfig := parseConfig
	if parseStartupConfig.validateSmoke && parseStartupConfig.startupMeasure == "none" {
		parseStartupConfig.startupMeasure = "browser"
	}
	parseStartupReport, parseErr2 := releaseMeasureStartup(parseStartupConfig, parseArtifacts)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseStartupReport != nil {
		parseArtifacts["startup_report"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseStartupReport.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	parseManifestPath := filepath.Join(parseConfig.outDir, parseConfig.manifestName)
	parseDiffReport, parseErr2 := releaseWriteDiffReport(parseConfig.compareManifest, parseManifestPath, parseAttribution, parseArtifacts, parseConfig.outDir)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseDiffReport != nil {
		parseArtifacts["diff_report"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseDiffReport.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	parseManifestPayload := map[string]any{
		"package": buildSummary.PackageDir,
		"profile": buildSummary.Profile.Name,
		"goos":    "js",
		"goarch":  "wasm",
		"flags": map[string]any{
			"toolchain":            firstNonEmpty(buildSummary.Profile.Toolchain, "go"),
			"target":               firstNonEmpty(buildSummary.Profile.Target, "js/wasm"),
			"trimpath":             buildSummary.Profile.Trimpath,
			"ldflags":              buildSummary.Profile.Ldflags,
			"gcflags":              buildSummary.Profile.GCFlags,
			"buildvcs":             firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"opt":                  buildSummary.Profile.Opt,
			"compression":          !parseConfig.skipCompression,
			"compressionPolicy":    parseConfig.compression,
			"compareManifest":      parseConfig.compareManifest,
			"postLinkOptimization": parseConfig.postLinkOpt,
			"sizeAttribution":      parseConfig.sizeAttribution,
			"startupMeasure":       parseConfig.startupMeasure,
			"validateSmoke":        parseConfig.validateSmoke,
			"gzip":                 isParseEmitGzip,
			"brotli":               isParseEmitBrotli,
		},
		"artifacts": parseArtifacts,
	}
	if parseOptimizer != nil {
		parseManifestPayload["optimizer"] = parseOptimizer
	}
	if parseAttribution != nil {
		parseManifestPayload["attribution"] = parseAttribution
	}
	if parseStartupReport != nil {
		parseManifestPayload["startup"] = parseStartupReport
	}
	if parseDiffReport != nil {
		parseManifestPayload["diff"] = parseDiffReport
	}
	parseEncodedManifest, parseErr2 := releaseMarshalIndent(parseManifestPayload, "", "  ")
	if parseErr2 != nil {
		return releaseSummary{}, fmt.Errorf("encode release manifest: %w", parseErr2)
	}
	parseEncodedManifest = append(parseEncodedManifest, '\n')
	if parseErr7 := os.WriteFile(parseManifestPath, parseEncodedManifest, 0644); parseErr7 != nil {
		return releaseSummary{}, fmt.Errorf("write release manifest: %w", parseErr7)
	}
	parseValidation, parseErr2 := releaseValidateSmoke(parseConfig, parseManifestPath, parseArtifacts, parseStartupReport)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseValidation != nil {
		parseArtifacts["validation_report"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseValidation.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
		parseManifestPayload["validation"] = parseValidation
		parseManifestPayload["artifacts"] = parseArtifacts
		parseEncodedManifest, parseErr2 = releaseMarshalIndent(parseManifestPayload, "", "  ")
		if parseErr2 != nil {
			return releaseSummary{}, fmt.Errorf("encode release manifest: %w", parseErr2)
		}
		parseEncodedManifest = append(parseEncodedManifest, '\n')
		if parseErr8 := os.WriteFile(parseManifestPath, parseEncodedManifest, 0644); parseErr8 != nil {
			return releaseSummary{}, fmt.Errorf("write release manifest: %w", parseErr8)
		}
	}
	return releaseSummary{
		OK:           true,
		Profile:      buildSummary.Profile,
		AppPath:      buildSummary.AppPath,
		ProjectRoot:  buildSummary.ProjectRoot,
		PackageDir:   buildSummary.PackageDir,
		OutDir:       parseConfig.outDir,
		ManifestPath: parseManifestPath,
		Artifacts:    parseArtifacts,
		Flags: map[string]any{
			"toolchain":            firstNonEmpty(buildSummary.Profile.Toolchain, "go"),
			"target":               firstNonEmpty(buildSummary.Profile.Target, "js/wasm"),
			"trimpath":             buildSummary.Profile.Trimpath,
			"ldflags":              buildSummary.Profile.Ldflags,
			"gcflags":              buildSummary.Profile.GCFlags,
			"buildvcs":             firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"opt":                  buildSummary.Profile.Opt,
			"compression":          !parseConfig.skipCompression,
			"compressionPolicy":    parseConfig.compression,
			"compareManifest":      parseConfig.compareManifest,
			"postLinkOptimization": parseConfig.postLinkOpt,
			"sizeAttribution":      parseConfig.sizeAttribution,
			"startupMeasure":       parseConfig.startupMeasure,
			"validateSmoke":        parseConfig.validateSmoke,
			"gzip":                 isParseEmitGzip,
			"brotli":               isParseEmitBrotli,
		},
		Optimizer:   parseOptimizer,
		Attribution: parseAttribution,
		Diff:        parseDiffReport,
		Startup:     parseStartupReport,
		Validation:  parseValidation,
		Resolution:  cloneResolutionTrace(parseConfig.resolution),
	}, nil
}

func releaseApplyPostLinkOptimization(parseMode string, parseWasmPath string) (*releaseOptimizerRecord, error) {
	parseNormalizedMode, parseErr := normalizeReleasePostLinkOptimization(parseMode)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseNormalizedMode == "none" {
		return nil, nil
	}
	if parseNormalizedMode != "wasm-opt" {
		return nil, fmt.Errorf("unsupported release post-link optimization %q", parseNormalizedMode)
	}
	parseCommand, parseErr := resolveReleaseWasmOptCommand()
	if parseErr != nil {
		return nil, parseErr
	}
	if !parseCommand.Available {
		return nil, errors.New("post-link optimization requested but wasm-opt is unavailable; install wasm-opt and ensure it is on PATH")
	}

	parseOptimizedPath := parseWasmPath + ".opt"
	if parseErr2 := os.RemoveAll(parseOptimizedPath); parseErr2 != nil {
		return nil, fmt.Errorf("prepare optimized wasm artifact: %w", parseErr2)
	}
	parseArgs := append([]string{}, parseCommand.PrefixArgs...)
	parseArgs = append(parseArgs, parseWasmPath, "-Oz", "-o", parseOptimizedPath)
	if _, parseErr3 := releaseRunCommand(parseCommand.Command, parseArgs, filepath.Dir(parseWasmPath), os.Environ()); parseErr3 != nil {
		return nil, fmt.Errorf("run post-link optimizer %q: %w", parseNormalizedMode, parseErr3)
	}
	parseOptimizedBytes, parseErr := os.ReadFile(parseOptimizedPath)
	if parseErr != nil {
		return nil, fmt.Errorf("read optimized wasm artifact: %w", parseErr)
	}
	if parseErr4 := os.WriteFile(parseWasmPath, parseOptimizedBytes, 0644); parseErr4 != nil {
		return nil, fmt.Errorf("replace release wasm artifact with optimized output: %w", parseErr4)
	}
	if parseErr5 := os.Remove(parseOptimizedPath); parseErr5 != nil && !os.IsNotExist(parseErr5) {
		return nil, fmt.Errorf("cleanup optimized wasm artifact: %w", parseErr5)
	}
	return &releaseOptimizerRecord{
		Mode: parseNormalizedMode,
		Tool: parseCommand.Label,
		Args: parseArgs,
	}, nil
}

type releaseCommandInfo struct {
	Available  bool
	Command    string
	PrefixArgs []string
	Label      string
}

func resolveReleaseWasmOptCommand() (releaseCommandInfo, error) {
	if parsePath, parseErr := releaseLookPath("wasm-opt"); parseErr == nil {
		return releaseCommandInfo{
			Available: true,
			Command:   "wasm-opt",
			Label:     parsePath,
		}, nil
	}
	return releaseCommandInfo{}, nil
}

func releaseWriteSizeAttribution(parseMode string, parsePackageDir string, parseOutDir string) (*releaseAttributionRecord, error) {
	parseNormalizedMode, parseErr := normalizeReleaseSizeAttributionMode(parseMode)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseNormalizedMode == "none" {
		return nil, nil
	}
	if parseNormalizedMode != "packages" {
		return nil, fmt.Errorf("unsupported release size attribution mode %q", parseNormalizedMode)
	}

	parsePackages, parseErr := releaseCollectPackageSizeAttributionForEntrypoints([]string{parsePackageDir})
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload := map[string]any{
		"mode":     parseNormalizedMode,
		"package":  parsePackageDir,
		"goos":     "js",
		"goarch":   "wasm",
		"packages": parsePackages,
	}
	parseEncoded, parseErr := releaseMarshalIndent(parsePayload, "", "  ")
	if parseErr != nil {
		return nil, fmt.Errorf("encode release size attribution: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	parseFileName := "wasm-package-size-attribution.json"
	if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, parseFileName), parseEncoded, 0644); parseErr2 != nil {
		return nil, fmt.Errorf("write release size attribution: %w", parseErr2)
	}
	return &releaseAttributionRecord{
		Mode:         parseNormalizedMode,
		Path:         parseFileName,
		PackageCount: len(parsePackages),
	}, nil
}

func releaseCollectPackageSizeAttribution(parsePackageDir string) ([]releasePackageSizeRecord, error) {
	return releaseCollectPackageSizeAttributionForEntrypoints([]string{parsePackageDir})
}

func releaseCollectPackageSizeAttributionForEntrypoints(parsePackageDirs []string) ([]releasePackageSizeRecord, error) {
	parseUniqueDirs := releaseUniqueAttributionPackageDirs(parsePackageDirs)
	parseRecordsByImportPath := make(map[string]releasePackageSizeRecord)
	for _, parsePackageDir := range parseUniqueDirs {
		parseOutput, parseErr := releaseRunCommand("go", []string{"list", "-deps", "-json", "-export", "."}, parsePackageDir, buildWasmGoEnv())
		if parseErr != nil {
			return nil, fmt.Errorf("collect release package attribution: %w", parseErr)
		}
		parseDecoder := json.NewDecoder(strings.NewReader(parseOutput))
		for {
			var parsePkg releaseGoListPackage
			if parseErr2 := parseDecoder.Decode(&parsePkg); parseErr2 != nil {
				if errors.Is(parseErr2, io.EOF) {
					break
				}
				return nil, fmt.Errorf("decode release package attribution: %w", parseErr2)
			}
			if strings.TrimSpace(parsePkg.ImportPath) == "" {
				continue
			}
			parseRecord, parseErr3 := releaseBuildPackageSizeRecord(parsePkg)
			if parseErr3 != nil {
				return nil, parseErr3
			}
			if parseRecord.ArchiveBytes == 0 && parseRecord.SourceBytes == 0 && parseRecord.FileCount == 0 {
				continue
			}
			parseExisting, hasExisting := parseRecordsByImportPath[parseRecord.ImportPath]
			if hasExisting {
				parseRecordsByImportPath[parseRecord.ImportPath] = releaseMergePackageSizeRecord(parseExisting, parseRecord)
				continue
			}
			parseRecordsByImportPath[parseRecord.ImportPath] = parseRecord
		}
	}

	parsePackages := make([]releasePackageSizeRecord, 0, len(parseRecordsByImportPath))
	for _, parseRecord := range parseRecordsByImportPath {
		parsePackages = append(parsePackages, parseRecord)
	}
	sort.Slice(parsePackages, func(parseI int, parseJ int) bool {
		if parsePackages[parseI].ArchiveBytes != parsePackages[parseJ].ArchiveBytes {
			return parsePackages[parseI].ArchiveBytes > parsePackages[parseJ].ArchiveBytes
		}
		if parsePackages[parseI].SourceBytes != parsePackages[parseJ].SourceBytes {
			return parsePackages[parseI].SourceBytes > parsePackages[parseJ].SourceBytes
		}
		return parsePackages[parseI].ImportPath < parsePackages[parseJ].ImportPath
	})
	return parsePackages, nil
}

func releaseUniqueAttributionPackageDirs(parsePackageDirs []string) []string {
	parseSeen := make(map[string]struct{}, len(parsePackageDirs))
	parseUnique := make([]string, 0, len(parsePackageDirs))
	for _, parsePackageDir := range parsePackageDirs {
		parseCleanDir := strings.TrimSpace(parsePackageDir)
		if parseCleanDir != "" {
			parseCleanDir = filepath.Clean(parseCleanDir)
		}
		if _, parseOk := parseSeen[parseCleanDir]; parseOk {
			continue
		}
		parseSeen[parseCleanDir] = struct{}{}
		parseUnique = append(parseUnique, parseCleanDir)
	}
	return parseUnique
}

func releaseMergePackageSizeRecord(parseExisting releasePackageSizeRecord, parseNext releasePackageSizeRecord) releasePackageSizeRecord {
	if strings.TrimSpace(parseExisting.Dir) == "" && strings.TrimSpace(parseNext.Dir) != "" {
		parseExisting.Dir = parseNext.Dir
	}
	if parseNext.ArchiveBytes > parseExisting.ArchiveBytes {
		parseExisting.ArchiveBytes = parseNext.ArchiveBytes
	}
	if parseNext.SourceBytes > parseExisting.SourceBytes {
		parseExisting.SourceBytes = parseNext.SourceBytes
	}
	if parseNext.FileCount > parseExisting.FileCount {
		parseExisting.FileCount = parseNext.FileCount
	}
	return parseExisting
}

func releaseBuildPackageSizeRecord(parsePkg releaseGoListPackage) (releasePackageSizeRecord, error) {
	parseRecord := releasePackageSizeRecord{
		ImportPath: strings.TrimSpace(parsePkg.ImportPath),
		Dir:        strings.TrimSpace(parsePkg.Dir),
	}
	if strings.TrimSpace(parsePkg.Export) != "" {
		parseInfo, parseErr := os.Stat(parsePkg.Export)
		if parseErr != nil {
			return releasePackageSizeRecord{}, fmt.Errorf("inspect export archive for %s: %w", parsePkg.ImportPath, parseErr)
		}
		parseRecord.ArchiveBytes = parseInfo.Size()
	}
	parseSourceFiles := map[string]struct{}{}
	for _, parseFile := range append(
		append(append(append(append(append(append(append(append(append([]string{}, parsePkg.GoFiles...), parsePkg.CgoFiles...), parsePkg.CFiles...), parsePkg.CXXFiles...), parsePkg.MFiles...), parsePkg.HFiles...), parsePkg.FFiles...), parsePkg.SFiles...), parsePkg.SysoFiles...),
		parsePkg.EmbedFiles...,
	) {
		parseTrimmed := strings.TrimSpace(parseFile)
		if parseTrimmed == "" || strings.TrimSpace(parsePkg.Dir) == "" {
			continue
		}
		parseSourceFiles[filepath.Join(parsePkg.Dir, filepath.FromSlash(parseTrimmed))] = struct{}{}
	}
	parseRecord.FileCount = len(parseSourceFiles)
	for parseFile2 := range parseSourceFiles {
		parseInfo2, parseErr2 := os.Stat(parseFile2)
		if parseErr2 != nil {
			return releasePackageSizeRecord{}, fmt.Errorf("inspect source file for %s: %w", parsePkg.ImportPath, parseErr2)
		}
		parseRecord.SourceBytes += parseInfo2.Size()
	}
	return parseRecord, nil
}
