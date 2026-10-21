# Versioning & migration policy

GoWebComponents follows [Semantic Versioning](https://semver.org) for its public Go API,
and — because this is a Go module — the import path encodes the major version. This document
commits the actual rules so a release's number is predictable, not a judgment call.

## What each bump means

| Bump | Trigger | Examples |
|---|---|---|
| **Major** (`/v6`, `/v7`, …) | A breaking change to an **exported** identifier in the library module, or any change that forces a consumer to edit code or that alters a consumer's `go.sum`. | Removing/renaming an exported func/type/method; changing an exported signature; moving a dependency out of the library module (see *Module split* below). |
| **Minor** (`x.Y.0`) | A backward-compatible **addition**. | New exported function/type/method; a new package; a new optional field with a zero-value default. |
| **Patch** (`x.y.Z`) | A backward-compatible **fix** with no API change. | Bug fixes, performance, docs, internal refactors, new tests. |

The v7 development branch and `v7.0.0-rc.1` release candidate use
**`github.com/monstercameron/GoWebComponents/v7`**; the latest published stable
major remains v6 until the stable v7.0.0 tag is published. A v6 consumer must
change its module requirement and imports from `/v6` to `/v7` to upgrade. RC
consumers should pin the exact RC version. The stable `v7.0.0` requirement is
available only after that tag is published.

**v7 requires Go 1.27.** Its `go.mod` declares `go 1.27.0`, with toolchain
`go1.27.1`. Raising the Go floor ships with the v7 major bump.

It previously moved to `/v5` in v5.0.0 because retiring runtime2 as a renderer removed `ui.ParallelRegion`
and the surrounding public API, and changed the `ui.Hydrate`/`ui.HydrateInto`
signatures — exactly the trigger in the Major row above. The change had been sitting
on the branch under `/v4`, where `go get -u` would have delivered it to consumers
with no import-path signal; that is what a major version exists to prevent.

"Exported API" is defined concretely as the surface captured by the **API-baseline tests**
(`internal/apidump` + each package's `api_baseline.txt`): exported types with their shape,
funcs, methods, consts, and vars. That makes the major/minor boundary mechanical, not
subjective:

- A baseline diff that **removes or changes** an existing line ⇒ **major**.
- A baseline diff that **only adds** lines ⇒ **minor**.
- No baseline diff ⇒ **patch** (or no version-affecting change).

Regenerating a baseline (`UPDATE_API_BASELINE=1 go test ./<pkg>/`) is therefore a deliberate
act that should be reviewed against this table before release.

## The CHANGELOG is the migration log

Every release's breaking changes and additions are recorded in `CHANGELOG.md` under the
version heading. A major release's entry must include the **migration steps** (old → new) for
each breaking change. A longer generated guide may accompany a major release, but the
changelog remains the authoritative migration summary and links to that guide.

## Module split in v7

The v7 development branch has moved the CLI and tool-only dependencies into
`tools/`, the browser suite into `test/playwrightgo/`, SDK-backed telemetry tests
into `telemetry/otel/`, and dependency-heavy examples into their own modules.
The root library module retains runtime dependencies. A fresh external consumer
of `/v7/ui` builds natively and for js/wasm with a six-line `go.sum`; it does not
pull in Playwright, Bubble Tea, AI SDK, or Chroma. From the repository root,
invoke the CLI with `go -C tools run ./gwc` and use paths relative to `tools/`
for its arguments. The module-path change and consumer dependency-graph change
ship together as v7.

## Support window & deprecation

A deprecated **Stable** API is not removed in the next release. The commitment:

- A symbol marked deprecated stays present, working, and documented for **at least two minor
  releases and at least 90 days** after the release that deprecated it, before a major release
  may remove it. The deprecating release names the replacement in `CHANGELOG.md`.
- The current major line receives fixes for **at least 12 months** after its successor major
  is published, so a consumer is never forced to upgrade a major on short notice.

The API-baseline tests below mechanically detect when a removal would breach this window (a
baseline line disappears), so the policy is enforced, not just promised.

## API-baseline merge gate

The API-baseline tests (`internal/apidump` + each package's `api_baseline.txt`) run as a
**required PR check** via `.github/workflows/api-baseline.yml` — not only at release. An
accidental breaking change to an exported surface fails the PR before merge, so a major-bump
trigger can never land silently on `main`. Intentional changes regenerate the baseline
(`UPDATE_API_BASELINE=1 go test ./<pkg>/`) as a reviewed commit, classified by the table above.

## Pre-1.0 / pre-stable surfaces

Packages or symbols documented as **Experimental** (see the reference manual's stability
notes) are exempt from the major-bump rule until they graduate to **Stable** — at which point
they gain an `api_baseline.txt` and fall under the table above. Graduating a surface to Stable
is itself a minor release.
