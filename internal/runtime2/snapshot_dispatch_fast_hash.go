package runtime2

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"math"
	"reflect"
	"sort"
)

// buildSnapshotDispatchFastHashInto computes one deterministic fast dispatch hash and returns reusable scratch bytes.
func buildSnapshotDispatchFastHashInto(parseEnvelope SnapshotEnvelope, parseScratch []byte) (uint64, []byte, error) {
	return buildSnapshotDispatchFastHashIntoWithSourceIDs(parseEnvelope, nil, parseScratch)
}

// buildSnapshotDispatchFastHashIntoWithSourceIDs computes one deterministic fast dispatch hash using canonical source IDs when available and returns reusable scratch bytes.
func buildSnapshotDispatchFastHashIntoWithSourceIDs(parseEnvelope SnapshotEnvelope, parseSourceIDs []string, parseScratch []byte) (uint64, []byte, error) {
	return buildSnapshotDispatchFastHashIntoWithSourceAndPropsKeys(parseEnvelope, parseSourceIDs, nil, parseScratch)
}

// buildSnapshotDispatchFastHashIntoWithSourceAndPropsKeys computes one deterministic fast dispatch hash using canonical source and prop key order when available and returns reusable scratch bytes.
func buildSnapshotDispatchFastHashIntoWithSourceAndPropsKeys(
	parseEnvelope SnapshotEnvelope,
	parseSourceIDs []string,
	parsePropsOrderedKeys []string,
	parseScratch []byte,
) (uint64, []byte, error) {
	return buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout(
		parseEnvelope,
		parseSourceIDs,
		parsePropsOrderedKeys,
		nil,
		parseScratch,
	)
}

// buildSnapshotDispatchFastHashIntoWithSourceAndPropsEntries computes one deterministic fast dispatch hash using canonical source IDs and one pre-sorted props entry layout.
func buildSnapshotDispatchFastHashIntoWithSourceAndPropsEntries(
	parseEnvelope SnapshotEnvelope,
	parseSourceIDs []string,
	parsePropsEntries []buildSnapshotDispatchMapEntry,
	parseScratch []byte,
) (uint64, []byte, error) {
	return buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout(
		parseEnvelope,
		parseSourceIDs,
		nil,
		parsePropsEntries,
		parseScratch,
	)
}

// buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout computes one deterministic fast dispatch hash using canonical source IDs plus one optional compact props layout.
func buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout(
	parseEnvelope SnapshotEnvelope,
	parseSourceIDs []string,
	parsePropsOrderedKeys []string,
	parsePropsEntries []buildSnapshotDispatchMapEntry,
	parseScratch []byte,
) (uint64, []byte, error) {
	if parseScratch == nil {
		parseScratch = make([]byte, 0, 256)
	}
	parseScratch = parseScratch[:0]
	parseHasher := buildSnapshotDispatchFastHasher()
	if parseHasher == nil {
		return 0, parseScratch, fmt.Errorf("runtime2: snapshot dispatch fast hash hasher is nil")
	}
	parseHasher.Reset()
	defer storeSnapshotDispatchFastHasher(parseHasher)
	getUpdatedScratch, parseEnvelopeWriteErr := writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(
		parseHasher,
		parseEnvelope,
		parseSourceIDs,
		parsePropsOrderedKeys,
		parsePropsEntries,
		parseScratch,
	)
	if parseEnvelopeWriteErr != nil {
		return 0, getUpdatedScratch, parseEnvelopeWriteErr
	}
	parseScratch = getUpdatedScratch
	buildDispatchFastHash := parseHasher.Sum64()
	if buildDispatchFastHash == 0 {
		buildDispatchFastHash = 1
	}
	return buildDispatchFastHash, parseScratch[:0], nil
}

// buildSnapshotDispatchFastHash computes one deterministic non-cryptographic hash for dispatch no-change prefiltering.
func buildSnapshotDispatchFastHash(parsePayload []byte) uint64 {
	buildDispatchFastHash := maphash.Bytes(storeSnapshotDispatchFastHashSeed, parsePayload)
	if buildDispatchFastHash == 0 {
		return 1
	}
	return buildDispatchFastHash
}

// buildSnapshotDispatchFastHasher acquires one reusable maphash state for fast dispatch hashing.
func buildSnapshotDispatchFastHasher() *maphash.Hash {
	parseHasher, hasHasher := storeSnapshotDispatchFastHasherPool.Get().(*maphash.Hash)
	if hasHasher && parseHasher != nil {
		parseHasher.Reset()
		parseHasher.SetSeed(storeSnapshotDispatchFastHashSeed)
		return parseHasher
	}
	parseHasher = &maphash.Hash{}
	parseHasher.SetSeed(storeSnapshotDispatchFastHashSeed)
	return parseHasher
}

