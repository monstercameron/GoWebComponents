# Forms

This page documents the current supported form model for GoWebComponents.

It intentionally distinguishes between shipped first-class behavior and application-owned conventions for server-backed workflows.

## At A Glance

- The primary shipped form API is `ui.UseForm[T]` for typed local form state.
- GoWebComponents supports controlled client forms, hydrated SSR forms, explicit JSON submits, and explicit multipart upload workflows.
- The framework helps with validation state, submit lifecycle, server-error mapping, CSRF token conventions, and browser file extraction.
- Transport, endpoint ownership, server validation, redirects, and storage rules remain application responsibilities.
- The recommended default is explicit, authoritative server-backed form handling rather than hidden framework transport.

## Quick Form Mode Chooser

Use a local controlled form when:

- validation is mostly client-side
- the form submits through an explicit event handler
- the UI benefits from typed field, dirty, touched, and pending state in one place

Use hydrated SSR plus `ui.UseForm[T]` when:

- the page is rendered on the server first but should adopt the same client form state model after hydration
- you want HTML-post or secure server-owned flows without inventing a separate client-only form abstraction

Use JSON submit handlers when:

- the form is fully client-controlled after hydration
- you need structured field-keyed validation responses or explicit client-side routing decisions after success

Use multipart only when:

- the workflow genuinely needs browser-managed file upload semantics
- progress, cancellation, or file transfer are first-class requirements

Rule of thumb: keep `ui.UseForm[T]` as the state owner, keep transport explicit, and choose HTML, JSON, or multipart based on the endpoint contract rather than mixing them casually.

## Example Shape

The current recommended shape is: typed form state in `ui.UseForm[T]`, synchronous validation before submit, and explicit async submission that maps failures back into the same form handle.

```go
package signup

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type signupForm struct {
	Name  string
	Email string
}

func SignupPanel() ui.Node {
	form := ui.UseForm(signupForm{})
	status := ui.UseState("")

	setName := ui.UseEvent(func(event ui.InputEvent) {
		form.SetField("Name", event.GetValue())
	})
	setEmail := ui.UseEvent(func(event ui.InputEvent) {
		form.SetField("Email", event.GetValue())
	})
	submit := ui.UseEvent(func(event ui.FormEvent) {
		event.PreventDefault()
		if !form.Validate(func(value signupForm) ui.FieldErrors {
			errors := ui.FieldErrors{}
			if value.Name == "" {
				errors["Name"] = "Name is required"
			}
			if value.Email == "" {
				errors["Email"] = "Email is required"
			}
			return errors
		}) {
			status.Set("Fix validation errors before submitting.")
			return
		}

		form.Submit(func(value signupForm) error {
			status.Set("Submitted " + value.Email)
			return nil
		})
	})

	value := form.Get()

	return html.Form(html.Props{OnSubmit: submit},
		html.Input(html.Props{Value: value.Name, OnInput: setName}),
		html.Input(html.Props{Value: value.Email, OnInput: setEmail}),
		html.Button(html.Props{Type: "submit"}, html.Text("Submit")),
		html.P(html.Props{}, html.Text(form.Error("Name"))),
		html.P(html.Props{}, html.Text(form.Error("Email"))),
		html.P(html.Props{}, html.Text(status.Get())),
	)
}
```

Why this shape matters:

- one typed handle owns field values, dirty state, touched state, validation state, and submit lifecycle
- inline field errors and aggregate submit state stay in the same public model
- server-backed workflows can still map authoritative failures back through `ApplyServerErrors(...)`
- transport remains explicit instead of being hidden behind framework-owned mutation magic

## Supported Form Modes

The current public form surface is centered on `ui.UseForm[T]` for typed local form state.

Today the supported modes are:

- Local controlled forms with typed struct-backed state through `Get`, `Set`, `Update`, and `SetField`
- Field lifecycle tracking through `Touch`, `Touched`, `TouchedAny`, `Dirty`, and `DirtyAny`
- Field summary helpers through `FieldStatus`, `FieldMessage`, and `HasFieldError`
- Synchronous validation through `Validate`
- Asynchronous validation through `ValidateAsync`
- Form-level and field-level error state through `SetErrors`, `SetFormError`, `Errors`, `Error`, `FormError`, and `ApplyServerErrors`
- Asynchronous submit lifecycle tracking through `Submit`, `Submitting`, `Submitted`, and `SubmitError`
- Submit-intent tracking through `SetSubmitIntent`, `SubmitIntent`, `ValidateIntent`, `SubmitWithIntent`, and `IntentPending`
- CSRF transport conventions through `NewCSRFToken`, `DefaultCSRFHeaderName`, and `DefaultCSRFFormFieldName`
- Browser file extraction through `ExtractFiles` and `File`
- Full reset back to the initial value or a new initial value through `Reset`

The current public model is best suited to:

