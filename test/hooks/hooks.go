package hooks

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/hooks"
)

type Harness[T any] = base.Harness[T]

func RenderHook[T any](tb stdtesting.TB, hook func() T) *Harness[T] {
	return base.RenderHook(tb, hook)
}
