package runtime2

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sort"
)

const (
	binarySourceValueKindBoolTrue  = byte(1)
	binarySourceValueKindBoolFalse = byte(2)
	binarySourceValueKindNumber    = byte(3)
	binarySourceValueKindString    = byte(4)
	binarySourceValueKindList      = byte(5)
	binarySourceValueKindNil       = byte(6)
	binarySourceValueKindMap       = byte(7)
	binarySourceValueListLimit     = 64
	binarySourceValueMapLimit      = 64
)

// BuildBinarySourceValue encodes one supported source value for binary snapshot transport.
func BuildBinarySourceValue(parseValue any) ([]byte, error) {
	return buildBinarySourceValueReflect(reflect.ValueOf(parseValue))
}

// buildBinarySourceValueReflect encodes one reflect value using the runtime2 binary value graph.
func buildBinarySourceValueReflect(parseValue reflect.Value) ([]byte, error) {
	if !parseValue.IsValid() {
		return []byte{binarySourceValueKindNil}, nil
	}
	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return []byte{binarySourceValueKindNil}, nil
		}
		return buildBinarySourceValueReflect(parseValue.Elem())
	case reflect.Bool:
		if parseValue.Bool() {
			return []byte{binarySourceValueKindBoolTrue}, nil
		}
		return []byte{binarySourceValueKindBoolFalse}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return buildBinarySourceNumberPayload(float64(parseValue.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return buildBinarySourceNumberPayload(float64(parseValue.Uint()))
	case reflect.Float32, reflect.Float64:
		return buildBinarySourceNumberPayload(parseValue.Convert(reflect.TypeOf(float64(0))).Float())
	case reflect.String:
		return buildBinarySourceStringPayload(parseValue.String())
	case reflect.Slice:
		if parseValue.IsNil() {
			return []byte{binarySourceValueKindNil}, nil
		}
		return buildBinarySourceListPayload(parseValue)
	case reflect.Array:
		return buildBinarySourceListPayload(parseValue)
	case reflect.Map:
		if parseValue.IsNil() {
			return []byte{binarySourceValueKindNil}, nil
		}
		return buildBinarySourceMapPayload(parseValue)
	case reflect.Struct:
		return buildBinarySourceStructPayload(parseValue)
	default:
		return nil, fmt.Errorf("runtime2: source value kind %s is unsupported", parseValue.Kind())
	}
}

// buildBinarySourceNumberPayload encodes one numeric value as a float64 payload.
func buildBinarySourceNumberPayload(parseValue float64) ([]byte, error) {
	parsePayload := make([]byte, 1+8)
	parsePayload[0] = binarySourceValueKindNumber
	binary.LittleEndian.PutUint64(parsePayload[1:9], math.Float64bits(parseValue))
	return parsePayload, nil
}

// buildBinarySourceStringPayload encodes one UTF-8 string payload.
func buildBinarySourceStringPayload(parseValue string) ([]byte, error) {
	if len(parseValue) > math.MaxUint32 {
		return nil, fmt.Errorf("runtime2: source string value is too large")
	}
	parsePayload := make([]byte, 1+4+len(parseValue))
	parsePayload[0] = binarySourceValueKindString
	binary.LittleEndian.PutUint32(parsePayload[1:5], uint32(len(parseValue)))
	copy(parsePayload[5:], parseValue)
	return parsePayload, nil
}

// buildBinarySourceListPayload encodes one list or array payload.
func buildBinarySourceListPayload(parseValue reflect.Value) ([]byte, error) {
	if parseValue.Len() > binarySourceValueListLimit {
		return nil, fmt.Errorf("runtime2: source list length %d exceeds limit %d", parseValue.Len(), binarySourceValueListLimit)
	}
	parsePayload := make([]byte, 0, 2+parseValue.Len()*5)
	parsePayload = append(parsePayload, binarySourceValueKindList, byte(parseValue.Len()))
	for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
		parseItemPayload, parseErr := buildBinarySourceValueReflect(parseValue.Index(parseIndex))
		if parseErr != nil {
			return nil, parseErr
		}
		parsePayload = appendBinaryUint32(parsePayload, uint32(len(parseItemPayload)))
		parsePayload = append(parsePayload, parseItemPayload...)
	}
	return parsePayload, nil
}

// buildBinarySourceMapPayload encodes one map payload with canonical key ordering.
func buildBinarySourceMapPayload(parseValue reflect.Value) ([]byte, error) {
	if parseValue.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("runtime2: source map key kind %s is unsupported", parseValue.Type().Key())
	}
	if parseValue.Len() > binarySourceValueMapLimit {
		return nil, fmt.Errorf("runtime2: source map length %d exceeds limit %d", parseValue.Len(), binarySourceValueMapLimit)
	}
	parseKeys := parseValue.MapKeys()
	parseKeyStrings := make([]string, 0, len(parseKeys))
	for _, parseKey := range parseKeys {
		parseKeyStrings = append(parseKeyStrings, parseKey.String())
	}
	sort.Strings(parseKeyStrings)
	parsePayload := make([]byte, 0, 2+len(parseKeyStrings)*7)
	parsePayload = append(parsePayload, binarySourceValueKindMap, byte(len(parseKeyStrings)))
	for _, parseKey := range parseKeyStrings {
		if len(parseKey) > math.MaxUint16 {
			return nil, fmt.Errorf("runtime2: source map key %q is too large", parseKey)
		}
		parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseKey)))
		parsePayload = append(parsePayload, parseKey...)
		parseValuePayload, parseErr := buildBinarySourceValueReflect(parseValue.MapIndex(reflect.ValueOf(parseKey)))
		if parseErr != nil {
			return nil, parseErr
		}
		parsePayload = appendBinaryUint32(parsePayload, uint32(len(parseValuePayload)))
		parsePayload = append(parsePayload, parseValuePayload...)
	}
	return parsePayload, nil
}

