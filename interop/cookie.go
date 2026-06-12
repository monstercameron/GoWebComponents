package interop

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// SameSite controls the SameSite cookie attribute.
type SameSite string

const (
	// SameSiteLax sets the SameSite attribute to Lax.
	SameSiteLax SameSite = "Lax"
	// SameSiteStrict sets the SameSite attribute to Strict.
	SameSiteStrict SameSite = "Strict"
	// SameSiteNone sets the SameSite attribute to None (requires Secure).
	SameSiteNone SameSite = "None"
)

// CookieOptions configures optional cookie attributes.
type CookieOptions struct {
	Path     string
	Domain   string
	MaxAge   int // seconds; 0 = session cookie; negative = expire now
	Secure   bool
	SameSite SameSite
}

// GetCookie reads a cookie by name from document.cookie, returning the
// URL-decoded value and a found bool. An error is returned when the underlying
// platform read fails.
func GetCookie(parseName string) (string, bool, error) {
	parseRaw, parseReadErr := readRawCookies()
	if parseReadErr != nil {
		return "", false, parseReadErr
	}
	parseDecoded, parseFound := parseCookieHeader(parseRaw, parseName)
	return parseDecoded, parseFound, nil
}

// SetCookie writes a cookie with the given name, value, and options. Passing
// no options produces a session cookie with no path, domain, or security
// attributes. When SameSite is SameSiteNone the Secure flag is forced on.
func SetCookie(parseName string, parseValue string, parseOptions ...CookieOptions) error {
	if strings.TrimSpace(parseName) == "" {
		return wrapError("SetCookie", parseName, CodeInvalid, errors.New("cookie name must not be empty"))
	}
	parseOpts := CookieOptions{}
	if len(parseOptions) > 0 {
		parseOpts = parseOptions[0]
	}
	parseSerialized := buildCookieString(parseName, parseValue, parseOpts)
	return writeRawCookie(parseSerialized)
}

// ExpireCookie deletes a cookie by setting Max-Age to -1. Any options passed
// should match the path and domain used when the cookie was set.
func ExpireCookie(parseName string, parseOptions ...CookieOptions) error {
	parseOpts := CookieOptions{MaxAge: -1}
	if len(parseOptions) > 0 {
		parseExisting := parseOptions[0]
		parseExisting.MaxAge = -1
		parseOpts = parseExisting
	}
	return SetCookie(parseName, "", parseOpts)
}

// buildCookieString serializes a Set-Cookie header value from its parts.
// SameSite=None automatically adds Secure. This function is pure Go and is
// used by both wasm and native builds.
func buildCookieString(parseName string, parseValue string, parseOpts CookieOptions) string {
	parseEncoded := url.QueryEscape(parseValue)
	parseParts := []string{parseName + "=" + parseEncoded}

	if parseOpts.Path != "" {
		parseParts = append(parseParts, "Path="+parseOpts.Path)
	}
	if parseOpts.Domain != "" {
		parseParts = append(parseParts, "Domain="+parseOpts.Domain)
	}
	if parseOpts.MaxAge != 0 {
		parseParts = append(parseParts, "Max-Age="+strconv.Itoa(parseOpts.MaxAge))
	}

	parseSecure := parseOpts.Secure
	if parseOpts.SameSite == SameSiteNone {
		parseSecure = true
	}
	if parseSecure {
		parseParts = append(parseParts, "Secure")
	}
	if parseOpts.SameSite != "" {
		parseParts = append(parseParts, "SameSite="+string(parseOpts.SameSite))
	}

	return strings.Join(parseParts, "; ")
}

// parseCookieHeader parses a raw "a=1; b=2" cookie string and returns the
// URL-decoded value of the named cookie and whether it was found.
func parseCookieHeader(parseRaw string, parseName string) (string, bool) {
	for _, parsePair := range strings.Split(parseRaw, ";") {
		parsePair = strings.TrimSpace(parsePair)
		if parsePair == "" {
			continue
		}
		parseEq := strings.IndexByte(parsePair, '=')
		if parseEq < 0 {
			continue
		}
		parsePairName := strings.TrimSpace(parsePair[:parseEq])
		parsePairValue := strings.TrimSpace(parsePair[parseEq+1:])
		if parsePairName != parseName {
			continue
		}
		parseDecoded, parseDecodeErr := url.QueryUnescape(parsePairValue)
		if parseDecodeErr != nil {
			// Value was not URL-encoded; return raw value.
			return parsePairValue, true
		}
		return parseDecoded, true
	}
	return "", false
}