// storeSnapshotDispatchFastHasher resets and returns one reusable maphash state to pool storage.
func storeSnapshotDispatchFastHasher(parseHasher *maphash.Hash) {
	if parseHasher == nil {
		return
	}
	parseHasher.Reset()
	storeSnapshotDispatchFastHasherPool.Put(parseHasher)
}

// writeSnapshotDispatchFastHashByte writes one marker byte into the fast dispatch hash.
func writeSnapshotDispatchFastHashByte(parseHasher *maphash.Hash, parseMarker byte) error {
	var parseMarkerBuffer [1]byte
	parseMarkerBuffer[0] = parseMarker
	_, parseErr := parseHasher.Write(parseMarkerBuffer[:])
	return parseErr
}

// writeSnapshotDispatchFastHashUint64 writes one uint64 value into the fast dispatch hash using little-endian encoding.
func writeSnapshotDispatchFastHashUint64(parseHasher *maphash.Hash, parseValue uint64) error {
	var parseUintBuffer [8]byte
	binary.LittleEndian.PutUint64(parseUintBuffer[:], parseValue)
	_, parseErr := parseHasher.Write(parseUintBuffer[:])
	return parseErr
}

// writeSnapshotDispatchFastHashByteAndUint64 writes one marker byte plus one uint64 payload in a single hash write.
func writeSnapshotDispatchFastHashByteAndUint64(parseHasher *maphash.Hash, parseMarker byte, parseValue uint64) error {
	var parseBuffer [9]byte
	parseBuffer[0] = parseMarker
	binary.LittleEndian.PutUint64(parseBuffer[1:], parseValue)
	_, parseErr := parseHasher.Write(parseBuffer[:])
	return parseErr
}

// writeSnapshotDispatchFastHashByteAndByte writes one marker byte plus one payload byte in a single hash write.
func writeSnapshotDispatchFastHashByteAndByte(parseHasher *maphash.Hash, parseMarker byte, parseValue byte) error {
	var parseBuffer [2]byte
	parseBuffer[0] = parseMarker
	parseBuffer[1] = parseValue
	_, parseErr := parseHasher.Write(parseBuffer[:])
	return parseErr
}

// writeSnapshotDispatchFastHashString writes one string payload into the fast dispatch hash without heap conversion.
func writeSnapshotDispatchFastHashString(parseHasher *maphash.Hash, parseValue string) error {
	if parseValue == "" {
		return nil
	}
	_, parseErr := parseHasher.WriteString(parseValue)
	return parseErr
}

// writeSnapshotDispatchFastHashBytes writes one canonical payload byte slice into the fast dispatch hash.
func writeSnapshotDispatchFastHashBytes(parseHasher *maphash.Hash, parseBytes []byte) error {
	if len(parseBytes) == 0 {
		return nil
	}
	_, parseErr := parseHasher.Write(parseBytes)
	return parseErr
}

// writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys writes one canonical dispatch envelope into the fast hasher.
func writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(
	parseHasher *maphash.Hash,
	parseEnvelope SnapshotEnvelope,
	parseSourceIDs []string,
	parsePropsOrderedKeys []string,
	parsePropsEntries []buildSnapshotDispatchMapEntry,
	parseScratch []byte,
) ([]byte, error) {
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
		parseHasher,
		getSnapshotDispatchHashMarkerEnvelope,
		uint64(len(parseEnvelope.RegionInstanceID)),
	); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchFastHashString(parseHasher, string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchFastHashUint64(parseHasher, parseEnvelope.Epoch); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchFastHashUint64(parseHasher, parseEnvelope.SourceVersion); parseErr != nil {
		return parseScratch, parseErr
	}
	parseScratch, parsePropsErr := writeSnapshotDispatchFastHashPropsValue(
		parseHasher,
		parseEnvelope.Props,
		parsePropsOrderedKeys,
		parsePropsEntries,
		parseScratch,
	)
	if parsePropsErr != nil {
		return parseScratch, parsePropsErr
	}
	return writeSnapshotDispatchFastHashAnyMap(parseHasher, parseEnvelope.Sources, parseSourceIDs, parseScratch)
}

