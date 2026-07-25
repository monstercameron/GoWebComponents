package ui

import "github.com/monstercameron/GoWebComponents/v5/validate"

// ValidateStruct validates the form's current value against the `validate:"..."`
// struct tags on T and stores the resulting field errors, returning true when the
// value is valid.
//
// It is the turnkey path for shared client/server validation: the SAME struct, with
// the SAME tags, validates here in the browser (via wasm) and in a server handler,
// so the two can never drift — no hand-written client validator required.
//
//	type Signup struct {
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age"   validate:"gte=13"`
//	}
//
//	form := ui.UseForm(Signup{})
//	if form.ValidateStruct() { /* submit; the server re-runs validate.Struct on the same type */ }
func (parseF Form[T]) ValidateStruct() bool {
	return parseF.Validate(func(parseValue T) FieldErrors {
		return FieldErrors(validate.Struct(parseValue).Fields())
	})
}
