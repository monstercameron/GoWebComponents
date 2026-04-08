# API Stability and Support Policy

This document defines which GoWebComponents surfaces are intended for long-term production use, how semantic versioning is applied, how deprecations work, and what support expectations enterprise adopters should plan around.

This policy applies to the module published at `github.com/monstercameron/GoWebComponents` beginning with the current `v3.x` public package layout.

## At A Glance

This page answers one practical question: what can adopters rely on staying stable, and what can maintainers still reshape?

Use it when you need to decide:

- whether a package or helper is safe for long-term production use
- whether a proposed change belongs in a patch, minor, or major release
- whether a new API must ship as stable, supported companion, experimental, or internal
- whether migration guidance is required before a release can go out

The goal is not just semver correctness. The goal is to make support expectations explicit before users build production dependencies on the wrong surface.

## Stability Tiers

GoWebComponents uses four stability tiers.

## How To Classify A Surface

Use this quick rule set when deciding where a new API belongs:

- Stable public API: default choice for core package APIs that production apps are expected to depend on long term
- Supported companion API: production-safe integrations whose payloads may grow additively without changing their core contract
- Experimental public API: public and usable, but still proving shape, defaults, or lifecycle semantics
- Internal and unsupported surface: implementation detail, migration shim, or repo-only helper that consumers should not treat as a contract

If maintainers are not willing to document migration guarantees for a surface, it should not be labeled stable.

### 1. Stable public API

These surfaces are intended for long-term production use and follow the full semver and deprecation guarantees in this document.

Current stable public surfaces are:

- `ui` core component authoring and rendering APIs such as `CreateElement`, `Fragment`, `Render`, `Hydrate`, `Portal`, `PortalTarget`, `CreateContext`, `UseContext`, `UseState`, `UseReducer`, `UseEffect`, `UseMemo`, `UseRef`, `UsePrevious`, `UseId`, `UseEvent`, `UseForm`, and `ErrorBoundary`
- `html` typed element builders and documented `html.Props` behavior
- `state` atoms, `UseComputed`, `UseDerived`, snapshot export or import helpers, and browser snapshot persistence helpers
- `fetch` low-level `Fetch`, `UseFetch`, and typed `UseResource`
- `router` core router creation, route registration, mounting, navigation, params, query helpers, redirects, and nested `GetOutlet()` composition
- `ui.RenderToString`, `ui.Hydrate`, SSR bootstrap helpers, and `router.HydrateMount` when used through documented package APIs

Stable means:

- Breaking signature, type-shape, or documented behavior changes only happen in a major release.
- Removals require a prior deprecation window.
- Migration guidance must exist before the breaking change ships.

### 2. Supported companion API

These APIs are supported for production use, but they are integration-oriented rather than core rendering primitives.

Current supported companion surface:

- `devtools` public APIs such as `Panel`, `UseSnapshot`, `SnapshotNow`, and exported inspection types
- `head` companion helpers such as `Compose`, `MetaName`, `MetaProperty`, `Robots`, `OpenGraph`, `Twitter`, and `SocialTags`

Supported companion APIs follow the same major-version breaking-change rules as stable APIs, with one extra caveat: additive diagnostics are allowed in minor releases.

That means minor releases may:

- add new diagnostic kinds
- add new profiling counters
- add new optional fields to snapshot payloads intended for keyed access
- improve diagnostic wording

Minor releases must not:

- remove documented fields or functions
- change the meaning of an existing documented field
- require consumers to rewrite existing integrations unless the release is major

### 3. Experimental public API

Experimental APIs are public and usable, but they are not yet locked for long-term compatibility. They may change in a minor release if the release notes and migration guide explain the change.

Current experimental surfaces are:

