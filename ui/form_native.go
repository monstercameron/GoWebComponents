//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"reflect"
	"sync"
)

type formState[T any] struct {
	value       T
	initial     T
	touched     map[string]bool
	dirty       map[string]bool
	errors      FieldErrors
	formError   string
	validating  bool
	validated   bool
	validateSeq int
	submitting  bool
	submitted   bool
	submitError error
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
