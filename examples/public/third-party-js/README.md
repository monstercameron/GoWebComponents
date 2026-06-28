# Third-party JS integration (typed bridge + native parity)

A worked example of using a third-party **JavaScript** library from Go behind a typed bridge,
with a pure-Go fallback so the same code runs on the server.

It wraps [`@sindresorhus/slugify`](https://github.com/sindresorhus/slugify):

```go
bridge := thirdpartyjs.LoadSlugify(ctx)        // dynamic-imports the ESM module in the browser
slug := bridge.Slugify(ctx, "Hello, World!")   // → "hello-world"
```

## The pattern

1. **Dynamic-import** the module with `interop.ImportModule(ctx, "https://esm.sh/...")`.
2. **Wrap it** in a typed Go struct (`SlugifyBridge`) whose methods `Call`/`CallDefault` into the
   module and type-assert the results — callers see a normal Go API, not `any`.
3. **Fall back in Go.** `interop.ImportModule` is a no-op stub on native/SSR (and a browser may be
   offline), so the bridge keeps a pure-Go equivalent. `LoadSlugify` returns an *unloaded* bridge
   that transparently routes to the fallback — callers never branch on platform.

## Native-stub parity

`slugify_test.go` runs on the **native** lane, where `ImportModule` is unavailable: it asserts the
bridge reports `Loaded() == false` yet still returns the correct slug via the Go fallback. That is
the guarantee — the identical call site produces the identical result on the server and in the
browser, so SSR and tests are never blocked by a browser-only dependency.

```sh
go test ./examples/public/third-party-js
```
