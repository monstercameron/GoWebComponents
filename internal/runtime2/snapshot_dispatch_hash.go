package runtime2

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync"
)

const (
	getSnapshotDispatchHashMarkerEnvelope = byte(1)
	getSnapshotDispatchHashMarkerNil      = byte(2)
	getSnapshotDispatchHashMarkerBool     = byte(3)
	getSnapshotDispatchHashMarkerNumber   = byte(4)
	getSnapshotDispatchHashMarkerString   = byte(5)
	getSnapshotDispatchHashMarkerList     = byte(6)
	getSnapshotDispatchHashMarkerMap      = byte(7)
	getSnapshotDispatchHashMarkerStruct   = byte(8)
)

// buildSnapshotDispatchMapKeyCache stores reusable map-key capacity for deterministic map hashing.
type buildSnapshotDispatchMapKeyCache struct {
	getKeys []string
}

var storeSnapshotDispatchMapKeyPool = sync.Pool{
	New: func() any {
		return &buildSnapshotDispatchMapKeyCache{
			getKeys: make([]string, 0, 8),
		}
	},
}

// buildSnapshotDispatchHash computes one deterministic SHA-256 snapshot hash for host dispatch no-change checks.
func buildSnapshotDispatchHash(parseEnvelope SnapshotEnvelope) ([sha256.Size]byte, error) {
	getDispatchHash, _, parseErr := buildSnapshotDispatchHashInto(parseEnvelope, nil)
	if parseErr != nil {
		return [sha256.Size]byte{}, parseErr
	}
	return getDispatchHash, nil
}

// buildSnapshotDispatchHashInto computes one deterministic SHA-256 snapshot hash and returns reusable scratch bytes.
func buildSnapshotDispatchHashInto(parseEnvelope SnapshotEnvelope, parseScratch []byte) ([sha256.Size]byte, []byte, error) {
	parseScratch = parseScratch[:0]
	parsePayload, parsePayloadErr := appendSnapshotDispatchEnvelope(parseScratch, parseEnvelope)
	if parsePayloadErr != nil {
		return [sha256.Size]byte{}, parseScratch, parsePayloadErr
	}
	return sha256.Sum256(parsePayload), parsePayload, nil
}

// appendSnapshotDispatchEnvelope appends one canonical envelope representation to parseDst.
func appendSnapshotDispatchEnvelope(parseDst []byte, parseEnvelope SnapshotEnvelope) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerEnvelope)
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEnvelope.RegionInstanceID)))
	parseDst = append(parseDst, string(parseEnvelope.RegionInstanceID)...)
	parseDst = appendSnapshotDispatchUint64(parseDst, parseEnvelope.Epoch)
	parseDst = appendSnapshotDispatchUint64(parseDst, parseEnvelope.SourceVersion)
	var parseErr error
	parseDst, parseErr = appendSnapshotDispatchValue(parseDst, parseEnvelope.Props)
	if parseErr != nil {
		return nil, parseErr
	}
	parseDst, parseErr = appendSnapshotDispatchAnyMap(parseDst, parseEnvelope.Sources)
	if parseErr != nil {
		return nil, parseErr
	}
	return parseDst, nil
}

// appendSnapshotDispatchValue appends one canonical any-typed value to parseDst.
func appendSnapshotDispatchValue(parseDst []byte, parseValue any) ([]byte, error) {
	switch getValue := parseValue.(type) {
	case nil:
		return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
	case bool:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerBool)
		if getValue {
			return append(parseDst, 1), nil
		}
		return append(parseDst, 0), nil
	case int:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int8:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int16:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int32:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int64:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint8:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint16:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint32:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint64:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uintptr:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case float32:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case float64:
		return appendSnapshotDispatchFloat64(parseDst, getValue)
	case string:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerString)
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getValue)))
		return append(parseDst, getValue...), nil
	case []any:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerList)
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getValue)))
		for _, getItem := range getValue {
			var parseErr error
			parseDst, parseErr = appendSnapshotDispatchValue(parseDst, getItem)
			if parseErr != nil {
				return nil, parseErr
			}
		}
		return parseDst, nil
	case map[string]any:
		return appendSnapshotDispatchAnyMap(parseDst, getValue)
	default:
		return appendSnapshotDispatchReflect(parseDst, reflect.ValueOf(parseValue))
	}
}

