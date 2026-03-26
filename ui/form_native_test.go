//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"errors"
	"testing"
	"time"
)

type sampleFormModel struct {
	Name   string
	Count  int
	Active bool
}

func waitForCondition(parseT *testing.T, parseTimeout time.Duration, parseCheck func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		if parseCheck() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	parseT.Fatal("timed out waiting for condition")
}

func TestCSRFAndServerActionHelpers(parseT *testing.T) {
	parseToken := NewCSRFToken("abc")
	parseName, parseValue := parseToken.Header()
	if parseName != DefaultCSRFHeaderName || parseValue != "abc" {
		parseT.Fatalf("unexpected default csrf header: %q=%q", parseName, parseValue)
	}
	parseFieldName, parseFieldValue := parseToken.FormField()
	if parseFieldName != DefaultCSRFFormFieldName || parseFieldValue != "abc" {
		parseT.Fatalf("unexpected default csrf form field: %q=%q", parseFieldName, parseFieldValue)
	}

	parseCustom := CSRFToken{Value: "x", HeaderName: " X-Custom ", FormFieldName: " _csrf "}
	if parseName2, _ := parseCustom.Header(); parseName2 != "X-Custom" {
		parseT.Fatalf("expected custom header name to trim whitespace, got %q", parseName2)
	}
	if parseField, _ := parseCustom.FormField(); parseField != "_csrf" {
		parseT.Fatalf("expected custom form field to trim whitespace, got %q", parseField)
	}

	parseFormErrors := ServerFormErrors{Error: "base", Message: " preferred "}
	if parseGot := parseFormErrors.FormMessage(); parseGot != "preferred" {
		parseT.Fatalf("expected message priority, got %q", parseGot)
	}
	parseFormErrors.Message = ""
	if parseGot2 := parseFormErrors.FormMessage(); parseGot2 != "base" {
		parseT.Fatalf("expected fallback to error, got %q", parseGot2)
	}

	parseResult := ServerActionResult{
		Error:   " action_error ",
		Message: " action_message ",
		Fields:  FieldErrors{"Name": "required"},
		Redirect: &ServerActionRedirect{
			Location: " /next ",
		},
		Refresh: &ServerActionRefresh{Revalidate: true},
	}
	parseOut := parseResult.FormErrors()
	if parseOut.Error != "action_error" || parseOut.Message != "action_message" || parseOut.Fields["Name"] != "required" {
		parseT.Fatalf("unexpected form errors projection: %#v", parseOut)
	}
	if !parseResult.HasRedirect() || parseResult.RedirectLocation() != "/next" {
		parseT.Fatalf("expected redirect helpers to surface location, got %q", parseResult.RedirectLocation())
	}
	if !parseResult.HasRefresh() {
		parseT.Fatal("expected refresh helper to detect revalidate flag")
	}

	parseNoRedirect := ServerActionResult{}
	if parseNoRedirect.HasRedirect() || parseNoRedirect.RedirectLocation() != "" {
		parseT.Fatalf("expected empty redirect helpers, got %#v", parseNoRedirect)
	}
	if parseNoRedirect.HasRefresh() {
		parseT.Fatal("expected no refresh when refresh config is nil")
	}
}