- concurrent-style scheduling primitives such as `ui.StartTransition`, `ui.UseTransition`, and `ui.UseDeferredValue`
- explicit async subtree primitives such as `ui.AsyncBoundary` and `ui.Lazy`
- additive `html` authoring sugar helpers documented in `docs/HTML_SUGAR.md`, including `Children`, `PropsOf`, `Textf`, `TextIf`, `When`, `ClassNames`, `If`, `IfElse`, `Unless`, `Map`, and the option-helper surface built on `PropsOf(...)`
- advanced cached-resource flows in `fetch.UseCachedResource`
- advanced router data and lifecycle APIs including route loaders, `UseRevalidator`, guards, route-managed metadata, and layout-heavy hydration flows
- optional SSR bootstrap transport details beyond the documented JSON bootstrap path, including alternative payload encodings
- the `plugin` companion package, including `Host`, `Plugin`, manifests, lifecycle helpers, and subsystem hook registration APIs
- the multi-client coordination proposal and current helper layer documented in `docs/MULTI_CLIENTS.md`

Experimental means:

- additive changes are allowed in minor releases
- behavioral tightening or API reshaping may happen in minor releases when required to reach a stable design
- changes still require release notes and migration guidance
- removals should still be announced before they happen unless the API was only shipped in the immediately previous minor release and was explicitly labeled experimental at launch

When an experimental surface matures, it should be reclassified as stable or supported companion API in this document and in package-level docs where relevant.

### 4. Internal and unsupported surface

These surfaces are implementation details and are not covered by semver compatibility promises for application code.

This includes:

- everything under `internal/`
- runtime details such as fibers, schedulers, hook slot layout, DOM adapter internals, and hydration bookkeeping
- example app code under `examples/`
- test helpers and benchmark harnesses under `test/` and `tools/`
- compatibility shims retained only so older code still compiles during migration, unless explicitly called out as stable

Consumers should not import or depend on these details for long-term production code.

## Compatibility Markers

The repo uses the following labels in docs and release notes:

- `Stable`: safe default for long-term production use
- `Supported companion`: production-safe, but integration payloads may grow additively
- `Experimental`: public but still evolving
- `Deprecated`: still supported temporarily, scheduled for removal after the published migration window
- `Internal`: not a supported consumer contract

APIs that exist only for backward compatibility, such as older router compatibility helpers, should be treated as deprecated compatibility surface even if they remain callable today.

## Release Review Checklist

Before shipping a release that touches public API, confirm all of the following:

- every changed surface still has a documented stability tier
- the release type matches the strongest compatibility impact of the change
- any deprecation includes a clear replacement path
- any behavior change that adopters will notice is reflected in `CHANGELOG.md` and, when needed, `docs/MIGRATIONS.md`
- experimental changes are clearly labeled as such rather than relying on ambiguity
- no internal helper or example-only API is accidentally being presented as supported product surface

If one of these checks fails, the release policy is incomplete even if the code itself is ready.

## Semver Policy

GoWebComponents follows semantic versioning for all stable and supported companion APIs.

### Patch releases

Patch releases may include:

- bug fixes
- security fixes
- performance improvements
- documentation updates
- test additions
- internal refactors that do not change public contracts
- conservative behavior fixes where the previous behavior was clearly a bug relative to existing docs or tests

Patch releases must not:

- remove or rename exported stable APIs
- change public function signatures
- change documented semantics in a way that requires consumer code changes
- silently change router matching, navigation, SSR, hydration, form, state, or fetch behavior that valid applications may rely on

### Minor releases

Minor releases may include:

- new packages
- new exported functions, types, and helpers
- new opt-in capabilities
- additive options on existing APIs when they do not invalidate existing code
- experimental API evolution with release notes and migration guidance

Minor releases must preserve backward compatibility for stable and supported companion APIs.

For Go consumers, the following count as breaking and therefore require a major release:

- removing, renaming, or changing the type of an exported symbol
- adding methods to an exported interface that consumers may implement
- changing the shape of exported structs in a way that breaks existing composite literals or field access
- changing generic type parameter expectations in a way that breaks valid existing code
- changing default behavior for documented APIs when existing applications would render, navigate, hydrate, validate, or fetch differently without opting in

