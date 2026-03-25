package provider

import "testing"

func TestExtractJSONObject(parseT *testing.T) {
	if parseGot := parseExtractJSONObject(""); parseGot != `{"memories":[]}` {
		parseT.Fatalf("expected empty extractJSONObject fallback payload, got %q", parseGot)
	}
	if parseGot2 := parseExtractJSONObject("```json\n{\"memories\":[{\"summary\":\"x\"}]}\n```"); parseGot2 != `{"memories":[{"summary":"x"}]}` {
		parseT.Fatalf("expected fenced JSON to be unwrapped, got %q", parseGot2)
	}
	if parseGot3 := parseExtractJSONObject("prefix text {\"memories\":[]} suffix"); parseGot3 != `{"memories":[]}` {
		parseT.Fatalf("expected surrounding prose to be stripped, got %q", parseGot3)
	}
	if parseGot4 := parseExtractJSONObject("not-json"); parseGot4 != "not-json" {
		parseT.Fatalf("expected non-object source to pass through, got %q", parseGot4)
	}
}