func TestFormBasicLifecycleAndFieldState(parseT *testing.T) {
	parseInitial := sampleFormModel{Name: "Atlas", Count: 2, Active: true}
	parseForm := UseForm(parseInitial)

	if parseGot := parseForm.Get(); parseGot.Name != "Atlas" || parseGot.Count != 2 || !parseGot.Active {
		parseT.Fatalf("unexpected initial form value: %#v", parseGot)
	}
	if parseForm.TouchedAny() || parseForm.DirtyAny() || parseForm.HasErrors() {
		parseT.Fatal("new form should be untouched, clean, and error-free")
	}

	parseForm.SetSubmitIntent(" save ")
	if parseGot2 := parseForm.SubmitIntent(); parseGot2 != "save" {
		parseT.Fatalf("expected submit intent trim, got %q", parseGot2)
	}

	parseForm.Set(sampleFormModel{Name: "Atlas Prime", Count: 2, Active: true})
	if !parseForm.Dirty("Name") || parseForm.Dirty("Count") {
		parseT.Fatalf("expected Name to be dirty and Count clean after Set, dirty=%v", parseForm.Dirty("Count"))
	}

	parseForm.Update(func(parseValue sampleFormModel) sampleFormModel {
		parseValue.Count = 5
		return parseValue
	})
	if parseGot3 := parseForm.Get(); parseGot3.Count != 5 {
		parseT.Fatalf("expected Update to mutate value, got %#v", parseGot3)
	}

	if parseOk := parseForm.SetField("Name", "Delta"); !parseOk {
		parseT.Fatal("expected SetField to update valid field")
	}
	if parseOk2 := parseForm.SetField("Count", int32(7)); !parseOk2 {
		parseT.Fatal("expected SetField to convert compatible numeric field type")
	}
	if parseOk3 := parseForm.SetField("Missing", "x"); parseOk3 {
		parseT.Fatal("expected SetField to reject missing field")
	}
	if parseOk4 := parseForm.SetField("Count", "not-int"); parseOk4 {
		parseT.Fatal("expected SetField to reject incompatible value type")
	}

	parseForm.Touch("Active")
	if !parseForm.Touched("Name") || !parseForm.Touched("Active") {
		parseT.Fatal("expected SetField/Touch to mark fields as touched")
	}
	if !parseForm.Dirty("Name") || !parseForm.Dirty("Count") {
		parseT.Fatal("expected Name and Count to be dirty after updates")
	}

	parseForm.SetErrors(FieldErrors{"Name": "required"})
	if !parseForm.HasFieldError("Name") || parseForm.FieldMessage("Name") != "required" {
		parseT.Fatalf("expected Name field error, got %q", parseForm.FieldMessage("Name"))
	}
	parseStatus := parseForm.FieldStatus("Name")
	if !parseStatus.Touched || !parseStatus.Dirty || parseStatus.Pending || parseStatus.Error != "required" {
		parseT.Fatalf("unexpected field status: %#v", parseStatus)
	}
	if !parseForm.Validated() || parseForm.Validating() {
		parseT.Fatalf("expected form to be validated and not validating")
	}

	parseForm.SetFormError("submit failed")
	if !parseForm.HasErrors() || parseForm.FormError() != "submit failed" {
		parseT.Fatalf("expected form-level error to be tracked, got %q", parseForm.FormError())
	}

	if parseOk5 := parseForm.ApplyServerErrors(ServerFormErrors{Fields: FieldErrors{"Count": "bad"}}); parseOk5 {
		parseT.Fatal("expected ApplyServerErrors to report not-clean when field errors exist")
	}
	if parseOk6 := parseForm.ApplyServerActionResult(ServerActionResult{Message: "invalid"}); parseOk6 {
		parseT.Fatal("expected ApplyServerActionResult to report not-clean when form message exists")
	}
	if parseOk7 := parseForm.ApplyServerErrors(ServerFormErrors{}); !parseOk7 {
		parseT.Fatal("expected empty server errors to be clean")
	}

	parseForm.Reset()
	if parseForm.TouchedAny() || parseForm.DirtyAny() || parseForm.HasErrors() || parseForm.SubmitIntent() != "" {
		parseT.Fatal("expected Reset() to clear touched, dirty, errors, and intent")
	}
	if parseGot4 := parseForm.Get(); parseGot4.Name != parseInitial.Name || parseGot4.Count != parseInitial.Count || parseGot4.Active != parseInitial.Active {
		parseT.Fatalf("expected Reset() to restore initial value, got %#v", parseGot4)
	}

	parseNext := sampleFormModel{Name: "Replacement", Count: 10}
	parseForm.Reset(parseNext)
	if parseGot5 := parseForm.Get(); parseGot5.Name != "Replacement" || parseGot5.Count != 10 {
		parseT.Fatalf("expected Reset(next) to replace baseline, got %#v", parseGot5)
	}
}

