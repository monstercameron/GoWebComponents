# Documentation Push TODO

This page is the public documentation mirror for the completed documentation
push.

## Current Status

The former checklist is complete and stale as an active backlog. The repo now
has a consolidated reference manual, an API browser chapter, public examples
site cross-links, and doclint guards that keep common documentation drift from
returning.

This file intentionally remains a status snapshot rather than a second editable
documentation backlog.

## Evidence

- `docs/REFERENCE_MANUAL/README.md` is the primary documentation entrypoint for
  application authors.
- `docs/REFERENCE_MANUAL/16-api-browser.md` maps the current public package
  surface across core, companion, support, and test packages.
- `examples/public-examples-site/assets/docs/reference-map.md` cross-links
  concepts to public APIs, examples, production caveats, and related docs.
- `docs/doclint` scans Markdown command blocks for stale repo-relative paths.
- `todos.md` contains no unchecked documentation-push backlog items.

## Refresh Rule

Do not add new documentation backlog checkboxes here. Track future
documentation work in the canonical repo backlog or in the owning source
document, then refresh this public mirror when the canonical status changes.
