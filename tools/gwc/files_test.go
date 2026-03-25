package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolveFilesConfigNormalizesInputs(t *testing.T) {
	root := t.TempDir()

	config, err := resolveFilesConfig(filesConfig{
		rootPath:    root,
		extensions:  []string{"js", ".TS", " js "},
		excludeDirs: []string{"node_modules", "Examples"},
		json:        true,
	})
	if err != nil {
		t.Fatalf("resolve files config: %v", err)
	}

	if config.rootPath != root {
		t.Fatalf("expected root %q, got %q", root, config.rootPath)
	}
	if !slices.Equal(config.extensions, []string{".js", ".ts"}) {
		t.Fatalf("unexpected normalized extensions: %#v", config.extensions)
	}
	if !slices.Equal(config.excludeDirs, []string{".git", "examples", "node_modules"}) {
		t.Fatalf("unexpected normalized exclude dirs: %#v", config.excludeDirs)
	}
	if !config.json {
		t.Fatalf("expected json mode to be preserved")
	}
}

func TestResolveFilesConfigRejectsInvalidInputs(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "not-a-directory.txt")
	if err := os.WriteFile(filePath, []byte("file"), 0644); err != nil {
		t.Fatalf("write file fixture: %v", err)
	}

	tests := []struct {
		name   string
		config filesConfig
		want   string
	}{
		{
			name:   "root must be directory",
			config: filesConfig{rootPath: filePath},
			want:   "files root is not a directory",
		},
		{
			name:   "extension must be plain",
			config: filesConfig{rootPath: root, extensions: []string{"scripts/app.js"}},
			want:   "extension filters must be plain file extensions",
		},
		{
			name:   "exclude-dir must be single name",
			config: filesConfig{rootPath: root, excludeDirs: []string{"foo/bar"}},
			want:   "exclude-dir values must be single directory names",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := resolveFilesConfig(test.config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestCollectFilesReportFiltersByExtensionAndExcludedDirs(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"app.js":                     "root js",
		"README.md":                  "docs",
		"nested/keep.JS":             "nested js",
		"nested/keep.txt":            "nested text",
		"examples/01-counter/app.js": "excluded example",
		"node_modules/pkg/index.js":  "excluded dependency",
		".git/hooks/post-checkout":   "excluded git",
	} {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	config, err := resolveFilesConfig(filesConfig{
		rootPath:    root,
		extensions:  []string{"js"},
		excludeDirs: []string{"examples", "node_modules"},
	})
	if err != nil {
		t.Fatalf("resolve files config: %v", err)
	}

	report, err := collectFilesReport(config)
	if err != nil {
		t.Fatalf("collect files report: %v", err)
	}

	expected := []string{"app.js", "nested/keep.JS"}
	if !slices.Equal(report.Files, expected) {
		t.Fatalf("expected files %#v, got %#v", expected, report.Files)
	}
	if report.Count != len(expected) {
		t.Fatalf("expected count %d, got %d", len(expected), report.Count)
	}
}

func TestRunFilesOutputsJSONReport(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"keep.js":   "js",
		"ignore.go": "go",
	} {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runFiles([]string{"-root", root, "-ext", "js", "-json"}); err != nil {
		t.Fatalf("run files: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	var report filesReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("decode files report: %v\noutput=%s", err, output)
	}
	if !report.OK {
		t.Fatalf("expected ok report, got %#v", report)
	}
	if report.Count != 1 || !slices.Equal(report.Files, []string{"keep.js"}) {
		t.Fatalf("unexpected report payload: %#v", report)
	}
}
