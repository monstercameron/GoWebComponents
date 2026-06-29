package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// checkFixReport summarizes what `gwc check --fix` changed: the deterministic structured edits
// it applied, and the diagnostics that have no auto-fix and still need a human/agent.
type checkFixReport struct {
	Applied []appliedFix `json:"applied"`
	Manual  []manualFix  `json:"manual"`
}

// appliedFix is one structured remediation that was applied to a file.
type appliedFix struct {
	File        string `json:"file"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// manualFix is one diagnostic code that carried no deterministic edit, with how many such
// diagnostics remain, so the report is honest about what `--fix` could not do.
type manualFix struct {
	Code        string `json:"code"`
	Count       int    `json:"count"`
	Remediation string `json:"remediation"`
}

// applyCheckFixes is the implementation of `gwc check --fix`. It (1) gofmt's the project (the
// always-safe normalization), then (2) applies the deterministic structured `Edits` the
// diagnostics carry — today the server-leak build-constraint rewrite — looping until the fixed
// point so a fix that exposes another is also applied, and (3) gofmt's again so the rewritten
// files stay formatted. It returns a report distinguishing what it fixed from what still needs
// a manual remediation, which the caller surfaces to the agent. This is the AI-native post-edit
// hook: an agent that imports a server package into a browser file runs `--fix` and the file is
// moved server-side, not merely reformatted.
func applyCheckFixes(parseL launcher, parseRoot string, parseJSON bool) (checkFixReport, error) {
	parseReport := checkFixReport{Applied: []appliedFix{}, Manual: []manualFix{}}

	if parseErr := runCheckGofmt(parseL, parseRoot, parseJSON); parseErr != nil {
		return parseReport, parseErr
	}

	parseManualCounts := map[string]int{}
	const parseMaxPasses = 8
	for parsePass := 0; parsePass < parseMaxPasses; parsePass++ {
		parseDiagnostics := collectFixableDiagnostics(parseRoot)
		parseApplied, parseManual, parseErr := applyAgenticEdits(parseRoot, parseDiagnostics)
		if parseErr != nil {
			return parseReport, parseErr
		}
		for parseCode, parseCount := range parseManual {
			if parseCount > parseManualCounts[parseCode] {
				parseManualCounts[parseCode] = parseCount
			}
		}
		if len(parseApplied) == 0 {
			break
		}
		parseReport.Applied = append(parseReport.Applied, parseApplied...)
	}

	if len(parseReport.Applied) > 0 {
		// Re-format: constraint rewrites can shift the blank line before `package`.
		if parseErr := runCheckGofmt(parseL, parseRoot, parseJSON); parseErr != nil {
			return parseReport, parseErr
		}
	}

	for parseCode, parseCount := range parseManualCounts {
		parseReport.Manual = append(parseReport.Manual, manualFix{
			Code:        parseCode,
			Count:       parseCount,
			Remediation: manualRemediationFor(parseCode),
		})
	}
	sort.Slice(parseReport.Manual, func(parseI int, parseJ int) bool {
		return parseReport.Manual[parseI].Code < parseReport.Manual[parseJ].Code
	})
	return parseReport, nil
}

// runCheckGofmt runs the existing fmt path so formatting stays single-sourced.
func runCheckGofmt(parseL launcher, parseRoot string, parseJSON bool) error {
	parseFmtArgs := []string{}
	if parseRoot != "" {
		parseFmtArgs = append(parseFmtArgs, "-root", parseRoot)
	}
	if parseJSON {
		parseFmtArgs = append(parseFmtArgs, "-json")
	}
	return parseL.runFmt(parseFmtArgs)
}

// collectFixableDiagnostics gathers the diagnostics that can carry deterministic edits. It runs
// the convention/hook/server-leak passes (not `go test`, which has no edits and is slow), so
// `--fix` stays fast and focused on the structured remediations.
func collectFixableDiagnostics(parseRoot string) []agenticDiagnostic {
	parseDiagnostics := []agenticDiagnostic{}
	parseDiagnostics = append(parseDiagnostics, collectServerLeakDiagnostics(parseRoot)...)
	parseDiagnostics = append(parseDiagnostics, collectCheckHookContextDiagnostics(parseRoot)...)
	return parseDiagnostics
}

// applyAgenticEdits applies every deterministic edit the diagnostics carry, de-duplicated by
// (file, old text) so the same constraint rewrite triggered by several leaked imports is applied
// once. It returns the applied fixes and a per-code count of diagnostics that carried NO edit
// (the manual-only remainder). An edit whose OldText is already absent is treated as
// already-applied (idempotent), not an error.
func applyAgenticEdits(parseRoot string, parseDiagnostics []agenticDiagnostic) ([]appliedFix, map[string]int, error) {
	parseApplied := []appliedFix{}
	parseManual := map[string]int{}
	parseSeenEdit := map[string]bool{}

	for _, parseDiagnostic := range parseDiagnostics {
		if len(parseDiagnostic.Edits) == 0 {
			parseManual[parseDiagnostic.Code]++
			continue
		}
		for _, parseEdit := range parseDiagnostic.Edits {
			parseKey := parseEdit.File + "\x00" + parseEdit.OldText + "\x00" + parseEdit.NewText
			if parseSeenEdit[parseKey] {
				continue
			}
			parseSeenEdit[parseKey] = true
			parseChanged, parseErr := applyOneEdit(parseRoot, parseEdit)
			if parseErr != nil {
				return parseApplied, parseManual, parseErr
			}
			if parseChanged {
				parseApplied = append(parseApplied, appliedFix{
					File:        parseEdit.File,
					Code:        parseDiagnostic.Code,
					Description: parseEdit.Description,
				})
			}
		}
	}
	return parseApplied, parseManual, nil
}

// applyOneEdit replaces the first occurrence of OldText with NewText in the edit's file,
// reporting whether the file content changed. A missing file or already-absent OldText is a
// no-op (idempotent), not an error.
func applyOneEdit(parseRoot string, parseEdit agenticTextEdit) (bool, error) {
	parsePath := filepath.Join(parseRoot, filepath.FromSlash(parseEdit.File))
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return false, nil
		}
		return false, fmt.Errorf("read %s for fix: %w", parseEdit.File, parseErr)
	}
	parseContent := string(parseData)
	if parseEdit.OldText == "" || !strings.Contains(parseContent, parseEdit.OldText) {
		return false, nil
	}
	parseUpdated := strings.Replace(parseContent, parseEdit.OldText, parseEdit.NewText, 1)
	if parseUpdated == parseContent {
		return false, nil
	}
	if parseErr := os.WriteFile(parsePath, []byte(parseUpdated), 0644); parseErr != nil {
		return false, fmt.Errorf("write %s for fix: %w", parseEdit.File, parseErr)
	}
	return true, nil
}

// manualRemediationFor returns guidance for a diagnostic code that has no deterministic edit, so
// `--fix` can tell the agent exactly what to do by hand instead of pretending it fixed it.
func manualRemediationFor(parseCode string) string {
	switch parseCode {
	case checkHookContextCode:
		return "Move the hook to the component's top level and capture the value it returns; a closure/helper/package-level hook call cannot be rewritten safely without changing behavior."
	case "GWC-CONVENTION-GODOC":
		return "Add a GoDoc comment whose first word is the symbol name immediately before the declaration."
	default:
		return "See the diagnostic's suggestion field."
	}
}