func TestFormValidationAndSubmissionAsyncPaths(parseT *testing.T) {
	parseForm := UseForm(sampleFormModel{Name: "Atlas", Count: 1})

	if parseOk := parseForm.Validate(nil); !parseOk {
		parseT.Fatal("nil validator should pass")
	}
	if parseOk2 := parseForm.Validate(func(parseValue sampleFormModel) FieldErrors {
		if parseValue.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	}); !parseOk2 {
		parseT.Fatal("expected validator to pass for non-empty name")
	}
	parseForm.SetField("Name", "")
	if parseOk3 := parseForm.Validate(func(parseValue2 sampleFormModel) FieldErrors {
		if parseValue2.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	}); parseOk3 {
		parseT.Fatal("expected validator to fail for empty name")
	}

	if parseOk4 := parseForm.ValidateIntent(" publish ", nil); !parseOk4 {
		parseT.Fatal("nil intent validator should pass")
	}
	if parseGot := parseForm.SubmitIntent(); parseGot != "publish" {
		parseT.Fatalf("expected intent to be recorded, got %q", parseGot)
	}
	if parseOk5 := parseForm.ValidateIntent("delete", func(parseValue3 sampleFormModel, parseIntent string) FieldErrors {
		if parseIntent == "delete" {
			return FieldErrors{"Name": "cannot delete"}
		}
		return nil
	}); parseOk5 {
		parseT.Fatal("expected intent validation error")
	}

	parseValidateDone := make(chan bool, 2)
	parseForm.ValidateAsync(func(parseValue4 sampleFormModel) (FieldErrors, string) {
		time.Sleep(20 * time.Millisecond)
		return FieldErrors{"Name": "stale"}, "stale"
	}, func(isOk bool) { parseValidateDone <- isOk })
	parseForm.ValidateAsync(func(parseValue5 sampleFormModel) (FieldErrors, string) {
		return nil, ""
	}, func(isOk2 bool) { parseValidateDone <- isOk2 })

	waitForCondition(parseT, 200*time.Millisecond, func() bool { return parseForm.Validated() && !parseForm.Validating() })
	if parseForm.HasErrors() {
		parseT.Fatalf("expected second validation result to win, got errors=%#v form=%q", parseForm.Errors(), parseForm.FormError())
	}
	<-parseValidateDone
	<-parseValidateDone

	parseSubmitRelease := make(chan struct{})
	parseForm.Submit(func(parseValue6 sampleFormModel) error {
		<-parseSubmitRelease
		return nil
	})
	waitForCondition(parseT, 200*time.Millisecond, func() bool { return parseForm.Submitting() })
	if parseIntentMismatch := !parseForm.IntentPending("publish") && parseForm.SubmitIntent() != ""; parseIntentMismatch {
		parseT.Fatal("expected intent mismatch check to stay false for non-matching intent")
	}
	close(parseSubmitRelease)
	waitForCondition(parseT, 200*time.Millisecond, func() bool { return !parseForm.Submitting() })
	if !parseForm.Submitted() || parseForm.SubmitError() != nil || parseForm.FormError() != "" {
		parseT.Fatalf("expected successful submission state, submitted=%t err=%v form=%q", parseForm.Submitted(), parseForm.SubmitError(), parseForm.FormError())
	}

	parseSubmitErrRelease := make(chan struct{})
	parseExpectedSubmitErr := errors.New("submit failed")
	parseForm.SubmitWithIntent("  archive  ", func(parseValue7 sampleFormModel, parseIntent2 string) error {
		if parseIntent2 != "archive" {
			parseT.Fatalf("expected trimmed intent archive, got %q", parseIntent2)
		}
		<-parseSubmitErrRelease
		return parseExpectedSubmitErr
	})
	waitForCondition(parseT, 200*time.Millisecond, func() bool { return parseForm.Submitting() && parseForm.IntentPending("archive") })
	close(parseSubmitErrRelease)
	waitForCondition(parseT, 200*time.Millisecond, func() bool { return !parseForm.Submitting() })
	if parseForm.Submitted() || !errors.Is(parseForm.SubmitError(), parseExpectedSubmitErr) || parseForm.FormError() != parseExpectedSubmitErr.Error() {
		parseT.Fatalf("expected failed submission state, submitted=%t err=%v form=%q", parseForm.Submitted(), parseForm.SubmitError(), parseForm.FormError())
	}

	parseZero := Form[sampleFormModel]{}
	if parseZero.Get() != (sampleFormModel{}) {
		parseT.Fatal("zero-value form should return zero value on Get")
	}
	parseZero.Set(sampleFormModel{Name: "ignored"})
	parseZero.Update(func(parseValue8 sampleFormModel) sampleFormModel { parseValue8.Name = "ignored"; return parseValue8 })
	if parseOk6 := parseZero.SetField("Name", "ignored"); parseOk6 {
		parseT.Fatal("zero-value form should not allow SetField")
	}
	parseZero.Touch("Name")
	if parseZero.Touched("Name") || parseZero.Dirty("Name") || parseZero.Submitting() || parseZero.Validating() || parseZero.Validated() || parseZero.Submitted() || parseZero.IntentPending("x") {
		parseT.Fatal("zero-value form guards should keep false state")
	}
	if parseZero.SubmitError() != nil {
		parseT.Fatal("zero-value form submit error should be nil")
	}
	parseZero.ValidateAsync(nil, func(isOk3 bool) {
		if !isOk3 {
			parseT.Fatal("zero-value form async validate should complete as true")
		}
	})
}

