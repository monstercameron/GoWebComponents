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