### Major releases

Major releases may include:

- removal of previously deprecated APIs
- compatibility cleanup of legacy shims
- behavior changes that require consumer code or deployment changes
- design resets for experimental APIs that have not yet been promoted to stable

Every major release must ship with a migration guide that covers the affected public surfaces.

## What Counts As A Breaking Change

For this project, a change is breaking if a consumer on a supported surface must modify code, markup, deployment configuration, or operational expectations to preserve the previously documented behavior.

That includes:

- route matching precedence changes
- different redirect or guard behavior for the same registered routes
- hydration behavior changes that alter whether existing DOM is reused or replaced for documented valid markup
- changed form submission, dirty-state, touched-state, or validation sequencing
- changed atom snapshot export or import semantics
- changed fetch hook cancellation, reload, or ready/error semantics
- changed HTML helper output for the same documented inputs
- changed SSR bootstrap script format for the documented default transport

Bug fixes are not considered breaking when the old behavior contradicted the documented contract, even if a consumer accidentally depended on the bug.

This policy treats operational and behavioral contracts as seriously as exported signatures. If a valid app must change routing, hydration, deployment, or migration assumptions to keep working, the change is breaking even when the Go type checker stays silent.

## Deprecation Lifecycle

Stable and supported companion APIs follow this lifecycle:

1. A replacement or clear alternative must exist first.
2. The API must be marked deprecated in Go doc comments with a `Deprecated:` notice.
3. The deprecation must be announced in `CHANGELOG.md` and linked from `docs/MIGRATIONS.md`.
4. The deprecated API remains supported for at least two minor releases and at least 90 days, whichever is longer.
5. Removal happens only in a major release.

Before removal, the migration guide must explain:

- what to replace
- why the old API is going away
- whether behavior changes as part of the migration
- any code patterns that need manual review

Experimental APIs can have a shorter lifecycle, but any non-trivial removal still requires release notes and migration guidance.

## Migration Guide Requirements

Any release that changes public behavior in a way adopters are likely to notice must update `docs/MIGRATIONS.md`.

Major-release migration guides must include dedicated sections for:

- runtime or component authoring changes
- router changes
- SSR and hydration changes
- forms and local-state changes
- shared state changes
- fetch or resource-loading changes
- testing or deployment adjustments if the release changes them

## Support Policy

GoWebComponents currently supports the latest released major version only.

Current support expectations are:

- no long-term-support release line is promised today
- no general security-backport policy exists for older majors
- fixes land on the latest patch release of the latest major
- enterprise adopters should plan to stay current within the active major and evaluate major upgrades during the published migration window

If the project later introduces LTS branches or backport windows, this document should be updated before those guarantees are advertised.

## Browser Support Changes

Browser-support policy is part of the public support contract.

That means:

- documented supported browser families and capability baselines must live in `docs/BROWSER_SUPPORT.md`
- release notes and migration guidance must call out any narrowing of the browser matrix or materially new browser-feature requirement
- patch releases must not silently drop a documented supported browser family
- any planned browser-support reduction should follow the same evidence-first discipline used for other support-policy changes, with a clear rationale and migration guidance

## Enterprise Adoption Guidance

Teams evaluating long-lived production use should treat the following as the safe baseline today:

- depend on documented stable public packages only
- avoid `internal/*`, test helpers, example code, and runtime internals
- prefer stable APIs over experimental ones for multi-year product commitments
- take patch releases within the active major promptly
- budget for migration work when adopting experimental router, async, or advanced cache features

## Maintainer Rule

When uncertainty exists, classify conservatively.

- Do not imply stability through omission.
- Do not leave new exported surface unlabeled.
- Do not rely on examples or tests alone to communicate support status.

If a team cannot tell whether a surface is safe to adopt, the policy is not doing its job.

## Maintenance Rule

When a new public API is added, the release should also decide and document its tier. New public surface should not remain unlabeled beyond the release that introduced it.
