package example100_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExample100QualityProofDocsCoverTargetPasses(parseT *testing.T) {
	parseChecks := []struct {
		path    string
		needles []string
	}{
		{
			path: filepath.Join("docs", "ACCESSIBILITY_CHECKLIST.md"),
			needles: []string{
				"## Keyboard Navigation",
				"## Landmarks And Headings",
				"## Forms, Names, Help Text, And Errors",
				"## Live Regions And Status Messages",
				"## Reduced Motion",
				"## Contrast And Non-Color State",
				"## Screen-Reader Smoke Path",
				"login",
				"First message",
				"Settings save",
				"Admin mutation path",
			},
		},
		{
			path: filepath.Join("docs", "PERFORMANCE_PROOF.md"),
			needles: []string{
				"## Runtime Claims",
				"## Cold-Start Checklist",
				"## First-Chat Trace",
				"## Dashboard Slice Responsiveness",
				"## Sidebar Scale Proof",
				"## Worker Value Proof",
				"## Performance Regression Checklist",
				"Public first paint",
				"Stub first chunk",
			},
		},
		{
			path: filepath.Join("docs", "BRIDGE_CHURN_HARDENING.md"),
			needles: []string{
				"## Bridge State Model",
				"## RPC Policy Table",
				"## Best-Effort Guard",
				"## Deferred Queue",
				"## Retry Policy",
				"## Write-Path Audit",
				"## Idempotency Review",
				"## User Experience",
				"## Observability Seam",
				"## Browser Regression Suite",
				"## Troubleshooting Notes",
				"## Secondary Bridge Decision",
				"## No Round-Robin Note",
				"`booting`",
				"`ready`",
				"`reconnecting`",
				"`degraded`",
				"`sleeping`",
				"`offline`",
			},
		},
		{
			path: "README.md",
			needles: []string{
				"## Quality Proofs",
				"docs/ACCESSIBILITY_CHECKLIST.md",
				"docs/PERFORMANCE_PROOF.md",
				"docs/BRIDGE_CHURN_HARDENING.md",
			},
		},
	}
	for _, parseCheck := range parseChecks {
		parseText := readQualityProofSubjectFile(parseT, parseCheck.path)
		for _, parseNeedle := range parseCheck.needles {
			if !strings.Contains(parseText, parseNeedle) {
				parseT.Fatalf("%s missing %q", parseCheck.path, parseNeedle)
			}
		}
	}
}

func readQualityProofSubjectFile(parseT *testing.T, parsePath string) string {
	parseT.Helper()
	parseBytes, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read %s: %v", parsePath, parseErr)
	}
	return string(parseBytes)
}
