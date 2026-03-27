package runtime2

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"slices"
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

type buildBinarySourceAnyMapEntry struct {
	getKey   string
	getValue any
}

type buildBinarySourceReflectMapEntry struct {
	getKey   string
	getValue reflect.Value
}

type buildBinarySourceAnyMapEntryCache struct {
	getEntries []buildBinarySourceAnyMapEntry
}

var storeBinarySourceMapKeyPool = sync.Pool{
	New: func() any {
		return &buildBinarySourceMapKeyCache{
			getKeys: make([]string, 0, 8),
		}
	},
}

var storeBinarySourceAnyMapEntryPool = sync.Pool{
	New: func() any {
		return &buildBinarySourceAnyMapEntryCache{
			getEntries: make([]buildBinarySourceAnyMapEntry, 0, 8),
		}
	},
}

// BuildBinarySourceValue encodes one supported source value for binary snapshot transport.
func BuildBinarySourceValue(parseValue any) ([]byte, error) {
	return buildBinarySourceValueInto(make([]byte, 0, 32), parseValue)
}

// buildBinarySourceValueAppendCapacityHint returns one coarse append-capacity hint for one source-value payload.
func buildBinarySourceValueAppendCapacityHint(parseValue any) int {
	switch getValue := parseValue.(type) {
	case nil, bool:
		return 1
	case int, int8, int16, int32, int64:
		return 9
	case uint, uint8, uint16, uint32, uint64, uintptr:
		return 9
	case float32, float64:
		return 9
	case string:
		return 5 + len(getValue)
	case []any:
		return 2 + len(getValue)*14
	case map[string]any:
		return 2 + len(getValue)*18
	default:
		return 9
	}
}

// buildBinarySourceReflectValueAppendCapacityHint returns one coarse append-capacity hint for one reflect source-value payload.
func buildBinarySourceReflectValueAppendCapacityHint(parseValue reflect.Value) int {
	if !parseValue.IsValid() {
		return 1
	}
	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return 1
		}
		return buildBinarySourceReflectValueAppendCapacityHint(parseValue.Elem())
	case reflect.Bool:
		return 1
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 9
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return 9
	case reflect.Float32, reflect.Float64:
		return 9
	case reflect.String:
		return 5 + parseValue.Len()
	case reflect.Slice, reflect.Array:
		return 2 + parseValue.Len()*14
	case reflect.Map:
		return 2 + parseValue.Len()*18
	default:
		return 9
	}
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
	parseCapacityHint := 2
	for _, parseItem := range parseList {
		parseCapacityHint += 4 + buildBinarySourceValueAppendCapacityHint(parseItem)
	}
	dst = slices.Grow(dst, parseCapacityHint)
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
	parseCapacityHint := 2
	parseEntries, parseEntryCache := buildBinarySourceAnyMapEntryBuffer(len(parseMap))
	for parseKey, parseValue := range parseMap {
		parseEntries = append(parseEntries, buildBinarySourceAnyMapEntry{
			getKey:   parseKey,
			getValue: parseValue,
		})
		parseCapacityHint += 2 + len(parseKey) + 4 + buildBinarySourceValueAppendCapacityHint(parseValue)
	}
	sortBinarySourceAnyMapEntries(parseEntries)
	defer storeBinarySourceAnyMapEntryBuffer(parseEntries, parseEntryCache)
	dst = slices.Grow(dst, parseCapacityHint)
	dst = append(dst, binarySourceValueKindMap, byte(len(parseEntries)))
	for _, parseEntry := range parseEntries {
		if len(parseEntry.getKey) > math.MaxUint16 {
			return nil, fmt.Errorf("runtime2: source map key %q is too large", parseEntry.getKey)
		}
		dst = binary.LittleEndian.AppendUint16(dst, uint16(len(parseEntry.getKey)))
		dst = append(dst, parseEntry.getKey...)
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var err error
		dst, err = buildBinarySourceValueInto(dst, parseEntry.getValue)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(dst[lenOff:], uint32(len(dst)-itemStart))
	}
	return dst, nil
}

