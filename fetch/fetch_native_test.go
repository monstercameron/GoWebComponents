//go:build !js || !wasm
// +build !js !wasm

package fetch

import (
	"context"
	"strings"
	"testing"
)

func TestFetchNativeTransportAndResultHelpers(parseT *testing.T) {
	parseResult := <-Fetch("/api/profile", Options{Method: "POST"})
	if parseResult.Err == nil || parseResult.Err.Error() != "fetch API unavailable in this environment" {
		parseT.Fatalf("expected native fetch fallback error, got %+v", parseResult)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseUploadCh := Upload(parseCtx, "/api/upload", Options{})
	parseUpdate, parseOk := <-parseUploadCh
	if !parseOk || !parseUpdate.Done || parseUpdate.Result.Err != context.Canceled {
		parseT.Fatalf("expected canceled native upload result, got update=%+v ok=%t", parseUpdate, parseOk)
	}
	if _, parseStillOpen := <-parseUploadCh; parseStillOpen {
		parseT.Fatal("expected native upload fallback channel to close after the terminal update")
	}

	if parseText := (Result{Data: 42}).Text(); parseText != "42" {
		parseT.Fatalf("expected non-string result text to format as 42, got %q", parseText)
	}
	if parseText := (Result{Data: "ready"}).Text(); parseText != "ready" {
		parseT.Fatalf("expected string-backed result text, got %q", parseText)
	}

	var parseTarget struct {
		Name string `json:"name"`
	}
	if parseErr := (Result{Data: `{"name":"Ada"}`}).DecodeJSON(&parseTarget); parseErr != nil || parseTarget.Name != "Ada" {
		parseT.Fatalf("expected DecodeJSON success, target=%+v err=%v", parseTarget, parseErr)
	}
	if parseErr := (Result{Data: `{"name":"Ada"}`}).DecodeJSON(nil); parseErr == nil || !strings.Contains(parseErr.Error(), "decode target is nil") {
		parseT.Fatalf("expected nil-target DecodeJSON error, got %v", parseErr)
	}
	if parseErr := (Result{}).DecodeJSON(&parseTarget); parseErr == nil || !strings.Contains(parseErr.Error(), "response body is empty") {
		parseT.Fatalf("expected empty-body DecodeJSON error, got %v", parseErr)
	}
	if parseErr := (Result{Data: "{"}).DecodeJSON(&parseTarget); parseErr == nil || !strings.Contains(parseErr.Error(), "decode response json") {
		parseT.Fatalf("expected invalid-json DecodeJSON error, got %v", parseErr)
	}

	if parseMessage := (HTTPError{}).Error(); parseMessage != "request failed" {
		parseT.Fatalf("expected zero HTTPError message, got %q", parseMessage)
	}
	if parseMessage := (HTTPError{Status: 422, StatusText: "Unprocessable Entity"}).Error(); parseMessage != "request failed with status 422 Unprocessable Entity" {
		parseT.Fatalf("unexpected HTTPError message: %q", parseMessage)
	}

	parseHeaders := parseRawHeaders("Content-Type: application/json\nX-Request-ID: abc123\nbroken-header\n")
	if parseHeaders["Content-Type"] != "application/json" || parseHeaders["X-Request-ID"] != "abc123" || len(parseHeaders) != 2 {
		parseT.Fatalf("unexpected parsed headers: %#v", parseHeaders)
	}
	if parseRawHeaders("") != nil {
		parseT.Fatal("expected empty raw headers to return nil")
	}

	ReturnChannel(make(chan Result))
}

func TestFetchNativeResourceHandles(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseFetchResource := UseFetch("/api/profile")
	parseFetchState := parseFetchResource.Get()
	if parseFetchState.Error != "fetch API unavailable in this environment" || parseFetchState.Loading {
		parseT.Fatalf("unexpected native UseFetch stub state: %+v", parseFetchState)
	}
	parseFetchResource.Refetch()

	parseResource := UseResource(func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		return "ready", nil
	}, "dep")
	parseState := parseResource.Get()
	if parseState.Loading || parseState.Ready || parseState.Error != nil || parseState.Value != "" {
		parseT.Fatalf("unexpected initial native resource state: %+v", parseState)
	}
	parseResource.Reload()
	parseResource.Cancel()

	var parseZero AsyncResource[string]
	if parseZeroState := parseZero.Get(); parseZeroState.Loading || parseZeroState.Ready || parseZeroState.Error != nil || parseZeroState.Value != "" {
		parseT.Fatalf("unexpected zero-value async resource state: %+v", parseZeroState)
	}
	parseZero.Reload()
	parseZero.Cancel()
}