// appendSnapshotDispatchAnyMap appends one canonical map[string]any value to parseDst.
func appendSnapshotDispatchAnyMap(parseDst []byte, parseValue map[string]any) ([]byte, error) {
	if parseValue == nil {
		return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
	}
	getMapLen := len(parseValue)
	if getMapLen == 0 {
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
		return appendSnapshotDispatchUint64(parseDst, 0), nil
	}
	if getMapLen == 1 {
		return appendSnapshotDispatchAnyMapSingle(parseDst, parseValue)
	}
	if getMapLen == 2 {
		return appendSnapshotDispatchAnyMapPair(parseDst, parseValue)
	}
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseKeyBuffer, parseKeyCache := buildSnapshotDispatchMapKeyBuffer(getMapLen)
	for parseKey := range parseValue {
		parseKeyBuffer = append(parseKeyBuffer, parseKey)
	}
	sort.Strings(parseKeyBuffer)
	defer storeSnapshotDispatchMapKeyBuffer(parseKeyBuffer, parseKeyCache)
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKeyBuffer)))
	for _, parseKey := range parseKeyBuffer {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
		parseDst = append(parseDst, parseKey...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchValue(parseDst, parseValue[parseKey])
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// appendSnapshotDispatchAnyMapSingle appends one canonical one-key map[string]any value without sort overhead.
func appendSnapshotDispatchAnyMapSingle(parseDst []byte, parseValue map[string]any) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseDst = appendSnapshotDispatchUint64(parseDst, 1)
	for parseKey, getItem := range parseValue {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
		parseDst = append(parseDst, parseKey...)
		return appendSnapshotDispatchValue(parseDst, getItem)
	}
	return parseDst, nil
}

// appendSnapshotDispatchAnyMapPair appends one canonical two-key map[string]any value without pooled sorting.
func appendSnapshotDispatchAnyMapPair(parseDst []byte, parseValue map[string]any) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseDst = appendSnapshotDispatchUint64(parseDst, 2)
	var getLeftKey string
	var getRightKey string
	hasLeftKey := false
	for parseKey := range parseValue {
		if !hasLeftKey {
			getLeftKey = parseKey
			hasLeftKey = true
			continue
		}
		getRightKey = parseKey
		break
	}
	if getRightKey < getLeftKey {
		getLeftKey, getRightKey = getRightKey, getLeftKey
	}
	var parseErr error
	parseDst, parseErr = appendSnapshotDispatchAnyMapPairEntry(parseDst, getLeftKey, parseValue[getLeftKey])
	if parseErr != nil {
		return nil, parseErr
	}
	return appendSnapshotDispatchAnyMapPairEntry(parseDst, getRightKey, parseValue[getRightKey])
}

// appendSnapshotDispatchAnyMapPairEntry appends one key/value entry inside a two-key map fast path.
func appendSnapshotDispatchAnyMapPairEntry(parseDst []byte, parseKey string, parseValue any) ([]byte, error) {
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
	parseDst = append(parseDst, parseKey...)
	return appendSnapshotDispatchValue(parseDst, parseValue)
}

