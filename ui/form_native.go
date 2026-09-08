//go:build !js || !wasm

package ui

import (
	"maps"
	"reflect"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
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
	mu    *sync.RWMutex
	state *formState[T]
}

// UseForm creates a typed form state container.
func UseForm[T any](parseInitial T) Form[T] {
	return Form[T]{
		mu: &sync.RWMutex{},
		state: &formState[T]{
			value:   parseInitial,
			initial: parseInitial,
			touched: map[string]bool{},
			dirty:   map[string]bool{},
			errors:  FieldErrors{},
		},
	}
}

// Get returns the current form value.
func (parseF Form[T]) Get() T {
	if parseF.state == nil || parseF.mu == nil {
		var parseZero T
		return parseZero
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.value
}

// Set replaces the current form value.
func (parseF Form[T]) Set(parseValue T) {
	if parseF.state == nil || parseF.mu == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	parseF.state.value = parseValue
	parseF.state.dirty = computeDirtyFields(parseF.state.initial, parseValue)
	parseF.state.formError = ""
}

// SetSubmitIntent records the current submit intent for intent-aware validation or submission flows.
func (parseF Form[T]) SetSubmitIntent(parseIntent string) {
	if parseF.state == nil || parseF.mu == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	parseF.state.submitIntent = strings.TrimSpace(parseIntent)
}

// SubmitIntent returns the most recently selected submit intent.
func (parseF Form[T]) SubmitIntent() string {
	if parseF.state == nil || parseF.mu == nil {
		return ""
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.submitIntent
}

// Update replaces the current form value using the previous value.
func (parseF Form[T]) Update(parseFn func(T) T) {
	if parseF.state == nil || parseF.mu == nil || parseFn == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	parseF.state.value = parseFn(parseF.state.value)
	parseF.state.dirty = computeDirtyFields(parseF.state.initial, parseF.state.value)
	parseF.state.formError = ""
}

// SetField updates one named struct field and marks it touched.
func (parseF Form[T]) SetField(parseName string, parseValue any) bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	parseNextValue, parseOk := assignNamedField(parseF.state.value, parseName, parseValue)
	if !parseOk {
		return false
	}
	parseF.state.value = parseNextValue
	if parseF.state.touched == nil {
		parseF.state.touched = map[string]bool{}
	}
	if parseF.state.dirty == nil {
		parseF.state.dirty = map[string]bool{}
	}
	if parseF.state.errors == nil {
		parseF.state.errors = FieldErrors{}
	}
	parseF.state.touched[parseName] = true
	if parseInitialField, parseOk2 := readNamedField(parseF.state.initial, parseName); parseOk2 {
		parseF.state.dirty[parseName] = !reflect.DeepEqual(parseInitialField, valueOfNamedField(parseF.state.value, parseName))
	} else {
		parseF.state.dirty[parseName] = true
	}
	delete(parseF.state.errors, parseName)
	parseF.state.formError = ""
	return true
}

// HasField reports whether T has a field addressable by name (the same names SetField/Touch use).
// SetField returns false for both an unknown field AND a type mismatch; HasField lets a test or a
// dev-time guard catch a misspelled field name specifically.
func (parseF Form[T]) HasField(parseName string) bool {
	var parseValue T
	if parseF.state != nil && parseF.mu != nil {
		parseF.mu.RLock() // read-only snapshot; mirror the other read accessors
		parseValue = parseF.state.value
		parseF.mu.RUnlock()
	}
	_, parseOk := readNamedField(parseValue, parseName)
	return parseOk
}

// MustSetField is SetField that panics when the field does not exist or the value's type does not
// match — turning a silent no-op into a loud failure that surfaces a typo'd field name during
// development. Prefer it where the field name is a hardcoded literal.
func (parseF Form[T]) MustSetField(parseName string, parseValue any) {
	if !parseF.SetField(parseName, parseValue) {
		panic("ui: MustSetField: unknown field or mismatched value type: " + parseName)
	}
}

// Touch marks one field as touched.
func (parseF Form[T]) Touch(parseName string) {
	if parseF.state == nil || parseF.mu == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	if parseF.state.touched == nil {
		parseF.state.touched = map[string]bool{}
	}
	parseF.state.touched[parseName] = true
}

// Touched reports whether a field has been touched.
func (parseF Form[T]) Touched(parseName string) bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.touched[parseName]
}

