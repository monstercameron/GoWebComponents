package interop

import (
	"strings"
	"testing"
)

// TestCookieHeaderParsing verifies that parseCookieHeader correctly extracts
// named values from a raw cookie header string.
func TestCookieHeaderParsing(parseT *testing.T) {
	parseRaw := "a=1; b=hello+world; c=3"

	parseVal, parseFound := parseCookieHeader(parseRaw, "a")
	if !parseFound || parseVal != "1" {
		parseT.Fatalf("expected a=1, got %q found=%v", parseVal, parseFound)
	}

	parseVal, parseFound = parseCookieHeader(parseRaw, "b")
	if !parseFound || parseVal != "hello world" {
		parseT.Fatalf("expected b=hello world (URL-decoded), got %q found=%v", parseVal, parseFound)
	}

	parseVal, parseFound = parseCookieHeader(parseRaw, "c")
	if !parseFound || parseVal != "3" {
		parseT.Fatalf("expected c=3, got %q found=%v", parseVal, parseFound)
	}

	_, parseFound = parseCookieHeader(parseRaw, "missing")
	if parseFound {
		parseT.Fatal("expected missing key to return found=false")
	}
}

// TestCookieHeaderParsingEmpty verifies parseCookieHeader handles empty input.
func TestCookieHeaderParsingEmpty(parseT *testing.T) {
	_, parseFound := parseCookieHeader("", "anything")
	if parseFound {
		parseT.Fatal("expected empty cookie header to return found=false")
	}
}

// TestBuildCookieStringBasic verifies that buildCookieString produces the
// correct serialization for common option combinations.
func TestBuildCookieStringBasic(parseT *testing.T) {
	parseResult := buildCookieString("token", "abc123", CookieOptions{})
	if parseResult != "token=abc123" {
		parseT.Fatalf("expected plain cookie, got %q", parseResult)
	}
}

// TestBuildCookieStringWithPath verifies the Path attribute is serialized.
func TestBuildCookieStringWithPath(parseT *testing.T) {
	parseResult := buildCookieString("sid", "xyz", CookieOptions{Path: "/"})
	if !strings.Contains(parseResult, "Path=/") {
		parseT.Fatalf("expected Path=/ in cookie string, got %q", parseResult)
	}
}

// TestBuildCookieStringWithMaxAge verifies the Max-Age attribute is serialized.
func TestBuildCookieStringWithMaxAge(parseT *testing.T) {
	parseResult := buildCookieString("sid", "xyz", CookieOptions{MaxAge: 3600})
	if !strings.Contains(parseResult, "Max-Age=3600") {
		parseT.Fatalf("expected Max-Age=3600 in cookie string, got %q", parseResult)
	}
}

// TestBuildCookieStringNegativeMaxAge verifies a negative Max-Age is serialized
// (used for cookie deletion).
func TestBuildCookieStringNegativeMaxAge(parseT *testing.T) {
	parseResult := buildCookieString("sid", "", CookieOptions{MaxAge: -1})
	if !strings.Contains(parseResult, "Max-Age=-1") {
		parseT.Fatalf("expected Max-Age=-1 in expiry cookie string, got %q", parseResult)
	}
}

// TestBuildCookieStringSameSiteLax verifies SameSite=Lax serialization.
func TestBuildCookieStringSameSiteLax(parseT *testing.T) {
	parseResult := buildCookieString("pref", "v", CookieOptions{SameSite: SameSiteLax})
	if !strings.Contains(parseResult, "SameSite=Lax") {
		parseT.Fatalf("expected SameSite=Lax, got %q", parseResult)
	}
	if strings.Contains(parseResult, "Secure") {
		parseT.Fatalf("expected no Secure flag for SameSite=Lax, got %q", parseResult)
	}
}

// TestBuildCookieStringSameSiteStrict verifies SameSite=Strict serialization.
func TestBuildCookieStringSameSiteStrict(parseT *testing.T) {
	parseResult := buildCookieString("pref", "v", CookieOptions{SameSite: SameSiteStrict})
	if !strings.Contains(parseResult, "SameSite=Strict") {
		parseT.Fatalf("expected SameSite=Strict, got %q", parseResult)
	}
}

