# 20 Portals

This example demonstrates the public `ui.Portal(...)` API.

## Current Status

This is the focused integrated portal example in the numbered examples set.

Use it when you want to inspect how modal, tooltip, and popover surfaces can render into a dedicated DOM host outside `#app` while the logical controls remain in the normal component tree.

Serve examples from the repo root with:

```powershell
go run ./tools/gwc examples
```

Then open:

`/examples/20-portals/portals.html`

It renders:

- a modal dialog
- a tooltip surface
- a popover card

All three overlays mount into `#portal-root` instead of the logical `#app` tree.

## What It Shows

- selector-targeted portal mounting through `ui.PortalProps{Target: ui.PortalTarget{Selector: "#portal-root"}}`
- overlay cleanup when the modal, tooltip, or popover closes
- separation between logical component ownership and physical DOM placement
