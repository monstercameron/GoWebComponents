package telemetryredaction

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"sync"
)

const RedactedValue = "[redacted]"

// Context describes one telemetry field being filtered.
type Context struct {
	Path  string
	Field string
	Value any
}

// Policy configures opt-in telemetry redaction. Field names are matched
// case-insensitively against the final path segment. Redactor errors drop the
// field so telemetry fails closed.
type Policy struct {
	Fields []string
	Redact func(Context) (any, error)
}

var state struct {
	mu     sync.RWMutex
	policy Policy
	fields map[string]struct{}
}

// Configure installs the current process-wide telemetry redaction policy.
func Configure(parsePolicy Policy) {
	parseFields := map[string]struct{}{}
	for _, parseField := range parsePolicy.Fields {
		parseNormalized := normalizeField(parseField)
		if parseNormalized != "" {
			parseFields[parseNormalized] = struct{}{}
		}
	}
	parsePolicy.Fields = append([]string(nil), parsePolicy.Fields...)
	state.mu.Lock()
	state.policy = parsePolicy
	if len(parseFields) == 0 {
		state.fields = nil
	} else {
		state.fields = parseFields
	}
	state.mu.Unlock()
}

// Current returns a copy of the active redaction policy.
func Current() Policy {
	state.mu.RLock()
	defer state.mu.RUnlock()
	parsePolicy := state.policy
	parsePolicy.Fields = append([]string(nil), parsePolicy.Fields...)
	return parsePolicy
}

// Field applies the active redaction policy to one field.
func Field(parsePath string, parseField string, parseValue any) (any, bool) {
	parsePolicy, parseMatched := currentPolicy(parsePath, parseField)
	if !parseMatched {
		return parseValue, true
	}
	if parsePolicy.Redact == nil {
		return RedactedValue, true
	}
	parseRedacted, parseErr := parsePolicy.Redact(Context{
		Path:  strings.TrimSpace(parsePath),
		Field: strings.TrimSpace(parseField),
		Value: parseValue,
	})
	if parseErr != nil {
		return nil, false
	}
	return normalizeValue(parseRedacted), true
}

// Value recursively redacts JSON-like telemetry values.
func Value(parsePath string, parseValue any) any {
	return redactValue(parsePath, parseValue)
}

// StringMap applies telemetry redaction to a string map and drops fields whose
// redactor fails.
func StringMap(parsePath string, parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return parseValues
	}
	parseRedacted := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseNextPath := JoinPath(parsePath, parseKey)
		parseFieldValue, parseKeep := Field(parseNextPath, parseKey, parseValue)
		if !parseKeep {
			continue
		}
		parseRedacted[parseKey] = fmt.Sprint(parseFieldValue)
	}
	return parseRedacted
}

func currentPolicy(parsePath string, parseField string) (Policy, bool) {
	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.fields) == 0 {
		return Policy{}, false
	}
	parseNormalizedField := normalizeField(parseField)
	if parseNormalizedField == "" {
		parseNormalizedField = normalizeField(lastPathSegment(parsePath))
	}
	_, parseMatched := state.fields[parseNormalizedField]
	if !parseMatched {
		return Policy{}, false
	}
	parsePolicy := state.policy
	parsePolicy.Fields = append([]string(nil), parsePolicy.Fields...)
	return parsePolicy, true
}

func redactValue(parsePath string, parseValue any) any {
	switch parseTyped := parseValue.(type) {
	case map[string]any:
		parseOut := make(map[string]any, len(parseTyped))
		for parseKey, parseNested := range parseTyped {
			parseNextPath := JoinPath(parsePath, parseKey)
			parseFieldValue, parseKeep := Field(parseNextPath, parseKey, parseNested)
			if !parseKeep {
				continue
			}
			parseOut[parseKey] = redactValue(parseNextPath, parseFieldValue)
		}
		return parseOut
	case map[string]string:
		return StringMap(parsePath, parseTyped)
	case []any:
		parseOut := make([]any, 0, len(parseTyped))
		for parseIndex, parseNested := range parseTyped {
			// Manual concat instead of fmt.Sprintf: avoids per-element interface
			// boxing and format parsing on this telemetry-wide path.
			parseOut = append(parseOut, redactValue(parsePath+"["+strconv.Itoa(parseIndex)+"]", parseNested))
		}
		return parseOut
	case []string:
		parseOut := make([]string, len(parseTyped))
		copy(parseOut, parseTyped)
		return parseOut
	default:
		return normalizeValue(parseValue)
	}
}

func normalizeValue(parseValue any) any {
	switch parseTyped := parseValue.(type) {
	case map[string]any:
		parseCopy := make(map[string]any, len(parseTyped))
		maps.Copy(parseCopy, parseTyped)
		return parseCopy
	case map[string]string:
		parseCopy := make(map[string]string, len(parseTyped))
		maps.Copy(parseCopy, parseTyped)
		return parseCopy
	case []any:
		return append([]any(nil), parseTyped...)
	case []string:
		return append([]string(nil), parseTyped...)
	default:
		return parseValue
	}
}

func normalizeField(parseField string) string {
	parseField = strings.TrimSpace(strings.ToLower(parseField))
	parseField = strings.NewReplacer("-", "", "_", "", " ", "", ".", "").Replace(parseField)
	return parseField
}

// JoinPath appends one telemetry field to a dotted redaction path.
func JoinPath(parseBase string, parseField string) string {
	parseBase = strings.TrimSpace(parseBase)
	parseField = strings.TrimSpace(parseField)
	if parseBase == "" {
		return parseField
	}
	if parseField == "" {
		return parseBase
	}
	return parseBase + "." + parseField
}

func lastPathSegment(parsePath string) string {
	parsePath = strings.TrimSpace(parsePath)
	if parsePath == "" {
		return ""
	}
	parsePath = strings.TrimRight(parsePath, ".")
	if parseIndex := strings.LastIndex(parsePath, "."); parseIndex >= 0 {
		return parsePath[parseIndex+1:]
	}
	return parsePath
}
