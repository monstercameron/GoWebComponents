# Security, Compliance, and Governance

This page defines the current security boundary for GoWebComponents applications.

Use it when evaluating SSR bootstrap safety, hydration trust boundaries, CSRF-aware form handling, browser storage risk, logging redaction, devtools exposure, and supply-chain review expectations.

## Current Status

Shipped today:

- typed SSR bootstrap transport through `ui.SSRBootstrap` and `ui.SSRBootstrapReference`
- bootstrap payload registration and read helpers through `ui.RegisterBootstrapPayload(...)` and `ui.ReadBootstrapPayload(...)`
- scoped bootstrap helpers for route data, form defaults, cache seeds, and session hints
- bootstrap schema versioning and decode-time validation for inline and sidecar payloads
- SSR observability hooks through `ui.ObserveSSR(...)` and the observed bootstrap render helpers
- CSRF naming helpers through `ui.NewCSRFToken(...)`, `ui.DefaultCSRFHeaderName`, and `ui.DefaultCSRFFormFieldName`
- structured logging through `logging.New(...)` plus optional browser event wiring through `logging.AttachBrowserConsole(...)`
- browser devtools snapshots through `devtools.SnapshotNow(...)`, `devtools.UseSnapshot(...)`, and `devtools.ExportSnapshotJSON(...)`
- structured diagnostics metadata that now includes code, docs, remediation, and recoverable classification

Not shipped today:

- a complete application security framework for auth, session issuance, CSP, or secret rotation
- formal compliance mappings for SOC 2, HIPAA, PCI, GDPR, or similar regimes
- a framework-owned redaction engine that can make unsafe application logs safe after the fact
- full security regression coverage for every server integration and browser persistence path

The current framework surface gives applications safer transport and inspection primitives. It does not replace application-owned security design.

## Threat Model

The main trust boundaries in the current project are:

- server-rendered HTML and bootstrap payloads crossing from trusted server execution into untrusted browser environments
- route-loader and API data crossing from same-origin handlers into hydrated client state
- browser interop calls crossing from Go code into ambient browser APIs and third-party scripts
- offline storage, mutation queues, cache state, and cross-tab channels persisting or relaying browser-visible data
- worker and multi-surface messaging moving JSON-shaped payloads across browser execution contexts
- logs, diagnostics, observability events, and devtools snapshots carrying operational detail into browser-visible or exported sinks

Core assumptions:

- the browser is not a secret-safe environment
- anything serialized into HTML, bootstrap payloads, browser storage, worker messages, or browser-visible logs must be treated as client-visible
- same-origin server handlers remain the authoritative boundary for auth, CSRF validation, redirects, access control, and sensitive data access

## Server-Only Versus Client-Safe Data

The current rule is explicit classification before serialization.

Client-safe data may include:

- route params, queries, and public route metadata
- public feature flags or presentation hints
- locale, directionality, environment labels, and similar non-secret startup state
- route data, cache seeds, and form defaults that the server is already willing to expose in page HTML
- session hints that describe client-visible state without carrying credentials or authorization secrets

Server-only data must never cross into bootstrap, storage, or framework-owned logs:

- session secrets
- bearer tokens
- raw auth-provider credentials
- CSRF secrets or token-generation material
- private upstream responses that exceed what the rendered page actually needs
- internal policy, moderation, or compliance metadata that is not explicitly intended for the client

Recommended rule:

- if the browser only needs the result, render the result instead of serializing the secret inputs
- if a value is not acceptable in page source, it is not acceptable in bootstrap, local storage, worker messages, logs, diagnostics, or devtools snapshots

## SSR, Bootstrap, And Hydration Rules

The shipped SSR surface is explicit and typed, but it is still a public-data channel.

Current security-relevant properties of the public bootstrap layer:

- `ui.RenderBootstrapScript(...)` and `ui.RenderBootstrapReferenceScript(...)` serialize data, not executable application logic
- `ui.SSRBootstrap` and `ui.SSRBootstrapReference` carry explicit versions so older clients reject newer payload schemas instead of silently misreading them
- `ui.ReadBootstrapScript(...)`, `ui.ReadBootstrapReferenceScript(...)`, and `ui.ReadBootstrapReference(...)` read only what the server already chose to expose
- typed helpers such as `ui.RegisterRouteBootstrapData(...)`, `ui.RegisterFormBootstrapDefaults(...)`, `ui.RegisterCacheBootstrapSeed(...)`, and `ui.RegisterSessionBootstrapHint(...)` make ownership clearer than ad hoc global JSON blobs

Required SSR rules:

- treat both inline bootstrap and sidecar bootstrap as public data channels, not as secret transport
- keep CSP, nonce, and inline-script policy explicit in the server document template or HTTP layer
- keep route loaders and SSR handlers responsible for narrowing upstream data before rendering or bootstrapping it
- keep user-generated HTML behind application-owned sanitization policy
- treat hydration mismatch recovery as a correctness concern, not a security boundary

## Forms, CSRF, And Server Authority

Server-backed mutations remain server-authoritative.

The current framework help is intentionally narrow:

