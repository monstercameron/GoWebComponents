package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	csrfCookieName    = "atlas_csrf"
	csrfHeaderName    = "X-CSRF-Token"
	csrfFormFieldName = "csrf_token"
)

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}
	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

func validateCSRFRequest(s *atlasServer, w http.ResponseWriter, r *http.Request) bool {
	if !sameOriginRequest(r) {
		s.writeError(w, http.StatusForbidden, "csrf_same_origin_failed", fmt.Errorf("same-origin validation failed"))
		return false
	}
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		s.writeError(w, http.StatusForbidden, "csrf_cookie_missing", fmt.Errorf("csrf cookie missing"))
		return false
	}
	requestToken, err := requestCSRFToken(r)
	if err != nil {
		s.writeError(w, http.StatusForbidden, "csrf_token_missing", err)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(requestToken)) != 1 {
		s.writeError(w, http.StatusForbidden, "csrf_token_invalid", fmt.Errorf("csrf token mismatch"))
		return false
	}
	return true
}

func requestCSRFToken(r *http.Request) (string, error) {
	if token := strings.TrimSpace(r.Header.Get(csrfHeaderName)); token != "" {
		return token, nil
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/json") {
		return "", fmt.Errorf("missing %s header", csrfHeaderName)
	}
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("parse form body: %w", err)
	}
	token := strings.TrimSpace(r.Form.Get(csrfFormFieldName))
	if token == "" {
		return "", fmt.Errorf("missing %s form field", csrfFormFieldName)
	}
	return token, nil
}

func sameOriginRequest(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return originMatchesRequest(origin, r)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return originMatchesRequest(referer, r)
	}
	return false
}

func originMatchesRequest(raw string, r *http.Request) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, requestScheme(r)) && strings.EqualFold(parsed.Host, r.Host)
}

func requestScheme(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		return strings.ToLower(forwarded)
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func generateCSRFToken() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err == nil {
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return base64.RawURLEncoding.EncodeToString([]byte("atlas-commerce-os-fallback-csrf-token"))
}
