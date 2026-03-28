# RelayDesk — Frontend Design Reference

## Aesthetic Direction: Precision Dark Luxury

> "The kind of interface that makes someone feel like they're operating something serious."

Think: Bloomberg Terminal precision meets Swiss instrument design meets bespoke fintech.
Dark, cold, technical — but completely refined. Electric, not garish.

The one thing someone will remember: **A site that feels like the AI is real** —
real model names, real latency numbers, real technical detail, wrapped in a UI so
precise it reads like a luxury instrument panel.

---

## Design Tokens

### Color Palette

| Token name            | Value                        | Role                                     |
|-----------------------|------------------------------|------------------------------------------|
| `bg-page`             | `#050508`                    | True near-black page background          |
| `bg-surface`          | `#0d0d12`                    | Slightly lifted surface (nav, footer)    |
| `bg-card`             | `#111118`                    | Card backgrounds                         |
| `border-default`      | `rgba(255,255,255,0.08)`     | Default card/section border              |
| `border-hover`        | `rgba(0,217,255,0.35)`       | Hover state border brightening           |
| `border-featured`     | `rgba(0,217,255,0.6)`        | Featured tier card border (Team plan)    |
| `accent-cyan`         | `#00d9ff`                    | Primary accent — electric, technical     |
| `accent-gold`         | `#d4a853`                    | Warm accent — featured/premium moments   |
| `accent-green`        | `#4ade80`                    | Mono detail — model names, status dots   |
| `text-heading`        | `#f0f0f8`                    | Headlines (slightly blue-tinted white)   |
| `text-body`           | `#8a8a9a`                    | Body copy                                |
| `text-muted`          | `rgba(255,255,255,0.35)`     | Tertiary / labels                        |
| `shadow-featured`     | `0 0 24px rgba(0,217,255,.15)` | Cyan glow for featured card             |

### Typography

| Role              | Font              | Weight   | Notes                              |
|-------------------|-------------------|----------|------------------------------------|
| Display headlines | **Syne**          | 700–800  | Extra-wide geometric, bold at size |
| Body / UI text    | **DM Sans**       | 400–500  | Neutral, refined, disappears well  |
| Numbers / mono    | **JetBrains Mono**| 400–500  | Model names, prices, latency stats |

**Source:** Google Fonts (single `<link>` in `bootstrap.go`)

**Anti-pattern:** No Inter, no Space Grotesk, no Roboto, no system-ui.

### Motion

| Effect              | CSS                                              | Use                              |
|---------------------|--------------------------------------------------|----------------------------------|
| Stagger reveal      | `translateY(24px)→0`, 600ms, 80ms per-word delay | Headline on page load            |
| Card hover          | `scale(1.01)` + border brighten                  | All feature/pricing cards        |
| Counter increment   | JS `requestAnimationFrame` from 0 to target      | Metric stats when scrolled in    |
| CTA magnetic        | `mousemove` 5px offset                           | Primary CTA buttons              |
| Progress bar        | Cyan `#00d9ff`, smooth width transition          | Auth loading shell               |

### Background Treatment (shared across all pages)

Replace all per-page copy-pasted orb `<div>`s with one shared function `renderPageBackground()`:

```
#050508 base
+ SVG hexagonal grid at 3% opacity (or circuit trace pattern)
+ single radial gradient fading to transparent at center
```

No blobs, no soft purple glows, no pink orbs.

---

## Component Architecture

### Current file structure (before refactor)

```
marketing_shared.go     — shared header/footer/hero primitives
landing_shell.go        — landing page shell + footer
landing_hero.go         — hero section + demo card
landing_sections.go     — product/why/pricing sections
pricing_shell.go        — full standalone pricing page
auth_shell.go           — auth loading + auth form
styles.go               — global CSS const
```

### Target component decomposition

Break rendering into three tiers: **tokens → atoms → molecules → shells**.

---

#### Tier 1 — Atoms (single-purpose, no layout)

These live in `marketing_shared.go` (or a new `marketing_atoms.go` if that file grows too large).

