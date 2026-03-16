//go:build js && wasm
// +build js,wasm

package main

import (
	"errors"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
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

type InputFieldProps struct {
	Label     string
	Name      string
	InputType string
	Value     string
	ErrorMsg  string
	OnChange  func(ui.Event)
}

var defaultFormData = FormData{AccountType: "personal", Newsletter: true}

func InputField(props InputFieldProps) ui.Node {
	borderClass := "border-white/10 focus:border-blue-500 focus:ring-blue-500"
	if props.ErrorMsg != "" {
		borderClass = "border-red-500/50 text-red-400 placeholder-red-300 focus:border-red-500 focus:ring-red-500"
	}

	return html.Div(html.Props{Class: "mb-5"},
		html.Label(html.Props{Class: "block text-sm font-medium text-gray-400 mb-2"}, html.Text(props.Label)),
		html.Input(html.Props{
			Type:        props.InputType,
			Name:        props.Name,
			Value:       props.Value,
			Class:       "block w-full px-4 py-3 bg-black/20 border rounded-lg shadow-sm focus:outline-none focus:ring-1 sm:text-sm text-white placeholder-gray-600 transition-all " + borderClass,
			OnInput:     ui.UseEvent(props.OnChange),
			Placeholder: "Enter " + strings.ToLower(props.Label),
		}),
		func() ui.Node {
			if props.ErrorMsg != "" {
				return html.P(html.Props{Class: "mt-2 text-sm text-red-400"}, html.Text(props.ErrorMsg))
			}
			return nil
		}(),
	)
}

func formPage(registration ui.Form[FormData], validate func(FormData) ui.FieldErrors) ui.Node {
	nav := router.UseNavigate()
	currentForm := registration.Get()
	currentErrors := registration.Errors()

	handleChange := func(field string) func(ui.Event) {
		return func(e ui.Event) {
			registration.SetField(field, e.GetValue())
		}
	}

	handleSubmit := ui.UseEvent(func(e ui.Event) {
		e.PreventDefault()

		if registration.Validate(validate) {
			registration.ValidateAsync(func(value FormData) (ui.FieldErrors, string) {
				time.Sleep(40 * time.Millisecond)
				errs := ui.FieldErrors{}
				if strings.EqualFold(value.Username, "admin") {
					errs["Username"] = "Username is reserved"
				}
				if strings.HasSuffix(strings.ToLower(strings.TrimSpace(value.Email)), "@blocked.test") {
					return errs, "Registrations from blocked.test are disabled"
				}
				return errs, ""
			}, func(valid bool) {
				if !valid {
					return
				}
				registration.Submit(func(value FormData) error {
					time.Sleep(40 * time.Millisecond)
					if strings.HasSuffix(strings.ToLower(strings.TrimSpace(value.Email)), "@retry.test") {
						return errors.New("Temporary signup outage. Please retry.")
					}
					nav.Replace("/success")
					return nil
				})
			})
		}
	})

	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
		html.Div(html.Props{Class: "sm:mx-auto sm:w-full sm:max-w-md"},
			html.Div(html.Props{Class: "text-center mb-8"},
				html.H2(html.Props{Class: "text-3xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"}, html.Text("Create your account")),
				html.P(html.Props{Class: "mt-2 text-sm text-gray-400"}, html.Text("Join our community today")),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 backdrop-blur-sm"},
				html.Form(html.Props{},
					func() ui.Node {
						if registration.FormError() == "" {
							return nil
						}
						return html.Div(html.Props{Class: "mb-5 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200"}, html.Text(registration.FormError()))
					}(),
					ui.CreateElement(InputField, InputFieldProps{Label: "Username", Name: "username", InputType: "text", Value: currentForm.Username, ErrorMsg: currentErrors["Username"], OnChange: handleChange("Username")}),
					ui.CreateElement(InputField, InputFieldProps{Label: "Email Address", Name: "email", InputType: "email", Value: currentForm.Email, ErrorMsg: currentErrors["Email"], OnChange: handleChange("Email")}),
					ui.CreateElement(InputField, InputFieldProps{Label: "Password", Name: "password", InputType: "password", Value: currentForm.Password, ErrorMsg: currentErrors["Password"], OnChange: handleChange("Password")}),
					ui.CreateElement(InputField, InputFieldProps{Label: "Confirm Password", Name: "confirm_password", InputType: "password", Value: currentForm.ConfirmPass, ErrorMsg: currentErrors["ConfirmPass"], OnChange: handleChange("ConfirmPass")}),
					html.Div(html.Props{Class: "mb-5"},
						html.Label(html.Props{Class: "block text-sm font-medium text-gray-400 mb-2"}, html.Text("Account Type")),
						html.Select(html.Props{Class: "block w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 sm:text-sm text-white", Value: currentForm.AccountType, OnChange: ui.UseEvent(func(e ui.Event) {
							registration.SetField("AccountType", e.GetValue())
						})},
							html.Option(html.Props{Value: "personal", Selected: currentForm.AccountType == "personal"}, html.Text("Personal")),
							html.Option(html.Props{Value: "business", Selected: currentForm.AccountType == "business"}, html.Text("Business")),
							html.Option(html.Props{Value: "enterprise", Selected: currentForm.AccountType == "enterprise"}, html.Text("Enterprise")),
						),
					),
					html.Div(html.Props{Class: "flex items-center mb-8"},
						html.Input(html.Props{ID: "newsletter", Type: "checkbox", Class: "h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-600 rounded bg-black/20", Checked: currentForm.Newsletter, OnChange: ui.UseEvent(func(e ui.Event) {
							registration.SetField("Newsletter", e.IsChecked())
						})}),
						html.Label(html.Props{For: "newsletter", Class: "ml-2 block text-sm text-gray-300"}, html.Text("Subscribe to our newsletter")),
					),
					html.Div(html.Props{Class: "mb-4 text-xs text-slate-400"}, html.Text(func() string {
						if registration.Validating() {
							return "Running async validation"
						}
						if registration.DirtyAny() {
							return "Form has unsaved changes"
						}
						return "Form is pristine"
					}())),
					html.Button(html.Props{Type: "button", OnClick: handleSubmit, Disabled: registration.Submitting() || registration.Validating(), Class: "w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-purple-500/20"}, html.Text(func() string {
						if registration.Validating() {
							return "Checking Account..."
						}
						if registration.Submitting() {
							return "Creating Account..."
						}
						if registration.SubmitError() != nil {
							return "Retry Sign Up"
						}
						return "Sign Up"
					}())),
				),
			),
		),
	)
}

