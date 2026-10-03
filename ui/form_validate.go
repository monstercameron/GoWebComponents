package ui

import (
	"fmt"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/v6/internal/errcode"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/validate"
)

// formValidateSeqCounter issues validation sequence numbers that are unique for the life of the
// process. ValidateAsync discards a result whose number no longer matches the form's current one,
// so a number must never be handed out twice: deriving it from the form's previous number, or
// resetting it to zero, let a validation that started before Reset collide with one started after.
var formValidateSeqCounter atomic.Int64

// nextFormValidateSeq returns a never-before-used validation sequence number.
func nextFormValidateSeq() int {
	return int(formValidateSeqCounter.Add(1))
}

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

// callFormRunner runs a submit callback and converts a panic into an error. The form's goroutines only
// clear Submitting once the callback returns, so a panic that merely unwound the goroutine left the
// form locked in the submitting state for good.
func callFormRunner(parseSubject string, parseRun func() error) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			runtime.ContainPanic("ui", runtime.PanicPhaseAsync, parseSubject, parseRecovered)
			parseErr = errcode.New(errcode.RuntimePanicAsync, "ui: "+parseSubject+" panicked", parseRecovered)
		}
	}()
	return parseRun()
}

// callFormValidator runs an async validator and converts a panic into a form-level error, so the
// form leaves the validating state instead of waiting on a result that will never arrive.
func callFormValidator[T any](parseValidate func(T) (FieldErrors, string), parseValue T) (parseErrors FieldErrors, parseFormError string) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			runtime.ContainPanic("ui", runtime.PanicPhaseAsync, "UseForm validator", parseRecovered)
			parseErrors = nil
			parseFormError = fmt.Sprintf("validation failed: %v", parseRecovered)
		}
	}()
	return parseValidate(parseValue)
}
