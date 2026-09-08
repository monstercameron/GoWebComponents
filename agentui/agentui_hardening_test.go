package agentui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/agentui"
)

// TestRenderToleratesNilRenderSpec pins that a component registered with a nil
// Render (a wiring mistake that still passes Validate) degrades to an empty node
// instead of panicking the whole render.
func TestRenderToleratesNilRenderSpec(parseT *testing.T) {
	parseRegistry := agentui.NewRegistry()
	parseRegistry.Register(agentui.ComponentSpec{Name: "broken"}) // Render left nil

	parseNode, parseErr := parseRegistry.Render(agentui.Node{Type: "broken", Text: "hi"})
	if parseErr != nil {
		parseT.Fatalf("validate/render returned error: %v", parseErr)
	}
	if parseNode == nil {
		parseT.Fatal("expected a non-nil fallback node for a nil-Render spec")
	}
}

// TestRegistryConcurrentRegisterAndRead exercises Register racing with reads to
// confirm the mutex is wired (best-effort without the race detector, which is
// unavailable on this platform — it at least crashes deterministically if a
// concurrent map read/write slips through).
func TestRegistryConcurrentRegisterAndRead(parseT *testing.T) {
	parseRegistry := agentui.NewRegistry()
	parseRegistry.Register(agentui.ComponentSpec{Name: "box", AllowedProps: []string{"class"}})

	parseDone := make(chan struct{})
	go func() {
		for parseI := 0; parseI < 500; parseI++ {
			parseRegistry.Register(agentui.ComponentSpec{Name: "box"})
		}
		close(parseDone)
	}()
	for parseI := 0; parseI < 500; parseI++ {
		_ = parseRegistry.Allowed("box")
		_ = parseRegistry.Catalog()
	}
	<-parseDone
}
