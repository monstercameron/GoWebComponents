package runtime2

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	matchDiagnosticBearerTokenPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-._~+/]+=*`)
	matchDiagnosticKeyValuePattern    = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]*(?:source(?:s|_values|_snapshot)?|snapshot|props|properties|secret|token|password|api[_-]?key|authorization|credential)[A-Za-z0-9_.-]*)\s*([:=])\s*("[^"]*"|'[^']*'|[^,\s;]+)`)
)

// RedactControlDiagnosticEnvelope scrubs diagnostic text fields before control-envelope emission or host-side processing.
func RedactControlDiagnosticEnvelope(parseEnvelope ControlEnvelope) ControlEnvelope {
	if parseEnvelope.Kind != ControlKindDiagnostic {
		return parseEnvelope
	}
	parseEnvelope.DiagnosticText = RedactDiagnosticText(parseEnvelope.DiagnosticText)
	return parseEnvelope
}

// RedactDiagnosticText removes source snapshot values and likely secrets from one diagnostic text payload.
func RedactDiagnosticText(parseDiagnosticText string) string {
	parseTrimmedText := strings.TrimSpace(parseDiagnosticText)
	if parseTrimmedText == "" {
		return parseDiagnosticText
	}
	if strings.HasPrefix(parseTrimmedText, "{") || strings.HasPrefix(parseTrimmedText, "[") {
		var parseJSONValue any
		if parseDecodeErr := json.Unmarshal([]byte(parseTrimmedText), &parseJSONValue); parseDecodeErr == nil {
			parseRedactedJSONValue := redactDiagnosticJSONValue("", parseJSONValue)
			parseRedactedJSON, parseEncodeErr := json.Marshal(parseRedactedJSONValue)
			if parseEncodeErr == nil {
				return string(parseRedactedJSON)
			}
		}
	}
	parseRedactedText := matchDiagnosticBearerTokenPattern.ReplaceAllString(parseDiagnosticText, "Bearer [redacted]")
	parseRedactedText = matchDiagnosticKeyValuePattern.ReplaceAllString(parseRedactedText, "${1}${2}[redacted]")
	return parseRedactedText
}

// redactDiagnosticJSONValue traverses one JSON diagnostic payload and redacts sensitive source or secret fields.
func redactDiagnosticJSONValue(parseFieldName string, parseValue any) any {
	if isDiagnosticSensitiveField(parseFieldName) {
		return "[redacted]"
	}
	switch parseTypedValue := parseValue.(type) {
	case map[string]any:
		parseRedactedMap := make(map[string]any, len(parseTypedValue))
		for parseMapKey, parseMapValue := range parseTypedValue {
			parseRedactedMap[parseMapKey] = redactDiagnosticJSONValue(parseMapKey, parseMapValue)
		}
		return parseRedactedMap
	case []any:
		parseRedactedArray := make([]any, len(parseTypedValue))
		for parseArrayIndex := range parseTypedValue {
			parseRedactedArray[parseArrayIndex] = redactDiagnosticJSONValue(parseFieldName, parseTypedValue[parseArrayIndex])
		}
		return parseRedactedArray
	default:
		return parseValue
	}
}

// isDiagnosticSensitiveField reports whether one diagnostic field name should have its value redacted.
func isDiagnosticSensitiveField(parseFieldName string) bool {
	parseNormalizedFieldName := strings.ToLower(strings.TrimSpace(parseFieldName))
	if parseNormalizedFieldName == "" {
		return false
	}
	parseNormalizedFieldName = strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(parseNormalizedFieldName)
	switch parseNormalizedFieldName {
	case "source", "sources", "sourcevalue", "sourcevalues", "sourcesnapshot", "sourcesnapshots", "snapshot", "snapshots",
		"props", "properties", "credential", "credentials", "password", "passwords", "secret", "secrets", "token", "tokens",
		"apikey", "authorization", "bearer":
		return true
	}
	if strings.Contains(parseNormalizedFieldName, "secret") ||
		strings.Contains(parseNormalizedFieldName, "token") ||
		strings.Contains(parseNormalizedFieldName, "password") ||
		strings.Contains(parseNormalizedFieldName, "apikey") ||
		strings.Contains(parseNormalizedFieldName, "authorization") ||
		strings.Contains(parseNormalizedFieldName, "credential") {
		return true
	}
	if strings.HasPrefix(parseNormalizedFieldName, "source") &&
		(strings.Contains(parseNormalizedFieldName, "value") ||
			strings.Contains(parseNormalizedFieldName, "snapshot") ||
			strings.HasSuffix(parseNormalizedFieldName, "payload")) {
		return true
	}
	return false
}
