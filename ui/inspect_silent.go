//go:build gwcsilent

package ui

// Building with `-tags gwcsilent` silences UseInspect entirely — the production no-op the
// audit asked for, so the dev `$inspect` calls cost nothing in a release build without
// editing call sites or wiring SetInspectSink by hand.
func init() {
	SetInspectSink(func(string) {})
}
