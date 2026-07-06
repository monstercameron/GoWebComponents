package logging

import (
	"errors"
	"testing"
)

func TestRedactInteractionValue(parseT *testing.T) {
	if parseGot := redactInteractionValue("password", "super-secret"); parseGot != redactedInteractionValue {
		parseT.Fatalf("expected password fields to be redacted, got %q", parseGot)
	}
	if parseGot2 := redactInteractionValue(" Password ", "super-secret"); parseGot2 != redactedInteractionValue {
		parseT.Fatalf("expected password fields to be redacted case-insensitively, got %q", parseGot2)
	}
	if parseGot3 := redactInteractionValue("text", "visible-value"); parseGot3 != "visible-value" {
		parseT.Fatalf("expected non-password values to pass through, got %q", parseGot3)
	}
}

func TestTelemetryRedactionScrubsConfiguredLogFields(parseT *testing.T) {
	parsePrevious := CurrentTelemetryRedaction()
	ConfigureTelemetryRedaction(RedactionPolicy{Fields: []string{"email", "token"}})
	parseT.Cleanup(func() { ConfigureTelemetryRedaction(parsePrevious) })

	parseRecord := buildLogRecord(nil, "info", "auth", "login", map[string]any{
		"email": "cam@example.com",
		"profile": map[string]any{
			"token": "secret-token",
			"name":  "Cam",
		},
	})
	parseAttrs := parseRecord["attributes"].(map[string]any)
	if parseAttrs["email"] != redactedInteractionValue {
		parseT.Fatalf("expected email to be redacted, got %#v", parseAttrs["email"])
	}
	parseProfile := parseAttrs["profile"].(map[string]any)
	if parseProfile["token"] != redactedInteractionValue || parseProfile["name"] != "Cam" {
		parseT.Fatalf("unexpected nested redaction result: %#v", parseProfile)
	}
}

// TestBuildLogRecordRedactsWalkedStructFields closes the #88 struct-field
// redaction gap end-to-end: a struct log value is walked into a map and its
// sensitive field is scrubbed in the FINAL emitted record (not just via the
// standalone helpers). It also documents the redaction boundary — the free-text
// message is emitted verbatim, so PII must live in fields, not the message.
func TestBuildLogRecordRedactsWalkedStructFields(parseT *testing.T) {
	parsePrevious := CurrentTelemetryRedaction()
	ConfigureTelemetryRedaction(RedactionPolicy{Fields: []string{"password"}})
	parseT.Cleanup(func() { ConfigureTelemetryRedaction(parsePrevious) })

	type parseCreds struct {
		User     string
		Password string `json:"password"`
	}
	parseRecord := buildLogRecord(nil, "info", "auth", "login", map[string]any{
		"account": parseCreds{User: "bob", Password: "s3cret"},
	})
	parseAttrs := parseRecord["attributes"].(map[string]any)
	parseAccount, parseOk := parseAttrs["account"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("struct field must normalize to a redactable map, got %T", parseAttrs["account"])
	}
	if parseAccount["password"] != redactedInteractionValue {
		parseT.Fatalf("struct password must be redacted end-to-end, got %#v", parseAccount["password"])
	}
	if parseAccount["User"] != "bob" {
		parseT.Fatalf("non-sensitive struct field must survive, got %#v", parseAccount["User"])
	}
	// Redaction boundary: the message passes through unredacted (documented #88 gap).
	if parseRecord["message"] != "login" {
		parseT.Fatalf("message should pass through verbatim, got %#v", parseRecord["message"])
	}
}

func TestTelemetryRedactionFailureDropsField(parseT *testing.T) {
	parsePrevious := CurrentTelemetryRedaction()
	ConfigureTelemetryRedaction(RedactionPolicy{
		Fields: []string{"token"},
		Redact: func(RedactionContext) (any, error) {
			return nil, errors.New("redactor unavailable")
		},
	})
	parseT.Cleanup(func() { ConfigureTelemetryRedaction(parsePrevious) })

	parseAttrs := buildLogAttributes(map[string]any{"token": "secret", "safe": "visible"})
	if _, parseFound := parseAttrs["token"]; parseFound {
		parseT.Fatalf("expected failed redaction to drop token field, got %#v", parseAttrs)
	}
	if parseAttrs["safe"] != "visible" {
		parseT.Fatalf("expected non-redacted field to remain, got %#v", parseAttrs)
	}
}
