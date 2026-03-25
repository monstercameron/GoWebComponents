# Adoption Maturity, Starters, and Enterprise Evaluation

This document defines adoption maturity criteria, starter and ecosystem reporting, and evaluation artifacts for teams deciding whether GoWebComponents is ready for production use.

Use it as the decision packet for maintainers, architects, and procurement reviewers.

## At A Glance

- production readiness is feature-area specific, not one global label
- the primary reference application is `examples/86-atlas-commerce-os`
- starters are tracked by support tier and verification cadence
- ecosystem entries are tracked with ownership and maintenance signals
- enterprise evaluation uses one concise packet and explicit risk labels
- learning and community paths extend beyond core runtime code contribution

## Production-Readiness Criteria By Feature Area

Use these readiness tiers:

- `Stable`: expected for production use with documented upgrade path
- `Supported`: usable with known limits and active maintenance
- `Experimental`: validation in progress; expect contract changes

### Feature-Area Criteria Matrix

| Feature Area | Current Tier | Promotion Gate |
| --- | --- | --- |
| Components, local state, rendering | Stable | Contract tests and migration notes remain green each release |
| Router core, params, guards, loaders | Stable | Route examples and regression suites stay current |
| Shared state and async cache basics | Stable | Cache invalidation and hydration paths remain documented and tested |
| Forms and server-post flows | Supported | More large-form reference coverage and sustained browser regression |
| SSR and hydration | Supported | Broader production app validation under real deployment topologies |
| PWA and offline queue flows | Supported | Long-session replay and conflict workflows validated in reference app |
| Compiler and server-interactive experiments | Experimental | Separate experiment lifecycle with explicit opt-in boundaries |

## Real-World Case Study And Reference Application

Primary reference application:

- `examples/86-atlas-commerce-os`

Why it is the current maturity anchor:

- combines routing, auth-aware shells, async data, forms, and workspace workflows
- exercises SSR/hydration-aware app structure and asset delivery expectations
- validates cross-cutting concerns in one medium-size codebase

Case-study use in reviews:

- architecture review source: route and shell composition
- operations review source: build, verify, and release flow
- product-workflow review source: auth, workspace, and mutation patterns

## Starter Matrix With Support Tiers And Cadence

Starter catalog and expectations:

| Starter | Intended App Shape | Tier | Verification Cadence |
| --- | --- | --- | --- |
| `minimal-client` | single-runtime client app | Supported | every release |
| `routed-spa` | route-heavy client app | Supported | every release |
| `ssr-app` | SSR + hydration app | Supported | every release |
| `reference-app` | production-shaped baseline | Supported | every release + quarterly deep review |

Cadence rules:

- release cadence validation confirms starter generation and baseline tests
- quarterly deep review validates docs alignment, dependency freshness, and migration clarity
- starter tier changes must be documented in release notes and migration docs

## Ecosystem Inventory And Maintenance Signals

Inventory categories:

- first-party runtime packages
- supported companion packages
- example-only integrations
- external community integrations

Maintenance signals per entry:

- owning team or maintainer
- support tier (`Stable`, `Supported`, `Experimental`)
- latest verification date
- test status and coverage summary
- compatibility notes and known limits

Minimum inventory update rule:

- update inventory at each release cut
- mark stale entries when verification age exceeds one release cycle

## Enterprise Evaluation Packet

Publish one concise packet for architecture and procurement review.

Required packet sections:

1. support posture and support-tier definitions
2. browser support and platform assumptions
3. security and session-boundary model
4. upgrade and migration discipline
5. operational ownership boundaries (app, framework, companion packages)
6. known maturity risks and open experimental surfaces

Packet sources:

- `docs/API_POLICY.md`
- `docs/BROWSER_SUPPORT.md`
- `docs/SECURITY.md`
- `docs/MIGRATIONS.md`
- this document

## Guided Learning Tracks And Workshops

Learning tracks should be sequential and task-driven.

Track set:

1. client-only app foundations
2. routed SPA and loader flows
3. SSR and hydration delivery
4. forms-heavy application workflows
5. offline-capable and cross-tab workflows

Workshop format requirements:

- each track has one runnable baseline and one extension exercise
- each track defines validation checkpoints and expected outputs
- each track links to one production-shaped example

## Community-Growth Paths Beyond Core Runtime

Public contribution lanes:

- documentation and migration guide improvements
- production-shaped examples and starter improvements
- companion-package integrations and maintenance
- testing utilities and regression scenarios
- case-study writeups and deployment recipes

Contribution governance requirements:

- each lane has explicit reviewers and acceptance criteria
- contribution scope and support tier impact are documented in PRs
- community-owned surfaces must still publish compatibility notes

## Review Checklist

- are readiness tiers feature-specific and current
- is the reference application maintained and cited consistently
- does the starter matrix include tier and cadence per starter
- is ecosystem ownership and support signal visibility sufficient
- does enterprise packet content map to concrete docs
- are learning tracks and non-runtime contribution lanes explicit
