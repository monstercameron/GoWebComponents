package logging

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	bytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(bytes)
}

func TestCloneFieldsReturnsDistinctCopy(t *testing.T) {
	if cloneFields(nil) != nil {
		t.Fatal("expected nil clone for nil input fields")
	}
	if cloneFields(Fields{}) != nil {
		t.Fatal("expected nil clone for empty fields")
	}

	source := Fields{"id": 42, "ok": true}
	cloned := cloneFields(source)
	if len(cloned) != 2 || cloned["id"] != 42 || cloned["ok"] != true {
		t.Fatalf("unexpected cloned fields: %#v", cloned)
	}
	source["id"] = 99
	if cloned["id"] != 42 {
		t.Fatalf("expected cloned map to be independent, got %#v", cloned)
	}
}

func TestGlobalLogAndScopedLoggerMethodsWriteStructuredOutput(t *testing.T) {
	output := captureStdout(t, func() {
		Log("info", " demo-scope ", "global message", nil)
		logger := New("feature")
		logger.Debug("debug msg", nil)
		logger.Info("info msg", nil)
		logger.Warn("warn msg", nil)
		logger.Error("error msg", nil)
		logger.Log("trace", "custom log", Fields{"count": 3})
	})

	for _, expected := range []string{
		"[demo-scope] INFO: global message",
		"[feature] DEBUG: debug msg",
		"[feature] INFO: info msg",
		"[feature] WARN: warn msg",
		"[feature] ERROR: error msg",
		"[feature] TRACE: custom log",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to include %q, got %q", expected, output)
		}
	}
	if !strings.Contains(output, "map[count:3]") {
		t.Fatalf("expected structured fields in output, got %q", output)
	}
}

func TestWriteStructuredWithoutScope(t *testing.T) {
	output := captureStdout(t, func() {
		writeStructured("warn", "", "plain warning", nil)
	})
	if strings.Contains(output, "[") {
		t.Fatalf("expected no scope prefix for empty scope, got %q", output)
	}
	if !strings.Contains(output, "WARN: plain warning") {
		t.Fatalf("expected uppercase level message, got %q", output)
	}
}

