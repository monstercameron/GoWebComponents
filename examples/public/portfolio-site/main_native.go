//go:build !js || !wasm

package main

// main is a no-op on native hosts: this example runs only under js/wasm, and the
// stub lets `go build ./...` and native tests of the shared files succeed.
func main() {}
