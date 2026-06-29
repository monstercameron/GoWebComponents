package app

import (
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
)

const parseLogRedactionText = "[REDACTED]"

var parseRedactSensitiveKeyEquals = map[string]struct{}{
	"password":           {},
	"secret":             {},
	"api_key":            {},
	"apikey":             {},
	"authorization":      {},
	"cookie":             {},
	"set_cookie":         {},
	"access_token":       {},
	"refresh_token":      {},
	"id_token":           {},
	"auth_token":         {},
	"bearer_token":       {},
	"client_secret":      {},
	"webhook_secret":     {},
	"provider_payload":   {},
	"payload_json":       {},
	"raw_payload":        {},
	"internal_note":      {},
	"internal_note_body": {},
	"note_body":          {},
}

var parseRedactSensitiveKeyFragments = []string{
	"password",
	"secret",
	"api_key",
	"apikey",
	"authorization",
	"cookie",
	"credential",
	"webhook_secret",
	"provider_payload",
	"payload_json",
	"raw_payload",
	"internal_note",
	"note_body",
}

var parseRedactBearerTokenPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9\-\._~\+/]+=*`)
var parseRedactJWTTokenPattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
var parseRedactKeyValueSecretPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|id[_-]?token|password|secret|cookie|authorization)\b(\s*[:=]\s*)([^,\s;]+)`)
var parseRedactEnvSecretAssignPattern = regexp.MustCompile(`(?i)\b[A-Z][A-Z0-9_]{2,}(_KEY|_SECRET|_TOKEN)\b(\s*[:=]\s*)([^,\s;]+)`)
var parseRedactEnvSecretKeyPattern = regexp.MustCompile(`(?i)\b[A-Z][A-Z0-9_]{2,}(_KEY|_SECRET|_TOKEN)\b`)
var parseRedactPathPattern = regexp.MustCompile(`([A-Za-z]:\\[^,\s;]+|\\.\\[^,\s;]+|/[^,\s;]+)`)

// parseScrubSecretString removes common credential, path, and token patterns from one string value.
func parseScrubSecretString(parseValue string) string {
	parseValue = strings.TrimSpace(parseValue)
	if parseValue == "" {
		return ""
	}
	parseValue = parseRedactBearerTokenPattern.ReplaceAllStringFunc(parseValue, func(_ string) string {
		return "Bearer " + parseLogRedactionText
	})
	parseValue = parseRedactJWTTokenPattern.ReplaceAllString(parseValue, parseLogRedactionText)
	parseValue = parseRedactKeyValueSecretPattern.ReplaceAllString(parseValue, "$1$2"+parseLogRedactionText)
	parseValue = parseRedactEnvSecretAssignPattern.ReplaceAllString(parseValue, parseLogRedactionText)
	parseValue = parseRedactEnvSecretKeyPattern.ReplaceAllString(parseValue, parseLogRedactionText)
	parseValue = parseRedactPathPattern.ReplaceAllString(parseValue, parseLogRedactionText)
	return strings.TrimSpace(parseValue)
}

// parseScrubSecretJSONString redacts secret-bearing values within one JSON payload while preserving JSON structure when possible.
func parseScrubSecretJSONString(parseValue string) string {
	parseValue = strings.TrimSpace(parseValue)
	if parseValue == "" {
		return ""
	}
	var parseDecoded any
	if parseErr := json.Unmarshal([]byte(parseValue), &parseDecoded); parseErr != nil {
		return parseScrubSecretString(parseValue)
	}
	parseScrubbed := parseRedactLogAnyValue(parseDecoded)
	parseEncoded, parseErr := json.Marshal(parseScrubbed)
	if parseErr != nil {
		return parseScrubSecretString(parseValue)
	}
	return string(parseEncoded)
}

// parseRedactLogAttr applies key-based and value-based sensitive-field redaction to one slog attribute.
func parseRedactLogAttr(parseAttr slog.Attr) slog.Attr {
	return parseBuildRedactedLogAttr("", parseAttr)
}

// parseBuildRedactedLogAttr applies key-based and value-based log redaction while preserving the current nested key path.
func parseBuildRedactedLogAttr(parseParentKeyPath string, parseAttr slog.Attr) slog.Attr {
	parseKey := strings.TrimSpace(parseAttr.Key)
	parseKeyPath := parseKey
	if strings.TrimSpace(parseParentKeyPath) != "" {
		parseKeyPath = strings.TrimSpace(parseParentKeyPath) + "." + parseKey
	}
	if parseIsSensitiveLogKey(parseKey) {
		return slog.String(parseKey, parseLogRedactionText)
	}
	parseResolvedAttr := parseAttr.Value.Resolve()
	switch parseResolvedAttr.Kind() {
	case slog.KindString:
		if parseShouldPreserveLogPathValue(parseKeyPath, parseResolvedAttr.String()) {
			return slog.String(parseKey, strings.TrimSpace(parseResolvedAttr.String()))
		}
		return slog.String(parseKey, parseRedactLogString(parseResolvedAttr.String()))
	case slog.KindAny:
		return slog.Any(parseKey, parseRedactLogAnyValueWithKeyPath(parseKeyPath, parseResolvedAttr.Any()))
	case slog.KindGroup:
		parseGroupAttrs := parseResolvedAttr.Group()
		parseRedactedGroupAttrs := make([]slog.Attr, 0, len(parseGroupAttrs))
		for _, parseGroupAttr := range parseGroupAttrs {
			parseRedactedGroupAttrs = append(parseRedactedGroupAttrs, parseBuildRedactedLogAttr(parseKeyPath, parseGroupAttr))
		}
		return slog.Attr{Key: parseKey, Value: slog.GroupValue(parseRedactedGroupAttrs...)}
	default:
		return parseAttr
	}
}

