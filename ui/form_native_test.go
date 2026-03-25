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
	hidden string
}

func waitForCondition(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

func TestCSRFAndServerActionHelpers(t *testing.T) {
	token := NewCSRFToken("abc")
	name, value := token.Header()
	if name != DefaultCSRFHeaderName || value != "abc" {
		t.Fatalf("unexpected default csrf header: %q=%q", name, value)
	}
	fieldName, fieldValue := token.FormField()
	if fieldName != DefaultCSRFFormFieldName || fieldValue != "abc" {
		t.Fatalf("unexpected default csrf form field: %q=%q", fieldName, fieldValue)
	}

	custom := CSRFToken{Value: "x", HeaderName: " X-Custom ", FormFieldName: " _csrf "}
	if name, _ := custom.Header(); name != "X-Custom" {
		t.Fatalf("expected custom header name to trim whitespace, got %q", name)
	}
	if field, _ := custom.FormField(); field != "_csrf" {
		t.Fatalf("expected custom form field to trim whitespace, got %q", field)
	}

	formErrors := ServerFormErrors{Error: "base", Message: " preferred "}
	if got := formErrors.FormMessage(); got != "preferred" {
		t.Fatalf("expected message priority, got %q", got)
	}
	formErrors.Message = ""
	if got := formErrors.FormMessage(); got != "base" {
		t.Fatalf("expected fallback to error, got %q", got)
	}

	result := ServerActionResult{
		Error:   " action_error ",
		Message: " action_message ",
		Fields:  FieldErrors{"Name": "required"},
		Redirect: &ServerActionRedirect{
			Location: " /next ",
		},
		Refresh: &ServerActionRefresh{Revalidate: true},
	}
	out := result.FormErrors()
	if out.Error != "action_error" || out.Message != "action_message" || out.Fields["Name"] != "required" {
		t.Fatalf("unexpected form errors projection: %#v", out)
	}
	if !result.HasRedirect() || result.RedirectLocation() != "/next" {
		t.Fatalf("expected redirect helpers to surface location, got %q", result.RedirectLocation())
	}
	if !result.HasRefresh() {
		t.Fatal("expected refresh helper to detect revalidate flag")
	}

	noRedirect := ServerActionResult{}
	if noRedirect.HasRedirect() || noRedirect.RedirectLocation() != "" {
		t.Fatalf("expected empty redirect helpers, got %#v", noRedirect)
	}
	if noRedirect.HasRefresh() {
		t.Fatal("expected no refresh when refresh config is nil")
	}
}

func TestFormBasicLifecycleAndFieldState(t *testing.T) {
	initial := sampleFormModel{Name: "Atlas", Count: 2, Active: true}
	form := UseForm(initial)

	if got := form.Get(); got.Name != "Atlas" || got.Count != 2 || !got.Active {
		t.Fatalf("unexpected initial form value: %#v", got)
	}
	if form.TouchedAny() || form.DirtyAny() || form.HasErrors() {
		t.Fatal("new form should be untouched, clean, and error-free")
	}

	form.SetSubmitIntent(" save ")
	if got := form.SubmitIntent(); got != "save" {
		t.Fatalf("expected submit intent trim, got %q", got)
	}

	form.Set(sampleFormModel{Name: "Atlas Prime", Count: 2, Active: true})
	if !form.Dirty("Name") || form.Dirty("Count") {
		t.Fatalf("expected Name to be dirty and Count clean after Set, dirty=%v", form.Dirty("Count"))
	}

	form.Update(func(value sampleFormModel) sampleFormModel {
		value.Count = 5
		return value
	})
	if got := form.Get(); got.Count != 5 {
		t.Fatalf("expected Update to mutate value, got %#v", got)
	}

	if ok := form.SetField("Name", "Delta"); !ok {
		t.Fatal("expected SetField to update valid field")
	}
	if ok := form.SetField("Count", int32(7)); !ok {
		t.Fatal("expected SetField to convert compatible numeric field type")
	}
	if ok := form.SetField("Missing", "x"); ok {
		t.Fatal("expected SetField to reject missing field")
	}
	if ok := form.SetField("Count", "not-int"); ok {
		t.Fatal("expected SetField to reject incompatible value type")
	}

	form.Touch("Active")
	if !form.Touched("Name") || !form.Touched("Active") {
		t.Fatal("expected SetField/Touch to mark fields as touched")
	}
	if !form.Dirty("Name") || !form.Dirty("Count") {
		t.Fatal("expected Name and Count to be dirty after updates")
	}

	form.SetErrors(FieldErrors{"Name": "required"})
	if !form.HasFieldError("Name") || form.FieldMessage("Name") != "required" {
		t.Fatalf("expected Name field error, got %q", form.FieldMessage("Name"))
	}
	status := form.FieldStatus("Name")
	if !status.Touched || !status.Dirty || status.Pending || status.Error != "required" {
		t.Fatalf("unexpected field status: %#v", status)
	}
	if !form.Validated() || form.Validating() {
		t.Fatalf("expected form to be validated and not validating")
	}

	form.SetFormError("submit failed")
	if !form.HasErrors() || form.FormError() != "submit failed" {
		t.Fatalf("expected form-level error to be tracked, got %q", form.FormError())
	}

	if ok := form.ApplyServerErrors(ServerFormErrors{Fields: FieldErrors{"Count": "bad"}}); ok {
		t.Fatal("expected ApplyServerErrors to report not-clean when field errors exist")
	}
	if ok := form.ApplyServerActionResult(ServerActionResult{Message: "invalid"}); ok {
		t.Fatal("expected ApplyServerActionResult to report not-clean when form message exists")
	}
	if ok := form.ApplyServerErrors(ServerFormErrors{}); !ok {
		t.Fatal("expected empty server errors to be clean")
	}

	form.Reset()
	if form.TouchedAny() || form.DirtyAny() || form.HasErrors() || form.SubmitIntent() != "" {
		t.Fatal("expected Reset() to clear touched, dirty, errors, and intent")
	}
	if got := form.Get(); got.Name != initial.Name || got.Count != initial.Count || got.Active != initial.Active {
		t.Fatalf("expected Reset() to restore initial value, got %#v", got)
	}

	next := sampleFormModel{Name: "Replacement", Count: 10}
	form.Reset(next)
	if got := form.Get(); got.Name != "Replacement" || got.Count != 10 {
		t.Fatalf("expected Reset(next) to replace baseline, got %#v", got)
	}
}

func TestFormValidationAndSubmissionAsyncPaths(t *testing.T) {
	form := UseForm(sampleFormModel{Name: "Atlas", Count: 1})

	if ok := form.Validate(nil); !ok {
		t.Fatal("nil validator should pass")
	}
	if ok := form.Validate(func(value sampleFormModel) FieldErrors {
		if value.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	}); !ok {
		t.Fatal("expected validator to pass for non-empty name")
	}
	form.SetField("Name", "")
	if ok := form.Validate(func(value sampleFormModel) FieldErrors {
		if value.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	}); ok {
		t.Fatal("expected validator to fail for empty name")
	}

	if ok := form.ValidateIntent(" publish ", nil); !ok {
		t.Fatal("nil intent validator should pass")
	}
	if got := form.SubmitIntent(); got != "publish" {
		t.Fatalf("expected intent to be recorded, got %q", got)
	}
	if ok := form.ValidateIntent("delete", func(value sampleFormModel, intent string) FieldErrors {
		if intent == "delete" {
			return FieldErrors{"Name": "cannot delete"}
		}
		return nil
	}); ok {
		t.Fatal("expected intent validation error")
	}

	validateDone := make(chan bool, 2)
	form.ValidateAsync(func(value sampleFormModel) (FieldErrors, string) {
		time.Sleep(20 * time.Millisecond)
		return FieldErrors{"Name": "stale"}, "stale"
	}, func(ok bool) { validateDone <- ok })
	form.ValidateAsync(func(value sampleFormModel) (FieldErrors, string) {
		return nil, ""
	}, func(ok bool) { validateDone <- ok })

	waitForCondition(t, 200*time.Millisecond, func() bool { return form.Validated() && !form.Validating() })
	if form.HasErrors() {
		t.Fatalf("expected second validation result to win, got errors=%#v form=%q", form.Errors(), form.FormError())
	}
	<-validateDone
	<-validateDone

	submitRelease := make(chan struct{})
	form.Submit(func(value sampleFormModel) error {
		<-submitRelease
		return nil
	})
	waitForCondition(t, 200*time.Millisecond, func() bool { return form.Submitting() })
	if !form.IntentPending("publish") && form.SubmitIntent() != "" {
		// keep branch coverage for intent-pending check on non-matching intent
	}
	close(submitRelease)
	waitForCondition(t, 200*time.Millisecond, func() bool { return !form.Submitting() })
	if !form.Submitted() || form.SubmitError() != nil || form.FormError() != "" {
		t.Fatalf("expected successful submission state, submitted=%t err=%v form=%q", form.Submitted(), form.SubmitError(), form.FormError())
	}

	submitErrRelease := make(chan struct{})
	expectedSubmitErr := errors.New("submit failed")
	form.SubmitWithIntent("  archive  ", func(value sampleFormModel, intent string) error {
		if intent != "archive" {
			t.Fatalf("expected trimmed intent archive, got %q", intent)
		}
		<-submitErrRelease
		return expectedSubmitErr
	})
	waitForCondition(t, 200*time.Millisecond, func() bool { return form.Submitting() && form.IntentPending("archive") })
	close(submitErrRelease)
	waitForCondition(t, 200*time.Millisecond, func() bool { return !form.Submitting() })
	if form.Submitted() || !errors.Is(form.SubmitError(), expectedSubmitErr) || form.FormError() != expectedSubmitErr.Error() {
		t.Fatalf("expected failed submission state, submitted=%t err=%v form=%q", form.Submitted(), form.SubmitError(), form.FormError())
	}

	zero := Form[sampleFormModel]{}
	if zero.Get() != (sampleFormModel{}) {
		t.Fatal("zero-value form should return zero value on Get")
	}
	zero.Set(sampleFormModel{Name: "ignored"})
	zero.Update(func(value sampleFormModel) sampleFormModel { value.Name = "ignored"; return value })
	if ok := zero.SetField("Name", "ignored"); ok {
		t.Fatal("zero-value form should not allow SetField")
	}
	zero.Touch("Name")
	if zero.Touched("Name") || zero.Dirty("Name") || zero.Submitting() || zero.Validating() || zero.Validated() || zero.Submitted() || zero.IntentPending("x") {
		t.Fatal("zero-value form guards should keep false state")
	}
	if zero.SubmitError() != nil {
		t.Fatal("zero-value form submit error should be nil")
	}
	zero.ValidateAsync(nil, func(ok bool) {
		if !ok {
			t.Fatal("zero-value form async validate should complete as true")
		}
	})
}

func TestFormInternalHelpers(t *testing.T) {
	errorsMap := cloneFieldErrors(FieldErrors{"a": "b"})
	errorsMap["a"] = "changed"
	original := FieldErrors{"a": "b"}
	if cloneFieldErrors(original)["a"] != "b" {
		t.Fatal("expected cloneFieldErrors to copy values")
	}
	if len(cloneFieldErrors(nil)) != 0 {
		t.Fatal("expected cloneFieldErrors(nil) to return empty map")
	}

	initial := sampleFormModel{Name: "A", Count: 1}
	current := sampleFormModel{Name: "B", Count: 2}
	dirty := computeDirtyFields(initial, current)
	if !dirty["Name"] || !dirty["Count"] {
		t.Fatalf("expected Name and Count dirty, got %#v", dirty)
	}
	if len(computeDirtyFields(1, 2)) != 0 {
		t.Fatal("non-struct dirty computation should return empty map")
	}

	next, ok := assignNamedField(initial, "Count", int32(9))
	if !ok || next.Count != 9 {
		t.Fatalf("expected assignNamedField conversion success, got ok=%t next=%#v", ok, next)
	}
	if _, ok := assignNamedField(initial, "Count", struct{}{}); ok {
		t.Fatal("expected assignNamedField to reject non-convertible type")
	}
	if _, ok := assignNamedField(initial, "Missing", 1); ok {
		t.Fatal("expected assignNamedField to reject unknown field")
	}
	if _, ok := assignNamedField(initial, "Count", nil); ok {
		t.Fatal("expected assignNamedField to reject invalid nil value")
	}

	if value, ok := readNamedField(initial, "Name"); !ok || value.(string) != "A" {
		t.Fatalf("expected readNamedField Name=A, got ok=%t value=%#v", ok, value)
	}
	if _, ok := readNamedField(initial, "Missing"); ok {
		t.Fatal("expected readNamedField to reject unknown field")
	}
	if _, ok := readNamedField(10, "Missing"); ok {
		t.Fatal("expected readNamedField to reject non-struct input")
	}
	if got := valueOfNamedField(initial, "Name"); got != "A" {
		t.Fatalf("expected valueOfNamedField Name=A, got %#v", got)
	}
	if got := valueOfNamedField(initial, "Missing"); got != nil {
		t.Fatalf("expected valueOfNamedField missing to return nil, got %#v", got)
	}
}

func TestFormErrorsReturnsClone(t *testing.T) {
	form := UseForm(sampleFormModel{Name: "Atlas"})
	form.SetErrors(FieldErrors{
		"Name":  "required",
		"Count": "must be positive",
	})

	errorsOne := form.Errors()
	errorsOne["Name"] = "mutated"
	errorsTwo := form.Errors()
	if errorsTwo["Name"] != "required" {
		t.Fatalf("expected Errors() to return a cloned map, got %#v", errorsTwo)
	}

	var zero Form[sampleFormModel]
	if errs := zero.Errors(); len(errs) != 0 {
		t.Fatalf("expected zero-value form errors to be empty, got %#v", errs)
	}
}
