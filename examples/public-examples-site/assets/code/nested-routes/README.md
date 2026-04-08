# Nested Routes Demo

This example demonstrates the new layout-route and outlet API with a small multi-level app.

## Current Status

This example is the focused integrated reference for nested layout routes in the browser-router examples set.

Use it when you want to see how parent layout shells stay mounted while child outlet content changes, including a deeper nested settings layout and a separate docs layout tree.

Serve examples from the repo root with:

```powershell
go run ./tools/gwc examples
```

Then open:

`/examples/public/nested-routes/nested-routes.html`

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
- explicit child rendering through `router.GetOutlet()`
- param access inside nested leaf routes through `router.UseParams()`