// parseShouldPreserveLogPathValue reports whether one path-like value is an allowed route field rather than a filesystem secret.
func parseShouldPreserveLogPathValue(parseKeyPath string, parseValue string) bool {
	parseNormalizedKeyPath := parseNormalizeLogKey(parseKeyPath)
	parseValue = strings.TrimSpace(parseValue)
	if parseValue == "" || !strings.HasPrefix(parseValue, "/") {
		return false
	}
	switch parseNormalizedKeyPath {
	case "route", "attributes_route", "request_path":
		return true
	default:
		return false
	}
}

// parseIsSensitiveLogKey reports whether one log key should be redacted regardless of value.
func parseIsSensitiveLogKey(parseKey string) bool {
	parseNormalizedKey := parseNormalizeLogKey(parseKey)
	if parseNormalizedKey == "" {
		return false
	}
	if _, hasParseExactMatch := parseRedactSensitiveKeyEquals[parseNormalizedKey]; hasParseExactMatch {
		return true
	}
	for _, parseFragment := range parseRedactSensitiveKeyFragments {
		if strings.Contains(parseNormalizedKey, parseFragment) {
			return true
		}
	}
	return false
}

// parseNormalizeLogKey lowercases and normalizes one log key for policy matching.
func parseNormalizeLogKey(parseKey string) string {
	parseNormalizedKey := strings.ToLower(strings.TrimSpace(parseKey))
	parseNormalizedKey = strings.NewReplacer(".", "_", "-", "_", " ", "_", "/", "_", "\\", "_").Replace(parseNormalizedKey)
	return parseNormalizedKey
}

// parseRedactLogString removes common token/credential patterns from one string value.
func parseRedactLogString(parseValue string) string {
	return parseScrubSecretString(parseValue)
}

// parseRedactLogAnyValue recursively redacts nested map/list structures used in slog.Any payloads.
func parseRedactLogAnyValue(parseValue any) any {
	return parseRedactLogAnyValueWithKeyPath("", parseValue)
}

// parseRedactLogAnyValueWithKeyPath recursively redacts nested slog.Any payloads while preserving the current nested key path.
func parseRedactLogAnyValueWithKeyPath(parseParentKeyPath string, parseValue any) any {
	switch parseTypedValue := parseValue.(type) {
	case nil:
		return nil
	case string:
		if parseShouldPreserveLogPathValue(parseParentKeyPath, parseTypedValue) {
			return strings.TrimSpace(parseTypedValue)
		}
		return parseRedactLogString(parseTypedValue)
	case error:
		return parseRedactLogString(parseTypedValue.Error())
	case map[string]any:
		parseRedactedMap := make(map[string]any, len(parseTypedValue))
		for parseMapKey, parseMapValue := range parseTypedValue {
			parseKeyPath := strings.TrimSpace(parseMapKey)
			if strings.TrimSpace(parseParentKeyPath) != "" {
				parseKeyPath = strings.TrimSpace(parseParentKeyPath) + "." + strings.TrimSpace(parseMapKey)
			}
			if parseIsSensitiveLogKey(parseMapKey) {
				parseRedactedMap[parseMapKey] = parseLogRedactionText
				continue
			}
			parseRedactedMap[parseMapKey] = parseRedactLogAnyValueWithKeyPath(parseKeyPath, parseMapValue)
		}
		return parseRedactedMap
	case []any:
		parseRedactedSlice := make([]any, 0, len(parseTypedValue))
		for _, parseSliceValue := range parseTypedValue {
			parseRedactedSlice = append(parseRedactedSlice, parseRedactLogAnyValueWithKeyPath(parseParentKeyPath, parseSliceValue))
		}
		return parseRedactedSlice
	case []string:
		parseRedactedSlice := make([]string, 0, len(parseTypedValue))
		for _, parseSliceValue := range parseTypedValue {
			if parseShouldPreserveLogPathValue(parseParentKeyPath, parseSliceValue) {
				parseRedactedSlice = append(parseRedactedSlice, strings.TrimSpace(parseSliceValue))
				continue
			}
			parseRedactedSlice = append(parseRedactedSlice, parseRedactLogString(parseSliceValue))
		}
		return parseRedactedSlice
	default:
		return parseTypedValue
	}
}
