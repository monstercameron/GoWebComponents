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

func InputField(parseProps InputFieldProps) ui.Node {
	parseBorderClass := "border-white/10 focus:border-blue-500 focus:ring-blue-500"
	if parseProps.ErrorMsg != "" {
		parseBorderClass = "border-red-500/50 text-red-400 placeholder-red-300 focus:border-red-500 focus:ring-red-500"
	}

	return html.Div(html.Props{Class: "mb-5"},
		html.Label(html.Props{Class: "block text-sm font-medium text-gray-400 mb-2"}, html.Text(parseProps.Label)),
		html.Input(html.Props{
			Type:        parseProps.InputType,
			Name:        parseProps.Name,
			Value:       parseProps.Value,
			Class:       "block w-full px-4 py-3 bg-black/20 border rounded-lg shadow-sm focus:outline-none focus:ring-1 sm:text-sm text-white placeholder-gray-600 transition-all " + parseBorderClass,
			OnInput:     ui.UseEvent(parseProps.OnChange),
			Placeholder: "Enter " + strings.ToLower(parseProps.Label),
		}),
		func() ui.Node {
			if parseProps.ErrorMsg != "" {
				return html.P(html.Props{Class: "mt-2 text-sm text-red-400"}, html.Text(parseProps.ErrorMsg))
			}
			return nil
		}(),
	)
}

