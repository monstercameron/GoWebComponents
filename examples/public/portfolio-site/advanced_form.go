//go:build js && wasm

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// AdvancedFormExample showcases sophisticated form handling with real-time validation.
// Demonstrates UseState, UseEffect, UseFetch, and UseEvent hooks working together
// to create a production-ready form with password strength, async submission, and error handling.
func AdvancedFormExample(_ Attrs) *Element {
	// Form field state management
	parseUsername, setUsername := UseState("")
	parseEmail, setEmail := UseState("")
	parsePassword, setPassword := UseState("")
	parseConfirmPass, setConfirmPass := UseState("")
	parseBio, setBio := UseState("")
	parseUserType, setUserType := UseState("developer")
	parseAgreeTerms, setAgreeTerms := UseState(false)

	// Form submission and validation state
	isSubmitting, setIsSubmitting := UseState(false)
	parseSubmitStatus, setSubmitStatus := UseState("")
	parsePasswordStrength, setPasswordStrength := UseState(0)
	parseFieldClass := "w-full rounded-md border border-white/10 bg-white/5 px-3 py-2 text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-400"
	parseFieldErrorClass := "w-full rounded-md border border-red-400/70 bg-red-500/10 px-3 py-2 text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-red-400"

	// Calculate password strength when password changes
	UseEffect(func() func() {
		parsePass := parsePassword()
		if parsePass == "" {
			setPasswordStrength(0)
			return nil
		}

		parseStrength := 0
		if len(parsePass) >= 8 {
			parseStrength += 25
		}
		if regexp.MustCompile(`[a-z]`).MatchString(parsePass) {
			parseStrength += 25
		}
		if regexp.MustCompile(`[A-Z]`).MatchString(parsePass) {
			parseStrength += 25
		}
		if regexp.MustCompile(`[0-9]`).MatchString(parsePass) {
			parseStrength += 12
		}
		if regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(parsePass) {
			parseStrength += 13
		}
		if parseStrength > 100 {
			parseStrength = 100
		}
		setPasswordStrength(parseStrength)
		return nil
	}, parsePassword())

	// Input handlers
	handleUsernameChange := UseEvent(func(parseEvent InputEvent) {
		parseValue := parseEvent.GetValue()
		setUsername(parseValue)
	})

	handleEmailChange := UseEvent(func(parseEvent2 InputEvent) {
		parseValue2 := parseEvent2.GetValue()
		setEmail(parseValue2)
	})

	handlePasswordChange := UseEvent(func(parseEvent3 InputEvent) {
		parseValue3 := parseEvent3.GetValue()
		setPassword(parseValue3)
	})

	handleConfirmPassChange := UseEvent(func(parseEvent4 InputEvent) {
		parseValue4 := parseEvent4.GetValue()
		setConfirmPass(parseValue4)
	})

	handleBioChange := UseEvent(func(parseEvent5 InputEvent) {
		parseValue5 := parseEvent5.GetValue()
		setBio(parseValue5)
	})

	handleUserTypeChange := UseEvent(func(parseEvent6 ChangeEvent) {
		parseValue6 := parseEvent6.GetValue()
		setUserType(parseValue6)
	})

	handleTermsChange := UseEvent(func(parseEvent7 ChangeEvent) {
		parseChecked := parseEvent7.IsChecked()
		setAgreeTerms(parseChecked)
	})

	// Form submission
	handleSubmit := UseEvent(func(parseEvent8 FormEvent) {
		parseEvent8.PreventDefault()
		setIsSubmitting(true)
		setSubmitStatus("")

		// Simulate async submission
		go func() {
			time.Sleep(2 * time.Second)

			// Validate form
			parseUsernameVal := parseUsername()
			parseEmailVal := parseEmail()
			parsePasswordVal := parsePassword()
			parseConfirmPassVal := parseConfirmPass()
			parseAgreeTermsVal := parseAgreeTerms()

			if parseUsernameVal == "" || parseEmailVal == "" || parsePasswordVal == "" {
				setSubmitStatus("error")
				setIsSubmitting(false)
				return
			}

			if parsePasswordVal != parseConfirmPassVal {
				setSubmitStatus("error")
				setIsSubmitting(false)
				return
			}

			if !parseAgreeTermsVal {
				setSubmitStatus("error")
				setIsSubmitting(false)
				return
			}

			// Success
			setSubmitStatus("success")
			setIsSubmitting(false)

			// Reset form
			setUsername("")
			setEmail("")
			setPassword("")
			setConfirmPass("")
			setBio("")
			setUserType("developer")
			setAgreeTerms(false)
		}()
	})

	// Helper functions for validation display
	parseUsernameError := func() string {
		parseVal := parseUsername()
		if parseVal == "" {
			return ""
		}
		if len(parseVal) < 3 || len(parseVal) > 20 {
			return "Username must be 3-20 characters"
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(parseVal) {
			return "Username can only contain letters, numbers, and underscores"
		}
		return ""
	}

	parseEmailError := func() string {
		parseVal2 := parseEmail()
		if parseVal2 == "" {
			return ""
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(parseVal2) {
			return "Please enter a valid email address"
		}
		return ""
	}

	parsePasswordError := func() string {
		parseVal3 := parsePassword()
		if parseVal3 == "" {
			return ""
		}
		if len(parseVal3) < 8 {
			return "Password must be at least 8 characters"
		}
		return ""
	}

	parseConfirmPassError := func() string {
		parseVal4 := parseConfirmPass()
		parsePassVal := parsePassword()
		if parseVal4 == "" {
			return ""
		}
		if parseVal4 != parsePassVal {
			return "Passwords do not match"
		}
		return ""
	}

	return Div(
		Attrs{"class": "rounded-2xl border border-white/10 bg-slate-950/70 p-8 shadow-2xl backdrop-blur-sm"},

		// Header
		Div(
			Attrs{"class": "text-center mb-8"},
			H3(Attrs{"class": "mb-2 text-2xl font-bold text-slate-100"}, "🔧 Advanced Form Example"),
			P(Attrs{"class": "text-slate-400"}, "Comprehensive form handling with validation, state management, and async processing"),
		),

		// Form
		Form(
			Attrs{"class": "space-y-6", "onsubmit": handleSubmit},

			// Username field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Username"),
				Input(Attrs{
					"type":    "text",
					"value":   parseUsername(),
					"oninput": handleUsernameChange,
					"class": func() string {
						if parseUsernameError() != "" {
							return parseFieldErrorClass
						}
						return parseFieldClass
					}(),
					"placeholder": "Enter username",
				}),
				func() *Element {
					if parseErr := parseUsernameError(); parseErr != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, parseErr)
					}
					return Text("")
				}(),
			),

			// Email field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Email"),
				Input(Attrs{
					"type":    "email",
					"value":   parseEmail(),
					"oninput": handleEmailChange,
					"class": func() string {
						if parseEmailError() != "" {
							return parseFieldErrorClass
						}
						return parseFieldClass
					}(),
					"placeholder": "Enter email",
				}),
				func() *Element {
					if parseErr2 := parseEmailError(); parseErr2 != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, parseErr2)
					}
					return Text("")
				}(),
			),

			// Password field with strength meter
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Password"),
				Input(Attrs{
					"type":    "password",
					"value":   parsePassword(),
					"oninput": handlePasswordChange,
					"class": func() string {
						if parsePasswordError() != "" {
							return parseFieldErrorClass
						}
						return parseFieldClass
					}(),
					"placeholder": "Enter password",
				}),

				// Password strength meter
				func() *Element {
					if parsePassword() != "" {
						parseStrength2 := parsePasswordStrength()
						return Div(
							Attrs{"class": "mt-2"},
							Div(Attrs{"class": "mb-1 flex justify-between text-xs text-slate-400"},
								Span(nil, "Password Strength"),
								Span(nil, Text(strconv.Itoa(parseStrength2)), "%"),
							),
							Div(Attrs{"class": "h-2 w-full rounded-full bg-white/10"},
								Div(Attrs{
									"class": func() string {
										if parseStrength2 < 30 {
											return "bg-red-500 h-2 rounded-full transition-all duration-300"
										} else if parseStrength2 < 70 {
											return "bg-yellow-500 h-2 rounded-full transition-all duration-300"
										}
										return "bg-green-500 h-2 rounded-full transition-all duration-300"
									}(),
									"style": "width: " + strconv.Itoa(parseStrength2) + "%;",
								}),
							),
						)
					}
					return Text("")
				}(),

				func() *Element {
					if parseErr3 := parsePasswordError(); parseErr3 != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, parseErr3)
					}
					return Text("")
				}(),
			),

			// Confirm Password field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Confirm Password"),
				Input(Attrs{
					"type":    "password",
					"value":   parseConfirmPass(),
					"oninput": handleConfirmPassChange,
					"class": func() string {
						if parseConfirmPassError() != "" {
							return parseFieldErrorClass
						}
						return parseFieldClass
					}(),
					"placeholder": "Confirm password",
				}),
				func() *Element {
					if parseErr4 := parseConfirmPassError(); parseErr4 != "" {
						return P(Attrs{"class": "text-sm text-red-600"}, parseErr4)
					}
					return Text("")
				}(),
			),

			// Bio field
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "Bio (optional)"),
				Textarea(Attrs{
					"value":       parseBio(),
					"oninput":     handleBioChange,
					"rows":        "4",
					"class":       parseFieldClass,
					"placeholder": "Tell us about yourself...",
				}),
				Div(
					Attrs{"class": "flex justify-between text-xs text-slate-500"},
					Span(nil, Text("Optional")),
					Span(nil, Text(fmt.Sprintf("%d/500 characters", len(parseBio())))),
				),
			),

			// User type selection
			Div(
				Attrs{"class": "space-y-2"},
				Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, "User Type"),
				Select(Attrs{
					"value":    parseUserType(),
					"onchange": handleUserTypeChange,
					"class":    parseFieldClass,
				},
					Option(Attrs{"value": "developer"}, "Developer"),
					Option(Attrs{"value": "designer"}, "Designer"),
					Option(Attrs{"value": "manager"}, "Project Manager"),
					Option(Attrs{"value": "other"}, "Other"),
				),
			),

			// Terms agreement
			Div(
				Attrs{"class": "flex items-center space-x-2"},
				Input(Attrs{
					"type":     "checkbox",
					"checked":  parseAgreeTerms(),
					"onchange": handleTermsChange,
					"class":    "h-4 w-4 rounded border-white/20 bg-slate-900 text-indigo-500 focus:ring-indigo-400 focus:ring-offset-0",
				}),
				Label(Attrs{"class": "text-sm text-slate-300"}, "I agree to the Terms and Conditions"),
			),

			// Submit button and status
			Div(
				Attrs{"class": "space-y-4"},
				Button(
					Attrs{
						"type":     "submit",
						"disabled": isSubmitting() || !parseAgreeTerms(),
						"class": func() string {
							if isSubmitting() || !parseAgreeTerms() {
								return "w-full cursor-not-allowed rounded-md border border-white/10 bg-slate-900 px-4 py-3 text-slate-500"
							}
							return "w-full rounded-md bg-indigo-500 px-4 py-3 text-white transition-colors hover:bg-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-400"
						}(),
					},
					func() string {
						if isSubmitting() {
							return "⏳ Submitting..."
						}
						return "✅ Create Account"
					}(),
				),

				// Status messages
				func() *Element {
					parseStatus := parseSubmitStatus()
					switch parseStatus {
					case "success":
						return Div(Attrs{"class": "rounded-md border border-emerald-400/40 bg-emerald-500/10 p-4 text-emerald-200"},
							P(Attrs{"class": "font-semibold"}, "✅ Success!"),
							P(nil, "Form submitted successfully. All data has been validated and processed."),
						)
					case "error":
						return Div(Attrs{"class": "rounded-md border border-red-400/40 bg-red-500/10 p-4 text-red-200"},
							P(Attrs{"class": "font-semibold"}, "❌ Error!"),
							P(nil, "Please fix the validation errors and try again."),
						)
					default:
						return Text("")
					}
				}(),
			),
		),
	)
}

