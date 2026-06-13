//go:build !js || !wasm

package logging

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
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

// decodeOutputRecords parses one newline-delimited JSON log stream into record maps.
func decodeOutputRecords(parseT *testing.T, parseOutput string) []map[string]any {
	parseT.Helper()
	parseLines := strings.FieldsFunc(parseOutput, func(parseRune rune) bool {
		return parseRune == '\n' || parseRune == '\r'
	})
	parseRecords := make([]map[string]any, 0, len(parseLines))
	for _, parseLine := range parseLines {
		if parseLine == "" {
			continue
		}
		parseRecord := map[string]any{}
		if parseErr := json.Unmarshal([]byte(parseLine), &parseRecord); parseErr != nil {
			parseT.Fatalf("decode log record %q: %v", parseLine, parseErr)
		}
		parseRecords = append(parseRecords, parseRecord)
	}
	return parseRecords
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

	parseRecords := decodeOutputRecords(parseT, parseOutput)
	if len(parseRecords) != 6 {
		parseT.Fatalf("expected 6 structured records, got %d", len(parseRecords))
	}
	for parseIndex, parseWant := range []struct {
		level        string
		scope        string
		message      string
		severityText string
	}{
		{level: "info", scope: "demo-scope", message: "global message", severityText: "INFO"},
		{level: "debug", scope: "feature", message: "debug msg", severityText: "DEBUG"},
		{level: "info", scope: "feature", message: "info msg", severityText: "INFO"},
		{level: "warn", scope: "feature", message: "warn msg", severityText: "WARN"},
		{level: "error", scope: "feature", message: "error msg", severityText: "ERROR"},
		{level: "trace", scope: "feature", message: "custom log", severityText: "TRACE"},
	} {
		parseRecord := parseRecords[parseIndex]
		if parseRecord["level"] != parseWant.level || parseRecord["scope"] != parseWant.scope || parseRecord["message"] != parseWant.message || parseRecord["severity_text"] != parseWant.severityText {
			parseT.Fatalf("unexpected record[%d]: %#v", parseIndex, parseRecord)
		}
	}

	parseLastRecord := parseRecords[len(parseRecords)-1]
	if parseLastRecord["count"].(float64) != 3 {
		parseT.Fatalf("expected mirrored top-level count field, got %#v", parseLastRecord)
	}
	parseAttributes, parseOk := parseLastRecord["attributes"].(map[string]any)
	if !parseOk || parseAttributes["count"].(float64) != 3 {
		parseT.Fatalf("expected structured attributes count field, got %#v", parseLastRecord["attributes"])
	}
}

func TestWriteStructuredWithoutScope(parseT *testing.T) {
	parseOutput := captureStdout(parseT, func() {
		writeStructured("warn", "", "plain warning", nil)
	})

	parseRecords := decodeOutputRecords(parseT, parseOutput)
	if len(parseRecords) != 1 {
		parseT.Fatalf("expected one record, got %d", len(parseRecords))
	}
	if parseRecords[0]["scope"] != "" || parseRecords[0]["severity_text"] != "WARN" || parseRecords[0]["message"] != "plain warning" {
		parseT.Fatalf("unexpected record without scope: %#v", parseRecords[0])
	}
}

func TestNativeWriteStructuredContextReportsMarshalFailure(parseT *testing.T) {
	parseOutput := captureStdout(parseT, func() {
		writeStructuredContext(context.Background(), "info", "test", "bad field", map[string]any{
			"bad": math.Inf(1),
		})
	})
	if !strings.Contains(parseOutput, `"logging marshal failed"`) {
		parseT.Fatalf("expected marshal failure fallback, got %q", parseOutput)
	}
	if !strings.Contains(parseOutput, `"level":"error"`) {
		parseT.Fatalf("expected fallback to be an error record, got %q", parseOutput)
	}
}

// TestNewContextAndWithContextIncludeStoredMetadata verifies stored logging context metadata is emitted by scoped loggers.
func TestNewContextAndWithContextIncludeStoredMetadata(parseT *testing.T) {
	parseCtx := context.Background()
	parseCtx = StoreCorrelationID(parseCtx, "corr-42")
	parseCtx = StoreTraceContext(parseCtx, TraceContext{
		TraceID:    "0123456789abcdef0123456789abcdef",
		SpanID:     "89abcdef01234567",
		Flags:      "01",
		TraceState: "tenant=demo",
	})

	parseOutput := captureStdout(parseT, func() {
		NewContext(parseCtx, "router").Info("blocked", "route", "/admin")
		New("router").WithContext(parseCtx).Info("retried", "attempt", 2)
	})

	parseRecords := decodeOutputRecords(parseT, parseOutput)
	if len(parseRecords) != 2 {
		parseT.Fatalf("expected 2 records, got %d", len(parseRecords))
	}
	for _, parseRecord := range parseRecords {
		if parseRecord["correlation_id"] != "corr-42" || parseRecord["trace_id"] != "0123456789abcdef0123456789abcdef" || parseRecord["span_id"] != "89abcdef01234567" {
			parseT.Fatalf("expected stored correlation and trace metadata, got %#v", parseRecord)
		}
		if parseRecord["traceparent"] != "00-0123456789abcdef0123456789abcdef-89abcdef01234567-01" || parseRecord["tracestate"] != "tenant=demo" {
			parseT.Fatalf("expected traceparent and tracestate metadata, got %#v", parseRecord)
		}
	}
	if parseRecords[0]["route"] != "/admin" || parseRecords[1]["attempt"].(float64) != 2 {
		parseT.Fatalf("expected structured attributes to survive context enrichment, got %#v %#v", parseRecords[0], parseRecords[1])
	}
}

// TestLogContextUsesUIContextFallback verifies native ui correlation helpers still enrich core logging output.
func TestLogContextUsesUIContextFallback(parseT *testing.T) {
	parseCtx := context.Background()
	parseCtx = ui.WithCorrelationID(parseCtx, "ui-corr-7")
	parseCtx = ui.WithW3CTraceContext(parseCtx, ui.W3CTraceContext{
		TraceID:    "fedcba9876543210fedcba9876543210",
		SpanID:     "76543210fedcba98",
		Flags:      "01",
		TraceState: "source=ui",
	})

	parseOutput := captureStdout(parseT, func() {
		LogContext(parseCtx, "info", "router", "ui bridged", "route", "/settings")
	})

	parseRecords := decodeOutputRecords(parseT, parseOutput)
	if len(parseRecords) != 1 {
		parseT.Fatalf("expected one bridged record, got %d", len(parseRecords))
	}
	parseRecord := parseRecords[0]
	if parseRecord["correlation_id"] != "ui-corr-7" || parseRecord["trace_id"] != "fedcba9876543210fedcba9876543210" || parseRecord["span_id"] != "76543210fedcba98" {
		parseT.Fatalf("expected ui correlation metadata to bridge into logs, got %#v", parseRecord)
	}
	if parseRecord["traceparent"] != "00-fedcba9876543210fedcba9876543210-76543210fedcba98-01" || parseRecord["tracestate"] != "source=ui" {
		parseT.Fatalf("expected ui trace headers in logs, got %#v", parseRecord)
	}
}
