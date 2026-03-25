# Single-Shell Routing Patterns

This document defines the recommended route and shell model for product-shaped GoWebComponents applications that serve public marketing routes and authenticated workspace routes from one browser runtime.

Use it when one app must own homepage, pricing, auth, and workspace flows without redirecting between separate HTML shells.

## At A Glance

- one `js/wasm` client shell should own public and private routes
- one router tree should host both marketing and workspace route families
- one shared shell layout should own navigation, metadata defaults, and global providers
- route-family shells should layer public and authenticated concerns without duplicate app roots
- code splitting should keep shell-entry cost stable as route count grows

## Single-Shell Runtime Pattern

The recommended pattern is one client runtime and one root route tree.

Core requirements:

- one top-level app shell mounts once and remains active across all route transitions
- auth state changes do not swap HTML documents or bootstrap a second wasm runtime
- public pages, auth pages, and workspace pages are route branches, not separate apps
- global providers (session, i18n, theme, telemetry, mutation queue, cache) are initialized once

### Reference Route Tree

```text
/
  (root-shell)
    /                  marketing home
    /features          marketing features
    /pricing           marketing pricing
    /contact           marketing contact
    /auth
      /signin
      /signup
      /reset
    /app               authenticated workspace shell
      /inbox
      /projects/:id
      /settings
```

### Layout Inheritance Rules

- root shell: global providers, top-level metadata defaults, cross-tab and offline hooks
- public shell: marketing header/footer and public-only route chrome
- auth shell: unauthenticated form layouts and redirect handling
- workspace shell: authenticated navigation, command surfaces, and workspace context

The auth boundary should sit at the route-branch level (`/app`) and be enforced with route guards, not separate server templates.

## Mixed Marketing And Workspace Conventions

Keep one router while separating concerns by route family.

### Naming And Grouping

- public routes: top-level, human-readable slugs (`/`, `/features`, `/pricing`)
- auth routes: explicit `/auth/*` family
- workspace routes: explicit `/app/*` family
- avoid sibling roots like `/dashboard` and `/marketing` that hide ownership boundaries

### Shared Versus Family Layouts

Shared across all routes:

- global providers and runtime boot
- analytics and diagnostics wiring
- cross-tab session or cache invalidation listeners

Public-only layers:

- campaign navigation
- public metadata defaults and social cards

Workspace-only layers:

- authenticated nav chrome and workspace-level loaders
- offline mutation replay banners and queue status

### Auth Boundary Placement

Use route-level guard boundaries:

- guard entry to `/app/*`
- preserve route intent so post-login redirects can restore target paths
- keep marketing routes always routable without auth context

## Canvas Workspace Routing Pattern

Canvas routes (editor, diagram, media, board) should live in the same authenticated shell as threaded and list workflows.

### Recommended Route Shape

```text
/app
  /documents/:docID
  /documents/:docID/canvas
  /boards/:boardID
  /threads/:threadID
```

### State Persistence Rules

Persist across route changes when returning to same resource scope:

- viewport position and zoom
- selected tool and panel state
- unsent local draft operations

Reset on route leave across resource identity:

- ephemeral selection tied to old entity IDs
- undo stack for unrelated resource IDs
- route-local autosave timers for the abandoned resource

### Back Button And Undo Semantics

- browser history should represent route transitions, not every canvas mutation
- undo and redo should stay in canvas-local history
- route param changes that identify a new resource should reset undo history intentionally

### Offline Mutation Queue Integration

Canvas autosave should write through the same mutation queue contract as form and entity updates:

- enqueue patch-like operations with stable entity and revision identifiers
- keep pending or conflict status visible in shell-level status UI
- replay on reconnect through the same conflict policy used by non-canvas mutations

## Progressive Route Expansion Pattern

Grow a shell in phases without increasing entry bundle cost linearly.

### Phase Model

1. authenticated workspace core (`/app/*`)
2. add marketing families (`/`, `/features`, `/pricing`)
3. add specialized route families (canvas, admin, ops)
4. add heavy optional surfaces behind lazy route boundaries

### Bundle Control Rules

- keep root shell and shared providers in base bundle
- split route-family entry modules (`marketing`, `workspace`, `admin`, `canvas`)
- lazy-load heavy widgets only inside their owning route families
- prefetch high-likelihood next routes from current journey context

### SEO And Prerender Considerations

For public marketing routes:

- prerender or SSR top marketing pages
- include canonical metadata and structured data through route metadata contracts
- emit deterministic route manifests for export tooling

For authenticated workspace routes:

- do not prerender user-private pages
- keep runtime-owned rendering and guarded redirects
- cache shell assets aggressively while keeping session data private

## Review Checklist

- does one wasm runtime own public and private routes
- is the auth boundary route-level rather than HTML-shell-level
- are marketing and workspace branches split cleanly under one router
- do canvas routes define persistence, reset, and undo boundaries
- does route expansion preserve shell-entry bundle size discipline
