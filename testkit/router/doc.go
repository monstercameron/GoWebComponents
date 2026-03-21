// Package routertest provides a first public router-test harness for js/wasm
// tests.
//
// It builds on testkit/render and the public router package so tests can set an
// initial path, navigate, inspect params and query state, and assert rendered
// route output without relying on repo-local router test helpers.
package routertest
