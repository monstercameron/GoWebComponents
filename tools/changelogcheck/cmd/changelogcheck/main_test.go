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

	if parseCode := run([]string{"changelogcheck"}, &parseStdout, &parseStderr); parseCode != 2 {
		t.Fatalf("run() exit = %d, want 2", parseCode)
	}
	if parseStdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", parseStdout.String())
	}
	if !strings.Contains(parseStderr.String(), "usage: changelogcheck") {
		t.Fatalf("stderr = %q, want usage", parseStderr.String())
	}
}

func TestRunMissingFile(t *testing.T) {
	var parseStderr bytes.Buffer
	parsePath := filepath.Join(t.TempDir(), "missing.md")

	if parseCode := run([]string{"changelogcheck", parsePath, "v1.2.3"}, bytes.NewBuffer(nil), &parseStderr); parseCode != 2 {
		t.Fatalf("run() exit = %d, want 2", parseCode)
	}
	if !strings.Contains(parseStderr.String(), "changelogcheck:") {
		t.Fatalf("stderr = %q, want prefixed read error", parseStderr.String())
	}
}

func TestRunNoEntry(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if parseErr := os.WriteFile(parsePath, []byte("# Changelog\n\n## 1.2.2\n"), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}
	var parseStderr bytes.Buffer

	if parseCode := run([]string{"changelogcheck", parsePath, "v1.2.3"}, bytes.NewBuffer(nil), &parseStderr); parseCode != 1 {
		t.Fatalf("run() exit = %d, want 1", parseCode)
	}
	if !strings.Contains(parseStderr.String(), "no CHANGELOG entry found") {
		t.Fatalf("stderr = %q, want no-entry message", parseStderr.String())
	}
}

func TestRunFoundEntry(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if parseErr := os.WriteFile(parsePath, []byte("# Changelog\n\n## v1.2.3\n"), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}
	var parseStdout bytes.Buffer
	var parseStderr bytes.Buffer

	if parseCode := run([]string{"changelogcheck", parsePath, "1.2.3"}, &parseStdout, &parseStderr); parseCode != 0 {
		t.Fatalf("run() exit = %d, want 0; stderr=%q", parseCode, parseStderr.String())
	}
	if !strings.Contains(parseStdout.String(), "found CHANGELOG entry") {
		t.Fatalf("stdout = %q, want success message", parseStdout.String())
	}
	if parseStderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", parseStderr.String())
	}
}
