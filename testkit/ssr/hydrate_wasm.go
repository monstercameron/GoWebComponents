//go:build js && wasm
// +build js,wasm

package ssr

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/testkit/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

// HydrationOptions configures the hydration smoke harness.
type HydrationOptions struct {
	Bootstrap ui.SSRBootstrap
	Strict    bool
}

// HydrationHarness wraps one hydration smoke run.
type HydrationHarness struct {
	tb        testing.TB
	fixture   *render.Fixture
	Bootstrap ui.SSRBootstrap
	cleaned   bool
}

// SmokeHydrate runs a lightweight hydration smoke test against the mock DOM fixture.
func SmokeHydrate(tb testing.TB, root ui.Node, options ...HydrationOptions) *HydrationHarness {
	tb.Helper()
	resolved := HydrationOptions{}
	if len(options) > 0 {
		resolved = options[0]
	}
	fixture := render.New(tb)
	rt := runtime.GetGlobalRuntime()
	if resolved.Bootstrap.IDSeed > 0 {
		rt.SetIDSeed(resolved.Bootstrap.IDSeed)
	}
	if len(resolved.Bootstrap.Atoms) > 0 {
		if err := rt.RestoreAtomSnapshot(resolved.Bootstrap.Atoms); err != nil {
			tb.Fatalf("ssr.SmokeHydrate failed to restore bootstrap atoms: %v", err)
		}
	}
	rt.SetNextHydrationStrict(resolved.Strict)
	if err := rt.HydrateInto(fixture.Target(), root); err != nil {
		tb.Fatalf("ssr.SmokeHydrate failed: %v", err)
	}
	fixture.Stabilize()
	harness := &HydrationHarness{
		tb:        tb,
		fixture:   fixture,
		Bootstrap: resolved.Bootstrap,
	}
	tb.Cleanup(func() {
		harness.Cleanup()
	})
	return harness
}

func (h *HydrationHarness) ByID(id string) *render.QueryNode {
	if h == nil || h.fixture == nil {
		return nil
	}
	return h.fixture.ByID(id)
}

func (h *HydrationHarness) ByText(text string) *render.QueryNode {
	if h == nil || h.fixture == nil {
		return nil
	}
	return h.fixture.ByText(text)
}

func (h *HydrationHarness) Text() string {
	if h == nil || h.fixture == nil {
		return ""
	}
	return h.fixture.Text()
}

func (h *HydrationHarness) Cleanup() {
	if h == nil || h.cleaned {
		return
	}
	h.cleaned = true
	if h.fixture != nil {
		h.fixture.Cleanup()
		h.fixture = nil
	}
}
