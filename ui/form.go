//go:build js && wasm

package ui

import (
	"reflect"
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
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

// NewCSRFToken is a core package helper.
func NewCSRFToken(parseValue string) CSRFToken {
	return CSRFToken{
		Value:         parseValue,
		HeaderName:    DefaultCSRFHeaderName,
		FormFieldName: DefaultCSRFFormFieldName,
	}
}

// Header is a core package helper.
func (parseT CSRFToken) Header() (string, string) {
	parseName := strings.TrimSpace(parseT.HeaderName)
	if parseName == "" {
		parseName = DefaultCSRFHeaderName
	}
	return parseName, parseT.Value
}

// FormField is a core package helper.
func (parseT CSRFToken) FormField() (string, string) {
	parseName := strings.TrimSpace(parseT.FormFieldName)
	if parseName == "" {
		parseName = DefaultCSRFFormFieldName
	}
	return parseName, parseT.Value
}

// FormMessage is a core package helper.
func (parseE ServerFormErrors) FormMessage() string {
	if parseMessage := strings.TrimSpace(parseE.Message); parseMessage != "" {
		return parseMessage
	}
	return strings.TrimSpace(parseE.Error)
}

// FormErrors is a core package helper.
func (parseR ServerActionResult) FormErrors() ServerFormErrors {
	return ServerFormErrors{
		Error:   strings.TrimSpace(parseR.Error),
		Message: strings.TrimSpace(parseR.Message),
		Fields:  cloneFieldErrors(parseR.Fields),
	}
}

// RedirectLocation is a core package helper.
func (parseR ServerActionResult) RedirectLocation() string {
	if parseR.Redirect == nil {
		return ""
	}
	return strings.TrimSpace(parseR.Redirect.Location)
}

// HasRedirect is a core package helper.
func (parseR ServerActionResult) HasRedirect() bool {
	return parseR.RedirectLocation() != ""
}

// HasRefresh is a core package helper.
func (parseR ServerActionResult) HasRefresh() bool {
	return parseR.Refresh != nil && (parseR.Refresh.Revalidate || len(parseR.Refresh.CacheKeys) > 0)
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
func UseForm[T any](parseInitial T) Form[T] {
	return Form[T]{state: UseState(formState[T]{
		value:   parseInitial,
		initial: parseInitial,
		touched: map[string]bool{},
		dirty:   map[string]bool{},
		errors:  FieldErrors{},
	})}
}

// Get returns the current form value.
func (parseF Form[T]) Get() T {
	if parseF.state.get == nil {
		var parseZero T
		return parseZero
	}
	return parseF.state.Get().value
}

// Set replaces the current form value.
func (parseF Form[T]) Set(parseValue T) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.value = parseValue
		parsePrev.dirty = computeDirtyFields(parsePrev.initial, parseValue)
		parsePrev.formError = ""
		return parsePrev
	})
}

// SetSubmitIntent records the current submit intent for intent-aware validation or submission flows.
func (parseF Form[T]) SetSubmitIntent(parseIntent string) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.submitIntent = strings.TrimSpace(parseIntent)
		return parsePrev
	})
}

// SubmitIntent returns the most recently selected submit intent.
func (parseF Form[T]) SubmitIntent() string {
	if parseF.state.get == nil {
		return ""
	}
	return parseF.state.Get().submitIntent
}

// Update replaces the current form value using the previous value.
func (parseF Form[T]) Update(parseFn func(T) T) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.value = parseFn(parsePrev.value)
		parsePrev.dirty = computeDirtyFields(parsePrev.initial, parsePrev.value)
		parsePrev.formError = ""
		return parsePrev
	})
}

