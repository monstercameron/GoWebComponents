package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunInitWritesMetadataAndMatrix verifies init command output artifacts.
func TestRunInitWritesMetadataAndMatrix(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseModule := "module example.com/initfixture\n\ngo 1.25\n"
	if parseWriteErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}
	parseMain := "package main\n\nfunc main() {}\n"
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr2)
	}
	if parseWriteErr3 := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html>"), 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseWriteErr3)
	}
	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		parseT.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if parseRunErr := (launcher{}).runInit([]string{"-root", parseRoot, "-skip-runtime-assets", "-json"}); parseRunErr != nil {
		parseT.Fatalf("run init: %v", parseRunErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		parseT.Fatalf("read init output: %v", parseOutputErr)
	}
	var parseSummary lifecycleInitSummary
	if parseDecodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); parseDecodeErr != nil {
		parseT.Fatalf("decode init summary: %v\n%s", parseDecodeErr, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful init summary, got %#v", parseSummary)
	}
	if !fileExists(parseSummary.MetadataPath) {
		parseT.Fatalf("expected metadata path to exist: %s", parseSummary.MetadataPath)
	}
	if !fileExists(parseSummary.FeatureMatrixPath) {
		parseT.Fatalf("expected feature matrix path to exist: %s", parseSummary.FeatureMatrixPath)
	}
	parseMetadata, parseFound, parseMetadataErr := loadScaffoldMetadata(parseRoot)
	if parseMetadataErr != nil {
		parseT.Fatalf("load scaffold metadata: %v", parseMetadataErr)
	}
	if !parseFound {
		parseT.Fatalf("expected metadata to exist after init")
	}
	if parseMetadata.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		parseT.Fatalf("expected schema version %d, got %d", currentScaffoldMetadataSchemaVersion, parseMetadata.SchemaVersion)
	}
	if parseMetadata.Tooling.AppPath != "main.go" || parseMetadata.Tooling.HTMLPath != "index.html" {
		parseT.Fatalf("unexpected tooling paths: %#v", parseMetadata.Tooling)
	}
}

// TestRunUpgradeBackfillsMetadataDefaults verifies upgrade command normalization behavior.
func TestRunUpgradeBackfillsMetadataDefaults(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseModule := "module example.com/upgradefixture\n\ngo 1.25\n"
	if parseWriteErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}
	parseMain := "package main\n\nfunc main() {}\n"
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr2)
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
		parseT.Fatalf("encode metadata fixture: %v", parseEncodeErr)
	}
	parseMetadataBytes = append(parseMetadataBytes, '\n')
	if parseWriteErr3 := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), parseMetadataBytes, 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseWriteErr3)
	}

	if parseRunErr := (launcher{}).runUpgrade([]string{"-root", parseRoot, "-skip-runtime-assets"}); parseRunErr != nil {
		parseT.Fatalf("run upgrade: %v", parseRunErr)
	}
	parseUpgraded, parseFound, parseErr := loadScaffoldMetadata(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("load upgraded metadata: %v", parseErr)
	}
	if !parseFound {
		parseT.Fatalf("expected upgraded metadata to exist")
	}
	if parseUpgraded.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		parseT.Fatalf("expected schema version %d, got %d", currentScaffoldMetadataSchemaVersion, parseUpgraded.SchemaVersion)
	}
	if strings.TrimSpace(parseUpgraded.Ownership.ProjectOwnership) == "" || strings.TrimSpace(parseUpgraded.Ownership.FrameworkSourceMode) == "" {
		parseT.Fatalf("expected ownership defaults to be filled, got %#v", parseUpgraded.Ownership)
	}
	if strings.TrimSpace(parseUpgraded.Tooling.WASMPath) == "" || strings.TrimSpace(parseUpgraded.Tooling.DevHost) == "" || strings.TrimSpace(parseUpgraded.Tooling.DevPort) == "" {
		parseT.Fatalf("expected tooling defaults to be filled, got %#v", parseUpgraded.Tooling)
	}
}