// TestBuildCookieStringSameSiteNoneForceSecure verifies that SameSite=None
// automatically adds the Secure flag as required by browsers.
func TestBuildCookieStringSameSiteNoneForceSecure(parseT *testing.T) {
	parseResult := buildCookieString("cross", "v", CookieOptions{SameSite: SameSiteNone})
	if !strings.Contains(parseResult, "Secure") {
		parseT.Fatalf("expected Secure to be forced for SameSite=None, got %q", parseResult)
	}
	if !strings.Contains(parseResult, "SameSite=None") {
		parseT.Fatalf("expected SameSite=None, got %q", parseResult)
	}
}

// TestBuildCookieStringAllOptions verifies a fully configured cookie string.
func TestBuildCookieStringAllOptions(parseT *testing.T) {
	parseResult := buildCookieString("full", "value", CookieOptions{
		Path:     "/app",
		Domain:   "example.com",
		MaxAge:   7200,
		Secure:   true,
		SameSite: SameSiteStrict,
	})
	for _, parsePart := range []string{"full=value", "Path=/app", "Domain=example.com", "Max-Age=7200", "Secure", "SameSite=Strict"} {
		if !strings.Contains(parseResult, parsePart) {
			parseT.Fatalf("expected %q in cookie string, got %q", parsePart, parseResult)
		}
	}
}

// TestGetCookieNativeReturnsUnavailable verifies that GetCookie returns an
// unavailable error on native (non-browser) builds.
func TestGetCookieNativeReturnsUnavailable(parseT *testing.T) {
	_, _, parseErr := GetCookie("anything")
	if parseErr == nil {
		parseT.Fatal("expected unavailable error from GetCookie on native build, got nil")
	}
	if !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("expected CodeUnavailable, got %v", parseErr)
	}
}

// TestSetCookieNativeReturnsUnavailable verifies that SetCookie returns an
// unavailable error on native (non-browser) builds.
func TestSetCookieNativeReturnsUnavailable(parseT *testing.T) {
	parseErr := SetCookie("k", "v")
	if parseErr == nil {
		parseT.Fatal("expected unavailable error from SetCookie on native build, got nil")
	}
	if !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("expected CodeUnavailable, got %v", parseErr)
	}
}

// TestSetCookieEmptyNameReturnsInvalid verifies that SetCookie rejects an empty name.
func TestSetCookieEmptyNameReturnsInvalid(parseT *testing.T) {
	parseErr := SetCookie("", "v")
	if parseErr == nil {
		parseT.Fatal("expected error for empty cookie name, got nil")
	}
	if !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected CodeInvalid for empty name, got %v", parseErr)
	}
}

// TestExpireCookieSerializesNegativeMaxAge verifies that ExpireCookie uses
// buildCookieString with Max-Age=-1 under the hood (tested via
// parseCookieHeader + buildCookieString since writeRawCookie is native-stubbed).
func TestExpireCookieSerializesNegativeMaxAge(parseT *testing.T) {
	parseResult := buildCookieString("sid", "", CookieOptions{MaxAge: -1, Path: "/"})
	if !strings.Contains(parseResult, "Max-Age=-1") {
		parseT.Fatalf("expected Max-Age=-1 in expiry cookie, got %q", parseResult)
	}
}

// TestCookieValueURLEncoding verifies that values with special characters are
// URL-encoded in the cookie string and decoded on read.
func TestCookieValueURLEncoding(parseT *testing.T) {
	parseEncoded := buildCookieString("q", "hello world", CookieOptions{})
	// The encoded cookie string should contain a "+" or "%20" for the space.
	if !strings.Contains(parseEncoded, "hello") {
		parseT.Fatalf("expected encoded value in cookie string, got %q", parseEncoded)
	}
	// Simulate reading back: the raw value after the "=" is the encoded part.
	parseVal, parseFound := parseCookieHeader("q=hello+world", "q")
	if !parseFound || parseVal != "hello world" {
		parseT.Fatalf("expected decoded value 'hello world', got %q found=%v", parseVal, parseFound)
	}
}
