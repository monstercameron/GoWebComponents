# Business-App Form Recipes

This guide turns the current `ui.UseForm[T]` surface into practical workflow patterns for internal tools and business applications.

Use it when a form needs more than field binding and basic validation:

- inline field errors
- pending submit UX
- server validation round-trips
- multi-intent submit buttons
- redirect-after-submit behavior
- uploads
- authoritative server mutation handling

## The Short Version

Recommended defaults for serious product forms:

- keep `ui.UseForm[T]` as the local field-state owner
- keep the server authoritative for business-rule validation and final mutation success
- map server field errors back into the same form handle with `ApplyServerErrors(...)`
- show visible pending state during submit, not spinner-only feedback
- use redirect-after-submit only when the workflow genuinely changes pages
- reserve optimistic submit behavior for clearly safe cases
- use multipart only when the workflow really needs file transfer

## 1. The Default Form Shape

For most business forms, the baseline shape should be:

1. typed form state in `ui.UseForm[T]`
2. synchronous client validation for obvious local issues
3. explicit authoritative submit handler
4. server validation or mutation result mapped back into the same form
5. targeted revalidation, reset, or redirect on success

That keeps one clear ownership model:

- local form state is client-owned
- mutation success is server-owned

## 2. Inline Validation And Field Projection

Use `Validate(...)` for immediate checks the client can know locally.

Good local validation:

- required fields
- obvious formatting rules
- mutually exclusive client-known field combinations

Then project the result back through the form helpers:

- `form.Error(...)`
- `form.FieldMessage(...)`
- `form.FieldStatus(...)`
- `form.HasFieldError(...)`

Recommended render rule:

- keep field copy close to the control
- reflect `Touched`, `Dirty`, `Pending`, and `Error` through `FieldStatus(...)`
- do not hand-roll separate field-status state when `UseForm` already owns it

## 3. Server Validation Round-Trips

For real business workflows, the server still owns:

- business rules
- auth-sensitive validation
- duplicate checks
- policy checks
- canonical normalization

Recommended pattern after a failed server submit:

1. decode the server response into `ui.ServerFormErrors`
2. call `form.ApplyServerErrors(...)`
3. keep the current field values intact
4. let the user correct and retry

This keeps local drafts and server-authoritative feedback inside the same public form surface.

## 4. Pending UX

Pending UX should be explicit and visible.

Recommended rules:

- disable only the controls that should not be repeated
- keep the form state visible while pending
- show visible copy such as `Saving changes...` or `Uploading attachment...`
- use `aria-busy` or equivalent semantics when the region is meaningfully busy

Useful helpers:

- `form.Submitting()`
- `form.SubmitError()`
- `form.Submitted()`
- `form.IntentPending(...)`

Do not rely on spinner-only feedback for business workflows with real latency.

## 5. Multi-Intent Buttons

Many internal forms have more than one valid action:

- save draft
- publish
- approve
- reject
- hold

Use the intent helpers instead of adding a second local state layer:

- `SetSubmitIntent(...)`
- `ValidateIntent(...)`
- `SubmitWithIntent(...)`
- `IntentPending(...)`

Recommended pattern:

- each button chooses an explicit intent string
- validation may differ by intent
- only the active intent button shows the pending treatment

This keeps “draft versus publish” or “approve versus reject” flows inside one typed form instead of splitting them across multiple ad hoc handlers.

## 6. Redirect After Submit

Redirect after submit is a workflow choice, not a default reflex.

Use redirect when:

- the user should land on a different page after success
- the successful mutation creates or confirms a different canonical resource view
- the server is handling an HTML-post success path

Do not redirect when:

- the user should stay in context and keep working on the same screen
- a simple success banner plus revalidation preserves orientation better
- the form is one part of a larger workspace flow

Practical defaults:

- HTML form success to a different page: redirect
- JSON mutation success on the same screen: keep the user in place, then revalidate or update the current screen
- validation failure: never redirect

## 7. Optimistic Versus Authoritative Submit

Default to authoritative server mutation behavior for business records.

Authoritative by default for:

- inventory
- approvals
- moderation
- receiving
- purchase-order status
- permission-sensitive writes

Optimistic behavior is reasonable only when:

- the user benefit is clear
- rollback is understandable
- the domain tolerates temporary divergence

Good optimistic cases:

- draft-only local staging
- small reversible UI affordances
- comment or note composition where the app already has a clean reconcile path

Bad optimistic cases:

- status changes with audit or policy implications
- server-normalized records
- workflows that frequently reject, enrich, or reorder writes

## 8. Upload Workflows

Use multipart only when the form truly includes files.

Recommended upload shape:

1. keep ordinary field state in `ui.UseForm[T]`
2. extract files explicitly from the browser event
3. submit multipart through the upload helpers or explicit multipart fetch
4. map structured server errors back into the form
5. show visible upload progress or pending copy when the transfer matters

Rules:

- do not force all forms into multipart just because one field is file-backed elsewhere
- validate type and size on both client and server
- keep upload progress separate from “mutation is accepted” messaging

## 9. Server-Owned Mutation Handling

The recommended business-app posture is still server-owned mutation handling.

That means:

- server validates the request
- server enforces auth and CSRF
- server decides success, redirect, normalization, and failure
- client reflects the result through the existing form handle

Good server responses:

- field-keyed validation errors
- one aggregate form error
- success payloads that allow local confirmation or revalidation
- redirect responses for HTML-post flows that should move elsewhere

Avoid inventing a different mutation response contract for every form.

When a workflow needs one authoritative contract that works for both progressive HTML posts and hydrated enhancements, use the server-action model defined in [SERVER_ACTIONS.md](SERVER_ACTIONS.md).

## 10. Accessibility And Error Copy

Business forms need understandable error and pending feedback, not just technically valid state.

Recommended rules:

- pair labels and controls clearly
- render field errors close to the field
- use visible form-level error copy for aggregate failures
- announce meaningful pending or success transitions when the workflow is long-lived or high-stakes
- keep focus movement intentional after failed submit

See [ACCESSIBILITY.md](ACCESSIBILITY.md) and `examples/79-form-accessibility`.

## 11. Recommended Defaults For Real Internal Apps

For most non-trivial business forms:

- `ui.UseForm[T]` owns field state
- `Validate(...)` handles obvious client-side issues
- server remains authoritative for business validation
- `ApplyServerErrors(...)` projects server feedback back into the form
- pending state is visible and action-specific
- redirect is used only when the workflow actually changes pages
- uploads are explicit and isolated to upload cases

## 12. Example Starting Points

- local form state and validation basics: `examples/51-use-form`
- more involved client form composition: `examples/10-advanced-form`
- accessibility-focused field and error behavior: `examples/79-form-accessibility`
- secure server-backed HTML forms and uploads: `examples/87-ssr-secure-forms`
- integrated internal-tool patterns: `examples/86-atlas-commerce-os`

## Review Checklist

- does the form have one clear destination style
- does the server remain authoritative for the mutation outcome
- can field and form errors map back into `ui.UseForm[T]` cleanly
- is pending feedback visible and specific to the active action
- is redirect only used where it preserves the workflow
- is optimistic behavior limited to workflows that can explain rollback