func successPage(registration ui.Form[FormData]) ui.Node {
	nav := router.UseNavigate()
	currentForm := registration.Get()
	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
		html.Div(html.Props{Class: "mt-8 sm:mx-auto sm:w-full sm:max-w-md"},
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 text-center backdrop-blur-sm"},
				html.Div(html.Props{Class: "mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-green-500/20 mb-6"},
					html.Span(html.Props{Class: "text-green-400 text-2xl"}, html.Text("✓")),
				),
				html.H3(html.Props{Class: "text-xl font-bold text-white mb-2"}, html.Text("Registration Successful!")),
				html.P(html.Props{Class: "mt-2 text-sm text-gray-400"}, html.Text("Welcome aboard, "+currentForm.Username)),
				html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Route-integrated success state")),
				html.Button(html.Props{
					Class: "mt-8 w-full inline-flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all",
					OnClick: ui.UseEvent(func() {
						registration.Reset(defaultFormData)
						nav.Replace("/")
					}),
				}, html.Text("Register Another Account")),
			),
		),
	)
}

func App() ui.Node {
	registration := ui.UseForm(defaultFormData)
	validate := func(data FormData) ui.FieldErrors {
		errs := ui.FieldErrors{}

		if len(data.Username) < 3 {
			errs["Username"] = "Username must be at least 3 characters"
		}
		if !strings.Contains(data.Email, "@") {
			errs["Email"] = "Please enter a valid email"
		}
		if len(data.Password) < 6 {
			errs["Password"] = "Password must be at least 6 characters"
		}
		if data.Password != data.ConfirmPass {
			errs["ConfirmPass"] = "Passwords do not match"
		}

		return errs
	}

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", func(_ router.Attrs) *router.Element {
		return formPage(registration, validate)
	})
	r.Register("/success", func(_ router.Attrs) *router.Element {
		return successPage(registration)
	})
	r.Register("*", func(_ router.Attrs) *router.Element {
		return formPage(registration, validate)
	})
	return r.Current()
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
