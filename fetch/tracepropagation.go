package fetch

import (
	"net/url"
	"strings"
)

// shouldInjectTraceparent decides whether a request gets a generated traceparent header. It
// returns false when the caller already set one (any letter case), and otherwise only when the
// request URL resolves to the page origin or to an origin listed in parseAllowedOrigins.
// An empty parsePageOrigin means the page origin is unknown, so only relative URLs qualify.
func shouldInjectTraceparent(parseHeaders map[string]any, parseURL string, parsePageOrigin string, parseAllowedOrigins []string) bool {
	for parseKey := range parseHeaders {
		if strings.EqualFold(strings.TrimSpace(parseKey), "traceparent") {
			return false
		}
	}
	parseOrigin, isKnown := getRequestOrigin(parseURL, parsePageOrigin)
	if !isKnown {
		return false
	}
	if parseOrigin == "" || strings.EqualFold(parseOrigin, normalizeOrigin(parsePageOrigin)) {
		return true
	}
	for _, parseAllowed := range parseAllowedOrigins {
		if strings.EqualFold(normalizeOrigin(parseAllowed), parseOrigin) {
			return true
		}
	}
	return false
}

// getRequestOrigin returns the scheme://host origin of a request URL. An empty origin with
// true means a relative URL, which always targets the page origin. The second result is false
// for URLs that cannot be parsed or that name no usable origin.
func getRequestOrigin(parseURL string, parsePageOrigin string) (string, bool) {
	parseRaw := strings.TrimSpace(parseURL)
	parseParsed, parseErr := url.Parse(parseRaw)
	if parseErr != nil {
		return "", false
	}
	if parseParsed.Host == "" {
		if parseParsed.Scheme != "" {
			return "", false
		}
		return "", true
	}
	parseScheme := strings.ToLower(parseParsed.Scheme)
	if parseScheme == "" {
		// Scheme-relative URL: it inherits the page scheme.
		parsePage, parsePageErr := url.Parse(parsePageOrigin)
		if parsePageErr != nil || parsePage.Scheme == "" {
			return "", false
		}
		parseScheme = strings.ToLower(parsePage.Scheme)
	}
	if parseScheme != "http" && parseScheme != "https" {
		return "", false
	}
	return parseScheme + "://" + strings.ToLower(parseParsed.Host), true
}

// normalizeOrigin lowercases an origin and drops any trailing slash or path.
func normalizeOrigin(parseOrigin string) string {
	parseParsed, parseErr := url.Parse(strings.TrimSpace(parseOrigin))
	if parseErr != nil || parseParsed.Scheme == "" || parseParsed.Host == "" {
		return strings.ToLower(strings.TrimRight(strings.TrimSpace(parseOrigin), "/"))
	}
	return strings.ToLower(parseParsed.Scheme) + "://" + strings.ToLower(parseParsed.Host)
}
