// Package hooks is the public, supported facade over the internal testkit/hooks harness for
// testing components' hook behavior. Import this package (not testkit/hooks directly): it
// re-exports the stable harness types and helpers so the implementation can evolve without
// breaking test code.
package hooks

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/v5/testkit/hooks"
)

type Harness[T any] = base.Harness[T]

func RenderHook[T any](parseTb stdtesting.TB, parseHook func() T) *Harness[T] {
	return base.RenderHook(parseTb, parseHook)
}
