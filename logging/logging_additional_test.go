//go:build !js || !wasm
// +build !js !wasm

package logging

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(parseT *testing.T, parseFn func()) string {
	parseT.Helper()
	parseOriginal := os.Stdout
	parseReader, parseWriter, parseErr := os.Pipe()
	if parseErr != nil {
		parseT.Fatalf("create stdout pipe: %v", parseErr)
	}
	os.Stdout = parseWriter
	defer func() {
		os.Stdout = parseOriginal
	}()

	parseFn()

	if parseErr2 := parseWriter.Close(); parseErr2 != nil {
		parseT.Fatalf("close stdout writer: %v", parseErr2)
	}
	parseBytes, parseErr := io.ReadAll(parseReader)
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	return string(parseBytes)
}

func TestCloneFieldsReturnsDistinctCopy(parseT *testing.T) {
	if cloneFields(nil) != nil {
		parseT.Fatal("expected nil clone for nil input fields")
	}
	if cloneFields(Fields{}) != nil {
		parseT.Fatal("expected nil clone for empty fields")
	}

	parseSource := Fields{"id": 42, "ok": true}
	parseCloned := cloneFields(parseSource)
	if len(parseCloned) != 2 || parseCloned["id"] != 42 || parseCloned["ok"] != true {
		parseT.Fatalf("unexpected cloned fields: %#v", parseCloned)
	}
	parseSource["id"] = 99
	if parseCloned["id"] != 42 {
		parseT.Fatalf("expected cloned map to be independent, got %#v", parseCloned)
	}
}

func TestGlobalLogAndScopedLoggerMethodsWriteStructuredOutput(parseT *testing.T) {
	parseOutput := captureStdout(parseT, func() {
		Log("info", " demo-scope ", "global message", nil)
		parseLogger := New("feature")
		parseLogger.Debug("debug msg", nil)
		parseLogger.Info("info msg", nil)
		parseLogger.Warn("warn msg", nil)
		parseLogger.Error("error msg", nil)
		parseLogger.Log("trace", "custom log", Fields{"count": 3})
	})

	for _, parseExpected := range []string{
		"[demo-scope] INFO: global message",
		"[feature] DEBUG: debug msg",
		"[feature] INFO: info msg",
		"[feature] WARN: warn msg",
		"[feature] ERROR: error msg",
		"[feature] TRACE: custom log",
	} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected output to include %q, got %q", parseExpected, parseOutput)
		}
	}
	if !strings.Contains(parseOutput, "map[count:3]") {
		parseT.Fatalf("expected structured fields in output, got %q", parseOutput)
	}
}

func TestWriteStructuredWithoutScope(parseT *testing.T) {
	parseOutput := captureStdout(parseT, func() {
		writeStructured("warn", "", "plain warning", nil)
	})
	if strings.Contains(parseOutput, "[") {
		parseT.Fatalf("expected no scope prefix for empty scope, got %q", parseOutput)
	}
	if !strings.Contains(parseOutput, "WARN: plain warning") {
		parseT.Fatalf("expected uppercase level message, got %q", parseOutput)
	}
}
