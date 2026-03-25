# Template Lowering Experiment

This is the bounded template-first experiment for the compiler-assisted backlog.

The experiment is intentionally narrow:

- one constrained HTML-like template input
- one readable generated Go output file
- no hidden runtime mode
- ordinary `html` and `ui` package output only

Input:

- `landing.template.html`

Generated output:

- `generated_landing.go`

Regenerate from the repo root:

```powershell
go run ./examples/13-browser-compiler/template_lowering/cmd/lower-template `
  -input ./examples/13-browser-compiler/template_lowering/landing.template.html `
  -output ./examples/13-browser-compiler/template_lowering/generated_landing.go
```

Current limits are deliberate:

- only a tiny supported tag set
- placeholder fields must be whole text nodes like `{{.Headline}}`
- no control flow, loops, or event binding

That makes the generated Go easy to inspect and keeps the experiment clearly optional instead of competing with the normal Go-first authoring model.
