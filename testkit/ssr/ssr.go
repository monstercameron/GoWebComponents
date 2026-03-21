package ssr

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// Snapshot captures one server-rendered HTML result.
type Snapshot struct {
	HTML string
}

// Render snapshots one UI tree through the public SSR surface.
func Render(tb testing.TB, root ui.Node) Snapshot {
	tb.Helper()
	markup, err := ui.RenderToString(root)
	if err != nil {
		tb.Fatalf("ssr.Render failed: %v", err)
	}
	return Snapshot{HTML: markup}
}

// Contains reports whether the rendered HTML contains the expected substring.
func (s Snapshot) Contains(substring string) bool {
	return strings.Contains(s.HTML, substring)
}

// RequirePayload reads one typed bootstrap payload entry and fails the test if it is missing.
func RequirePayload[T any](tb testing.TB, bootstrap ui.SSRBootstrap, key string) ui.SSRPayloadValue[T] {
	tb.Helper()
	value, ok, err := ui.ReadBootstrapPayload[T](bootstrap, key)
	if err != nil {
		tb.Fatalf("ssr.RequirePayload failed for key %q: %v", key, err)
	}
	if !ok {
		tb.Fatalf("ssr.RequirePayload could not find key %q", key)
	}
	return value
}