func TestFormInternalHelpers(parseT *testing.T) {
	parseErrorsMap := cloneFieldErrors(FieldErrors{"a": "b"})
	parseErrorsMap["a"] = "changed"
	parseOriginal := FieldErrors{"a": "b"}
	if cloneFieldErrors(parseOriginal)["a"] != "b" {
		parseT.Fatal("expected cloneFieldErrors to copy values")
	}
	if len(cloneFieldErrors(nil)) != 0 {
		parseT.Fatal("expected cloneFieldErrors(nil) to return empty map")
	}

	parseInitial := sampleFormModel{Name: "A", Count: 1}
	parseCurrent := sampleFormModel{Name: "B", Count: 2}
	parseDirty := computeDirtyFields(parseInitial, parseCurrent)
	if !parseDirty["Name"] || !parseDirty["Count"] {
		parseT.Fatalf("expected Name and Count dirty, got %#v", parseDirty)
	}
	if len(computeDirtyFields(1, 2)) != 0 {
		parseT.Fatal("non-struct dirty computation should return empty map")
	}

	parseNext, parseOk := assignNamedField(parseInitial, "Count", int32(9))
	if !parseOk || parseNext.Count != 9 {
		parseT.Fatalf("expected assignNamedField conversion success, got ok=%t next=%#v", parseOk, parseNext)
	}
	if _, parseOk2 := assignNamedField(parseInitial, "Count", struct{}{}); parseOk2 {
		parseT.Fatal("expected assignNamedField to reject non-convertible type")
	}
	if _, parseOk3 := assignNamedField(parseInitial, "Missing", 1); parseOk3 {
		parseT.Fatal("expected assignNamedField to reject unknown field")
	}
	if _, parseOk4 := assignNamedField(parseInitial, "Count", nil); parseOk4 {
		parseT.Fatal("expected assignNamedField to reject invalid nil value")
	}

	if parseValue, parseOk5 := readNamedField(parseInitial, "Name"); !parseOk5 || parseValue.(string) != "A" {
		parseT.Fatalf("expected readNamedField Name=A, got ok=%t value=%#v", parseOk5, parseValue)
	}
	if _, parseOk6 := readNamedField(parseInitial, "Missing"); parseOk6 {
		parseT.Fatal("expected readNamedField to reject unknown field")
	}
	if _, parseOk7 := readNamedField(10, "Missing"); parseOk7 {
		parseT.Fatal("expected readNamedField to reject non-struct input")
	}
	if parseGot := valueOfNamedField(parseInitial, "Name"); parseGot != "A" {
		parseT.Fatalf("expected valueOfNamedField Name=A, got %#v", parseGot)
	}
	if parseGot2 := valueOfNamedField(parseInitial, "Missing"); parseGot2 != nil {
		parseT.Fatalf("expected valueOfNamedField missing to return nil, got %#v", parseGot2)
	}
}

func TestFormErrorsReturnsClone(parseT *testing.T) {
	parseForm := UseForm(sampleFormModel{Name: "Atlas"})
	parseForm.SetErrors(FieldErrors{
		"Name":  "required",
		"Count": "must be positive",
	})

	parseErrorsOne := parseForm.Errors()
	parseErrorsOne["Name"] = "mutated"
	parseErrorsTwo := parseForm.Errors()
	if parseErrorsTwo["Name"] != "required" {
		parseT.Fatalf("expected Errors() to return a cloned map, got %#v", parseErrorsTwo)
	}

	var parseZero Form[sampleFormModel]
	if parseErrs := parseZero.Errors(); len(parseErrs) != 0 {
		parseT.Fatalf("expected zero-value form errors to be empty, got %#v", parseErrs)
	}
}