| Function                           | Signature sketch                                      | Purpose                                    |
|------------------------------------|-------------------------------------------------------|--------------------------------------------|
| `renderPageBackground()`           | `() → ui.Node`                                        | Single shared bg for every marketing page  |
| `renderBrandMark()`                | `() → ui.Node`                                        | The RD monogram badge                      |
| `renderHeroBadge(text string)`     | `(string) → ui.Node`                                  | Pill eyebrow label (e.g. "Moody · modern") |
| `renderSectionEyebrow(text)`       | `(string) → ui.Node`                                  | Uppercase tracking label above H2s         |
| `renderSectionDivider(label)`      | `(string) → ui.Node`                                  | Thin 1px ruled line with section label     |
| `renderCtaPrimary(label, href)`    | `(string, string) → ui.Node`                          | Cyan-fill CTA button                       |
| `renderCtaSecondary(label, href)`  | `(string, string) → ui.Node`                          | Glass/ghost CTA button                     |
| `renderNavLink(current, target, label)` | `(string, string, string) → ui.Node`             | Active-aware nav link                      |
| `renderFooterLink(label, href)`    | `(string, string) → ui.Node`                          | Single footer list item link               |
| `renderStatusDot(isLive bool)`     | `(bool) → ui.Node`                                    | Green pulsing dot for live status          |
| `renderMonoLabel(text)`            | `(string) → ui.Node`                                  | JetBrains Mono detail text                 |

---

#### Tier 2 — Molecules (composed from atoms, one job)

| Function                              | Args sketch                            | Purpose                                         |
|---------------------------------------|----------------------------------------|-------------------------------------------------|
| `renderMarketingHeader(path, nav, actions)` | `(string, ui.Node, ui.Node) → ui.Node` | Sticky header shell with brand + nav + actions |
| `renderMarketingFooter(columns)`      | `(...ui.Node) → ui.Node`               | Footer grid with brand blurb + link columns     |
| `renderFooterColumn(title, links)`    | `(string, ...ui.Node) → ui.Node`       | Single titled column in footer                  |
| `renderHeroHeading(eyebrow, h1, body, actions)` | `(string, string, string, ...ui.Node) → ui.Node` | Full hero copy block         |
| `renderHeroMetricStrip(metrics)`      | `([]heroMetric) → ui.Node`             | Horizontal strip of 3 mono metric+label cells   |
| `renderFeatureCard(eyebrow, title, body, accentClass)` | `...→ ui.Node`            | Single product feature card                     |
| `renderProofCard(stat, label, body)`  | `(string, string, string) → ui.Node`   | Proof/stat card in Why section                  |
| `renderPricingCard(plan, featured)`   | `(plan, bool) → ui.Node`               | Single pricing tier card                        |
| `renderCompareRow(label, vals)`       | `(string, []string) → ui.Node`         | Single row in pricing compare table             |
| `renderFAQItem(q, a)`                 | `(string, string) → ui.Node`           | Single FAQ question+answer card                 |
| `renderDemoMessageBubble(role, text)` | `(string, string) → ui.Node`           | AI or user message bubble in demo card          |
| `renderDemoMetricChip(label, value)`  | `(string, string) → ui.Node`           | Small chip inside demo card                     |
| `renderDemoComposer(placeholder)`     | `(string) → ui.Node`                   | Input row at bottom of demo card                |
| `renderDemoCard()`                    | `() → ui.Node`                         | Full mock chat card (composes above 3)          |
| `renderAuthFormCard(view, auth, isSignup)` | `...→ ui.Node`                    | Auth login/signup form card                     |
| `renderAuthStatCard(title, body)`     | `(string, string) → ui.Node`           | Stat card on auth page left column              |

---

#### Tier 3 — Shells (full page sections, compose molecules)

