# Code Conventions

These conventions are applied consistently across the framework. New code is
expected to match them so the codebase reads as one voice.

## `parse`-prefixed local variables

**Every local variable is prefixed `parse`**: `parseKey`, `parseValue`,
`parseErr`, `parseResult`, `parseIndex`. This includes loop variables, receiver
helpers, and intermediate results.

```go
func ScopeCacheKey(parseSegments ...string) string {
	parseParts := make([]string, 0, len(parseSegments))
	for _, parseSegment := range parseSegments {
		parseSegment = strings.TrimSpace(parseSegment)
		if parseSegment != "" {
			parseParts = append(parseParts, parseSegment)
		}
	}
	return strings.Join(parseParts, ":")
}
```

Why a single uniform prefix:

- It makes locals instantly distinguishable from package-level identifiers,
  exported API, and struct fields at a glance, with no per-name judgement.
- It keeps generated, hand-written, and refactored code visually identical,
  which matters in a codebase where the same logic often exists in a
  `*_native.go` / `*_wasm.go` pair.
- It removes a whole class of naming bikeshedding from review.

The prefix applies to **locals only**. Exported and package-level identifiers,
struct fields, method receivers' field names, and constants keep ordinary Go
names.

## Function naming: `verbSubject[Object]`

Functions start with a **verb**, then the owning **subject** (domain, not data
type), and an optional **object** only when needed.

```
renderSidebar
renderSidebarConvRow
getModelLabel
storeConvList
handleUserDeleteConv
buildToolbarSelectOption
```

Rules:

- All functions and variables start with a verb where one applies.
- The subject names the domain (`sidebar`, `thread`, `cache`), not the data
  type (`string`, `slice`).
- Add an object only when it disambiguates.
- Booleans read as predicates: `isStreaming`, `hasExactCost`, `canRetry`,
  `shouldEscalate`.

Common verbs: `get set store cache clear render build handle filter format
parse apply reset`.

## Documentation comments

- Every exported function, type, const block, and var needs a GoDoc comment.
- The **first word of the comment is the symbol name** (`// ScopeCacheKey
  builds ...`), per `go vet`/`golint` expectations.
- Comment **intent**, not mechanics. Explain *why* a non-obvious block exists,
  not what each line does.

## Platform split files

Browser-only code lives in `*_wasm.go` (`//go:build js && wasm`); the
non-browser counterpart lives in `*_native.go` with a matching inverse build
tag and provides a stub (typically returning an `unavailable(...)` error). Keep
the **exported surface identical** across the pair so callers compile on either
target. Pure-Go logic that needs no platform access goes in an untagged file so
it is unit-testable natively.

## Errors and panics

- Do not swallow errors to make a test pass, and do not hide panics. Browser
  code recovers at goroutine and `js.FuncOf` boundaries and emits a structured
  diagnostic (crash containment) rather than killing the page.
- Prefer the smallest root-cause fix; preserve useful diagnostics.

## Deprecations

When deprecating a public API, keep it working and emit a one-time structured
deprecation diagnostic that names the replacement, rather than removing or
silently aliasing it. Legacy `gwc` flag aliases follow this pattern.
