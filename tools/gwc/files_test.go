package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolveFilesConfigNormalizesInputs(parseT *testing.T) {
	parseRoot := parseT.TempDir()

	parseConfig, parseErr := resolveFilesConfig(filesConfig{
		rootPath:    parseRoot,
		extensions:  []string{"js", ".TS", " js "},
		excludeDirs: []string{"node_modules", "Examples"},
		json:        true,
	})
	if parseErr != nil {
		parseT.Fatalf("resolve files config: %v", parseErr)
	}

	if parseConfig.rootPath != parseRoot {
		parseT.Fatalf("expected root %q, got %q", parseRoot, parseConfig.rootPath)
	}
	if !slices.Equal(parseConfig.extensions, []string{".js", ".ts"}) {
		parseT.Fatalf("unexpected normalized extensions: %#v", parseConfig.extensions)
	}
	if !slices.Equal(parseConfig.excludeDirs, []string{".git", "examples", "node_modules"}) {
		parseT.Fatalf("unexpected normalized exclude dirs: %#v", parseConfig.excludeDirs)
	}
	if !parseConfig.json {
		parseT.Fatalf("expected json mode to be preserved")
	}
}

func TestResolveFilesConfigRejectsInvalidInputs(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseFilePath := filepath.Join(parseRoot, "not-a-directory.txt")
	if parseErr := os.WriteFile(parseFilePath, []byte("file"), 0644); parseErr != nil {
		parseT.Fatalf("write file fixture: %v", parseErr)
	}

	parseTests := []struct {
		name   string
		config filesConfig
		want   string
	}{
		{
			name:   "root must be directory",
			config: filesConfig{rootPath: parseFilePath},
			want:   "files root is not a directory",
		},
		{
			name:   "extension must be plain",
			config: filesConfig{rootPath: parseRoot, extensions: []string{"scripts/app.js"}},
			want:   "extension filters must be plain file extensions",
		},
		{
			name:   "exclude-dir must be single name",
			config: filesConfig{rootPath: parseRoot, excludeDirs: []string{"foo/bar"}},
			want:   "exclude-dir values must be single directory names",
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			_, parseErr2 := resolveFilesConfig(parseTest.config)
			if parseErr2 == nil || !strings.Contains(parseErr2.Error(), parseTest.want) {
				parseT2.Fatalf("expected %q error, got %v", parseTest.want, parseErr2)
			}
		})
	}
}

func TestCollectFilesReportFiltersByExtensionAndExcludedDirs(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for parsePath, parseContent := range map[string]string{
		"app.js":                     "root js",
		"README.md":                  "docs",
		"nested/keep.JS":             "nested js",
		"nested/keep.txt":            "nested text",
		"examples/01-counter/app.js": "excluded example",
		"node_modules/pkg/index.js":  "excluded dependency",
		".git/hooks/post-checkout":   "excluded git",
	} {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFullPath), 0755); parseErr != nil {
			parseT.Fatalf("mkdir %q: %v", parsePath, parseErr)
		}
		if parseErr2 := os.WriteFile(parseFullPath, []byte(parseContent), 0644); parseErr2 != nil {
			parseT.Fatalf("write %q: %v", parsePath, parseErr2)
		}
	}

	parseConfig, parseErr3 := resolveFilesConfig(filesConfig{
		rootPath:    parseRoot,
		extensions:  []string{"js"},
		excludeDirs: []string{"examples", "node_modules"},
	})
	if parseErr3 != nil {
		parseT.Fatalf("resolve files config: %v", parseErr3)
	}

	parseReport, parseErr3 := collectFilesReport(parseConfig)
	if parseErr3 != nil {
		parseT.Fatalf("collect files report: %v", parseErr3)
	}

	parseExpected := []string{"app.js", "nested/keep.JS"}
	if !slices.Equal(parseReport.Files, parseExpected) {
		parseT.Fatalf("expected files %#v, got %#v", parseExpected, parseReport.Files)
	}
	if parseReport.Count != len(parseExpected) {
		parseT.Fatalf("expected count %d, got %d", len(parseExpected), parseReport.Count)
	}
}

func TestRunFilesOutputsJSONReport(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for parsePath, parseContent := range map[string]string{
		"keep.js":   "js",
		"ignore.go": "go",
	} {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFullPath), 0755); parseErr != nil {
			parseT.Fatalf("mkdir %q: %v", parsePath, parseErr)
		}
		if parseErr2 := os.WriteFile(parseFullPath, []byte(parseContent), 0644); parseErr2 != nil {
			parseT.Fatalf("write %q: %v", parsePath, parseErr2)
		}
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).runFiles([]string{"-root", parseRoot, "-ext", "js", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run files: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}

	var parseReport filesReport
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr5 != nil {
		parseT.Fatalf("decode files report: %v\noutput=%s", parseErr5, parseOutput)
	}
	if !parseReport.OK {
		parseT.Fatalf("expected ok report, got %#v", parseReport)
	}
	if parseReport.Count != 1 || !slices.Equal(parseReport.Files, []string{"keep.js"}) {
		parseT.Fatalf("unexpected report payload: %#v", parseReport)
	}
}
