package runtime2

import (
	"fmt"
	"sort"
	"strings"
)

// RenderPropKind identifies the property encoding category in render IR.
type RenderPropKind uint8

const (
	renderPropKindInvalid RenderPropKind = iota

	RenderPropKindClass
	RenderPropKindStyle
	RenderPropKindAria
	RenderPropKindData
	RenderPropKindTextAdjacent
)

// RenderPropRecordRaw stores the wire-format fields for one render prop record.
type RenderPropRecordRaw struct {
	Kind     uint8
	KeyRef   uint32
	ValueRef uint32
}

// RenderPropRecord stores one validated render prop record.
type RenderPropRecord struct {
	Kind  RenderPropKind
	Key   string
	Value string
}

// ParseRenderPropKind decodes and validates one raw render-prop kind.
func ParseRenderPropKind(parseRaw uint8) (RenderPropKind, error) {
	parseKind := RenderPropKind(parseRaw)
	switch parseKind {
	case RenderPropKindClass, RenderPropKindStyle, RenderPropKindAria, RenderPropKindData, RenderPropKindTextAdjacent:
		return parseKind, nil
	case renderPropKindInvalid:
		return renderPropKindInvalid, fmt.Errorf("runtime2: render prop kind %d is invalid", parseRaw)
	default:
		return renderPropKindInvalid, fmt.Errorf("runtime2: render prop kind %d is unsupported", parseRaw)
	}
}

// ParseRenderPropRecord decodes and validates one render prop record against the string table.
func ParseRenderPropRecord(parseRaw RenderPropRecordRaw, parseStringTable RenderStringTable) (RenderPropRecord, error) {
	parseKind, parseErr := ParseRenderPropKind(parseRaw.Kind)
	if parseErr != nil {
		return RenderPropRecord{}, parseErr
	}
	getKey, getKeyErr := parseStringTable.GetRenderStringByRef(parseRaw.KeyRef)
	if getKeyErr != nil {
		return RenderPropRecord{}, fmt.Errorf("runtime2: key reference %d is invalid: %w", parseRaw.KeyRef, getKeyErr)
	}
	if strings.TrimSpace(getKey) == "" {
		return RenderPropRecord{}, fmt.Errorf("runtime2: key reference %d resolved to an empty key", parseRaw.KeyRef)
	}
	getValue, getValueErr := parseStringTable.GetRenderStringByRef(parseRaw.ValueRef)
	if getValueErr != nil {
		return RenderPropRecord{}, fmt.Errorf("runtime2: value reference %d is invalid: %w", parseRaw.ValueRef, getValueErr)
	}
	return RenderPropRecord{
		Kind:  parseKind,
		Key:   getKey,
		Value: getValue,
	}, nil
}

// ParseRenderPropRecords decodes, canonicalizes, and validates a list of render prop records.
func ParseRenderPropRecords(parseRawRecords []RenderPropRecordRaw, parseStringTable RenderStringTable) ([]RenderPropRecord, error) {
	parseRecords := make([]RenderPropRecord, 0, len(parseRawRecords))
	for parseIndex, parseRawRecord := range parseRawRecords {
		parseRecord, parseErr := ParseRenderPropRecord(parseRawRecord, parseStringTable)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: prop record %d is invalid: %w", parseIndex, parseErr)
		}
		parseRecords = append(parseRecords, parseRecord)
	}
	sort.Slice(parseRecords, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftRecord := parseRecords[parseLeftIndex]
		parseRightRecord := parseRecords[parseRightIndex]
		if parseLeftRecord.Key == parseRightRecord.Key {
			return parseLeftRecord.Kind < parseRightRecord.Kind
		}
		return parseLeftRecord.Key < parseRightRecord.Key
	})
	for parseIndex := 1; parseIndex < len(parseRecords); parseIndex++ {
		if parseRecords[parseIndex-1].Key == parseRecords[parseIndex].Key {
			return nil, fmt.Errorf("runtime2: duplicate prop key %q", parseRecords[parseIndex].Key)
		}
	}
	return parseRecords, nil
}
