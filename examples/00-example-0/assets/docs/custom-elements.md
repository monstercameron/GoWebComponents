# Custom Elements

This page defines the supported custom-element and web-component model for GoWebComponents applications.

Use it when you need to consume browser-defined custom elements from `html` trees without dropping into ad hoc `syscall/js`.

## At A Glance

There are three distinct cases in this area:

- use `html.Tag(...)` when an unknown element only needs plain attributes and normal children
- use `html.CustomElement(...)` when the element needs explicit property assignment, reflected attributes, presence flags, slots, or custom-event integration
- use the export prototype pattern only when experimenting with wrapping a GoWebComponents subtree inside a browser custom-element class

The stable story today is consumption of browser-defined custom elements. Exporting GoWebComponents components as standards-based custom elements is still experimental.

## Quick Decision Guide

Choose the integration path by ownership:

- browser element already exists and only needs markup: prefer `html.Tag(...)`
- browser element already exists and has a documented property or event contract: prefer `html.CustomElement(...)`
- GoWebComponents should render inside a wrapper-owned host or shadow root: use `ui.RenderInto(...)` through the experimental export prototype

That keeps the framework boundary clear: stable consumption first, experimental export second.

## Supported Model

The supported public goal today is consuming browser-defined custom elements from GoWebComponents.

- Render hosts with `html.CustomElement(...)` when the element needs explicit property assignment, reflected attributes, or slotted children.
- Keep using `html.Tag(...)` for unknown tags that only need plain attributes and children.
- Use `interop.GetDocument()` plus `interop.Element` or `interop.EventTarget` when you need an imperative handle after mount.

Exporting GoWebComponents components as standards-based custom elements is still experimental.

The current repo only ships a prototype pattern, not a stable package-level export API:

- use `ui.RenderInto(...)` when a wrapper already owns the host or shadow-root target node
- keep the browser custom-element class itself in JS for now
- treat attribute observation, disconnect cleanup, and shadow-root setup as wrapper-owned concerns until a first-class API exists

## Mapping Rules

`html.CustomElementProps` separates the channels deliberately:

- `Props`: standard HTML metadata already supported by `html.Props`, such as `ID`, `Class`, `Data`, `Aria`, `Role`, and common built-in event handlers.
- `Attributes`: reflected string attributes that should exist in both SSR markup and the live browser DOM.
- `Presence`: boolean presence attributes for cases where a custom element watches `hasAttribute(...)` or observed attributes instead of properties.
- `Properties`: client-side property assignment for numbers, objects, slices, maps, callbacks, opaque JS-wrapped values, or any payload that should not be stringified into SSR markup.

Practical rule of thumb:

- strings can live in either `Attributes` or `Properties`
- booleans that are meant to be presence flags belong in `Presence`
- numbers, structs, slices, maps, callbacks, and host objects belong in `Properties`
- if the third-party element documents an observed attribute contract, use `Attributes` or `Presence`; otherwise prefer `Properties`

## Slots And Children

Pass light-DOM children normally.

To target a named slot, set `html.Props{Slot: "name"}` on the child node:

```go
html.CustomElement("demo-card", html.CustomElementProps{}, 
    html.Div(html.Props{Slot: "summary"}, html.Text("Queued for review")),
)
```

This keeps slot wiring in normal tree composition instead of inventing a second child API.

## Custom Events

Use the `interop` event surface for custom events emitted by web components.

- `element.Events()` returns an `interop.EventTarget`
- `target.Subscribe("rating-change", ...)` gives raw `interop.CustomEvent`
- `interop.DecodeCustomEvent[T](...)` projects `detail` into a typed Go struct
- `interop.SubscribeDecoded[T](...)` handles subscription plus typed detail decode in one step

Event `detail` values are expected to stay JSON-shaped if they need typed projection back into Go.

## SSR And Hydration

Custom-element hosts are rendered as inert tags on the server.

- reflected values from `Props`, `Attributes`, and `Presence` are serialized into SSR markup
- `Properties` are intentionally client-only and are not emitted into HTML
- during client render or hydration, property values are applied to the upgraded element host after the DOM node exists

Do not rely on SSR to transfer object properties, callback references, or large structured widget config. If the server must communicate an initial value, reflect it through an attribute or embed page bootstrap data separately and reconstruct the property on the client.

Hydration rule:

- treat the custom element as the owner of its own internal upgraded state
- use reflected attributes for stable server-visible state
- use property assignment for client-owned state that can safely be re-applied after hydration

## Export Prototype

`examples/89-exported-custom-element` demonstrates the current exploratory pattern for exposing a GoWebComponents subtree as a standards-based custom element:

1. A browser custom-element class owns `connectedCallback`, `attributeChangedCallback`, `disconnectedCallback`, and optional `attachShadow(...)`.
2. That wrapper calls into Go through a narrow mount bridge.
3. Go renders into the wrapper-owned root with `ui.RenderInto(...)`.
4. Disconnect cleanup renders an empty tree back into that same root.

This proves the runtime can mount into explicit DOM nodes cleanly, but it does not yet commit the framework to one permanent exported-custom-element API shape.

## Style And Shadow-DOM Expectations

Current expectation:

- consuming third-party custom elements should assume the custom element owns its own internal style model
- exported GoWebComponents prototypes should default to shadow DOM when style isolation is required
- light-DOM export is still possible, but then external CSS ownership and selector collisions become the caller's responsibility
- wrapper-owned shadow roots should carry the component-local `<style>` subtree or another explicit style injection path instead of assuming global stylesheet inheritance

Practical guidance:

- choose shadow DOM when the exported widget should be embeddable in arbitrary host pages without leaking or inheriting layout styles
- choose light DOM only when the embedding surface explicitly wants shared CSS and accepts the coupling
- do not assume one exported component can transparently support both modes without documenting differences in slot behavior, typography inheritance, and theming hooks

## Example

See `examples/88-web-components` for consuming a browser-defined custom element from Go code, and `examples/89-exported-custom-element` for the current export-side prototype into a shadow-root host.
