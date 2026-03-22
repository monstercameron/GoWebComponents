package ssr

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Snapshot = base.Snapshot

func Render(tb stdtesting.TB, root ui.Node) Snapshot {
	return base.Render(tb, root)
}

func RequirePayload[T any](tb stdtesting.TB, bootstrap ui.SSRBootstrap, key string) ui.SSRPayloadValue[T] {
	return base.RequirePayload[T](tb, bootstrap, key)
}
