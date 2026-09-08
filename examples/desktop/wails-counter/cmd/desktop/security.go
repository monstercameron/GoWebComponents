package main

import (
	"net/http"
	"net/url"
	"strings"
)

const desktopContentPolicy = "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none'"

// isDesktopOrigin accepts only the exact packaged Windows asset origin.
// The pinned runtime's broader hostname-prefix routing is not an authorization rule.
func isDesktopOrigin(parseRaw string) bool {
	parseURL, parseErr := url.Parse(parseRaw)
	return parseErr == nil && parseURL.Scheme == "http" && parseURL.Host == "wails.localhost" && parseURL.User == nil
}

// guardDesktopAssets rejects foreign-origin access before Wails' runtime handlers.
// Runtime requests must carry a browser-provided local origin or referrer. A page
// suppressing both cannot access native services. This is not protection against
// hostile code already running within the trusted local document.
func guardDesktopAssets(parseNext http.Handler) http.Handler {
	return http.HandlerFunc(func(parseWriter http.ResponseWriter, parseRequest *http.Request) {
		parseWriter.Header().Set("Content-Security-Policy", desktopContentPolicy)
		parseWriter.Header().Set("X-Content-Type-Options", "nosniff")
		parseWriter.Header().Set("Referrer-Policy", "same-origin")
		parseOrigin := parseRequest.Header.Get("Origin")
		parseReferer := parseRequest.Referer()
		if parseRequest.Host != "wails.localhost" ||
			(parseOrigin != "" && !isDesktopOrigin(parseOrigin)) ||
			(parseReferer != "" && !isDesktopOrigin(parseReferer)) {
			http.Error(parseWriter, "foreign desktop origin denied", http.StatusForbidden)
			return
		}
		// Runtime JS assets may be loaded by the bootstrap; executable runtime
		// endpoints must also prove a same-origin initiating document.
		isRuntime := strings.HasPrefix(parseRequest.URL.Path, "/wails/") && !strings.HasSuffix(parseRequest.URL.Path, ".js")
		if isRuntime && parseOrigin == "" && parseReferer == "" {
			http.Error(parseWriter, "desktop runtime requires a local initiating document", http.StatusForbidden)
			return
		}
		parseNext.ServeHTTP(parseWriter, parseRequest)
	})
}
