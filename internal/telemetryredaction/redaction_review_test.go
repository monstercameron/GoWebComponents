package telemetryredaction

import (
	"strings"
	"testing"
)

// configureReviewPolicy installs a redact-everything-matching policy and restores the zero
// policy when the test ends.
func configureReviewPolicy(parseT *testing.T, parseFields ...string) {
	parseT.Helper()
	Configure(Policy{Fields: parseFields})
	parseT.Cleanup(func() { Configure(Policy{}) })
}

// TestFieldMatchesFinalSegmentOfDottedKey pins the documented rule "matched against the final
// path segment" for flat, OpenTelemetry-style dotted attribute keys: a key such as
// "http.request.header.authorization" is a secret even though no one-segment key exists.
func TestFieldMatchesFinalSegmentOfDottedKey(parseT *testing.T) {
	configureReviewPolicy(parseT, "authorization", "Api-Key")
	for _, parseKey := range []string{
		"http.request.header.authorization",
		"HTTP.Request.Header.Authorization",
		"client.api_key",
		"x.API-KEY",
	} {
		parseGot, parseKeep := Field("log.attributes."+parseKey, parseKey, "Bearer sekret")
		if !parseKeep || parseGot != RedactedValue {
			parseT.Errorf("key %q was not redacted: %v (keep=%v)", parseKey, parseGot, parseKeep)
		}
	}
	if parseGot, _ := Field("log.attributes.http.method", "http.method", "GET"); parseGot != "GET" {
		parseT.Errorf("unrelated dotted key was redacted: %v", parseGot)
	}
}

// TestValueRedactsSecretsInsideJSONStrings pins that a string value that is itself a JSON
// document is scanned: {"body":"{\"password\":\"hunter2\"}"} would otherwise leak the password
// because the key-based policy only sees the outer key.
func TestValueRedactsSecretsInsideJSONStrings(parseT *testing.T) {
	configureReviewPolicy(parseT, "password", "token")
	parseInput := map[string]any{
		"body":  `{"user":"ann","password":"hunter2","nested":{"Token":"abc123"}}`,
		"list":  `[{"password":"p4ss"},{"ok":1}]`,
		"plain": "password=hunter2 is just text",
		"num":   `{"id":12345678901234567890}`,
	}
	parseOut := Value("evt", parseInput).(map[string]any)
	for _, parseKey := range []string{"body", "list"} {
		parseText, _ := parseOut[parseKey].(string)
		for _, parseSecret := range []string{"hunter2", "abc123", "p4ss"} {
			if strings.Contains(parseText, parseSecret) {
				parseT.Errorf("%s still contains %q: %s", parseKey, parseSecret, parseText)
			}
		}
		if !strings.Contains(parseText, RedactedValue) {
			parseT.Errorf("%s has no redaction marker: %s", parseKey, parseText)
		}
	}
	if !strings.Contains(parseOut["body"].(string), `"user":"ann"`) {
		parseT.Errorf("non-secret fields were lost from body: %v", parseOut["body"])
	}
	// Text that is not JSON, and JSON without a matching key, must pass through byte-for-byte.
	if parseOut["plain"] != "password=hunter2 is just text" {
		parseT.Errorf("plain text was modified: %v", parseOut["plain"])
	}
	if parseOut["num"] != `{"id":12345678901234567890}` {
		parseT.Errorf("untouched JSON was rewritten (large integer precision?): %v", parseOut["num"])
	}
}