// buildBinarySourceStructPayload encodes one exported-field struct as a canonical map payload.
func buildBinarySourceStructPayload(parseValue reflect.Value) ([]byte, error) {
	parseFieldMap := make(map[string]any, parseValue.NumField())
	parseType := parseValue.Type()
	for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
		parseField := parseType.Field(parseIndex)
		if parseField.PkgPath != "" {
			continue
		}
		parseFieldMap[parseField.Name] = parseValue.Field(parseIndex).Interface()
	}
	return buildBinarySourceMapPayload(reflect.ValueOf(parseFieldMap))
}

// ParseBinarySourceValue decodes one supported binary source value payload.
func ParseBinarySourceValue(parsePayload []byte) (any, error) {
	if len(parsePayload) == 0 {
		return nil, fmt.Errorf("runtime2: binary source value payload is empty")
	}
	parseKind := parsePayload[0]
	switch parseKind {
	case binarySourceValueKindBoolTrue:
		return true, nil
	case binarySourceValueKindBoolFalse:
		return false, nil
	case binarySourceValueKindNumber:
		parseNumberBits, _, parseErr := parseBinaryUint64(parsePayload, 1, "source-value", "number")
		if parseErr != nil {
			return nil, parseErr
		}
		return math.Float64frombits(parseNumberBits), nil
	case binarySourceValueKindString:
		parseLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, 1, "source-value", "string length")
		if parseErr != nil {
			return nil, parseErr
		}
		parseSpan, _, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parseLength), "source-value", "string payload")
		if parseErr != nil {
			return nil, parseErr
		}
		return string(parseSpan), nil
	case binarySourceValueKindList:
		return parseBinarySourceListValue(parsePayload)
	case binarySourceValueKindNil:
		return nil, nil
	case binarySourceValueKindMap:
		return parseBinarySourceMapValue(parsePayload)
	default:
		return nil, fmt.Errorf("runtime2: binary source value kind %d is unsupported", parseKind)
	}
}

// parseBinarySourceListValue decodes one list payload from the binary source-value graph.
func parseBinarySourceListValue(parsePayload []byte) ([]any, error) {
	parseCountSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, 1, 1, "source-value", "list count")
	if parseErr != nil {
		return nil, parseErr
	}
	parseCount := int(parseCountSpan[0])
	parseList := make([]any, 0, parseCount)
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseItemLength, parseNextOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "source-value", "list item length")
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode list item[%d] length: %w", parseIndex, parseErr)
		}
		parseItemSpan, parseItemEnd, parseErr := parseBinaryPayloadSpan(parsePayload, parseNextOffset, int(parseItemLength), "source-value", "list item payload")
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode list item[%d] payload: %w", parseIndex, parseErr)
		}
		parseItemValue, parseErr := ParseBinarySourceValue(parseItemSpan)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode list item[%d]: %w", parseIndex, parseErr)
		}
		parseList = append(parseList, parseItemValue)
		parseOffset = parseItemEnd
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-value list has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseList, nil
}

// parseBinarySourceMapValue decodes one canonical map payload from the binary source-value graph.
func parseBinarySourceMapValue(parsePayload []byte) (map[string]any, error) {
	parseCountSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, 1, 1, "source-value", "map count")
	if parseErr != nil {
		return nil, parseErr
	}
	parseCount := int(parseCountSpan[0])
	parseMap := make(map[string]any, parseCount)
	parsePreviousKey := ""
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseKey, parseNextOffset, parseErr := parseBinaryString(parsePayload, parseOffset, "source-value", "map key")
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode map key[%d]: %w", parseIndex, parseErr)
		}
		if parseIndex > 0 && parseKey <= parsePreviousKey {
			return nil, fmt.Errorf("runtime2: source-value map key %q is not in canonical order", parseKey)
		}
		parseValueLength, parseValueOffset, parseErr := parseBinaryUint32(parsePayload, parseNextOffset, "source-value", "map value length")
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode map value[%d] length: %w", parseIndex, parseErr)
		}
		parseValueSpan, parseValueEnd, parseErr := parseBinaryPayloadSpan(parsePayload, parseValueOffset, int(parseValueLength), "source-value", "map value payload")
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode map value[%d] payload: %w", parseIndex, parseErr)
		}
		parseValue, parseErr := ParseBinarySourceValue(parseValueSpan)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode map value[%d]: %w", parseIndex, parseErr)
		}
		parseMap[parseKey] = parseValue
		parsePreviousKey = parseKey
		parseOffset = parseValueEnd
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-value map has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseMap, nil
}
