package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	tailwindDefaultVersion      = "v4.1.17"
	tailwindReleaseDownloadBase = "https://github.com/tailwindlabs/tailwindcss/releases/download"
)

var runTailwindCommand = func(l launcher, args []string) error {
	return l.runTailwind(args)
}

var tailwindGetwd = os.Getwd

var tailwindHTTPClient = &http.Client{Timeout: 2 * time.Minute}

var tailwindFetchURLBytes = fetchTailwindURLBytes

var tailwindRunCommand = executeTailwindCommand

type tailwindConfig struct {
	rootPath            string
	staticDirPath       string
	inputFilePath       string
	outputFilePath      string
	manifestFilePath    string
	binaryFilePath      string
	cacheDirPath        string
	versionTag          string
	shouldSkipManifest  bool
	shouldForceDownload bool
	shouldPrintJSON     bool
}

type tailwindSummary struct {
	OK               bool   `json:"ok"`
	RootPath         string `json:"rootPath"`
	StaticDirPath    string `json:"staticDirPath"`
	InputFilePath    string `json:"inputFilePath"`
	OutputFilePath   string `json:"outputFilePath"`
	ManifestFilePath string `json:"manifestFilePath,omitempty"`
	BinaryFilePath   string `json:"binaryFilePath"`
	VersionTag       string `json:"versionTag"`
	Bytes            int64  `json:"bytes"`
	CommandOutput    string `json:"commandOutput,omitempty"`
}

type tailwindManifestRecord struct {
	relativeFilePath string
	literalValues    []string
}

// runTailwind parses launcher flags and builds shared Tailwind CSS assets.
func (l launcher) runTailwind(args []string) error {
	flagSet := flag.NewFlagSet("tailwind", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)

	rootPath := flagSet.String("root", "", "Project root used to resolve Tailwind defaults")
	staticDirPath := flagSet.String("static-dir", "", "Static directory that owns tailwind.input.css")
	inputFilePath := flagSet.String("input", "", "Tailwind input CSS file")
	outputFilePath := flagSet.String("output", "", "Tailwind output CSS file")
	manifestFilePath := flagSet.String("manifest", "", "Tailwind manifest output file")
	binaryFilePath := flagSet.String("binary", "", "Optional explicit Tailwind CLI binary path")
	cacheDirPath := flagSet.String("cache-dir", "", "Directory that stores downloaded Tailwind CLI binaries")
	versionTag := flagSet.String("version", tailwindDefaultVersion, "Tailwind CLI release tag (for example v4.1.17)")
	shouldSkipManifest := flagSet.Bool("skip-manifest", false, "Skip regenerating generated/tailwind-manifest.html")
	shouldForceDownload := flagSet.Bool("force-download", false, "Force re-downloading the Tailwind CLI binary")
	shouldPrintJSON := flagSet.Bool("json", false, "Emit machine-readable JSON output")
	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveTailwindConfig(tailwindConfig{
		rootPath:            firstNonEmpty(*rootPath, l.repoRoot),
		staticDirPath:       *staticDirPath,
		inputFilePath:       *inputFilePath,
		outputFilePath:      *outputFilePath,
		manifestFilePath:    *manifestFilePath,
		binaryFilePath:      *binaryFilePath,
		cacheDirPath:        *cacheDirPath,
		versionTag:          *versionTag,
		shouldSkipManifest:  *shouldSkipManifest,
		shouldForceDownload: *shouldForceDownload,
		shouldPrintJSON:     *shouldPrintJSON,
	})
	if err != nil {
		return err
	}

	summary, err := executeTailwindBuild(config)
	if err != nil {
		return err
	}
	if config.shouldPrintJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printTailwindSummary(summary)
	return nil
}

