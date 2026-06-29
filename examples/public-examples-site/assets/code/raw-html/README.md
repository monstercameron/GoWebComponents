# Raw HTML Nodes

Renders rich markup and a trusted inline SVG by **parsing them into real DOM nodes** — never via
`innerHTML`. The framework keeps a single, safe path to the DOM.

## What it shows
- `html.RawHTML(markup)` — parses an HTML fragment, **sanitizes** it (the `<b>` survives, an embedded
  `<script>` is stripped), and returns real element nodes.
- `html.RawHTMLUnsafe(markup)` — for trusted content (an inline `<svg>`), parsed into real nodes in
  the correct namespace, still without touching `innerHTML`.
- Mixing parsed nodes into a normal component tree alongside hand-authored elements.

## Run
```
gwc dev examples/public/raw-html
```
Inspect the DOM: the sanitized fragment has no `<script>`, and the SVG is real, namespaced nodes.
