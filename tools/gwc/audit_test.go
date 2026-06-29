package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeAuditFixture writes a go.mod (+ optional go.sum) into a temp dir and returns
// the root.
func writeAuditFixture(parseT *testing.T, parseGoMod string, parseGoSum string) string {
	parseT.Helper()
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseGoMod), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseGoSum != "" {
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.sum"), []byte(parseGoSum), 0644); parseErr != nil {
			parseT.Fatalf("write go.sum: %v", parseErr)
		}
	}
	return parseRoot
}

const auditGoMod = `module example.com/app

go 1.26.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/google/uuid v1.6.0
	local/thing v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace local/thing => ./internal/thing
`

const auditGoSum = `github.com/gorilla/websocket v1.5.3 h1:abc=
github.com/gorilla/websocket v1.5.3/go.mod h1:def=
github.com/google/uuid v1.6.0 h1:ghi=
github.com/google/uuid v1.6.0/go.mod h1:jkl=
github.com/cespare/xxhash/v2 v2.3.0 h1:mno=
`

// TestAuditParsesModuleAndDirectDeps proves the go.mod parser extracts the module,
// go version, and direct (non-indirect) deps while excluding locally-replaced ones.
func TestAuditParsesModuleAndDirectDeps(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, auditGoSum)
	parseReport := buildAuditReport(parseRoot, 0)

	if parseReport.Module != "example.com/app" {
		parseT.Fatalf("module: got %q", parseReport.Module)
	}
	if parseReport.GoVersion != "1.26.0" {
		parseT.Fatalf("goVersion: got %q", parseReport.GoVersion)
	}
	// websocket + uuid are external; local/thing is locally replaced and excluded;
	// xxhash + x/sys are indirect and excluded.
	if parseReport.DirectExternalCount != 2 {
		parseT.Fatalf("expected 2 direct external deps, got %d (%v)", parseReport.DirectExternalCount, parseReport.DirectExternalDeps)
	}
	if parseReport.LocalReplaceCount != 1 {
		parseT.Fatalf("expected 1 local replace, got %d", parseReport.LocalReplaceCount)
	}
}

// TestAuditCountsTransitiveAndVerifiesChecksums proves go.sum module counting and
// the checksum-verified verdict.
func TestAuditCountsTransitiveAndVerifiesChecksums(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, auditGoSum)
	parseReport := buildAuditReport(parseRoot, 0)
	if parseReport.TransitiveModuleCount != 3 {
		parseT.Fatalf("expected 3 distinct go.sum modules, got %d", parseReport.TransitiveModuleCount)
	}
	if !parseReport.ChecksumsVerified {
		parseT.Fatal("expected checksums verified")
	}
}

// TestAuditZeroNPMIsOK proves a project with no npm artifacts passes with the
// zero-npm verdict.
func TestAuditZeroNPMIsOK(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, auditGoSum)
	parseReport := buildAuditReport(parseRoot, 0)
	if !parseReport.ZeroNPM || !parseReport.OK {
		parseT.Fatalf("expected zero-npm OK report, got OK=%t zeroNPM=%t", parseReport.OK, parseReport.ZeroNPM)
	}
}

// TestAuditDetectsNPMArtifact proves a package.json in the build tree fails the
// audit with a supply-chain-surface diagnostic.
func TestAuditDetectsNPMArtifact(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, auditGoSum)
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "package.json"), []byte(`{"name":"x"}`), 0644); parseErr != nil {
		parseT.Fatalf("write package.json: %v", parseErr)
	}
	parseReport := buildAuditReport(parseRoot, 0)
	if parseReport.ZeroNPM || parseReport.OK {
		parseT.Fatalf("expected npm surface to fail the audit, got OK=%t zeroNPM=%t", parseReport.OK, parseReport.ZeroNPM)
	}
	if !auditHasCode(parseReport, "GWC-AUDIT-NPM-SURFACE") {
		parseT.Fatalf("expected GWC-AUDIT-NPM-SURFACE diagnostic, got %+v", parseReport.Diagnostics)
	}
}

// TestAuditExcludesResearchAndExampleDirs proves npm artifacts under excluded dirs
// (research/, examples/, node_modules/) are NOT counted against the module.
func TestAuditExcludesResearchAndExampleDirs(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, auditGoSum)
	parseBenchDir := filepath.Join(parseRoot, "research", "bench")
	if parseErr := os.MkdirAll(parseBenchDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseBenchDir, "package.json"), []byte(`{}`), 0644); parseErr != nil {
		parseT.Fatalf("write: %v", parseErr)
	}
	parseReport := buildAuditReport(parseRoot, 0)
	if !parseReport.ZeroNPM {
		parseT.Fatalf("research/ npm artifacts must be excluded, got %v", parseReport.NPMArtifacts)
	}
}

// TestAuditBudgetExceeded proves the direct-dependency budget gate.
func TestAuditBudgetExceeded(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, auditGoSum)
	parseReport := buildAuditReport(parseRoot, 1) // 2 external deps > budget 1
	if !parseReport.BudgetExceeded || parseReport.OK {
		parseT.Fatalf("expected budget exceeded failure, got OK=%t exceeded=%t", parseReport.OK, parseReport.BudgetExceeded)
	}
	if !auditHasCode(parseReport, "GWC-AUDIT-DEP-BUDGET") {
		parseT.Fatalf("expected GWC-AUDIT-DEP-BUDGET diagnostic, got %+v", parseReport.Diagnostics)
	}
	// Budget that fits passes.
	if parsePass := buildAuditReport(parseRoot, 5); parsePass.BudgetExceeded || !parsePass.OK {
		parseT.Fatalf("expected a generous budget to pass, got OK=%t exceeded=%t", parsePass.OK, parsePass.BudgetExceeded)
	}
}

// TestAuditWarnsOnMissingChecksums proves a module with no go.sum gets a warning.
func TestAuditWarnsOnMissingChecksums(parseT *testing.T) {
	parseRoot := writeAuditFixture(parseT, auditGoMod, "")
	parseReport := buildAuditReport(parseRoot, 0)
	if parseReport.ChecksumsVerified {
		parseT.Fatal("expected checksums unverified without go.sum")
	}
	if !auditHasCode(parseReport, "GWC-AUDIT-NO-CHECKSUMS") {
		parseT.Fatalf("expected GWC-AUDIT-NO-CHECKSUMS warning, got %+v", parseReport.Diagnostics)
	}
}

func auditHasCode(parseReport auditReport, parseCode string) bool {
	for _, parseDiagnostic := range parseReport.Diagnostics {
		if parseDiagnostic.Code == parseCode {
			return true
		}
	}
	return false
}
