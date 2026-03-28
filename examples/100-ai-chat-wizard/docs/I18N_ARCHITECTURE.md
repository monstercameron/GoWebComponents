# Example 100 i18n Architecture

This document defines the target server-owned localization model for Example 100.

## 1) Target server-owned i18n architecture

### Server-owned responsibilities

- Canonical catalog storage for each supported locale.
- Namespace-scoped catalog assembly for boot-critical and route-specific copy.
- Catalog versioning metadata (`version`, `generated_at`, integrity hash) for cache control.
- Overlay resolution for environment/site, tenant, and experiment/campaign variants.
- Publish-time validation gates (placeholder parity, schema checks, prohibited term checks).

### Client-owned responsibilities

- Locale selection (browser hint, user preference, explicit picker state).
- Hydration and rendering from server-injected catalog payloads.
- Route-level namespace requests for non-boot namespaces after authenticated shell startup.
- Local cache management and invalidation using server catalog version metadata.

### Embedded emergency fallback responsibilities

- Minimal built-in English fallback for boot/auth safety copy only.
- Fallback is used only when server catalog fetch/bootstrap fails or payload is invalid.
- Fallback coverage should remain intentionally narrow and not become a second full catalog.

### Boot path changes in the target model

1. Server resolves locale and injects boot namespaces into initial shell payload.
2. Client renders first paint from injected catalog without waiting for post-hydration fetch.
3. Client validates version metadata and requests additional namespaces as routes expand.
4. Client switches locale only on explicit locale change (or locale mismatch correction), not by default post-hydration swap.

## 2) Namespace ownership map

| Namespace | Primary owner | Secondary owner | Where new keys should be added |
| --- | --- | --- | --- |
| `marketing` | Product marketing/content | Platform web team | Server catalog source for marketing locale pack; no new durable keys in `client/app/i18n.go`. |
| `auth` | Identity/auth product team | Platform web team | Server catalog source for auth namespace; keep client fallback to minimum boot/auth safety strings only. |
| `chat` | Core chat product team | AI runtime team | Server catalog source for chat namespace; add only emergency fallback strings in client for stream/error safety. |
| `settings` | Core chat product team | Account/preferences team | Server catalog source for settings namespace, with route-level namespace fetch after authenticated shell boot. |
| `billing` | Billing/revenue team | Customer operations | Server catalog source for billing namespace; enforce pricing vocabulary contract centrally. |
| `dashboard` | Admin/platform operations team | Security/compliance | Server catalog source for dashboard namespace, including mutation confirmation/preflight copy. |

Ownership rule:
- New durable keys must be added to the server-owned catalog pipeline by namespace owner.
- `client/app/i18n.go` remains fallback-only and must not be expanded with routine product copy.

## 3) Layered catalog precedence contract

Catalog assembly order is fixed from lowest to highest precedence:

1. app default
2. locale translation
3. environment/site override
4. tenant override
5. experiment/campaign override

Resolution rules:
- Highest-precedence non-empty value wins for each key.
- Missing override keys fall through to the next lower layer.
- Overrides must never delete required fallback keys; they can only replace values.
- Effective catalog payload must include version metadata that identifies both base locale version and applied override set.

## 4) Incremental migration plan from `client/app/i18n.go`

Goal: move durable copy to server catalogs route-by-route without breaking current rendering.

Phase sequence:
1. Boot-critical public/auth keys:
   - migrate `marketing` and `auth` keys required for `/`, `/home`, `/pricing`, `/signup` first paint.
   - keep matching fallback keys in `client/app/i18n.go` until first-paint parity is stable.
2. Authenticated shell keys:
   - migrate `chat` and common nav/status keys used by `/app` thread surfaces.
3. Account/settings keys:
   - migrate `settings` and `billing` namespaces with route-level lazy namespace fetch.
4. Dashboard keys:
   - migrate `dashboard` namespace once role-scoped route loading is stable.
5. Fallback reduction:
   - trim `client/app/i18n.go` down to emergency-only English strings after each namespace has stable server ownership.

Route-by-route rollout rule:
- Do not migrate all routes at once.
- For each route class, complete: key move -> boot/lazy load wiring -> browser regression -> fallback reduction.

## 5) First-paint requirements for boot-injected namespaces

First-paint contract:
- Boot payload must include all copy required for initial route render of `marketing` and `auth`.
- Client first paint must not show raw translation keys.
- Client first paint must not show an untranslated flash followed by immediate swap.
- Hydration may only change visible copy when locale actually changes, or when boot payload integrity/version validation fails and controlled fallback is required.

Operational check:
- Browser smoke should verify no visible double-swap on `/`, `/home`, `/pricing`, and `/signup` when locale is unchanged.

## 6) Cache and fallback contract

Catalog source selection rules:

1. Server-fresh catalog (preferred):
   - use when boot payload or namespace fetch succeeds and version is current.
2. Cached catalog (last-known-good):
   - use when fetch fails transiently and cache version matches current locale + compatible schema.
3. Built-in English fallback (emergency):
   - use only when server payload is missing/invalid and no valid cache is available.

Cache rules:
- Cache key includes locale + namespace + catalog version.
- Cache invalidates on locale change, catalog-version change, or schema version mismatch.
- Stale cache may be used only for bounded continuity; client must revalidate in background.

## 7) Placeholder and interpolation policy

Variable syntax:
- Use `{{variable_name}}` placeholders in source catalogs.

Policy rules:
- Placeholder names must be snake_case and stable across locales for the same key.
- Every localized variant of a key must contain exactly the same placeholder set as the source variant.
- Escaping/literal brace behavior must be explicitly supported by catalog tooling.

Validation gates:
- Publish/build must fail on placeholder mismatch across locales.
- Publish/build must fail on unknown placeholder names in runtime-bound keys.
- Publish/build must fail on malformed interpolation syntax.

## 8) Future extensibility note

This contract is intentionally structured to support later phases without breaking client rendering contracts:

- tenant-specific overrides
- white-label copy packs
- pricing-copy experiments and campaign overrides
- temporary incident/status banners
- eventual CMS/admin-managed catalog publishing flow