// resolveTailwindConfig normalizes CLI inputs and applies repo conventions.
func resolveTailwindConfig(config tailwindConfig) (tailwindConfig, error) {
	resolvedConfig := config
	cwdPath, err := tailwindGetwd()
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind working directory: %w", err)
	}

	resolvedConfig.rootPath = strings.TrimSpace(resolvedConfig.rootPath)
	if resolvedConfig.rootPath == "" {
		resolvedConfig.rootPath = cwdPath
	}
	resolvedConfig.rootPath, err = normalizePath(cwdPath, resolvedConfig.rootPath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind root path: %w", err)
	}
	rootInfo, err := os.Stat(resolvedConfig.rootPath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("stat tailwind root path: %w", err)
	}
	if !rootInfo.IsDir() {
		return tailwindConfig{}, fmt.Errorf("tailwind root path is not a directory: %s", resolvedConfig.rootPath)
	}

	if strings.TrimSpace(resolvedConfig.staticDirPath) == "" {
		resolvedConfig.staticDirPath = filepath.Join(resolvedConfig.rootPath, "examples", "static")
	}
	resolvedConfig.staticDirPath, err = normalizePath(cwdPath, resolvedConfig.staticDirPath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind static dir: %w", err)
	}
	staticInfo, err := os.Stat(resolvedConfig.staticDirPath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("stat tailwind static dir: %w", err)
	}
	if !staticInfo.IsDir() {
		return tailwindConfig{}, fmt.Errorf("tailwind static dir is not a directory: %s", resolvedConfig.staticDirPath)
	}

	if strings.TrimSpace(resolvedConfig.inputFilePath) == "" {
		resolvedConfig.inputFilePath = filepath.Join(resolvedConfig.staticDirPath, "tailwind.input.css")
	}
	resolvedConfig.inputFilePath, err = normalizeExistingPath(cwdPath, resolvedConfig.inputFilePath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind input file: %w", err)
	}

	if strings.TrimSpace(resolvedConfig.outputFilePath) == "" {
		resolvedConfig.outputFilePath = filepath.Join(resolvedConfig.staticDirPath, "css", "tailwind.css")
	}
	resolvedConfig.outputFilePath, err = normalizePath(cwdPath, resolvedConfig.outputFilePath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind output file: %w", err)
	}

	if strings.TrimSpace(resolvedConfig.manifestFilePath) == "" {
		resolvedConfig.manifestFilePath = filepath.Join(resolvedConfig.staticDirPath, "generated", "tailwind-manifest.html")
	}
	resolvedConfig.manifestFilePath, err = normalizePath(cwdPath, resolvedConfig.manifestFilePath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind manifest file: %w", err)
	}

	if strings.TrimSpace(resolvedConfig.cacheDirPath) == "" {
		resolvedConfig.cacheDirPath = filepath.Join(resolvedConfig.rootPath, "third_party", "tailwindcss", "bin")
	}
	resolvedConfig.cacheDirPath, err = normalizePath(cwdPath, resolvedConfig.cacheDirPath)
	if err != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind cache dir: %w", err)
	}

	resolvedConfig.versionTag, err = normalizeTailwindVersionTag(resolvedConfig.versionTag)
	if err != nil {
		return tailwindConfig{}, err
	}

	resolvedConfig.binaryFilePath = strings.TrimSpace(resolvedConfig.binaryFilePath)
	if resolvedConfig.binaryFilePath != "" {
		resolvedConfig.binaryFilePath, err = normalizeExistingPath(cwdPath, resolvedConfig.binaryFilePath)
		if err != nil {
			return tailwindConfig{}, fmt.Errorf("resolve tailwind binary file: %w", err)
		}
	}

	return resolvedConfig, nil
}

// normalizeTailwindVersionTag validates and normalizes a Tailwind release tag.
func normalizeTailwindVersionTag(versionTag string) (string, error) {
	normalizedTag := strings.TrimSpace(versionTag)
	if normalizedTag == "" {
		normalizedTag = tailwindDefaultVersion
	}
	if !strings.HasPrefix(normalizedTag, "v") {
		normalizedTag = "v" + normalizedTag
	}
	if strings.ContainsAny(normalizedTag, " \t\r\n/\\") {
		return "", fmt.Errorf("invalid tailwind version tag %q", versionTag)
	}
	return normalizedTag, nil
}

