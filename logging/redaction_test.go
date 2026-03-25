package logging

import "testing"

func TestRedactInteractionValue(t *testing.T) {
	if got := redactInteractionValue("password", "super-secret"); got != redactedInteractionValue {
		t.Fatalf("expected password fields to be redacted, got %q", got)
	}
	if got := redactInteractionValue(" Password ", "super-secret"); got != redactedInteractionValue {
		t.Fatalf("expected password fields to be redacted case-insensitively, got %q", got)
	}
	if got := redactInteractionValue("text", "visible-value"); got != "visible-value" {
		t.Fatalf("expected non-password values to pass through, got %q", got)
	}
}
