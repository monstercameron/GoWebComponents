package example100_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateAgent4DocSectionCoverage verifies that required Agent 4 documentation sections exist in their expected files.
func TestValidateAgent4DocSectionCoverage(parseT *testing.T) {
	parseT.Helper()
	parseSectionChecks := []struct {
		getPath     string
		getContains []string
	}{
		{
			getPath: "DESIGN.md",
			getContains: []string{
				"### Core dashboard surfaces",
				"### Dashboard KPI definitions (planned)",
				"### Dashboard endpoint and query map (planned)",
				"### Dashboard permissions model (planned)",
			},
		},
		{
			getPath: "FLOWS.md",
			getContains: []string{
				"## Visit-To-First-Chat (Current Runtime)",
				"## Admin-Dashboard Journey (Current Runtime)",
				"## Admin Operational Workflows (Product Definition)",
				"## Dashboard Day-In-The-Life Workflows",
				"## Public-Route Bug-Fix Workflow",
				"## First-Chat Bug-Fix Workflow",
				"## Admin-Dashboard Bug-Fix Workflow",
				"## Admin List Conventions",
				"## Disable vs Suspend Semantics",
			},
		},
		{
			getPath: "OPERATOR_RUNBOOK.md",
			getContains: []string{
				"## 7) Dashboard Review Playbook (Planned)",
				"## 8) Mutation Preflight Guide",
			},
		},
		{
			getPath: "README.md",
			getContains: []string{
				"## Route model",
				"## Runtime pieces",
				"## Repo layout rules",
				"## Pricing philosophy and billing vocabulary",
			},
		},
		{
			getPath: "MANUAL_SMOKE.md",
			getContains: []string{
				"## Release Gate Checklist",
				"## Testing Story Matrix",
			},
		},
		{
			getPath: "DOCS_MAP.md",
			getContains: []string{
				"# Example 100 Docs Map",
				"README.md",
				"FLOWS.md",
				"MANUAL_SMOKE.md",
				"SCHEMA_TABLES.md",
				"CHANGELOG.md",
			},
		},
		{
			getPath: "SCHEMA_TABLES.md",
			getContains: []string{
				"## Wiring Snapshot",
				"## Story Alignment",
			},
		},
	}
	for _, parseCheck := range parseSectionChecks {
		parseText := readAgent4DocSubjectFile(parseT, parseCheck.getPath)
		assertAgent4DocSubjectContains(parseT, parseCheck.getPath, parseText, parseCheck.getContains)
	}
}

// TestValidateAgent4DocPathLayout verifies that Agent 4 file-layout cleanup landed with expected file presence/absence and ignore rules.
func TestValidateAgent4DocPathLayout(parseT *testing.T) {
	parseT.Helper()
	parseMustExist := []string{
		"DOCS_MAP.md",
		"OPERATOR_RUNBOOK.md",
		filepath.Join("docs", "BUG_REPORT_TEMPLATES.md"),
		filepath.Join("docs", "PERFORMANCE.md"),
	}
	for _, parsePath := range parseMustExist {
		parseInfo, parseErr := os.Stat(parsePath)
		if parseErr != nil {
			parseT.Fatalf("expected path to exist %q: %v", parsePath, parseErr)
		}
		if parseInfo.IsDir() {
			parseT.Fatalf("expected path to be a file %q", parsePath)
		}
	}

	parseMustNotExist := []string{
		"BUG_REPORT_TEMPLATES.md",
		"PERFORMANCE.md",
		"BENCHMARKS.md",
		"server.stdout.log",
		"server.stderr.log",
	}
	for _, parsePath := range parseMustNotExist {
		if _, parseErr := os.Stat(parsePath); !os.IsNotExist(parseErr) {
			parseT.Fatalf("expected stale path to be removed %q", parsePath)
		}
	}

	parseGitIgnoreText := readAgent4DocSubjectFile(parseT, ".gitignore")
	assertAgent4DocSubjectContains(parseT, ".gitignore", parseGitIgnoreText, []string{
		"log/",
		"server.stdout.log",
		"server.stderr.log",
	})
}

// TestValidateAgent4PricingVocabularyContract verifies the documented plan boundary and customer-facing billing vocabulary contract.
func TestValidateAgent4PricingVocabularyContract(parseT *testing.T) {
	parseT.Helper()
	parseReadmeText := readAgent4DocSubjectFile(parseT, "README.md")
	assertAgent4DocSubjectContains(parseT, "README.md", parseReadmeText, []string{
		"### Plan boundary",
		"`Pro`: one serious operator",
		"`Team`: shared workspace",
		"`Enterprise`: contract/security/compliance path",
		"`platform fee + usage + service premium`",
		"### Customer-facing vocabulary contract",
		"`platform fee`",
		"`usage`",
		"`service premium`",
		"`free`",
		"`unlimited`",
		"`all models included`",
		"`no token caps`",
	})
}

// readAgent4DocSubjectFile loads one Agent 4 doc file from the example root.
func readAgent4DocSubjectFile(parseT *testing.T, parsePath string) string {
	parseT.Helper()
	parseBytes, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read file %q: %v", parsePath, parseErr)
	}
	return string(parseBytes)
}

// assertAgent4DocSubjectContains asserts that one doc text contains all required tokens.
func assertAgent4DocSubjectContains(parseT *testing.T, parsePath string, parseText string, parseNeedles []string) {
	parseT.Helper()
	for _, parseNeedle := range parseNeedles {
		if !strings.Contains(parseText, parseNeedle) {
			parseT.Fatalf("file %q missing required content %q", parsePath, parseNeedle)
		}
	}
}