// executeTailwindBuild regenerates the manifest, resolves the CLI binary, and writes tailwind.css.
func executeTailwindBuild(config tailwindConfig) (tailwindSummary, error) {
	summary := tailwindSummary{
		OK:             true,
		RootPath:       config.rootPath,
		StaticDirPath:  config.staticDirPath,
		InputFilePath:  config.inputFilePath,
		OutputFilePath: config.outputFilePath,
		VersionTag:     config.versionTag,
	}

	if !config.shouldSkipManifest {
		manifestPath, err := buildTailwindManifestFile(config)
		if err != nil {
			return tailwindSummary{}, err
		}
		summary.ManifestFilePath = manifestPath
	}

	binaryPath, err := resolveTailwindBinaryPath(config)
	if err != nil {
		return tailwindSummary{}, err
	}
	if strings.TrimSpace(config.binaryFilePath) == "" {
		if err := ensureTailwindBinaryFile(config, binaryPath); err != nil {
			return tailwindSummary{}, err
		}
	}
	summary.BinaryFilePath = binaryPath

	if err := os.MkdirAll(filepath.Dir(config.outputFilePath), 0o755); err != nil {
		return tailwindSummary{}, fmt.Errorf("prepare tailwind output directory: %w", err)
	}

	commandOutput, err := tailwindRunCommand(binaryPath, config.inputFilePath, config.outputFilePath, config.rootPath)
	if err != nil {
		return tailwindSummary{}, err
	}
	summary.CommandOutput = commandOutput

	outputInfo, err := os.Stat(config.outputFilePath)
	if err != nil {
		return tailwindSummary{}, fmt.Errorf("stat tailwind output file: %w", err)
	}
	summary.Bytes = outputInfo.Size()
	return summary, nil
}

// resolveTailwindBinaryPath chooses the explicit or cached Tailwind binary path.
func resolveTailwindBinaryPath(config tailwindConfig) (string, error) {
	if strings.TrimSpace(config.binaryFilePath) != "" {
		return config.binaryFilePath, nil
	}
	assetName, err := resolveTailwindAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	return filepath.Join(config.cacheDirPath, config.versionTag, assetName), nil
}

// resolveTailwindAssetName maps GOOS and GOARCH to the official Tailwind release asset name.
func resolveTailwindAssetName(goos string, goarch string) (string, error) {
	switch goos {
	case "windows":
		if goarch == "amd64" {
			return "tailwindcss-windows-x64.exe", nil
		}
	case "darwin":
		switch goarch {
		case "amd64":
			return "tailwindcss-macos-x64", nil
		case "arm64":
			return "tailwindcss-macos-arm64", nil
		}
	case "linux":
		switch goarch {
		case "amd64":
			return "tailwindcss-linux-x64", nil
		case "arm64":
			return "tailwindcss-linux-arm64", nil
		}
	}
	return "", fmt.Errorf("unsupported tailwind platform: %s/%s", goos, goarch)
}

// ensureTailwindBinaryFile downloads the requested Tailwind binary when it is missing or stale.
func ensureTailwindBinaryFile(config tailwindConfig, binaryFilePath string) error {
	if !config.shouldForceDownload {
		if binaryInfo, err := os.Stat(binaryFilePath); err == nil && !binaryInfo.IsDir() {
			return nil
		}
	}
	assetName, err := resolveTailwindAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	return downloadTailwindBinaryFile(config.versionTag, assetName, binaryFilePath)
}