- browser-only forms that validate locally before submit
- forms that call JSON or imperative mutation endpoints from an event handler
- SSR forms that hydrate into the same `ui.UseForm` state model after first paint

The repo now ships small helpers for the repetitive parts of secure form posting, server validation mapping, and browser-managed multipart uploads, but endpoint ownership and server-side file handling remain application concerns.

## Recommended Authoring Pattern

Use a typed struct as the source of truth and keep `ui.UseForm[T]` as the local state owner.

Recommended flow:

1. Bind fields through `SetField` or `Update`.
2. Run synchronous validation before attempting submit.
3. Use `ValidateAsync` only when the check truly depends on background work or server knowledge.
4. Use `Submit` for async submit lifecycle state instead of hand-rolled loading flags.
5. Use `SubmitWithIntent` and `IntentPending` when the form has actions such as draft versus publish or approve versus reject.
6. Use `FieldStatus` or `FieldMessage` when rendering inline field copy instead of repeating touched, dirty, pending, and error lookups in every component.
7. Reflect field errors inline and use a form-level error for aggregate failures.

Runnable examples:

- [examples/51-use-form](../examples/51-use-form)
- [examples/10-advanced-form](../examples/10-advanced-form)
- [examples/79-form-accessibility](../examples/79-form-accessibility)
- [examples/87-ssr-secure-forms](../examples/87-ssr-secure-forms)
- [examples/86-atlas-commerce-os](../examples/86-atlas-commerce-os)

## Form-Post Destination Conventions

GoWebComponents does not currently ship a framework-owned HTTP form transport. The recommended convention is to keep transport explicit and choose one of these destination styles per form:

- Native browser form posts to `net/http` handlers when progressive enhancement or no-JavaScript fallback matters
- JSON endpoints when the form is fully controlled by hydrated UI state and the client wants structured error handling
- Multipart endpoints only for forms that truly need file transfer semantics

Recommended destination conventions:

- `POST` the canonical action for mutations; keep reads as `GET`
- use SSR or routed page URLs for HTML-returning form handlers
- use `/api/...` endpoints for JSON-returning handlers
- keep one canonical destination per submit path instead of mixing HTML and JSON response shapes on the same endpoint without negotiation
- return field-keyed validation payloads from JSON endpoints so clients can map issues back into `ui.UseForm`

Public helper surface for CSRF-aware posting:

- `ui.NewCSRFToken(token)` to normalize the default header and hidden-field names
- `token.Header()` to retrieve the header name and value for imperative requests
- `token.FormField()` together with `html.HiddenInput(...)` for SSR or progressive HTML forms

For `net/http` handlers, keep the handler contract explicit:

- HTML form handler: accept form fields, validate, then either re-render the page with field errors or redirect on success
- JSON handler: accept structured payload, validate, then return JSON with aggregate and field-keyed error data or success metadata
- multipart handler: reserve for upload cases and keep file handling separate from ordinary text-only form endpoints

## Multipart Upload Workflow

The current multipart story is explicit rather than magical.

Use these pieces together:

- `html.Input(html.Props{Type: "file", Accept: ...})` to declare the file picker contract
- `ui.ExtractFiles(event)` to turn a typed browser event into `[]ui.File`
- `fetch.MultipartBody` and `fetch.MultipartFile` to describe the payload
- `fetch.Upload(...)` when the UI needs progress or cancellation
- `fetch.Fetch(...)` with `MultipartBody` when the request is multipart but the UI does not need progress reporting
- `fetch.Result.Status`, `fetch.Result.Headers`, and `fetch.Result.DecodeJSON(...)` when the server returns structured success or validation payloads after upload

Recommended rules:

- keep ordinary text-only forms on JSON or HTML posts instead of forcing multipart everywhere
- validate file type and size on both client and server
- omit manual `Content-Type` headers for multipart requests so the browser can set the correct boundary
- treat upload progress as transport feedback, not proof that the server has accepted the mutation
- map structured server validation failures back into `ui.UseForm` through `ApplyServerErrors`

## Redirect-After-Submit Semantics

The current recommended redirect policy is:

- use `303 See Other` after successful server-handled HTML form posts that should land on a different page
- avoid redirecting for validation failures; re-render the same page or return field-keyed JSON errors instead
- for fully client-controlled JSON submits, treat navigation as an explicit app decision after success rather than as an implicit transport concern
- when the destination stays on the same route, prefer inline success state or explicit revalidation over a gratuitous redirect

Practical rules:

- server HTML form success to a new page: redirect
- server HTML form success on the same page: allow inline confirmation when that preserves context better than a full reload
- client JSON success to a new route: call the router explicitly after the mutation succeeds
- failed submit with recoverable field errors: stay on the same screen and map errors back into the form state

The focused request-time rendered example for this policy is [examples/87-ssr-secure-forms](../examples/87-ssr-secure-forms), which shows CSRF-aware quote and multipart upload forms, server validation round-trips, and `303 See Other` redirects after success.

