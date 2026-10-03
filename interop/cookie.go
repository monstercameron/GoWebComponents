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
	if sanitizeCookieName(parseName) == "" {
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
	// The value is percent-encoded, but the name is emitted raw. Strip the
	// structural/header-injection bytes (';', '=', CR, LF, control chars) that
	// are illegal in a cookie name anyway, so a crafted name cannot corrupt the
	// Set-Cookie string or inject a second directive. Valid names are unchanged,
	// so previously written cookies still read back.
	parseSafeName := sanitizeCookieName(parseName)
	parseEncoded := url.QueryEscape(parseValue)
	parseParts := []string{parseSafeName + "=" + parseEncoded}

	if parsePath := sanitizeCookieAttribute(parseOpts.Path); parsePath != "" {
		parseParts = append(parseParts, "Path="+parsePath)
	}
	if parseDomain := sanitizeCookieAttribute(parseOpts.Domain); parseDomain != "" {
		parseParts = append(parseParts, "Domain="+parseDomain)
	}
	if parseOpts.MaxAge != 0 {
		parseParts = append(parseParts, "Max-Age="+strconv.Itoa(parseOpts.MaxAge))
	}

	parseSecure := parseOpts.Secure
	if normalizeCookieSameSite(parseOpts.SameSite) == SameSiteNone {
		parseSecure = true
	}
	if parseSecure {
		parseParts = append(parseParts, "Secure")
	}
	if parseSameSite := normalizeCookieSameSite(parseOpts.SameSite); parseSameSite != "" {
		parseParts = append(parseParts, "SameSite="+string(parseSameSite))
	}

	return strings.Join(parseParts, "; ")
}

// sanitizeCookieAttribute drops control bytes and ";" from a Path or Domain
// value so it cannot terminate its directive and inject another one.
func sanitizeCookieAttribute(parseValue string) string {
	var parseBuilder strings.Builder
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		parseC := parseValue[parseIndex]
		if parseC < 0x20 || parseC == 0x7f || parseC == ';' {
			continue
		}
		parseBuilder.WriteByte(parseC)
	}
	return strings.TrimSpace(parseBuilder.String())
}

// normalizeCookieSameSite maps a SameSite value onto its canonical constant and
// returns the empty value for anything the cookie grammar does not allow.
func normalizeCookieSameSite(parseValue SameSite) SameSite {
	switch strings.ToLower(strings.TrimSpace(string(parseValue))) {
	case "lax":
		return SameSiteLax
	case "strict":
		return SameSiteStrict
	case "none":
		return SameSiteNone
	}
	return ""
}

// sanitizeCookieName drops bytes that are illegal in a cookie name and could
// otherwise break out of the name token in the Set-Cookie header.
func sanitizeCookieName(parseName string) string {
	var parseBuilder strings.Builder
	parseBuilder.Grow(len(parseName))
	for parseIndex := 0; parseIndex < len(parseName); parseIndex++ {
		parseC := parseName[parseIndex]
		if parseC <= ' ' || parseC == ';' || parseC == '=' || parseC == ',' || parseC == 0x7f {
			continue
		}
		parseBuilder.WriteByte(parseC)
	}
	return parseBuilder.String()
}

// parseCookieHeader parses a raw "a=1; b=2" cookie string and returns the
// URL-decoded value of the named cookie and whether it was found.
func parseCookieHeader(parseRaw string, parseName string) (string, bool) {
	for parsePair := range strings.SplitSeq(parseRaw, ";") {
		parsePair = strings.TrimSpace(parsePair)
		if parsePair == "" {
			continue
		}
		before, after, ok := strings.Cut(parsePair, "=")
		if !ok {
			continue
		}
		parsePairName := strings.TrimSpace(before)
		parsePairValue := strings.TrimSpace(after)
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
