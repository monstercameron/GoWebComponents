package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunTemplateLoweringRequiresInput verifies CLI validation for missing input.
func TestRunTemplateLoweringRequiresInput(parseT *testing.T) {
	parseErr := runTemplateLowering(nil, &bytes.Buffer{}, &bytes.Buffer{})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "-input is required") {
		parseT.Fatalf("expected missing input error, got %v", parseErr)
	}
}

// TestRunTemplateLoweringWritesStdoutAndFile verifies lowered output emission to stdout and files.
func TestRunTemplateLoweringWritesStdoutAndFile(parseT *testing.T) {
	parseTemplatePath := filepath.Join(parseT.TempDir(), "landing.template.html")
	parseTemplate := `<section><h1>{{.Headline}}</h1><p>{{.Summary}}</p></section>`
	if parseErr := os.WriteFile(parseTemplatePath, []byte(parseTemplate), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile(template): %v", parseErr)
	}

	var parseStdout bytes.Buffer
	var parseStderr bytes.Buffer
	parseArgs := []string{"-input", parseTemplatePath, "-package", "demo", "-struct", "LandingProps", "-func", "RenderLanding"}
	if parseErr2 := runTemplateLowering(parseArgs, &parseStdout, &parseStderr); parseErr2 != nil {
		parseT.Fatalf("runTemplateLowering(stdout): %v stderr=%s", parseErr2, parseStderr.String())
	}
	for _, parseNeedle := range []string{"package demo", "type LandingProps struct", "Headline", "Summary", "func RenderLanding"} {
		if !strings.Contains(parseStdout.String(), parseNeedle) {
			parseT.Fatalf("expected stdout to contain %q\n%s", parseNeedle, parseStdout.String())
		}
	}

	parseOutputPath := filepath.Join(parseT.TempDir(), "generated.go")
	if parseErr3 := runTemplateLowering(append(parseArgs, "-output", parseOutputPath), &bytes.Buffer{}, &bytes.Buffer{}); parseErr3 != nil {
		parseT.Fatalf("runTemplateLowering(file): %v", parseErr3)
	}
	parseOutputBytes, parseErr4 := os.ReadFile(parseOutputPath)
	if parseErr4 != nil {
		parseT.Fatalf("ReadFile(output): %v", parseErr4)
	}
	if !strings.Contains(string(parseOutputBytes), "RenderLanding") {
		parseT.Fatalf("expected generated file to contain render function, got %q", string(parseOutputBytes))
	}
}
