# GWC v5 — the size story, honestly

Plan item **P6.5**. Its acceptance criterion asks that this document state the Go
wasm floor, the current numbers, and that React-bundle parity is not a goal —
and that it be readable as honest rather than defensive.

So the numbers come first, before any explanation of them.

Measured 2026-07-25 on the two-artifact example
(`examples/v5-two-artifact`), `GOOS=js GOARCH=wasm`, `-trimpath -ldflags="-s -w"`,
maximum compression. Reproduce with `go test ./examples/v5-two-artifact/ -v`.

| Artifact | Raw | gzip | brotli |
|---|---:|---:|---:|
| `app.wasm` — render thread | 6.33 MB | 1.73 MB | **1.26 MB** |
| `services.wasm` — domain worker | 9.41 MB | 2.64 MB | 1.96 MB |
| empty Go program (the floor) | 1.80 MB | 0.54 MB | — |

## The floor

An empty Go js/wasm program — literally `func main() { select {} }` — is
**541 KB gzipped**. That is the Go runtime, its scheduler, and its garbage
collector. No framework work removes it, and no application work removes it.

It is **34% of v5's own 1.6 MB budget before a single line of framework code.**

This is the single most important number in this document, and it is the one
most often left out of comparisons. A framework size quoted without its runtime
floor makes the addressable portion look far larger than it is.

## M5, stated plainly

M5 targets `app.wasm` **gzipped** under 1.6 MB, from a 2.37 MB v4 baseline.

- **Gzipped: 1.73 MB. M5 is missed by 8%.**
- Of the 1.6 MB budget, 541 KB is the floor, so the addressable budget is
  1.06 MB and actual framework + app code is 1.19 MB. The overrun is **12% of
  the part anyone can do anything about.**
- Splitting into two artifacts moved it 2.37 → 1.73 MB, a **27% reduction**, by
  keeping the SQLite engine and its wasm interpreter out of the render thread's
  binary. Packaging cannot close the remaining gap; that needs framework-side
  size work.

**Brotli changes the practical answer.** The same binary is **1.26 MB brotli**,
which is 21% *under* the 1.6 MB target. Every browser that can run wasm
negotiates brotli, so 1.26 MB is what a user actually downloads.

Both numbers are true and they are not the same claim:

- Against the metric **as written** — gzip — v5 misses M5.
- Against **what ships**, v5 is comfortably inside it.

The metric is not being retroactively edited to produce a pass. M5 is recorded
as missed, and this paragraph exists so nobody quotes 1.26 MB without saying
which compression produced it.

## What has not been measured

Stated because a gap is more useful than an estimate.

- **`wasm-opt -Oz` is unmeasured.** Binaryen is not installed in the environment
  these numbers came from. Its delta is not estimated. P6.1's brotli half is
  measured exactly (27% under gzip for `app.wasm`, 26% for `services.wasm`); its
  wasm-opt half is simply not done.
- **Route-level code splitting (P6.2) is not implemented,** so every number here
  is a whole-application download.
- **Time-to-first-command (M10) is unmeasured.** It is a browser measurement —
  instantiation, compilation, and the first round trip — and `services.wasm`'s
  size is only its input. The size is recorded; the timing is not.

## React-bundle parity is not a goal

This is the defensive-sounding part, so it is worth being precise about what is
and is not being claimed.

A React application ships JavaScript to a runtime the browser already has. A Go
application ships its runtime. Those are different products, and 541 KB of the
difference is decided before either team writes code. A framework that quoted
itself against React without saying so would be comparing a total against a
delta.

What that does **not** mean:

- It is not a reason to stop measuring. Every number above is measured, in CI,
  with a ratchet that fails if `app.wasm` grows.
- It is not a reason to dismiss size as a concern. Size is cold start, cold start
  is the first thing a user experiences, and 1.26 MB is not free.
- It is not a claim that Go wasm size cannot improve. The addressable 1.19 MB is
  addressable.

What it does mean: **"smaller than a React bundle" is not a target v5 pursues,**
because hitting it would require abandoning the Go runtime rather than improving
the framework. v5's size goals are stated against its own floor and its own
previous version, which are the comparisons it can actually act on.

## Where the size goes, and why the split was worth it

`services.wasm` is 1.96 MB brotli — larger than `app.wasm` — because it carries
`ncruces/go-sqlite3` and the `wazero` interpreter that runs it.

That is the whole argument for the two-artifact split. Before it, every one of
those bytes was in the binary the render thread had to download and instantiate
before the first paint. Now they load in a worker, in parallel, off the path to
first render.

The split is verified on the **dependency graph** rather than by size, because a
binary can be small for reasons unrelated to the engine and would stop being
small the moment someone added a convenience import.
`TestAppArtifactDoesNotLinkTheEngine` fails the build if `app.wasm` ever links
wazero, ncruces, modernc, or `db/sqlite`.

## Reproducing

```
go test ./examples/v5-two-artifact/ -v      # all sizes, both compressions
go test ./examples/v5-two-artifact/ -run TestM5      # M5 and its breakdown
go test ./examples/v5-two-artifact/ -run TestP61     # gzip vs brotli
```

The Go floor is measured the same way, against an empty `main` with the same
flags and toolchain — not quoted from documentation.
