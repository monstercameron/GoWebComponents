package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/auth"
)

// captureServerLogBuffer routes stdlib log output into a test buffer and restores logger settings on cleanup.
func captureServerLogBuffer(parseT *testing.T) *bytes.Buffer {
	parseT.Helper()

	parseBuffer := &bytes.Buffer{}
	parsePrevWriter := log.Writer()
	parsePrevFlags := log.Flags()
	parsePrevPrefix := log.Prefix()
	log.SetOutput(parseBuffer)
	log.SetFlags(0)
	log.SetPrefix("")
	parseT.Cleanup(func() {
		log.SetOutput(parsePrevWriter)
		log.SetFlags(parsePrevFlags)
		log.SetPrefix(parsePrevPrefix)
	})
	return parseBuffer
}

// decodeServerLogEntry decodes the first JSON log line emitted during a request.
func decodeServerLogEntry(parseT *testing.T, parseBuffer *bytes.Buffer) map[string]any {
	parseT.Helper()

	parseLines := strings.SplitSeq(strings.TrimSpace(parseBuffer.String()), "\n")
	for parseLine := range parseLines {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			continue
		}
		parsePayload := map[string]any{}
		if parseErr := json.Unmarshal([]byte(parseTrimmed), &parsePayload); parseErr != nil {
			parseT.Fatalf("decode log entry %q: %v", parseTrimmed, parseErr)
		}
		return parsePayload
	}
	parseT.Fatalf("expected at least one server log line, got %q", parseBuffer.String())
	return nil
}

// decodeServerLogEntryByEvent decodes the first JSON log line whose event field matches parseEvent.
func decodeServerLogEntryByEvent(parseT *testing.T, parseBuffer *bytes.Buffer, parseEvent string) map[string]any {
	parseT.Helper()

	parseLines := strings.SplitSeq(strings.TrimSpace(parseBuffer.String()), "\n")
	for parseLine := range parseLines {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			continue
		}
		parsePayload := map[string]any{}
		if parseErr := json.Unmarshal([]byte(parseTrimmed), &parsePayload); parseErr != nil {
			parseT.Fatalf("decode log entry %q: %v", parseTrimmed, parseErr)
		}
		if strings.TrimSpace(parsePayload["event"].(string)) == parseEvent {
			return parsePayload
		}
	}
	parseT.Fatalf("expected %q log line, got %q", parseEvent, parseBuffer.String())
	return nil
}

// TestPageEntryStructuredServerLogIncludesRouteAndBootstrap validates the SSR route-entry log payload shape.
func TestPageEntryStructuredServerLogIncludesRouteAndBootstrap(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()
	parseServer.cfg.LogsEnabled = true

	parseLogBuffer := captureServerLogBuffer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/app/dashboard?q=desk&page=2&secret=sensitive", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseEntry := decodeServerLogEntry(parseT, parseLogBuffer)
	if parseEvent := strings.TrimSpace(parseEntry["event"].(string)); parseEvent != "page.entry" {
		parseT.Fatalf("expected page.entry event, got %#v", parseEntry["event"])
	}
	if parseMethod := strings.TrimSpace(parseEntry["method"].(string)); parseMethod != http.MethodGet {
		parseT.Fatalf("expected GET method, got %#v", parseEntry["method"])
	}
	if parsePath := strings.TrimSpace(parseEntry["path"].(string)); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected /app/dashboard path, got %#v", parseEntry["path"])
	}
	if parseSurface := strings.TrimSpace(parseEntry["surface"].(string)); parseSurface != "internal" {
		parseT.Fatalf("expected internal surface, got %#v", parseEntry["surface"])
	}
	if parseStatus := int(parseEntry["status"].(float64)); parseStatus != http.StatusOK {
		parseT.Fatalf("expected status 200, got %#v", parseEntry["status"])
	}
	if parseDurationMS := int(parseEntry["duration_ms"].(float64)); parseDurationMS < 0 {
		parseT.Fatalf("expected non-negative duration, got %#v", parseEntry["duration_ms"])
	}
	parseQuery := strings.TrimSpace(parseEntry["query"].(string))
	if !strings.Contains(parseQuery, "q=desk") || !strings.Contains(parseQuery, "page=2") {
		parseT.Fatalf("expected query to include q and page filters, got %q", parseQuery)
	}
	if strings.Contains(parseQuery, "secret=") {
		parseT.Fatalf("expected query guardrail to suppress secret parameter, got %q", parseQuery)
	}
	if parseMode := strings.TrimSpace(parseEntry["bootstrap_mode"].(string)); parseMode != "inline" {
		parseT.Fatalf("expected inline bootstrap mode, got %#v", parseEntry["bootstrap_mode"])
	}
	if strings.TrimSpace(parseEntry["bootstrap_bytes"].(string)) == "" {
		parseT.Fatalf("expected bootstrap_bytes value, got %#v", parseEntry["bootstrap_bytes"])
	}
}

// TestMutationStructuredServerLogIncludesTypeIdentifierAndResultState validates mutation logging fields.
func TestMutationStructuredServerLogIncludesTypeIdentifierAndResultState(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCSRFTok, parseCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/receiving/rcv-illinois-001")
	parseServer.cfg.LogsEnabled = true
	parseLogBuffer := captureServerLogBuffer(parseT)

	parseForm := url.Values{
		"csrf_token":          {parseCSRFTok},
		"status":              {"closed"},
		"discrepancy_summary": {"Structured logging validation closeout."},
	}
	parseReq := httptest.NewRequest(http.MethodPost, "/api/app/receiving/rcv-illinois-001/reconcile", strings.NewReader(parseForm.Encode()))
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Header.Set("Referer", "http://example.com/app/receiving/rcv-illinois-001")
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReq.AddCookie(parseCSRFCookie)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseRes.Code)
	}
	parseEntry := decodeServerLogEntryByEvent(parseT, parseLogBuffer, "mutation.result")
	if parseType := strings.TrimSpace(parseEntry["mutation_type"].(string)); parseType != "internal_receiving_reconcile" {
		parseT.Fatalf("expected internal_receiving_reconcile mutation type, got %#v", parseEntry["mutation_type"])
	}
	if parseMutationID := strings.TrimSpace(parseEntry["mutation_id"].(string)); parseMutationID != "rcv-illinois-001" {
		parseT.Fatalf("expected receiving session id, got %#v", parseEntry["mutation_id"])
	}
	if parseResultState := strings.TrimSpace(parseEntry["result_state"].(string)); parseResultState != "receiving-reconciled" {
		parseT.Fatalf("expected receiving-reconciled result state, got %#v", parseEntry["result_state"])
	}
	if parseStatus := int(parseEntry["status"].(float64)); parseStatus != http.StatusSeeOther {
		parseT.Fatalf("expected redirect status for form mutation, got %#v", parseEntry["status"])
	}
}
