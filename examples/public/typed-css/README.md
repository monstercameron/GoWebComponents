# Typed CSS

A click counter styled entirely with the **typed CSS** layer (`css` + `css/u`) — type-checked,
content-hashed, zero-runtime-string-soup styling authored right next to the markup.

The view is written **bare**: it dot-imports `css/u` for utilities and `html/shorthand` for elements,
so there are no `css.`/`u.` qualifiers anywhere. Each style folds to a deterministic, content-hashed
class; identical rules dedupe to one class and the stylesheet is injected once.

## What it shows
- Composing styles with the Tailwind-shaped utility layer (`Flex`, `Pad`, `Rounded`, `Hover(...)`,
  `Transition(...)`) on top of the raw typed-CSS primitives.
- `css.Class(...)` as the clsx-style entry point — mixing literal strings, typed rules, variant
  slices, and pre-folded sheets in one call.
- Type-safe values (`Hex`, `RGBA`, `Spacing`, `Rem`) instead of free-form strings, so a typo is a
  compile error rather than a silently-broken rule.
- `css.SeedFromDocument()` on boot so server-rendered rules are adopted (not re-injected) on hydration.

## Run
```
gwc dev examples/public/typed-css
```
The emitted CSS is hardened against `</style>` breakout and the class names are stable across builds,
so the same source always produces the same stylesheet.
