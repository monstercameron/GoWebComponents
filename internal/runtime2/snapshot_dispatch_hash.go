package runtime2

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
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

// buildSnapshotDispatchMapEntry stores one key/value pair for deterministic map hashing without follow-up map lookups.
type buildSnapshotDispatchMapEntry struct {
	getKey   string
	getValue any
}

// buildSnapshotDispatchMapEntryCache stores reusable map-entry capacity for deterministic map hashing.
type buildSnapshotDispatchMapEntryCache struct {
	getEntries []buildSnapshotDispatchMapEntry
}

var storeSnapshotDispatchMapEntryPool = sync.Pool{
	New: func() any {
		return &buildSnapshotDispatchMapEntryCache{
			getEntries: make([]buildSnapshotDispatchMapEntry, 0, 8),
		}
	},
}

var storeSnapshotDispatchHasherPool = sync.Pool{
	New: func() any {
		return sha256.New()
	},
}

// buildSnapshotDispatchHash computes one deterministic SHA-256 snapshot hash for host dispatch no-change checks.
func buildSnapshotDispatchHash(parseEnvelope SnapshotEnvelope) ([sha256.Size]byte, error) {
	return buildSnapshotDispatchHashStreamed(parseEnvelope)
}

// buildSnapshotDispatchHashInto computes one deterministic SHA-256 snapshot hash and returns reusable scratch bytes.
func buildSnapshotDispatchHashInto(parseEnvelope SnapshotEnvelope, parseScratch []byte) ([sha256.Size]byte, []byte, error) {
	return buildSnapshotDispatchHashIntoWithSourceIDs(parseEnvelope, nil, parseScratch)
}

// buildSnapshotDispatchHashIntoWithSourceIDs computes one deterministic SHA-256 snapshot hash using canonical source IDs when available and returns reusable scratch bytes.
func buildSnapshotDispatchHashIntoWithSourceIDs(parseEnvelope SnapshotEnvelope, parseSourceIDs []string, parseScratch []byte) ([sha256.Size]byte, []byte, error) {
	if parseScratch == nil {
		parseScratch = make([]byte, 0, 256)
	}
	parseScratch = parseScratch[:0]
	parsePayload, parsePayloadErr := appendSnapshotDispatchEnvelopeWithSourceIDs(parseScratch, parseEnvelope, parseSourceIDs)
	if parsePayloadErr != nil {
		return [sha256.Size]byte{}, parseScratch, parsePayloadErr
	}
	return sha256.Sum256(parsePayload), parsePayload[:0], nil
}

// buildSnapshotDispatchFastHashInto computes one deterministic fast dispatch hash and returns reusable scratch bytes.
func buildSnapshotDispatchFastHashInto(parseEnvelope SnapshotEnvelope, parseScratch []byte) (uint64, []byte, error) {
	return buildSnapshotDispatchFastHashIntoWithSourceIDs(parseEnvelope, nil, parseScratch)
}

// buildSnapshotDispatchFastHashIntoWithSourceIDs computes one deterministic fast dispatch hash using canonical source IDs when available and returns reusable scratch bytes.
func buildSnapshotDispatchFastHashIntoWithSourceIDs(parseEnvelope SnapshotEnvelope, parseSourceIDs []string, parseScratch []byte) (uint64, []byte, error) {
	if parseScratch == nil {
		parseScratch = make([]byte, 0, 256)
	}
	parseScratch = parseScratch[:0]
	parsePayload, parsePayloadErr := appendSnapshotDispatchEnvelopeWithSourceIDs(parseScratch, parseEnvelope, parseSourceIDs)
	if parsePayloadErr != nil {
		return 0, parseScratch, parsePayloadErr
	}
	return buildSnapshotDispatchFastHash(parsePayload), parsePayload[:0], nil
}

// buildSnapshotDispatchFastHash computes one deterministic non-cryptographic hash for dispatch no-change prefiltering.
func buildSnapshotDispatchFastHash(parsePayload []byte) uint64 {
	const (
		getDispatchFastHashOffset uint64 = 14695981039346656037
		getDispatchFastHashPrime  uint64 = 1099511628211
	)
	buildDispatchFastHash := getDispatchFastHashOffset
	for _, getPayloadByte := range parsePayload {
		buildDispatchFastHash ^= uint64(getPayloadByte)
		buildDispatchFastHash *= getDispatchFastHashPrime
	}
	if buildDispatchFastHash == 0 {
		return 1
	}
	return buildDispatchFastHash
}

// buildSnapshotDispatchHashStreamed computes one deterministic SHA-256 dispatch hash without staging payload bytes.
func buildSnapshotDispatchHashStreamed(parseEnvelope SnapshotEnvelope) ([sha256.Size]byte, error) {
	parseHasher := buildSnapshotDispatchHasher()
	if parseHasher == nil {
		return [sha256.Size]byte{}, fmt.Errorf("runtime2: snapshot dispatch hash hasher is nil")
	}
	parseHasher.Reset()
	defer storeSnapshotDispatchHasher(parseHasher)
	_, parseErr := writeSnapshotDispatchEnvelopeHash(parseHasher, nil, parseEnvelope)
	if parseErr != nil {
		return [sha256.Size]byte{}, parseErr
	}
	parseHash := [sha256.Size]byte{}
	_ = parseHasher.Sum(parseHash[:0])
	return parseHash, nil
}

