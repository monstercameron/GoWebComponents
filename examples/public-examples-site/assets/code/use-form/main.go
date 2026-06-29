//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

type signupForm struct {
	Name   string
	Email  string
	OptIn  bool
	Region string
}

func validateSignup(parseValue signupForm) ui.FieldErrors {
	parseErrors := ui.FieldErrors{}
	if len(strings.TrimSpace(parseValue.Name)) < 2 {
		parseErrors["Name"] = "Enter at least 2 characters"
	}
	if !strings.Contains(parseValue.Email, "@") {
		parseErrors["Email"] = "Enter a valid email"
	}
	return parseErrors
}

func useFormExample() ui.Node {
	parseForm := ui.UseForm(signupForm{Region: "us"})
	parseStatus := ui.UseState("Edit the fields, validate, then submit.")

	setName := ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField("Name", parseEvent.GetValue()) })
	setEmail := ui.UseEvent(func(parseEvent2 ui.InputEvent) { parseForm.SetField("Email", parseEvent2.GetValue()) })
	parseToggleOptIn := ui.UseEvent(func(parseEvent3 ui.ChangeEvent) { parseForm.SetField("OptIn", parseEvent3.IsChecked()) })
	setUS := ui.UseEvent(func() { parseForm.SetField("Region", "us") })
	setEU := ui.UseEvent(func() { parseForm.SetField("Region", "eu") })
	parseValidateSync := ui.UseEvent(func() {
		if parseForm.Validate(validateSignup) {
			parseStatus.Set("Synchronous validation passed.")
		} else {
			parseStatus.Set("Synchronous validation found field errors.")
		}
	})
	parseValidateAsync := ui.UseEvent(func() {
		parseStatus.Set("Running async validation...")
		parseForm.ValidateAsync(func(parseValue2 signupForm) (ui.FieldErrors, string) {
			time.Sleep(300 * time.Millisecond)
			parseErrors := validateSignup(parseValue2)
			parseFormError := ""
			if strings.HasSuffix(strings.ToLower(parseValue2.Email), "@blocked.test") {
				parseFormError = "This domain is blocked for the demo"
			}
			return parseErrors, parseFormError
		}, func(isValid bool) {
			if isValid {
				parseStatus.Set("Async validation passed.")
			} else {
				parseStatus.Set("Async validation returned field or form errors.")
			}
		})
	})
	parseSubmit := ui.UseEvent(func(parseEvent4 ui.FormEvent) {
		parseEvent4.PreventDefault()
		if !parseForm.Validate(validateSignup) {
			parseStatus.Set("Fix validation errors before submitting.")
			return
		}
		parseStatus.Set("Submitting...")
		parseForm.Submit(func(parseValue3 signupForm) error {
			time.Sleep(350 * time.Millisecond)
			if strings.HasSuffix(strings.ToLower(parseValue3.Email), "@blocked.test") {
				return fmt.Errorf("server rejected %s", parseValue3.Email)
			}
			return nil
		})
	})
	reset := ui.UseEvent(func() {
		parseForm.Reset(signupForm{Region: "us"})
		parseStatus.Set("Form reset to initial values.")
	})

	parseValue := parseForm.Get()
	parseRegion := parseValue.Region
	if parseRegion == "" {
		parseRegion = "us"
	}

	parseResult := parseStatus.Get()
	if parseForm.Submitting() {
		parseResult = "Submitting..."
	} else if parseForm.Submitted() {
		parseResult = "Submit succeeded."
	} else if parseForm.SubmitError() != nil {
		parseResult = parseForm.SubmitError().Error()
	} else if parseForm.FormError() != "" {
		parseResult = parseForm.FormError()
	}

	return shared.ExamplePage(
		"ui.UseForm",
		"Manage field values, dirty state, validation, and submission lifecycle together",
		"UseForm keeps multi-field local form bookkeeping in one typed handle: field updates, touched and dirty state, sync and async validation, and async submission status.",
		shared.ExamplePanel("Form fields",
			html.Form(html.Props{OnSubmit: parseSubmit, Class: "mt-3 grid gap-4"},
				html.Div(html.Props{},
					html.Label(html.Props{For: "signup-name", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Name")),
					html.Input(html.Props{ID: "signup-name", Value: parseValue.Name, OnInput: setName, Placeholder: "Ada", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
					html.P(html.Props{Class: "mt-2 text-sm text-rose-300"}, html.Text(parseForm.Error("Name"))),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: "signup-email", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Email")),
					html.Input(html.Props{ID: "signup-email", Value: parseValue.Email, OnInput: setEmail, Placeholder: "ada@example.com", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
					html.P(html.Props{Class: "mt-2 text-sm text-rose-300"}, html.Text(parseForm.Error("Email"))),
				),
				html.Label(html.Props{Class: "flex items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
					html.Input(html.Props{Type: "checkbox", Checked: parseValue.OptIn, OnChange: parseToggleOptIn}),
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
				shared.ExampleButton("Validate", parseValidateSync),
				shared.ExampleButton("Async validate", parseValidateAsync),
				shared.ExampleButton("Reset", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Touched any", fmt.Sprintf("%t", parseForm.TouchedAny())),
				shared.ExampleStat("Dirty any", fmt.Sprintf("%t", parseForm.DirtyAny())),
				shared.ExampleStat("Validating", fmt.Sprintf("%t", parseForm.Validating())),
				shared.ExampleStat("Submitting", fmt.Sprintf("%t", parseForm.Submitting())),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Submitted", fmt.Sprintf("%t", parseForm.Submitted())),
				shared.ExampleStat("Has errors", fmt.Sprintf("%t", parseForm.HasErrors())),
				shared.ExampleStat("Region", parseRegion),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseResult)),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(useFormExample))
	exampleboot.WaitExampleRuntime()
}
