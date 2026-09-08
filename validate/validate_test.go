package validate_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/validate"
)

// TestRegisterRuleCustomValidation proves a registered custom rule participates in Struct
// validation by tag name, receives the tag argument, and reports its own message on failure.
func TestRegisterRuleCustomValidation(parseT *testing.T) {
	validate.RegisterRule("hasprefix", func(parseV any, parseArg string) (bool, string) {
		parseS, _ := parseV.(string)
		return strings.HasPrefix(parseS, parseArg), "must start with " + parseArg
	})

	type doc struct {
		Code string `json:"code" validate:"required,hasprefix=SKU-"`
	}

	if parseRes := validate.Struct(doc{Code: "SKU-9"}); !parseRes.Valid() {
		parseT.Fatalf("expected valid, got %v", parseRes.Errors)
	}

	parseRes := validate.Struct(doc{Code: "X"})
	if parseRes.Valid() {
		parseT.Fatal("expected the custom rule to fail")
	}
	parseFound := false
	for _, parseErr := range parseRes.Errors {
		if parseErr.Rule == "hasprefix" {
			parseFound = true
			if parseErr.Message != "must start with SKU-" {
				parseT.Fatalf("unexpected custom message %q", parseErr.Message)
			}
		}
	}
	if !parseFound {
		parseT.Fatalf("expected a hasprefix failure, got %v", parseRes.Errors)
	}
}

// TestRegisterRuleCannotShadowBuiltinOrBreakForwardCompat proves a custom rule cannot override a
// built-in (required still fires), and an unregistered unknown rule is ignored, not an error.
func TestRegisterRuleCannotShadowBuiltinOrBreakForwardCompat(parseT *testing.T) {
	// Attempt to shadow the built-in "required" — must be ignored.
	validate.RegisterRule("required", func(parseV any, parseArg string) (bool, string) { return true, "" })
	type needsName struct {
		Name string `validate:"required"`
	}
	if validate.Struct(needsName{}).Valid() {
		parseT.Fatal("built-in required must not be shadowable via RegisterRule")
	}

	// An unknown, unregistered rule is ignored (forward compatible).
	type future struct {
		X string `validate:"somethingfromtomorrow"`
	}
	if !validate.Struct(future{X: "x"}).Valid() {
		parseT.Fatal("an unregistered unknown rule must be ignored, not fail validation")
	}
}

type signupCF struct {
	Password string `json:"password" validate:"required,min=8"`
	Confirm  string `json:"confirm"`
}

// TestCheckMergesTagAndCrossFieldRules proves Check runs the struct-tag rules AND the cross-field
// rules into one merged Result.
func TestCheckMergesTagAndCrossFieldRules(parseT *testing.T) {
	parseMatch := func(parseS signupCF) *validate.FieldError {
		if parseS.Password != parseS.Confirm {
			return validate.Fail("confirm", "must match password")
		}
		return nil
	}

	// Valid on both axes.
	if parseRes := validate.Check(signupCF{Password: "longenough", Confirm: "longenough"}, parseMatch); !parseRes.Valid() {
		parseT.Fatalf("expected valid, got %v", parseRes.Errors)
	}

	// Fails BOTH a tag rule (min=8) AND the cross-field rule — both must appear.
	parseRes := validate.Check(signupCF{Password: "short", Confirm: "different"}, parseMatch)
	if parseRes.Valid() {
		parseT.Fatal("expected failures")
	}
	parseFields := parseRes.Fields()
	if parseFields["password"] == "" {
		parseT.Fatalf("expected the tag rule (min) failure on password, got %v", parseFields)
	}
	if parseFields["confirm"] != "must match password" {
		parseT.Fatalf("expected the cross-field failure on confirm, got %v", parseFields)
	}
}

// TestCheckSameFieldTagAndCrossConflict pins the documented behavior when a tag rule and a
// cross-field rule target the SAME field: Fields() (first-per-field) shows the tag message, while
// All() surfaces both. This is the most surprising path, so it's nailed down explicitly.
func TestCheckSameFieldTagAndCrossConflict(parseT *testing.T) {
	// Password fails the min=8 tag AND a cross rule that also targets "password".
	parseRule := func(parseS signupCF) *validate.FieldError {
		return validate.Fail("password", "cross says no")
	}
	parseRes := validate.Check(signupCF{Password: "short", Confirm: "short"}, parseRule)

	if parseGot := parseRes.Fields()["password"]; parseGot == "cross says no" {
		parseT.Fatalf("Fields() must keep the FIRST (tag) error on a conflict, got the cross one: %q", parseGot)
	}
	parseAll := parseRes.All()["password"]
	if len(parseAll) < 2 {
		parseT.Fatalf("All() must surface both the tag and cross failures on password, got %v", parseAll)
	}
	parseHasCross := false
	for _, parseMsg := range parseAll {
		if parseMsg == "cross says no" {
			parseHasCross = true
		}
	}
	if !parseHasCross {
		parseT.Fatalf("All() must include the cross-field message, got %v", parseAll)
	}
}

// TestCheckMultipleCrossFailuresAllSurvive proves every failing cross rule is recorded (append, no
// dedup/overwrite), in call order after the tag errors.
func TestCheckMultipleCrossFailuresAllSurvive(parseT *testing.T) {
	parseRes := validate.Check(signupCF{Password: "longenough", Confirm: "longenough"},
		func(signupCF) *validate.FieldError { return validate.Fail("a", "first") },
		func(signupCF) *validate.FieldError { return validate.Fail("b", "second") },
	)
	parseFields := parseRes.Fields()
	if parseFields["a"] != "first" || parseFields["b"] != "second" {
		parseT.Fatalf("both cross-field failures must survive, got %v", parseFields)
	}
}