// buildBinarySourceAnyMapEntryBuffer acquires one reusable map-entry buffer with at least the requested capacity.
func buildBinarySourceAnyMapEntryBuffer(parseMinimumCapacity int) ([]buildBinarySourceAnyMapEntry, *buildBinarySourceAnyMapEntryCache) {
	parseEntryCache, hasEntryCache := storeBinarySourceAnyMapEntryPool.Get().(*buildBinarySourceAnyMapEntryCache)
	if !hasEntryCache || parseEntryCache == nil {
		parseEntryCache = &buildBinarySourceAnyMapEntryCache{}
	}
	if cap(parseEntryCache.getEntries) < parseMinimumCapacity {
		parseEntryCache.getEntries = make([]buildBinarySourceAnyMapEntry, 0, parseMinimumCapacity)
	}
	return parseEntryCache.getEntries[:0], parseEntryCache
}

// storeBinarySourceAnyMapEntryBuffer clears one entry buffer and returns it to pool state.
func storeBinarySourceAnyMapEntryBuffer(
	parseEntries []buildBinarySourceAnyMapEntry,
	parseEntryCache *buildBinarySourceAnyMapEntryCache,
) {
	if parseEntryCache == nil {
		return
	}
	for parseIndex := range parseEntries {
		parseEntries[parseIndex].getKey = ""
		parseEntries[parseIndex].getValue = nil
	}
	parseEntryCache.getEntries = parseEntries[:0]
	storeBinarySourceAnyMapEntryPool.Put(parseEntryCache)
}

// sortBinarySourceAnyMapEntries sorts one map-entry slice by canonical key ordering.
func sortBinarySourceAnyMapEntries(parseEntries []buildBinarySourceAnyMapEntry) {
	if len(parseEntries) < 2 {
		return
	}
	// Small maps dominate runtime2 source payloads; insertion sort avoids sort.Slice closure overhead for tiny N.
	if len(parseEntries) <= 8 {
		for parseIndex := 1; parseIndex < len(parseEntries); parseIndex++ {
			parseEntry := parseEntries[parseIndex]
			parseWalk := parseIndex - 1
			for parseWalk >= 0 && parseEntries[parseWalk].getKey > parseEntry.getKey {
				parseEntries[parseWalk+1] = parseEntries[parseWalk]
				parseWalk--
			}
			parseEntries[parseWalk+1] = parseEntry
		}
		return
	}
	sort.Slice(parseEntries, func(parseLeft int, parseRight int) bool {
		return parseEntries[parseLeft].getKey < parseEntries[parseRight].getKey
	})
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
	parseCapacityHint := 2
	for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
		parseCapacityHint += 4 + buildBinarySourceReflectValueAppendCapacityHint(parseValue.Index(parseIndex))
	}
	dst = slices.Grow(dst, parseCapacityHint)
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
	parseMapEntries := make([]buildBinarySourceReflectMapEntry, 0, len(parseKeys))
	parseCapacityHint := 2
	for _, parseKey := range parseKeys {
		parseMapValue := parseValue.MapIndex(parseKey)
		parseKeyString := parseKey.String()
		parseMapEntries = append(parseMapEntries, buildBinarySourceReflectMapEntry{
			getKey:   parseKeyString,
			getValue: parseMapValue,
		})
		parseCapacityHint += 2 + len(parseKeyString) + 4 + buildBinarySourceReflectValueAppendCapacityHint(parseMapValue)
	}
	sort.Slice(parseMapEntries, func(parseLeft int, parseRight int) bool {
		return parseMapEntries[parseLeft].getKey < parseMapEntries[parseRight].getKey
	})
	dst = slices.Grow(dst, parseCapacityHint)
	dst = append(dst, binarySourceValueKindMap, byte(len(parseMapEntries)))
	for _, parseMapEntry := range parseMapEntries {
		parseKey := parseMapEntry.getKey
		if len(parseKey) > math.MaxUint16 {
			return nil, fmt.Errorf("runtime2: source map key %q is too large", parseKey)
		}
		dst = binary.LittleEndian.AppendUint16(dst, uint16(len(parseKey)))
		dst = append(dst, parseKey...)
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var err error
		dst, err = buildBinarySourceValueReflectInto(dst, parseMapEntry.getValue)
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
	return parseBinarySourceValueAt(parsePayload, 0, len(parsePayload))
}