// downloadTailwindBinaryFile downloads a Tailwind release binary and writes it to a stable cache path.
func downloadTailwindBinaryFile(versionTag string, assetName string, binaryFilePath string) error {
	checksumURL := fmt.Sprintf("%s/%s/sha256sums.txt", tailwindReleaseDownloadBase, versionTag)
	checksumBytes, checksumErr := tailwindFetchURLBytes(checksumURL)

	assetURL := fmt.Sprintf("%s/%s/%s", tailwindReleaseDownloadBase, versionTag, assetName)
	assetBytes, err := tailwindFetchURLBytes(assetURL)
	if err != nil {
		return fmt.Errorf("download tailwind binary: %w", err)
	}
	if checksumErr == nil {
		checksumByAsset := parseTailwindChecksums(string(checksumBytes))
		if expectedChecksum, hasChecksum := checksumByAsset[assetName]; hasChecksum {
			if err := verifyTailwindChecksum(assetBytes, expectedChecksum); err != nil {
				return fmt.Errorf("verify tailwind checksum: %w", err)
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(binaryFilePath), 0o755); err != nil {
		return fmt.Errorf("prepare tailwind cache directory: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(binaryFilePath), filepath.Base(binaryFilePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create tailwind temp file: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
	}()

	if _, err := tempFile.Write(assetBytes); err != nil {
		return fmt.Errorf("write tailwind temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close tailwind temp file: %w", err)
	}
	if err := os.Chmod(tempFilePath, 0o755); err != nil {
		return fmt.Errorf("chmod tailwind binary: %w", err)
	}
	if err := os.Remove(binaryFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("replace existing tailwind binary: %w", err)
	}
	if err := os.Rename(tempFilePath, binaryFilePath); err != nil {
		return fmt.Errorf("finalize tailwind binary: %w", err)
	}
	return nil
}

// fetchTailwindURLBytes downloads one URL and returns its response bytes.
func fetchTailwindURLBytes(downloadURL string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("prepare tailwind download request: %w", err)
	}
	response, err := tailwindHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %s", downloadURL, response.Status)
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", downloadURL, err)
	}
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("download %s: empty response", downloadURL)
	}
	return bodyBytes, nil
}

// parseTailwindChecksums parses sha256sums.txt into asset-name keyed hashes.
func parseTailwindChecksums(checksumText string) map[string]string {
	checksumByAsset := map[string]string{}
	for _, lineText := range strings.Split(checksumText, "\n") {
		trimmedLine := strings.TrimSpace(lineText)
		if trimmedLine == "" {
			continue
		}
		fields := strings.Fields(trimmedLine)
		if len(fields) < 2 {
			continue
		}
		assetName := strings.TrimPrefix(fields[1], "*")
		checksumByAsset[assetName] = strings.ToLower(fields[0])
	}
	return checksumByAsset
}

// verifyTailwindChecksum validates downloaded bytes against an expected SHA-256 hash.
func verifyTailwindChecksum(assetBytes []byte, expectedChecksum string) error {
	computedChecksum := sha256.Sum256(assetBytes)
	computedChecksumText := strings.ToLower(hex.EncodeToString(computedChecksum[:]))
	if computedChecksumText != strings.ToLower(strings.TrimSpace(expectedChecksum)) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expectedChecksum, computedChecksumText)
	}
	return nil
}

// executeTailwindCommand invokes the Tailwind CLI with the repository input and output files.
func executeTailwindCommand(binaryFilePath string, inputFilePath string, outputFilePath string, workDirPath string) (string, error) {
	command := exec.Command(binaryFilePath, "-i", inputFilePath, "-o", outputFilePath)
	command.Dir = workDirPath
	commandOutput, err := command.CombinedOutput()
	trimmedOutput := strings.TrimSpace(string(commandOutput))
	if err != nil {
		if trimmedOutput == "" {
			return "", fmt.Errorf("run tailwind cli: %w", err)
		}
		return trimmedOutput, fmt.Errorf("run tailwind cli: %s", trimmedOutput)
	}
	return trimmedOutput, nil
}

