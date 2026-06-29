# Accessible Radio Group

An interactive **WAI-ARIA radio group** with roving tabindex and focus-follows-selection, built by
composing existing primitives rather than reimplementing each control.

## What it shows
- `role="radiogroup"` / `role="radio"` with `aria-checked`, the contract assistive tech expects.
- Roving tabindex via `ui.UseCompositeNavigation` so the group is a single tab stop and arrow keys
  move selection.
- Focus-follows-selection using the `ui.UseDOMRef` element ref (`.Focus()`), so the active radio is
  always the focused one.

## Run
```
gwc dev examples/public/radiogroup
```
Tab into the group, then use the arrow keys — selection and focus move together.
