//go:build js && wasm

package ssr

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/testkit/render"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// HydrationOptions configures the hydration smoke harness.
type HydrationOptions struct {
	Bootstrap ui.SSRBootstrap
	Strict    bool
	Markup    string
}

// HydrationHarness wraps one hydration smoke run.
type HydrationHarness struct {
	tb        testing.TB
	fixture   *render.Fixture
	Bootstrap ui.SSRBootstrap
	Markup    string
	Seeded    render.SeededMarkup
	cleaned   bool
}

// SmokeHydrate runs a lightweight hydration smoke test against the mock DOM fixture.
func SmokeHydrate(parseTb testing.TB, parseRoot ui.Node, parseOptions ...HydrationOptions) *HydrationHarness {
	parseTb.Helper()
	parseResolved := HydrationOptions{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	parseFixture := render.New(parseTb)
	parseRt := runtime.GetGlobalRuntime()
	if parseResolved.Bootstrap.IDSeed > 0 {
		parseRt.SetIDSeed(parseResolved.Bootstrap.IDSeed)
	}
	if len(parseResolved.Bootstrap.Atoms) > 0 {
		if parseErr := parseRt.RestoreAtomSnapshot(parseResolved.Bootstrap.Atoms); parseErr != nil {
			parseTb.Fatalf("ssr.SmokeHydrate failed to restore bootstrap atoms: %v", parseErr)
		}
	}
	parseRt.SetNextHydrationStrict(parseResolved.Strict)
	if parseErr2 := parseRt.HydrateInto(parseFixture.Target(), parseRoot); parseErr2 != nil {
		parseTb.Fatalf("ssr.SmokeHydrate failed: %v", parseErr2)
	}
	parseFixture.Stabilize()
	parseHarness := &HydrationHarness{
		tb:        parseTb,
		fixture:   parseFixture,
		Bootstrap: parseResolved.Bootstrap,
	}
	parseTb.Cleanup(func() {
		parseHarness.Cleanup()
	})
	return parseHarness
}

// RoundTripHydrate seeds real server markup first, then hydrates the same UI tree into it.
func RoundTripHydrate(parseTb testing.TB, parseRoot ui.Node, parseOptions ...HydrationOptions) *HydrationHarness {
	parseTb.Helper()
	parseResolved := HydrationOptions{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	parseMarkup := strings.TrimSpace(parseResolved.Markup)
	if parseMarkup == "" {
		parseRendered, parseErr := ui.RenderToString(parseRoot)
		if parseErr != nil {
			parseTb.Fatalf("ssr.RoundTripHydrate failed to render server markup: %v", parseErr)
		}
		parseMarkup = parseRendered
	}
	parseFixture := render.New(parseTb)
	parseSeeded := parseFixture.SeedHTML(parseMarkup)
	parseRt := runtime.GetGlobalRuntime()
	if parseResolved.Bootstrap.IDSeed > 0 {
		parseRt.SetIDSeed(parseResolved.Bootstrap.IDSeed)
	}
	if len(parseResolved.Bootstrap.Atoms) > 0 {
		if parseErr2 := parseRt.RestoreAtomSnapshot(parseResolved.Bootstrap.Atoms); parseErr2 != nil {
			parseTb.Fatalf("ssr.RoundTripHydrate failed to restore bootstrap atoms: %v", parseErr2)
		}
	}
	parseRt.SetNextHydrationStrict(parseResolved.Strict)
	if parseErr3 := parseRt.HydrateInto(parseFixture.Target(), parseRoot); parseErr3 != nil {
		parseTb.Fatalf("ssr.RoundTripHydrate failed: %v", parseErr3)
	}
	parseFixture.Stabilize()
	parseHarness := &HydrationHarness{
		tb:        parseTb,
		fixture:   parseFixture,
		Bootstrap: parseResolved.Bootstrap,
		Markup:    parseMarkup,
		Seeded:    parseSeeded,
	}
	parseTb.Cleanup(func() {
		parseHarness.Cleanup()
	})
	return parseHarness
}

// RoundTripHydrateMismatch seeds mutated server markup before hydration to force mismatch paths.
func RoundTripHydrateMismatch(parseTb testing.TB, parseRoot ui.Node, buildMutate func(string) string, parseOptions ...HydrationOptions) *HydrationHarness {
	parseTb.Helper()
	parseMarkup, parseErr := ui.RenderToString(parseRoot)
	if parseErr != nil {
		parseTb.Fatalf("ssr.RoundTripHydrateMismatch failed to render server markup: %v", parseErr)
	}
	parseMutate := buildMutate
	if parseMutate == nil {
		parseMutate = buildHydrationMismatchMarkup
	}
	parseMutated := strings.TrimSpace(parseMutate(parseMarkup))
	if parseMutated == "" {
		parseMutated = buildHydrationMismatchMarkup(parseMarkup)
	}
	parseResolved := HydrationOptions{Markup: parseMutated}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
		parseResolved.Markup = parseMutated
	}
	return RoundTripHydrate(parseTb, parseRoot, parseResolved)
}

func (parseH *HydrationHarness) ByID(parseId string) *render.QueryNode {
	if parseH == nil || parseH.fixture == nil {
		return nil
	}
	return parseH.fixture.ByID(parseId)
}

func (parseH *HydrationHarness) ByText(parseText string) *render.QueryNode {
	if parseH == nil || parseH.fixture == nil {
		return nil
	}
	return parseH.fixture.ByText(parseText)
}

func (parseH *HydrationHarness) Text() string {
	if parseH == nil || parseH.fixture == nil {
		return ""
	}
	return parseH.fixture.Text()
}

func (parseH *HydrationHarness) Cleanup() {
	if parseH == nil || parseH.cleaned {
		return
	}
	parseH.cleaned = true
	if parseH.fixture != nil {
		parseH.fixture.Cleanup()
		parseH.fixture = nil
	}
}

// buildHydrationMismatchMarkup mutates one rendered markup string for mismatch testing.
func buildHydrationMismatchMarkup(parseMarkup string) string {
	parseMarkup = strings.TrimSpace(parseMarkup)
	if parseMarkup == "" {
		return `<div data-gwc-hydration-mismatch="server">server-mismatch</div>`
	}
	if strings.Contains(parseMarkup, ">") {
		return strings.Replace(parseMarkup, ">", ` data-gwc-hydration-mismatch="server">`, 1)
	}
	return parseMarkup + `<!--gwc-hydration-mismatch-->`
}
