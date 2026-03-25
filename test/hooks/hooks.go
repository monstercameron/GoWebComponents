package hooks

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/hooks"
)

type Harness[T any] = base.Harness[T]

func RenderHook[T any](parseTb stdtesting.TB, parseHook func() T) *Harness[T] {
	return base.RenderHook(parseTb, parseHook)
}
