package logging

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/internal/telemetryredaction"
)

const redactedInteractionValue = telemetryredaction.RedactedValue

// RedactionContext describes one telemetry field being filtered.
type RedactionContext = telemetryredaction.Context

// RedactionPolicy configures opt-in telemetry redaction for logs, panic
// reports, and devtools exports. Field names are matched case-insensitively
// against the final path segment. When Redact returns an error the field is
// dropped to fail closed.
type RedactionPolicy = telemetryredaction.Policy

// ConfigureTelemetryRedaction installs the current process-wide telemetry
// redaction policy. Passing the zero value disables policy-based redaction.
func ConfigureTelemetryRedaction(parsePolicy RedactionPolicy) {
	telemetryredaction.Configure(parsePolicy)
}

// CurrentTelemetryRedaction returns a copy of the active redaction policy.
func CurrentTelemetryRedaction() RedactionPolicy {
	return telemetryredaction.Current()
}

// RedactTelemetryField applies the active redaction policy to one field.
// The boolean return is false when the field must be dropped.
func RedactTelemetryField(parsePath string, parseField string, parseValue any) (any, bool) {
	return telemetryredaction.Field(parsePath, parseField, parseValue)
}

// RedactTelemetryValue recursively redacts JSON-like telemetry values.
func RedactTelemetryValue(parsePath string, parseValue any) any {
	return telemetryredaction.Value(parsePath, parseValue)
}

// RedactStringMap applies telemetry redaction to a string map and drops fields
// whose redactor fails.
func RedactStringMap(parsePath string, parseValues map[string]string) map[string]string {
	return telemetryredaction.StringMap(parsePath, parseValues)
}

func redactInteractionValue(parseInputKind string, parseInputValue string) string {
	if strings.EqualFold(strings.TrimSpace(parseInputKind), "password") {
		return redactedInteractionValue
	}
	return parseInputValue
}