// SetField updates one named struct field and marks it touched.
func (parseF Form[T]) SetField(parseName string, parseValue interface{}) bool {
	if parseF.state.get == nil {
		return false
	}

	isParseUpdated := false
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		if parsePrev.touched == nil {
			parsePrev.touched = map[string]bool{}
		}
		if parsePrev.dirty == nil {
			parsePrev.dirty = map[string]bool{}
		}
		if parsePrev.errors == nil {
			parsePrev.errors = FieldErrors{}
		}

		parseNextValue, parseOk := assignNamedField(parsePrev.value, parseName, parseValue)
		if !parseOk {
			return parsePrev
		}
		isParseUpdated = true
		parsePrev.value = parseNextValue
		parsePrev.touched[parseName] = true
		if parseInitialField, parseOk2 := readNamedField(parsePrev.initial, parseName); parseOk2 {
			parsePrev.dirty[parseName] = !reflect.DeepEqual(parseInitialField, valueOfNamedField(parsePrev.value, parseName))
		} else {
			parsePrev.dirty[parseName] = true
		}
		delete(parsePrev.errors, parseName)
		parsePrev.formError = ""
		return parsePrev
	})
	return isParseUpdated
}

// HasField reports whether T has a field addressable by name (the same names SetField/Touch use).
// SetField returns false for both an unknown field AND a type mismatch; HasField lets a test or a
// dev-time guard catch a misspelled field name specifically.
func (parseF Form[T]) HasField(parseName string) bool {
	var parseValue T
	if parseF.state.get != nil {
		parseValue = parseF.state.get().value
	}
	_, parseOk := readNamedField(parseValue, parseName)
	return parseOk
}

// MustSetField is SetField that panics when the field does not exist or the value's type does not
// match — turning a silent no-op into a loud failure that surfaces a typo'd field name during
// development. Prefer it where the field name is a hardcoded literal.
func (parseF Form[T]) MustSetField(parseName string, parseValue interface{}) {
	if !parseF.SetField(parseName, parseValue) {
		panic("ui: MustSetField: unknown field or mismatched value type: " + parseName)
	}
}

// Touch marks one field as touched.
func (parseF Form[T]) Touch(parseName string) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		if parsePrev.touched == nil {
			parsePrev.touched = map[string]bool{}
		}
		parsePrev.touched[parseName] = true
		return parsePrev
	})
}

// Touched reports whether a field has been touched.
func (parseF Form[T]) Touched(parseName string) bool {
	if parseF.state.get == nil {
		return false
	}
	return parseF.state.Get().touched[parseName]
}

// Dirty reports whether a field differs from its initial value.
func (parseF Form[T]) Dirty(parseName string) bool {
	if parseF.state.get == nil {
		return false
	}
	return parseF.state.Get().dirty[parseName]
}

// SetErrors replaces the current field error map.
func (parseF Form[T]) SetErrors(parseErrors FieldErrors) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.errors = cloneFieldErrors(parseErrors)
		parsePrev.validated = true
		parsePrev.validating = false
		return parsePrev
	})
}

// SetFormError sets the form-level error message.
func (parseF Form[T]) SetFormError(parseMessage string) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.formError = parseMessage
		return parsePrev
	})
}

// Errors returns a copy of the current field error map.
func (parseF Form[T]) Errors() FieldErrors {
	if parseF.state.get == nil {
		return FieldErrors{}
	}
	return cloneFieldErrors(parseF.state.Get().errors)
}

// Error returns the field error for name.
func (parseF Form[T]) Error(parseName string) string {
	if parseF.state.get == nil {
		return ""
	}
	return parseF.state.Get().errors[parseName]
}

// HasFieldError reports whether a field currently has an error message.
func (parseF Form[T]) HasFieldError(parseName string) bool {
	return parseF.FieldMessage(parseName) != ""
}

// FieldMessage returns the current message for one field.
func (parseF Form[T]) FieldMessage(parseName string) string {
	return parseF.Error(parseName)
}

// FieldStatus returns the current touched, dirty, pending, and error state for one field.
func (parseF Form[T]) FieldStatus(parseName string) FieldStatus {
	parseStatus := FieldStatus{Name: parseName}
	if parseF.state.get == nil {
		return parseStatus
	}
	parseState := parseF.state.Get()
	parseStatus.Touched = parseState.touched[parseName]
	parseStatus.Dirty = parseState.dirty[parseName]
	parseStatus.Pending = parseState.validating || parseState.submitting
	parseStatus.Error = parseState.errors[parseName]
	return parseStatus
}

