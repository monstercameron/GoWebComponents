package main

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/auth"
)

func TestEnsureCSRFCookieReuseAndCreate(parseT *testing.T) {
	parseT.Run("reuse existing cookie", func(parseT2 *testing.T) {
		parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
		parseReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "existing-token"})
		parseRes := httptest.NewRecorder()

		parseToken := ensureCSRFCookie(parseRes, parseReq)

		if parseToken != "existing-token" {
			parseT2.Fatalf("ensureCSRFCookie() = %q, want existing-token", parseToken)
		}
		if len(parseRes.Result().Cookies()) != 0 {
			parseT2.Fatalf("expected no Set-Cookie when reusing token, got %+v", parseRes.Result().Cookies())
		}
	})

	parseT.Run("create new cookie", func(parseT3 *testing.T) {
		parseReq2 := httptest.NewRequest(http.MethodGet, "/", nil)
		parseRes2 := httptest.NewRecorder()

		parseToken2 := ensureCSRFCookie(parseRes2, parseReq2)

		if parseToken2 == "" {
			parseT3.Fatal("expected generated csrf token")
		}
		if _, parseErr := base64.RawURLEncoding.DecodeString(parseToken2); parseErr != nil {
			parseT3.Fatalf("generated token was not base64url: %v", parseErr)
		}
		parseCookies := parseRes2.Result().Cookies()
		if len(parseCookies) != 1 {
			parseT3.Fatalf("expected one csrf cookie, got %+v", parseCookies)
		}
		if parseCookies[0].Name != csrfCookieName || parseCookies[0].Value != parseToken2 || parseCookies[0].Path != "/" || !parseCookies[0].HttpOnly || parseCookies[0].SameSite != http.SameSiteStrictMode {
			parseT3.Fatalf("unexpected csrf cookie: %+v", parseCookies[0])
		}
	})
}

func TestRequestCSRFTokenAndOriginHelpers(parseT *testing.T) {
	parseHeaderReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseHeaderReq.Header.Set(csrfHeaderName, " header-token ")
	if parseGot, parseErr := requestCSRFToken(parseHeaderReq); parseErr != nil || parseGot != "header-token" {
		parseT.Fatalf("requestCSRFToken(header) = %q, %v; want header-token, nil", parseGot, parseErr)
	}

	parseJsonReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	parseJsonReq.Header.Set("Content-Type", "application/json")
	if _, parseErr2 := requestCSRFToken(parseJsonReq); parseErr2 == nil || !strings.Contains(parseErr2.Error(), csrfHeaderName) {
		parseT.Fatalf("expected missing json header error, got %v", parseErr2)
	}

	parseFormReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("csrf_token=form-token"))
	parseFormReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if parseGot2, parseErr3 := requestCSRFToken(parseFormReq); parseErr3 != nil || parseGot2 != "form-token" {
		parseT.Fatalf("requestCSRFToken(form) = %q, %v; want form-token, nil", parseGot2, parseErr3)
	}

	parseMissingFormReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("x=1"))
	parseMissingFormReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, parseErr4 := requestCSRFToken(parseMissingFormReq); parseErr4 == nil || !strings.Contains(parseErr4.Error(), csrfFormFieldName) {
		parseT.Fatalf("expected missing form field error, got %v", parseErr4)
	}

	parseReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseReq.Host = "example.com"
	parseReq.Header.Set("Origin", "http://example.com")
	if !sameOriginRequest(parseReq) {
		parseT.Fatal("expected sameOriginRequest to accept matching Origin")
	}

	parseRefererReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseRefererReq.Host = "example.com"
	parseRefererReq.Header.Set("Referer", "http://example.com/path")
	if !sameOriginRequest(parseRefererReq) {
		parseT.Fatal("expected sameOriginRequest to accept matching Referer")
	}

	parseMismatchReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseMismatchReq.Host = "example.com"
	parseMismatchReq.Header.Set("Origin", "https://other.example.com")
	if sameOriginRequest(parseMismatchReq) {
		parseT.Fatal("expected sameOriginRequest to reject mismatched origin")
	}

	if sameOriginRequest(httptest.NewRequest(http.MethodPost, "/", nil)) {
		parseT.Fatal("expected sameOriginRequest to reject missing origin and referer")
	}

	parseForwardedReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseForwardedReq.Header.Set("X-Forwarded-Proto", "HTTPS")
	if parseGot3 := requestScheme(parseForwardedReq); parseGot3 != "https" {
		parseT.Fatalf("requestScheme(forwarded) = %q, want https", parseGot3)
	}

	parseTlsReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseTlsReq.TLS = &tls.ConnectionState{}
	if parseGot4 := requestScheme(parseTlsReq); parseGot4 != "https" {
		parseT.Fatalf("requestScheme(tls) = %q, want https", parseGot4)
	}

	if parseGot5 := requestScheme(httptest.NewRequest(http.MethodPost, "/", nil)); parseGot5 != "http" {
		parseT.Fatalf("requestScheme(default) = %q, want http", parseGot5)
	}

	parseMatchReq := httptest.NewRequest(http.MethodPost, "/", nil)
	parseMatchReq.Host = "example.com"
	if !originMatchesRequest("http://example.com/ok", parseMatchReq) {
		parseT.Fatal("expected originMatchesRequest to accept matching origin")
	}
	if originMatchesRequest("://bad", parseMatchReq) || originMatchesRequest("/relative", parseMatchReq) {
		parseT.Fatal("expected originMatchesRequest to reject invalid origins")
	}
}

