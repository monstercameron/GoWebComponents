package ssr

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Snapshot = base.Snapshot
type StaticExport = base.StaticExport
type ExportedRoute = base.ExportedRoute

func Render(parseTb stdtesting.TB, parseRoot ui.Node) Snapshot {
	return base.Render(parseTb, parseRoot)
}

func RequirePayload[T any](parseTb stdtesting.TB, parseBootstrap ui.SSRBootstrap, parseKey string) ui.SSRPayloadValue[T] {
	return base.RequirePayload[T](parseTb, parseBootstrap, parseKey)
}

func LoadStaticExport(parseTb stdtesting.TB, parseOutputDir string) StaticExport {
	return base.LoadStaticExport(parseTb, parseOutputDir)
}
