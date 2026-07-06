package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var runDeployCommand = func(l launcher, args []string) error {
	return l.runDeploy(args)
}

type deployConfig struct {
	rootPath     string
	manifestPath string
	adapterName  string
	targetPath   string
	json         bool
}

type deploySummary struct {
	OK          bool     `json:"ok"`
	Root        string   `json:"root"`
	Manifest    string   `json:"manifest"`
	Adapter     string   `json:"adapter"`
	Target      string   `json:"target"`
	ArtifactDir string   `json:"artifactDir"`
	Files       []string `json:"files,omitempty"`
	Checks      []string `json:"checks,omitempty"`
}

type deployAdapter interface {
	applyDeployPackage(config deployConfig, manifest releaseManifestSnapshot, artifactFiles []string) (deploySummary, error)
}

type deployFilesystemAdapter struct{}

type deployZipAdapter struct{}

// runDeploy executes deployment packaging through an explicit adapter contract.
func (parseL launcher) runDeploy(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("deploy", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root for default manifest resolution")
	parseManifest := parseFlags.String("manifest", "", "Path to a release manifest; defaults to <root>/bin/wasm-release/wasm-release-manifest.json")
	parseAdapter := parseFlags.String("adapter", "filesystem", "Deploy adapter: filesystem or zip")
	parseTarget := parseFlags.String("target", "", "Deploy target path; directory for filesystem adapter, .zip path for zip adapter")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON summary")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr := parseDeployConfig(deployConfig{
		rootPath:     *parseRoot,
		manifestPath: *parseManifest,
		adapterName:  *parseAdapter,
		targetPath:   *parseTarget,
		json:         *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	applySummary, applyErr := applyDeployPackage(parseConfig)
	if applyErr != nil {
		return applyErr
	}
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(applySummary)
	}
	renderDeploySummary(applySummary)
	return nil
}

// parseDeployConfig resolves and validates deployment command configuration.
func parseDeployConfig(parseConfig deployConfig) (deployConfig, error) {
	parseRootPath, parseErr := parseLifecycleRootPath(parseConfig.rootPath)
	if parseErr != nil {
		return deployConfig{}, parseErr
	}
	parseManifestPath := strings.TrimSpace(parseConfig.manifestPath)
	if parseManifestPath == "" {
		parseManifestPath = filepath.Join(parseRootPath, "bin", "wasm-release", "wasm-release-manifest.json")
	}
	parseManifestPath, parseErr = normalizeExistingPath(parseRootPath, parseManifestPath)
	if parseErr != nil {
		return deployConfig{}, fmt.Errorf("resolve deploy manifest path: %w", parseErr)
	}
	parseAdapterName := strings.ToLower(strings.TrimSpace(parseConfig.adapterName))
	if parseAdapterName == "" {
		parseAdapterName = "filesystem"
	}
	if parseAdapterName != "filesystem" && parseAdapterName != "zip" {
		return deployConfig{}, fmt.Errorf("unknown deploy adapter %q", parseConfig.adapterName)
	}

	parseTargetPath := strings.TrimSpace(parseConfig.targetPath)
	if parseTargetPath == "" {
		if parseAdapterName == "zip" {
			parseTargetPath = filepath.Join(filepath.Dir(parseManifestPath), "deploy-package.zip")
		} else {
			parseTargetPath = filepath.Join(filepath.Dir(parseManifestPath), "deploy")
		}
	}
	parseTargetPath, parseErr = normalizePath(parseRootPath, parseTargetPath)
	if parseErr != nil {
		return deployConfig{}, fmt.Errorf("resolve deploy target path: %w", parseErr)
	}
	if parseAdapterName == "zip" && !strings.EqualFold(filepath.Ext(parseTargetPath), ".zip") {
		return deployConfig{}, errors.New("zip deploy adapter requires a .zip target path")
	}

	return deployConfig{
		rootPath:     parseRootPath,
		manifestPath: parseManifestPath,
		adapterName:  parseAdapterName,
		targetPath:   parseTargetPath,
		json:         parseConfig.json,
	}, nil
}

