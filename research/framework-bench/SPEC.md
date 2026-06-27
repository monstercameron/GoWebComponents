# Bill Splitter — canonical app spec

This is the **single source of truth** every framework implementation must match. The
DOM structure, behavior, formulas, and visual design are identical across apps; only
the framework idioms differ. Reviewers compare implementations against *this* file.

The app is a tip/bill splitter chosen to exercise a broad framework surface while
staying small: local component state, shared/global state across components, derived
values, a keyed list, a presets list, conditionals, forms, and styling.

## State

| Name          | Kind                | Type            | Default   | Owner / scope                              |
|---------------|---------------------|-----------------|-----------|--------------------------------------------|
| `bill`        | local component     | number (USD)    | `0`       | root inputs                                |
| `tipPercent`  | local component     | number (%)      | `18`      | root inputs (set by presets or custom)     |
| `people`      | local component     | integer ≥ 1     | `1`       | root inputs (stepper)                      |
| `theme`       | **shared / global** | `"light"\|"dark"` | `"light"` | read by root + header toggle + footer      |
| `roundUp`     | **shared / global** | boolean         | `false`   | read by header toggle + results + footer   |

`theme` and `roundUp` MUST be implemented with the framework's **shared/global state**
primitive (atoms / stores / context / signals / services), not prop-drilled, so the
comparison exercises that surface.

## Tip presets

A list rendered by iteration (keyed): `[10, 15, 18, 20, 25]`. Each renders a button
labeled `"<n>%"`. The button is **active** when `tipPercent === n`. Clicking sets
`tipPercent = n`. A custom `%` number input sits after the presets and sets
`tipPercent` to its value (active styling falls off the presets when no preset matches).

## Derived values (pure, recomputed on render)

```
tipAmount            = bill * tipPercent / 100
total                = bill + tipAmount
perPersonRaw         = people > 0 ? total / people : 0
perPerson            = roundUp ? ceil(perPersonRaw) : perPersonRaw
totalCollected       = roundUp ? perPerson * people : total
roundingExtra        = max(0, totalCollected - total)
effectiveTipPercent  = bill > 0 ? (totalCollected - bill) / bill * 100 : tipPercent
```

`ceil` is the language's ceiling. `effectiveTipPercent` is shown rounded to 1 decimal.

## Behavior

- **Bill input**: numeric, min 0, step 0.01. Empty/invalid parses to `0`.
- **People stepper**: `−` and `+` buttons; `people` clamps to `≥ 1`. `−` is disabled at 1.
- **Custom tip**: numeric, min 0; setting it updates `tipPercent`.
- **Round up toggle** (header): flips `roundUp`. `aria-pressed` reflects state.
- **Theme toggle** (header): flips `theme` between light/dark. `aria-pressed` reflects dark.
- **Empty state**: when `bill <= 0`, results show `$0.00` and the empty hint is visible.
- **Rounding note**: shown only when `roundUp` AND `roundingExtra > 0`.

## Currency formatting

US dollars, 2 decimals, thousands separators, `$` prefix, e.g. `$1,234.56`.
Equivalent to `Intl.NumberFormat("en-US",{style:"currency",currency:"USD"})`.
Negative is not possible (inputs clamp to ≥ 0).

## DOM structure (canonical)

Semantic class names use the `bs-` prefix. Every implementation reproduces this tree
and these classes (GWC is the documented exception — see README: it generates its
classes via typed `css/u`, matching the *visual* design rather than the literal class
strings). `data-theme` on the root drives the theme.

```html
<main class="bs-app" data-theme="light">
  <header class="bs-header">
    <h1 class="bs-title">Bill Splitter</h1>
    <div class="bs-header-actions">
      <button class="bs-toggle" aria-pressed="false">Round up</button>
      <button class="bs-toggle" aria-pressed="false">Dark</button>
    </div>
  </header>

  <section class="bs-card bs-inputs">
    <label class="bs-field">
      <span class="bs-label">Bill amount</span>
      <div class="bs-input-wrap">
        <span class="bs-prefix">$</span>
        <input class="bs-input" type="number" min="0" step="0.01" />
      </div>
    </label>

    <div class="bs-field">
      <span class="bs-label">Tip</span>
      <div class="bs-presets">
        <button class="bs-preset bs-preset--active">18%</button>   <!-- one per preset -->
        <input class="bs-preset-custom" type="number" min="0" placeholder="Custom %" />
      </div>
    </div>

    <div class="bs-field">
      <span class="bs-label">People</span>
      <div class="bs-stepper">
        <button class="bs-step" aria-label="Fewer people">−</button>
        <span class="bs-count">1</span>
        <button class="bs-step" aria-label="More people">+</button>
      </div>
    </div>
  </section>

  <section class="bs-card bs-results">
    <div class="bs-result-row"><span>Tip</span><span>$0.00</span></div>
    <div class="bs-result-row"><span>Total</span><span>$0.00</span></div>
    <div class="bs-result-hero">
      <span class="bs-result-hero-label">Per person</span>
      <span class="bs-result-hero-value">$0.00</span>
    </div>
    <p class="bs-note">Rounding up collects $0.18 extra · effective tip 20.0%</p> <!-- conditional -->
    <p class="bs-empty">Enter a bill amount to begin.</p>                          <!-- conditional -->
  </section>

  <section class="bs-card bs-breakdown">
    <h2 class="bs-subtitle">Per-person breakdown</h2>
    <ul class="bs-people">
      <li class="bs-person"><span>Person 1</span><span>$0.00</span></li>           <!-- one per person -->
    </ul>
  </section>

  <footer class="bs-footer">Splitting $0.00 between 1 · light theme</footer>
</main>
```

The footer text: `Splitting <total> between <people> · <theme> theme`.

## Design tokens

Shared by `shared/styles.css` (non-GWC apps) and reproduced by GWC's `css/u` utilities.

| Token        | Light       | Dark        |
|--------------|-------------|-------------|
| bg           | `#f8fafc`   | `#0f172a`   |
| surface      | `#ffffff`   | `#1e293b`   |
| text         | `#0f172a`   | `#f1f5f9`   |
| muted        | `#64748b`   | `#94a3b8`   |
| border       | `#e2e8f0`   | `#334155`   |
| accent       | `#0ea5e9`   | `#38bdf8`   |
| accent-text  | `#ffffff`   | `#04293c`   |

Radii: card `16px`, control `10px`. Spacing scale: `4 / 8 / 12 / 16 / 24`.
Font: `system-ui, sans-serif`. Max content width `28rem`, centered.

## Comparison dimensions (filled in per app README)

Lines of code (impl only), explicit shared-state mechanism, reactivity model,
build toolchain, dependency count, dev-server command, and notes.
