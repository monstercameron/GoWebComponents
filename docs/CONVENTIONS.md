# Code Conventions

These conventions are applied consistently across the framework. New code is
expected to match them so the codebase reads as one voice.

## Local variable naming (idiomatic Go)

Use clear, idiomatic Go names for local variables, parameters, and receivers
(`key`, `value`, `err`, `result`, `idx`, `r`, `s`), following standard Go
conventions (short names in tight scopes, descriptive names in wider scopes).

```go
func ScopeCacheKey(segments ...string) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment != "" {
			parts = append(parts, segment)
		}
	}
	return strings.Join(parts, ":")
}
```

> **Historical note (`parse`-prefixed locals):** Older parts of the codebase
> prefixed local variables with `parse` (`parseKey`, `parseErr`, `parseRt`).
> That convention is **retired for new and edited code**. Do not add new
> `parse`-prefixed locals, and do not perform mechanical repo-wide renames of
> untouched legacy files solely to strip the prefix.

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

- Functions start with a verb where one applies.
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

## JSON wire compatibility

Treat JSON tags on exported payloads, persisted records, generated metadata,
and launcher config as wire contracts. Do not convert `omitempty` to `omitzero`
or omit previously emitted zero-valued struct/time fields unless the payload
contract explicitly changes and has a regression test for the new shape.

## Deprecations

When deprecating a public API, keep it working and emit a one-time structured
deprecation diagnostic that names the replacement, rather than removing or
silently aliasing it. `gwc lint` flags exported deprecated functions and
methods that do not call `deprecation.Warn(...)`. Legacy `gwc` flag aliases
follow this pattern.
