package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsage(t *testing.T) {
	var parseStdout bytes.Buffer
	var parseStderr bytes.Buffer

	if parseCode := run([]string{"sbom"}, &parseStdout, &parseStderr); parseCode != 2 {
		t.Fatalf("run() exit = %d, want 2", parseCode)
	}
	if parseStdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", parseStdout.String())
	}
	if !strings.Contains(parseStderr.String(), "usage: sbom") {
		t.Fatalf("stderr = %q, want usage", parseStderr.String())
	}
}

func TestRunWriteError(t *testing.T) {
	var parseStderr bytes.Buffer
	parseOut := filepath.Join(t.TempDir(), "missing", "sbom.json")

	if parseCode := run([]string{"sbom", parseOut, "."}, bytes.NewBuffer(nil), &parseStderr); parseCode != 1 {
		t.Fatalf("run() exit = %d, want 1", parseCode)
	}
	if !strings.Contains(parseStderr.String(), "sbom:") {
		t.Fatalf("stderr = %q, want prefixed error", parseStderr.String())
	}
}

func TestRunWritesSBOM(t *testing.T) {
	parseOut := filepath.Join(t.TempDir(), "sbom.json")
	var parseStdout bytes.Buffer
	var parseStderr bytes.Buffer

	if parseCode := run([]string{"sbom", parseOut, "."}, &parseStdout, &parseStderr); parseCode != 0 {
		t.Fatalf("run() exit = %d, want 0; stderr=%q", parseCode, parseStderr.String())
	}
	parseData, parseErr := os.ReadFile(parseOut)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if !strings.Contains(string(parseData), `"bomFormat": "CycloneDX"`) {
		t.Fatalf("SBOM output = %s, want CycloneDX document", string(parseData))
	}
	if !strings.Contains(parseStdout.String(), "wrote CycloneDX SBOM") {
		t.Fatalf("stdout = %q, want success message", parseStdout.String())
	}
}
