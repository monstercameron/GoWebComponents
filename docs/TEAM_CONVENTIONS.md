# Team-Scale Conventions

This document defines team-scale conventions for medium and large GoWebComponents applications.

Use it to keep architecture, code review, and shared component workflows consistent across multiple contributors.

## At A Glance

- split shared UI and domain code intentionally
- standardize framework usage patterns before teams diverge
- review hydration, routing, async, and interop changes with explicit checklists
- enforce repository hygiene for imports, generated code, tests, and outputs
- assign explicit ownership for route areas, shared state, and deployment boundaries
- keep onboarding and component-library workflows repeatable

## Shared UI And Domain Abstractions

Recommended layering:

```text
internal/
  app/
    routes/
    features/
    shell/
  ui/
    design/
    components/
  domain/
    entities/
    services/
  platform/
    api/
    storage/
```

Conventions:

- shared UI in `internal/ui/*` should avoid embedding route-specific domain behavior
- feature modules in `internal/app/features/*` own route-local composition and workflow hooks
- domain services in `internal/domain/*` should not depend on browser-only packages
- avoid cross-feature imports from sibling route modules when a shared abstraction exists

## Framework Usage Consistency Across Teams

Define one preferred style per surface:

- hooks: prefer small workflow hooks that wrap `ui.UseState` or `ui.UseReducer` for feature logic
- shared state: use `state` atoms/selectors for cross-route state; avoid ad hoc globals
- loaders and async resources: keep route-critical fetches in loaders, component-local fetches in feature hooks
- forms: standardize on one validation and submit outcome projection pattern per app

Team rule:

- new feature PRs should follow existing app patterns unless they also include migration notes

## Code-Review And Migration Checklists

### Code-Review Checklist

- hydration-sensitive paths: server/client markup and bootstrap assumptions remain aligned
- route changes: params, redirects, guards, and metadata updated together
- async flows: loading, retry, cancel, and error states are explicit
- interop boundaries: browser-only APIs stay behind typed adapters and feature guards
- diagnostics: warnings and actionable logs remain available for failure paths

### Migration Checklist

- identify changed public package contracts
- update app wrappers and integration points first
- run targeted regression suites for routes, forms, and hydration
- document app-specific migration notes and rollback plan

## Linting, Formatting, And Repository Hygiene

Minimum standards for application repos:

- `gofmt` and `goimports` in CI
- deterministic import grouping and no unused exports
- generated artifacts tracked or ignored by explicit policy
- one test location convention (`test/` and package-local tests)
- release artifacts emitted to dedicated output directories

Hygiene guardrails:

- do not commit temporary wasm outputs outside designated build directories
- keep generated code ownership documented via `go:generate` comments
- keep example or fixture assets versioned when they are test contracts

## Multi-Person Ownership Of App Architecture

Ownership split model:

- route-family owners: workspace, marketing, admin, auth
- shared-state owners: cache keys, mutation queue, persistence policy
- platform owners: deployment, observability, release pipeline

Coordination rules:

- architecture decisions are recorded in app docs, not only in PR threads
- shared wrappers require cross-owner review
- route-family changes affecting shell contracts require shell-owner approval

## Medium-Sized App Structure Example

```text
cmd/
  web/main.go
  server/main.go
internal/
  app/
    routes/
      marketing/
      workspace/
      admin/
    features/
      billing/
      projects/
      notifications/
    shell/
      layout.go
      nav.go
    loaders/
    forms/
  ui/
    design/
    components/
  domain/
  services/
web/
  templates/
  static/
    wasm/
    css/
test/
  browser/
  integration/
```

Boundary rule:

- dependencies should flow from route/features to shared UI/domain, not the reverse

## Team Onboarding Playbook

Onboarding sequence for new engineers:

1. read app architecture overview and route map
2. run local dev, test, and verify commands
3. complete one guided feature edit touching route, state, and tests
4. complete one debugging exercise for loader or hydration failure

Onboarding deliverables:

- route ownership map
- state and cache ownership map
- deployment and environment map
- testing strategy map (unit, browser, SSR/hydration)

## Design-System And Component-Library Workflow

Recommended workflow:

- design tokens and primitives live in shared UI package
- feature code composes primitives rather than re-implementing styles
- accessibility guarantees are owned at primitive/component-library level
- SSR-safe behavior is required for shared components used in server-rendered routes

Release workflow:

- publish component changes with changelog entries and migration notes
- keep visual and accessibility regression tests for shared components

## Packaging And Release Guidance For Internal Component Libraries

Packaging rules:

- version shared UI packages with semver
- define supported framework version range per package release
- publish compatibility notes for breaking design-token or API changes
- validate one consumer app before promoting a release

Release checklist:

1. run unit and browser regression for component package
2. run one integration smoke test in consuming app
3. publish release notes with upgrade actions
4. update starter or template dependencies if required

## Review Checklist

- are shared UI and domain boundaries explicit and enforced
- do teams follow one framework usage style for hooks, loaders, and forms
- are review and migration checklists used for framework-heavy changes
- are repository hygiene standards documented and automated
- is ownership split clear across routes, state, and deployment concerns
- are onboarding and component-library workflows reproducible
