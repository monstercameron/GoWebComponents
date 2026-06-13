package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type generatedStarterMetadata struct {
	Preset struct {
		Features []string `json:"features,omitempty"`
	} `json:"preset"`
	Enterprise struct {
		Features []string `json:"features,omitempty"`
	} `json:"enterprise"`
}

func readStarterFile(t *testing.T, relativePath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(relativePath))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return string(data)
}

func starterHasFeature(features []string, target string) bool {
	for _, feature := range features {
		if strings.EqualFold(strings.TrimSpace(feature), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func TestStarterMetadataIncludesSelectedFeatures(t *testing.T) {
	var metadata generatedStarterMetadata
	if err := json.Unmarshal([]byte(readStarterFile(t, "gwc-start.json")), &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	features := append([]string{}, metadata.Preset.Features...)
	features = append(features, metadata.Enterprise.Features...)
	for _, expected := range []string{
		"ssr",
		"hydration",
	} {
		if !starterHasFeature(features, expected) {
			t.Fatalf("expected scaffold metadata to record selected feature %q, got %v", expected, features)
		}
	}
}

func TestStarterFeatureMatrixMarksSelectedFeatures(t *testing.T) {
	matrix := readStarterFile(t, "FEATURE_MATRIX.md")
	for _, feature := range []string{
		"ssr",
		"hydration",
	} {
		if !strings.Contains(matrix, "- selected "+string(rune(96))+feature+string(rune(96))) {
			t.Fatalf("expected feature matrix to mark %q as selected", feature)
		}
	}
}

