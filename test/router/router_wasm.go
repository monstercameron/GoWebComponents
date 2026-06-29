//go:build js && wasm

package routertest

import (
	stdtesting "testing"

	appRouter "github.com/monstercameron/GoWebComponents/v4/router"
	base "github.com/monstercameron/GoWebComponents/v4/testkit/router"
)

type Fixture = base.Fixture

func NewHash(parseTb stdtesting.TB, parseOptions ...appRouter.RouterOptions) *Fixture {
	return base.NewHash(parseTb, parseOptions...)
}

func NewHistory(parseTb stdtesting.TB, parseOptions ...appRouter.RouterOptions) *Fixture {
	return base.NewHistory(parseTb, parseOptions...)
}
