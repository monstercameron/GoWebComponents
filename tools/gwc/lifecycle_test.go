package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunInitWritesMetadataAndMatrix verifies init command output artifacts.
func TestRunInitWritesMetadataAndMatrix(t *testing.T) {
	parseRoot := t.TempDir()
	parseModule := "module example.com/initfixture\n\ngo 1.25\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	parseMain := "package main\n\nfunc main() {}\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); writeErr != nil {
		t.Fatalf("write main.go: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html>"), 0644); writeErr != nil {
		t.Fatalf("write index.html: %v", writeErr)
	}
	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		t.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if runErr := (launcher{}).runInit([]string{"-root", parseRoot, "-skip-runtime-assets", "-json"}); runErr != nil {
		t.Fatalf("run init: %v", runErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		t.Fatalf("read init output: %v", parseOutputErr)
	}
	var parseSummary lifecycleInitSummary
	if decodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); decodeErr != nil {
		t.Fatalf("decode init summary: %v\n%s", decodeErr, parseOutput)
	}
	if !parseSummary.OK {
		t.Fatalf("expected successful init summary, got %#v", parseSummary)
	}
	if !fileExists(parseSummary.MetadataPath) {
		t.Fatalf("expected metadata path to exist: %s", parseSummary.MetadataPath)
	}
	if !fileExists(parseSummary.FeatureMatrixPath) {
		t.Fatalf("expected feature matrix path to exist: %s", parseSummary.FeatureMatrixPath)
	}
	parseMetadata, parseFound, parseMetadataErr := loadScaffoldMetadata(parseRoot)
	if parseMetadataErr != nil {
		t.Fatalf("load scaffold metadata: %v", parseMetadataErr)
	}
	if !parseFound {
		t.Fatalf("expected metadata to exist after init")
	}
	if parseMetadata.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", currentScaffoldMetadataSchemaVersion, parseMetadata.SchemaVersion)
	}
	if parseMetadata.Tooling.AppPath != "main.go" || parseMetadata.Tooling.HTMLPath != "index.html" {
		t.Fatalf("unexpected tooling paths: %#v", parseMetadata.Tooling)
	}
}

// TestRunUpgradeBackfillsMetadataDefaults verifies upgrade command normalization behavior.
func TestRunUpgradeBackfillsMetadataDefaults(t *testing.T) {
	parseRoot := t.TempDir()
	parseModule := "module example.com/upgradefixture\n\ngo 1.25\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	parseMain := "package main\n\nfunc main() {}\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); writeErr != nil {
		t.Fatalf("write main.go: %v", writeErr)
	}
	parseMetadata := scaffoldMetadata{
		SchemaVersion: 0,
		ProjectName:   "upgradefixture",
		Tooling: scaffoldToolingMetadata{
			AppPath: "main.go",
		},
	}
	parseMetadataBytes, parseEncodeErr := json.MarshalIndent(parseMetadata, "", "  ")
	if parseEncodeErr != nil {
		t.Fatalf("encode metadata fixture: %v", parseEncodeErr)
	}
	parseMetadataBytes = append(parseMetadataBytes, '\n')
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), parseMetadataBytes, 0644); writeErr != nil {
		t.Fatalf("write gwc-start.json: %v", writeErr)
	}

	if runErr := (launcher{}).runUpgrade([]string{"-root", parseRoot, "-skip-runtime-assets"}); runErr != nil {
		t.Fatalf("run upgrade: %v", runErr)
	}
	parseUpgraded, parseFound, parseErr := loadScaffoldMetadata(parseRoot)
	if parseErr != nil {
		t.Fatalf("load upgraded metadata: %v", parseErr)
	}
	if !parseFound {
		t.Fatalf("expected upgraded metadata to exist")
	}
	if parseUpgraded.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", currentScaffoldMetadataSchemaVersion, parseUpgraded.SchemaVersion)
	}
	if strings.TrimSpace(parseUpgraded.Ownership.ProjectOwnership) == "" || strings.TrimSpace(parseUpgraded.Ownership.FrameworkSourceMode) == "" {
		t.Fatalf("expected ownership defaults to be filled, got %#v", parseUpgraded.Ownership)
	}
	if strings.TrimSpace(parseUpgraded.Tooling.WASMPath) == "" || strings.TrimSpace(parseUpgraded.Tooling.DevHost) == "" || strings.TrimSpace(parseUpgraded.Tooling.DevPort) == "" {
		t.Fatalf("expected tooling defaults to be filled, got %#v", parseUpgraded.Tooling)
	}
}

// TestRunMigrateWritesCompatibilityReport verifies migrate report generation for legacy APIs.
func TestRunMigrateWritesCompatibilityReport(t *testing.T) {
	parseRoot := t.TempDir()
	parseModule := "module example.com/migratefixture\n\ngo 1.25\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	parseMain := `package main

func main() {
	// GoRegisterRoute("/home", nil)
	// GoGetRoute()
}
`
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); writeErr != nil {
		t.Fatalf("write main.go: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html>"), 0644); writeErr != nil {
		t.Fatalf("write index.html: %v", writeErr)
	}
	if runErr := (launcher{}).runInit([]string{"-root", parseRoot, "-skip-runtime-assets", "-force"}); runErr != nil {
		t.Fatalf("seed init for migrate: %v", runErr)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		t.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if runErr := (launcher{}).runMigrate([]string{"-root", parseRoot, "-skip-runtime-assets", "-json"}); runErr != nil {
		t.Fatalf("run migrate: %v", runErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		t.Fatalf("read migrate output: %v", parseOutputErr)
	}
	var parseSummary lifecycleMigrateSummary
	if decodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); decodeErr != nil {
		t.Fatalf("decode migrate summary: %v\n%s", decodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.FindingCount < 2 {
		t.Fatalf("expected migrate findings, got %#v", parseSummary)
	}
	if !fileExists(parseSummary.ReportPath) {
		t.Fatalf("expected migrate report to exist: %s", parseSummary.ReportPath)
	}
}
