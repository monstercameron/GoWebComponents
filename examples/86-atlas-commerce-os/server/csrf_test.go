package main

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
)

func TestEnsureCSRFCookieReuseAndCreate(t *testing.T) {
	t.Run("reuse existing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "existing-token"})
		res := httptest.NewRecorder()

		token := ensureCSRFCookie(res, req)

		if token != "existing-token" {
			t.Fatalf("ensureCSRFCookie() = %q, want existing-token", token)
		}
		if len(res.Result().Cookies()) != 0 {
			t.Fatalf("expected no Set-Cookie when reusing token, got %+v", res.Result().Cookies())
		}
	})

	t.Run("create new cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()

		token := ensureCSRFCookie(res, req)

		if token == "" {
			t.Fatal("expected generated csrf token")
		}
		if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
			t.Fatalf("generated token was not base64url: %v", err)
		}
		cookies := res.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("expected one csrf cookie, got %+v", cookies)
		}
		if cookies[0].Name != csrfCookieName || cookies[0].Value != token || cookies[0].Path != "/" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatalf("unexpected csrf cookie: %+v", cookies[0])
		}
	})
}

func TestRequestCSRFTokenAndOriginHelpers(t *testing.T) {
	headerReq := httptest.NewRequest(http.MethodPost, "/", nil)
	headerReq.Header.Set(csrfHeaderName, " header-token ")
	if got, err := requestCSRFToken(headerReq); err != nil || got != "header-token" {
		t.Fatalf("requestCSRFToken(header) = %q, %v; want header-token, nil", got, err)
	}

	jsonReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	jsonReq.Header.Set("Content-Type", "application/json")
	if _, err := requestCSRFToken(jsonReq); err == nil || !strings.Contains(err.Error(), csrfHeaderName) {
		t.Fatalf("expected missing json header error, got %v", err)
	}

	formReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("csrf_token=form-token"))
	formReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got, err := requestCSRFToken(formReq); err != nil || got != "form-token" {
		t.Fatalf("requestCSRFToken(form) = %q, %v; want form-token, nil", got, err)
	}

	missingFormReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("x=1"))
	missingFormReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := requestCSRFToken(missingFormReq); err == nil || !strings.Contains(err.Error(), csrfFormFieldName) {
		t.Fatalf("expected missing form field error, got %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "http://example.com")
	if !sameOriginRequest(req) {
		t.Fatal("expected sameOriginRequest to accept matching Origin")
	}

	refererReq := httptest.NewRequest(http.MethodPost, "/", nil)
	refererReq.Host = "example.com"
	refererReq.Header.Set("Referer", "http://example.com/path")
	if !sameOriginRequest(refererReq) {
		t.Fatal("expected sameOriginRequest to accept matching Referer")
	}

	mismatchReq := httptest.NewRequest(http.MethodPost, "/", nil)
	mismatchReq.Host = "example.com"
	mismatchReq.Header.Set("Origin", "https://other.example.com")
	if sameOriginRequest(mismatchReq) {
		t.Fatal("expected sameOriginRequest to reject mismatched origin")
	}

	if sameOriginRequest(httptest.NewRequest(http.MethodPost, "/", nil)) {
		t.Fatal("expected sameOriginRequest to reject missing origin and referer")
	}

	forwardedReq := httptest.NewRequest(http.MethodPost, "/", nil)
	forwardedReq.Header.Set("X-Forwarded-Proto", "HTTPS")
	if got := requestScheme(forwardedReq); got != "https" {
		t.Fatalf("requestScheme(forwarded) = %q, want https", got)
	}

	tlsReq := httptest.NewRequest(http.MethodPost, "/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	if got := requestScheme(tlsReq); got != "https" {
		t.Fatalf("requestScheme(tls) = %q, want https", got)
	}

	if got := requestScheme(httptest.NewRequest(http.MethodPost, "/", nil)); got != "http" {
		t.Fatalf("requestScheme(default) = %q, want http", got)
	}

	matchReq := httptest.NewRequest(http.MethodPost, "/", nil)
	matchReq.Host = "example.com"
	if !originMatchesRequest("http://example.com/ok", matchReq) {
		t.Fatal("expected originMatchesRequest to accept matching origin")
	}
	if originMatchesRequest("://bad", matchReq) || originMatchesRequest("/relative", matchReq) {
		t.Fatal("expected originMatchesRequest to reject invalid origins")
	}
}

func TestValidateCSRFRequestBranches(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set(csrfHeaderName, "valid-token")
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "valid-token"})
		res := httptest.NewRecorder()

		if !validateCSRFRequest(server, res, req) {
			t.Fatalf("expected validateCSRFRequest to succeed, got body %q", res.Body.String())
		}
	})

	t.Run("same origin failed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "http://other.example.com")
		res := httptest.NewRecorder()

		if validateCSRFRequest(server, res, req) {
			t.Fatal("expected same-origin validation failure")
		}
		if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "csrf_same_origin_failed") {
			t.Fatalf("unexpected same-origin failure response: code=%d body=%q", res.Code, res.Body.String())
		}
	})

	t.Run("cookie missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set(csrfHeaderName, "valid-token")
		res := httptest.NewRecorder()

		if validateCSRFRequest(server, res, req) {
			t.Fatal("expected missing-cookie validation failure")
		}
		if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "csrf_cookie_missing") {
			t.Fatalf("unexpected missing-cookie response: code=%d body=%q", res.Code, res.Body.String())
		}
	})

	t.Run("token missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(`{}`))
		req.Host = "example.com"
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "valid-token"})
		res := httptest.NewRecorder()

		if validateCSRFRequest(server, res, req) {
			t.Fatal("expected missing-token validation failure")
		}
		if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "csrf_token_missing") {
			t.Fatalf("unexpected missing-token response: code=%d body=%q", res.Code, res.Body.String())
		}
	})

	t.Run("token invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/app/products", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set(csrfHeaderName, "different-token")
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "valid-token"})
		res := httptest.NewRecorder()

		if validateCSRFRequest(server, res, req) {
			t.Fatal("expected token-mismatch validation failure")
		}
		if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "csrf_token_invalid") {
			t.Fatalf("unexpected invalid-token response: code=%d body=%q", res.Code, res.Body.String())
		}
	})
}

func TestGenerateCSRFToken(t *testing.T) {
	token := generateCSRFToken()
	if token == "" {
		t.Fatal("expected generated token")
	}
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		t.Fatalf("generateCSRFToken() returned invalid base64url token: %v", err)
	}
}

func TestCSRFCookieCanSeedPageRequests(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app/products", nil)
	req.Header.Set("Host", "example.com")
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "prefilled-csrf"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	if strings.Contains(res.Header().Get("Set-Cookie"), csrfCookieName+"=") {
		t.Fatalf("expected no replacement csrf Set-Cookie header, got %q", res.Header().Get("Set-Cookie"))
	}
	if !strings.Contains(res.Body.String(), `"csrf":"prefilled-csrf"`) {
		t.Fatalf("expected bootstrap payload to reuse csrf token, got %q", res.Body.String())
	}
}
