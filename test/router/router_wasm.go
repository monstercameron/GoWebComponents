//go:build js && wasm

package routertest

import (
	stdtesting "testing"

	appRouter "github.com/monstercameron/GoWebComponents/v7/router"
	base "github.com/monstercameron/GoWebComponents/v7/testkit/router"
)

type Fixture = base.Fixture

// NewHash creates a hash-router fixture with v7 router options.
func NewHash(parseTb stdtesting.TB, parseOptions ...appRouter.RouterOption) *Fixture {
	return base.NewHash(parseTb, parseOptions...)
}

// NewHistory creates a history-router fixture with v7 router options.
func NewHistory(parseTb stdtesting.TB, parseOptions ...appRouter.RouterOption) *Fixture {
	return base.NewHistory(parseTb, parseOptions...)
}
