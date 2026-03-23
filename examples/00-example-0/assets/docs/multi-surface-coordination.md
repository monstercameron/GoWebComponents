# Multi-Surface Coordination

This page documents the current first-class model for coordinating multiple active browser surfaces in GoWebComponents.

Use it when an application needs a popup or secondary window to cooperate with the main app without hand-rolling `window.open` and `postMessage` plumbing.

## At A Glance

- the current surface is a typed same-origin window-messaging layer over `postMessage`
- it is designed for opener and popup workflows, not general browser-wide peer discovery
- the main helpers are `OpenSecondaryWindowChannel(...)`, `WindowOpenerChannel(...)`, typed window-envelope decoding, and `SurfaceSignal` convenience helpers
- ownership stays explicit: one surface owns lifecycle and canonical writes, the other acts as a specialized collaborator

## Quick API Chooser

- use `interop.OpenSecondaryWindowChannel(...)` in the opener when the current app launches and owns the popup lifecycle
- use `interop.WindowOpenerChannel(...)` inside the popup when it needs to talk back to its opener
- use `interop.SubscribeDecodedWindow[T](...)` when the workflow has its own typed payloads
- use `interop.SubscribeSurfaceSignals(...)` plus `PublishSessionExpired(...)`, `PublishRouteFocus(...)`, `PublishSelection(...)`, or `PublishIntent(...)` when the payload is one of the common session or surface-coordination cases
- use cross-tab channels instead of window channels when there is no direct opener or popup relationship

## Example Shape

This is the current intended flow: the opener creates the child window, both sides subscribe to typed surface signals, and collaboration happens through explicit route, selection, or intent messages rather than silent shared-state mutation.

```go
package shell

import "github.com/atdiar/particleui/interop"

func openInspector() (func(), error) {
    channel, err := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
        URL:  "./multi-window-console-popup.html",
        Name: "example:multi-window:ops",
    })
    if err != nil {
        return nil, err
    }

    subscription, err := interop.SubscribeSurfaceSignals(channel, func(message interop.DecodedWindowEnvelope[interop.SurfaceSignal], err error) {
        if err != nil {
            return
        }
        _ = message
    })
    if err != nil {
        _ = channel.Close()
        return nil, err
    }

    if err := interop.PublishRouteFocus(channel, "/orders/42", "tab=activity", "order-heading"); err != nil {
        _ = subscription.Cancel()
        _ = channel.Close()
        return nil, err
    }

    return func() {
        _ = subscription.Cancel()
        _ = channel.Close()
    }, nil
}
```

## Current Shipped Surface

The public surface today is:

- `interop.OpenSecondaryWindowChannel(options)`
- `interop.WindowOpenerChannel(options)`
- `interop.WindowChannel.Publish(payload)`
- `interop.WindowChannel.Subscribe(handler)`
- `interop.WindowChannel.Focus()`
- `interop.WindowChannel.Close()`
- `interop.WindowChannel.Closed()`
- `interop.WindowChannel.TargetOrigin()`
- `interop.DecodeWindowEnvelope[T](message)`
- `interop.SubscribeDecodedWindow[T](channel, handler)`
- `interop.SurfaceSignal`
- `interop.SubscribeSurfaceSignals(channel, handler)`
- `interop.PublishLogout(channel, reason)`
- `interop.PublishSessionExpired(channel, reason, returnTo, expiresAt)`
- `interop.PublishRouteFocus(channel, path, query, focusID)`
- `interop.PublishSelection(channel, scope, id, revision)`
- `interop.PublishIntent(channel, action, target, params)`

## Coordination Model

The current first-class coordination model is intentionally narrower than a general multi-surface runtime.

Shipped scope today:

- same-origin popup or secondary-window workflows opened from the current app
- opener-to-popup and popup-to-opener typed message exchange
- explicit focus and close control for opener-owned secondary windows
- cross-tab peer synchronization through [`docs/CROSS_TAB.md`](cross-tab-synchronization.md)

Not yet first-class:

- embedded iframe coordination
- browser side panels or extension surfaces
- external native control surfaces
- general parent/child window routing integration
- fully automatic session and route propagation across every surface

The distinction from cross-tab synchronization matters:

