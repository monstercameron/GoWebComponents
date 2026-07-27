//go:build !js || !wasm

package ui_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Atom scope during server-side rendering.
//
// THE BUG THESE PIN: the atom registry used to be reached through the
// process-wide runtime singleton, and AtomRegistry.InitAtom is init-if-absent. A
// server render never unmounts, so request 1's value stayed in the registry and
// request 2's UseAtom initial value was ignored — request 2's HTML contained
// request 1's data. On a real server that is one visitor's state rendered into
// another visitor's page, so these tests are a privacy regression gate, not a
// style preference.
//
// THE CONTRACT: each SSR render is a fresh world. UseAtom's initial value wins on
// every render; a write during a render is visible to the rest of THAT render and
// to nothing else; explicitly seeded request state beats the component's initial.
// In the browser the opposite is true and must stay true — atoms are global to
// the page and outlive every render — which the reconciler test at the bottom
// guards. See internal/runtime/ssr_atom_scope.go.

// atomProbe renders one atom's value, taking its initial value per render, which
// is exactly the shape a request-scoped value has (initial derived from the
// request).
func atomProbe(parseID string, parseInitial string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseAtom := state.UseAtom(parseID, parseInitial)
		return html.Div(html.Props{}, html.Text(parseAtom.Get()))
	})
}

func TestSSRAtomInitialValueWinsOnEveryRender(parseT *testing.T) {
	parseRender := func(parseInitial string) string {
		parseMarkup, parseErr := ui.RenderToString(atomProbe("ssr-scope:tenant", parseInitial))
		if parseErr != nil {
			parseT.Fatalf("RenderToString(%q): %v", parseInitial, parseErr)
		}
		return parseMarkup
	}

	if parseGot := parseRender("ACME-CORP"); parseGot != "<div>ACME-CORP</div>" {
		parseT.Fatalf("render 1 = %q, want <div>ACME-CORP</div>", parseGot)
	}
	// The regression: this used to render ACME-CORP, because render 1 seeded the
	// process-global registry and init-if-absent ignored the new initial value.
	if parseGot := parseRender("HOOLI-INC"); parseGot != "<div>HOOLI-INC</div>" {
		parseT.Errorf("render 2 = %q, want <div>HOOLI-INC</div> — atom state leaked across server renders", parseGot)
	}
	// A third render proves it is per-render scoping and not a one-shot reset.
	if parseGot := parseRender("INITECH"); parseGot != "<div>INITECH</div>" {
		parseT.Errorf("render 3 = %q, want <div>INITECH</div>", parseGot)
	}
}

func TestSSRAtomWriteStaysInsideItsOwnRender(parseT *testing.T) {
	// Writer publishes into the atom; Reader is rendered AFTER it in the same
	// tree, so it must observe the write. A second render of the same tree must
	// not: it starts from its own initial value.
	parseView := func(parseInitial string, parseWrite string) ui.Node {
		parseWriter := func() ui.Node {
			parseAtom := state.UseAtom("ssr-scope:cart", parseInitial)
			if parseWrite != "" {
				parseAtom.Set(parseWrite)
			}
			return html.Span(html.Props{}, html.Text("w"))
		}
		parseReader := func() ui.Node {
			parseAtom := state.UseAtom("ssr-scope:cart", parseInitial)
			return html.Em(html.Props{}, html.Text(parseAtom.Get()))
		}
		return html.Div(html.Props{},
			ui.CreateElement(parseWriter),
			ui.CreateElement(parseReader),
		)
	}

	parseFirst, parseErr := ui.RenderToString(parseView("empty", "3-items"))
	if parseErr != nil {
		parseT.Fatalf("render 1: %v", parseErr)
	}
	if !strings.Contains(parseFirst, "<em>3-items</em>") {
		parseT.Fatalf("render 1 = %q; a write during a render must be visible to later components in the SAME render", parseFirst)
	}

	parseSecond, parseErr2 := ui.RenderToString(parseView("empty", ""))
	if parseErr2 != nil {
		parseT.Fatalf("render 2: %v", parseErr2)
	}
	if strings.Contains(parseSecond, "3-items") {
		parseT.Errorf("render 2 = %q; render 1's Set leaked into render 2", parseSecond)
	}
	if !strings.Contains(parseSecond, "<em>empty</em>") {
		parseT.Errorf("render 2 = %q, want the reader to observe its own initial value", parseSecond)
	}
}

