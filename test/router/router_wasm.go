//go:build js && wasm
// +build js,wasm

package routertest

import (
	stdtesting "testing"

	appRouter "github.com/monstercameron/GoWebComponents/router"
	base "github.com/monstercameron/GoWebComponents/testkit/router"
)

type Fixture = base.Fixture

func NewHash(tb stdtesting.TB, options ...appRouter.RouterOptions) *Fixture {
	return base.NewHash(tb, options...)
}

func NewHistory(tb stdtesting.TB, options ...appRouter.RouterOptions) *Fixture {
	return base.NewHistory(tb, options...)
}