| Function                  | File                  | Composes                                              |
|---------------------------|-----------------------|-------------------------------------------------------|
| `renderLandingShell()`    | `landing_shell.go`    | background + header + main sections + footer          |
| `renderLandingHeroSection()` | `landing_hero.go`  | hero heading + metric strip + demo card               |
| `renderProductSection()`  | `landing_sections.go` | section header + feature cards grid                   |
| `renderWhySection()`      | `landing_sections.go` | pull-quote card + proof cards                         |
| `renderPricingSection()`  | `landing_sections.go` | section header + pricing cards                        |
| `renderPricingShell()`    | `pricing_shell.go`    | background + header + hero + plans + compare + FAQ + contact + footer |
| `renderAuthShell()`       | `auth_shell.go`       | background + header + hero copy + form card + footer  |
| `renderAuthLoadingShell()`| `auth_shell.go`       | loading card with cyan progress bar                   |

---

## Page-by-Page Design Specs

### All Marketing Pages — Shared

**Header:**
- `position: sticky`, `bg-[#050508]/80`, `border-b border-white/[0.05]`, `backdrop-blur-sm`
- Brand: RD monogram (cyan accent) + "RelayDesk" in Syne 600 + DM Sans subtitle
- Nav links: uppercase, `tracking-widest`, DM Sans 13px, `text-[#8a8a9a]` → `text-[#f0f0f8]` hover
- Primary CTA: `bg-[#00d9ff] text-black font-semibold` — the one cyan button that stands out

**Background:** `renderPageBackground()` — one function, used everywhere.

**Footer:** Horizontal rule + two-row minimal layout. Brand left, links right. No 4-column grid.

---

### Landing Hero — `landing_hero.go`

**Layout:** Asymmetric — headline bleeds left past the grid, demo card overlaps right with negative margin.

**Headline:** Syne 800, `text-8xl xl:text-9xl`, tight tracking `tracking-[-0.06em]`, warm white `#f0f0f8`

**Metric strip** (replaces "Clear / Calm / Flexible"):
```
< 800ms        GPT-4o · Claude 3.7 · Cerebras        99.9% uptime
──────────     ─────────────────────────────────      ──────────────
Response time  Available models                       Reliability
```
All in JetBrains Mono with a green `#4ade80` tint. Third stat has a pulsing green dot.

**Demo card:**
- Background: `#0d0d12`, border `1px solid rgba(255,255,255,.08)`
- Avatar badge: cyan `#00d9ff` with "RD" in black
- Model name shown in green mono: `gpt-4o · streaming`
- Send button: `↑` (`\u2191`) not the literal string "up" ← **fix immediately**
- "streaming..." indicator with animated blinking cursor `|`

---

### Product Section — `landing_sections.go`

**Cards:** Solid `#111118`, `1px solid rgba(255,255,255,.08)`, `border-radius: 12px`
On hover: border → `rgba(0,217,255,.25)`, `scale(1.01)` — engineered, not bubbly.
Drop the gradient glass treatment entirely.

**Left intro copy:** Variant-aware — currently hardcoded. Add switch on `parsePage`.

**Section divider** between Product and Why: `<hr>` styled as `1px solid rgba(255,255,255,.06)` with `CAPABILITIES` label centered in a `bg-[#050508]` pill.

---

### Why Section — `landing_sections.go`

**Left card:** Pull-quote style. Massive Syne italic:
```
  "You are not
   selling AI."
```
`text-6xl font-semibold italic` — the rest of the copy is secondary, smaller.

**Right proof cards:** Numbers only — `3×`, `40%`, `Day 1` — with DM Sans label below.
No more "Faster adoption / Higher trust" verb-body pattern.

---

### Pricing Section / Page

**Card hierarchy:**
- Starter: `#111118`, standard border, subdued
- Team (featured): `border border-[#00d9ff]/60`, `box-shadow: 0 0 24px rgba(0,217,255,.15)`, gold `#d4a853` badge
- Enterprise/Custom: `border-dashed border-white/20`, "Talk to sales" ghosted treatment

**Compare table** (expand from 4 → 8 rows):
```
Capability          Starter     Team        Enterprise
─────────────────────────────────────────────────────
Workspaces          1           Up to 5     Unlimited
Seats               1           Up to 15    Custom
Model access        GPT-4o      All models  All + priority
Shared workspace    —           ✓           ✓
Admin controls      Basic       Standard    Advanced
API access          —           —           ✓
Data retention      30 days     90 days     Custom
SLA / Support       Email       Priority    Dedicated
```