- `ui.NewCSRFToken(...)` normalizes the default header and hidden-field names
- `ui.DefaultCSRFHeaderName` and `ui.DefaultCSRFFormFieldName` make transport naming consistent across server-rendered forms and imperative requests
- `ui.UseForm(...)` can apply server-shaped errors and preserve CSRF token conventions, but it does not validate tokens itself

Current recommended model:

- source the CSRF token from the current server-rendered page or bootstrap payload
- set a matching server cookie when using the double-submit pattern
- emit the token through `csrf_token` for HTML forms or `X-CSRF-Token` for imperative mutation requests unless the server owns a different contract
- treat `403` CSRF failures as authoritative server failures rather than retrying blindly

The reference example for this boundary is `examples/87-ssr-secure-forms`, which demonstrates request-time rendering, CSRF-aware posts, multipart validation, and post-submit redirects.

## Logging, Diagnostics, Devtools, And Secret Handling

Framework-owned inspection surfaces are redaction-first, but they are still exposure surfaces.

Current rules:

- never emit raw cookies, authorization headers, CSRF secrets, bootstrap payload bodies, or full form submissions into framework-owned logs
- prefer route ids, status codes, durations, cache keys, mutation kinds, counts, and payload-size metrics over raw payload content
- keep correlation ids opaque and non-user-identifying
- treat diagnostics metadata and devtools snapshots under the same redaction boundary as logs
- do not assume snapshot export is safe for production support bundles unless the application has already reviewed what the runtime can observe

Current shipped surfaces to treat as browser-visible or exportable:

- `logging.AttachBrowserConsole(...)` on `js/wasm` entrypoints
- `ui.ObserveSSR(...)` and the observed SSR/bootstrap helpers
- `devtools.SnapshotNow(...)`, `devtools.UseSnapshot(...)`, and `devtools.ExportSnapshotJSON(...)`
- runtime diagnostics surfaced through error boundaries, devtools, and diagnostic metadata fields

Application code may choose to log or export more detail, but that is outside the framework-owned safe default.

## Multi-Client, Worker, And Browser Storage Boundaries

The same browser-visible data rules apply to multi-client coordination, workers, and persisted state.

Recommended rules:

- treat peer messages, worker payloads, persisted queues, and snapshot exports as public client-side data unless the application adds its own protection
- do not persist bearer tokens, session secrets, CSRF secrets, or raw cookies in browser queues, caches, or storage-backed state
- require logout and user-switch flows to purge persisted data that could leak across principals on the same device
- put explicit retention windows on cached route data, dead-letter queues, and persisted snapshots instead of relying on browser eviction alone

The current helper layer already fails closed in several browser coordination cases, including stale popup or opener handles and unauthorized multi-client topic access, but operators still own the overall policy.

## Supply-Chain Review Practices

The current supply-chain posture should be conservative and reviewable.

Recommended practices:

- treat Go module dependencies, browser-test tooling, and asset build steps as separate trust boundaries
- prefer official toolchain artifacts for Go, `wasm_exec.js`, and browser-test runners
- document any build or post-processing step that rewrites wasm, JavaScript, or generated HTML before release
- review lockfile and dependency changes with the same care as framework code changes when they affect shipped assets or release pipelines
- keep release artifacts reproducible from the tagged source tree and documented build steps

For enterprise review, the important question is not only what code runs in production, but also what tools shaped the produced wasm, JavaScript, HTML, and release pipeline.

## Incident Response And Vulnerability Reporting

Security issues need a documented path, not just commit history.

Recommended workflow:

1. acknowledge the report and establish a private triage path
2. determine affected versions and blast radius
3. prepare the fix, focused validation, and release notes
4. publish the patched release and mitigation guidance
5. disclose the issue with enough detail for operators to assess their own exposure

When a bug touches auth boundaries, SSR output safety, secret handling, or supply-chain integrity, triage it separately from ordinary defects.

## Compliance-Oriented Deployment Guidance

GoWebComponents is not a compliance certification, but the current docs aim to make downstream review easier.

Operators remain responsible for:

- data retention and deletion policy
- consent and privacy policy
- encryption and key management
- access control and session policy
- audit logging requirements
- regulatory mapping to SOC 2, HIPAA, PCI, GDPR, or other frameworks

Recommended deployment posture:

- keep build and release pipelines reproducible enough to explain how shipped wasm, JavaScript, HTML, and manifest artifacts were produced
- separate development, staging, and production environments so they do not share secrets or logging sinks accidentally
- align audit logging with the application's own auth, mutation, and administrative boundaries instead of assuming framework lifecycle logs are sufficient evidence
- review CDN, reverse-proxy, browser-cache, and browser-storage behavior so public caches and local persistence do not outlive intended policy windows
- apply explicit retention and deletion policy to browser-visible logs, bootstrap payloads, stored drafts, caches, and exported snapshots when the domain requires it

## Current Boundary

This document describes the current framework boundary.

It does claim:

- typed and versioned bootstrap transport with decode-time validation
- explicit CSRF naming helpers and documented server-authoritative form patterns
- redaction-first guidance for logs, diagnostics, observability, and devtools snapshots
- clear operator ownership for secrets, auth, compliance mapping, and retention policy

It does not claim:

- turnkey auth, CSP, or secret-management infrastructure
- formal compliance templates or certification mappings
- complete security regression coverage for every integration surface

Those remain application or future backlog work.