// writeSnapshotDispatchFastHashPropsValue writes one canonical props payload into the fast hasher and reuses ordered keys when available.
func writeSnapshotDispatchFastHashPropsValue(
	parseHasher *maphash.Hash,
	parseProps any,
	parseOrderedKeys []string,
	parseEntries []buildSnapshotDispatchMapEntry,
	parseScratch []byte,
) ([]byte, error) {
	if len(parseEntries) > 0 {
		return writeSnapshotDispatchFastHashAnyMapEntries(parseHasher, parseEntries, parseScratch)
	}
	parsePropsMap, hasPropsMap := parseProps.(map[string]any)
	if hasPropsMap {
		return writeSnapshotDispatchFastHashAnyMap(parseHasher, parsePropsMap, parseOrderedKeys, parseScratch)
	}
	return writeSnapshotDispatchFastHashValue(parseHasher, parseProps, parseScratch)
}

// writeSnapshotDispatchFastHashValue writes one canonical value payload into the fast hasher.
func writeSnapshotDispatchFastHashValue(
	parseHasher *maphash.Hash,
	parseValue any,
	parseScratch []byte,
) ([]byte, error) {
	switch getValue := parseValue.(type) {
	case nil:
		return parseScratch, writeSnapshotDispatchFastHashByte(parseHasher, getSnapshotDispatchHashMarkerNil)
	case bool:
		if getValue {
			return parseScratch, writeSnapshotDispatchFastHashByteAndByte(parseHasher, getSnapshotDispatchHashMarkerBool, 1)
		}
		return parseScratch, writeSnapshotDispatchFastHashByteAndByte(parseHasher, getSnapshotDispatchHashMarkerBool, 0)
	case int:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case int8:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case int16:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case int32:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case int64:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case uint:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case uint8:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case uint16:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case uint32:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case uint64:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case uintptr:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case float32:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, float64(getValue))
	case float64:
		return parseScratch, writeSnapshotDispatchFastHashFloat64(parseHasher, getValue)
	case string:
		if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
			parseHasher,
			getSnapshotDispatchHashMarkerString,
			uint64(len(getValue)),
		); parseErr != nil {
			return parseScratch, parseErr
		}
		return parseScratch, writeSnapshotDispatchFastHashString(parseHasher, getValue)
	case []bool:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []int:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []int8:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []int16:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []int32:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []int64:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []uint:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []uint8:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []uint16:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []uint32:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []uint64:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []uintptr:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []float32:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []float64:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []string:
		return writeSnapshotDispatchFastHashScalarList(parseHasher, getValue, parseScratch)
	case []any:
		if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
			parseHasher,
			getSnapshotDispatchHashMarkerList,
			uint64(len(getValue)),
		); parseErr != nil {
			return parseScratch, parseErr
		}
		for _, getItem := range getValue {
			var parseErr error
			parseScratch, parseErr = writeSnapshotDispatchFastHashValue(parseHasher, getItem, parseScratch)
			if parseErr != nil {
				return parseScratch, parseErr
			}
		}
		return parseScratch, nil
	case map[string]bool:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]int:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]int8:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]int16:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]int32:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]int64:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]uint:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]uint8:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]uint16:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]uint32:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]uint64:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]uintptr:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]float32:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]float64:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]string:
		return writeSnapshotDispatchFastHashScalarMap(parseHasher, getValue, parseScratch)
	case map[string]any:
		return writeSnapshotDispatchFastHashAnyMap(parseHasher, getValue, nil, parseScratch)
	default:
		parseEncodedValue, parseEncodeErr := appendSnapshotDispatchReflect(parseScratch[:0], reflect.ValueOf(parseValue))
		if parseEncodeErr != nil {
			return parseScratch, parseEncodeErr
		}
		if parseErr := writeSnapshotDispatchFastHashBytes(parseHasher, parseEncodedValue); parseErr != nil {
			return parseEncodedValue, parseErr
		}
		return parseEncodedValue[:0], nil
	}
}

// writeSnapshotDispatchFastHashScalarList writes one canonical typed scalar slice into the fast hasher without reflection fallback.
func writeSnapshotDispatchFastHashScalarList[T serializableScalar](
	parseHasher *maphash.Hash,
	parseValue []T,
	parseScratch []byte,
) ([]byte, error) {
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
		parseHasher,
		getSnapshotDispatchHashMarkerList,
		uint64(len(parseValue)),
	); parseErr != nil {
		return parseScratch, parseErr
	}
	for _, getItem := range parseValue {
		var parseErr error
		parseScratch, parseErr = writeSnapshotDispatchFastHashValue(parseHasher, any(getItem), parseScratch)
		if parseErr != nil {
			return parseScratch, parseErr
		}
	}
	return parseScratch, nil
}

