//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"reflect"
	"strings"
	"sync"
)

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

// NewCSRFToken creates a CSRFToken with the given value and default header and form field names.
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
	mu    *sync.RWMutex
	state *formState[T]
}

// UseForm creates a typed form state container.
func UseForm[T any](initial T) Form[T] {
	return Form[T]{
		mu: &sync.RWMutex{},
		state: &formState[T]{
			value:   initial,
			initial: initial,
			touched: map[string]bool{},
			dirty:   map[string]bool{},
			errors:  FieldErrors{},
		},
	}
}

// Get returns the current form value.
func (f Form[T]) Get() T {
	if f.state == nil || f.mu == nil {
		var zero T
		return zero
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.value
}

// Set replaces the current form value.
func (f Form[T]) Set(value T) {
	if f.state == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.value = value
	f.state.dirty = computeDirtyFields(f.state.initial, value)
	f.state.formError = ""
}

// SetSubmitIntent records the current submit intent for intent-aware validation or submission flows.
func (f Form[T]) SetSubmitIntent(intent string) {
	if f.state == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.submitIntent = strings.TrimSpace(intent)
}

// SubmitIntent returns the most recently selected submit intent.
func (f Form[T]) SubmitIntent() string {
	if f.state == nil || f.mu == nil {
		return ""
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.submitIntent
}

// Update replaces the current form value using the previous value.
func (f Form[T]) Update(fn func(T) T) {
	if f.state == nil || f.mu == nil || fn == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.value = fn(f.state.value)
	f.state.dirty = computeDirtyFields(f.state.initial, f.state.value)
	f.state.formError = ""
}

// SetField updates one named struct field and marks it touched.
func (f Form[T]) SetField(name string, value interface{}) bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	nextValue, ok := assignNamedField(f.state.value, name, value)
	if !ok {
		return false
	}
	f.state.value = nextValue
	if f.state.touched == nil {
		f.state.touched = map[string]bool{}
	}
	if f.state.dirty == nil {
		f.state.dirty = map[string]bool{}
	}
	if f.state.errors == nil {
		f.state.errors = FieldErrors{}
	}
	f.state.touched[name] = true
	if initialField, ok := readNamedField(f.state.initial, name); ok {
		f.state.dirty[name] = !reflect.DeepEqual(initialField, valueOfNamedField(f.state.value, name))
	} else {
		f.state.dirty[name] = true
	}
	delete(f.state.errors, name)
	f.state.formError = ""
	return true
}

// Touch marks one field as touched.
func (f Form[T]) Touch(name string) {
	if f.state == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state.touched == nil {
		f.state.touched = map[string]bool{}
	}
	f.state.touched[name] = true
}

// Touched reports whether a field has been touched.
func (f Form[T]) Touched(name string) bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.touched[name]
}

// Dirty reports whether a field differs from its initial value.
func (f Form[T]) Dirty(name string) bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.dirty[name]
}

// SetErrors replaces the current field error map.
func (f Form[T]) SetErrors(errors FieldErrors) {
	if f.state == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.errors = cloneFieldErrors(errors)
	f.state.validated = true
	f.state.validating = false
}

// SetFormError sets the form-level error message.
func (f Form[T]) SetFormError(message string) {
	if f.state == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.formError = message
}

// Errors returns a copy of the current field error map.
func (f Form[T]) Errors() FieldErrors {
	if f.state == nil || f.mu == nil {
		return FieldErrors{}
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return cloneFieldErrors(f.state.errors)
}

// Error returns the field error for name.
func (f Form[T]) Error(name string) string {
	if f.state == nil || f.mu == nil {
		return ""
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.errors[name]
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
	if f.state == nil || f.mu == nil {
		return status
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	status.Touched = f.state.touched[name]
	status.Dirty = f.state.dirty[name]
	status.Pending = f.state.validating || f.state.submitting
	status.Error = f.state.errors[name]
	return status
}

// FormError returns the form-level error message.
func (f Form[T]) FormError() string {
	if f.state == nil || f.mu == nil {
		return ""
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.formError
}

// TouchedAny reports whether any field has been touched.
func (f Form[T]) TouchedAny() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, touched := range f.state.touched {
		if touched {
			return true
		}
	}
	return false
}

// DirtyAny reports whether any field differs from its initial value.
func (f Form[T]) DirtyAny() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, dirty := range f.state.dirty {
		if dirty {
			return true
		}
	}
	return false
}

// HasErrors reports whether the form currently has field or form-level errors.
func (f Form[T]) HasErrors() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.state.errors) > 0 || f.state.formError != ""
}

