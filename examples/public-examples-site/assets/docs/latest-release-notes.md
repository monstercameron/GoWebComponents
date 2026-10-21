# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v7.0.2 — performance patch

Host elements with event-only or special-property props now avoid allocating
an unused compact attribute view. A focused native benchmark fell from 216 B
and 3 allocations per element to 152 B and 2 allocations; string-only props
were unchanged. The public `CreateElement` props-copy contract, exported API,
and Go 1.27 requirement are unchanged.

The [same-runner Wasm suite](https://github.com/monstercameron/GoWebComponents/actions/runs/39613566908)
passed with 148 matched entries and a 0.21% faster geometric mean. The
[Example 201 browser comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39618939912)
passed all 19 scenarios with a 6.36% faster geometric mean. The
[native drift suite](https://github.com/monstercameron/GoWebComponents/actions/runs/39618937798)
passed all per-entry gates across 158 matches; its overall geometric mean was
2.70% slower in that run, including slower unrelated projection and
virtualization entries. CI also balances browser capture order and gives one
asynchronous router Wasm test a longer scheduling deadline.