// Dirty reports whether a field differs from its initial value.
func (parseF Form[T]) Dirty(parseName string) bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.dirty[parseName]
}

// SetErrors replaces the current field error map.
func (parseF Form[T]) SetErrors(parseErrors FieldErrors) {
	if parseF.state == nil || parseF.mu == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	parseF.state.errors = cloneFieldErrors(parseErrors)
	parseF.state.validated = true
	parseF.state.validating = false
}

// SetFormError sets the form-level error message.
func (parseF Form[T]) SetFormError(parseMessage string) {
	if parseF.state == nil || parseF.mu == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	parseF.state.formError = parseMessage
}

// Errors returns a copy of the current field error map.
func (parseF Form[T]) Errors() FieldErrors {
	if parseF.state == nil || parseF.mu == nil {
		return FieldErrors{}
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return cloneFieldErrors(parseF.state.errors)
}

// Error returns the field error for name.
func (parseF Form[T]) Error(parseName string) string {
	if parseF.state == nil || parseF.mu == nil {
		return ""
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.errors[parseName]
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
	if parseF.state == nil || parseF.mu == nil {
		return parseStatus
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	parseStatus.Touched = parseF.state.touched[parseName]
	parseStatus.Dirty = parseF.state.dirty[parseName]
	parseStatus.Pending = parseF.state.validating || parseF.state.submitting
	parseStatus.Error = parseF.state.errors[parseName]
	return parseStatus
}

// FormError returns the form-level error message.
func (parseF Form[T]) FormError() string {
	if parseF.state == nil || parseF.mu == nil {
		return ""
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.formError
}

// TouchedAny reports whether any field has been touched.
func (parseF Form[T]) TouchedAny() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	for _, parseTouched := range parseF.state.touched {
		if parseTouched {
			return true
		}
	}
	return false
}

// DirtyAny reports whether any field differs from its initial value.
func (parseF Form[T]) DirtyAny() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	for _, parseDirty := range parseF.state.dirty {
		if parseDirty {
			return true
		}
	}
	return false
}

// HasErrors reports whether the form currently has field or form-level errors.
func (parseF Form[T]) HasErrors() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return len(parseF.state.errors) > 0 || parseF.state.formError != ""
}

// ApplyServerErrors projects a structured server validation response onto the form state.
func (parseF Form[T]) ApplyServerErrors(parseResponse ServerFormErrors) bool {
	if parseF.state == nil || parseF.mu == nil {
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
	if parseF.state == nil || parseF.mu == nil {
		return true
	}
	if parseValidate == nil {
		parseF.SetFormError("")
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
	if parseF.state == nil || parseF.mu == nil {
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
	if parseF.state == nil || parseF.mu == nil {
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
	parseF.mu.Lock()
	parseSequence := parseF.state.validateSeq + 1
	parseF.state.validating = true
	parseF.state.validated = false
	parseF.state.formError = ""
	parseF.state.validateSeq = parseSequence
	parseF.mu.Unlock()

	go func(parseValue T, parseExpectedSeq int) {
		defer runtime.RecoverContainedPanic("ui", "UseForm validator")
		parseErrors, parseFormError := parseValidate(parseValue)
		isParseValid := len(parseErrors) == 0 && parseFormError == ""
		parseF.mu.Lock()
		if parseF.state != nil && parseF.state.validateSeq == parseExpectedSeq {
			parseF.state.errors = cloneFieldErrors(parseErrors)
			parseF.state.formError = parseFormError
			parseF.state.validating = false
			parseF.state.validated = true
		}
		parseF.mu.Unlock()
		if parseOnComplete != nil {
			parseOnComplete(isParseValid)
		}
	}(parseSnapshot, parseSequence)
}

// Submit runs the submit function in a goroutine and updates submission lifecycle state.
func (parseF Form[T]) Submit(parseRun func(T) error) {
	if parseF.state == nil || parseF.mu == nil || parseRun == nil {
		return
	}
	parseSnapshot := parseF.Get()
	parseF.mu.Lock()
	parseF.state.submitIntent = ""
	parseF.state.submitting = true
	parseF.state.submitted = false
	parseF.state.submitError = nil
	parseF.state.formError = ""
	parseF.mu.Unlock()

	go func(parseValue T) {
		defer runtime.RecoverContainedPanic("ui", "UseForm submit runner")
		parseErr := parseRun(parseValue)
		parseF.mu.Lock()
		if parseF.state != nil {
			parseF.state.submitting = false
			parseF.state.submitError = parseErr
			parseF.state.submitted = parseErr == nil
			parseF.state.submitIntent = ""
			if parseErr != nil {
				parseF.state.formError = parseErr.Error()
			} else {
				parseF.state.formError = ""
			}
		}
		parseF.mu.Unlock()
	}(parseSnapshot)
}

// SubmitWithIntent runs the submit function with an explicit intent and tracks that intent while submission is pending.
func (parseF Form[T]) SubmitWithIntent(parseIntent string, parseRun func(T, string) error) {
	if parseF.state == nil || parseF.mu == nil || parseRun == nil {
		return
	}
	parseTrimmedIntent := strings.TrimSpace(parseIntent)
	parseSnapshot := parseF.Get()
	parseF.mu.Lock()
	parseF.state.submitIntent = parseTrimmedIntent
	parseF.state.submitting = true
	parseF.state.submitted = false
	parseF.state.submitError = nil
	parseF.state.formError = ""
	parseF.mu.Unlock()

	go func(parseValue T, parseActiveIntent string) {
		defer runtime.RecoverContainedPanic("ui", "UseForm intent runner")
		parseErr := parseRun(parseValue, parseActiveIntent)
		parseF.mu.Lock()
		if parseF.state != nil {
			parseF.state.submitting = false
			parseF.state.submitError = parseErr
			parseF.state.submitted = parseErr == nil
			parseF.state.submitIntent = parseActiveIntent
			if parseErr != nil {
				parseF.state.formError = parseErr.Error()
			} else {
				parseF.state.formError = ""
			}
		}
		parseF.mu.Unlock()
	}(parseSnapshot, parseTrimmedIntent)
}

// Submitting reports whether a submission is in flight.
func (parseF Form[T]) Submitting() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.submitting
}

// Validating reports whether async validation is in flight.
func (parseF Form[T]) Validating() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.validating
}

// Validated reports whether validation has completed at least once.
func (parseF Form[T]) Validated() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.validated
}

// Submitted reports whether the last submission completed successfully.
func (parseF Form[T]) Submitted() bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.submitted
}

// IntentPending reports whether the given intent is the currently pending submit action.
func (parseF Form[T]) IntentPending(parseIntent string) bool {
	if parseF.state == nil || parseF.mu == nil {
		return false
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.submitting && parseF.state.submitIntent == strings.TrimSpace(parseIntent)
}

// SubmitError returns the last submission error.
func (parseF Form[T]) SubmitError() error {
	if parseF.state == nil || parseF.mu == nil {
		return nil
	}
	parseF.mu.RLock()
	defer parseF.mu.RUnlock()
	return parseF.state.submitError
}

// Reset restores the form to its initial value or the provided next value.
func (parseF Form[T]) Reset(parseNext ...T) {
	if parseF.state == nil || parseF.mu == nil {
		return
	}
	parseF.mu.Lock()
	defer parseF.mu.Unlock()
	if len(parseNext) > 0 {
		parseF.state.initial = parseNext[0]
		parseF.state.value = parseNext[0]
	} else {
		parseF.state.value = parseF.state.initial
	}
	parseF.state.submitIntent = ""
	parseF.state.touched = map[string]bool{}
	parseF.state.dirty = map[string]bool{}
	parseF.state.errors = FieldErrors{}
	parseF.state.formError = ""
	parseF.state.validating = false
	parseF.state.validated = false
	parseF.state.validateSeq = 0
	parseF.state.submitting = false
	parseF.state.submitted = false
	parseF.state.submitError = nil
}

// cloneFieldErrors is a core package helper.
func cloneFieldErrors(parseErrors FieldErrors) FieldErrors {
	if len(parseErrors) == 0 {
		return FieldErrors{}
	}
	parseClone := make(FieldErrors, len(parseErrors))
	maps.Copy(parseClone, parseErrors)
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
func assignNamedField[T any](parseTarget T, parseName string, parseValue any) (T, bool) {
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
func readNamedField[T any](parseValue T, parseName string) (any, bool) {
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
func valueOfNamedField[T any](parseValue T, parseName string) any {
	parseField, parseOk := readNamedField(parseValue, parseName)
	if !parseOk {
		return nil
	}
	return parseField
}
