//go:build !js || !wasm
// +build !js !wasm

package routertest

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/router"
)

type Fixture = base.Fixture
type Inspection = base.Inspection

func NewHash(parseTb stdtesting.TB, parseOptions ...interface{}) *Fixture {
	return base.NewHash(parseTb, parseOptions...)
}

func NewHistory(parseTb stdtesting.TB, parseOptions ...interface{}) *Fixture {
	return base.NewHistory(parseTb, parseOptions...)
}
