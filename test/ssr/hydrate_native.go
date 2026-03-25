//go:build !js || !wasm
// +build !js !wasm

package ssr

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
)

type HydrationOptions = base.HydrationOptions
type HydrationHarness = base.HydrationHarness

func SmokeHydrate(tb stdtesting.TB, root interface{}, options ...HydrationOptions) *HydrationHarness {
	return base.SmokeHydrate(tb, root, options...)
}

func RoundTripHydrate(tb stdtesting.TB, root interface{}, options ...HydrationOptions) *HydrationHarness {
	return base.RoundTripHydrate(tb, root, options...)
}

func RoundTripHydrateMismatch(tb stdtesting.TB, root interface{}, buildMutate func(string) string, options ...HydrationOptions) *HydrationHarness {
	return base.RoundTripHydrateMismatch(tb, root, buildMutate, options...)
}
