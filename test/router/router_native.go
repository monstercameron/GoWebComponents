//go:build !js || !wasm
// +build !js !wasm

package routertest

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/router"
)

type Fixture = base.Fixture
type Inspection = base.Inspection

func NewHash(tb stdtesting.TB, options ...interface{}) *Fixture {
	return base.NewHash(tb, options...)
}

func NewHistory(tb stdtesting.TB, options ...interface{}) *Fixture {
	return base.NewHistory(tb, options...)
}
