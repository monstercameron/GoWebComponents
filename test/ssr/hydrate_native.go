//go:build !js || !wasm

package ssr

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
)

type HydrationOptions = base.HydrationOptions
type HydrationHarness = base.HydrationHarness

var (
	smokeHydrateBase             = base.SmokeHydrate
	roundTripHydrateBase         = base.RoundTripHydrate
	roundTripHydrateMismatchBase = base.RoundTripHydrateMismatch
)

func SmokeHydrate(parseTb stdtesting.TB, parseRoot any, parseOptions ...HydrationOptions) *HydrationHarness {
	return smokeHydrateBase(parseTb, parseRoot, parseOptions...)
}

func RoundTripHydrate(parseTb stdtesting.TB, parseRoot any, parseOptions ...HydrationOptions) *HydrationHarness {
	return roundTripHydrateBase(parseTb, parseRoot, parseOptions...)
}

func RoundTripHydrateMismatch(parseTb stdtesting.TB, parseRoot any, buildMutate func(string) string, parseOptions ...HydrationOptions) *HydrationHarness {
	return roundTripHydrateMismatchBase(parseTb, parseRoot, buildMutate, parseOptions...)
}
