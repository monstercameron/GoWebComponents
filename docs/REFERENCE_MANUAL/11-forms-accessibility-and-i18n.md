# 11 Forms Accessibility And I18n

Use this chapter when you are building typed forms, mapping authoritative server validation back into UI state, wiring focus and announcements, or making one app work across multiple locales.

It is the right chapter for:

- `ui.UseForm[T]` field state, validation, submit lifecycle, submit intents, and structured server-error projection
- CSRF naming helpers through `ui.NewCSRFToken(...)`
- accessibility helpers such as `ui.UseAnnouncer()`, `ui.UseFocusManager()`, `ui.UseFocusTrap(...)`, and `ui.AccessibleOverlay(...)`
- locale state through `i18n.UseLocale(...)`, runtime lookup through `i18n.Provider(...)` and `i18n.UseI18n()`, and deterministic catalog lookup through `i18n.Bundle`
- locale-aware routing helpers through `i18n.PrefixPath(...)` and `i18n.ResolvePath(...)`

Use another chapter instead when:

- you need fetch transport, resource lifecycle, or mutation replay first: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need router registration and guards first: go to [08 Routing](08-routing.md)
- you need SSR bootstrap transport in depth: go to [09 SSR And Hydration](09-ssr-and-hydration.md)
- you need deployment, service workers, or installability first: go to [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## Overview

This chapter works best if you keep three ownership boundaries separate:

- `ui.UseForm[T]` owns local field values, touched and dirty state, validation state, pending state, and intent-aware submit state
- the server owns canonical auth, CSRF validation, business-rule validation, redirects, and authoritative success or failure
- accessibility and locale behavior remain explicit application work, with framework helpers for the repetitive wiring

That separation avoids three common mistakes:

- treating client validation as the mutation authority
- assuming accessibility semantics happen automatically just because the app uses a component framework
- collapsing locale state, translation lookup, and route policy into one opaque global

## Stability Note

The main forms, accessibility, and i18n entrypoints are `Stable`:

- `ui.UseForm(...)`, `FieldStatus(...)`, `FieldMessage(...)`, `Validate(...)`, `ValidateAsync(...)`, `Submit(...)`, `SubmitWithIntent(...)`, `ApplyServerErrors(...)`, and `ApplyServerActionResult(...)`
- `ui.NewCSRFToken(...)`, `ui.DefaultCSRFHeaderName`, and `ui.DefaultCSRFFormFieldName`
- `ui.UseFocusManager()`, `ui.UseFocusTrap(...)`, `ui.UseAnnouncer()`, and `ui.AccessibleOverlay(...)`
- `i18n.NewBundle(...)`, `Bundle.Register(...)`, `Bundle.RegisterNamespace(...)`, `i18n.NewLazyBundle(...)` / `LazyBundle.EnsureLocale(...)` (lazy locale loading), `i18n.UseLocale(...)`, `i18n.Provider(...)`, `i18n.UseI18n()`, `i18n.PrefixPath(...)`, `i18n.ResolvePath(...)`, and `i18n.BundleFromSSRBootstrap(...)`

Important advanced boundaries:

- the typed server-action result envelope is public, but the transport and endpoint remain application-owned
- `ui.Overlay(...)` and `ui.UseOverlayStack(...)` are the broader overlay layering APIs when a modal-focused `AccessibleOverlay(...)` is no longer enough
- older docs may still mention `ui.ExtractFiles(...)`; the current exported helper is `ui.GetFiles(...)`

## Minimal Example

Start with a small typed form that validates locally, announces failures, and moves focus to the first invalid control.

```go
package main

import (
	"fmt"
	"strings"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type reviewForm struct {
	Name    string
	Email   string
	Channel string
}

// validateReviewForm returns field-keyed errors for one accessibility review request.
func validateReviewForm(getValue reviewForm) ui.FieldErrors {
	getErrors := ui.FieldErrors{}
	if len(strings.TrimSpace(getValue.Name)) < 2 {
		getErrors["Name"] = "Name must be at least 2 characters."
	}
	if !strings.Contains(getValue.Email, "@") {
		getErrors["Email"] = "Email must include @."
	}
	if strings.TrimSpace(getValue.Channel) == "" {
		getErrors["Channel"] = "Choose a notification channel."
	}
	return getErrors
}

// renderReviewRequestForm keeps validation, announcements, and focus-to-error behavior in one feature owner.
func renderReviewRequestForm() ui.Node {
	getForm := ui.UseForm(reviewForm{})
	getFocus := ui.UseFocusManager()
	getAnnouncer := ui.UseAnnouncer()

	getNameID := ui.UseId() + "-name"
	getEmailID := ui.UseId() + "-email"
	getChannelID := ui.UseId() + "-channel"
	getFieldIDs := map[string]string{
		"Name":    getNameID,
		"Email":   getEmailID,
		"Channel": getChannelID,
	}

	handleUserName := ui.UseEvent(func(getEvent ui.InputEvent) { getForm.SetField("Name", getEvent.GetValue()) })
	handleUserEmail := ui.UseEvent(func(getEvent ui.InputEvent) { getForm.SetField("Email", getEvent.GetValue()) })
	handleUserChannel := ui.UseEvent(func(getEvent ui.ChangeEvent) { getForm.SetField("Channel", getEvent.GetValue()) })
	handleUserSubmit := ui.UseEvent(func(getEvent ui.FormEvent) {
		getEvent.PreventDefault()

		getErrors := validateReviewForm(getForm.Get())
		if len(getErrors) > 0 {
			getForm.SetErrors(getErrors)
			getAnnouncer.Assertive(fmt.Sprintf("Please correct %d field errors.", len(getErrors)))
			getFocus.FocusFirstError(getErrors, getFieldIDs, "Name", "Email", "Channel")
			return
		}

		getAnnouncer.Polite("Review request is ready to submit.")
	})

	getValue := getForm.Get()

	return h.Main(
		h.Class("space-y-5 p-6"),
		getAnnouncer.Region(),
		h.Form(
			h.OnSubmit(handleUserSubmit),
			h.Class("space-y-4"),
			h.Div(
				h.Label(h.For(getNameID), "Reviewer name"),
				h.Input(
					h.ID(getNameID),
					h.Value(getValue.Name),
					h.OnInput(handleUserName),
					h.Aria("invalid", fmt.Sprintf("%t", getForm.HasFieldError("Name"))),
					h.Aria("describedby", getNameID+"-error"),
				),
				h.P(h.ID(getNameID+"-error"), getForm.FieldMessage("Name")),
			),
			h.Div(
				h.Label(h.For(getEmailID), "Email"),
				h.Input(
					h.ID(getEmailID),
					h.Value(getValue.Email),
					h.OnInput(handleUserEmail),
					h.Aria("invalid", fmt.Sprintf("%t", getForm.HasFieldError("Email"))),
					h.Aria("describedby", getEmailID+"-error"),
				),
				h.P(h.ID(getEmailID+"-error"), getForm.FieldMessage("Email")),
			),
			h.Div(
				h.Label(h.For(getChannelID), "Notification channel"),
				h.Select(
					h.ID(getChannelID),
					h.Value(getValue.Channel),
					h.OnChange(handleUserChannel),
					h.Option(h.Value(""), "Select one"),
					h.Option(h.Value("email"), "Email"),
					h.Option(h.Value("slack"), "Slack"),
				),
				h.P(h.ID(getChannelID+"-error"), getForm.FieldMessage("Channel")),
			),
			h.Button(h.Type("submit"), "Validate request"),
		),
	)
}
```

Why this is the right first pattern:

- one typed form handle owns values and validation messages
- spoken feedback and focus movement stay inside the same feature
- the HTML remains ordinary and accessible instead of hiding state in ad hoc DOM code

### Field names are strings — catch typos

`SetField(name, value)` takes the struct field name as a string and returns `false` for **both** an
unknown field and a mismatched value type — so a typo (`SetField("Eamil", …)`) silently no-ops. Two
helpers make that catchable:

- **`MustSetField(name, value)`** — same as `SetField` but *panics* on an unknown field or type
  mismatch. Prefer it where the field name is a hardcoded literal, so a typo fails loudly in
  development instead of silently dropping writes.
- **`HasField(name) bool`** — reports whether the field exists; useful in a test or a dev-time guard
  to assert the names a form wires up are real.

```go
getForm.MustSetField("Name", getEvent.GetValue()) // panics if "Name" isn't a field of the model
if !getForm.HasField("Channel") { /* misspelled in this build — fix it */ }
```

## Production-Shaped Example

For a real server-backed form, keep the mutation authority on the server, use intent-aware submit state in `ui.UseForm[T]`, and project the typed result envelope back into the same form state.

```go
package editor

import (
	"strings"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type articleForm struct {
	Title string
	Body  string
}

// validateArticleIntent validates the current form value in the context of one submit intent.
func validateArticleIntent(getValue articleForm, getIntent string) ui.FieldErrors {
	getErrors := ui.FieldErrors{}
	if strings.TrimSpace(getValue.Title) == "" {
		getErrors["Title"] = "Title is required."
	}
	if strings.TrimSpace(getValue.Body) == "" {
		getErrors["Body"] = "Body is required."
	}
	if getIntent == "publish" && len(strings.TrimSpace(getValue.Body)) < 40 {
		getErrors["Body"] = "Published content needs a longer body."
	}
	return getErrors
}

// runArticleAction simulates one server-owned action result while keeping CSRF naming explicit.
func runArticleAction(getValue articleForm, getIntent string, getToken ui.CSRFToken) ui.ServerActionResult {
	getHeaderName, getHeaderValue := getToken.Header()
	_, _ = getHeaderName, getHeaderValue

	if strings.Contains(strings.ToLower(getValue.Title), "forbidden") {
		return ui.ServerActionResult{
			Outcome: ui.ServerActionOutcomeValidationError,
			Message: "Server rejected the current article.",
			Fields:  ui.FieldErrors{"Title": "Title contains a blocked term."},
		}
	}
	if getIntent == "publish" {
		return ui.ServerActionResult{
			Outcome: ui.ServerActionOutcomeRedirect,
			Redirect: &ui.ServerActionRedirect{
				Location: "/editor/published",
			},
		}
	}
	return ui.ServerActionResult{
		Outcome: ui.ServerActionOutcomeSuccess,
		Message: "Draft saved.",
	}
}

// renderArticleEditor keeps local field state in UseForm while leaving authoritative outcomes to the server action.
func renderArticleEditor(getTokenValue string) ui.Node {
	getForm := ui.UseForm(articleForm{})
	getFocus := ui.UseFocusManager()
	getAnnouncer := ui.UseAnnouncer()
	getStatus := ui.UseState("Choose draft or publish.")
	getFieldIDs := map[string]string{"Title": "article-title", "Body": "article-body"}

	handleUserTitle := ui.UseEvent(func(getEvent ui.InputEvent) { getForm.SetField("Title", getEvent.GetValue()) })
	handleUserBody := ui.UseEvent(func(getEvent ui.InputEvent) { getForm.SetField("Body", getEvent.GetValue()) })
	handleUserDraft := ui.UseEvent(func() {
		if !getForm.ValidateIntent("draft", validateArticleIntent) {
			getAnnouncer.Assertive("Draft validation failed.")
			getFocus.FocusFirstError(getForm.Errors(), getFieldIDs, "Title", "Body")
			return
		}
		getForm.SubmitWithIntent("draft", func(getValue articleForm, getIntent string) error {
			getResult := runArticleAction(getValue, getIntent, ui.NewCSRFToken(getTokenValue))
			if !getForm.ApplyServerActionResult(getResult) {
				getAnnouncer.Assertive(getResult.FormErrors().FormMessage())
				getFocus.FocusFirstError(getForm.Errors(), getFieldIDs, "Title", "Body")
				return nil
			}
			getStatus.Set(getResult.Message)
			getAnnouncer.Polite(getResult.Message)
			return nil
		})
	})
	handleUserPublish := ui.UseEvent(func() {
		if !getForm.ValidateIntent("publish", validateArticleIntent) {
			getAnnouncer.Assertive("Publish validation failed.")
			getFocus.FocusFirstError(getForm.Errors(), getFieldIDs, "Title", "Body")
			return
		}
		getForm.SubmitWithIntent("publish", func(getValue articleForm, getIntent string) error {
			getResult := runArticleAction(getValue, getIntent, ui.NewCSRFToken(getTokenValue))
			if !getForm.ApplyServerActionResult(getResult) {
				getAnnouncer.Assertive(getResult.FormErrors().FormMessage())
				getFocus.FocusFirstError(getForm.Errors(), getFieldIDs, "Title", "Body")
				return nil
			}
			if getResult.HasRedirect() {
				getStatus.Set("Redirect to " + getResult.RedirectLocation())
				getAnnouncer.Polite("Article published.")
				return nil
			}
			getStatus.Set("Article saved.")
			return nil
		})
	})

	getValue := getForm.Get()

	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		getAnnouncer.Region(),
		h.Input(h.ID("article-title"), h.Value(getValue.Title), h.OnInput(handleUserTitle)),
		h.Textarea(h.ID("article-body"), h.OnInput(handleUserBody), h.Text(getValue.Body)),
		h.P(getForm.FormError()),
		h.P(getStatus.Get()),
		h.Div(
			h.Class("flex gap-3"),
			h.Button(h.Type("button"), h.Disabled(getForm.IntentPending("draft")), h.OnClick(handleUserDraft), "Save draft"),
			h.Button(h.Type("button"), h.Disabled(getForm.IntentPending("publish")), h.OnClick(handleUserPublish), "Publish"),
		),
	)
}
```

Why this is the production baseline:

- local field state stays in `UseForm[T]`
- the server-action result remains the authority for field errors, form errors, redirects, and success
- submit intent is explicit, so draft and publish do not need two parallel form state machines
- CSRF naming stays aligned whether the action is progressive HTML, enhanced submit, or imperative request

## Scale-Up Example

In a larger app, keep locale state, routing policy, and accessibility shell behavior in one top-level owner so feature forms can stay narrow and local.

```go
package shell

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

var workspaceBundle = func() *i18n.Bundle {
	getBundle := i18n.NewBundle(i18n.BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	getBundle.Register("en", i18n.Catalog{
		"shell": {
			"heading": i18n.Message{Text: "Workspace settings"},
		},
	})
	getBundle.Register("fr", i18n.Catalog{
		"shell": {
			"heading": i18n.Message{Text: "Parametres de l'espace de travail"},
		},
	})
	return getBundle
}()

type workspaceShellProps struct {
	Path string
}

// renderWorkspaceShell resolves locale from the current path and provides one runtime for the whole shell.
func renderWorkspaceShell(getProps workspaceShellProps) ui.Node {
	getResolved := i18n.ResolvePath(getProps.Path, i18n.RouteOptions{
		SupportedLocales:  []string{"en", "fr"},
		DefaultLocale:     "en",
		OmitDefaultPrefix: true,
	})
	getLocale := i18n.UseLocale(i18n.LocaleOptions{
		InitialLocale:    getResolved.Locale,
		SupportedLocales: []string{"en", "fr"},
		FallbackLocale:   "en",
		PersistenceKey:   "workspace-locale",
	})

	return i18n.Provider(i18n.ProviderProps{
		Locale: getLocale,
		Bundle: workspaceBundle,
		Child:  ui.CreateElement(renderWorkspaceShellBody, workspaceShellProps{Path: getResolved.BasePath}),
	})
}

// renderWorkspaceShellBody owns route announcements, heading focus, and locale-prefixed links for the current shell.
func renderWorkspaceShellBody(getProps workspaceShellProps) ui.Node {
	getIntl := i18n.UseI18n()
	getAnnouncer := ui.UseAnnouncer()
	getFocus := ui.UseFocusManager()

	ui.UseEffect(func() func() {
		getAnnouncer.Polite("Loaded " + getProps.Path)
		getFocus.FocusSelector("#workspace-heading")
		return nil
	}, getProps.Path, getIntl.Locale())

	return h.Main(
		h.Raw(map[string]any{"lang": getIntl.Locale(), "dir": string(getIntl.Direction())}),
		getAnnouncer.Region(),
		h.H1(h.ID("workspace-heading"), getIntl.T("shell", "heading")),
		h.Nav(
			h.A(h.Href(getIntl.PrefixPath("/settings")), "Settings"),
			h.A(h.Href(getIntl.PrefixPath("/billing")), "Billing"),
		),
	)
}
```

How this scales:

- one app shell owns locale persistence, route announcements, `lang` or `dir`, and locale-prefixed link generation
- feature routes and forms can just call `i18n.UseI18n()` instead of reparsing the URL or loading ad hoc message maps
- server loaders can use `i18n.ResolvePath(...)` to choose locale-specific content without duplicating entire route trees per market

## Shared Client/Server Validation (default path)

Validate with **one schema on both sides**. Put `validate:"..."` tags on the request struct, then
call `form.ValidateStruct()` in the browser and `validate.Struct(req)` in the server handler — the
*same* type, the *same* rules, no hand-written client validator that can drift from the server:

```go
type Signup struct {
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=13"`
}

