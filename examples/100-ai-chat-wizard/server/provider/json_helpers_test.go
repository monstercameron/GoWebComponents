package provider

import "testing"

func TestExtractJSONObject(t *testing.T) {
	if got := extractJSONObject(""); got != `{"memories":[]}` {
		t.Fatalf("expected empty extractJSONObject fallback payload, got %q", got)
	}
	if got := extractJSONObject("```json\n{\"memories\":[{\"summary\":\"x\"}]}\n```"); got != `{"memories":[{"summary":"x"}]}` {
		t.Fatalf("expected fenced JSON to be unwrapped, got %q", got)
	}
	if got := extractJSONObject("prefix text {\"memories\":[]} suffix"); got != `{"memories":[]}` {
		t.Fatalf("expected surrounding prose to be stripped, got %q", got)
	}
	if got := extractJSONObject("not-json"); got != "not-json" {
		t.Fatalf("expected non-object source to pass through, got %q", got)
	}
}
