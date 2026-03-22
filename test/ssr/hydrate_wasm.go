//go:build js && wasm
// +build js,wasm

package ssr

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

type HydrationOptions = base.HydrationOptions
type HydrationHarness = base.HydrationHarness

func SmokeHydrate(tb stdtesting.TB, root ui.Node, options ...HydrationOptions) *HydrationHarness {
	return base.SmokeHydrate(tb, root, options...)
}