// writeSnapshotDispatchFastHashScalarMap writes one canonical typed scalar map into the fast hasher without reflection fallback.
func writeSnapshotDispatchFastHashScalarMap[T serializableScalar](
	parseHasher *maphash.Hash,
	parseValue map[string]T,
	parseScratch []byte,
) ([]byte, error) {
	if parseValue == nil {
		return parseScratch, writeSnapshotDispatchFastHashByte(parseHasher, getSnapshotDispatchHashMarkerNil)
	}
	getMapLen := len(parseValue)
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
		parseHasher,
		getSnapshotDispatchHashMarkerMap,
		uint64(getMapLen),
	); parseErr != nil {
		return parseScratch, parseErr
	}
	if getMapLen == 0 {
		return parseScratch, nil
	}
	if getMapLen == 1 {
		return writeSnapshotDispatchFastHashScalarMapSingle(parseHasher, parseValue, parseScratch)
	}
	if getMapLen == 2 {
		return writeSnapshotDispatchFastHashScalarMapPair(parseHasher, parseValue, parseScratch)
	}
	parseEntries, parseEntryCache := buildSnapshotDispatchMapEntryBuffer(getMapLen)
	for parseKey, parseItem := range parseValue {
		parseEntries = append(parseEntries, buildSnapshotDispatchMapEntry{
			getKey:   parseKey,
			getValue: any(parseItem),
		})
	}
	defer storeSnapshotDispatchMapEntryBuffer(parseEntries, parseEntryCache)
	sort.Slice(parseEntries, func(parseLeft int, parseRight int) bool {
		return parseEntries[parseLeft].getKey < parseEntries[parseRight].getKey
	})
	for _, parseEntry := range parseEntries {
		var parseErr error
		parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, parseEntry.getKey, parseEntry.getValue, parseScratch)
		if parseErr != nil {
			return parseScratch, parseErr
		}
	}
	return parseScratch, nil
}

// writeSnapshotDispatchFastHashScalarMapSingle writes one canonical one-key typed scalar map into the fast hasher without sort overhead.
func writeSnapshotDispatchFastHashScalarMapSingle[T serializableScalar](
	parseHasher *maphash.Hash,
	parseValue map[string]T,
	parseScratch []byte,
) ([]byte, error) {
	for parseKey, getValue := range parseValue {
		return writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, parseKey, any(getValue), parseScratch)
	}
	return parseScratch, nil
}

// writeSnapshotDispatchFastHashScalarMapPair writes one canonical two-key typed scalar map into the fast hasher without pooled sorting.
func writeSnapshotDispatchFastHashScalarMapPair[T serializableScalar](
	parseHasher *maphash.Hash,
	parseValue map[string]T,
	parseScratch []byte,
) ([]byte, error) {
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
	getLeftValue := parseValue[getLeftKey]
	getRightValue := parseValue[getRightKey]
	var parseErr error
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, getLeftKey, any(getLeftValue), parseScratch)
	if parseErr != nil {
		return parseScratch, parseErr
	}
	return writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, getRightKey, any(getRightValue), parseScratch)
}

// writeSnapshotDispatchFastHashFloat64 writes one canonical finite numeric payload into the fast hasher.
func writeSnapshotDispatchFastHashFloat64(parseHasher *maphash.Hash, parseValue float64) error {
	if math.IsNaN(parseValue) || math.IsInf(parseValue, 0) {
		return fmt.Errorf("runtime2: snapshot dispatch hash unsupported non-finite number")
	}
	return writeSnapshotDispatchFastHashByteAndUint64(parseHasher, getSnapshotDispatchHashMarkerNumber, math.Float64bits(parseValue))
}