// Client (wasm): ui.UseForm + ValidateStruct runs validate.Struct on Signup.
form := ui.UseForm(Signup{})
if form.ValidateStruct() {
	// submit; the server re-runs validate.Struct on the identical type
}

// Server handler (native): the same struct, the same tags.
//   if res := validate.Struct(req); !res.Valid() { return res.Fields() }
```

`form.ValidateStruct()` is the turnkey default — prefer it over hand-rolling a client validator.
The `validate` package is dependency-free and compiles to both wasm and native, which is what makes
the single-schema guarantee real. Worked end-to-end example:
[`examples/public/shared-form-validation`](../../examples/public/shared-form-validation/). For
ad-hoc per-field checks not expressible as tags, `form.Validate(func(T) ui.FieldErrors)` remains
available; async/server-driven validation flows through `form.ValidateAsync(...)` and
`form.ApplyServerErrors(...)`.

**Custom tag rules.** Register a domain rule once with `validate.RegisterRule` and use it as a tag
on either side — it runs in `validate.Struct` everywhere, preserving the single-schema guarantee:

```go
validate.RegisterRule("sku", func(v any, arg string) (bool, string) {
    s, _ := v.(string)
    return s == "" || strings.HasPrefix(s, "SKU-"), "must start with SKU-"
})

