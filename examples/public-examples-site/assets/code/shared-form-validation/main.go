//go:build js && wasm
// +build js,wasm

package main

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

// Signup is the SHARED validation schema. The same struct, with the same `validate:"..."` tags,
// is validated here in the browser (via ui.Form.ValidateStruct, which runs validate.Struct on
// wasm) AND in a server handler (validate.Struct(req)). There is no second, hand-written client
// validator that can drift from the server's rules — one struct, one set of rules, both sides.
type Signup struct {
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=13"`
}

// SharedValidationExample renders a form whose client-side validation is the exact same schema a
// server handler would run.
func SharedValidationExample() ui.Node {
	form := ui.UseForm(Signup{})
	submitted := ui.UseState(false)

	onEmail := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Email", event.GetValue()) })
	onAge := ui.UseEvent(func(event ui.InputEvent) {
		// The <input> value is a string; Age is an int, so parse before SetField (no implicit
		// coercion). An unparseable value resolves to 0, which fails the gte=13 rule as expected.
		age, _ := strconv.Atoi(strings.TrimSpace(event.GetValue()))
		form.SetField("Age", age)
	})
	onSubmit := ui.UseEvent(func() {
		// ValidateStruct runs validate.Struct on the SAME Signup type the server validates.
		submitted.Set(form.ValidateStruct())
	})

	return Main(ClassStr("mx-auto max-w-md space-y-3 p-6"),
		H1("Shared form validation"),
		P(ClassStr("text-slate-500"), Text("One Signup struct + validate tags; validated identically on client and server.")),
		Div(ClassStr("space-y-1"),
			Label(For("email"), "Email"),
			Input(ID("email"), Type("email"), OnInput(onEmail)),
			If(form.HasFieldError("Email"), P(ClassStr("text-red-500 text-sm"), Text(form.Error("Email")))),
		),
		Div(ClassStr("space-y-1"),
			Label(For("age"), "Age"),
			Input(ID("age"), Type("number"), OnInput(onAge)),
			If(form.HasFieldError("Age"), P(ClassStr("text-red-500 text-sm"), Text(form.Error("Age")))),
		),
		Button(Type("button"), OnClick(onSubmit), "Sign up"),
		If(submitted.Get() && !form.HasErrors(), P(ClassStr("text-emerald-500"), Text("Valid — the server will accept this (same schema)."))),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(SharedValidationExample))
	exampleboot.WaitExampleRuntime()
}
