package devtools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ExportSnapshotJSON serializes a devtools snapshot into stable JSON.
func ExportSnapshotJSON(snapshot Snapshot) ([]byte, error) {
	return json.Marshal(snapshot)
}

// CompareSnapshots compares two snapshots and reports the top-level sections that changed.
func CompareSnapshots(previous, current Snapshot) (SnapshotComparison, error) {
	previousJSON, err := ExportSnapshotJSON(previous)
	if err != nil {
		return SnapshotComparison{}, err
	}
	currentJSON, err := ExportSnapshotJSON(current)
	if err != nil {
		return SnapshotComparison{}, err
	}

	comparison := SnapshotComparison{
		Equal:               bytes.Equal(previousJSON, currentJSON),
		PreviousFingerprint: fingerprint(previousJSON),
		CurrentFingerprint:  fingerprint(currentJSON),
		PreviousSize:        len(previousJSON),
		CurrentSize:         len(currentJSON),
	}
	if comparison.Equal {
		return comparison, nil
	}

	sections := []struct {
		name     string
		previous any
		current  any
	}{
		{name: "route", previous: previous.Route, current: current.Route},
		{name: "cache", previous: previous.Cache, current: current.Cache},
		{name: "multiClient", previous: previous.MultiClient, current: current.MultiClient},
		{name: "boundaries", previous: previous.Boundaries, current: current.Boundaries},
		{name: "coordination", previous: previous.Coordination, current: current.Coordination},
		{name: "extensions", previous: previous.Extensions, current: current.Extensions},
		{name: "tree", previous: previous.Tree, current: current.Tree},
		{name: "stats", previous: previous.Stats, current: current.Stats},
		{name: "profiling", previous: previous.Profiling, current: current.Profiling},
		{name: "hydration", previous: previous.Hydration, current: current.Hydration},
		{name: "diagnostics", previous: previous.Diagnostics, current: current.Diagnostics},
		{name: "logs", previous: previous.Logs, current: current.Logs},
	}

	for _, section := range sections {
		previousSection, err := json.Marshal(section.previous)
		if err != nil {
			return SnapshotComparison{}, fmt.Errorf("export previous %s section: %w", section.name, err)
		}
		currentSection, err := json.Marshal(section.current)
		if err != nil {
			return SnapshotComparison{}, fmt.Errorf("export current %s section: %w", section.name, err)
		}
		if !bytes.Equal(previousSection, currentSection) {
			comparison.ChangedSections = append(comparison.ChangedSections, section.name)
		}
	}

	return comparison, nil
}

func fingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