func TestSSRAtomSeedFromRequestBeatsComponentInitial(parseT *testing.T) {
	// The other half of the contract: request state is an INPUT to the render.
	parseMarkup, parseErr := ui.RenderToStringWithAtoms(
		atomProbe("ssr-scope:locale", "en-US"),
		map[string]any{"ssr-scope:locale": "ja-JP"},
	)
	if parseErr != nil {
		parseT.Fatalf("RenderToStringWithAtoms: %v", parseErr)
	}
	if parseMarkup != "<div>ja-JP</div>" {
		parseT.Errorf("seeded render = %q, want <div>ja-JP</div> (seed must beat the component initial)", parseMarkup)
	}

	// And the seed belongs to that request only.
	parsePlain, parseErr2 := ui.RenderToString(atomProbe("ssr-scope:locale", "en-US"))
	if parseErr2 != nil {
		parseT.Fatalf("RenderToString after seeded render: %v", parseErr2)
	}
	if parsePlain != "<div>en-US</div>" {
		parseT.Errorf("unseeded render = %q, want <div>en-US</div> — a previous request's seed leaked", parsePlain)
	}
}

func TestSSRStreamAtomScopeIsPerRequest(parseT *testing.T) {
	// The streaming serializer is a second SSR entry point with its own walk; it
	// needs its own scope or the leak simply moves house.
	parseStream := func(parseInitial string, parseOptions ...ui.SSRStreamOptions) string {
		var parseBuffer bytes.Buffer
		if parseErr := ui.RenderToStream(context.Background(), &parseBuffer, atomProbe("ssr-scope:stream-user", parseInitial), parseOptions...); parseErr != nil {
			parseT.Fatalf("RenderToStream(%q): %v", parseInitial, parseErr)
		}
		return parseBuffer.String()
	}

	if parseGot := parseStream("user-a"); !strings.Contains(parseGot, "user-a") {
		parseT.Fatalf("stream 1 = %q, want user-a", parseGot)
	}
	if parseGot := parseStream("user-b"); strings.Contains(parseGot, "user-a") || !strings.Contains(parseGot, "user-b") {
		parseT.Errorf("stream 2 = %q, want user-b only — atom state leaked between streamed requests", parseGot)
	}
	if parseGot := parseStream("user-c", ui.SSRStreamOptions{InitialAtoms: map[string]any{"ssr-scope:stream-user": "seeded-user"}}); !strings.Contains(parseGot, "seeded-user") {
		parseT.Errorf("seeded stream = %q, want the seeded value to win", parseGot)
	}
}

// TestAtomsPersistAcrossReconcilerRenders is the counterweight: the fix must NOT
// leak into the browser semantic. An atom is process-global and outlives renders
// there on purpose — that is what makes it an atom rather than component state.
// The reconciler path (the one a browser and a native mockdom render both take)
// must therefore still keep the FIRST value when a later render passes a
// different initial. If this test ever fails, the SSR scope has escaped into the
// live render path.
func TestAtomsPersistAcrossReconcilerRenders(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRt := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter})
	parseRoot := parseAdapter.CreateElement("div")

	parseProbe := func(parseInitial string) *runtime.Element {
		return runtime.CreateElement(func() *runtime.Element {
			parseAtom := state.UseAtom("reconciler-scope:session", parseInitial)
			return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseAtom.Get()))
		}, map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, parseProbe("live-value")); parseErr != nil {
		parseT.Fatalf("first RenderInto: %v", parseErr)
	}
	if parseErr := parseRt.RenderInto(parseRoot, parseProbe("ignored-second-initial")); parseErr != nil {
		parseT.Fatalf("second RenderInto: %v", parseErr)
	}

	parseSpan, _ := parseAdapter.GetChildren(parseRoot)[0].(*mockdom.MockDOMNode)
	if parseSpan == nil {
		parseT.Fatal("reconciler render produced no span")
	}
	if parseSpan.TextContent != "live-value" {
		parseT.Errorf("reconciler render = %q, want live-value — atoms must stay shared and persistent outside SSR", parseSpan.TextContent)
	}
}
