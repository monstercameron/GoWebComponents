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
func (parseL launcher) runTailwind(parseArgs []string) error {
	parseFlagSet := flag.NewFlagSet("tailwind", flag.ContinueOnError)
	parseFlagSet.SetOutput(os.Stdout)

	parseRootPath := parseFlagSet.String("root", "", "Project root used to resolve Tailwind defaults")
	parseStaticDirPath := parseFlagSet.String("static-dir", "", "Static directory that owns tailwind.input.css")
	parseInputFilePath := parseFlagSet.String("input", "", "Tailwind input CSS file")
	parseOutputFilePath := parseFlagSet.String("output", "", "Tailwind output CSS file")
	parseManifestFilePath := parseFlagSet.String("manifest", "", "Tailwind manifest output file")
	parseBinaryFilePath := parseFlagSet.String("binary", "", "Optional explicit Tailwind CLI binary path")
	cacheDirPath := parseFlagSet.String("cache-dir", "", "Directory that stores downloaded Tailwind CLI binaries")
	parseVersionTag := parseFlagSet.String("version", tailwindDefaultVersion, "Tailwind CLI release tag (for example v4.1.17)")
	shouldSkipManifest := parseFlagSet.Bool("skip-manifest", false, "Skip regenerating generated/tailwind-manifest.html")
	shouldForceDownload := parseFlagSet.Bool("force-download", false, "Force re-downloading the Tailwind CLI binary")
	shouldPrintJSON := parseFlagSet.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFlagSet.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveTailwindConfig(tailwindConfig{
		rootPath:            firstNonEmpty(*parseRootPath, parseL.repoRoot),
		staticDirPath:       *parseStaticDirPath,
		inputFilePath:       *parseInputFilePath,
		outputFilePath:      *parseOutputFilePath,
		manifestFilePath:    *parseManifestFilePath,
		binaryFilePath:      *parseBinaryFilePath,
		cacheDirPath:        *cacheDirPath,
		versionTag:          *parseVersionTag,
		shouldSkipManifest:  *shouldSkipManifest,
		shouldForceDownload: *shouldForceDownload,
		shouldPrintJSON:     *shouldPrintJSON,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeTailwindBuild(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.shouldPrintJSON {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printTailwindSummary(parseSummary)
	return nil
}

// resolveTailwindConfig normalizes CLI inputs and applies repo conventions.
func resolveTailwindConfig(parseConfig tailwindConfig) (tailwindConfig, error) {
	parseResolvedConfig := parseConfig
	parseCwdPath, parseErr := tailwindGetwd()
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind working directory: %w", parseErr)
	}

	parseResolvedConfig.rootPath = strings.TrimSpace(parseResolvedConfig.rootPath)
	if parseResolvedConfig.rootPath == "" {
		parseResolvedConfig.rootPath = parseCwdPath
	}
	parseResolvedConfig.rootPath, parseErr = normalizePath(parseCwdPath, parseResolvedConfig.rootPath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind root path: %w", parseErr)
	}
	parseRootInfo, parseErr := os.Stat(parseResolvedConfig.rootPath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("stat tailwind root path: %w", parseErr)
	}
	if !parseRootInfo.IsDir() {
		return tailwindConfig{}, fmt.Errorf("tailwind root path is not a directory: %s", parseResolvedConfig.rootPath)
	}

	if strings.TrimSpace(parseResolvedConfig.staticDirPath) == "" {
		parseResolvedConfig.staticDirPath = filepath.Join(parseResolvedConfig.rootPath, "examples", "static")
	}
	parseResolvedConfig.staticDirPath, parseErr = normalizePath(parseCwdPath, parseResolvedConfig.staticDirPath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind static dir: %w", parseErr)
	}
	parseStaticInfo, parseErr := os.Stat(parseResolvedConfig.staticDirPath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("stat tailwind static dir: %w", parseErr)
	}
	if !parseStaticInfo.IsDir() {
		return tailwindConfig{}, fmt.Errorf("tailwind static dir is not a directory: %s", parseResolvedConfig.staticDirPath)
	}

	if strings.TrimSpace(parseResolvedConfig.inputFilePath) == "" {
		parseResolvedConfig.inputFilePath = filepath.Join(parseResolvedConfig.staticDirPath, "tailwind.input.css")
	}
	parseResolvedConfig.inputFilePath, parseErr = normalizeExistingPath(parseCwdPath, parseResolvedConfig.inputFilePath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind input file: %w", parseErr)
	}

	if strings.TrimSpace(parseResolvedConfig.outputFilePath) == "" {
		parseResolvedConfig.outputFilePath = filepath.Join(parseResolvedConfig.staticDirPath, "css", "tailwind.css")
	}
	parseResolvedConfig.outputFilePath, parseErr = normalizePath(parseCwdPath, parseResolvedConfig.outputFilePath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind output file: %w", parseErr)
	}

	if strings.TrimSpace(parseResolvedConfig.manifestFilePath) == "" {
		parseResolvedConfig.manifestFilePath = filepath.Join(parseResolvedConfig.staticDirPath, "generated", "tailwind-manifest.html")
	}
	parseResolvedConfig.manifestFilePath, parseErr = normalizePath(parseCwdPath, parseResolvedConfig.manifestFilePath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind manifest file: %w", parseErr)
	}

	if strings.TrimSpace(parseResolvedConfig.cacheDirPath) == "" {
		parseResolvedConfig.cacheDirPath = filepath.Join(parseResolvedConfig.rootPath, "third_party", "tailwindcss", "bin")
	}
	parseResolvedConfig.cacheDirPath, parseErr = normalizePath(parseCwdPath, parseResolvedConfig.cacheDirPath)
	if parseErr != nil {
		return tailwindConfig{}, fmt.Errorf("resolve tailwind cache dir: %w", parseErr)
	}

	parseResolvedConfig.versionTag, parseErr = normalizeTailwindVersionTag(parseResolvedConfig.versionTag)
	if parseErr != nil {
		return tailwindConfig{}, parseErr
	}

	parseResolvedConfig.binaryFilePath = strings.TrimSpace(parseResolvedConfig.binaryFilePath)
	if parseResolvedConfig.binaryFilePath != "" {
		parseResolvedConfig.binaryFilePath, parseErr = normalizeExistingPath(parseCwdPath, parseResolvedConfig.binaryFilePath)
		if parseErr != nil {
			return tailwindConfig{}, fmt.Errorf("resolve tailwind binary file: %w", parseErr)
		}
	}

	return parseResolvedConfig, nil
}

// normalizeTailwindVersionTag validates and normalizes a Tailwind release tag.
func normalizeTailwindVersionTag(parseVersionTag string) (string, error) {
	parseNormalizedTag := strings.TrimSpace(parseVersionTag)
	if parseNormalizedTag == "" {
		parseNormalizedTag = tailwindDefaultVersion
	}
	if !strings.HasPrefix(parseNormalizedTag, "v") {
		parseNormalizedTag = "v" + parseNormalizedTag
	}
	if strings.ContainsAny(parseNormalizedTag, " \t\r\n/\\") {
		return "", fmt.Errorf("invalid tailwind version tag %q", parseVersionTag)
	}
	return parseNormalizedTag, nil
}

// executeTailwindBuild regenerates the manifest, resolves the CLI binary, and writes tailwind.css.
func executeTailwindBuild(parseConfig tailwindConfig) (tailwindSummary, error) {
	parseSummary := tailwindSummary{
		OK:             true,
		RootPath:       parseConfig.rootPath,
		StaticDirPath:  parseConfig.staticDirPath,
		InputFilePath:  parseConfig.inputFilePath,
		OutputFilePath: parseConfig.outputFilePath,
		VersionTag:     parseConfig.versionTag,
	}

	if !parseConfig.shouldSkipManifest {
		parseManifestPath, parseErr := buildTailwindManifestFile(parseConfig)
		if parseErr != nil {
			return tailwindSummary{}, parseErr
		}
		parseSummary.ManifestFilePath = parseManifestPath
	}

	parseBinaryPath, parseErr2 := resolveTailwindBinaryPath(parseConfig)
	if parseErr2 != nil {
		return tailwindSummary{}, parseErr2
	}
	if strings.TrimSpace(parseConfig.binaryFilePath) == "" {
		if parseErr3 := ensureTailwindBinaryFile(parseConfig, parseBinaryPath); parseErr3 != nil {
			return tailwindSummary{}, parseErr3
		}
	}
	parseSummary.BinaryFilePath = parseBinaryPath

	if parseErr4 := os.MkdirAll(filepath.Dir(parseConfig.outputFilePath), 0o755); parseErr4 != nil {
		return tailwindSummary{}, fmt.Errorf("prepare tailwind output directory: %w", parseErr4)
	}

	parseCommandOutput, parseErr2 := tailwindRunCommand(parseBinaryPath, parseConfig.inputFilePath, parseConfig.outputFilePath, parseConfig.rootPath)
	if parseErr2 != nil {
		return tailwindSummary{}, parseErr2
	}
	parseSummary.CommandOutput = parseCommandOutput

	parseOutputInfo, parseErr2 := os.Stat(parseConfig.outputFilePath)
	if parseErr2 != nil {
		return tailwindSummary{}, fmt.Errorf("stat tailwind output file: %w", parseErr2)
	}
	parseSummary.Bytes = parseOutputInfo.Size()
	return parseSummary, nil
}

// resolveTailwindBinaryPath chooses the explicit or cached Tailwind binary path.
func resolveTailwindBinaryPath(parseConfig tailwindConfig) (string, error) {
	if strings.TrimSpace(parseConfig.binaryFilePath) != "" {
		return parseConfig.binaryFilePath, nil
	}
	parseAssetName, parseErr := resolveTailwindAssetName(runtime.GOOS, runtime.GOARCH)
	if parseErr != nil {
		return "", parseErr
	}
	return filepath.Join(parseConfig.cacheDirPath, parseConfig.versionTag, parseAssetName), nil
}

// resolveTailwindAssetName maps GOOS and GOARCH to the official Tailwind release asset name.
func resolveTailwindAssetName(parseGoos string, parseGoarch string) (string, error) {
	switch parseGoos {
	case "windows":
		switch parseGoarch {
		case "amd64":
			return "tailwindcss-windows-x64.exe", nil
		case "arm64":
			return "tailwindcss-windows-arm64.exe", nil
		}
	case "darwin":
		switch parseGoarch {
		case "amd64":
			return "tailwindcss-macos-x64", nil
		case "arm64":
			return "tailwindcss-macos-arm64", nil
		}
	case "linux":
		switch parseGoarch {
		case "amd64":
			return "tailwindcss-linux-x64", nil
		case "arm64":
			return "tailwindcss-linux-arm64", nil
		}
	}
	return "", fmt.Errorf("unsupported tailwind platform: %s/%s", parseGoos, parseGoarch)
}

// ensureTailwindBinaryFile downloads the requested Tailwind binary when it is missing or stale.
func ensureTailwindBinaryFile(parseConfig tailwindConfig, parseBinaryFilePath string) error {
	if !parseConfig.shouldForceDownload {
		if parseBinaryInfo, parseErr := os.Stat(parseBinaryFilePath); parseErr == nil && !parseBinaryInfo.IsDir() {
			return nil
		}
	}
	parseAssetName, parseErr2 := resolveTailwindAssetName(runtime.GOOS, runtime.GOARCH)
	if parseErr2 != nil {
		return parseErr2
	}
	return downloadTailwindBinaryFile(parseConfig.versionTag, parseAssetName, parseBinaryFilePath)
}

// downloadTailwindBinaryFile downloads a Tailwind release binary and writes it to a stable cache path.
func downloadTailwindBinaryFile(parseVersionTag string, parseAssetName string, parseBinaryFilePath string) error {
	parseChecksumURL := fmt.Sprintf("%s/%s/sha256sums.txt", tailwindReleaseDownloadBase, parseVersionTag)
	parseChecksumBytes, parseChecksumErr := tailwindFetchURLBytes(parseChecksumURL)

	parseAssetURL := fmt.Sprintf("%s/%s/%s", tailwindReleaseDownloadBase, parseVersionTag, parseAssetName)
	parseAssetBytes, parseErr := tailwindFetchURLBytes(parseAssetURL)
	if parseErr != nil {
		return fmt.Errorf("download tailwind binary: %w", parseErr)
	}
	// Integrity check. A checksum MISMATCH always fails closed. The two
	// fail-OPEN paths (checksum manifest unreachable, or the manifest lacks an
	// entry for this asset) are kept — some older Tailwind releases ship no
	// sha256sums.txt, and hard-failing would break the dev command for them —
	// but they must never be SILENT: a downloaded executable being installed
	// without verification is exactly what an operator needs to see.
	if parseChecksumErr != nil {
		fmt.Fprintf(os.Stderr, "warning: tailwind checksum manifest unavailable (%v); installing %s WITHOUT integrity verification\n", parseChecksumErr, parseAssetName)
	} else {
		parseChecksumByAsset := parseTailwindChecksums(string(parseChecksumBytes))
		if parseExpectedChecksum, hasChecksum := parseChecksumByAsset[parseAssetName]; hasChecksum {
			if parseErr2 := verifyTailwindChecksum(parseAssetBytes, parseExpectedChecksum); parseErr2 != nil {
				return fmt.Errorf("verify tailwind checksum: %w", parseErr2)
			}
		} else {
			fmt.Fprintf(os.Stderr, "warning: tailwind checksum manifest has no entry for %s; installing WITHOUT integrity verification\n", parseAssetName)
		}
	}

	if parseErr3 := os.MkdirAll(filepath.Dir(parseBinaryFilePath), 0o755); parseErr3 != nil {
		return fmt.Errorf("prepare tailwind cache directory: %w", parseErr3)
	}

	parseTempFile, parseErr := os.CreateTemp(filepath.Dir(parseBinaryFilePath), filepath.Base(parseBinaryFilePath)+".*.tmp")
	if parseErr != nil {
		return fmt.Errorf("create tailwind temp file: %w", parseErr)
	}
	parseTempFilePath := parseTempFile.Name()
	defer func() {
		_ = parseTempFile.Close()
		_ = os.Remove(parseTempFilePath)
	}()

	if _, parseErr4 := parseTempFile.Write(parseAssetBytes); parseErr4 != nil {
		return fmt.Errorf("write tailwind temp file: %w", parseErr4)
	}
	if parseErr5 := parseTempFile.Close(); parseErr5 != nil {
		return fmt.Errorf("close tailwind temp file: %w", parseErr5)
	}
	if parseErr6 := os.Chmod(parseTempFilePath, 0o755); parseErr6 != nil {
		return fmt.Errorf("chmod tailwind binary: %w", parseErr6)
	}
	if parseErr7 := os.Remove(parseBinaryFilePath); parseErr7 != nil && !os.IsNotExist(parseErr7) {
		return fmt.Errorf("replace existing tailwind binary: %w", parseErr7)
	}
	if parseErr8 := os.Rename(parseTempFilePath, parseBinaryFilePath); parseErr8 != nil {
		return fmt.Errorf("finalize tailwind binary: %w", parseErr8)
	}
	return nil
}

// fetchTailwindURLBytes downloads one URL and returns its response bytes.
func fetchTailwindURLBytes(parseDownloadURL string) ([]byte, error) {
	parseRequest, parseErr := http.NewRequest(http.MethodGet, parseDownloadURL, nil)
	if parseErr != nil {
		return nil, fmt.Errorf("prepare tailwind download request: %w", parseErr)
	}
	parseResponse, parseErr := tailwindHTTPClient.Do(parseRequest)
	if parseErr != nil {
		return nil, fmt.Errorf("download %s: %w", parseDownloadURL, parseErr)
	}
	defer parseResponse.Body.Close()

	if parseResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %s", parseDownloadURL, parseResponse.Status)
	}
	parseBodyBytes, parseErr := io.ReadAll(parseResponse.Body)
	if parseErr != nil {
		return nil, fmt.Errorf("read %s: %w", parseDownloadURL, parseErr)
	}
	if len(parseBodyBytes) == 0 {
		return nil, fmt.Errorf("download %s: empty response", parseDownloadURL)
	}
	return parseBodyBytes, nil
}

// parseTailwindChecksums parses sha256sums.txt into asset-name keyed hashes.
func parseTailwindChecksums(parseChecksumText string) map[string]string {
	parseChecksumByAsset := map[string]string{}
	for parseLineText := range strings.SplitSeq(parseChecksumText, "\n") {
		parseTrimmedLine := strings.TrimSpace(parseLineText)
		if parseTrimmedLine == "" {
			continue
		}
		parseFields := strings.Fields(parseTrimmedLine)
		if len(parseFields) < 2 {
			continue
		}
		parseAssetName := strings.TrimPrefix(parseFields[1], "*")
		parseChecksumByAsset[parseAssetName] = strings.ToLower(parseFields[0])
	}
	return parseChecksumByAsset
}

// verifyTailwindChecksum validates downloaded bytes against an expected SHA-256 hash.
func verifyTailwindChecksum(parseAssetBytes []byte, parseExpectedChecksum string) error {
	parseComputedChecksum := sha256.Sum256(parseAssetBytes)
	parseComputedChecksumText := strings.ToLower(hex.EncodeToString(parseComputedChecksum[:]))
	if parseComputedChecksumText != strings.ToLower(strings.TrimSpace(parseExpectedChecksum)) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", parseExpectedChecksum, parseComputedChecksumText)
	}
	return nil
}

// executeTailwindCommand invokes the Tailwind CLI with the repository input and output files.
func executeTailwindCommand(parseBinaryFilePath string, parseInputFilePath string, parseOutputFilePath string, parseWorkDirPath string) (string, error) {
	parseCommand := exec.Command(parseBinaryFilePath, "-i", parseInputFilePath, "-o", parseOutputFilePath)
	parseCommand.Dir = parseWorkDirPath
	parseCommandOutput, parseErr := parseCommand.CombinedOutput()
	parseTrimmedOutput := strings.TrimSpace(string(parseCommandOutput))
	if parseErr != nil {
		if parseTrimmedOutput == "" {
			return "", fmt.Errorf("run tailwind cli: %w", parseErr)
		}
		return parseTrimmedOutput, fmt.Errorf("run tailwind cli: %s", parseTrimmedOutput)
	}
	return parseTrimmedOutput, nil
}

// buildTailwindManifestFile renders generated/tailwind-manifest.html from scanned source literals.
func buildTailwindManifestFile(parseConfig tailwindConfig) (string, error) {
	parseManifestRows, parseErr := collectTailwindManifestRows(parseConfig.staticDirPath)
	if parseErr != nil {
		return "", parseErr
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parseConfig.manifestFilePath), 0o755); parseErr2 != nil {
		return "", fmt.Errorf("prepare tailwind manifest directory: %w", parseErr2)
	}

	parseManifestParts := []string{
		"<!DOCTYPE html>",
		"<html>",
		"<body>",
	}
	parseManifestParts = append(parseManifestParts, parseManifestRows...)
	parseManifestParts = append(parseManifestParts, "</body>", "</html>", "")
	parseManifestText := strings.Join(parseManifestParts, "\n")

	if parseErr3 := os.WriteFile(parseConfig.manifestFilePath, []byte(parseManifestText), 0o644); parseErr3 != nil {
		return "", fmt.Errorf("write tailwind manifest: %w", parseErr3)
	}
	return parseConfig.manifestFilePath, nil
}

