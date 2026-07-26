package runtime

import "testing"

// Probe C: removing an event handler whose prop name is NOT in propMetaCache.
// getPropMeta returns shouldReset=false for unknown names, so the removal path
// calls RemoveAttribute instead of SetProperty(name, nil) — but the handler was
// installed as a PROPERTY.
func TestProbe_UnlistedEventHandlerRemoval(parseT *testing.T) {
	parseCases := []string{
		"onmouseover", "onmouseout", "onpointerenter", "onpointerleave",
		"onpointercancel", "onkeypress", "onpaste", "oncopy", "ontoggle",
		"onfocusin", "onfocusout", "onauxclick", "onbeforeinput",
		"ondragenter", "ondragleave", "ontouchcancel", "onanimationstart",
		"ontransitionstart", "onselect", "oninvalid",
	}
	// Controls: names that ARE in propMetaCache.
	parseControls := []string{"onclick", "oninput", "onblur"}

	parseCheck := func(parseName string) bool {
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
		parseApp := parseAdapter.CreateElement("div")

		parseHandler := func() {}
		parseWith := CreateElement("section", nil,
			CreateElement("button", map[string]any{"id": "b", parseName: parseHandler}, "x"),
		)
		parseWithout := CreateElement("section", nil,
			CreateElement("button", map[string]any{"id": "b"}, "x"),
		)

		renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseWith)
		parseBtn := findNodeByID(parseApp, "b")
		if parseBtn == nil || parseBtn.properties[parseName] == nil {
			parseT.Fatalf("setup: %s was not installed as a property", parseName)
		}

		renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseWithout)
		return parseBtn.properties[parseName] != nil // true == still attached == leaked
	}

	for _, parseName := range parseControls {
		if parseCheck(parseName) {
			parseT.Errorf("control %s unexpectedly leaked", parseName)
		}
	}
	parseLeaked := []string{}
	for _, parseName := range parseCases {
		if parseCheck(parseName) {
			parseLeaked = append(parseLeaked, parseName)
		}
	}
	if len(parseLeaked) > 0 {
		parseT.Errorf("handlers still attached after removal (%d/%d): %v",
			len(parseLeaked), len(parseCases), parseLeaked)
	}
}
