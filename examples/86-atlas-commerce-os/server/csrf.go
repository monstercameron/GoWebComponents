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

func ensureCSRFCookie(parseW http.ResponseWriter, parseR *http.Request) string {
	if parseCookie, parseErr := parseR.Cookie(csrfCookieName); parseErr == nil && strings.TrimSpace(parseCookie.Value) != "" {
		return parseCookie.Value
	}
	parseToken := generateCSRFToken()
	http.SetCookie(parseW, &http.Cookie{
		Name:     csrfCookieName,
		Value:    parseToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return parseToken
}

func validateCSRFRequest(parseS *atlasServer, parseW http.ResponseWriter, parseR *http.Request) bool {
	if !sameOriginRequest(parseR) {
		parseS.writeError(parseW, http.StatusForbidden, "csrf_same_origin_failed", fmt.Errorf("same-origin validation failed"))
		return false
	}
	parseCookie, parseErr := parseR.Cookie(csrfCookieName)
	if parseErr != nil || strings.TrimSpace(parseCookie.Value) == "" {
		parseS.writeError(parseW, http.StatusForbidden, "csrf_cookie_missing", fmt.Errorf("csrf cookie missing"))
		return false
	}
	parseRequestToken, parseErr := requestCSRFToken(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusForbidden, "csrf_token_missing", parseErr)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(parseCookie.Value), []byte(parseRequestToken)) != 1 {
		parseS.writeError(parseW, http.StatusForbidden, "csrf_token_invalid", fmt.Errorf("csrf token mismatch"))
		return false
	}
	return true
}

func requestCSRFToken(parseR *http.Request) (string, error) {
	if parseToken := strings.TrimSpace(parseR.Header.Get(csrfHeaderName)); parseToken != "" {
		return parseToken, nil
	}
	parseContentType := strings.ToLower(strings.TrimSpace(parseR.Header.Get("Content-Type")))
	if strings.Contains(parseContentType, "application/json") {
		return "", fmt.Errorf("missing %s header", csrfHeaderName)
	}
	if parseErr := parseR.ParseForm(); parseErr != nil {
		return "", fmt.Errorf("parse form body: %w", parseErr)
	}
	parseToken2 := strings.TrimSpace(parseR.Form.Get(csrfFormFieldName))
	if parseToken2 == "" {
		return "", fmt.Errorf("missing %s form field", csrfFormFieldName)
	}
	return parseToken2, nil
}

func sameOriginRequest(parseR *http.Request) bool {
	if parseOrigin := strings.TrimSpace(parseR.Header.Get("Origin")); parseOrigin != "" {
		return originMatchesRequest(parseOrigin, parseR)
	}
	if parseReferer := strings.TrimSpace(parseR.Header.Get("Referer")); parseReferer != "" {
		return originMatchesRequest(parseReferer, parseR)
	}
	return false
}

func originMatchesRequest(parseRaw string, parseR *http.Request) bool {
	parseParsed, parseErr := url.Parse(parseRaw)
	if parseErr != nil {
		return false
	}
	if parseParsed.Scheme == "" || parseParsed.Host == "" {
		return false
	}
	return strings.EqualFold(parseParsed.Scheme, requestScheme(parseR)) && strings.EqualFold(parseParsed.Host, parseR.Host)
}

func requestScheme(parseR *http.Request) string {
	if parseForwarded := strings.TrimSpace(parseR.Header.Get("X-Forwarded-Proto")); parseForwarded != "" {
		return strings.ToLower(parseForwarded)
	}
	if parseR.TLS != nil {
		return "https"
	}
	return "http"
}

func generateCSRFToken() string {
	parseRaw := make([]byte, 32)
	if _, parseErr := rand.Read(parseRaw); parseErr == nil {
		return base64.RawURLEncoding.EncodeToString(parseRaw)
	}
	return base64.RawURLEncoding.EncodeToString([]byte("atlas-commerce-os-fallback-csrf-token"))
}