**FAQ** (replace current 3 soft questions with 5 objection-handlers):
1. "Where does my conversation data go?" — data handling, retention, deletion
2. "Which AI models power this?" — GPT-4o, Claude 3.7 Sonnet, Cerebras
3. "Can we self-host or deploy privately?" — Enterprise tier answer
4. "What happens if we go over our seat count?" — upgrade path, no surprise charges
5. "Can we cancel at any time?" — monthly billing, no lock-in

**Dead CTAs to fix:**
- "Book a sales call" → `mailto:sales@relaydesk.com` (or a real contact form)
- "Email the team" → `mailto:hello@relaydesk.com`
- Pricing header: "Open chat" + "Open app" both → `chatRouteRoot` — change first to "Log in" → `authLandingRoute`

---

### Auth Shell — `auth_shell.go`

**Background:** Same `renderPageBackground()` as marketing — remove the purple/pink orbs.

**Loading bar:** `#00d9ff` electric cyan (currently cyan-400/sky-300 mix — unify to design token).

**Form card:** `bg-[#111118]`, `border border-white/[0.08]`, no backdrop-blur — precise, not glassy.

---

## Content Gaps (trust signals, not just design)

For a credible LLM business these must be added to the landing/pricing pages:

1. **Data handling paragraph** — where prompts go, model provider data policies, retention
2. **Model transparency** — name the actual models (GPT-4o, Claude 3.7, Cerebras) visibly on landing
3. **Social proof** — even one quote or "trusted by" strip stops the "is this real?" question
4. **Free trial vs. paid CTA distinction** — "Start for free" must be visually distinct from "Buy now"
5. **Demo request path** — Custom/Enterprise tier needs a real contact form, not a link to the chat app

---

## Implementation Checklist

### Step 1 — Bootstrap (no WASM rebuild needed)
- [ ] `bootstrap.go`: Swap Space Grotesk → Syne + DM Sans + JetBrains Mono
- [ ] `styles.go`: Add `font-family` on body, add stagger/counter keyframes

### Step 2 — Quick fixes (smallest safe first)
- [ ] `landing_shell.go`: `(c)` → `\u00a9`
- [ ] `landing_hero.go`: `"up"` → `\u2191`
- [ ] `pricing_shell.go`: Deduplicate header CTAs

### Step 3 — Shared atoms + molecules
- [ ] Refactor `marketing_shared.go`: add `renderPageBackground()`, `renderSectionDivider()`, `renderCtaPrimary()`, `renderCtaSecondary()`, `renderStatusDot()`, `renderMonoLabel()`
- [ ] Refactor `renderMarketingHeaderShell` → `renderMarketingHeader` with sticky + blur
- [ ] Replace `renderMarketingHeaderAction` with `renderCtaPrimary` / `renderCtaSecondary`
- [ ] Slim down footer to minimal two-row layout

### Step 4 — Landing pages
- [ ] `landing_hero.go`: New asymmetric layout, Syne headline, metric strip, updated demo card
- [ ] `landing_sections.go`: New card style, variant-aware left copy, pull-quote Why section
- [ ] `landing_shell.go`: Wire in `renderPageBackground()`, new footer

### Step 5 — Pricing page
- [ ] `pricing_shell.go`: Expand compare table to 8 rows
- [ ] `pricing_shell.go`: Replace 3 FAQ items with 5 objection-handlers
- [ ] `pricing_shell.go`: Fix all dead CTAs to real mailto/form targets
- [ ] `pricing_shell.go`: Featured card cyan border + glow, Enterprise dashed

### Step 6 — Auth shell
- [ ] `auth_shell.go`: Replace orb background with `renderPageBackground()`
- [ ] `auth_shell.go`: Loading bar to `#00d9ff`

### Step 7 — Validate
- [ ] `go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard`
- [ ] Manual smoke: `/`, `/home`, `/capabilities`, `/pricing`, `/` (auth), loading shell
