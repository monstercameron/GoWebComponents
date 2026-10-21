# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v7.0.1 — performance patch

Host elements whose only prop is a reconciliation `key` no longer reserve an
empty host-attribute view during construction. This reduces work and memory
used by keyed lists while preserving the public `CreateElement` props-copy
contract and rendered output. This patch does not change the public API or the
Go 1.27 requirement.