// TestCheckSkipsNilRuleAndContainsPanic proves a nil rule is ignored and a panicking cross-field
// rule is contained (recorded as a failure, never crashing Check).
func TestCheckSkipsNilRuleAndContainsPanic(parseT *testing.T) {
	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("Check must contain a panicking rule, got: %v", parseR)
		}
	}()

	parseBoom := func(parseS signupCF) *validate.FieldError {
		panic("rule bug")
	}
	parseRes := validate.Check(signupCF{Password: "longenough", Confirm: "longenough"}, nil, parseBoom)
	if parseRes.Valid() {
		parseT.Fatal("a panicking rule must record a failure")
	}
	parseFound := false
	for _, parseErr := range parseRes.Errors {
		if strings.Contains(parseErr.Message, "panicked") {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatalf("expected a contained-panic failure, got %v", parseRes.Errors)
	}
}

// TestRegisterRuleOnNonStringField proves a rule receives the field's actual Go value (not always
// a string), so numeric/bool rules work via a comma-ok assertion.
func TestRegisterRuleOnNonStringField(parseT *testing.T) {
	validate.RegisterRule("even", func(parseV any, parseArg string) (bool, string) {
		parseN, parseOK := parseV.(int)
		return parseOK && parseN%2 == 0, "must be even"
	})
	type counter struct {
		N int `validate:"even"`
	}
	if parseRes := validate.Struct(counter{N: 4}); !parseRes.Valid() {
		parseT.Fatalf("expected 4 to pass even, got %v", parseRes.Errors)
	}
	if validate.Struct(counter{N: 3}).Valid() {
		parseT.Fatal("expected 3 to fail even")
	}
}

// TestRegisterRuleReplaceSemantics proves re-registering a name swaps the rule (doc says
// "adds or replaces").
func TestRegisterRuleReplaceSemantics(parseT *testing.T) {
	validate.RegisterRule("swap", func(any, string) (bool, string) { return true, "" })
	validate.RegisterRule("swap", func(any, string) (bool, string) { return false, "now fails" })
	type s struct {
		X string `validate:"swap"`
	}
	parseRes := validate.Struct(s{X: "anything"})
	if parseRes.Valid() {
		parseT.Fatal("expected the replacement rule (always-fail) to take effect")
	}
}

// TestRegisterRulePanicIsContained proves a panicking custom rule fails the field cleanly rather
// than crashing Struct — a buggy rule must not 500 a server request or crash a client render.
func TestRegisterRulePanicIsContained(parseT *testing.T) {
	validate.RegisterRule("boom", func(parseV any, parseArg string) (bool, string) {
		_ = parseV.(int) // wrong-type assertion on a string field → panics
		return true, ""
	})
	type s struct {
		X string `validate:"boom"`
	}
	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("a panicking rule must be contained, not propagate: %v", parseR)
		}
	}()
	parseRes := validate.Struct(s{X: "str"})
	if parseRes.Valid() {
		parseT.Fatal("a panicking rule must fail its field")
	}
	parseFound := false
	for _, parseErr := range parseRes.Errors {
		if parseErr.Rule == "boom" && strings.Contains(parseErr.Message, "panicked") {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatalf("expected a contained-panic field error, got %v", parseRes.Errors)
	}
}

// TestRegisterRuleSkippedByOmitemptyOnZero proves omitempty short-circuits a custom rule on a zero
// value, exactly as it does for built-ins.
func TestRegisterRuleSkippedByOmitemptyOnZero(parseT *testing.T) {
	validate.RegisterRule("nope", func(any, string) (bool, string) { return false, "always fails" })
	type s struct {
		X string `validate:"omitempty,nope"`
	}
	if !validate.Struct(s{X: ""}).Valid() {
		parseT.Fatal("omitempty must skip the custom rule on a zero value")
	}
	if validate.Struct(s{X: "set"}).Valid() {
		parseT.Fatal("a non-zero value must still run the custom rule")
	}
}

// TestRegisterRuleIgnoresBlankNameAndNilFn proves the defensive guards in RegisterRule.
func TestRegisterRuleIgnoresBlankNameAndNilFn(parseT *testing.T) {
	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("RegisterRule panicked on degenerate input: %v", parseR)
		}
	}()
	validate.RegisterRule("", func(any, string) (bool, string) { return false, "x" })
	validate.RegisterRule("nilfn", nil)
	type t struct {
		X string `validate:"nilfn"`
	}
	if !validate.Struct(t{X: "x"}).Valid() {
		parseT.Fatal("a nil-fn registration must be a no-op, leaving the rule unrecognized")
	}
}

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

// TestOmitemptySkipsRulesWhenZero proves an `omitempty` field passes when empty but is
// still validated by its other rules when present.
func TestOmitemptySkipsRulesWhenZero(parseT *testing.T) {
	type form struct {
		Sort string `json:"sort" validate:"omitempty,oneof=asc desc"`
		Page int    `json:"page" validate:"omitempty,gte=1"`
	}

	if parseRes := validate.Struct(form{}); !parseRes.Valid() {
		parseT.Fatalf("expected empty optional fields to validate, got %v", parseRes.Errors)
	}
	if parseRes := validate.Struct(form{Sort: "desc", Page: 2}); !parseRes.Valid() {
		parseT.Fatalf("expected valid present values to pass, got %v", parseRes.Errors)
	}
	parseRes := validate.Struct(form{Sort: "sideways", Page: 0})
	if parseRes.Valid() {
		parseT.Fatal("expected an invalid present value to fail oneof")
	}
	if parseRes.Fields()["sort"] == "" {
		parseT.Fatalf("expected a sort error for a bad present value, got %#v", parseRes.Fields())
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