func TestValidateCSRFRequestBranches(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseT.Run("success", func(parseT2 *testing.T) {
		parseReq := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		parseReq.Host = "example.com"
		parseReq.Header.Set("Origin", "http://example.com")
		parseReq.Header.Set(csrfHeaderName, "valid-token")
		parseReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "valid-token"})
		parseRes := httptest.NewRecorder()

		if !validateCSRFRequest(parseServer, parseRes, parseReq) {
			parseT2.Fatalf("expected validateCSRFRequest to succeed, got body %q", parseRes.Body.String())
		}
	})

	parseT.Run("same origin failed", func(parseT3 *testing.T) {
		parseReq2 := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		parseReq2.Host = "example.com"
		parseReq2.Header.Set("Origin", "http://other.example.com")
		parseRes2 := httptest.NewRecorder()

		if validateCSRFRequest(parseServer, parseRes2, parseReq2) {
			parseT3.Fatal("expected same-origin validation failure")
		}
		if parseRes2.Code != http.StatusForbidden || !strings.Contains(parseRes2.Body.String(), "csrf_same_origin_failed") {
			parseT3.Fatalf("unexpected same-origin failure response: code=%d body=%q", parseRes2.Code, parseRes2.Body.String())
		}
	})

	parseT.Run("cookie missing", func(parseT4 *testing.T) {
		parseReq3 := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		parseReq3.Host = "example.com"
		parseReq3.Header.Set("Origin", "http://example.com")
		parseReq3.Header.Set(csrfHeaderName, "valid-token")
		parseRes3 := httptest.NewRecorder()

		if validateCSRFRequest(parseServer, parseRes3, parseReq3) {
			parseT4.Fatal("expected missing-cookie validation failure")
		}
		if parseRes3.Code != http.StatusForbidden || !strings.Contains(parseRes3.Body.String(), "csrf_cookie_missing") {
			parseT4.Fatalf("unexpected missing-cookie response: code=%d body=%q", parseRes3.Code, parseRes3.Body.String())
		}
	})

	parseT.Run("token missing", func(parseT5 *testing.T) {
		parseReq4 := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(`{}`))
		parseReq4.Host = "example.com"
		parseReq4.Header.Set("Origin", "http://example.com")
		parseReq4.Header.Set("Content-Type", "application/json")
		parseReq4.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "valid-token"})
		parseRes4 := httptest.NewRecorder()

		if validateCSRFRequest(parseServer, parseRes4, parseReq4) {
			parseT5.Fatal("expected missing-token validation failure")
		}
		if parseRes4.Code != http.StatusForbidden || !strings.Contains(parseRes4.Body.String(), "csrf_token_missing") {
			parseT5.Fatalf("unexpected missing-token response: code=%d body=%q", parseRes4.Code, parseRes4.Body.String())
		}
	})

	parseT.Run("token invalid", func(parseT6 *testing.T) {
		parseReq5 := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		parseReq5.Host = "example.com"
		parseReq5.Header.Set("Origin", "http://example.com")
		parseReq5.Header.Set(csrfHeaderName, "different-token")
		parseReq5.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "valid-token"})
		parseRes5 := httptest.NewRecorder()

		if validateCSRFRequest(parseServer, parseRes5, parseReq5) {
			parseT6.Fatal("expected token-mismatch validation failure")
		}
		if parseRes5.Code != http.StatusForbidden || !strings.Contains(parseRes5.Body.String(), "csrf_token_invalid") {
			parseT6.Fatalf("unexpected invalid-token response: code=%d body=%q", parseRes5.Code, parseRes5.Body.String())
		}
	})
}

func TestGenerateCSRFToken(parseT *testing.T) {
	parseToken := generateCSRFToken()
	if parseToken == "" {
		parseT.Fatal("expected generated token")
	}
	if _, parseErr := base64.RawURLEncoding.DecodeString(parseToken); parseErr != nil {
		parseT.Fatalf("generateCSRFToken() returned invalid base64url token: %v", parseErr)
	}
}

func TestCSRFCookieCanSeedPageRequests(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/products", nil)
	parseReq.Header.Set("Host", "example.com")
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "prefilled-csrf"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	if strings.Contains(parseRes.Header().Get("Set-Cookie"), csrfCookieName+"=") {
		parseT.Fatalf("expected no replacement csrf Set-Cookie header, got %q", parseRes.Header().Get("Set-Cookie"))
	}
	if !strings.Contains(parseRes.Body.String(), `"csrf":"prefilled-csrf"`) {
		parseT.Fatalf("expected bootstrap payload to reuse csrf token, got %q", parseRes.Body.String())
	}
}
