# Enterprise Pilot Readiness

This document defines the minimum bar before GoWebComponents should be recommended for a serious internal enterprise pilot.

Use it as a go/no-go checklist for architecture, engineering, security, and operations reviewers.

## At A Glance

- pilot readiness is a threshold gate, not a marketing claim
- each gate area must have an explicit owner and validation evidence
- unresolved critical gaps in one area block pilot recommendation

## Minimum Bar For An Enterprise Pilot

All six areas below must pass.

### 1. Correctness

- core flows render and hydrate correctly under representative app routes
- route guards, loader failures, and form submission failures fail predictably
- offline mutation replay and conflict handling produce deterministic outcomes
- actionable diagnostics are available for user-visible failures

### 2. Testing

- unit coverage exists for route, state, and mutation control paths
- browser coverage exists for critical end-to-end product journeys
- hydration-sensitive behavior has explicit regression coverage
- release candidates run a documented verify workflow before signoff

### 3. Observability

- logs include stable domains and correlation identifiers
- route, loader, mutation, and hydration failures are measurable
- production dashboards expose error-rate and latency signals for key flows
- incident responders can trace one user flow across client and server events

### 4. Security

- session and auth boundaries are documented and validated
- CSP, cookie, and token handling policies are enforced for deployment targets
- sensitive diagnostics and logs are redacted or avoided by policy
- dependency and supply-chain checks are part of the release process

### 5. Deployment

- build artifacts are reproducible and include manifest integrity checks
- environment configuration ownership is explicit by stage
- rollback steps are documented and exercised for release candidates
- static assets, wasm, and server endpoints align with cache and version policy

### 6. Support And Ownership

- route-family, state, and platform ownership is assigned
- escalation paths exist for runtime, deployment, and data incidents
- upgrade and migration policy is documented for app teams
- enterprise decision makers have one concise evaluation packet

## Production-Shaped Reference Application

The current production-shaped reference application is:

- `examples/86-atlas-commerce-os`

This example is the baseline used for enterprise-pilot validation because it exercises one coherent medium-size app surface rather than isolated feature snippets.

### Required Capability Coverage

The reference app should continuously demonstrate:

- SSR and hydration route ownership
- authenticated and unauthenticated route families
- form workflows and mutation handling
- async data and loader-driven flows
- offline-friendly behavior and replay visibility
- observability hooks and structured diagnostics
- deployment-oriented asset and bootstrap conventions

### Reference-App Validation Rule

Promote pilot readiness only when the reference app remains green for targeted release workflows and representative browser journeys.

## Operational Incident Runbooks

Use this runbook set for production incidents during pilot and post-pilot operation.

### Runbook Template

Each incident type should follow the same sequence:

1. detect and classify severity
2. capture correlation IDs and affected route or flow
3. apply containment (disable feature flag, degrade behavior, or rollback)
4. recover service and validate core user journeys
5. document root cause and preventative follow-up

### Hydration Failures

- symptoms: mismatch warnings, fallback-to-client rendering, broken first interaction
- first checks: bootstrap payload shape, route metadata, server/client markup drift
- containment: route-level fallback or temporary SSR disable for affected branch

### Loader Failures

- symptoms: repeated route-load errors, empty shells, stuck pending state
- first checks: upstream dependency health, auth/session propagation, timeout policy
- containment: stale cache fallback, bounded retry, and user-visible error surface

### Offline Replay Issues

- symptoms: mutation queue growth, repeated conflict rejections, duplicate writes
- first checks: queue entry identity, conflict policy, reconnect ordering
- containment: pause replay, surface pending state, allow operator-assisted retry

### Cache Corruption

- symptoms: stale or contradictory state between routes or tabs
- first checks: cache-key collisions, invalidation gaps, persistence migration drift
- containment: targeted cache namespace reset and revalidation

### Multi-Window Or Multi-Tab Sync Problems

- symptoms: inconsistent selected context or stale auth/session hints across surfaces
- first checks: channel naming, target origin, signal payload versions
- containment: force surface reconnect and re-publish canonical state from owner

### Degraded Route Performance

- symptoms: route transition latency spikes, time-to-interactive regression
- first checks: bundle growth, loader response latency, render-count regressions
- containment: disable heavy optional widgets, defer non-critical loaders, rollback

## Upgrade Rehearsal Guidance

Use this workflow before rolling framework upgrades into production apps.

### Rehearsal Sequence

1. branch from the current production baseline
2. upgrade framework dependencies and update migration notes
3. run targeted contract tests for routes, forms, async loaders, and hydration
4. run browser regression for critical product journeys
5. compare benchmark and route-latency results against previous baseline
6. run deployment validation checklist in staging
7. execute rollback rehearsal and confirm artifact rollback path

### Required Upgrade Evidence

- contract-test pass report for high-risk flows
- before/after benchmark summary for key routes and render-heavy screens
- staging release verification results
- explicit signoff from route, platform, and security owners

### Minimum Command Shape

- `go run ./tools/gwc test -lane unit -lane hydration -lane browser`
- `go run ./tools/gwc verify -app .\\cmd\\web\\main.go -root .`
- `go run ./tools/gwc bench -root . -lane baseline`

## Deployment Validation Checklist

Run this checklist before any production promotion.

### Artifact Integrity

- wasm and static assets match release manifest entries
- checksums or hashes match expected build outputs
- bootstrap and template assets reference the correct hashed artifacts

### Configuration Correctness

- environment values are present for target stage
- feature flags and auth/session settings match release plan
- route and API endpoint base URLs are consistent

### Observability Wiring

- log sinks and trace correlation are enabled
- alert thresholds for route, loader, and mutation failures are active
- dashboard panels for key user flows are populated

### Cache And Compression

- immutable assets use long-lived cache headers
- bootstrap or HTML shells use short-lived cache policy
- gzip or brotli outputs are present and served correctly

### Security Headers And CSP

- CSP policy matches deployed script/style/asset requirements
- security headers are present on HTML and API responses
- no sensitive diagnostic payloads leak in production responses

### SSR And Bootstrap Behavior

- server-rendered routes return expected shell and metadata
- hydration resumes without mismatch spikes on critical paths
- bootstrap payload size and shape remain within expected limits

## Sustained-Load And Long-Session Validation

Use medium-duration scenarios to validate behavior that short smoke tests miss.

### Reference-App Scenario Set

Run these against `examples/86-atlas-commerce-os`:

1. sustained route navigation and loader churn over 30 to 60 minutes
2. long-lived authenticated session with background polling and mutations
3. offline and reconnect cycles with queued mutation replay
4. cross-tab or multi-surface activity with shared-state invalidation
5. repeated form submissions and route transitions under moderate latency

### Validation Goals

- detect memory growth and degraded interactivity over time
- detect replay or cache drift after reconnect cycles
- confirm diagnostics stay actionable under extended usage
- confirm route and mutation latency remains within target budgets

### Cadence

- run before pilot signoff
- run on every release candidate for pilot-facing apps
- run after major router, cache, or hydration behavior changes

## Go/No-Go Rule

Recommend an enterprise pilot only when all areas above are green with recorded evidence.

If one area is yellow or red, treat the pilot as blocked until mitigation and re-validation complete.

## Review Checklist

- are all six gate areas explicitly owned
- is evidence current for the target release candidate
- are security and deployment gaps tracked with due dates
- are support and escalation paths documented for pilot teams