// collectTailwindManifestRows scans examples sources and produces deterministic manifest rows.
func collectTailwindManifestRows(parseStaticDirPath string) ([]string, error) {
	parseExamplesDirPath := filepath.Dir(parseStaticDirPath)
	parseManifestRecords := make([]tailwindManifestRecord, 0, 256)
	parseWalkErr := filepath.WalkDir(parseExamplesDirPath, func(parseWalkFilePath string, parseWalkDirEntry fs.DirEntry, parseWalkError error) error {
		if parseWalkError != nil {
			return parseWalkError
		}
		if parseWalkFilePath == parseExamplesDirPath {
			return nil
		}
		if parseWalkDirEntry.IsDir() {
			if shouldSkipTailwindDirName(parseWalkDirEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldIncludeTailwindExtension(filepath.Ext(parseWalkDirEntry.Name())) {
			return nil
		}

		parseSourceBytes, parseErr := os.ReadFile(parseWalkFilePath)
		if parseErr != nil {
			return fmt.Errorf("read tailwind source file %s: %w", parseWalkFilePath, parseErr)
		}
		parseLiteralValues := extractTailwindLiteralValues(string(parseSourceBytes))
		if len(parseLiteralValues) == 0 {
			return nil
		}
		parseRelativeFilePath, parseErr := filepath.Rel(parseExamplesDirPath, parseWalkFilePath)
		if parseErr != nil {
			return fmt.Errorf("resolve tailwind relative path for %s: %w", parseWalkFilePath, parseErr)
		}
		parseManifestRecords = append(parseManifestRecords, tailwindManifestRecord{
			relativeFilePath: filepath.ToSlash(parseRelativeFilePath),
			literalValues:    parseLiteralValues,
		})
		return nil
	})
	if parseWalkErr != nil {
		return nil, fmt.Errorf("walk tailwind source files: %w", parseWalkErr)
	}

	sort.Slice(parseManifestRecords, func(parseLeftIndex int, parseRightIndex int) bool {
		return parseManifestRecords[parseLeftIndex].relativeFilePath < parseManifestRecords[parseRightIndex].relativeFilePath
	})

	parseManifestRows := make([]string, 0, len(parseManifestRecords)*3)
	for _, parseManifestRecord := range parseManifestRecords {
		parseManifestRows = append(parseManifestRows, "<!-- "+escapeTailwindHTML(parseManifestRecord.relativeFilePath)+" -->")
		for _, parseLiteralValue := range parseManifestRecord.literalValues {
			parseManifestRows = append(parseManifestRows, `<div class="`+escapeTailwindHTML(parseLiteralValue)+`"></div>`)
		}
	}
	return parseManifestRows, nil
}

// shouldSkipTailwindDirName reports whether a directory is excluded from manifest scanning.
func shouldSkipTailwindDirName(parseDirName string) bool {
	switch strings.ToLower(strings.TrimSpace(parseDirName)) {
	case ".git", ".npm-cache", "bin", "css", "generated", "images", "modules", "node_modules":
		return true
	default:
		return false
	}
}

// shouldIncludeTailwindExtension reports whether a file extension is scanned for class literals.
func shouldIncludeTailwindExtension(parseFileExtension string) bool {
	switch strings.ToLower(strings.TrimSpace(parseFileExtension)) {
	case ".go", ".html", ".js", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

// extractTailwindLiteralValues returns normalized string-literal candidates for Tailwind scanning.
func extractTailwindLiteralValues(parseSourceText string) []string {
	parseLiteralValues := make([]string, 0, 64)
	parseSourceIndex := 0
	for parseSourceIndex < len(parseSourceText) {
		parseSourceByte := parseSourceText[parseSourceIndex]
		if parseSourceByte != '"' && parseSourceByte != '`' {
			parseSourceIndex++
			continue
		}
		parseLiteralValue, parseNextIndex, hasClosedQuote := parseTailwindQuotedLiteral(parseSourceText, parseSourceIndex, parseSourceByte)
		if !hasClosedQuote {
			parseSourceIndex++
			continue
		}
		parseSourceIndex = parseNextIndex
		parseNormalizedLiteral := normalizeTailwindLiteralValue(parseLiteralValue)
		if parseNormalizedLiteral != "" {
			parseLiteralValues = append(parseLiteralValues, parseNormalizedLiteral)
		}
	}
	return parseLiteralValues
}

// parseTailwindQuotedLiteral parses one quoted or raw string literal from source text.
func parseTailwindQuotedLiteral(parseSourceText string, parseStartIndex int, parseQuoteByte byte) (string, int, bool) {
	var parseLiteralBuilder strings.Builder
	parseIndex := parseStartIndex + 1
	shouldEscapeNext := false
	for parseIndex < len(parseSourceText) {
		parseCurrentByte := parseSourceText[parseIndex]
		if parseQuoteByte == '"' {
			if shouldEscapeNext {
				parseLiteralBuilder.WriteByte(parseCurrentByte)
				shouldEscapeNext = false
				parseIndex++
				continue
			}
			if parseCurrentByte == '\\' {
				parseLiteralBuilder.WriteByte(parseCurrentByte)
				shouldEscapeNext = true
				parseIndex++
				continue
			}
		}
		if parseCurrentByte == parseQuoteByte {
			return parseLiteralBuilder.String(), parseIndex + 1, true
		}
		parseLiteralBuilder.WriteByte(parseCurrentByte)
		parseIndex++
	}
	return "", len(parseSourceText), false
}

// normalizeTailwindLiteralValue mirrors the Tailwind manifest literal normalization rules.
func normalizeTailwindLiteralValue(parseLiteralValue string) string {
	parseNormalizedValue := parseLiteralValue
	parseNormalizedValue = strings.ReplaceAll(parseNormalizedValue, `\"`, `"`)
	parseNormalizedValue = strings.ReplaceAll(parseNormalizedValue, `\\`, `\`)
	parseNormalizedValue = strings.ReplaceAll(parseNormalizedValue, "\\`", "`")
	parseNormalizedValue = strings.ReplaceAll(parseNormalizedValue, `\n`, " ")
	parseNormalizedValue = strings.ReplaceAll(parseNormalizedValue, `\r`, " ")
	parseNormalizedValue = strings.ReplaceAll(parseNormalizedValue, `\t`, " ")
	return strings.Join(strings.Fields(parseNormalizedValue), " ")
}

// escapeTailwindHTML escapes HTML-special characters for manifest-safe output.
func escapeTailwindHTML(parseSourceText string) string {
	parseEscapedValue := strings.ReplaceAll(parseSourceText, "&", "&amp;")
	parseEscapedValue = strings.ReplaceAll(parseEscapedValue, "<", "&lt;")
	parseEscapedValue = strings.ReplaceAll(parseEscapedValue, ">", "&gt;")
	parseEscapedValue = strings.ReplaceAll(parseEscapedValue, `"`, "&quot;")
	return parseEscapedValue
}

// printTailwindSummary writes a concise human-readable tailwind build summary.
func printTailwindSummary(parseSummary tailwindSummary) {
	fmt.Println("GWC tailwind")
	fmt.Printf("  root:         %s\n", parseSummary.RootPath)
	fmt.Printf("  static dir:   %s\n", parseSummary.StaticDirPath)
	fmt.Printf("  input:        %s\n", parseSummary.InputFilePath)
	fmt.Printf("  output:       %s\n", parseSummary.OutputFilePath)
	if strings.TrimSpace(parseSummary.ManifestFilePath) != "" {
		fmt.Printf("  manifest:     %s\n", parseSummary.ManifestFilePath)
	} else {
		fmt.Println("  manifest:     <skipped>")
	}
	fmt.Printf("  binary:       %s\n", parseSummary.BinaryFilePath)
	fmt.Printf("  version:      %s\n", parseSummary.VersionTag)
	fmt.Printf("  bytes:        %d\n", parseSummary.Bytes)
	if strings.TrimSpace(parseSummary.CommandOutput) != "" {
		// ensure CLI notes remain visible when Tailwind emits upgrade hints or warnings.
		fmt.Printf("  cli output:   %s\n", parseSummary.CommandOutput)
	}
}
