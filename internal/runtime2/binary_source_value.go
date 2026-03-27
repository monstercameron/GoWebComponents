package runtime2

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync"
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

type buildBinarySourceMapKeyCache struct {
	getKeys []string
}

var storeBinarySourceMapKeyPool = sync.Pool{
	New: func() any {
		return &buildBinarySourceMapKeyCache{
			getKeys: make([]string, 0, 8),
		}
	},
}

// BuildBinarySourceValue encodes one supported source value for binary snapshot transport.
func BuildBinarySourceValue(parseValue any) ([]byte, error) {
	return buildBinarySourceValueInto(make([]byte, 0, 32), parseValue)
}

// buildBinarySourceValueInto appends one encoded source value into dst and returns the extended slice.
func buildBinarySourceValueInto(dst []byte, parseValue any) ([]byte, error) {
	switch getValue := parseValue.(type) {
	case nil:
		return append(dst, binarySourceValueKindNil), nil
	case bool:
		if getValue {
			return append(dst, binarySourceValueKindBoolTrue), nil
		}
		return append(dst, binarySourceValueKindBoolFalse), nil
	case int:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case int8:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case int16:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case int32:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case int64:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case uint:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case uint8:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case uint16:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case uint32:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case uint64:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case uintptr:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case float32:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(getValue))), nil
	case float64:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(getValue)), nil
	case string:
		if len(getValue) > math.MaxUint32 {
			return nil, fmt.Errorf("runtime2: source string value is too large")
		}
		dst = append(dst, binarySourceValueKindString)
		dst = binary.LittleEndian.AppendUint32(dst, uint32(len(getValue)))
		return append(dst, getValue...), nil
	case []any:
		return buildBinarySourceAnyListInto(dst, getValue)
	case map[string]any:
		return buildBinarySourceAnyMapInto(dst, getValue)
	default:
		return buildBinarySourceValueReflectInto(dst, reflect.ValueOf(parseValue))
	}
}

// buildBinarySourceAnyListInto appends one []any list payload into dst and returns the extended slice.
func buildBinarySourceAnyListInto(dst []byte, parseList []any) ([]byte, error) {
	if parseList == nil {
		return append(dst, binarySourceValueKindNil), nil
	}
	if len(parseList) > binarySourceValueListLimit {
		return nil, fmt.Errorf("runtime2: source list length %d exceeds limit %d", len(parseList), binarySourceValueListLimit)
	}
	dst = append(dst, binarySourceValueKindList, byte(len(parseList)))
	for _, parseItem := range parseList {
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var err error
		dst, err = buildBinarySourceValueInto(dst, parseItem)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(dst[lenOff:], uint32(len(dst)-itemStart))
	}
	return dst, nil
}

// buildBinarySourceAnyMapInto appends one map[string]any payload into dst with canonical key ordering and returns the extended slice.
func buildBinarySourceAnyMapInto(dst []byte, parseMap map[string]any) ([]byte, error) {
	if parseMap == nil {
		return append(dst, binarySourceValueKindNil), nil
	}
	if len(parseMap) > binarySourceValueMapLimit {
		return nil, fmt.Errorf("runtime2: source map length %d exceeds limit %d", len(parseMap), binarySourceValueMapLimit)
	}
	parseKeyStrings, parseKeyCache := buildBinarySourceMapKeyBuffer(len(parseMap))
	for parseKey := range parseMap {
		parseKeyStrings = append(parseKeyStrings, parseKey)
	}
	sort.Strings(parseKeyStrings)
	defer storeBinarySourceMapKeyBuffer(parseKeyStrings, parseKeyCache)
	dst = append(dst, binarySourceValueKindMap, byte(len(parseKeyStrings)))
	for _, parseKey := range parseKeyStrings {
		if len(parseKey) > math.MaxUint16 {
			return nil, fmt.Errorf("runtime2: source map key %q is too large", parseKey)
		}
		dst = binary.LittleEndian.AppendUint16(dst, uint16(len(parseKey)))
		dst = append(dst, parseKey...)
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var err error
		dst, err = buildBinarySourceValueInto(dst, parseMap[parseKey])
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(dst[lenOff:], uint32(len(dst)-itemStart))
	}
	return dst, nil
}

