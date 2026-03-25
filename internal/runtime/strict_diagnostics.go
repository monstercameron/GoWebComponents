package runtime

import (
	"fmt"
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
func ConfigureStrictDiagnostics(options StrictDiagnosticsOptions) {
	strictDiagnosticsState.mu.Lock()
	defer strictDiagnosticsState.mu.Unlock()
	options.Codes = append([]string(nil), options.Codes...)
	options.Sources = append([]string(nil), options.Sources...)
	options.Classifications = append([]DiagnosticClassification(nil), options.Classifications...)
	if !options.RecoverableOnly {
		options.RecoverableOnly = true
	}
	strictDiagnosticsState.options = options
}

// CurrentStrictDiagnosticsOptions returns the current strict-diagnostics behavior.
func CurrentStrictDiagnosticsOptions() StrictDiagnosticsOptions {
	strictDiagnosticsState.mu.RLock()
	defer strictDiagnosticsState.mu.RUnlock()
	options := strictDiagnosticsState.options
	options.Codes = append([]string(nil), options.Codes...)
	options.Sources = append([]string(nil), options.Sources...)
	options.Classifications = append([]DiagnosticClassification(nil), options.Classifications...)
	return options
}

func shouldEscalateDiagnosticStrictly(source string, severity DiagnosticSeverity, classification DiagnosticClassification, details diagnosticDetails) bool {
	options := CurrentStrictDiagnosticsOptions()
	if !options.Enabled {
		return false
	}
	if options.RecoverableOnly && !details.Recoverable {
		return false
	}
	if !strictDiagnosticStringMatch(source, options.Sources) {
		return false
	}
	if !strictDiagnosticStringMatch(details.Code, options.Codes) {
		return false
	}
	if len(options.Classifications) > 0 {
		matched := false
		for _, item := range options.Classifications {
			if item == classification {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	_ = severity
	return true
}

func strictDiagnosticStringMatch(value string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	trimmedValue := strings.TrimSpace(value)
	for _, filter := range filters {
		if strings.EqualFold(strings.TrimSpace(filter), trimmedValue) {
			return true
		}
	}
	return false
}

func escalateStrictDiagnostic(source string, details diagnosticDetails, message string, path string, componentStack []string) {
	options := CurrentStrictDiagnosticsOptions()
	consequence := strings.TrimSpace(options.EscalationConsequence)
	if consequence == "" {
		consequence = "strict diagnostic mode escalated a recoverable framework warning into a hard failure for development or test runs."
	}
	code := strings.TrimSpace(details.Code)
	if code == "" {
		code = "diagnostic"
	}
	panic(ActionableFrameworkPanic(ActionablePanicOptions{
		Source:         strings.TrimSpace(source),
		Subject:        "strict diagnostics",
		Message:        fmt.Sprintf("strict diagnostics escalated %s: %s", code, strings.TrimSpace(message)),
		Path:           strings.TrimSpace(path),
		ComponentStack: append([]string(nil), componentStack...),
		Consequence:    consequence,
	}))
}