// ApplyServerErrors projects a structured server validation response onto the form state.
func (f Form[T]) ApplyServerErrors(response ServerFormErrors) bool {
	if f.state == nil || f.mu == nil {
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
	if f.state == nil || f.mu == nil {
		return true
	}
	if validate == nil {
		f.SetFormError("")
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
	if f.state == nil || f.mu == nil {
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
	if f.state == nil || f.mu == nil {
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
	f.mu.Lock()
	sequence := f.state.validateSeq + 1
	f.state.validating = true
	f.state.validated = false
	f.state.formError = ""
	f.state.validateSeq = sequence
	f.mu.Unlock()

	go func(value T, expectedSeq int) {
		errors, formError := validate(value)
		valid := len(errors) == 0 && formError == ""
		f.mu.Lock()
		if f.state != nil && f.state.validateSeq == expectedSeq {
			f.state.errors = cloneFieldErrors(errors)
			f.state.formError = formError
			f.state.validating = false
			f.state.validated = true
		}
		f.mu.Unlock()
		if onComplete != nil {
			onComplete(valid)
		}
	}(snapshot, sequence)
}

// Submit runs the submit function in a goroutine and updates submission lifecycle state.
func (f Form[T]) Submit(run func(T) error) {
	if f.state == nil || f.mu == nil || run == nil {
		return
	}
	snapshot := f.Get()
	f.mu.Lock()
	f.state.submitting = true
	f.state.submitted = false
	f.state.submitError = nil
	f.state.formError = ""
	f.mu.Unlock()

	go func(value T) {
		err := run(value)
		f.mu.Lock()
		if f.state != nil {
			f.state.submitting = false
			f.state.submitError = err
			f.state.submitted = err == nil
			if err != nil {
				f.state.formError = err.Error()
			} else {
				f.state.formError = ""
			}
		}
		f.mu.Unlock()
	}(snapshot)
}

// SubmitWithIntent runs the submit function with an explicit intent and tracks that intent while submission is pending.
func (f Form[T]) SubmitWithIntent(intent string, run func(T, string) error) {
	if f.state == nil || f.mu == nil || run == nil {
		return
	}
	trimmedIntent := strings.TrimSpace(intent)
	snapshot := f.Get()
	f.mu.Lock()
	f.state.submitIntent = trimmedIntent
	f.state.submitting = true
	f.state.submitted = false
	f.state.submitError = nil
	f.state.formError = ""
	f.mu.Unlock()

	go func(value T, activeIntent string) {
		err := run(value, activeIntent)
		f.mu.Lock()
		if f.state != nil {
			f.state.submitting = false
			f.state.submitError = err
			f.state.submitted = err == nil
			f.state.submitIntent = activeIntent
			if err != nil {
				f.state.formError = err.Error()
			} else {
				f.state.formError = ""
			}
		}
		f.mu.Unlock()
	}(snapshot, trimmedIntent)
}

// Submitting reports whether a submission is in flight.
func (f Form[T]) Submitting() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.submitting
}

// Validating reports whether async validation is in flight.
func (f Form[T]) Validating() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.validating
}

// Validated reports whether validation has completed at least once.
func (f Form[T]) Validated() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.validated
}

// Submitted reports whether the last submission completed successfully.
func (f Form[T]) Submitted() bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.submitted
}

// IntentPending reports whether the given intent is the currently pending submit action.
func (f Form[T]) IntentPending(intent string) bool {
	if f.state == nil || f.mu == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.submitting && f.state.submitIntent == strings.TrimSpace(intent)
}

// SubmitError returns the last submission error.
func (f Form[T]) SubmitError() error {
	if f.state == nil || f.mu == nil {
		return nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.submitError
}

// Reset restores the form to its initial value or the provided next value.
func (f Form[T]) Reset(next ...T) {
	if f.state == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(next) > 0 {
		f.state.initial = next[0]
		f.state.value = next[0]
	} else {
		f.state.value = f.state.initial
	}
	f.state.submitIntent = ""
	f.state.touched = map[string]bool{}
	f.state.dirty = map[string]bool{}
	f.state.errors = FieldErrors{}
	f.state.formError = ""
	f.state.validating = false
	f.state.validated = false
	f.state.validateSeq = 0
	f.state.submitting = false
	f.state.submitted = false
	f.state.submitError = nil
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
