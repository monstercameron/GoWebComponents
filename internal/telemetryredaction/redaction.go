package telemetryredaction

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/diagnostics"
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
		// Contract: callers MUST pass JSON-decoded telemetry values — the shapes
		// json.Unmarshal into `any` produces (map[string]any, []any, and scalars).
		// A struct, a pointer, or a map with non-string keys cannot be walked, so
		// its nested fields are returned WITHOUT redaction. Every in-tree caller
		// normalizes first (a json.Marshal→Unmarshal round-trip, or logging's
		// normalizeLogValue), so this is unreachable today; the guard exists so a
		// future caller passing a raw struct with secret fields is caught in
		// dev/test instead of silently leaking. Scalars pass through with no cost.
		if !isJSONScalar(parseValue) {
			reportNonJSONTelemetryValue(parsePath, parseValue)
		}
		return normalizeValue(parseValue)
	}
}

// isJSONScalar reports whether parseValue is a leaf that json.Unmarshal-into-any
// can produce (or a common Go numeric/bool scalar), so the dev guard does not flag
// ordinary passthrough values. Deliberately a cheap type switch — NO reflection —
// because it runs on the telemetry-wide hot path for every non-collection value.
func isJSONScalar(parseValue any) bool {
	switch parseValue.(type) {
	case nil, bool, string,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return true
	default:
		return false
	}
}

// nonJSONReported dedups the dev guard so a repeated non-JSON type cannot spam the
// diagnostics sink on the telemetry hot path.
var (
	nonJSONMu       sync.Mutex
	nonJSONReported = map[string]struct{}{}
)

// reportNonJSONTelemetryValue warns (once per distinct Go type) that a value which
// is not JSON-like reached Value() and was therefore passed through unredacted. A
// test seam.
var reportNonJSONTelemetryValue = func(parsePath string, parseValue any) {
	parseTypeName := fmt.Sprintf("%T", parseValue)
	nonJSONMu.Lock()
	_, parseSeen := nonJSONReported[parseTypeName]
	if !parseSeen {
		nonJSONReported[parseTypeName] = struct{}{}
	}
	nonJSONMu.Unlock()
	if parseSeen {
		return
	}
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Code:     "GWC-TELEMETRY-NONJSON",
		Headline: fmt.Sprintf("non-JSON telemetry value of type %s was not redacted", parseTypeName),
		Summary:  fmt.Sprintf("telemetryredaction.Value received a %s at path %q; it is not a JSON-like shape (map[string]any / []any / scalar), so it was returned WITHOUT walking its fields — any nested secret field is unredacted", parseTypeName, parsePath),
		Next:     "normalize telemetry to JSON-decoded values before redaction (json.Marshal then Unmarshal into any, or logging.normalizeLogValue); do not pass raw structs or pointers to Value",
	}))
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