// FormError returns the form-level error message.
func (parseF Form[T]) FormError() string {
	if parseF.state.get == nil {
		return ""
	}
	return parseF.state.Get().formError
}

// TouchedAny reports whether any field has been touched.
func (parseF Form[T]) TouchedAny() bool {
	if parseF.state.get == nil {
		return false
	}
	for _, parseTouched := range parseF.state.Get().touched {
		if parseTouched {
			return true
		}
	}
	return false
}

// DirtyAny reports whether any field differs from its initial value.
func (parseF Form[T]) DirtyAny() bool {
	if parseF.state.get == nil {
		return false
	}
	for _, parseDirty := range parseF.state.Get().dirty {
		if parseDirty {
			return true
		}
	}
	return false
}

// HasErrors reports whether the form currently has field or form-level errors.
func (parseF Form[T]) HasErrors() bool {
	if parseF.state.get == nil {
		return false
	}
	parseState := parseF.state.Get()
	return len(parseState.errors) > 0 || parseState.formError != ""
}

// ApplyServerErrors projects a structured server validation response onto the form state.
func (parseF Form[T]) ApplyServerErrors(parseResponse ServerFormErrors) bool {
	if parseF.state.get == nil {
		return false
	}
	parseF.SetErrors(parseResponse.Fields)
	parseF.SetFormError(parseResponse.FormMessage())
	return len(parseResponse.Fields) == 0 && parseResponse.FormMessage() == ""
}

// ApplyServerActionResult projects a typed server-action envelope onto the existing
// form error surface and returns whether the result is free of form-level errors.
func (parseF Form[T]) ApplyServerActionResult(parseResult ServerActionResult) bool {
	return parseF.ApplyServerErrors(parseResult.FormErrors())
}

// Validate runs synchronous validation and stores the resulting field errors.
func (parseF Form[T]) Validate(parseValidate func(T) FieldErrors) bool {
	if parseF.state.get == nil {
		return true
	}
	if parseValidate == nil {
		parseF.SetErrors(nil)
		return true
	}
	parseErrors := parseValidate(parseF.Get())
	parseF.SetFormError("")
	parseF.SetErrors(parseErrors)
	return len(parseErrors) == 0
}

// ValidateIntent runs validation against the current value plus an explicit submit intent.
func (parseF Form[T]) ValidateIntent(parseIntent string, parseValidate func(T, string) FieldErrors) bool {
	if parseF.state.get == nil {
		return true
	}
	parseTrimmedIntent := strings.TrimSpace(parseIntent)
	if parseValidate == nil {
		parseF.SetSubmitIntent(parseTrimmedIntent)
		parseF.SetFormError("")
		parseF.SetErrors(nil)
		return true
	}
	parseF.SetSubmitIntent(parseTrimmedIntent)
	parseErrors := parseValidate(parseF.Get(), parseTrimmedIntent)
	parseF.SetFormError("")
	parseF.SetErrors(parseErrors)
	return len(parseErrors) == 0
}

// ValidateAsync runs asynchronous validation and updates form state when it completes.
func (parseF Form[T]) ValidateAsync(parseValidate func(T) (FieldErrors, string), parseOnComplete func(bool)) {
	if parseF.state.get == nil {
		if parseOnComplete != nil {
			parseOnComplete(true)
		}
		return
	}
	if parseValidate == nil {
		parseF.SetFormError("")
		parseF.SetErrors(nil)
		if parseOnComplete != nil {
			parseOnComplete(true)
		}
		return
	}

	parseSnapshot := parseF.Get()
	parseStateSnapshot := parseF.state.Get()
	parseSequence := parseStateSnapshot.validateSeq + 1
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.validating = true
		parsePrev.validated = false
		parsePrev.formError = ""
		parsePrev.validateSeq = parseSequence
		return parsePrev
	})

	go func(parseValue T, parseExpectedSeq int) {
		defer runtime.RecoverContainedPanic("ui", "UseForm validator")
		parseErrors, parseFormError := parseValidate(parseValue)
		isParseValid := len(parseErrors) == 0 && parseFormError == ""
		parseF.state.Update(func(parsePrev2 formState[T]) formState[T] {
			if parsePrev2.validateSeq != parseExpectedSeq {
				return parsePrev2
			}
			parsePrev2.errors = cloneFieldErrors(parseErrors)
			parsePrev2.formError = parseFormError
			parsePrev2.validating = false
			parsePrev2.validated = true
			return parsePrev2
		})
		if parseOnComplete != nil {
			parseOnComplete(isParseValid)
		}
	}(parseSnapshot, parseSequence)
}