// applyDeployPackage validates release artifacts and delegates packaging to the selected adapter.
func applyDeployPackage(applyConfig deployConfig) (deploySummary, error) {
	applyManifest, applyManifestErr := releaseReadManifestSnapshot(applyConfig.manifestPath)
	if applyManifestErr != nil {
		return deploySummary{}, applyManifestErr
	}
	applyArtifactDir := filepath.Dir(applyConfig.manifestPath)
	applyArtifactFiles, applyChecks, applyValidateErr := applyDeployArtifacts(applyArtifactDir, applyManifest, applyConfig.manifestPath)
	if applyValidateErr != nil {
		return deploySummary{}, applyValidateErr
	}
	applyAdapter, applyAdapterErr := buildDeployAdapter(applyConfig.adapterName)
	if applyAdapterErr != nil {
		return deploySummary{}, applyAdapterErr
	}
	applySummary, applyPackageErr := applyAdapter.applyDeployPackage(applyConfig, applyManifest, applyArtifactFiles)
	if applyPackageErr != nil {
		return deploySummary{}, applyPackageErr
	}
	applySummary.OK = true
	applySummary.Root = applyConfig.rootPath
	applySummary.Manifest = applyConfig.manifestPath
	applySummary.Adapter = applyConfig.adapterName
	applySummary.ArtifactDir = applyArtifactDir
	applySummary.Checks = append(applyChecks, applySummary.Checks...)
	return applySummary, nil
}

// applyDeployArtifacts validates manifest-listed artifacts and returns relative files to package.
func applyDeployArtifacts(applyArtifactDir string, applyManifest releaseManifestSnapshot, applyManifestPath string) ([]string, []string, error) {
	applyFiles := []string{filepath.Base(applyManifestPath)}
	applyChecks := []string{"manifest parses as a js/wasm release record"}
	for _, applyRecord := range applyManifest.Artifacts {
		// Contain the manifest-supplied path inside the artifact dir. Without
		// this, a record path like "../../etc/passwd" (or an absolute path)
		// would let a crafted manifest read arbitrary files into the deploy
		// package and, via the filesystem adapter, write them outside the
		// target dir (path traversal / zip-slip).
		if applyErr := ensureDeployPathContained(applyArtifactDir, applyRecord.Path); applyErr != nil {
			return nil, nil, applyErr
		}
		applyPath := filepath.Join(applyArtifactDir, filepath.FromSlash(applyRecord.Path))
		applyInfo, applyErr := os.Stat(applyPath)
		if applyErr != nil {
			return nil, nil, fmt.Errorf("validate deploy artifact %s: %w", applyRecord.Path, applyErr)
		}
		if applyInfo.IsDir() {
			return nil, nil, fmt.Errorf("validate deploy artifact %s: path resolves to a directory", applyRecord.Path)
		}
		applyFiles = append(applyFiles, filepath.ToSlash(applyRecord.Path))
		applyChecks = append(applyChecks, fmt.Sprintf("artifact %s exists", applyRecord.Path))
	}
	sort.Strings(applyFiles)
	return applyFiles, applyChecks, nil
}

// ensureDeployPathContained rejects a manifest artifact path that is absolute
// or escapes the artifact directory once resolved, so a crafted manifest cannot
// pull files from outside the release tree into the deploy package.
func ensureDeployPathContained(applyArtifactDir string, applyRecordPath string) error {
	applyRelPath := filepath.FromSlash(strings.TrimSpace(applyRecordPath))
	if applyRelPath == "" {
		return errors.New("validate deploy artifact: empty artifact path")
	}
	if filepath.IsAbs(applyRelPath) {
		return fmt.Errorf("validate deploy artifact %s: absolute artifact paths are not allowed", applyRecordPath)
	}
	applyResolved := filepath.Clean(filepath.Join(applyArtifactDir, applyRelPath))
	applyBase := filepath.Clean(applyArtifactDir)
	applyRel, applyErr := filepath.Rel(applyBase, applyResolved)
	if applyErr != nil {
		return fmt.Errorf("validate deploy artifact %s: %w", applyRecordPath, applyErr)
	}
	if applyRel == ".." || strings.HasPrefix(applyRel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("validate deploy artifact %s: path escapes the artifact directory", applyRecordPath)
	}
	return nil
}

// buildDeployAdapter selects a deployment adapter implementation by name.
func buildDeployAdapter(buildName string) (deployAdapter, error) {
	switch strings.ToLower(strings.TrimSpace(buildName)) {
	case "filesystem":
		return deployFilesystemAdapter{}, nil
	case "zip":
		return deployZipAdapter{}, nil
	default:
		return nil, fmt.Errorf("unknown deploy adapter %q", buildName)
	}
}

