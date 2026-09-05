# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v5.0.5 - 2026-09-05

Version 5.0.5 reduces production-wasm renderer overhead across mounts, updates,
events, and hook-heavy trees. It adds compact host/event props, batched DOM
mutations, lazy serialized-tree binding, commit pruning, typed hook/component
storage, and bounded recycling for plain host fibers.

Against the vendored React 19.2.4 production harness, the 19-scenario same-run
geometric mean improved from 2.614x React at the untouched v5.0.4 baseline to a
best observed 1.211x; the release-policy run was 1.465x. The full suite has not
yet crossed below React, and the rejected GC/deadline experiments are recorded
so later work does not repeat them.

See the repository root `CHANGELOG.md` for the full entry, methodology, and
measurements.
