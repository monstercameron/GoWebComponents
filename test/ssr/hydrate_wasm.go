//go:build js && wasm

package ssr

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/v6/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type HydrationOptions = base.HydrationOptions
type HydrationHarness = base.HydrationHarness

func SmokeHydrate(parseTb stdtesting.TB, parseRoot ui.Node, parseOptions ...HydrationOptions) *HydrationHarness {
	return base.SmokeHydrate(parseTb, parseRoot, parseOptions...)
}

func RoundTripHydrate(parseTb stdtesting.TB, parseRoot ui.Node, parseOptions ...HydrationOptions) *HydrationHarness {
	return base.RoundTripHydrate(parseTb, parseRoot, parseOptions...)
}

func RoundTripHydrateMismatch(parseTb stdtesting.TB, parseRoot ui.Node, buildMutate func(string) string, parseOptions ...HydrationOptions) *HydrationHarness {
	return base.RoundTripHydrateMismatch(parseTb, parseRoot, buildMutate, parseOptions...)
}
