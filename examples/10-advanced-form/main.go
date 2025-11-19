//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
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

func InputField(label, name, inputType, value, errorMsg string, onChange func(dom.GoEvent)) *dom.Element {
	borderClass := "border-white/10 focus:border-blue-500 focus:ring-blue-500"
	if errorMsg != "" {
		borderClass = "border-red-500/50 text-red-400 placeholder-red-300 focus:border-red-500 focus:ring-red-500"
	}

	return dom.Div(
		dom.Attrs{"class": "mb-5"},
		dom.Label(
			dom.Attrs{"class": "block text-sm font-medium text-gray-400 mb-2"},
			label,
		),
		dom.Input(dom.Attrs{
			"type":        inputType,
			"name":        name,
			"value":       value,
			"class":       "block w-full px-4 py-3 bg-black/20 border rounded-lg shadow-sm focus:outline-none focus:ring-1 sm:text-sm text-white placeholder-gray-600 transition-all " + borderClass,
			"oninput":     hooks.GoUseFunc(onChange),
			"placeholder": "Enter " + strings.ToLower(label),
		}),
		func() *dom.Element {
			if errorMsg != "" {
				return dom.P(dom.Attrs{"class": "mt-2 text-sm text-red-400"}, errorMsg)
			}
			return nil
		}(),
	)
}

func App(_ dom.Attrs) *dom.Element {
	form, setForm := hooks.UseState(FormData{
		AccountType: "personal",
		Newsletter:  true,
	})

	errors, setErrors := hooks.UseState(FormErrors{})
	isSubmitting, setIsSubmitting := hooks.UseState(false)
	success, setSuccess := hooks.UseState(false)

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

	handleChange := func(field string) func(dom.GoEvent) {
		return func(e dom.GoEvent) {
			val := e.GetValue()
			newForm := form()

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

			setForm(newForm)

			// Clear error for this field
			newErrors := errors()
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
			setErrors(newErrors)
		}
	}

	handleSubmit := hooks.GoUseFunc(func(e dom.GoEvent) {
		e.PreventDefault()

		errs, isValid := validate(form())
		setErrors(errs)

		if isValid {
			setIsSubmitting(true)
			// Simulate API call
			go func() {
				// In a real app, you'd use time.Sleep here, but we can't easily in WASM without blocking
				// So we just set state immediately for this demo, or use a timeout wrapper
				// For simplicity in this demo, we'll just update state
				setIsSubmitting(false)
				setSuccess(true)
			}()
		}
	})

	if success() {
		return dom.Div(
			dom.Attrs{"class": "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
			dom.Div(
				dom.Attrs{"class": "mt-8 sm:mx-auto sm:w-full sm:max-w-md"},
				dom.Div(
					dom.Attrs{"class": "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 text-center backdrop-blur-sm"},
					dom.Div(
						dom.Attrs{"class": "mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-green-500/20 mb-6"},
						dom.Span(dom.Attrs{"class": "text-green-400 text-2xl"}, "✓"),
					),
					dom.H3(dom.Attrs{"class": "text-xl font-bold text-white mb-2"}, "Registration Successful!"),
					dom.P(dom.Attrs{"class": "mt-2 text-sm text-gray-400"}, "Welcome aboard, "+form().Username),
					dom.Button(
						dom.Attrs{
							"class": "mt-8 w-full inline-flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all",
							"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
								setSuccess(false)
								setForm(FormData{AccountType: "personal", Newsletter: true})
							}),
						},
						"Register Another Account",
					),
				),
			),
		)
	}

	return dom.Div(
		dom.Attrs{"class": "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
		dom.Div(
			dom.Attrs{"class": "sm:mx-auto sm:w-full sm:max-w-md"},
			dom.Div(
				dom.Attrs{"class": "text-center mb-8"},
				dom.H2(dom.Attrs{"class": "text-3xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"}, "Create your account"),
				dom.P(dom.Attrs{"class": "mt-2 text-sm text-gray-400"}, "Join our community today"),
			),
			dom.Div(
				dom.Attrs{"class": "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 backdrop-blur-sm"},
				dom.Form(
					dom.Attrs{"onsubmit": handleSubmit},

					InputField("Username", "username", "text", form().Username, errors().Username, handleChange("Username")),
					InputField("Email Address", "email", "email", form().Email, errors().Email, handleChange("Email")),
					InputField("Password", "password", "password", form().Password, errors().Password, handleChange("Password")),
					InputField("Confirm Password", "confirm_password", "password", form().ConfirmPass, errors().ConfirmPass, handleChange("ConfirmPass")),

					dom.Div(
						dom.Attrs{"class": "mb-5"},
						dom.Label(dom.Attrs{"class": "block text-sm font-medium text-gray-400 mb-2"}, "Account Type"),
						dom.Select(
							dom.Attrs{
								"class": "block w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 sm:text-sm text-white",
								"onchange": hooks.GoUseFunc(func(e dom.GoEvent) {
									newForm := form()
									newForm.AccountType = e.GetValue()
									setForm(newForm)
								}),
							},
							dom.Option(dom.Attrs{"value": "personal", "selected": fmt.Sprintf("%v", form().AccountType == "personal")}, "Personal"),
							dom.Option(dom.Attrs{"value": "business", "selected": fmt.Sprintf("%v", form().AccountType == "business")}, "Business"),
							dom.Option(dom.Attrs{"value": "enterprise", "selected": fmt.Sprintf("%v", form().AccountType == "enterprise")}, "Enterprise"),
						),
					),

					dom.Div(
						dom.Attrs{"class": "flex items-center mb-8"},
						dom.Input(dom.Attrs{
							"id":      "newsletter",
							"type":    "checkbox",
							"class":   "h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-600 rounded bg-black/20",
							"checked": fmt.Sprintf("%v", form().Newsletter),
							"onchange": hooks.GoUseFunc(func(e dom.GoEvent) {
								newForm := form()
								newForm.Newsletter = e.IsChecked()
								setForm(newForm)
							}),
						}),
						dom.Label(
							dom.Attrs{"for": "newsletter", "class": "ml-2 block text-sm text-gray-300"},
							"Subscribe to our newsletter",
						),
					),

					dom.Button(
						dom.Attrs{
							"type":     "submit",
							"class":    "w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-purple-500/20",
							"disabled": fmt.Sprintf("%v", isSubmitting()),
						},
						func() string {
							if isSubmitting() {
								return "Creating Account..."
							}
							return "Sign Up"
						}(),
					),
				),
			),
		),
	)
}

func main() {
	render.To(dom.CreateElement(App, nil), "body")
}