// parseBinarySourceValueAt decodes one supported binary source value payload span from parseOffset for parseValueLength bytes.
func parseBinarySourceValueAt(parsePayload []byte, parseOffset int, parseValueLength int) (any, error) {
	if parseValueLength == 0 {
		return nil, fmt.Errorf("runtime2: binary source value payload is empty")
	}
	parseValueLimit := parseOffset + parseValueLength
	parseKind := parsePayload[parseOffset]
	switch parseKind {
	case binarySourceValueKindBoolTrue:
		return true, nil
	case binarySourceValueKindBoolFalse:
		return false, nil
	case binarySourceValueKindNumber:
		if parseValueLength != 9 {
			return nil, fmt.Errorf("runtime2: source-value number payload length %d is invalid", parseValueLength)
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(parsePayload[parseOffset+1 : parseOffset+9])), nil
	case binarySourceValueKindString:
		if parseValueLength < 5 {
			return nil, fmt.Errorf("runtime2: source-value string payload is truncated")
		}
		parseLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset+1 : parseOffset+5]))
		if parseLength > parseValueLength-5 {
			return nil, fmt.Errorf("runtime2: source-value string length %d exceeds payload size %d", parseLength, parseValueLength-5)
		}
		if 5+parseLength != parseValueLength {
			return nil, fmt.Errorf("runtime2: source-value string has %d trailing bytes", parseValueLength-(5+parseLength))
		}
		return string(parsePayload[parseOffset+5 : parseOffset+5+parseLength]), nil
	case binarySourceValueKindList:
		return parseBinarySourceListValue(parsePayload[parseOffset:parseValueLimit])
	case binarySourceValueKindNil:
		return nil, nil
	case binarySourceValueKindMap:
		return parseBinarySourceMapValue(parsePayload[parseOffset:parseValueLimit])
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
	parseList := make([]any, parseCount)
	parseOffset := 2
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseRemaining := len(parsePayload) - parseOffset
		if parseRemaining < 4 {
			return nil, fmt.Errorf("runtime2: decode list item[%d] length: payload is truncated", parseIndex)
		}
		parseItemLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset:]))
		parseOffset += 4
		parseRemaining -= 4
		if parseItemLength > parseRemaining {
			return nil, fmt.Errorf(
				"runtime2: decode list item[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseItemLength,
				parseRemaining,
			)
		}
		parseItemValue, parseErr := parseBinarySourceValueAt(parsePayload, parseOffset, parseItemLength)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode list item[%d]: %w", parseIndex, parseErr)
		}
		parseList[parseIndex] = parseItemValue
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
		parseRemaining := len(parsePayload) - parseOffset
		if parseRemaining < 2 {
			return nil, fmt.Errorf("runtime2: decode map key[%d]: key length is truncated", parseIndex)
		}
		parseKeyLength := int(binary.LittleEndian.Uint16(parsePayload[parseOffset:]))
		parseOffset += 2
		parseRemaining -= 2
		if parseKeyLength > parseRemaining {
			return nil, fmt.Errorf(
				"runtime2: decode map key[%d]: key length %d exceeds payload size %d",
				parseIndex,
				parseKeyLength,
				parseRemaining,
			)
		}
		parseKey := string(parsePayload[parseOffset : parseOffset+parseKeyLength])
		parseOffset += parseKeyLength
		parseRemaining -= parseKeyLength
		if parseIndex > 0 && parseKey <= parsePreviousKey {
			return nil, fmt.Errorf("runtime2: source-value map key %q is not in canonical order", parseKey)
		}
		if parseRemaining < 4 {
			return nil, fmt.Errorf("runtime2: decode map value[%d] length: payload is truncated", parseIndex)
		}
		parseValueLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset:]))
		parseOffset += 4
		parseRemaining -= 4
		if parseValueLength > parseRemaining {
			return nil, fmt.Errorf(
				"runtime2: decode map value[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseValueLength,
				parseRemaining,
			)
		}
		parseValue, parseErr := parseBinarySourceValueAt(parsePayload, parseOffset, parseValueLength)
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
