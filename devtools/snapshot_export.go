package devtools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/logging"
)

// ExportSnapshotJSON serializes a devtools snapshot into stable JSON.
func ExportSnapshotJSON(parseSnapshot Snapshot) ([]byte, error) {
	parseEncoded, parseErr := json.Marshal(parseSnapshot)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseGeneric any
	if parseErr2 := json.Unmarshal(parseEncoded, &parseGeneric); parseErr2 != nil {
		return nil, parseErr2
	}
	return json.Marshal(logging.RedactTelemetryValue("devtools.snapshot", parseGeneric))
}

// CompareSnapshots compares two snapshots and reports the top-level sections that changed.
func CompareSnapshots(parsePrevious, parseCurrent Snapshot) (SnapshotComparison, error) {
	parsePreviousJSON, parseErr := ExportSnapshotJSON(parsePrevious)
	if parseErr != nil {
		return SnapshotComparison{}, parseErr
	}
	parseCurrentJSON, parseErr := ExportSnapshotJSON(parseCurrent)
	if parseErr != nil {
		return SnapshotComparison{}, parseErr
	}

	parseComparison := SnapshotComparison{
		Equal:               bytes.Equal(parsePreviousJSON, parseCurrentJSON),
		PreviousFingerprint: fingerprint(parsePreviousJSON),
		CurrentFingerprint:  fingerprint(parseCurrentJSON),
		PreviousSize:        len(parsePreviousJSON),
		CurrentSize:         len(parseCurrentJSON),
	}
	if parseComparison.Equal {
		return parseComparison, nil
	}

	parseSections := []struct {
		name     string
		previous any
		current  any
	}{
		{name: "route", previous: parsePrevious.Route, current: parseCurrent.Route},
		{name: "cache", previous: parsePrevious.Cache, current: parseCurrent.Cache},
		{name: "multiClient", previous: parsePrevious.MultiClient, current: parseCurrent.MultiClient},
		{name: "boundaries", previous: parsePrevious.Boundaries, current: parseCurrent.Boundaries},
		{name: "coordination", previous: parsePrevious.Coordination, current: parseCurrent.Coordination},
		{name: "extensions", previous: parsePrevious.Extensions, current: parseCurrent.Extensions},
		{name: "tree", previous: parsePrevious.Tree, current: parseCurrent.Tree},
		{name: "stats", previous: parsePrevious.Stats, current: parseCurrent.Stats},
		{name: "profiling", previous: parsePrevious.Profiling, current: parseCurrent.Profiling},
		{name: "hydration", previous: parsePrevious.Hydration, current: parseCurrent.Hydration},
		{name: "diagnostics", previous: parsePrevious.Diagnostics, current: parseCurrent.Diagnostics},
		{name: "logs", previous: parsePrevious.Logs, current: parseCurrent.Logs},
	}

	for _, parseSection := range parseSections {
		parsePreviousSection, parseErr2 := json.Marshal(parseSection.previous)
		if parseErr2 != nil {
			return SnapshotComparison{}, fmt.Errorf("export previous %s section: %w", parseSection.name, parseErr2)
		}
		parseCurrentSection, parseErr2 := json.Marshal(parseSection.current)
		if parseErr2 != nil {
			return SnapshotComparison{}, fmt.Errorf("export current %s section: %w", parseSection.name, parseErr2)
		}
		if !bytes.Equal(parsePreviousSection, parseCurrentSection) {
			parseComparison.ChangedSections = append(parseComparison.ChangedSections, parseSection.name)
		}
	}

	return parseComparison, nil
}

func fingerprint(parseData []byte) string {
	parseSum := sha256.Sum256(parseData)
	return hex.EncodeToString(parseSum[:])
}