// applyDeployPackage copies artifacts into a deployment directory.
func (deployFilesystemAdapter) applyDeployPackage(parseConfig deployConfig, parseManifest releaseManifestSnapshot, parseArtifactFiles []string) (deploySummary, error) {
	applyTargetDir := parseConfig.targetPath
	if applyErr := os.MkdirAll(applyTargetDir, 0755); applyErr != nil {
		return deploySummary{}, fmt.Errorf("create deploy target directory: %w", applyErr)
	}
	applySourceDir := filepath.Dir(parseConfig.manifestPath)
	applyFiles := []string{}
	for _, applyFile := range parseArtifactFiles {
		applySourcePath := filepath.Join(applySourceDir, filepath.FromSlash(applyFile))
		applyTargetPath := filepath.Join(applyTargetDir, filepath.FromSlash(applyFile))
		if applyErr := os.MkdirAll(filepath.Dir(applyTargetPath), 0755); applyErr != nil {
			return deploySummary{}, applyErr
		}
		applyCopyErr := applyDeployFile(applySourcePath, applyTargetPath)
		if applyCopyErr != nil {
			return deploySummary{}, applyCopyErr
		}
		applyFiles = append(applyFiles, filepath.ToSlash(applyTargetPath))
	}
	sort.Strings(applyFiles)
	return deploySummary{
		Target: parseConfig.targetPath,
		Files:  applyFiles,
		Checks: []string{"filesystem deployment package copied"},
	}, nil
}

// applyDeployPackage archives artifacts into a deployment zip file.
func (deployZipAdapter) applyDeployPackage(parseConfig deployConfig, parseManifest releaseManifestSnapshot, parseArtifactFiles []string) (deploySummary, error) {
	applySourceDir := filepath.Dir(parseConfig.manifestPath)
	if applyErr := os.MkdirAll(filepath.Dir(parseConfig.targetPath), 0755); applyErr != nil {
		return deploySummary{}, applyErr
	}
	applyFile, applyErr := os.Create(parseConfig.targetPath)
	if applyErr != nil {
		return deploySummary{}, fmt.Errorf("create deploy zip: %w", applyErr)
	}
	defer applyFile.Close()
	applyZip := zip.NewWriter(applyFile)
	for _, applyEntry := range parseArtifactFiles {
		applySourcePath := filepath.Join(applySourceDir, filepath.FromSlash(applyEntry))
		applyBytes, applyReadErr := os.ReadFile(applySourcePath)
		if applyReadErr != nil {
			applyZip.Close()
			return deploySummary{}, fmt.Errorf("read deploy artifact for zip: %w", applyReadErr)
		}
		applyWriter, applyCreateErr := applyZip.Create(filepath.ToSlash(applyEntry))
		if applyCreateErr != nil {
			applyZip.Close()
			return deploySummary{}, fmt.Errorf("create deploy zip entry: %w", applyCreateErr)
		}
		if _, applyWriteErr := applyWriter.Write(applyBytes); applyWriteErr != nil {
			applyZip.Close()
			return deploySummary{}, fmt.Errorf("write deploy zip entry: %w", applyWriteErr)
		}
	}
	if applyCloseErr := applyZip.Close(); applyCloseErr != nil {
		return deploySummary{}, fmt.Errorf("finalize deploy zip: %w", applyCloseErr)
	}
	return deploySummary{
		Target: parseConfig.targetPath,
		Files:  []string{parseConfig.targetPath},
		Checks: []string{"zip deployment package created"},
	}, nil
}

// applyDeployFile copies one file path while preserving content exactly.
func applyDeployFile(parseSourcePath string, parseTargetPath string) error {
	applySource, applyOpenErr := os.Open(parseSourcePath)
	if applyOpenErr != nil {
		return fmt.Errorf("open deploy source file: %w", applyOpenErr)
	}
	defer applySource.Close()
	applyTarget, applyCreateErr := os.Create(parseTargetPath)
	if applyCreateErr != nil {
		return fmt.Errorf("create deploy target file: %w", applyCreateErr)
	}
	defer applyTarget.Close()
	if _, applyCopyErr := io.Copy(applyTarget, applySource); applyCopyErr != nil {
		return fmt.Errorf("copy deploy file: %w", applyCopyErr)
	}
	return nil
}

// renderDeploySummary prints a human-readable deployment packaging summary.
func renderDeploySummary(renderSummary deploySummary) {
	fmt.Println("GWC deploy")
	fmt.Printf("  root:         %s\n", renderSummary.Root)
	fmt.Printf("  manifest:     %s\n", renderSummary.Manifest)
	fmt.Printf("  adapter:      %s\n", renderSummary.Adapter)
	fmt.Printf("  target:       %s\n", renderSummary.Target)
	fmt.Printf("  files:        %d\n", len(renderSummary.Files))
	if len(renderSummary.Checks) > 0 {
		fmt.Println("  checks:")
		for _, renderCheck := range renderSummary.Checks {
			fmt.Printf("    - %s\n", renderCheck)
		}
	}
}
