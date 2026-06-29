//go:build !js || !wasm

package browser

import (
	"testing"
)

type Options struct {
	Path           string
	HashRouting    bool
	LocalStorage   map[string]string
	SessionStorage map[string]string
	MediaMatches   map[string]bool
}

type Environment struct{}
type OpenCall struct {
	URL      string
	Name     string
	Features string
}
type BroadcastMessage struct {
	Channel string
	Data    any
}
type MockWorker struct{}
type MockWindow struct{}

var nativeBrowserInstallFatal = func(parseTb testing.TB) {
	parseTb.Helper()
	parseTb.Fatal("test/browser Install requires js/wasm tests")
}

func Install(parseTb testing.TB, parseOptions ...Options) *Environment {
	parseTb.Helper()
	nativeBrowserInstallFatal(parseTb)
	return nil
}