func formPage(parseRegistration ui.Form[FormData], parseValidate func(FormData) ui.FieldErrors) ui.Node {
	parseNav := router.UseNavigate()
	parseCurrentForm := parseRegistration.Get()
	parseCurrentErrors := parseRegistration.Errors()

	handleChange := func(parseField string) func(ui.Event) {
		return func(parseE ui.Event) {
			parseRegistration.SetField(parseField, parseE.GetValue())
		}
	}

	handleSubmit := ui.UseEvent(func(parseE2 ui.Event) {
		parseE2.PreventDefault()

		if parseRegistration.Validate(parseValidate) {
			parseRegistration.ValidateAsync(func(parseValue FormData) (ui.FieldErrors, string) {
				time.Sleep(40 * time.Millisecond)
				parseErrs := ui.FieldErrors{}
				if strings.EqualFold(parseValue.Username, "admin") {
					parseErrs["Username"] = "Username is reserved"
				}
				if strings.HasSuffix(strings.ToLower(strings.TrimSpace(parseValue.Email)), "@blocked.test") {
					return parseErrs, "Registrations from blocked.test are disabled"
				}
				return parseErrs, ""
			}, func(isValid bool) {
				if !isValid {
					return
				}
				parseRegistration.Submit(func(parseValue2 FormData) error {
					time.Sleep(40 * time.Millisecond)
					if strings.HasSuffix(strings.ToLower(strings.TrimSpace(parseValue2.Email)), "@retry.test") {
						return errors.New("Temporary signup outage. Please retry.")
					}
					parseNav.Replace("/success")
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
						if parseRegistration.FormError() == "" {
							return nil
						}
						return html.Div(html.Props{Class: "mb-5 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200"}, html.Text(parseRegistration.FormError()))
					}(),
					ui.CreateElement(InputField, InputFieldProps{Label: "Username", Name: "username", InputType: "text", Value: parseCurrentForm.Username, ErrorMsg: parseCurrentErrors["Username"], OnChange: handleChange("Username")}),
					ui.CreateElement(InputField, InputFieldProps{Label: "Email Address", Name: "email", InputType: "email", Value: parseCurrentForm.Email, ErrorMsg: parseCurrentErrors["Email"], OnChange: handleChange("Email")}),
					ui.CreateElement(InputField, InputFieldProps{Label: "Password", Name: "password", InputType: "password", Value: parseCurrentForm.Password, ErrorMsg: parseCurrentErrors["Password"], OnChange: handleChange("Password")}),
					ui.CreateElement(InputField, InputFieldProps{Label: "Confirm Password", Name: "confirm_password", InputType: "password", Value: parseCurrentForm.ConfirmPass, ErrorMsg: parseCurrentErrors["ConfirmPass"], OnChange: handleChange("ConfirmPass")}),
					html.Div(html.Props{Class: "mb-5"},
						html.Label(html.Props{Class: "block text-sm font-medium text-gray-400 mb-2"}, html.Text("Account Type")),
						html.Select(html.Props{Class: "block w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 sm:text-sm text-white", Value: parseCurrentForm.AccountType, OnChange: ui.UseEvent(func(parseE3 ui.Event) {
							parseRegistration.SetField("AccountType", parseE3.GetValue())
						})},
							html.Option(html.Props{Value: "personal", Selected: parseCurrentForm.AccountType == "personal"}, html.Text("Personal")),
							html.Option(html.Props{Value: "business", Selected: parseCurrentForm.AccountType == "business"}, html.Text("Business")),
							html.Option(html.Props{Value: "enterprise", Selected: parseCurrentForm.AccountType == "enterprise"}, html.Text("Enterprise")),
						),
					),
					html.Div(html.Props{Class: "flex items-center mb-8"},
						html.Input(html.Props{ID: "newsletter", Type: "checkbox", Class: "h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-600 rounded bg-black/20", Checked: parseCurrentForm.Newsletter, OnChange: ui.UseEvent(func(parseE4 ui.Event) {
							parseRegistration.SetField("Newsletter", parseE4.IsChecked())
						})}),
						html.Label(html.Props{For: "newsletter", Class: "ml-2 block text-sm text-gray-300"}, html.Text("Subscribe to our newsletter")),
					),
					html.Div(html.Props{Class: "mb-4 text-xs text-slate-400"}, html.Text(func() string {
						if parseRegistration.Validating() {
							return "Running async validation"
						}
						if parseRegistration.DirtyAny() {
							return "Form has unsaved changes"
						}
						return "Form is pristine"
					}())),
					html.Button(html.Props{Type: "button", OnClick: handleSubmit, Disabled: parseRegistration.Submitting() || parseRegistration.Validating(), Class: "w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-purple-500/20"}, html.Text(func() string {
						if parseRegistration.Validating() {
							return "Checking Account..."
						}
						if parseRegistration.Submitting() {
							return "Creating Account..."
						}
						if parseRegistration.SubmitError() != nil {
							return "Retry Sign Up"
						}
						return "Sign Up"
					}())),
				),
			),
		),
	)
}

func successPage(parseRegistration ui.Form[FormData]) ui.Node {
	parseNav := router.UseNavigate()
	parseCurrentForm := parseRegistration.Get()
	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] flex flex-col justify-center py-12 sm:px-6 lg:px-8"},
		html.Div(html.Props{Class: "mt-8 sm:mx-auto sm:w-full sm:max-w-md"},
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 py-8 px-4 shadow-2xl sm:rounded-xl sm:px-10 text-center backdrop-blur-sm"},
				html.Div(html.Props{Class: "mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-green-500/20 mb-6"},
					html.Span(html.Props{Class: "text-green-400 text-2xl"}, html.Text("✓")),
				),
				html.H3(html.Props{Class: "text-xl font-bold text-white mb-2"}, html.Text("Registration Successful!")),
				html.P(html.Props{Class: "mt-2 text-sm text-gray-400"}, html.Text("Welcome aboard, "+parseCurrentForm.Username)),
				html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Route-integrated success state")),
				html.Button(html.Props{
					Class: "mt-8 w-full inline-flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all",
					OnClick: ui.UseEvent(func() {
						parseRegistration.Reset(defaultFormData)
						parseNav.Replace("/")
					}),
				}, html.Text("Register Another Account")),
			),
		),
	)
}

func App() ui.Node {
	parseRegistration := ui.UseForm(defaultFormData)
	parseValidate := func(parseData FormData) ui.FieldErrors {
		parseErrs := ui.FieldErrors{}

		if len(parseData.Username) < 3 {
			parseErrs["Username"] = "Username must be at least 3 characters"
		}
		if !strings.Contains(parseData.Email, "@") {
			parseErrs["Email"] = "Please enter a valid email"
		}
		if len(parseData.Password) < 6 {
			parseErrs["Password"] = "Password must be at least 6 characters"
		}
		if parseData.Password != parseData.ConfirmPass {
			parseErrs["ConfirmPass"] = "Passwords do not match"
		}

		return parseErrs
	}

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", func(_ router.Attrs) *router.Element {
		return formPage(parseRegistration, parseValidate)
	})
	parseR.Register("/success", func(_ router.Attrs) *router.Element {
		return successPage(parseRegistration)
	})
	parseR.Register("*", func(_ router.Attrs) *router.Element {
		return formPage(parseRegistration, parseValidate)
	})
	return parseR.Current()
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
