# Example 100 Accessibility Checklist

This checklist is the release gate for public routes, auth, chat, settings, and dashboard surfaces. Run it after changes to `client/app/*shell*.go`, route files, form controls, modal/drawer code, dashboard slices, or styles.

## Keyboard Navigation

Scope:

- Public marketing routes: `/`, `/home`, `/pricing`, `/signup`.
- Auth routes: `/login`, `/signup`, reset flows when enabled.
- Authenticated shell: `/app`, `/app/thread/:publicID`, `/app/settings`, `/app/dashboard/*`.

Checklist:

- The first tab stop on public/auth pages is a skip link to the main content.
- Tab order follows visual order for nav, primary action, forms, shell sidebar, thread body, composer, settings sections, and dashboard filters.
- `:focus-visible` is visible against the current background on links, buttons, custom selects, toggles, sidebar rows, and table actions.
- Modal, drawer, tooltip, quote prompt, delete confirmation, speech upgrade, and settings overlays restore focus to the trigger after close.
- Escape closes overlays that can be dismissed, without trapping focus in the shell.
- No route change moves focus into a hidden sidebar, hidden canvas overlay, or closed settings panel.

Focused test seam:

- `client/app/app_shell_test.go` owns route/shell rendering invariants.
- `client/app/settings_route_test.go` owns settings route restoration.
- `client/app/sidebar_organization_wasm_test.go` owns sidebar scale/navigation grouping.
- Manual keyboard smoke is still required for real browser focus restoration.

## Landmarks And Headings

Expected outline:

- Public routes expose `header/nav/main/footer`, one page-level `h1`, and section headings in source order.
- Auth screens expose one `main` landmark, one `h1`, and form groups with visible headings.
- Chat shell exposes navigation/sidebar, a main chat region, a composer form region, and optional complementary canvas/settings/dashboard regions.
- Settings exposes a dialog title, section navigation, and a single active section heading.
- Dashboard routes expose a main dashboard heading plus one heading per slice: Business, Customers, Chats, Providers, Ops.

Regression check:

- Any new visible route must be added to the README source map and this outline.
- Any route-level heading added only for visual styling must still preserve heading order.

## Forms, Names, Help Text, And Errors

Audit these controls:

- Signup, login, and reset fields.
- Settings fields for display name, tone, prompt, thinking, speech, memories, locale, billing, and security.
- Sidebar search/filter controls and dashboard filters.
- Admin mutation confirmation forms.

Rules:

- Every input has a programmatic name from a label, `aria-label`, or labelled control text.
- Placeholder text is never the only accessible name.
- Helper text is visible or associated with `aria-describedby`.
- Errors are associated with the relevant field or group and also announced through a status region.
- Icon-only actions must have `aria-label`.
- Custom selects/toggles expose selected state through text, `aria-pressed`, `aria-selected`, or checked semantics.

## Live Regions And Status Messages

Use `ui.UseAnnouncer()` or a local `role="status"` region for:

- Streamed chat completion.
- Settings save success/failure, including late failures after the settings modal closes.
- Auth submit outcomes.
- Dashboard loading, denied, empty, stale, and error states.
- Admin mutation queued/succeeded/failed states.
- Bridge states: reconnecting, degraded, sleeping, and offline.

Do not announce every stream chunk. Announce meaningful transitions: started, completed, failed, reconnected, saved, denied.

## Reduced Motion

Current code seam:

- `renderAppShell` uses `ui.UsePrefersReducedMotion()`.
- `styles.go` and `styles_motion.go` disable spring/keyframe motion under `prefers-reduced-motion` and `.reduce-motion`.
- `settings_route.go` should avoid smooth scroll when reduced motion is expanded to that helper.

Checklist:

- Route transitions, modal entry, overlay fade, hover treatments, scroll affordances, dashboard transitions, and dev-tour overlays remain understandable with motion disabled.
- State changes do not rely on movement alone; a label, shape, icon, or text state remains.

## Contrast And Non-Color State

Audit targets:

- Primary/secondary/destructive buttons.
- Status chips and bridge banners.
- Dashboard charts, KPI tiles, tables, drill-down rows, and disabled states.
- Form errors and alert banners.

Rules:

- Text contrast meets WCAG AA for normal and large text.
- Focus rings meet contrast against the component and page background.
- Error, warning, success, disabled, active, and selected states include text, shape, border, icon, or state attributes beyond hue.
- Dashboard charts must include labels or table equivalents for values that are otherwise color-coded.

## Screen-Reader Smoke Path

Run with NVDA, VoiceOver, or Narrator:

1. Login: navigate to `/login`, submit valid demo credentials, confirm auth outcome is announced and focus lands in the authenticated shell without losing route context.
2. First message: from `/app`, send one prompt, confirm stream start/completion is understandable and the composer remains reachable.
3. Settings save: open settings, change display name/tone/prompt/thinking/memory/locale, save, confirm the modal closes and late errors announce through the toast/status region.
4. Admin mutation path: as a seeded admin/superuser, open one dashboard mutation preview, confirm/deny, and confirm status plus support/request metadata are reachable.

The browser smoke is required because static tests cannot prove assistive-tech focus movement or live-region timing.
