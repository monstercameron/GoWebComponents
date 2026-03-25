# SSR Secure Forms Demo

This example is a focused request-time rendered form flow that demonstrates:

- CSRF-protected HTML form posts using the public `ui.NewCSRFToken(...)` naming helpers
- server-side validation round-trips that preserve submitted values on the same page
- `multipart/form-data` uploads with typed `html.Props{EncType: ...}` markup
- `303 See Other` redirects after successful quote requests and file uploads
- the progressive half of the shared server-action contract documented in `docs/SERVER_ACTIONS.md`

## Run The Server

From the repo root:

```powershell
go run ./examples/87-ssr-secure-forms
```

Then open:

- `http://127.0.0.1:8087/`

Use the quote request form to see server validation round-trips, and use the upload form to see CSRF-aware multipart handling plus redirect-after-submit behavior.

For the hydrated counterpart to the same server-owned mutation model, pair this example with `examples/86-atlas-commerce-os`, where structured server failures project back into `ui.UseForm[T]` while the server still owns success and mutation authority.
