# Devtools Example

A minimal example focused on the public `devtools` package.

## Current Status

This is a small standalone diagnostics surface that embeds `devtools.Panel` directly into one page.

Use it when you want to inspect tree shape, hook state, profiling counters, and structured diagnostics without the larger Atlas shell or the narrower single-purpose `66` through `69` devtools examples.

It demonstrates:

- the embeddable in-browser devtools panel
- component tree and hook inspection without the larger OMI shell
- runtime profiling counters
- structured diagnostics, including a deliberate mixed keyed/unkeyed list warning

Serve examples from the repo root with:

```powershell
go run ./tools/gwc examples
```

Then open `/examples/16-devtools/devtools.html`.

