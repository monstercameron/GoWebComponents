# Security, Compliance, and Governance

This page defines the intended security model and governance expectations for GoWebComponents applications and framework integrations.

Use it when evaluating SSR bootstrap safety, hydration trust boundaries, server-versus-client data handling, browser storage risk, logging redaction, and supply-chain review expectations.

## Threat Model

The primary trust boundaries in the current project are:

- server-rendered HTML and bootstrap payloads crossing from trusted server execution into untrusted browser environments
- route-loader and API data crossing from server-owned handlers into hydrated client state
- browser interop calls crossing from Go code into ambient browser APIs and third-party scripts
- offline storage, mutation queues, and cross-tab channels persisting or relaying data in browser-controlled storage
- worker and multi-surface messaging moving JSON-shaped payloads across browser execution contexts
- structured logs, diagnostics, and traces carrying operational detail into developer tools or external sinks

The core assumption is:

- the browser is not a secret-safe environment
- anything serialized into HTML, bootstrap payloads, local storage, or browser-visible logs must be treated as client-visible
- same-origin server handlers remain the authoritative security boundary for auth, CSRF, validation, redirects, and sensitive data access

## Server-Only Versus Client-Safe Data

The intended rule is explicit classification before serialization.

Client-safe data may include:

- route params, queries, and public route metadata
- public feature flags or presentation hints
- locale, directionality, and other non-secret startup state
- cache seeds or route data that the server is already comfortable rendering into HTML

Server-only data must never cross into bootstrap or browser-visible logs:

- session secrets
- bearer tokens
- raw auth-provider credentials
- CSRF secrets or token-generation material
- private upstream API responses that exceed what the rendered page actually needs
- internal policy, moderation, or compliance metadata that is not explicitly intended for the client

Recommended rule:

- if the browser only needs the result, render the result instead of serializing the secret inputs
- if a value is not acceptable in page source, it is not acceptable in bootstrap, local storage, worker messages, or framework-owned logs

## Secure-By-Default SSR And Hydration

SSR and hydration should default toward escaped, explicit, and minimally serialized output.

The intended secure defaults are:

- escape user-controlled text in rendered HTML unless an application deliberately uses a documented raw-output escape hatch
- keep bootstrap payloads JSON-shaped and serialization-safe rather than embedding executable code
- treat inline bootstrap as public data and sidecar bootstrap as public data delivered out-of-band, not as a secret channel
- keep CSP, nonce, and inline-script policies explicit at the server document-template layer
- avoid deriving browser-executed behavior from unvalidated user-controlled markup or script fragments

Recommended SSR rules:

- keep user-generated HTML rendering behind application-owned sanitization policy
- keep metadata, preload hints, and route-owned head output explicit rather than concatenating raw strings
- keep route loaders and SSR handlers responsible for narrowing sensitive upstream data before rendering or bootstrapping it
- treat hydration mismatch recovery as a correctness concern, not a security layer

## Logging, Diagnostics, And Secret Handling

Framework-owned diagnostics and logs must be redaction-first.

The intended policy is:

- never emit raw cookies, authorization headers, CSRF secrets, bootstrap payload bodies, or full form submissions
- prefer route ids, status codes, durations, cache keys, payload-size buckets, and mutation kinds over raw payload content
- keep correlation ids opaque and non-user-identifying
- avoid copying browser storage contents, worker payload bodies, or interop return objects into logs by default
- treat exported traces and diagnostics under the same redaction boundary as logs

For the experimental multi-client surface documented in `docs/MULTI_CLIENTS.md`, the same-origin default, topic-level authorization, origin validation, and no-raw-binary-logging rules apply to peer coordination just as they do to other browser-visible interop flows.

The current helper layer now enforces a first runtime slice of that policy by rejecting privileged multi-client topic families for non-privileged roles, surfacing same-peer origin mismatch as structured unauthorized errors on window subscriptions, and failing closed on stale popup or opener handles.

Application code may choose to log more detail, but that is outside the framework-owned safe default.

## Supply-Chain Review Practices

The intended supply-chain posture is conservative and reviewable.

Recommended practices:

- treat Go module dependencies, npm-based dev tooling, browser-test packages, and build-time asset tooling as separately reviewable trust boundaries
- prefer official toolchain artifacts for Go, `wasm_exec.js`, and browser-test runners
- document any new build or post-processing step that rewrites wasm, JS, or generated HTML before release
- review lockfile and dependency changes with the same care as framework code changes when they affect shipped assets or release pipelines
- keep generated release artifacts reproducible from the tagged source tree and documented build steps

For enterprise review, the important question is not only "what code runs in production?" but also "what tools shape the produced wasm, JS, HTML, and test or release pipeline?"

## Incident Response And Vulnerability Reporting

Security issues need a documented path, not only commit history.

The intended incident-response posture is:

- provide a clear security-reporting contact or process in release-facing documentation
- triage reported vulnerabilities separately from ordinary bugs when user data, auth boundaries, SSR output safety, or supply-chain integrity may be affected
- prioritize fixes that narrow active exposure before broader cleanup or refactoring
- publish patch and disclosure guidance once the affected versions, mitigation path, and upgrade recommendation are clear

Recommended workflow:

1. acknowledge the report and establish a private triage path
2. determine affected versions and blast radius
3. prepare the fix, tests, and release notes
4. publish the patched release and migration or mitigation guidance
5. disclose the issue with enough detail for consumers to assess their own exposure

## Compliance-Oriented Deployment Guidance

Regulated environments need operational guidance beyond "it works."

Recommended deployment posture:

- keep build and release pipelines reproducible enough to explain how shipped wasm, JS, HTML, and manifest artifacts were produced
- separate environments clearly so development, staging, and production do not share secrets, mutable infrastructure, or logging sinks accidentally
- align audit logging with the application's own auth, mutation, and administrative boundaries rather than assuming framework lifecycle logs are sufficient evidence
- apply explicit retention and deletion policy to browser-visible logs, bootstrap payloads, and stored offline data when the application domain requires it
- review CDN, reverse-proxy, and browser-storage behavior for regulated workloads so public caches and local persistence do not outlive their intended policy window

The framework can document these constraints, but application operators still own the actual control implementation.

## Compliance-Oriented Posture

GoWebComponents itself is not a compliance certification.

The intended posture is:

- provide clear data-boundary guidance so applications can meet their own regulatory obligations
- keep server-only and client-safe data handling explicit
- keep logging and diagnostics redaction-compatible with regulated environments
- keep deployment, browser-support, and offline-storage guidance explicit enough for downstream audits

Applications remain responsible for:

- data retention
- consent and privacy policy
- encryption and key management
- access control
- audit logging requirements
- regulatory mapping to frameworks such as SOC 2, HIPAA, PCI, or GDPR

## Current Boundary

This document defines the intended security and governance contract.

It does not yet claim:

- complete security regression test coverage for all critical surfaces
- formal compliance templates or certification mappings

Those remain separate backlog work.
