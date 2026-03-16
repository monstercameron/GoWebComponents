# Nested Routes Demo

This example demonstrates the new layout-route and outlet API with a small multi-level app.

Open:

`/examples/19-nested-routes/nested-routes.html`

Routes included:

- `/` landing page
- `/dashboard/overview`
- `/dashboard/reports/:id`
- `/dashboard/settings/profile`
- `/dashboard/settings/team`
- `/docs/getting-started`
- `/docs/routing`

What it shows:

- a dashboard shell kept alive across nested dashboard pages
- a second nested settings shell under the dashboard route tree
- a docs section with its own persistent layout and child outlet
- explicit child rendering through `router.Outlet()`
- param access inside nested leaf routes through `router.UseParams()`
