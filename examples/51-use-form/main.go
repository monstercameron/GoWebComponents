//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type signupForm struct {
	Name   string
	Email  string
	OptIn  bool
	Region string
}

func validateSignup(value signupForm) ui.FieldErrors {
	errors := ui.FieldErrors{}
	if len(strings.TrimSpace(value.Name)) < 2 {
		errors["Name"] = "Enter at least 2 characters"
	}
	if !strings.Contains(value.Email, "@") {
		errors["Email"] = "Enter a valid email"
	}
	return errors
}

func useFormExample() ui.Node {
	form := ui.UseForm(signupForm{Region: "us"})
	status := ui.UseState("Edit the fields, validate, then submit.")

	setName := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Name", event.GetValue()) })
	setEmail := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Email", event.GetValue()) })
	toggleOptIn := ui.UseEvent(func(event ui.ChangeEvent) { form.SetField("OptIn", event.IsChecked()) })
	setUS := ui.UseEvent(func() { form.SetField("Region", "us") })
	setEU := ui.UseEvent(func() { form.SetField("Region", "eu") })
	validateSync := ui.UseEvent(func() {
		if form.Validate(validateSignup) {
			status.Set("Synchronous validation passed.")
		} else {
			status.Set("Synchronous validation found field errors.")
		}
	})
	validateAsync := ui.UseEvent(func() {
		status.Set("Running async validation...")
		form.ValidateAsync(func(value signupForm) (ui.FieldErrors, string) {
			time.Sleep(300 * time.Millisecond)
			errors := validateSignup(value)
			formError := ""
			if strings.HasSuffix(strings.ToLower(value.Email), "@blocked.test") {
				formError = "This domain is blocked for the demo"
			}
			return errors, formError
		}, func(valid bool) {
			if valid {
				status.Set("Async validation passed.")
			} else {
				status.Set("Async validation returned field or form errors.")
			}
		})
	})
	submit := ui.UseEvent(func(event ui.FormEvent) {
		event.PreventDefault()
		if !form.Validate(validateSignup) {
			status.Set("Fix validation errors before submitting.")
			return
		}
		status.Set("Submitting...")
		form.Submit(func(value signupForm) error {
			time.Sleep(350 * time.Millisecond)
			if strings.HasSuffix(strings.ToLower(value.Email), "@blocked.test") {
				return fmt.Errorf("server rejected %s", value.Email)
			}
			return nil
		})
	})
	reset := ui.UseEvent(func() {
		form.Reset(signupForm{Region: "us"})
		status.Set("Form reset to initial values.")
	})

	value := form.Get()
	region := value.Region
	if region == "" {
		region = "us"
	}

	result := status.Get()
	if form.Submitting() {
		result = "Submitting..."
	} else if form.Submitted() {
		result = "Submit succeeded."
	} else if form.SubmitError() != nil {
		result = form.SubmitError().Error()
	} else if form.FormError() != "" {
		result = form.FormError()
	}

	return shared.ExamplePage(
		"ui.UseForm",
		"Manage field values, dirty state, validation, and submission lifecycle together",
		"UseForm keeps multi-field local form bookkeeping in one typed handle: field updates, touched and dirty state, sync and async validation, and async submission status.",
		shared.ExamplePanel("Form fields",
			html.Form(html.Props{OnSubmit: submit, Class: "mt-3 grid gap-4"},
				html.Div(html.Props{},
					html.Label(html.Props{For: "signup-name", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Name")),
					html.Input(html.Props{ID: "signup-name", Value: value.Name, OnInput: setName, Placeholder: "Ada", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
					html.P(html.Props{Class: "mt-2 text-sm text-rose-300"}, html.Text(form.Error("Name"))),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: "signup-email", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Email")),
					html.Input(html.Props{ID: "signup-email", Value: value.Email, OnInput: setEmail, Placeholder: "ada@example.com", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
					html.P(html.Props{Class: "mt-2 text-sm text-rose-300"}, html.Text(form.Error("Email"))),
				),
				html.Label(html.Props{Class: "flex items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
					html.Input(html.Props{Type: "checkbox", Checked: value.OptIn, OnChange: toggleOptIn}),
					html.Text("Receive release updates"),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					shared.ExampleButton("Region: US", setUS),
					shared.ExampleButton("Region: EU", setEU),
					html.Button(html.Props{Type: "submit", Class: "rounded-full border border-emerald-500/40 bg-emerald-500/10 px-5 py-3 font-semibold text-emerald-100 hover:bg-emerald-500/20"}, html.Text("Submit")),
				),
			),
		),
		shared.ExamplePanel("Form state",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Validate", validateSync),
				shared.ExampleButton("Async validate", validateAsync),
				shared.ExampleButton("Reset", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Touched any", fmt.Sprintf("%t", form.TouchedAny())),
				shared.ExampleStat("Dirty any", fmt.Sprintf("%t", form.DirtyAny())),
				shared.ExampleStat("Validating", fmt.Sprintf("%t", form.Validating())),
				shared.ExampleStat("Submitting", fmt.Sprintf("%t", form.Submitting())),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Submitted", fmt.Sprintf("%t", form.Submitted())),
				shared.ExampleStat("Has errors", fmt.Sprintf("%t", form.HasErrors())),
				shared.ExampleStat("Region", region),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(result)),
			shared.ExampleCode(
				`form := ui.UseForm(signupForm{Region: "us"})`,
				`form.SetField("Email", value)`,
				`form.Validate(validateSignup)`,
				`form.Submit(func(value signupForm) error { ... })`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useFormExample), "#app")
	select {}
}