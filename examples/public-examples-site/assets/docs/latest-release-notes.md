# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v5.0.2 - 2026-07-28

Version 5.0.2 changes examples, documentation, measurement, and CI without
changing framework behavior. Atlas Commerce OS is now the reference v5
application, the performance harness uses real trusted browser input, and the
production-readiness measurements were repeated on a quiet machine.

The corrected measurements showed that long frames came from coalesced update
batches rather than slow rendering. The release also adds CI coverage for Atlas
boot, server-function code generation, and catalog module-path resolution.

See the repository root `CHANGELOG.md` for the full entry, methodology, and
measurements.