// FormField creates a labeled form field with validation
func FormField(parseLabel string, parseInput *Element, parseErrorMsg string, isValidating bool) *Element {
	return Div(
		Attrs{"class": "space-y-2"},
		Label(Attrs{"class": "block text-sm font-medium text-slate-300"}, parseLabel),
		parseInput,
		func() *Element {
			if isValidating {
				return P(Attrs{"class": "flex items-center text-sm text-cyan-300"},
					Span(Attrs{"class": "mr-2"}, "⏳"),
					"Validating...",
				)
			}
			if parseErrorMsg != "" {
				return P(Attrs{"class": "text-sm text-red-600 flex items-center"},
					Span(Attrs{"class": "mr-2"}, "❌"),
					parseErrorMsg,
				)
			}
			return Div(nil)
		}(),
	)
}

// PasswordStrengthMeter shows password strength visualization
func PasswordStrengthMeter(parseStrength int) *Element {
	var parseStrengthText string
	var parseStrengthColor string
	var parseBarWidth string

	switch {
	case parseStrength < 30:
		parseStrengthText = "Weak"
		parseStrengthColor = "text-red-600"
		parseBarWidth = "25%"
	case parseStrength < 60:
		parseStrengthText = "Fair"
		parseStrengthColor = "text-yellow-600"
		parseBarWidth = "50%"
	case parseStrength < 80:
		parseStrengthText = "Good"
		parseStrengthColor = "text-blue-600"
		parseBarWidth = "75%"
	default:
		parseStrengthText = "Strong"
		parseStrengthColor = "text-green-600"
		parseBarWidth = "100%"
	}

	return Div(
		Attrs{"class": "space-y-1"},
		Div(
			Attrs{"class": "flex justify-between items-center"},
			Span(Attrs{"class": "text-xs text-slate-400"}, "Password Strength:"),
			Span(Attrs{"class": "text-xs font-medium " + parseStrengthColor}, parseStrengthText),
		),
		Div(
			Attrs{"class": "h-2 w-full rounded-full bg-white/10"},
			Div(Attrs{
				"class": "h-2 rounded-full transition-all duration-300 " + getStrengthBarColor(parseStrength),
				"style": "width: " + parseBarWidth,
			}),
		),
	)
}

// Helper functions
func getStrengthBarColor(parseStrength int) string {
	switch {
	case parseStrength < 30:
		return "bg-red-500"
	case parseStrength < 60:
		return "bg-yellow-500"
	case parseStrength < 80:
		return "bg-blue-500"
	default:
		return "bg-green-500"
	}
}
