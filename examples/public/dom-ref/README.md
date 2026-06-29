# DOM Ref + Focus-on-Mount

Attaches a real DOM ref to an input and **focuses it on mount** — then unmounts it to prove the ref
detaches cleanly. No `getElementById`, no `autofocus` attribute.

## What it shows
- `ui.UseDOMRef()` + `shorthand.Ref(ref)` to capture the live element during the commit phase.
- `ui.UseAutoFocus(ref, when)` to focus the referenced element whenever it is (re)shown — on first
  mount and each time it is revealed.
- The ref releasing its element on unmount (toggle it off and back on), so no stale node is held.

## Run
```
gwc dev examples/public/dom-ref
```
Toggle the input off and on; it re-focuses on each reveal.
