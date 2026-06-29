# Versioning & migration policy

GoWebComponents follows [Semantic Versioning](https://semver.org) for its public Go API,
and — because this is a Go module — the import path encodes the major version. This document
commits the actual rules so a release's number is predictable, not a judgment call.

## What each bump means

| Bump | Trigger | Examples |
|---|---|---|
| **Major** (`/v5`, `/v6`, …) | A breaking change to an **exported** identifier in the library module, or any change that forces a consumer to edit code or that alters a consumer's `go.sum`. | Removing/renaming an exported func/type/method; changing an exported signature; moving a dependency out of the library module (see *Module split* below). |
| **Minor** (`x.Y.0`) | A backward-compatible **addition**. | New exported function/type/method; a new package; a new optional field with a zero-value default. |
| **Patch** (`x.y.Z`) | A backward-compatible **fix** with no API change. | Bug fixes, performance, docs, internal refactors, new tests. |

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
each breaking change. There is no separate migration guide to fall out of date; the changelog
entry is the migration guide.

## Module split (the open `/v4` decision)

The plan's F9 item proposes moving heavy tool-only dependencies (`playwright-go`,
`bubbletea`, `charmbracelet/*`, the AI SDKs) into a separate **`tools` module** so the
library module retains only runtime deps. Whether that lands as a major or a minor is decided
by this rule:

- If any **library-consumer** `require`/`go.sum` line changes (it will, because transitive
  tool deps disappear from consumers' graphs), the split is a **breaking** change to the
  module graph and **must** ship as a major (`/v4` → import-path bump).
- If it is a pure internal reorg with **no** change to any public import path or consumer
  `go.sum`, it can ship as a **minor**.

That decision is the gate for doing the split; it is intentionally **not** made unilaterally.

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