## CSRF Token Source And Refresh Rules

The current recommended token model is:

- SSR or bootstrap-capable apps should source the CSRF token from the server-rendered bootstrap payload for the current page load
- the server should also set a matching CSRF cookie so the request can be validated as a double-submit pair
- HTML forms should emit the token through the default hidden field name `csrf_token` unless the server owns a different contract
- JSON or imperative mutation clients should send the same token through the default `X-CSRF-Token` header unless the server owns a different contract

Refresh rules:

- when the server returns a fresh HTML document or fresh bootstrap payload, treat that response as the new token source of truth
- long-lived pages may refresh tokens through an explicit app-owned endpoint, but the framework does not currently ship a token-refresh transport
- if the server rotates tokens per request or per session event, the application must update the in-memory token source before the next mutation
- `403` CSRF failures should be treated as authoritative server failures rather than retried blindly

## Current Server Error Mapping Pattern

The current recommended JSON error shape is represented by `ui.ServerFormErrors` and is already visible in Atlas:

- an aggregate `error` or `message` string for form-level failures
- a field-keyed `fields` object for control-specific messages

Client mapping pattern:

1. Clear previous aggregate and field errors before the next submit attempt.
2. Decode the response into `ui.ServerFormErrors`.
3. Call `form.ApplyServerErrors(response)` to project field and form errors back onto the public form state.
4. Only mark submit success when the transport succeeded and no field or form errors remain.

## Submit-Intent Helpers

The current public form surface now supports intent-aware submit flows without a separate app-local state hook.

Use these helpers when the same form can be submitted with distinct meanings such as draft versus publish, approve versus reject, or hold versus receive.

Available helpers:

- `SetSubmitIntent(intent)` to record the active intent before a later submit or validation step
- `SubmitIntent()` to read the most recently selected intent
- `ValidateIntent(intent, validate)` to run validation rules that differ by action
- `SubmitWithIntent(intent, run)` to submit while carrying the explicit intent into the callback
- `IntentPending(intent)` to keep pending UI attached to the clicked action only

Recommended pattern:

1. Let each action button choose an explicit intent string.
2. Run `ValidateIntent(...)` when validation rules differ by action.
3. Run `SubmitWithIntent(...)` when the submit callback needs to know which action was chosen.
4. Use `IntentPending(...)` to disable or annotate only the active action instead of dimming the whole form.

## Optimistic Versus Authoritative Submit Behavior

The default recommendation is to treat server-backed form writes as authoritative unless the domain is clearly safe for optimistic UI.

Use authoritative behavior by default for:

- moderation decisions
- inventory, threshold, receiving, or purchase-order mutations
- any mutation where the server may reject, normalize, or enrich the payload
- any mutation that depends on auth, CSRF, or server-owned business rules

Optimistic behavior is reasonable only when all of these are true:

- the user benefit of immediate feedback is clear
- rollback or correction is straightforward
- the domain can tolerate temporary divergence until the server responds
- the app already has a coherent retry and reset story

Recommended rules:

- optimistic local drafts: yes
- optimistic transient UI affordances such as button state or local banners: yes
- optimistic server-backed record creation or moderation state changes: no by default
- failed authoritative submit: keep the form values, map field or form errors back, and let the user retry
- successful authoritative submit: revalidate, reset, or navigate based on the workflow rather than assuming local speculative state is still correct

## Security Boundary

`ui.UseForm` manages local state. It is not a security boundary.

Applications still own:

- CSRF defense for server-backed mutations
- authn or authz enforcement for write endpoints
- input validation on the server
- redirect safety and allowlist rules
- upload limits and storage rules

Atlas demonstrates one current server-backed pattern:

- hidden `csrf_token` field for SSR forms
- matching CSRF cookie and header or field validation on mutation requests
- field-keyed JSON validation errors for hydrated forms

The dedicated request-time rendered example is [examples/87-ssr-secure-forms](../examples/87-ssr-secure-forms), which keeps the forms server-owned and shows the current recommended secure HTML-post flow without requiring Atlas-specific domain context.

See [examples/86-atlas-commerce-os/server/README.md](../examples/86-atlas-commerce-os/server/README.md) for the current repo example of that server contract.
See [SECURITY.md](SECURITY.md) for the broader server-only data and logging redaction policy.

## Related Docs

- [START_HERE.md](START_HERE.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md#rendering-and-local-state)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [examples/README.md](../examples/README.md)

## Review Checklist

Before standardizing a form workflow, verify all of the following:

- the form has one clear destination style: HTML post, JSON mutation, or multipart upload
- validation rules are split deliberately between local checks and server-authoritative checks
- field-keyed server errors can map back into `ui.UseForm[T]` without custom ad hoc parsing at every call site
- CSRF token sourcing and refresh rules are explicit for server-backed mutations
- redirects happen only on success paths that genuinely need navigation
- optimistic behavior is opt-in and justified by the domain rather than treated as the default
