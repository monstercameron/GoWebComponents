package validate_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/validate"
)

type signup struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,min=2,max=20"`
	Age   int    `json:"age" validate:"gte=13,lte=120"`
	Plan  string `json:"plan" validate:"oneof=free pro team"`
}

// TestValidStructPasses proves a fully-valid struct yields no errors.
func TestValidStructPasses(parseT *testing.T) {
	parseRes := validate.Struct(signup{Email: "a@b.com", Name: "Ada", Age: 30, Plan: "pro"})
	if !parseRes.Valid() {
		parseT.Fatalf("expected valid, got %v", parseRes.Errors)
	}
}

// TestRequired proves required catches empty strings, zero numbers, and nil slices.
func TestRequired(parseT *testing.T) {
	type form struct {
		S     string   `validate:"required"`
		Items []string `validate:"required"`
	}
	parseRes := validate.Struct(form{})
	parseFields := parseRes.Fields()
	if parseFields["S"] != "is required" {
		parseT.Fatalf("expected S required, got %v", parseFields)
	}
	if parseFields["Items"] != "is required" {
		parseT.Fatalf("expected Items required, got %v", parseFields)
	}
}

// TestStringLengthRules proves min/max measure string length and report characters.
func TestStringLengthRules(parseT *testing.T) {
	parseRes := validate.Struct(signup{Email: "a@b.com", Name: "A", Age: 30, Plan: "free"})
	if parseMsg := parseRes.Fields()["name"]; parseMsg != "must be at least 2 characters" {
		parseT.Fatalf("expected min-length message, got %q", parseMsg)
	}
	parseLong := validate.Struct(signup{Email: "a@b.com", Name: "ThisNameIsWayTooLongToFit", Age: 30, Plan: "free"})
	if parseMsg := parseLong.Fields()["name"]; parseMsg != "must be at most 20 characters" {
		parseT.Fatalf("expected max-length message, got %q", parseMsg)
	}
}

// TestNumericRules proves gte/lte compare numeric values.
func TestNumericRules(parseT *testing.T) {
	parseYoung := validate.Struct(signup{Email: "a@b.com", Name: "Ada", Age: 10, Plan: "free"})
	if parseMsg := parseYoung.Fields()["age"]; parseMsg != "must be at least 13" {
		parseT.Fatalf("expected gte message, got %q", parseMsg)
	}
	parseOld := validate.Struct(signup{Email: "a@b.com", Name: "Ada", Age: 999, Plan: "free"})
	if parseMsg := parseOld.Fields()["age"]; parseMsg != "must be at most 120" {
		parseT.Fatalf("expected lte message, got %q", parseMsg)
	}
}

// TestEmailRule proves the email format check.
func TestEmailRule(parseT *testing.T) {
	parseRes := validate.Struct(signup{Email: "not-an-email", Name: "Ada", Age: 30, Plan: "free"})
	if parseMsg := parseRes.Fields()["email"]; parseMsg != "must be a valid email address" {
		parseT.Fatalf("expected email message, got %q", parseMsg)
	}
}

// TestOneOf proves the membership rule and its message.
func TestOneOf(parseT *testing.T) {
	parseRes := validate.Struct(signup{Email: "a@b.com", Name: "Ada", Age: 30, Plan: "enterprise"})
	if parseMsg := parseRes.Fields()["plan"]; parseMsg != "must be one of: free, pro, team" {
		parseT.Fatalf("expected oneof message, got %q", parseMsg)
	}
}

// TestLenAndCharClasses proves len, alpha, alphanum, numeric.
func TestLenAndCharClasses(parseT *testing.T) {
	type form struct {
		Code string `validate:"len=4,alphanum"`
		Word string `validate:"alpha"`
		Zip  string `validate:"numeric"`
	}
	parseRes := validate.Struct(form{Code: "ab", Word: "abc123", Zip: "12a"})
	parseFields := parseRes.Fields()
	if parseFields["Code"] != "must be exactly 4 long" {
		parseT.Fatalf("len: %v", parseFields)
	}
	if parseFields["Word"] != "must contain only letters" {
		parseT.Fatalf("alpha: %v", parseFields)
	}
	if parseFields["Zip"] != "must contain only digits" {
		parseT.Fatalf("numeric: %v", parseFields)
	}
}

// TestNestedStructDottedPaths proves recursion into nested structs with dotted paths.
func TestNestedStructDottedPaths(parseT *testing.T) {
	type address struct {
		Zip string `json:"zip" validate:"required,numeric"`
	}
	type person struct {
		Name string  `json:"name" validate:"required"`
		Home address `json:"home"`
	}
	parseRes := validate.Struct(person{Name: "Ada", Home: address{Zip: "abc"}})
	if parseMsg := parseRes.Fields()["home.zip"]; parseMsg != "must contain only digits" {
		parseT.Fatalf("expected dotted nested path home.zip, got %v", parseRes.Fields())
	}
}

// TestNilPointerStructIsSafe proves a nil pointer-to-struct neither panics nor errors.
func TestNilPointerStructIsSafe(parseT *testing.T) {
	type form struct {
		S string `validate:"required"`
	}
	var parsePtr *form
	if parseRes := validate.Struct(parsePtr); !parseRes.Valid() {
		parseT.Fatalf("nil pointer should be valid (nothing to check), got %v", parseRes.Errors)
	}
	// Pointer value is validated through the pointer.
	if parseRes := validate.Struct(&form{}); parseRes.Valid() {
		parseT.Fatal("expected pointer-to-struct to be validated")
	}
}

// TestFieldsReturnsFirstErrorPerField proves Fields() collapses to one message and
// is assignable to a map[string]string (the ui.FieldErrors shape).
func TestFieldsReturnsFirstErrorPerField(parseT *testing.T) {
	type form struct {
		Name string `validate:"required,min=3"`
	}
	parseRes := validate.Struct(form{Name: ""})
	var parseAsFieldErrors map[string]string = parseRes.Fields() // compile-time shape check
	if len(parseAsFieldErrors) != 1 || parseAsFieldErrors["Name"] != "is required" {
		parseT.Fatalf("expected one first-error per field, got %v", parseAsFieldErrors)
	}
	if parseAll := parseRes.All()["Name"]; len(parseAll) < 1 {
		parseT.Fatalf("All() should list every failed rule, got %v", parseAll)
	}
}

// TestErrorInterface proves a Result renders as an error for server handlers.
func TestErrorInterface(parseT *testing.T) {
	type form struct {
		S string `validate:"required"`
	}
	var parseErr error = validate.Struct(form{})
	if parseErr.Error() == "" {
		parseT.Fatal("expected non-empty error string for an invalid struct")
	}
	if validate.Struct(form{S: "ok"}).Error() != "" {
		parseT.Fatal("expected empty error string for a valid struct")
	}
}

// TestUnexportedAndUntaggedSkipped proves unexported fields and fields without a
// validate tag are ignored, and unknown rules don't fail.
func TestUnexportedAndUntaggedSkipped(parseT *testing.T) {
	type form struct {
		Public  string `validate:"required"`
		ignored string //nolint
		Other   string `validate:"someunknownrule"`
	}
	parseRes := validate.Struct(form{Public: "x", Other: ""})
	_ = parseRes
	if !parseRes.Valid() {
		parseT.Fatalf("expected valid (untagged/unexported/unknown ignored), got %v", parseRes.Errors)
	}
}