// writeSnapshotDispatchFastHashAnyMap writes one canonical map payload into the fast hasher and reuses ordered keys when available.
func writeSnapshotDispatchFastHashAnyMap(
	parseHasher *maphash.Hash,
	parseValue map[string]any,
	parseOrderedKeys []string,
	parseScratch []byte,
) ([]byte, error) {
	if parseValue == nil {
		return parseScratch, writeSnapshotDispatchFastHashByte(parseHasher, getSnapshotDispatchHashMarkerNil)
	}
	if len(parseOrderedKeys) > 0 && len(parseOrderedKeys) == len(parseValue) {
		return writeSnapshotDispatchFastHashAnyMapOrdered(parseHasher, parseValue, parseOrderedKeys, parseScratch)
	}
	getMapLen := len(parseValue)
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
		parseHasher,
		getSnapshotDispatchHashMarkerMap,
		uint64(getMapLen),
	); parseErr != nil {
		return parseScratch, parseErr
	}
	if getMapLen == 0 {
		return parseScratch, nil
	}
	if getMapLen == 1 {
		for parseKey, getValue := range parseValue {
			return writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, parseKey, getValue, parseScratch)
		}
		return parseScratch, nil
	}
	if getMapLen == 2 {
		return writeSnapshotDispatchFastHashAnyMapPair(parseHasher, parseValue, parseScratch)
	}
	parseEntries, parseEntryCache := buildSnapshotDispatchMapEntryBuffer(getMapLen)
	for parseKey, parseItem := range parseValue {
		parseEntries = append(parseEntries, buildSnapshotDispatchMapEntry{
			getKey:   parseKey,
			getValue: parseItem,
		})
	}
	defer storeSnapshotDispatchMapEntryBuffer(parseEntries, parseEntryCache)
	sort.Slice(parseEntries, func(parseLeft int, parseRight int) bool {
		return parseEntries[parseLeft].getKey < parseEntries[parseRight].getKey
	})
	for _, parseEntry := range parseEntries {
		var parseErr error
		parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, parseEntry.getKey, parseEntry.getValue, parseScratch)
		if parseErr != nil {
			return parseScratch, parseErr
		}
	}
	return parseScratch, nil
}

// writeSnapshotDispatchFastHashAnyMapEntries writes one canonical map payload from one pre-sorted key/value entry layout.
func writeSnapshotDispatchFastHashAnyMapEntries(
	parseHasher *maphash.Hash,
	parseEntries []buildSnapshotDispatchMapEntry,
	parseScratch []byte,
) ([]byte, error) {
	if len(parseEntries) == 0 {
		return parseScratch, writeSnapshotDispatchFastHashByte(parseHasher, getSnapshotDispatchHashMarkerNil)
	}
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
		parseHasher,
		getSnapshotDispatchHashMarkerMap,
		uint64(len(parseEntries)),
	); parseErr != nil {
		return parseScratch, parseErr
	}
	for _, parseEntry := range parseEntries {
		var parseErr error
		parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(
			parseHasher,
			parseEntry.getKey,
			parseEntry.getValue,
			parseScratch,
		)
		if parseErr != nil {
			return parseScratch, parseErr
		}
	}
	return parseScratch, nil
}

// writeSnapshotDispatchFastHashAnyMapOrdered writes one canonical map payload using caller-provided ordered keys.
func writeSnapshotDispatchFastHashAnyMapOrdered(
	parseHasher *maphash.Hash,
	parseValue map[string]any,
	parseOrderedKeys []string,
	parseScratch []byte,
) ([]byte, error) {
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(
		parseHasher,
		getSnapshotDispatchHashMarkerMap,
		uint64(len(parseOrderedKeys)),
	); parseErr != nil {
		return parseScratch, parseErr
	}
	for _, parseKey := range parseOrderedKeys {
		getValue, hasOrderedValue := parseValue[parseKey]
		if !hasOrderedValue {
			return parseScratch, fmt.Errorf("runtime2: snapshot dispatch ordered key %q is missing from map payload", parseKey)
		}
		var parseErr error
		parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, parseKey, getValue, parseScratch)
		if parseErr != nil {
			return parseScratch, parseErr
		}
	}
	return parseScratch, nil
}

// writeSnapshotDispatchFastHashAnyMapPair writes one canonical two-key map payload without pooled sorting.
func writeSnapshotDispatchFastHashAnyMapPair(
	parseHasher *maphash.Hash,
	parseValue map[string]any,
	parseScratch []byte,
) ([]byte, error) {
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
	getLeftValue := parseValue[getLeftKey]
	getRightValue := parseValue[getRightKey]
	var parseErr error
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, getLeftKey, getLeftValue, parseScratch)
	if parseErr != nil {
		return parseScratch, parseErr
	}
	return writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, getRightKey, getRightValue, parseScratch)
}

// writeSnapshotDispatchFastHashAnyMapEntry writes one key/value map entry into the fast hasher.
func writeSnapshotDispatchFastHashAnyMapEntry(
	parseHasher *maphash.Hash,
	parseKey string,
	parseValue any,
	parseScratch []byte,
) ([]byte, error) {
	if parseErr := writeSnapshotDispatchFastHashUint64(parseHasher, uint64(len(parseKey))); parseErr != nil {
		return parseScratch, parseErr
	}
	if parseErr := writeSnapshotDispatchFastHashString(parseHasher, parseKey); parseErr != nil {
		return parseScratch, parseErr
	}
	return writeSnapshotDispatchFastHashValue(parseHasher, parseValue, parseScratch)
}

// buildSnapshotDispatchHashStreamed computes one deterministic SHA-256 dispatch hash without staging payload bytes.