// appendSnapshotDispatchReflect appends one canonical reflect value to parseDst.
func appendSnapshotDispatchReflect(parseDst []byte, parseValue reflect.Value) ([]byte, error) {
	if !parseValue.IsValid() {
		return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
	}
	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
		}
		return appendSnapshotDispatchReflect(parseDst, parseValue.Elem())
	case reflect.Bool:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerBool)
		if parseValue.Bool() {
			return append(parseDst, 1), nil
		}
		return append(parseDst, 0), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return appendSnapshotDispatchFloat64(parseDst, float64(parseValue.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return appendSnapshotDispatchFloat64(parseDst, float64(parseValue.Uint()))
	case reflect.Float32, reflect.Float64:
		return appendSnapshotDispatchFloat64(parseDst, parseValue.Float())
	case reflect.String:
		getStringValue := parseValue.String()
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerString)
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getStringValue)))
		return append(parseDst, getStringValue...), nil
	case reflect.Slice:
		if parseValue.IsNil() {
			return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
		}
		return appendSnapshotDispatchReflectList(parseDst, parseValue)
	case reflect.Array:
		return appendSnapshotDispatchReflectList(parseDst, parseValue)
	case reflect.Map:
		if parseValue.IsNil() {
			return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
		}
		return appendSnapshotDispatchReflectMap(parseDst, parseValue)
	case reflect.Struct:
		return appendSnapshotDispatchReflectStruct(parseDst, parseValue)
	default:
		return nil, fmt.Errorf("runtime2: snapshot dispatch hash unsupported kind %s", parseValue.Kind())
	}
}

// appendSnapshotDispatchReflectList appends one canonical reflect list or array value to parseDst.
func appendSnapshotDispatchReflectList(parseDst []byte, parseValue reflect.Value) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerList)
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(parseValue.Len()))
	for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchReflect(parseDst, parseValue.Index(parseIndex))
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// appendSnapshotDispatchReflectMap appends one canonical reflect map value to parseDst.
func appendSnapshotDispatchReflectMap(parseDst []byte, parseValue reflect.Value) ([]byte, error) {
	if parseValue.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("runtime2: snapshot dispatch hash unsupported map key kind %s", parseValue.Type().Key())
	}
	getMapLen := parseValue.Len()
	if getMapLen == 0 {
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
		return appendSnapshotDispatchUint64(parseDst, 0), nil
	}
	if getMapLen == 1 {
		return appendSnapshotDispatchReflectMapSingle(parseDst, parseValue)
	}
	if getMapLen == 2 {
		return appendSnapshotDispatchReflectMapPair(parseDst, parseValue)
	}
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseKeyBuffer, parseKeyCache := buildSnapshotDispatchMapKeyBuffer(getMapLen)
	parseMapIter := parseValue.MapRange()
	for parseMapIter.Next() {
		parseKeyBuffer = append(parseKeyBuffer, parseMapIter.Key().String())
	}
	sort.Strings(parseKeyBuffer)
	defer storeSnapshotDispatchMapKeyBuffer(parseKeyBuffer, parseKeyCache)
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKeyBuffer)))
	for _, parseKey := range parseKeyBuffer {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
		parseDst = append(parseDst, parseKey...)
		parseMapValue := parseValue.MapIndex(reflect.ValueOf(parseKey))
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchReflect(parseDst, parseMapValue)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// appendSnapshotDispatchReflectMapSingle appends one canonical one-key reflect map value without sort overhead.
func appendSnapshotDispatchReflectMapSingle(parseDst []byte, parseValue reflect.Value) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseDst = appendSnapshotDispatchUint64(parseDst, 1)
	parseMapIter := parseValue.MapRange()
	if !parseMapIter.Next() {
		return parseDst, nil
	}
	parseKey := parseMapIter.Key().String()
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
	parseDst = append(parseDst, parseKey...)
	return appendSnapshotDispatchReflect(parseDst, parseMapIter.Value())
}