type Item struct {
    Code string `validate:"required,sku"` // built-in + custom rule compose
}
```

The rule receives the field value (type-assert it) and the tag argument after `=` (e.g. `arg=="5"`
for `validate:"between=5"`). Registration is concurrency-safe; a blank name, a nil function, or a
built-in name (`required`, `email`, `min`, …) is ignored, so the core rule set can't be shadowed.

**Cross-field rules.** Constraints that span fields (password==confirm, end≥start, "required if
plan==team") can't be a per-field tag. `validate.Check` runs the struct tags *and* programmatic
cross-field rules into one merged `Result`, preserving the single-schema guarantee:

```go
res := validate.Check(signup,
    func(s Signup) *validate.FieldError {
        if s.Password != s.Confirm {
            return validate.Fail("confirm", "must match password")
        }
        return nil // valid
    },
)
// res.Fields() now carries both tag failures and the cross-field failure.
```

A rule returns `nil` when valid or `validate.Fail(field, message)` to attach a failure. Nil rules
are skipped and a panicking rule is contained (recorded as a failure, never crashing the caller).

## Form Workflows

The recommended form ladder is:

1. start with `ui.UseForm[T]` plus synchronous `Validate(...)`
2. add `ValidateAsync(...)` only when the check truly depends on background or server knowledge
3. use `Submit(...)` or `SubmitWithIntent(...)` instead of hand-rolled pending flags
4. map authoritative failures back with `ApplyServerErrors(...)` or `ApplyServerActionResult(...)`
5. keep redirect or refresh policy server-owned for server actions

Use `FieldStatus(...)`, `FieldMessage(...)`, and `HasFieldError(...)` when the UI needs a field-level summary without repeating touched, dirty, pending, and error lookups inline.

For file uploads:

- the current exported browser-file helper is `ui.GetFiles(...)`
- pair it with `fetch.MultipartBody`, `fetch.MultipartFile`, and `fetch.Upload(...)` when progress or cancellation matters
- keep multipart reserved for workflows that really need browser-managed file semantics

For secure server-backed forms:

- emit `ui.NewCSRFToken(...).FormField()` in request-time rendered HTML forms
- use `token.Header()` for imperative or hydrated requests
- treat `403` CSRF failures and typed server validation as authoritative server outcomes, not retryable client guesses

## Accessibility Workflows

The accessibility model is explicit and HTML-first.

Use these rules:

- keep labels, ids, `aria-describedby`, `aria-invalid`, and visible error text in ordinary markup
- use `ui.UseId()` for stable ids instead of manual string coordination
- use `ui.UseFocusManager()` for focus restoration, route-heading focus, and focus-to-error behavior
- use `ui.UseAnnouncer()` for route changes, async status changes, and validation summaries
- use `ui.AccessibleOverlay(...)` for modal-first dialogs, and move to `ui.Overlay(...)` only when you need broader stack-aware layering
- use `ui.UseFocusTrap(...)` when you need explicit trap ownership outside the higher-level overlay contract

One important boundary: the framework gives you focus and announcement primitives, but it does not infer semantics for you. If the app needs a spoken route change, error summary, or dialog label, application code still needs to provide it.

## Locale And Routing Workflows

Treat locale state, bundle lookup, and route policy as separate concerns:

- `i18n.UseLocale(...)` owns runtime locale state, optional browser detection, and persistence
- `i18n.Bundle` owns message catalogs and fallback lookup
- `i18n.Provider(...)` and `i18n.UseI18n()` expose translation and formatting inside components
- `i18n.PrefixPath(...)` and `i18n.ResolvePath(...)` help build locale-aware route policy without making locale a router global

For SSR:

- trim the initial message subset on the server with `Bundle.ToSSRBootstrap(...)`
- restore it on the client with `i18n.BundleFromSSRBootstrap(...)`
- keep `lang` and `dir` explicit in the owned shell markup instead of assuming document mutation happens automatically

## API Families

| Family | Primary APIs | Stability | Use this when | Do not use this when | Notes |
| --- | --- | --- | --- | --- | --- |
| Local form state | `UseForm`, `SetField`, `Validate`, `ValidateAsync`, `Submit`, `Reset` | `Stable` | one feature owns several related fields and needs touched, dirty, error, and pending state together | one uncontrolled HTML form post with no client state is enough | `UseForm[T]` is the default local form owner |
| Intent-aware mutations | `SetSubmitIntent`, `ValidateIntent`, `SubmitWithIntent`, `IntentPending` | `Stable` | one form supports actions such as draft versus publish or approve versus reject | the actions really belong to separate screens or separate forms | keep the server result authoritative |
| Server result projection | `ServerFormErrors`, `ServerActionResult`, `ApplyServerErrors`, `ApplyServerActionResult` | `Stable` | the server returns field or form errors that should land back in the same UI state | the server result is only success or transport failure | progressive and hydrated flows should converge here |
| CSRF helpers | `NewCSRFToken`, `Header`, `FormField` | `Stable` | the app wants one consistent naming contract for HTML and imperative mutation requests | the server owns a completely different CSRF contract | current defaults are `X-CSRF-Token` and `csrf_token` |
| Browser file extraction | `GetFiles`, `ui.File` | `Stable` | file inputs feed multipart uploads or client-side validation | the form is text-only | older docs may still say `ExtractFiles` |
| Focus management | `UseFocusManager`, `FocusSelector`, `FocusFirstError`, `Restore` | `Stable` | route changes, dialogs, validation summaries, or async transitions need explicit focus movement | the browser default focus order is already correct | keep focus ownership local to the feature or shell |
| Live-region announcements | `UseAnnouncer`, `Polite`, `Assertive`, `Region` | `Stable` | the app needs spoken status for validation, route changes, or async work | every tiny local state change would trigger noise | over-announcing is a real regression |
| Modal accessibility | `AccessibleOverlay`, `UseFocusTrap` | `Stable` | one dialog needs focus containment, dismissal, and background suppression | you need full stack-aware multi-overlay behavior | move to `ui.Overlay(...)` for advanced layering |
| Locale runtime | `UseLocale`, `Provider`, `UseI18n` | `Stable` | locale can change at runtime and components need one translation or formatting runtime | locale is static at build time and there is no runtime switching | separate locale state from route policy |
| Catalogs and SSR locale bootstrap | `NewBundle`, `Register`, `BundleFromSSRBootstrap`, `ToSSRBootstrap` | `Stable` | translation lookup and fallback must be deterministic across SSR and hydration | ad hoc message maps are enough for a one-off demo | transfer only the initial locale subset |
| Locale-aware routing | `PrefixPath`, `ResolvePath` | `Stable` | locale belongs in the URL policy | the router should stay locale-neutral | loader code still owns localized content choice |

## Design Notes And Boundaries

- `ui.UseForm[T]` is local state, not a mutation transport. The server still owns canonical validation, auth, redirects, and side effects.
- The server-action result envelope is the shared contract between progressive HTML form posts and hydrated enhancement. It should not diverge into two business-rule paths.
- Accessibility in this repo is explicit and semantic. The framework does not infer labels, announcements, or route-heading focus automatically.
- `ui.AccessibleOverlay(...)` is modal-first. When the app needs nested sheets, popovers, menus, and stack coordination, move to the broader overlay surface intentionally.
- `i18n` keeps locale state, catalog lookup, formatting, and path helpers small and composable. It does not own translation file formats or global document mutation policy. For **lazy locale loading**, `i18n.LazyBundle` owns the load-once-per-locale orchestration (dedup + register) — you supply the transport via a `LocaleLoader` (e.g. fetch / `interop.ImportModule`) and call `EnsureLocale(locale)` before switching, so only the active locale ships on first paint.
- Locale routing is application policy layered on top of `router`, not a router-owned global mode.

## Common Failure Modes

- duplicating field state, pending flags, and server-error state outside `ui.UseForm[T]`
- treating client validation success as proof that the server must also accept the mutation
- forgetting to project typed server field errors back into the current form state
- relying on color alone for validation feedback without focus movement or text
- announcing every keystroke instead of only meaningful status changes
- opening dialogs without a stable title, description, and focus-restoration path
- reparsing the URL for locale in every route instead of resolving it once in the shell or loader
- shipping the full translation catalog in SSR bootstrap when only the initial locale subset is needed
- using outdated docs that still refer to `ExtractFiles(...)` instead of the current `GetFiles(...)`

## Validation

Use the smallest checks for the exact slice you changed:

```powershell
go run ./tools/gwc build -app .\examples\public\use-form\main.go -root .\examples\public\use-form
go run ./tools/gwc build -app .\examples\public\accessible-overlay\main.go -root .\examples\public\accessible-overlay
go run ./tools/gwc build -app .\examples\public\form-accessibility\main.go -root .\examples\public\form-accessibility
go run ./tools/gwc build -app .\examples\public\routed-accessibility\main.go -root .\examples\public\routed-accessibility
go run ./tools/gwc build -app .\examples\public\locale-switcher\main.go -root .\examples\public\locale-switcher
go run ./tools/gwc build -app .\examples\public\server-side-rendering-internationalization-bootstrap\main.go -root .\examples\public\server-side-rendering-internationalization-bootstrap
go run ./tools/gwc build -app .\examples\public\locale-routing\main.go -root .\examples\public\locale-routing
go run .\examples\server\server-side-rendering-secure-forms
go test ./ui ./i18n
```

For manual browser checks, confirm:

- focus moves to the first invalid field after validation failure
- polite and assertive announcements are not duplicated or stale
- modal dialogs restore focus to the trigger after close
- locale switches update `lang`, `dir`, translated copy, and locale-prefixed links together

## Typed Message Accessors (`gwc i18n gen`)

To make message keys and interpolation parameters compile-checked, generate typed accessors from a
base-locale bundle. Given `messages.en.json`:

```json
{ "greeting": { "hello": "Hello {name}" } }
```

`gwc i18n gen -bundle messages.en.json -pkg .` writes `i18n_keys_gen.go` with one typed accessor per
message, wrapping `i18n.Runtime.T` and taking a string argument for each `{param}`:

```go
msg := i18nkeys.GreetingHello("Ada") // a typo in the key, or a missing/extra param, is a compile error
```

`gwc i18n check` is the CI staleness gate (regenerate + diff). This is the i18n counterpart to
`gwc routes gen` / `gwc server gen`, closing the last stringly-typed surface.

### Optional fields (`omitempty`)

The `validate` package supports an `omitempty` rule: an optional field whose value is the zero value
skips its remaining rules, and is validated normally when present.

```go
type profile struct {
	Name    string `validate:"required"`
	Website string `validate:"omitempty,url"` // validated only when non-empty
}
```

## Topic Pagination
Topic 11 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [10 Browser Interop And Workers](10-browser-interop-and-workers.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md)
