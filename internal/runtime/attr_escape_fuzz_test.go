//go:build !js || !wasm

package runtime

import (
	"strings"
	"testing"
)

// FuzzSerializeSSRAttrBreakout proves the attribute serializer is breakout-safe:
// for any attribute name and string value, a successfully serialized attribute
// has the exact form name="escaped-value" — so the only double quotes are the
// two delimiters and no raw < or > survives in the value. That guarantees a
// crafted value can neither close the attribute early (injecting a new attribute
// such as onmouseover=) nor open a markup tag. Invalid names are rejected, never
// emitted. The serializer must never panic.
func FuzzSerializeSSRAttrBreakout(parseF *testing.F) {
	parseF.Add("title", `a" onmouseover="alert(1)`)
	parseF.Add("href", `"><script>alert(1)</script>`)
	parseF.Add("data-x", "plain")
	parseF.Add("a b", "value")
	parseF.Add("", "value")
	parseF.Add("title", "&<>\"'\x00")

	parseF.Fuzz(func(parseT *testing.T, parseName, parseValue string) {
		parseOut, parseOk := serializeSSRAttr(parseName, parseValue)
		if !parseOk {
			if parseOut != "" {
				parseT.Fatalf("rejected attr returned non-empty %q", parseOut)
			}
			return
		}

		// A rejected name must never reach here; an accepted one must be intact.
		parsePrefix := parseName + `="`
		if !strings.HasPrefix(parseOut, parsePrefix) || !strings.HasSuffix(parseOut, `"`) {
			parseT.Fatalf("unexpected shape: name=%q out=%q", parseName, parseOut)
		}

		// Exactly two double quotes — the delimiters — so the value cannot have
		// closed the attribute early.
		if parseQuotes := strings.Count(parseOut, `"`); parseQuotes != 2 {
			parseT.Fatalf("value broke out of quotes (%d quotes): name=%q out=%q", parseQuotes, parseName, parseOut)
		}

		// No raw angle brackets anywhere — escaping must have neutralized them.
		parseBody := strings.TrimSuffix(strings.TrimPrefix(parseOut, parsePrefix), `"`)
		if strings.ContainsAny(parseBody, "<>") {
			parseT.Fatalf("raw angle bracket survived in value: name=%q out=%q", parseName, parseOut)
		}
	})
}
