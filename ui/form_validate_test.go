//go:build !js || !wasm

package ui

import "testing"

type validateStructSignup struct {
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=13"`
}

// TestFormValidateStructRejectsAndPopulatesErrors proves the turnkey path: a struct
// with validate tags fails ValidateStruct and the form's errors are keyed by the
// json field names, with no hand-written validator.
func TestFormValidateStructRejectsAndPopulatesErrors(parseT *testing.T) {
	parseForm := UseForm(validateStructSignup{Email: "not-an-email", Age: 5})

	if parseForm.ValidateStruct() {
		parseT.Fatal("expected an invalid signup to fail ValidateStruct")
	}
	parseErrors := parseForm.Errors()
	if parseErrors["email"] == "" {
		parseT.Fatalf("expected an email error, got %#v", parseErrors)
	}
	if parseErrors["age"] == "" {
		parseT.Fatalf("expected an age error, got %#v", parseErrors)
	}
}

// TestFormValidateStructAcceptsValidValue proves a valid value passes and clears.
func TestFormValidateStructAcceptsValidValue(parseT *testing.T) {
	parseForm := UseForm(validateStructSignup{Email: "cam@example.com", Age: 34})

	if !parseForm.ValidateStruct() {
		parseT.Fatalf("expected a valid signup to pass, errors=%#v", parseForm.Errors())
	}
	if len(parseForm.Errors()) != 0 {
		parseT.Fatalf("expected no errors on a valid value, got %#v", parseForm.Errors())
	}
}
