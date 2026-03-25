//go:build js && wasm
// +build js,wasm

package ui

import (
	"reflect"
	"strings"
)

type FieldErrors map[string]string

const DefaultCSRFHeaderName = "X-CSRF-Token"
const DefaultCSRFFormFieldName = "csrf_token"

type CSRFToken struct {
	Value         string
	HeaderName    string
	FormFieldName string
}

type ServerFormErrors struct {
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Fields  FieldErrors `json:"fields,omitempty"`
}

type ServerActionOutcome string

const (
	ServerActionOutcomeSuccess         ServerActionOutcome = "success"
	ServerActionOutcomeRedirect        ServerActionOutcome = "redirect"
	ServerActionOutcomeValidationError ServerActionOutcome = "validation_error"
	ServerActionOutcomeAuthError       ServerActionOutcome = "auth_error"
	ServerActionOutcomeRetryableError  ServerActionOutcome = "retryable_error"
)

type ServerActionRedirect struct {
	Location string `json:"location,omitempty"`
	Replace  bool   `json:"replace,omitempty"`
}

type ServerActionFlash struct {
	Kind    string `json:"kind,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
}

type ServerActionRefresh struct {
	Revalidate bool     `json:"revalidate,omitempty"`
	CacheKeys  []string `json:"cacheKeys,omitempty"`
}

type ServerActionResult struct {
	Outcome  ServerActionOutcome   `json:"outcome,omitempty"`
	Error    string                `json:"error,omitempty"`
	Message  string                `json:"message,omitempty"`
	Fields   FieldErrors           `json:"fields,omitempty"`
	Redirect *ServerActionRedirect `json:"redirect,omitempty"`
	Flash    *ServerActionFlash    `json:"flash,omitempty"`
	Refresh  *ServerActionRefresh  `json:"refresh,omitempty"`
}

type FieldStatus struct {
	Name    string
	Touched bool
	Dirty   bool
	Pending bool
	Error   string
}

func NewCSRFToken(value string) CSRFToken {
	return CSRFToken{
		Value:         value,
		HeaderName:    DefaultCSRFHeaderName,
		FormFieldName: DefaultCSRFFormFieldName,
	}
}

func (t CSRFToken) Header() (string, string) {
	name := strings.TrimSpace(t.HeaderName)
	if name == "" {
		name = DefaultCSRFHeaderName
	}
	return name, t.Value
}

func (t CSRFToken) FormField() (string, string) {
	name := strings.TrimSpace(t.FormFieldName)
	if name == "" {
		name = DefaultCSRFFormFieldName
	}
	return name, t.Value
}

func (e ServerFormErrors) FormMessage() string {
	if message := strings.TrimSpace(e.Message); message != "" {
		return message
	}
	return strings.TrimSpace(e.Error)
}

func (r ServerActionResult) FormErrors() ServerFormErrors {
	return ServerFormErrors{
		Error:   strings.TrimSpace(r.Error),
		Message: strings.TrimSpace(r.Message),
		Fields:  cloneFieldErrors(r.Fields),
	}
}

func (r ServerActionResult) RedirectLocation() string {
	if r.Redirect == nil {
		return ""
	}
	return strings.TrimSpace(r.Redirect.Location)
}

func (r ServerActionResult) HasRedirect() bool {
	return r.RedirectLocation() != ""
}

func (r ServerActionResult) HasRefresh() bool {
	return r.Refresh != nil && (r.Refresh.Revalidate || len(r.Refresh.CacheKeys) > 0)
}

type formState[T any] struct {
	value        T
	initial      T
	submitIntent string
	touched      map[string]bool
	dirty        map[string]bool
	errors       FieldErrors
	formError    string
	validating   bool
	validated    bool
	validateSeq  int
	submitting   bool
	submitted    bool
	submitError  error
}

// Form exposes local form state, validation helpers, and submission lifecycle state.
type Form[T any] struct {
	state State[formState[T]]
}

// UseForm creates a typed form state container.
func UseForm[T any](initial T) Form[T] {
	return Form[T]{state: UseState(formState[T]{
		value:   initial,
		initial: initial,
		touched: map[string]bool{},
		dirty:   map[string]bool{},
		errors:  FieldErrors{},
	})}
}

// Get returns the current form value.
func (f Form[T]) Get() T {
	if f.state.get == nil {
		var zero T
		return zero
	}
	return f.state.Get().value
}

// Set replaces the current form value.
func (f Form[T]) Set(value T) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.value = value
		prev.dirty = computeDirtyFields(prev.initial, value)
		prev.formError = ""
		return prev
	})
}

// SetSubmitIntent records the current submit intent for intent-aware validation or submission flows.
func (f Form[T]) SetSubmitIntent(intent string) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.submitIntent = strings.TrimSpace(intent)
		return prev
	})
}

// SubmitIntent returns the most recently selected submit intent.
func (f Form[T]) SubmitIntent() string {
	if f.state.get == nil {
		return ""
	}
	return f.state.Get().submitIntent
}

// Update replaces the current form value using the previous value.
func (f Form[T]) Update(fn func(T) T) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.value = fn(prev.value)
		prev.dirty = computeDirtyFields(prev.initial, prev.value)
		prev.formError = ""
		return prev
	})
}

// SetField updates one named struct field and marks it touched.
func (f Form[T]) SetField(name string, value interface{}) bool {
	if f.state.get == nil {
		return false
	}

	updated := false
	f.state.Update(func(prev formState[T]) formState[T] {
		if prev.touched == nil {
			prev.touched = map[string]bool{}
		}
		if prev.dirty == nil {
			prev.dirty = map[string]bool{}
		}
		if prev.errors == nil {
			prev.errors = FieldErrors{}
		}

		nextValue, ok := assignNamedField(prev.value, name, value)
		if !ok {
			return prev
		}
		updated = true
		prev.value = nextValue
		prev.touched[name] = true
		if initialField, ok := readNamedField(prev.initial, name); ok {
			prev.dirty[name] = !reflect.DeepEqual(initialField, valueOfNamedField(prev.value, name))
		} else {
			prev.dirty[name] = true
		}
		delete(prev.errors, name)
		prev.formError = ""
		return prev
	})
	return updated
}

// Touch marks one field as touched.
func (f Form[T]) Touch(name string) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		if prev.touched == nil {
			prev.touched = map[string]bool{}
		}
		prev.touched[name] = true
		return prev
	})
}

// Touched reports whether a field has been touched.
func (f Form[T]) Touched(name string) bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().touched[name]
}

// Dirty reports whether a field differs from its initial value.
func (f Form[T]) Dirty(name string) bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().dirty[name]
}

// SetErrors replaces the current field error map.
func (f Form[T]) SetErrors(errors FieldErrors) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.errors = cloneFieldErrors(errors)
		prev.validated = true
		prev.validating = false
		return prev
	})
}

// SetFormError sets the form-level error message.
func (f Form[T]) SetFormError(message string) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.formError = message
		return prev
	})
}

// Errors returns a copy of the current field error map.
func (f Form[T]) Errors() FieldErrors {
	if f.state.get == nil {
		return FieldErrors{}
	}
	return cloneFieldErrors(f.state.Get().errors)
}

// Error returns the field error for name.
func (f Form[T]) Error(name string) string {
	if f.state.get == nil {
		return ""
	}
	return f.state.Get().errors[name]
}

// HasFieldError reports whether a field currently has an error message.
func (f Form[T]) HasFieldError(name string) bool {
	return f.FieldMessage(name) != ""
}

// FieldMessage returns the current message for one field.
func (f Form[T]) FieldMessage(name string) string {
	return f.Error(name)
}

// FieldStatus returns the current touched, dirty, pending, and error state for one field.
func (f Form[T]) FieldStatus(name string) FieldStatus {
	status := FieldStatus{Name: name}
	if f.state.get == nil {
		return status
	}
	state := f.state.Get()
	status.Touched = state.touched[name]
	status.Dirty = state.dirty[name]
	status.Pending = state.validating || state.submitting
	status.Error = state.errors[name]
	return status
}

// FormError returns the form-level error message.
func (f Form[T]) FormError() string {
	if f.state.get == nil {
		return ""
	}
	return f.state.Get().formError
}

// TouchedAny reports whether any field has been touched.
func (f Form[T]) TouchedAny() bool {
	if f.state.get == nil {
		return false
	}
	for _, touched := range f.state.Get().touched {
		if touched {
			return true
		}
	}
	return false
}

// DirtyAny reports whether any field differs from its initial value.
func (f Form[T]) DirtyAny() bool {
	if f.state.get == nil {
		return false
	}
	for _, dirty := range f.state.Get().dirty {
		if dirty {
			return true
		}
	}
	return false
}

// HasErrors reports whether the form currently has field or form-level errors.
func (f Form[T]) HasErrors() bool {
	if f.state.get == nil {
		return false
	}
	state := f.state.Get()
	return len(state.errors) > 0 || state.formError != ""
}

// ApplyServerErrors projects a structured server validation response onto the form state.
func (f Form[T]) ApplyServerErrors(response ServerFormErrors) bool {
	if f.state.get == nil {
		return false
	}
	f.SetErrors(response.Fields)
	f.SetFormError(response.FormMessage())
	return len(response.Fields) == 0 && response.FormMessage() == ""
}

// ApplyServerActionResult projects a typed server-action envelope onto the existing
// form error surface and returns whether the result is free of form-level errors.
func (f Form[T]) ApplyServerActionResult(result ServerActionResult) bool {
	return f.ApplyServerErrors(result.FormErrors())
}

// Validate runs synchronous validation and stores the resulting field errors.
func (f Form[T]) Validate(validate func(T) FieldErrors) bool {
	if f.state.get == nil {
		return true
	}
	if validate == nil {
		f.SetErrors(nil)
		return true
	}
	errors := validate(f.Get())
	f.SetFormError("")
	f.SetErrors(errors)
	return len(errors) == 0
}

// ValidateIntent runs validation against the current value plus an explicit submit intent.
func (f Form[T]) ValidateIntent(intent string, validate func(T, string) FieldErrors) bool {
	if f.state.get == nil {
		return true
	}
	trimmedIntent := strings.TrimSpace(intent)
	if validate == nil {
		f.SetSubmitIntent(trimmedIntent)
		f.SetFormError("")
		f.SetErrors(nil)
		return true
	}
	f.SetSubmitIntent(trimmedIntent)
	errors := validate(f.Get(), trimmedIntent)
	f.SetFormError("")
	f.SetErrors(errors)
	return len(errors) == 0
}

// ValidateAsync runs asynchronous validation and updates form state when it completes.
func (f Form[T]) ValidateAsync(validate func(T) (FieldErrors, string), onComplete func(bool)) {
	if f.state.get == nil {
		if onComplete != nil {
			onComplete(true)
		}
		return
	}
	if validate == nil {
		f.SetFormError("")
		f.SetErrors(nil)
		if onComplete != nil {
			onComplete(true)
		}
		return
	}

	snapshot := f.Get()
	stateSnapshot := f.state.Get()
	sequence := stateSnapshot.validateSeq + 1
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.validating = true
		prev.validated = false
		prev.formError = ""
		prev.validateSeq = sequence
		return prev
	})

	go func(value T, expectedSeq int) {
		errors, formError := validate(value)
		valid := len(errors) == 0 && formError == ""
		f.state.Update(func(prev formState[T]) formState[T] {
			if prev.validateSeq != expectedSeq {
				return prev
			}
			prev.errors = cloneFieldErrors(errors)
			prev.formError = formError
			prev.validating = false
			prev.validated = true
			return prev
		})
		if onComplete != nil {
			onComplete(valid)
		}
	}(snapshot, sequence)
}

// Submit runs the submit function in a goroutine and updates submission lifecycle state.
func (f Form[T]) Submit(run func(T) error) {
	if f.state.get == nil || run == nil {
		return
	}
	snapshot := f.Get()
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.submitting = true
		prev.submitted = false
		prev.submitError = nil
		prev.formError = ""
		return prev
	})
	go func(value T) {
		err := run(value)
		f.state.Update(func(prev formState[T]) formState[T] {
			prev.submitting = false
			prev.submitError = err
			prev.submitted = err == nil
			if err != nil {
				prev.formError = err.Error()
			} else {
				prev.formError = ""
			}
			return prev
		})
	}(snapshot)
}

// SubmitWithIntent runs the submit function with an explicit intent and tracks that intent while submission is pending.
func (f Form[T]) SubmitWithIntent(intent string, run func(T, string) error) {
	if f.state.get == nil || run == nil {
		return
	}
	trimmedIntent := strings.TrimSpace(intent)
	snapshot := f.Get()
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.submitIntent = trimmedIntent
		prev.submitting = true
		prev.submitted = false
		prev.submitError = nil
		prev.formError = ""
		return prev
	})
	go func(value T, activeIntent string) {
		err := run(value, activeIntent)
		f.state.Update(func(prev formState[T]) formState[T] {
			prev.submitting = false
			prev.submitError = err
			prev.submitted = err == nil
			prev.submitIntent = activeIntent
			if err != nil {
				prev.formError = err.Error()
			} else {
				prev.formError = ""
			}
			return prev
		})
	}(snapshot, trimmedIntent)
}

// Submitting reports whether a submission is in flight.
func (f Form[T]) Submitting() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().submitting
}

// Validating reports whether async validation is in flight.
func (f Form[T]) Validating() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().validating
}

// Validated reports whether validation has completed at least once.
func (f Form[T]) Validated() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().validated
}

// Submitted reports whether the last submission completed successfully.
func (f Form[T]) Submitted() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().submitted
}

// IntentPending reports whether the given intent is the currently pending submit action.
func (f Form[T]) IntentPending(intent string) bool {
	if f.state.get == nil {
		return false
	}
	state := f.state.Get()
	return state.submitting && state.submitIntent == strings.TrimSpace(intent)
}

// SubmitError returns the last submission error.
func (f Form[T]) SubmitError() error {
	if f.state.get == nil {
		return nil
	}
	return f.state.Get().submitError
}

// Reset restores the form to its initial value or the provided next value.
func (f Form[T]) Reset(next ...T) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		if len(next) > 0 {
			prev.initial = next[0]
			prev.value = next[0]
		} else {
			prev.value = prev.initial
		}
		prev.submitIntent = ""
		prev.touched = map[string]bool{}
		prev.dirty = map[string]bool{}
		prev.errors = FieldErrors{}
		prev.formError = ""
		prev.validating = false
		prev.validated = false
		prev.validateSeq = 0
		prev.submitting = false
		prev.submitted = false
		prev.submitError = nil
		return prev
	})
}

func cloneFieldErrors(errors FieldErrors) FieldErrors {
	if len(errors) == 0 {
		return FieldErrors{}
	}
	clone := make(FieldErrors, len(errors))
	for key, value := range errors {
		clone[key] = value
	}
	return clone
}

func computeDirtyFields[T any](initial T, current T) map[string]bool {
	dirty := map[string]bool{}
	initialValue := reflect.ValueOf(initial)
	currentValue := reflect.ValueOf(current)
	if initialValue.Kind() != reflect.Struct || currentValue.Kind() != reflect.Struct {
		return dirty
	}
	initialType := initialValue.Type()
	for index := 0; index < initialValue.NumField(); index++ {
		field := initialType.Field(index)
		if field.PkgPath != "" {
			continue
		}
		if !reflect.DeepEqual(initialValue.Field(index).Interface(), currentValue.Field(index).Interface()) {
			dirty[field.Name] = true
		}
	}
	return dirty
}

func assignNamedField[T any](target T, name string, value interface{}) (T, bool) {
	ptr := reflect.New(reflect.TypeOf(target))
	ptr.Elem().Set(reflect.ValueOf(target))
	field := ptr.Elem().FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		return target, false
	}

	provided := reflect.ValueOf(value)
	if !provided.IsValid() {
		return target, false
	}
	switch {
	case provided.Type() == field.Type():
		field.Set(provided)
	case provided.Type().AssignableTo(field.Type()):
		field.Set(provided)
	case provided.Type().ConvertibleTo(field.Type()):
		field.Set(provided.Convert(field.Type()))
	default:
		return target, false
	}
	return ptr.Elem().Interface().(T), true
}

func readNamedField[T any](value T, name string) (interface{}, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	field := rv.FieldByName(name)
	if !field.IsValid() {
		return nil, false
	}
	return field.Interface(), true
}

func valueOfNamedField[T any](value T, name string) interface{} {
	field, ok := readNamedField(value, name)
	if !ok {
		return nil
	}
	return field
}