// buildTailwindManifestFile renders generated/tailwind-manifest.html from scanned source literals.
func buildTailwindManifestFile(config tailwindConfig) (string, error) {
	manifestRows, err := collectTailwindManifestRows(config.staticDirPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(config.manifestFilePath), 0o755); err != nil {
		return "", fmt.Errorf("prepare tailwind manifest directory: %w", err)
	}

	manifestParts := []string{
		"<!DOCTYPE html>",
		"<html>",
		"<body>",
	}
	manifestParts = append(manifestParts, manifestRows...)
	manifestParts = append(manifestParts, "</body>", "</html>", "")
	manifestText := strings.Join(manifestParts, "\n")

	if err := os.WriteFile(config.manifestFilePath, []byte(manifestText), 0o644); err != nil {
		return "", fmt.Errorf("write tailwind manifest: %w", err)
	}
	return config.manifestFilePath, nil
}

// collectTailwindManifestRows scans examples sources and produces deterministic manifest rows.
func collectTailwindManifestRows(staticDirPath string) ([]string, error) {
	examplesDirPath := filepath.Dir(staticDirPath)
	manifestRecords := make([]tailwindManifestRecord, 0, 256)
	walkErr := filepath.WalkDir(examplesDirPath, func(walkFilePath string, walkDirEntry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if walkFilePath == examplesDirPath {
			return nil
		}
		if walkDirEntry.IsDir() {
			if shouldSkipTailwindDirName(walkDirEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldIncludeTailwindExtension(filepath.Ext(walkDirEntry.Name())) {
			return nil
		}

		sourceBytes, err := os.ReadFile(walkFilePath)
		if err != nil {
			return fmt.Errorf("read tailwind source file %s: %w", walkFilePath, err)
		}
		literalValues := extractTailwindLiteralValues(string(sourceBytes))
		if len(literalValues) == 0 {
			return nil
		}
		relativeFilePath, err := filepath.Rel(examplesDirPath, walkFilePath)
		if err != nil {
			return fmt.Errorf("resolve tailwind relative path for %s: %w", walkFilePath, err)
		}
		manifestRecords = append(manifestRecords, tailwindManifestRecord{
			relativeFilePath: filepath.ToSlash(relativeFilePath),
			literalValues:    literalValues,
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk tailwind source files: %w", walkErr)
	}

	sort.Slice(manifestRecords, func(leftIndex int, rightIndex int) bool {
		return manifestRecords[leftIndex].relativeFilePath < manifestRecords[rightIndex].relativeFilePath
	})

	manifestRows := make([]string, 0, len(manifestRecords)*3)
	for _, manifestRecord := range manifestRecords {
		manifestRows = append(manifestRows, "<!-- "+escapeTailwindHTML(manifestRecord.relativeFilePath)+" -->")
		for _, literalValue := range manifestRecord.literalValues {
			manifestRows = append(manifestRows, `<div class="`+escapeTailwindHTML(literalValue)+`"></div>`)
		}
	}
	return manifestRows, nil
}

// shouldSkipTailwindDirName reports whether a directory is excluded from manifest scanning.
func shouldSkipTailwindDirName(dirName string) bool {
	switch strings.ToLower(strings.TrimSpace(dirName)) {
	case ".git", ".npm-cache", "bin", "css", "generated", "images", "modules", "node_modules":
		return true
	default:
		return false
	}
}

// shouldIncludeTailwindExtension reports whether a file extension is scanned for class literals.
func shouldIncludeTailwindExtension(fileExtension string) bool {
	switch strings.ToLower(strings.TrimSpace(fileExtension)) {
	case ".go", ".html", ".js", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

// extractTailwindLiteralValues returns normalized string-literal candidates for Tailwind scanning.
func extractTailwindLiteralValues(sourceText string) []string {
	literalValues := make([]string, 0, 64)
	sourceIndex := 0
	for sourceIndex < len(sourceText) {
		sourceByte := sourceText[sourceIndex]
		if sourceByte != '"' && sourceByte != '`' {
			sourceIndex++
			continue
		}
		literalValue, nextIndex, hasClosedQuote := parseTailwindQuotedLiteral(sourceText, sourceIndex, sourceByte)
		if !hasClosedQuote {
			sourceIndex++
			continue
		}
		sourceIndex = nextIndex
		normalizedLiteral := normalizeTailwindLiteralValue(literalValue)
		if normalizedLiteral != "" {
			literalValues = append(literalValues, normalizedLiteral)
		}
	}
	return literalValues
}

// parseTailwindQuotedLiteral parses one quoted or raw string literal from source text.
func parseTailwindQuotedLiteral(sourceText string, startIndex int, quoteByte byte) (string, int, bool) {
	var literalBuilder strings.Builder
	parseIndex := startIndex + 1
	shouldEscapeNext := false
	for parseIndex < len(sourceText) {
		currentByte := sourceText[parseIndex]
		if quoteByte == '"' {
			if shouldEscapeNext {
				literalBuilder.WriteByte(currentByte)
				shouldEscapeNext = false
				parseIndex++
				continue
			}
			if currentByte == '\\' {
				literalBuilder.WriteByte(currentByte)
				shouldEscapeNext = true
				parseIndex++
				continue
			}
		}
		if currentByte == quoteByte {
			return literalBuilder.String(), parseIndex + 1, true
		}
		literalBuilder.WriteByte(currentByte)
		parseIndex++
	}
	return "", len(sourceText), false
}

// normalizeTailwindLiteralValue mirrors the Tailwind manifest literal normalization rules.
func normalizeTailwindLiteralValue(literalValue string) string {
	normalizedValue := literalValue
	normalizedValue = strings.ReplaceAll(normalizedValue, `\"`, `"`)
	normalizedValue = strings.ReplaceAll(normalizedValue, `\\`, `\`)
	normalizedValue = strings.ReplaceAll(normalizedValue, "\\`", "`")
	normalizedValue = strings.ReplaceAll(normalizedValue, `\n`, " ")
	normalizedValue = strings.ReplaceAll(normalizedValue, `\r`, " ")
	normalizedValue = strings.ReplaceAll(normalizedValue, `\t`, " ")
	return strings.Join(strings.Fields(normalizedValue), " ")
}

// escapeTailwindHTML escapes HTML-special characters for manifest-safe output.
func escapeTailwindHTML(sourceText string) string {
	escapedValue := strings.ReplaceAll(sourceText, "&", "&amp;")
	escapedValue = strings.ReplaceAll(escapedValue, "<", "&lt;")
	escapedValue = strings.ReplaceAll(escapedValue, ">", "&gt;")
	escapedValue = strings.ReplaceAll(escapedValue, `"`, "&quot;")
	return escapedValue
}

// printTailwindSummary writes a concise human-readable tailwind build summary.
func printTailwindSummary(summary tailwindSummary) {
	fmt.Println("GWC tailwind")
	fmt.Printf("  root:         %s\n", summary.RootPath)
	fmt.Printf("  static dir:   %s\n", summary.StaticDirPath)
	fmt.Printf("  input:        %s\n", summary.InputFilePath)
	fmt.Printf("  output:       %s\n", summary.OutputFilePath)
	if strings.TrimSpace(summary.ManifestFilePath) != "" {
		fmt.Printf("  manifest:     %s\n", summary.ManifestFilePath)
	} else {
		fmt.Println("  manifest:     <skipped>")
	}
	fmt.Printf("  binary:       %s\n", summary.BinaryFilePath)
	fmt.Printf("  version:      %s\n", summary.VersionTag)
	fmt.Printf("  bytes:        %d\n", summary.Bytes)
	if strings.TrimSpace(summary.CommandOutput) != "" {
		// ensure CLI notes remain visible when Tailwind emits upgrade hints or warnings.
		fmt.Printf("  cli output:   %s\n", summary.CommandOutput)
	}
}