// buildBinarySourceMapKeyBuffer acquires one reusable map-key buffer with at least the requested capacity.
func buildBinarySourceMapKeyBuffer(parseMinimumCapacity int) ([]string, *buildBinarySourceMapKeyCache) {
	parseKeyCache, hasKeyCache := storeBinarySourceMapKeyPool.Get().(*buildBinarySourceMapKeyCache)
	if !hasKeyCache || parseKeyCache == nil {
		parseKeyCache = &buildBinarySourceMapKeyCache{}
	}
	if cap(parseKeyCache.getKeys) < parseMinimumCapacity {
		parseKeyCache.getKeys = make([]string, 0, parseMinimumCapacity)
	}
	return parseKeyCache.getKeys[:0], parseKeyCache
}

// storeBinarySourceMapKeyBuffer returns one cleared key buffer to pool state.
func storeBinarySourceMapKeyBuffer(parseKeyStrings []string, parseKeyCache *buildBinarySourceMapKeyCache) {
	if parseKeyCache == nil {
		return
	}
	for parseIndex := range parseKeyStrings {
		parseKeyStrings[parseIndex] = ""
	}
	parseKeyCache.getKeys = parseKeyStrings[:0]
	storeBinarySourceMapKeyPool.Put(parseKeyCache)
}

// buildBinarySourceValueReflectInto appends one reflect-based encoded value into dst and returns the extended slice.
func buildBinarySourceValueReflectInto(dst []byte, parseValue reflect.Value) ([]byte, error) {
	if !parseValue.IsValid() {
		return append(dst, binarySourceValueKindNil), nil
	}
	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return append(dst, binarySourceValueKindNil), nil
		}
		return buildBinarySourceValueReflectInto(dst, parseValue.Elem())
	case reflect.Bool:
		if parseValue.Bool() {
			return append(dst, binarySourceValueKindBoolTrue), nil
		}
		return append(dst, binarySourceValueKindBoolFalse), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(parseValue.Int()))), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(float64(parseValue.Uint()))), nil
	case reflect.Float32, reflect.Float64:
		dst = append(dst, binarySourceValueKindNumber)
		return binary.LittleEndian.AppendUint64(dst, math.Float64bits(parseValue.Float())), nil
	case reflect.String:
		s := parseValue.String()
		if len(s) > math.MaxUint32 {
			return nil, fmt.Errorf("runtime2: source string value is too large")
		}
		dst = append(dst, binarySourceValueKindString)
		dst = binary.LittleEndian.AppendUint32(dst, uint32(len(s)))
		return append(dst, s...), nil
	case reflect.Slice:
		if parseValue.IsNil() {
			return append(dst, binarySourceValueKindNil), nil
		}
		return buildBinarySourceListInto(dst, parseValue)
	case reflect.Array:
		return buildBinarySourceListInto(dst, parseValue)
	case reflect.Map:
		if parseValue.IsNil() {
			return append(dst, binarySourceValueKindNil), nil
		}
		return buildBinarySourceMapInto(dst, parseValue)
	case reflect.Struct:
		return buildBinarySourceStructInto(dst, parseValue)
	default:
		return nil, fmt.Errorf("runtime2: source value kind %s is unsupported", parseValue.Kind())
	}
}

// buildBinarySourceListInto appends one reflect list or array payload into dst and returns the extended slice.
func buildBinarySourceListInto(dst []byte, parseValue reflect.Value) ([]byte, error) {
	if parseValue.Len() > binarySourceValueListLimit {
		return nil, fmt.Errorf("runtime2: source list length %d exceeds limit %d", parseValue.Len(), binarySourceValueListLimit)
	}
	dst = append(dst, binarySourceValueKindList, byte(parseValue.Len()))
	for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var err error
		dst, err = buildBinarySourceValueReflectInto(dst, parseValue.Index(parseIndex))
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(dst[lenOff:], uint32(len(dst)-itemStart))
	}
	return dst, nil
}

// buildBinarySourceMapInto appends one reflect map payload into dst with canonical key ordering and returns the extended slice.
func buildBinarySourceMapInto(dst []byte, parseValue reflect.Value) ([]byte, error) {
	if parseValue.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("runtime2: source map key kind %s is unsupported", parseValue.Type().Key())
	}
	if parseValue.Len() > binarySourceValueMapLimit {
		return nil, fmt.Errorf("runtime2: source map length %d exceeds limit %d", parseValue.Len(), binarySourceValueMapLimit)
	}
	parseKeys := parseValue.MapKeys()
	parseKeyStrings, parseKeyCache := buildBinarySourceMapKeyBuffer(len(parseKeys))
	for _, parseKey := range parseKeys {
		parseKeyStrings = append(parseKeyStrings, parseKey.String())
	}
	sort.Strings(parseKeyStrings)
	defer storeBinarySourceMapKeyBuffer(parseKeyStrings, parseKeyCache)
	dst = append(dst, binarySourceValueKindMap, byte(len(parseKeyStrings)))
	for _, parseKey := range parseKeyStrings {
		if len(parseKey) > math.MaxUint16 {
			return nil, fmt.Errorf("runtime2: source map key %q is too large", parseKey)
		}
		dst = binary.LittleEndian.AppendUint16(dst, uint16(len(parseKey)))
		dst = append(dst, parseKey...)
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var err error
		dst, err = buildBinarySourceValueReflectInto(dst, parseValue.MapIndex(reflect.ValueOf(parseKey)))
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(dst[lenOff:], uint32(len(dst)-itemStart))
	}
	return dst, nil
}

