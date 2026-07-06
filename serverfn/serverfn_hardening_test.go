package serverfn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleRejectsOversizedBody pins that a request body over the configured
// limit is rejected with 413 rather than read unbounded into memory.
func TestHandleRejectsOversizedBody(parseT *testing.T) {
	SetMaxRequestBytes(64)
	parseT.Cleanup(func() { SetMaxRequestBytes(0) })

	parseMux := http.NewServeMux()
	Handle(parseMux, "Echo", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
		return echoResp{Greeting: parseReq.Name}, nil
	})
	parseServer := httptest.NewServer(parseMux)
	parseT.Cleanup(parseServer.Close)

	parseBig := `{"name":"` + strings.Repeat("x", 4096) + `"}`
	parseResp, parseErr := http.Post(parseServer.URL+Endpoint("Echo"), "application/json", strings.NewReader(parseBig))
	if parseErr != nil {
		parseT.Fatalf("post: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusRequestEntityTooLarge {
		parseT.Fatalf("status = %d, want 413", parseResp.StatusCode)
	}
}

// TestHandleRecoversPanic pins that a panic inside a server function becomes a
// clean 500 rather than resetting the connection.
func TestHandleRecoversPanic(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Boom", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			panic("kaboom")
		})
	})

	_, parseErr := Call[echoReq, echoResp](context.Background(), "Boom", echoReq{Name: "x"})
	if parseErr == nil {
		parseT.Fatal("expected error from panicking server function")
	}
	var parseServerErr *ServerError
	if !asServerError(parseErr, &parseServerErr) || parseServerErr.Status != http.StatusInternalServerError {
		parseT.Fatalf("expected 500 ServerError, got %v", parseErr)
	}
}

func asServerError(parseErr error, parseTarget **ServerError) bool {
	parseServerErr, parseOk := parseErr.(*ServerError)
	if parseOk {
		*parseTarget = parseServerErr
	}
	return parseOk
}

// csrfTestServer registers an Echo endpoint on a raw httptest server and returns its URL,
// bypassing Call (which always sends application/json) so the CSRF gate can be probed with
// arbitrary headers the way a forged cross-origin request would.
func csrfTestServer(parseT *testing.T) string {
	parseT.Helper()
	parseMux := http.NewServeMux()
	Handle(parseMux, "Echo", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
		return echoResp{Greeting: parseReq.Name}, nil
	})
	parseServer := httptest.NewServer(parseMux)
	parseT.Cleanup(parseServer.Close)
	return parseServer.URL + Endpoint("Echo")
}

// TestHandleRejectsNonJSONContentType pins the primary CSRF defense: a CORS-safelisted
// "simple" content type (text/plain) — which a forged cross-origin form/fetch can send
// with cookies and no preflight — is rejected with 415 before the body is even read.
func TestHandleRejectsNonJSONContentType(parseT *testing.T) {
	parseURL := csrfTestServer(parseT)
	parseResp, parseErr := http.Post(parseURL, "text/plain", strings.NewReader(`{"name":"attacker"}`))
	if parseErr != nil {
		parseT.Fatalf("post: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusUnsupportedMediaType {
		parseT.Fatalf("text/plain must be rejected with 415, got %d", parseResp.StatusCode)
	}

	// A missing Content-Type is likewise rejected.
	parseReq, _ := http.NewRequest(http.MethodPost, parseURL, strings.NewReader(`{}`))
	parseReq.Header.Del("Content-Type")
	parseBlank, parseErr := http.DefaultClient.Do(parseReq)
	if parseErr != nil {
		parseT.Fatalf("blank-ct post: %v", parseErr)
	}
	defer parseBlank.Body.Close()
	if parseBlank.StatusCode != http.StatusUnsupportedMediaType {
		parseT.Fatalf("missing Content-Type must be rejected with 415, got %d", parseBlank.StatusCode)
	}
}

// TestHandleAcceptsJSONContentTypeWithCharset pins that a legitimate JSON request with a
// charset parameter (as some clients send) is NOT falsely rejected by the media-type gate.
func TestHandleAcceptsJSONContentTypeWithCharset(parseT *testing.T) {
	parseURL := csrfTestServer(parseT)
	parseResp, parseErr := http.Post(parseURL, "application/json; charset=utf-8", strings.NewReader(`{"name":"ok"}`))
	if parseErr != nil {
		parseT.Fatalf("post: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseT.Fatalf("application/json;charset=utf-8 must be accepted, got %d", parseResp.StatusCode)
	}
}

// TestHandleRejectsCrossSiteFetch pins the defense-in-depth check: a browser-set
// Sec-Fetch-Site: cross-site header (which JS cannot forge) is rejected with 403 even when
// the Content-Type is JSON.
func TestHandleRejectsCrossSiteFetch(parseT *testing.T) {
	parseURL := csrfTestServer(parseT)
	parseReq, _ := http.NewRequest(http.MethodPost, parseURL, strings.NewReader(`{"name":"x"}`))
	parseReq.Header.Set("Content-Type", "application/json")
	parseReq.Header.Set("Sec-Fetch-Site", "cross-site")
	parseResp, parseErr := http.DefaultClient.Do(parseReq)
	if parseErr != nil {
		parseT.Fatalf("post: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusForbidden {
		parseT.Fatalf("cross-site fetch must be rejected with 403, got %d", parseResp.StatusCode)
	}

	// same-origin is the common case and must pass.
	parseReq2, _ := http.NewRequest(http.MethodPost, parseURL, strings.NewReader(`{"name":"x"}`))
	parseReq2.Header.Set("Content-Type", "application/json")
	parseReq2.Header.Set("Sec-Fetch-Site", "same-origin")
	parseResp2, parseErr := http.DefaultClient.Do(parseReq2)
	if parseErr != nil {
		parseT.Fatalf("same-origin post: %v", parseErr)
	}
	defer parseResp2.Body.Close()
	if parseResp2.StatusCode != http.StatusOK {
		parseT.Fatalf("same-origin fetch must be accepted, got %d", parseResp2.StatusCode)
	}
}

// TestCSRFProtectionToggle pins that SetCSRFProtection(false) restores the permissive
// behavior for deployments that must accept raw non-JSON callers.
func TestCSRFProtectionToggle(parseT *testing.T) {
	SetCSRFProtection(false)
	parseT.Cleanup(func() { SetCSRFProtection(true) })

	parseURL := csrfTestServer(parseT)
	parseResp, parseErr := http.Post(parseURL, "text/plain", strings.NewReader(`{"name":"raw"}`))
	if parseErr != nil {
		parseT.Fatalf("post: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseT.Fatalf("with CSRF protection off, text/plain must be accepted, got %d", parseResp.StatusCode)
	}
}