// writeSnapshotDispatchEnvelopeHash writes one canonical dispatch envelope representation into a hasher and reuses parseScratch for nested value encoding.
func writeSnapshotDispatchEnvelopeHash(parseHasher hash.Hash, parseScratch []byte, parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := writeSnapshotDispatchHashByte(parseHasher, getSnapshotDispatchHashMarkerEnvelope); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchHashUint64(parseHasher, uint64(len(parseEnvelope.RegionInstanceID))); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchHashString(parseHasher, string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchHashUint64(parseHasher, parseEnvelope.Epoch); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchHashUint64(parseHasher, parseEnvelope.SourceVersion); parseErr != nil {
		return parseScratch, parseErr
	}
	parseValueBytes, parseValueErr := appendSnapshotDispatchValue(parseScratch[:0], parseEnvelope.Props)
	if parseValueErr != nil {
		return parseScratch, parseValueErr
	}
	if parseErr := writeSnapshotDispatchHashBytes(parseHasher, parseValueBytes); parseErr != nil {
		return parseValueBytes, parseErr
	}
	parseMapBytes, parseMapErr := appendSnapshotDispatchAnyMap(parseValueBytes[:0], parseEnvelope.Sources)
	if parseMapErr != nil {
		return parseValueBytes, parseMapErr
	}
	if parseErr := writeSnapshotDispatchHashBytes(parseHasher, parseMapBytes); parseErr != nil {
		return parseMapBytes, parseErr
	}
	return parseMapBytes, nil
}

// writeSnapshotDispatchHashByte writes one marker byte into the dispatch hash.
func writeSnapshotDispatchHashByte(parseHasher hash.Hash, parseMarker byte) error {
	var parseMarkerBuffer [1]byte
	parseMarkerBuffer[0] = parseMarker
	_, parseErr := parseHasher.Write(parseMarkerBuffer[:])
	return parseErr
}

// writeSnapshotDispatchHashUint64 writes one uint64 value into the dispatch hash using little-endian encoding.
func writeSnapshotDispatchHashUint64(parseHasher hash.Hash, parseValue uint64) error {
	var parseUintBuffer [8]byte
	binary.LittleEndian.PutUint64(parseUintBuffer[:], parseValue)
	_, parseErr := parseHasher.Write(parseUintBuffer[:])
	return parseErr
}

// writeSnapshotDispatchHashString writes one string payload into the dispatch hash without heap conversion.
func writeSnapshotDispatchHashString(parseHasher hash.Hash, parseValue string) error {
	var parseChunkBuffer [256]byte
	for parseValue != "" {
		parseChunkSize := copy(parseChunkBuffer[:], parseValue)
		if _, parseErr := parseHasher.Write(parseChunkBuffer[:parseChunkSize]); parseErr != nil {
			return parseErr
		}
		parseValue = parseValue[parseChunkSize:]
	}
	return nil
}

// writeSnapshotDispatchHashBytes writes one canonical payload byte slice into the dispatch hash.
func writeSnapshotDispatchHashBytes(parseHasher hash.Hash, parseBytes []byte) error {
	if len(parseBytes) == 0 {
		return nil
	}
	_, parseErr := parseHasher.Write(parseBytes)
	return parseErr
}

// buildSnapshotDispatchHasher acquires one reusable SHA-256 hasher.
func buildSnapshotDispatchHasher() hash.Hash {
	parseHasher, hasHasher := storeSnapshotDispatchHasherPool.Get().(hash.Hash)
	if hasHasher && parseHasher != nil {
		return parseHasher
	}
	return sha256.New()
}

// storeSnapshotDispatchHasher resets and returns one SHA-256 hasher to reuse pool state.
func storeSnapshotDispatchHasher(parseHasher hash.Hash) {
	if parseHasher == nil {
		return
	}
	parseHasher.Reset()
	storeSnapshotDispatchHasherPool.Put(parseHasher)
}

// appendSnapshotDispatchEnvelope appends one canonical envelope representation to parseDst.
func appendSnapshotDispatchEnvelope(parseDst []byte, parseEnvelope SnapshotEnvelope) ([]byte, error) {
	return appendSnapshotDispatchEnvelopeWithSourceIDs(parseDst, parseEnvelope, nil)
}

// appendSnapshotDispatchEnvelopeWithSourceIDs appends one canonical envelope representation and uses canonical source IDs to avoid source-map sorting when available.
func appendSnapshotDispatchEnvelopeWithSourceIDs(parseDst []byte, parseEnvelope SnapshotEnvelope, parseSourceIDs []string) ([]byte, error) {
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
	parseDst, parseErr = appendSnapshotDispatchSourceMap(parseDst, parseEnvelope.Sources, parseSourceIDs)
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
	parseEntries, parseEntryCache := buildSnapshotDispatchMapEntryBuffer(getMapLen)
	for parseKey, parseItem := range parseValue {
		parseEntries = append(parseEntries, buildSnapshotDispatchMapEntry{
			getKey:   parseKey,
			getValue: parseItem,
		})
	}
	defer storeSnapshotDispatchMapEntryBuffer(parseEntries, parseEntryCache)
	sort.Slice(parseEntries, func(getLeftIndex, getRightIndex int) bool {
		return parseEntries[getLeftIndex].getKey < parseEntries[getRightIndex].getKey
	})
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntries)))
	for _, parseEntry := range parseEntries {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntry.getKey)))
		parseDst = append(parseDst, parseEntry.getKey...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchValue(parseDst, parseEntry.getValue)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// appendSnapshotDispatchSourceMap appends one canonical source map and uses canonical source IDs to avoid map-key sorting when available.
func appendSnapshotDispatchSourceMap(parseDst []byte, parseValue map[string]any, parseSourceIDs []string) ([]byte, error) {
	parseOrderedDst, hasOrderedSourceMap, parseOrderedErr := appendSnapshotDispatchAnyMapWithOrderedKeys(parseDst, parseValue, parseSourceIDs)
	if parseOrderedErr != nil {
		return nil, parseOrderedErr
	}
	if hasOrderedSourceMap {
		return parseOrderedDst, nil
	}
	return appendSnapshotDispatchAnyMap(parseDst, parseValue)
}

// appendSnapshotDispatchAnyMapWithOrderedKeys appends one canonical map[string]any value in caller-provided key order when that order exactly covers the map.
func appendSnapshotDispatchAnyMapWithOrderedKeys(parseDst []byte, parseValue map[string]any, parseOrderedKeys []string) ([]byte, bool, error) {
	if parseValue == nil || len(parseOrderedKeys) == 0 || len(parseOrderedKeys) != len(parseValue) {
		return parseDst, false, nil
	}
	getBaseLen := len(parseDst)
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseOrderedKeys)))
	for _, parseKey := range parseOrderedKeys {
		getItem, hasOrderedItem := parseValue[parseKey]
		if !hasOrderedItem {
			return parseDst[:getBaseLen], false, nil
		}
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseKey)))
		parseDst = append(parseDst, parseKey...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchValue(parseDst, getItem)
		if parseErr != nil {
			return nil, false, parseErr
		}
	}
	return parseDst, true, nil
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
	type buildSnapshotDispatchReflectMapEntry struct {
		getKey   string
		getValue reflect.Value
	}
	parseEntries := make([]buildSnapshotDispatchReflectMapEntry, 0, getMapLen)
	parseMapIter := parseValue.MapRange()
	for parseMapIter.Next() {
		parseEntries = append(parseEntries, buildSnapshotDispatchReflectMapEntry{
			getKey:   parseMapIter.Key().String(),
			getValue: parseMapIter.Value(),
		})
	}
	sort.Slice(parseEntries, func(getLeftIndex, getRightIndex int) bool {
		return parseEntries[getLeftIndex].getKey < parseEntries[getRightIndex].getKey
	})
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntries)))
	for _, parseEntry := range parseEntries {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntry.getKey)))
		parseDst = append(parseDst, parseEntry.getKey...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchReflect(parseDst, parseEntry.getValue)
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

// buildSnapshotDispatchMapEntryBuffer acquires one reusable map-entry buffer with at least parseMinimumCapacity.
func buildSnapshotDispatchMapEntryBuffer(parseMinimumCapacity int) ([]buildSnapshotDispatchMapEntry, *buildSnapshotDispatchMapEntryCache) {
	parseEntryCache, hasEntryCache := storeSnapshotDispatchMapEntryPool.Get().(*buildSnapshotDispatchMapEntryCache)
	if !hasEntryCache || parseEntryCache == nil {
		parseEntryCache = &buildSnapshotDispatchMapEntryCache{}
	}
	if cap(parseEntryCache.getEntries) < parseMinimumCapacity {
		parseEntryCache.getEntries = make([]buildSnapshotDispatchMapEntry, 0, parseMinimumCapacity)
	}
	return parseEntryCache.getEntries[:0], parseEntryCache
}

// storeSnapshotDispatchMapEntryBuffer returns one cleared map-entry buffer to pool state.
func storeSnapshotDispatchMapEntryBuffer(
	parseEntries []buildSnapshotDispatchMapEntry,
	parseEntryCache *buildSnapshotDispatchMapEntryCache,
) {
	if parseEntryCache == nil {
		return
	}
	for parseIndex := range parseEntries {
		parseEntries[parseIndex].getKey = ""
		parseEntries[parseIndex].getValue = nil
	}
	parseEntryCache.getEntries = parseEntries[:0]
	storeSnapshotDispatchMapEntryPool.Put(parseEntryCache)
}
