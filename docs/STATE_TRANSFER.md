# State Transfer

This page records the current server-to-client state transfer contract for `ui.SSRBootstrap`.

Use it when deciding what belongs in bootstrap, how transferred values merge with client state, and which parts of that payload become runtime-owned during hydration.

## Current Bootstrap Surface

`ui.SSRBootstrap` currently exposes five buckets:

- `Route`: path, query, and params for the initial route
- `Atoms`: atom snapshot values restored into runtime state before hydration
- `Data`: application-owned public bootstrap payloads such as cache seeds or other resumable hints
- `I18n`: locale, direction, fallback locale, and the initial message subset
- `IDSeed`: the initial `ui.UseId()` seed for hydration-safe ID continuity

The transport formats already shipped are:

- inline JSON through `RenderBootstrapScript(...)`
- sidecar JSON through `SSRBootstrapReference{Format: "json"}`
- sidecar CBOR through `SSRBootstrapReference{Format: "cbor"}`

## State Classification Guidance

Treat transferred values as one of four classes:

- Public bootstrap state:
  Route params, locale, text direction, non-secret route metadata, and other values the server is already comfortable rendering into HTML.
- Runtime-owned resumable state:
  Atom snapshots and ID seeds that the client runtime actively restores before hydration and then owns afterward.
- Cache or feature seeds:
  Public `Data` payloads such as `fetchCache` entries or other app-owned warm-start hints that a package or application may import into its own client registry.
- Server-only secrets:
  Session tokens, internal policy results, private API credentials, CSRF secrets, or any value that should never appear in page HTML or bootstrap sidecars.

Rules:

- if a value is secret on the server, it does not belong in `SSRBootstrap`
- if a value is needed only to shape HTML already emitted, prefer rendering the result instead of transferring the source secret
- if a value must be resumed as live client state, put it in the runtime-owned bucket instead of inventing ad hoc globals
- if a value is only a warm-start hint, keep it in `Data` and document the owning package that consumes it
- prefer compact, JSON-shaped values even when CBOR is used as the transport

See [SECURITY.md](SECURITY.md) for the broader server-only versus client-safe policy around that rule.

## Merge Semantics

Hydration should treat each bootstrap bucket differently instead of using one blanket merge rule.

Current merge contract:

- `Route`
  The transferred route is an immutable description of the server-rendered entry route. It should align the first client resume, but later client navigation fully replaces it.
- `Atoms`
  Atom snapshots merge by key into the runtime registry before hydration. Bootstrap-provided atom keys overwrite the runtime's default atom values for that first resume.
- `Data`
  `Data` is application-owned and package-owned by convention. Consumers should merge by namespaced key, not by blindly flattening every nested field into one global object.
- `I18n`
  The bootstrap locale payload replaces the initial client locale seed for first resume, after which normal client locale changes own the active bundle.
- `IDSeed`
  The transferred seed replaces the runtime's current seed for the hydration pass so client-generated IDs continue from the server-rendered sequence.

Recommended app-level rule for `Data`:

- merge at the top-level key boundary
- let each key owner define nested merge behavior for its own payload
- avoid one generic deep-merge policy across unrelated bootstrap keys

For example:

- `fetchCache` should be consumed by the shared-cache bootstrap restore path
- `featureFlags` should be treated as immutable startup hints unless the app has an explicit live refresh model
- `themeHint` should seed first render but not override a fresher persisted preference without an explicit policy

## Ownership During Hydration

Ownership answers who controls a value after the client resumes.

Current ownership model:

- runtime-owned after restore:
  `Atoms`, `IDSeed`
- application-owned but hydration-aligned:
  `Route`, `I18n`
- hint-owned until consumed:
  `Data`

More specifically:

- atom snapshots become runtime state immediately through `RestoreAtomSnapshot(...)`
- the ID seed becomes runtime-owned immediately through `SetIDSeed(...)`
- route bootstrap is an initial routing hint for the first hydrated screen, not a permanent lock on future navigation
- i18n bootstrap is the first locale seed; later client locale changes may replace it normally
- `Data` does nothing by itself until a package or app consumes it, so ownership stays with that consumer

## Recommended Transport Choices

- inline JSON:
  Best for small, public payloads needed immediately at first paint
- sidecar JSON:
  Best when inline script size starts to bloat HTML but JSON readability is still useful
- sidecar CBOR:
  Best when the payload is still public but larger and strongly benefits from a denser binary transport

Practical guidance:

- keep inline payloads intentionally small
- move bulk cache seeds or large message bundles into sidecars before they silently dominate HTML size
- treat transport choice as a delivery concern only; it does not change the ownership or secrecy rules above

## Non-Goals For The Current Contract

- secret transfer channels inside bootstrap
- generic deep-merge semantics for every bootstrap subfield
- client authority over values that should remain server-only
- streaming-state protocols mixed into the current hydration bootstrap shape