// appendSnapshotDispatchReflectMapPair appends one canonical two-key reflect map value without pooled sorting.
func appendSnapshotDispatchReflectMapPair(parseDst []byte, parseValue reflect.Value) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseDst = appendSnapshotDispatchUint64(parseDst, 2)
	parseMapIter := parseValue.MapRange()
	if !parseMapIter.Next() {
		return parseDst, nil
	}
	getLeftKey := parseMapIter.Key().String()
	getLeftValue := parseMapIter.Value()
	if !parseMapIter.Next() {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getLeftKey)))
		parseDst = append(parseDst, getLeftKey...)
		return appendSnapshotDispatchReflect(parseDst, getLeftValue)
	}
	getRightKey := parseMapIter.Key().String()
	getRightValue := parseMapIter.Value()
	if getRightKey < getLeftKey {
		getLeftKey, getRightKey = getRightKey, getLeftKey
		getLeftValue, getRightValue = getRightValue, getLeftValue
	}
	var parseErr error
	parseDst, parseErr = appendSnapshotDispatchReflectMapPairEntry(parseDst, getLeftKey, getLeftValue)
	if parseErr != nil {
		return nil, parseErr
	}
	return appendSnapshotDispatchReflectMapPairEntry(parseDst, getRightKey, getRightValue)
}

// appendSnapshotDispatchReflectMapPairEntry appends one key/value entry inside a two-key reflect map fast path.
func appendSnapshotDispatchReflectMapPairEntry(parseDst []byte, parseKey string, parseValue reflect.Value) ([]byte, error) {
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
	parseDst = append(parseDst, parseKey...)
	return appendSnapshotDispatchReflect(parseDst, parseValue)
}

// appendSnapshotDispatchReflectStruct appends one canonical reflect struct value to parseDst.
func appendSnapshotDispatchReflectStruct(parseDst []byte, parseValue reflect.Value) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerStruct)
	parseValueType := parseValue.Type()
	parseFieldCount := 0
	for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
		if parseValueType.Field(parseIndex).PkgPath == "" {
			parseFieldCount++
		}
	}
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(parseFieldCount))
	for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
		parseField := parseValueType.Field(parseIndex)
		if parseField.PkgPath != "" {
			continue
		}
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseField.Name)))
		parseDst = append(parseDst, parseField.Name...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchReflect(parseDst, parseValue.Field(parseIndex))
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// appendSnapshotDispatchFloat64 appends one canonical finite numeric value to parseDst.
func appendSnapshotDispatchFloat64(parseDst []byte, parseValue float64) ([]byte, error) {
	if math.IsNaN(parseValue) || math.IsInf(parseValue, 0) {
		return nil, fmt.Errorf("runtime2: snapshot dispatch hash unsupported non-finite number")
	}
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerNumber)
	return appendSnapshotDispatchUint64(parseDst, math.Float64bits(parseValue)), nil
}

// appendSnapshotDispatchUint64 appends one uint64 value to parseDst.
func appendSnapshotDispatchUint64(parseDst []byte, parseValue uint64) []byte {
	return binary.LittleEndian.AppendUint64(parseDst, parseValue)
}

// buildSnapshotDispatchMapKeyBuffer acquires one reusable key buffer with at least parseMinimumCapacity.
func buildSnapshotDispatchMapKeyBuffer(parseMinimumCapacity int) ([]string, *buildSnapshotDispatchMapKeyCache) {
	parseKeyCache, hasKeyCache := storeSnapshotDispatchMapKeyPool.Get().(*buildSnapshotDispatchMapKeyCache)
	if !hasKeyCache || parseKeyCache == nil {
		parseKeyCache = &buildSnapshotDispatchMapKeyCache{}
	}
	if cap(parseKeyCache.getKeys) < parseMinimumCapacity {
		parseKeyCache.getKeys = make([]string, 0, parseMinimumCapacity)
	}
	return parseKeyCache.getKeys[:0], parseKeyCache
}

// storeSnapshotDispatchMapKeyBuffer returns one cleared key buffer to pool state.
func storeSnapshotDispatchMapKeyBuffer(parseKeyBuffer []string, parseKeyCache *buildSnapshotDispatchMapKeyCache) {
	if parseKeyCache == nil {
		return
	}
	for parseIndex := range parseKeyBuffer {
		parseKeyBuffer[parseIndex] = ""
	}
	parseKeyCache.getKeys = parseKeyBuffer[:0]
	storeSnapshotDispatchMapKeyPool.Put(parseKeyCache)
}
