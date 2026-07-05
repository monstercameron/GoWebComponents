package runtime

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// StrictDiagnosticsOptions controls whether selected recoverable diagnostics escalate into hard failures.
type StrictDiagnosticsOptions struct {
	Enabled               bool
	Codes                 []string
	Sources               []string
	Classifications       []DiagnosticClassification
	RecoverableOnly       bool
	EscalationConsequence string
}

var strictDiagnosticsState struct {
	mu      sync.RWMutex
	options StrictDiagnosticsOptions
}

// ConfigureStrictDiagnostics sets the current strict-diagnostics behavior.
func ConfigureStrictDiagnostics(parseOptions StrictDiagnosticsOptions) {
	strictDiagnosticsState.mu.Lock()
	defer strictDiagnosticsState.mu.Unlock()
	parseOptions.Codes = append([]string(nil), parseOptions.Codes...)
	parseOptions.Sources = append([]string(nil), parseOptions.Sources...)
	parseOptions.Classifications = append([]DiagnosticClassification(nil), parseOptions.Classifications...)
	if !parseOptions.RecoverableOnly {
		parseOptions.RecoverableOnly = true
	}
	strictDiagnosticsState.options = parseOptions
}

// strictDiagnosticsEnabled reports whether strict escalation is on without
// cloning the option slices; every diagnostic report checks this, and strict
// mode is off by default.
func strictDiagnosticsEnabled() bool {
	strictDiagnosticsState.mu.RLock()
	defer strictDiagnosticsState.mu.RUnlock()
	return strictDiagnosticsState.options.Enabled
}

// CurrentStrictDiagnosticsOptions returns the current strict-diagnostics behavior.
func CurrentStrictDiagnosticsOptions() StrictDiagnosticsOptions {
	strictDiagnosticsState.mu.RLock()
	defer strictDiagnosticsState.mu.RUnlock()
	parseOptions := strictDiagnosticsState.options
	parseOptions.Codes = append([]string(nil), parseOptions.Codes...)
	parseOptions.Sources = append([]string(nil), parseOptions.Sources...)
	parseOptions.Classifications = append([]DiagnosticClassification(nil), parseOptions.Classifications...)
	return parseOptions
}

// shouldEscalateDiagnosticStrictly is a core package helper.
func shouldEscalateDiagnosticStrictly(parseSource string, parseSeverity DiagnosticSeverity, parseClassification DiagnosticClassification, parseDetails diagnosticDetails) bool {
	parseOptions := CurrentStrictDiagnosticsOptions()
	if !parseOptions.Enabled {
		return false
	}
	if parseOptions.RecoverableOnly && !parseDetails.Recoverable {
		return false
	}
	if !strictDiagnosticStringMatch(parseSource, parseOptions.Sources) {
		return false
	}
	if !strictDiagnosticStringMatch(parseDetails.Code, parseOptions.Codes) {
		return false
	}
	if len(parseOptions.Classifications) > 0 {
		isParseMatched := slices.Contains(parseOptions.Classifications, parseClassification)
		if !isParseMatched {
			return false
		}
	}
	_ = parseSeverity
	return true
}

// strictDiagnosticStringMatch is a core package helper.
func strictDiagnosticStringMatch(parseValue string, parseFilters []string) bool {
	if len(parseFilters) == 0 {
		return true
	}
	parseTrimmedValue := strings.TrimSpace(parseValue)
	for _, filter := range parseFilters {
		if strings.EqualFold(strings.TrimSpace(filter), parseTrimmedValue) {
			return true
		}
	}
	return false
}

// escalateStrictDiagnostic is a core package helper.
func escalateStrictDiagnostic(parseSource string, parseDetails diagnosticDetails, parseMessage string, parsePath string, parseComponentStack []string) {
	parseOptions := CurrentStrictDiagnosticsOptions()
	parseConsequence := strings.TrimSpace(parseOptions.EscalationConsequence)
	if parseConsequence == "" {
		parseConsequence = "strict diagnostic mode escalated a recoverable framework warning into a hard failure for development or test runs."
	}
	parseCode := strings.TrimSpace(parseDetails.Code)
	if parseCode == "" {
		parseCode = "diagnostic"
	}
	panic(ActionableFrameworkPanic(ActionablePanicOptions{
		Source:         strings.TrimSpace(parseSource),
		Subject:        "strict diagnostics",
		Message:        fmt.Sprintf("strict diagnostics escalated %s: %s", parseCode, strings.TrimSpace(parseMessage)),
		Path:           strings.TrimSpace(parsePath),
		ComponentStack: append([]string(nil), parseComponentStack...),
		Consequence:    parseConsequence,
	}))
}
