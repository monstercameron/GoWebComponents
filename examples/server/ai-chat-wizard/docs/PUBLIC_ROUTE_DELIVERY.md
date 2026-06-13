# Public Route Delivery

Public-route delivery is the path from an unauthenticated browser request to a hydrated marketing or auth surface. It covers `GET /`, `/home`, `/pricing`, `/signup`, `/login`, and the public info/legal routes named in `client/app/routes.go`.

## Request Shape

```text
browser direct load
  |
  v
server/app/server.go shell handler
  |
  |-- returns the same app shell for public and app routes
  |-- serves /app/chat.wasm and worker assets from bin/client
  |-- leaves route interpretation to the WASM router
  v
client/main.go and client/app/app.go
```

The server does not render a separate marketing application. It returns shell HTML and boot assets, then the Go WASM app decides whether the current path is a public landing route, auth route, or authenticated app route. This keeps direct load, refresh, and SPA navigation on the same route contract.

## Boot Shell

The boot shell exists for the interval before the WASM app has resolved auth and runtime readiness. `shouldRenderLandingShellEarly` allows public landing routes to bypass the authenticated loading state when no session is present. `shouldRenderAuthLoadingShell` keeps unresolved auth from flashing the wrong private surface.

Public pages need stable first paint, so they use route-critical copy and shell structure that can render before user-specific RPC data exists. The shell must not depend on the gRPC tunnel being ready to show marketing, login, signup, pricing, or public legal pages.

## I18n Bootstrap

The current architecture is moving toward server-owned catalogs. The boot payload must include public/auth copy needed for the first route render, especially `marketing` and `auth`. Lazy namespace loading can handle non-critical authenticated surfaces later. The detailed namespace ownership and fallback rules live in [I18N Architecture](I18N_ARCHITECTURE.md).

The important public-route rule is: first paint must not show raw catalog keys or visibly swap core auth/marketing copy after hydration. That is why public-route regression coverage checks `/home`, `/pricing`, `/signup`, and `/` for stable boot-injected copy.

## Hydration And Auth Entry

```text
public path
  |
  |-- unauthenticated: render landing/auth mode
  |-- authenticated: allow app redirect or post-login route restore
  v
auth form
  |
  |-- Login or Signup RPC
  |-- token persisted in localStorage
  |-- GetSession validates role and workspace state
  v
/app or normalized post-login route
```

`client/app/auth.go` owns token storage, post-login route intent, and safe route normalization. A public CTA can store an intended app route, but only safe `/app` paths are restored after login. Admin/dashboard intents are gated by the role summary from `GetSession`.

The dedicated `/signup` route forces signup mode so direct links and CTA navigation do not rely on stale client state. `/login` and `/` remain auth-entry surfaces. Password reset and update-password shells exist, but the customer-facing reset/update submit path is documented elsewhere as not fully wired.

## Why The Public Side Is Shaped This Way

The public side is shell-first because Example 100 is testing one routed Go WASM app, not a split SSR marketing site plus a separate app bundle. That design makes every route transition exercise the same router, app state, auth bootstrap, and shell composition. The cost is that public first paint must be carefully bootstrapped. The benefit is that refreshes, callback returns, route guards, and authenticated handoff all use one source of truth.
