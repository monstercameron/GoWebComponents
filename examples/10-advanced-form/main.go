//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type FormData struct {
	Username    string
	Email       string
	Password    string
	ConfirmPass string
	AccountType string
	Newsletter  bool
	Bio         string
}

type FormErrors struct {
	Username    string
	Email       string
	Password    string
	ConfirmPass string
}

func InputField(label, name, inputType, value, errorMsg string, onChange func(ui.Event)) ui.Node {
	borderClass := "border-white/10 focus:border-blue-500 focus:ring-blue-500"
	if errorMsg != "" {
		borderClass = "border-red-500/50 text-red-400 placeholder-red-300 focus:border-red-500 focus:ring-red-500"
	}

	return html.Div(html.Props{Class: "mb-5"},
		html.Label(html.Props{Class: "block text-sm font-medium text-gray-400 mb-2"}, html.Text(label)),
		html.Input(html.Props{
			Type:        inputType,
			Name:        name,
			Value:       value,
			Class:       "block w-full px-4 py-3 bg-black/20 border rounded-lg shadow-sm focus:outline-none focus:ring-1 sm:text-sm text-white placeholder-gray-600 transition-all " + borderClass,
			OnInput:     ui.UseEvent(onChange),
			Placeholder: "Enter " + strings.ToLower(label),
		}),
		func() ui.Node {
			if errorMsg != "" {
				return html.P(html.Props{Class: "mt-2 text-sm text-red-400"}, html.Text(errorMsg))
			}
			return nil
		}(),
	)
}

func App() ui.Node {
	form := ui.UseState(FormData{AccountType: "personal", Newsletter: true})
	errors := ui.UseState(FormErrors{})
	isSubmitting := ui.UseState(false)
	success := ui.UseState(false)

	validate := func(data FormData) (FormErrors, bool) {
		errs := FormErrors{}
		isValid := true

		if len(data.Username) < 3 {
			errs.Username = "Username must be at least 3 characters"
			isValid = false
		}
		if !strings.Contains(data.Email, "@") {
			errs.Email = "Please enter a valid email"
			isValid = false
		}
		if len(data.Password) < 6 {
			errs.Password = "Password must be at least 6 characters"
			isValid = false
		}
		if data.Password != data.ConfirmPass {
			errs.ConfirmPass = "Passwords do not match"
			isValid = false
		}

		return errs, isValid
	}

	handleChange := func(field string) func(ui.Event) {
		return func(e ui.Event) {
			val := e.GetValue()
			newForm := form.Get()

			switch field {
			case "Username":
				newForm.Username = val
			case "Email":
				newForm.Email = val
			case "Password":
				newForm.Password = val
			case "ConfirmPass":
				newForm.ConfirmPass = val
			case "Bio":
				newForm.Bio = val
			}

			form.Set(newForm)

			// Clear error for this field
			newErrors := errors.Get()
			switch field {
			case "Username":
				newErrors.Username = ""
			case "Email":
				newErrors.Email = ""
			case "Password":
				newErrors.Password = ""
			case "ConfirmPass":
				newErrors.ConfirmPass = ""
			}
			errors.Set(newErrors)
		}
	}

	handleSubmit := ui.UseEvent(func(e ui.Event) {
		e.PreventDefault()

		errs, isValid := validate(form.Get())
		errors.Set(errs)

		if isValid {
			isSubmitting.Set(true)
			// Simulate API call
			go func() {
				isSubmitting.Set(false)
				success.Set(true)
			}()
		}
	})

	if success.Get() {
		return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
			html.Div(html.Props{Class: "mt-8 sm:mx-auto sm:w-full sm:max-w-md"},
				html.Div(html.Props{Class: "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 text-center backdrop-blur-sm"},
					html.Div(html.Props{Class: "mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-green-500/20 mb-6"},
						html.Span(html.Props{Class: "text-green-400 text-2xl"}, html.Text("✓")),
					),
					html.H3(html.Props{Class: "text-xl font-bold text-white mb-2"}, html.Text("Registration Successful!")),
					html.P(html.Props{Class: "mt-2 text-sm text-gray-400"}, html.Text("Welcome aboard, "+form.Get().Username)),
					html.Button(html.Props{
						Class:   "mt-8 w-full inline-flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all",
						OnClick: ui.UseEvent(func() { success.Set(false); form.Set(FormData{AccountType: "personal", Newsletter: true}) }),
					}, html.Text("Register Another Account")),
				),
			),
		)
	}

	currentForm := form.Get()
	currentErrors := errors.Get()

	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
		html.Div(html.Props{Class: "sm:mx-auto sm:w-full sm:max-w-md"},
			html.Div(html.Props{Class: "text-center mb-8"},
				html.H2(html.Props{Class: "text-3xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"}, html.Text("Create your account")),
				html.P(html.Props{Class: "mt-2 text-sm text-gray-400"}, html.Text("Join our community today")),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 backdrop-blur-sm"},
				html.Form(html.Props{OnSubmit: handleSubmit},
					ui.CreateElement(InputField, "Username", "username", "text", currentForm.Username, currentErrors.Username, handleChange("Username")),
					ui.CreateElement(InputField, "Email Address", "email", "email", currentForm.Email, currentErrors.Email, handleChange("Email")),
					ui.CreateElement(InputField, "Password", "password", "password", currentForm.Password, currentErrors.Password, handleChange("Password")),
					ui.CreateElement(InputField, "Confirm Password", "confirm_password", "password", currentForm.ConfirmPass, currentErrors.ConfirmPass, handleChange("ConfirmPass")),
					html.Div(html.Props{Class: "mb-5"},
						html.Label(html.Props{Class: "block text-sm font-medium text-gray-400 mb-2"}, html.Text("Account Type")),
						html.Select(html.Props{Class: "block w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 sm:text-sm text-white", Value: currentForm.AccountType, OnChange: ui.UseEvent(func(e ui.Event) {
							next := form.Get()
							next.AccountType = e.GetValue()
							form.Set(next)
						})},
							html.Option(html.Props{Value: "personal", Selected: currentForm.AccountType == "personal"}, html.Text("Personal")),
							html.Option(html.Props{Value: "business", Selected: currentForm.AccountType == "business"}, html.Text("Business")),
							html.Option(html.Props{Value: "enterprise", Selected: currentForm.AccountType == "enterprise"}, html.Text("Enterprise")),
						),
					),
					html.Div(html.Props{Class: "flex items-center mb-8"},
						html.Input(html.Props{ID: "newsletter", Type: "checkbox", Class: "h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-600 rounded bg-black/20", Checked: currentForm.Newsletter, OnChange: ui.UseEvent(func(e ui.Event) {
							next := form.Get()
							next.Newsletter = e.IsChecked()
							form.Set(next)
						})}),
						html.Label(html.Props{For: "newsletter", Class: "ml-2 block text-sm text-gray-300"}, html.Text("Subscribe to our newsletter")),
					),
					html.Button(html.Props{Type: "submit", Disabled: isSubmitting.Get(), Class: "w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-purple-500/20"}, html.Text(func() string {
						if isSubmitting.Get() {
							return "Creating Account..."
						}
						return "Sign Up"
					}())),
				),
			),
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(App), "body")
}