- cross-tab sync is peer-to-peer state hint broadcasting with no direct window handle
- multi-surface coordination is an explicit parent/child relationship with a concrete opened window, focus control, and targeted `postMessage` exchange

## Current Recommended Flow

The smallest reliable workflow today is:

1. opener creates a named window channel
2. popup binds back to the opener using the same channel name
3. both sides subscribe before relying on the connection
4. opener publishes route, selection, session, or intent signals as needed
5. both sides treat close, orphaning, and publish failure as normal operational states

## Popup And Secondary-Window Channels

Open a secondary window from the main app:

```go
popup, err := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
    URL:  "/inspector.html",
    Name: "inspector",
})
```

Inside the popup, talk back to the opener:

```go
opener, err := interop.WindowOpenerChannel(interop.WindowChannelOptions{
    Name: "inspector",
})
```

Both sides then use:

- `Publish(payload)`
- `Subscribe(...)`
- `SubscribeDecodedWindow[T](...)`

Messages are JSON-shaped and wrapped in a `WindowEnvelope` with:

- `Name`
- `Payload`
- `Source`
- `SentAt`

`TargetOrigin` defaults to the current window origin when not provided explicitly. That keeps the default path same-origin and avoids silently widening message delivery.

For the most common opener or popup coordination cases, `interop` now also ships a typed `SurfaceSignal` payload with dedicated helpers for:

- logout propagation
- session-expiry hints with optional `returnTo`
- route focus handoff
- active-document selection
- window-intent actions such as focus or close requests

Those helpers are thin wrappers over the same `WindowChannel` transport. They are convenience for common multi-surface payloads, not a second runtime.

## Ownership And Synchronization Rules

The current ownership model is:

- the opener is authoritative for popup lifecycle
- the popup is a specialized view or control surface, not the canonical owner of shared domain state
- shared state changes should still flow back to one authoritative store, loader, cache key, or server write path

Recommended rules:

- let the opener own close or focus control for the popup
- let the popup send intents, selections, and diagnostics rather than silently mutating canonical state in isolation
- use cross-tab channels for peer tab invalidation, and window channels for targeted parent/child interaction
- if both surfaces can edit the same logical entity, carry explicit revision metadata and reject stale writes instead of blindly applying the latest arriving message
- prefer invalidation or intent messages when ownership is ambiguous

This is the current answer to multi-surface synchronization rules: one surface should remain authoritative, and the others should act as mirrored or specialized collaborators.

## Teardown And Orphaned Surfaces

Popup loss and opener loss are normal operational states.

The current guidance is:

- if a popup closes unexpectedly, the opener should detect `WindowChannel.Closed()` and treat the popup as disconnected rather than as a fatal app error
- if the opener disappears, the popup becomes orphaned and should switch into a degraded but explicit state instead of assuming the coordinator still exists
- orphaned surfaces should stop assuming remote writes or focus actions will succeed
- reconnect by opening a new popup or re-establishing the opener relationship, not by pretending the old channel is still valid
- if a surface owns only auxiliary UI, it is reasonable to close it when the authoritative opener goes away; if it still has operator value, keep it visible but clearly marked orphaned

The runtime does not auto-remount, auto-close, or auto-transfer ownership between windows. Applications are expected to poll `Closed()` or handle publish failures and then update their own UI state intentionally.

## Recommended Usage Today

- keep popup coordination same-origin unless there is a deliberate reason not to
- use one named window channel per workflow such as `inspector`, `preview`, or `operator-console`
- keep payloads JSON-shaped and typed through `SubscribeDecodedWindow[T](...)`
- prefer `SubscribeSurfaceSignals(...)` and the dedicated publish helpers when the payload is one of the common shared session, route, selection, or intent workflows
- close popup channels deliberately when the owning surface no longer needs them
- treat orphaned or closed child windows as normal operational state, not as exceptional control flow

See `examples/95-multi-window-console` for the current popup-inspector style reference flow built on this surface.

## Review Checklist

- does one surface clearly own lifecycle, focus, and canonical writes
- are same-origin and `TargetOrigin` rules explicit instead of implicit
- do popup and opener UIs handle `Closed()` and publish failure as normal degraded states
- are common route, session, selection, and intent cases using the typed signal helpers instead of ad hoc string payloads
- is cross-tab synchronization kept separate from opener or popup coordination when no direct window handle exists