// TestRunMigrateWritesCompatibilityReport verifies migrate report generation for legacy APIs.
func TestRunMigrateWritesCompatibilityReport(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseModule := "module example.com/migratefixture\n\ngo 1.25\n"
	if parseWriteErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}
	parseMain := `package main

func main() {
	// GoRegisterRoute("/home", nil)
	// GoGetRoute()
}
`
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr2)
	}
	if parseWriteErr3 := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html>"), 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseWriteErr3)
	}
	if parseRunErr := (launcher{}).runInit([]string{"-root", parseRoot, "-skip-runtime-assets", "-force"}); parseRunErr != nil {
		parseT.Fatalf("seed init for migrate: %v", parseRunErr)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		parseT.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if parseRunErr2 := (launcher{}).runMigrate([]string{"-root", parseRoot, "-skip-runtime-assets", "-json"}); parseRunErr2 != nil {
		parseT.Fatalf("run migrate: %v", parseRunErr2)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		parseT.Fatalf("read migrate output: %v", parseOutputErr)
	}
	var parseSummary lifecycleMigrateSummary
	if parseDecodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); parseDecodeErr != nil {
		parseT.Fatalf("decode migrate summary: %v\n%s", parseDecodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.FindingCount < 2 {
		parseT.Fatalf("expected migrate findings, got %#v", parseSummary)
	}
	if !fileExists(parseSummary.ReportPath) {
		parseT.Fatalf("expected migrate report to exist: %s", parseSummary.ReportPath)
	}
}

// TestRunMigrateApplyRewritesLegacyRouterCalls verifies parser-backed migration rewrites.
func TestRunMigrateApplyRewritesLegacyRouterCalls(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseModule := "module example.com/migrateapplyfixture\n\ngo 1.25\n"
	if parseWriteErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}
	parseMain := `package main

func main() {
	parseRouter.GoRegisterRoute("/home", nil)
	_ = parseRouter.GoGetRoute()
	_ = "parseRouter.GoRegisterRoute(\"/literal\", nil)"
	// parseRouter.GoGetRoute()
}
`
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseWriteErr2 := os.WriteFile(parseMainPath, []byte(parseMain), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr2)
	}
	if parseWriteErr3 := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html>"), 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseWriteErr3)
	}
	if parseRunErr := (launcher{}).runInit([]string{"-root", parseRoot, "-skip-runtime-assets", "-force"}); parseRunErr != nil {
		parseT.Fatalf("seed init for migrate apply: %v", parseRunErr)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		parseT.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if parseRunErr2 := (launcher{}).runMigrate([]string{"-root", parseRoot, "-skip-runtime-assets", "-apply", "-json"}); parseRunErr2 != nil {
		parseT.Fatalf("run migrate apply: %v", parseRunErr2)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		parseT.Fatalf("read migrate output: %v", parseOutputErr)
	}
	var parseSummary lifecycleMigrateSummary
	if parseDecodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); parseDecodeErr != nil {
		parseT.Fatalf("decode migrate summary: %v\n%s", parseDecodeErr, parseOutput)
	}
	if !parseSummary.Applied || parseSummary.RewriteCount != 2 {
		parseT.Fatalf("expected two applied rewrites, got %#v", parseSummary)
	}
	parseRewrittenBytes, parseReadErr := os.ReadFile(parseMainPath)
	if parseReadErr != nil {
		parseT.Fatalf("read rewritten main.go: %v", parseReadErr)
	}
	parseRewritten := string(parseRewrittenBytes)
	for _, parseWant := range []string{
		"parseRouter.Register(\"/home\", nil)",
		"_ = parseRouter.Current()",
		"\"parseRouter.GoRegisterRoute(\\\"/literal\\\", nil)\"",
		"// parseRouter.GoGetRoute()",
	} {
		if !strings.Contains(parseRewritten, parseWant) {
			parseT.Fatalf("expected rewritten file to contain %q, got %q", parseWant, parseRewritten)
		}
	}
}
