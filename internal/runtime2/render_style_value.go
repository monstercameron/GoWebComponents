package runtime2

import (
	"fmt"
	"sort"
	"strings"
)

// FormatRenderStyleValue normalizes a supported style payload into a canonical string.
func FormatRenderStyleValue(parseRaw interface{}) (string, error) {
	switch parseStyleValue := parseRaw.(type) {
	case string:
		return formatRenderStyleString(parseStyleValue)
	case map[string]string:
		parseStyleMap := make(map[string]interface{}, len(parseStyleValue))
		for parseKey, parseValue := range parseStyleValue {
			parseStyleMap[parseKey] = parseValue
		}
		return formatRenderStyleMap(parseStyleMap)
	case map[string]interface{}:
		return formatRenderStyleMap(parseStyleValue)
	default:
		return "", fmt.Errorf("runtime2: unsupported style payload type %T", parseRaw)
	}
}

// formatRenderStyleString canonicalizes a style string into sorted key order.
func formatRenderStyleString(parseRaw string) (string, error) {
	parseStyleMap := make(map[string]interface{})
	parseSegments := strings.Split(parseRaw, ";")
	for _, parseSegment := range parseSegments {
		parseSegment = strings.TrimSpace(parseSegment)
		if parseSegment == "" {
			continue
		}
		parseParts := strings.SplitN(parseSegment, ":", 2)
		if len(parseParts) != 2 {
			return "", fmt.Errorf("runtime2: invalid style segment %q", parseSegment)
		}
		parseKey := strings.TrimSpace(parseParts[0])
		parseValue := strings.TrimSpace(parseParts[1])
		if parseKey == "" {
			return "", fmt.Errorf("runtime2: style key is required")
		}
		parseStyleMap[parseKey] = parseValue
	}
	return formatRenderStyleMap(parseStyleMap)
}

// formatRenderStyleMap canonicalizes one style map into sorted key order.
func formatRenderStyleMap(parseRaw map[string]interface{}) (string, error) {
	parseKeys := make([]string, 0, len(parseRaw))
	for parseKey := range parseRaw {
		if strings.TrimSpace(parseKey) == "" {
			return "", fmt.Errorf("runtime2: style key is required")
		}
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseSegments := make([]string, 0, len(parseKeys))
	for _, parseKey := range parseKeys {
		formatStyleScalarValue, formatStyleScalarValueErr := formatRenderStyleScalar(parseRaw[parseKey])
		if formatStyleScalarValueErr != nil {
			return "", fmt.Errorf("runtime2: style key %q is invalid: %w", parseKey, formatStyleScalarValueErr)
		}
		parseSegments = append(parseSegments, parseKey+":"+formatStyleScalarValue)
	}
	return strings.Join(parseSegments, ";"), nil
}

// formatRenderStyleScalar converts one supported style scalar value into a string.
func formatRenderStyleScalar(parseRaw interface{}) (string, error) {
	switch parseValue := parseRaw.(type) {
	case string:
		return parseValue, nil
	case bool:
		if parseValue {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", parseValue), nil
	case int8:
		return fmt.Sprintf("%d", parseValue), nil
	case int16:
		return fmt.Sprintf("%d", parseValue), nil
	case int32:
		return fmt.Sprintf("%d", parseValue), nil
	case int64:
		return fmt.Sprintf("%d", parseValue), nil
	case uint:
		return fmt.Sprintf("%d", parseValue), nil
	case uint8:
		return fmt.Sprintf("%d", parseValue), nil
	case uint16:
		return fmt.Sprintf("%d", parseValue), nil
	case uint32:
		return fmt.Sprintf("%d", parseValue), nil
	case uint64:
		return fmt.Sprintf("%d", parseValue), nil
	case float32:
		return fmt.Sprintf("%g", parseValue), nil
	case float64:
		return fmt.Sprintf("%g", parseValue), nil
	case nil:
		return "", fmt.Errorf("style value is nil")
	default:
		return "", fmt.Errorf("unsupported nested style shape %T", parseRaw)
	}
}
