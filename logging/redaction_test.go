package logging

import "testing"

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
