//go:build js && wasm
// +build js,wasm

package ui

import "reflect"

type FieldErrors map[string]string

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

type Form[T any] struct {
	state State[formState[T]]
}

func UseForm[T any](initial T) Form[T] {
	return Form[T]{state: UseState(formState[T]{
		value:   initial,
		initial: initial,
		touched: map[string]bool{},
		dirty:   map[string]bool{},
		errors:  FieldErrors{},
	})}
}

func (f Form[T]) Get() T {
	if f.state.get == nil {
		var zero T
		return zero
	}
	return f.state.Get().value
}

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

func (f Form[T]) Touched(name string) bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().touched[name]
}

func (f Form[T]) Dirty(name string) bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().dirty[name]
}

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

func (f Form[T]) SetFormError(message string) {
	if f.state.get == nil {
		return
	}
	f.state.Update(func(prev formState[T]) formState[T] {
		prev.formError = message
		return prev
	})
}

func (f Form[T]) Errors() FieldErrors {
	if f.state.get == nil {
		return FieldErrors{}
	}
	return cloneFieldErrors(f.state.Get().errors)
}

func (f Form[T]) Error(name string) string {
	if f.state.get == nil {
		return ""
	}
	return f.state.Get().errors[name]
}

func (f Form[T]) FormError() string {
	if f.state.get == nil {
		return ""
	}
	return f.state.Get().formError
}

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

func (f Form[T]) HasErrors() bool {
	if f.state.get == nil {
		return false
	}
	state := f.state.Get()
	return len(state.errors) > 0 || state.formError != ""
}

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

func (f Form[T]) Submitting() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().submitting
}

func (f Form[T]) Validating() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().validating
}

func (f Form[T]) Validated() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().validated
}

func (f Form[T]) Submitted() bool {
	if f.state.get == nil {
		return false
	}
	return f.state.Get().submitted
}

func (f Form[T]) SubmitError() error {
	if f.state.get == nil {
		return nil
	}
	return f.state.Get().submitError
}

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
