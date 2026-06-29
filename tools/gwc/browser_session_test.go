//go:build playwrightgo

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenBrowserSessionRequiresURL pins the precondition: openBrowserSession
// rejects an empty url before it ever launches a browser.
func TestOpenBrowserSessionRequiresURL(t *testing.T) {
	if _, parseErr := openBrowserSession("", 9222, 1280, 800); parseErr == nil {
		t.Fatal("openBrowserSession(\"\") = nil error, want a required-url error")
	}
}

// TestOpenBrowserSessionOpensHeadedWindow pins the copilot-dev feature: a headed
// (visible), sandbox-disabled window opens on the dev URL and reports a CDP
// endpoint copilot can attach to. Skipped where no display is available (e.g. a
// headless CI box), since a headed window genuinely needs one.
func TestOpenBrowserSessionOpensHeadedWindow(t *testing.T) {
	parseSrv := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		_, _ = parseW.Write([]byte("<!doctype html><title>dev</title><h1>copilot dev</h1>"))
	}))
	defer parseSrv.Close()

	parsePort := 9444
	parseSession, parseErr := openBrowserSession(parseSrv.URL, parsePort, 1024, 768)
	if parseErr != nil {
		t.Skipf("headed browser unavailable (no display?): %v", parseErr)
	}
	defer parseSession.close()

	if !parseSession.Result.Headed {
		t.Fatal("Result.Headed = false, want true")
	}
	if !parseSession.Result.NoSandbox {
		t.Fatal("Result.NoSandbox = false, want true")
	}
	parseWant := fmt.Sprintf("http://127.0.0.1:%d", parsePort)
	if parseSession.Result.CDPEndpoint != parseWant {
		t.Fatalf("Result.CDPEndpoint = %q, want %q", parseSession.Result.CDPEndpoint, parseWant)
	}
	if !strings.HasPrefix(parseSession.Result.URL, "http://127.0.0.1:") {
		t.Fatalf("Result.URL = %q, want the served dev URL", parseSession.Result.URL)
	}
}