// Submit runs the submit function in a goroutine and updates submission lifecycle state.
func (parseF Form[T]) Submit(parseRun func(T) error) {
	if parseF.state.get == nil || parseRun == nil {
		return
	}
	parseSnapshot := parseF.Get()
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.submitIntent = ""
		parsePrev.submitting = true
		parsePrev.submitted = false
		parsePrev.submitError = nil
		parsePrev.formError = ""
		return parsePrev
	})
	go func(parseValue T) {
		defer runtime.RecoverContainedPanic("ui", "UseForm submit runner")
		parseErr := parseRun(parseValue)
		parseF.state.Update(func(parsePrev2 formState[T]) formState[T] {
			parsePrev2.submitting = false
			parsePrev2.submitError = parseErr
			parsePrev2.submitted = parseErr == nil
			parsePrev2.submitIntent = ""
			if parseErr != nil {
				parsePrev2.formError = parseErr.Error()
			} else {
				parsePrev2.formError = ""
			}
			return parsePrev2
		})
	}(parseSnapshot)
}

// SubmitWithIntent runs the submit function with an explicit intent and tracks that intent while submission is pending.
func (parseF Form[T]) SubmitWithIntent(parseIntent string, parseRun func(T, string) error) {
	if parseF.state.get == nil || parseRun == nil {
		return
	}
	parseTrimmedIntent := strings.TrimSpace(parseIntent)
	parseSnapshot := parseF.Get()
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		parsePrev.submitIntent = parseTrimmedIntent
		parsePrev.submitting = true
		parsePrev.submitted = false
		parsePrev.submitError = nil
		parsePrev.formError = ""
		return parsePrev
	})
	go func(parseValue T, parseActiveIntent string) {
		defer runtime.RecoverContainedPanic("ui", "UseForm intent runner")
		parseErr := parseRun(parseValue, parseActiveIntent)
		parseF.state.Update(func(parsePrev2 formState[T]) formState[T] {
			parsePrev2.submitting = false
			parsePrev2.submitError = parseErr
			parsePrev2.submitted = parseErr == nil
			parsePrev2.submitIntent = parseActiveIntent
			if parseErr != nil {
				parsePrev2.formError = parseErr.Error()
			} else {
				parsePrev2.formError = ""
			}
			return parsePrev2
		})
	}(parseSnapshot, parseTrimmedIntent)
}

// Submitting reports whether a submission is in flight.
func (parseF Form[T]) Submitting() bool {
	if parseF.state.get == nil {
		return false
	}
	return parseF.state.Get().submitting
}

// Validating reports whether async validation is in flight.
func (parseF Form[T]) Validating() bool {
	if parseF.state.get == nil {
		return false
	}
	return parseF.state.Get().validating
}

// Validated reports whether validation has completed at least once.
func (parseF Form[T]) Validated() bool {
	if parseF.state.get == nil {
		return false
	}
	return parseF.state.Get().validated
}

// Submitted reports whether the last submission completed successfully.
func (parseF Form[T]) Submitted() bool {
	if parseF.state.get == nil {
		return false
	}
	return parseF.state.Get().submitted
}

// IntentPending reports whether the given intent is the currently pending submit action.
func (parseF Form[T]) IntentPending(parseIntent string) bool {
	if parseF.state.get == nil {
		return false
	}
	parseState := parseF.state.Get()
	return parseState.submitting && parseState.submitIntent == strings.TrimSpace(parseIntent)
}

// SubmitError returns the last submission error.
func (parseF Form[T]) SubmitError() error {
	if parseF.state.get == nil {
		return nil
	}
	return parseF.state.Get().submitError
}

