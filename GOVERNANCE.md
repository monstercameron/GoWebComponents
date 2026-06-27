# Project governance

This document records how decisions are made and how quality is enforced, so the bar is a
process, not a mood.

## Decision-making

GoWebComponents is maintainer-led. Changes land through pull requests reviewed against the
rules below; the maintainer is the final arbiter on scope and design direction (see the
reference manual's *Scope And Non-Goals*). Proposals that widen the supported public surface
must come with a clear ownership story that survives the semver policy.

## What "stand up to scrutiny" means here (the gates)

Quality is enforced mechanically by checks in the repo, not by reviewer memory:

- **Public API can't drift silently.** Stable packages carry `api_baseline.txt` golden tests
  (`internal/apidump`). Any change to an exported type/func/method/const/var fails CI until
  the golden is regenerated *with intent* — which, per [`VERSIONING.md`](./VERSIONING.md),
  also decides whether the release is major or minor.
- **Dependency footprint is budgeted.** `gwc supplychain` proves the zero-npm posture, counts
  direct/transitive Go deps against a budget, and verifies module checksums. A growth in the
  library module's dependency footprint is a reviewable event, not a silent one.
- **Known vulnerabilities are gated.** `gwc vuln` runs govulncheck and fails on a *reachable*
  advisory.
- **Every feature ships tested.** Native + wasm where relevant, with e2e coverage on complex
  surfaces; the headless story-runner (`workbench.RunStories`) turns examples into smoke
  tests.

## Stability tiers

Surfaces are **Experimental** or **Stable** (see the reference manual's stability notes).
Experimental surfaces may change without a major bump; graduating one to Stable gives it an
API baseline and brings it under the versioning policy — and is itself a minor release.

## Contributing

See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for how to build, test, and submit changes. The
short version: every change is atomic to its feature, carries tests, keeps `go vet` clean,
and updates `CHANGELOG.md`.

## Structural note

As a maintainer-led project, the community-governance dimension has a structural ceiling
(there is no multi-org steering committee). The intent here is not bureaucracy but
*legibility*: anyone can read the gates above and predict whether a change is acceptable and
how it will be versioned.
