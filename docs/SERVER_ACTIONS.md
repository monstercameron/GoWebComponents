# Server Actions

This document defines the first-class server action contract for form submissions in GoWebComponents.

Use it when the same mutation must work in both of these modes:

- as a normal HTML form post with no JavaScript
- as a hydrated enhancement with richer pending, error, and retry UX

The goal is one server-owned mutation contract, not one contract for progressive forms and another for hydrated UI.

## At A Glance

- a server action is a server-owned form mutation endpoint with one typed result envelope
- the browser may submit it as a plain form post or through hydrated enhancement
- success, redirect, validation failure, auth failure, and retryable failure all use the same action semantics
- `ui.UseForm[T]` remains the local field-state owner; the server remains the mutation authority

## Contract

Treat a form-oriented server action as one explicit unit with these parts:

- action identifier or route: the authoritative server endpoint that owns the mutation
- request shape: ordinary form fields or multipart payload shaped for progressive browser submission first
- auth and CSRF boundary: same-origin session, CSRF validation, and permission checks remain server-owned
- typed result envelope: one result model that both HTML-post and hydrated callers can interpret
- follow-up instruction: stay in place, redirect, refresh data, or retry later

## Action Outcomes

Every server action should resolve into one of these outcomes:

- success: the mutation was accepted and the current screen may confirm or refresh
- redirect: the server chose a different canonical destination after success
- validation failure: field or form errors should project back into the current form without losing local intent
- auth failure: unauthenticated, unauthorized, or expired-session state blocks completion and may redirect to sign-in
- retryable failure: temporary server, network, or dependency failure where the user can safely retry

These outcomes should exist regardless of whether the request arrived as a browser-native form post or a hydrated enhanced submit.

## Ownership Model

Use this split consistently:

- `ui.UseForm[T]` owns local values, touched state, dirty state, client validation, and pending UX
- the server action owns canonical validation, normalization, auth, CSRF, mutation side effects, redirect choice, and authoritative success
- the result envelope is the bridge between them

Do not invent separate mutation semantics for HTML posts, JSON endpoints, and hydrated handlers when they all represent the same underlying action.

## Progressive And Hydrated Behavior

Progressive-first is the baseline:

- the form must still be submit-capable as a normal HTML post when the workflow claims progressive enhancement support
- the server must return an understandable next step for a non-JavaScript browser

Hydrated enhancement may then add:

- visible pending state
- intent-specific disablement
- structured error projection into `ui.UseForm[T]`
- local success banners or refresh behavior before or instead of a full redirect

Hydrated enhancement must not create a second authoritative mutation path with different business rules.

## Recommended Shape

For one form mutation:

1. define one server-owned action route
2. accept browser-native form or multipart input
3. validate auth, CSRF, and business rules on the server
4. return one typed action result envelope
5. map that envelope back into either HTML-post behavior or hydrated form behavior

That keeps the transport adaptable while the action semantics stay fixed.

## Relationship To Existing Surfaces

- use `ui.UseForm[T]` for local form state
- use `ui.ServerActionResult`, `ui.ServerActionRedirect`, `ui.ServerActionFlash`, and `ui.ServerActionRefresh` for the typed result envelope
- use `result.FormErrors()` or `form.ApplyServerActionResult(result)` to project field and form errors back into the existing public form lifecycle
- use redirect-after-submit rules from `BUSINESS_APP_FORM_RECIPES.md`
- use `net/http` handler conventions from `FORMS.md`
- use this server-action contract when those pieces need one shared mutation model

This document defines the contract. The typed envelope details and helper mapping functions belong in the next layer of server-action result guidance.

## Current Example References

Use these shipped examples together when evaluating the current server-action story:

- `examples/87-ssr-secure-forms`: progressive same-origin HTML form posts with CSRF validation, server-owned validation failures, and redirect-after-submit
- `examples/86-atlas-commerce-os`: hydrated mutation UX that maps structured server failures back through `ui.UseForm[T]` while keeping the server authoritative for the write outcome

That pairing is the current integrated reference slice for server actions:

- progressive HTML post remains the baseline delivery path
- hydrated enhancement may add pending, inline error, and refresh behavior
- both should still converge on the same server-owned action semantics and typed result envelope