// Reset restores the form to its initial value or the provided next value.
func (parseF Form[T]) Reset(parseNext ...T) {
	if parseF.state.get == nil {
		return
	}
	parseF.state.Update(func(parsePrev formState[T]) formState[T] {
		if len(parseNext) > 0 {
			parsePrev.initial = parseNext[0]
			parsePrev.value = parseNext[0]
		} else {
			parsePrev.value = parsePrev.initial
		}
		parsePrev.submitIntent = ""
		parsePrev.touched = map[string]bool{}
		parsePrev.dirty = map[string]bool{}
		parsePrev.errors = FieldErrors{}
		parsePrev.formError = ""
		parsePrev.validating = false
		parsePrev.validated = false
		parsePrev.validateSeq = 0
		parsePrev.submitting = false
		parsePrev.submitted = false
		parsePrev.submitError = nil
		return parsePrev
	})
}

// cloneFieldErrors is a core package helper.
func cloneFieldErrors(parseErrors FieldErrors) FieldErrors {
	if len(parseErrors) == 0 {
		return FieldErrors{}
	}
	parseClone := make(FieldErrors, len(parseErrors))
	for parseKey, parseValue := range parseErrors {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

// computeDirtyFields is a core package helper.
func computeDirtyFields[T any](parseInitial T, parseCurrent T) map[string]bool {
	parseDirty := map[string]bool{}
	parseInitialValue := reflect.ValueOf(parseInitial)
	parseCurrentValue := reflect.ValueOf(parseCurrent)
	if parseInitialValue.Kind() != reflect.Struct || parseCurrentValue.Kind() != reflect.Struct {
		return parseDirty
	}
	parseInitialType := parseInitialValue.Type()
	for parseIndex := 0; parseIndex < parseInitialValue.NumField(); parseIndex++ {
		parseField := parseInitialType.Field(parseIndex)
		if parseField.PkgPath != "" {
			continue
		}
		if !reflect.DeepEqual(parseInitialValue.Field(parseIndex).Interface(), parseCurrentValue.Field(parseIndex).Interface()) {
			parseDirty[parseField.Name] = true
		}
	}
	return parseDirty
}

// assignNamedField is a core package helper.
func assignNamedField[T any](parseTarget T, parseName string, parseValue interface{}) (T, bool) {
	parsePtr := reflect.New(reflect.TypeOf(parseTarget))
	parsePtr.Elem().Set(reflect.ValueOf(parseTarget))
	parseField := parsePtr.Elem().FieldByName(parseName)
	if !parseField.IsValid() || !parseField.CanSet() {
		return parseTarget, false
	}

	parseProvided := reflect.ValueOf(parseValue)
	if !parseProvided.IsValid() {
		return parseTarget, false
	}
	switch {
	case parseProvided.Type() == parseField.Type():
		parseField.Set(parseProvided)
	case parseProvided.Type().AssignableTo(parseField.Type()):
		parseField.Set(parseProvided)
	case parseProvided.Type().ConvertibleTo(parseField.Type()):
		parseField.Set(parseProvided.Convert(parseField.Type()))
	default:
		return parseTarget, false
	}
	return parsePtr.Elem().Interface().(T), true
}

// readNamedField is a core package helper.
func readNamedField[T any](parseValue T, parseName string) (interface{}, bool) {
	parseRv := reflect.ValueOf(parseValue)
	if parseRv.Kind() != reflect.Struct {
		return nil, false
	}
	parseField := parseRv.FieldByName(parseName)
	// CanInterface guards an unexported-but-existing field: FieldByName finds it, but .Interface()
	// would panic. Treat it as absent — consistent with SetField (which rejects unsettable fields).
	if !parseField.IsValid() || !parseField.CanInterface() {
		return nil, false
	}
	return parseField.Interface(), true
}

// valueOfNamedField is a core package helper.
func valueOfNamedField[T any](parseValue T, parseName string) interface{} {
	parseField, parseOk := readNamedField(parseValue, parseName)
	if !parseOk {
		return nil
	}
	return parseField
}
