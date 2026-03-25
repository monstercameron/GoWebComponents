// Package render provides a first public component-fixture harness for js/wasm
// tests built on the framework's public UI surface.
//
// The initial slice focuses on mounting a component tree into a controlled mock
// DOM fixture, rerendering it, and querying rendered output without reaching
// into repo-local runtime test helpers. On js/wasm builds the fixture runtime
// is process-global, so fixture-owning tests must run sequentially.
package render
