# Failure, Recovery, And Edge-Case Stories

Executable tests should cover these edge-case groups before a flow is considered release-ready.

| Story | Coverage |
| --- | --- |
| `recovery-not-found` | Missing public product, missing warehouse, missing warehouse availability pairing, missing internal record, and unknown route entry. |
| `recovery-server-error` | Public and internal server-error recovery pages with route-appropriate retry or return actions. |
| `recovery-failed-write` | Failed writes preserve user-entered form data and show field-level plus summary-level errors. |
| `recovery-network-interruption` | Async public secondary panels and internal side panels recover after interrupted network work without corrupting visible state. |
| `recovery-duplicate-submit-timeout` | Duplicate-submit prevention and retry behavior after timed-out write flows. |
| `recovery-invalid-input-state` | Malformed query params, unsupported sort keys, unsupported density values, invalid locale values, and corrupted browser-storage snapshots. |
| `recovery-async-navigation-race` | Route refresh during pending mutation, overlay open during revalidation, and navigation away during async follow-up work. |

The manifest validates that every recovery story has at least one route target and concrete assertions.