// buildBinarySourceStructInto appends one exported-field struct as a canonical map payload into dst and returns the extended slice.
func buildBinarySourceStructInto(dst []byte, parseValue reflect.Value) ([]byte, error) {
	parseFieldMap := make(map[string]any, parseValue.NumField())
	parseType := parseValue.Type()
	for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
		parseField := parseType.Field(parseIndex)
		if parseField.PkgPath != "" {
			continue
		}
		parseFieldMap[parseField.Name] = parseValue.Field(parseIndex).Interface()
	}
	return buildBinarySourceAnyMapInto(dst, parseFieldMap)
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
		if len(parsePayload) != 9 {
			return nil, fmt.Errorf("runtime2: source-value number payload length %d is invalid", len(parsePayload))
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(parsePayload[1:9])), nil
	case binarySourceValueKindString:
		if len(parsePayload) < 5 {
			return nil, fmt.Errorf("runtime2: source-value string payload is truncated")
		}
		parseLength := int(binary.LittleEndian.Uint32(parsePayload[1:5]))
		if parseLength > len(parsePayload)-5 {
			return nil, fmt.Errorf("runtime2: source-value string length %d exceeds payload size %d", parseLength, len(parsePayload)-5)
		}
		if 5+parseLength != len(parsePayload) {
			return nil, fmt.Errorf("runtime2: source-value string has %d trailing bytes", len(parsePayload)-(5+parseLength))
		}
		return string(parsePayload[5 : 5+parseLength]), nil
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
	if len(parsePayload) < 2 {
		return nil, fmt.Errorf("runtime2: source-value list payload is truncated")
	}
	parseCount := int(parsePayload[1])
	parseList := make([]any, 0, parseCount)
	parseOffset := 2
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		if parseOffset+4 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode list item[%d] length: payload is truncated", parseIndex)
		}
		parseItemLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
		parseOffset += 4
		if parseItemLength < 0 || parseItemLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode list item[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseItemLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseItemValue, parseErr := ParseBinarySourceValue(parsePayload[parseOffset : parseOffset+parseItemLength])
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode list item[%d]: %w", parseIndex, parseErr)
		}
		parseList = append(parseList, parseItemValue)
		parseOffset += parseItemLength
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-value list has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseList, nil
}

// parseBinarySourceMapValue decodes one canonical map payload from the binary source-value graph.
func parseBinarySourceMapValue(parsePayload []byte) (map[string]any, error) {
	if len(parsePayload) < 2 {
		return nil, fmt.Errorf("runtime2: source-value map payload is truncated")
	}
	parseCount := int(parsePayload[1])
	parseMap := make(map[string]any, parseCount)
	parsePreviousKey := ""
	parseOffset := 2
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		if parseOffset+2 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode map key[%d]: key length is truncated", parseIndex)
		}
		parseKeyLength := int(binary.LittleEndian.Uint16(parsePayload[parseOffset : parseOffset+2]))
		parseOffset += 2
		if parseKeyLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode map key[%d]: key length %d exceeds payload size %d",
				parseIndex,
				parseKeyLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseKey := string(parsePayload[parseOffset : parseOffset+parseKeyLength])
		parseOffset += parseKeyLength
		if parseIndex > 0 && parseKey <= parsePreviousKey {
			return nil, fmt.Errorf("runtime2: source-value map key %q is not in canonical order", parseKey)
		}
		if parseOffset+4 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode map value[%d] length: payload is truncated", parseIndex)
		}
		parseValueLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
		parseOffset += 4
		if parseValueLength < 0 || parseValueLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode map value[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseValueLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseValue, parseErr := ParseBinarySourceValue(parsePayload[parseOffset : parseOffset+parseValueLength])
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode map value[%d]: %w", parseIndex, parseErr)
		}
		parseMap[parseKey] = parseValue
		parsePreviousKey = parseKey
		parseOffset += parseValueLength
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-value map has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseMap, nil
}
