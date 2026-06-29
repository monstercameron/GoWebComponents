package runtime2

import (
	"fmt"
)

// RenderNodeRecordRaw stores the wire-format fields for one render-node record.
type RenderNodeRecordRaw struct {
	NodeID     uint64
	Kind       uint8
	ChildStart uint32
	ChildCount uint32
	PropStart  uint32
	PropCount  uint32
	TextRef    uint32
	Flags      uint32
	KeyHash    uint64
	KeyText    string
}

// RenderNodeRecord stores one validated render-node record.
type RenderNodeRecord struct {
	NodeID     uint64
	Kind       RenderNodeKind
	ChildStart uint32
	ChildCount uint32
	PropStart  uint32
	PropCount  uint32
	TextRef    uint32
	Flags      uint32
	KeyHash    uint64
	KeyText    string
}

// ParseRenderNodeRecord decodes and validates a raw render-node record.
func ParseRenderNodeRecord(parseRaw RenderNodeRecordRaw) (RenderNodeRecord, error) {
	parseKind, parseErr := ParseRenderNodeKind(parseRaw.Kind)
	if parseErr != nil {
		return RenderNodeRecord{}, parseErr
	}
	if _, parseErr = parseRenderNodeSpan(parseRaw.ChildStart, parseRaw.ChildCount); parseErr != nil {
		return RenderNodeRecord{}, fmt.Errorf("runtime2: child span is invalid: %w", parseErr)
	}
	if _, parseErr = parseRenderNodeSpan(parseRaw.PropStart, parseRaw.PropCount); parseErr != nil {
		return RenderNodeRecord{}, fmt.Errorf("runtime2: prop span is invalid: %w", parseErr)
	}
	hasRenderNodeKeyHash := parseRaw.KeyHash != 0
	hasRenderNodeKeyText := parseRuntimeHasTrimmedNonWhitespaceText(parseRaw.KeyText)
	if hasRenderNodeKeyHash != hasRenderNodeKeyText {
		return RenderNodeRecord{}, fmt.Errorf("runtime2: keyed metadata requires both non-zero key hash and non-empty key payload")
	}
	return RenderNodeRecord{
		NodeID:     parseRaw.NodeID,
		Kind:       parseKind,
		ChildStart: parseRaw.ChildStart,
		ChildCount: parseRaw.ChildCount,
		PropStart:  parseRaw.PropStart,
		PropCount:  parseRaw.PropCount,
		TextRef:    parseRaw.TextRef,
		Flags:      parseRaw.Flags,
		KeyHash:    parseRaw.KeyHash,
		KeyText:    parseRaw.KeyText,
	}, nil
}

// parseRenderNodeSpan validates one table span and returns the exclusive end.
func parseRenderNodeSpan(parseStart uint32, parseCount uint32) (uint32, error) {
	if parseCount == 0 {
		return parseStart, nil
	}
	parseEnd := uint64(parseStart) + uint64(parseCount)
	if parseEnd > uint64(^uint32(0)) {
		return 0, fmt.Errorf("span overflow start=%d count=%d", parseStart, parseCount)
	}
	return uint32(parseEnd), nil
}
