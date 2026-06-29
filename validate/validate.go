// Package validate is a dependency-free, struct-tag validator for GoWebComponents.
//
// It is built for the framework's one-language advantage: the SAME Go struct, with
// the SAME `validate:"..."` tags, validates identically on the server (in an HTTP
// handler) and on the client (in ui.UseForm), so client and server validation can
// never drift. It uses only reflect/regexp/strconv/strings/fmt — no syscall, net,
// or filesystem — so it compiles to both `GOOS=js GOARCH=wasm` and native.
//
//	type Signup struct {
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age"   validate:"gte=13,lte=120"`
//	    Plan  string `json:"plan"  validate:"oneof=free pro team"`
//	}
//
//	res := validate.Struct(signup)
//	if !res.Valid() { /* res.Fields() is map[string]string, ready for ui.FieldErrors */ }
package validate

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// FieldError is one validation failure: the (possibly dotted) field name, the rule
// that failed, and a human-readable message.
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Result is the outcome of a validation pass.
type Result struct {
	Errors []FieldError `json:"errors,omitempty"`
}

// Valid reports whether the value passed every rule.
func (parseR Result) Valid() bool {
	return len(parseR.Errors) == 0
}

// Fields returns a field -> first-error-message map. Its type is map[string]string,
// identical to ui.FieldErrors, so it drops straight into a form:
//
//	form.Validate(func(v Signup) ui.FieldErrors { return ui.FieldErrors(validate.Struct(v).Fields()) })
func (parseR Result) Fields() map[string]string {
	parseOut := make(map[string]string, len(parseR.Errors))
	for _, parseErr := range parseR.Errors {
		if _, parseSeen := parseOut[parseErr.Field]; !parseSeen {
			parseOut[parseErr.Field] = parseErr.Message
		}
	}
	return parseOut
}

// All returns a field -> all-messages map (every failed rule per field).
func (parseR Result) All() map[string][]string {
	parseOut := make(map[string][]string, len(parseR.Errors))
	for _, parseErr := range parseR.Errors {
		parseOut[parseErr.Field] = append(parseOut[parseErr.Field], parseErr.Message)
	}
	return parseOut
}

// Error implements the error interface so a Result can be returned as an error from
// a server handler. It renders "field: message; ..." or "" when valid.
func (parseR Result) Error() string {
	if parseR.Valid() {
		return ""
	}
	parseParts := make([]string, 0, len(parseR.Errors))
	for _, parseErr := range parseR.Errors {
		parseParts = append(parseParts, parseErr.Field+": "+parseErr.Message)
	}
	return strings.Join(parseParts, "; ")
}

const maxValidateDepth = 16

var (
	reEmail    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	reURL      = regexp.MustCompile(`^https?://[^\s]+$`)
	reAlpha    = regexp.MustCompile(`^[A-Za-z]+$`)
	reAlphanum = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	reNumeric  = regexp.MustCompile(`^[0-9]+$`)
)

// Struct validates v — a struct or a (possibly nil) pointer to a struct — against
// the `validate` tags on its exported fields, recursing into nested struct fields
// with dotted field paths (e.g. "address.zip"). A nil pointer or non-struct value
// yields a valid (empty) result.
func Struct(parseValue any) Result {
	parseResult := Result{}
	parseRV := reflect.ValueOf(parseValue)
	validateStruct(parseRV, "", 0, &parseResult)
	return parseResult
}

// CrossRule is a programmatic, struct-level validation rule for constraints that span more than
// one field — password==confirm, end>=start, "required if plan==team" — which per-field tags
// cannot express. Return nil when valid, or a *FieldError (use Fail) describing the failure.
type CrossRule[T any] func(parseValue T) *FieldError

// Check validates value's struct tags (exactly like Struct) AND the given cross-field rules,
// merging every failure into one Result. It keeps the single-schema guarantee for relationships
// between fields: the same Check call runs in ui.UseForm on the client and in the server handler.
//
//	res := validate.Check(signup,
//	    func(s Signup) *validate.FieldError {
//	        if s.Password != s.Confirm {
//	            return validate.Fail("confirm", "must match password")
//	        }
//	        return nil
//	    },
//	)
//
// A nil rule is skipped. A rule that panics is contained (it records a failure rather than crashing
// the caller), mirroring custom tag rules.
func Check[T any](parseValue T, parseRules ...CrossRule[T]) Result {
	parseResult := Struct(parseValue)
	for _, parseRule := range parseRules {
		if parseRule == nil {
			continue
		}
		if parseErr := callCrossRule(parseRule, parseValue); parseErr != nil {
			parseResult.Errors = append(parseResult.Errors, *parseErr)
		}
	}
	return parseResult
}

// Fail builds a *FieldError for a cross-field rule, attaching the message to field.
func Fail(parseField, parseMessage string) *FieldError {
	return &FieldError{Field: parseField, Rule: "cross_field", Message: parseMessage}
}

// callCrossRule invokes a cross-field rule, converting a panic into a failure so a buggy rule does
// not crash Check (which on the server would 500 a request).
func callCrossRule[T any](parseRule CrossRule[T], parseValue T) (parseErr *FieldError) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = &FieldError{Field: "", Rule: "cross_field", Message: fmt.Sprintf("validation rule panicked: %v", parseRecovered)}
		}
	}()
	return parseRule(parseValue)
}

// validateStruct walks one struct value, applying field rules and recursing.
func validateStruct(parseRV reflect.Value, parsePrefix string, parseDepth int, parseResult *Result) {
	if parseDepth > maxValidateDepth {
		return
	}
	for parseRV.Kind() == reflect.Pointer {
		if parseRV.IsNil() {
			return
		}
		parseRV = parseRV.Elem()
	}
	if parseRV.Kind() != reflect.Struct {
		return
	}
	parseType := parseRV.Type()
	for parseI := 0; parseI < parseType.NumField(); parseI++ {
		parseField := parseType.Field(parseI)
		if !parseField.IsExported() {
			continue
		}
		parseName := parsePrefix + fieldName(parseField)
		parseFieldValue := parseRV.Field(parseI)

		// Recurse into nested struct (or pointer-to-struct) fields.
		parseElem := parseFieldValue
		for parseElem.Kind() == reflect.Pointer && !parseElem.IsNil() {
			parseElem = parseElem.Elem()
		}
		if parseElem.Kind() == reflect.Struct && parseElem.Type() != reflect.TypeFor[reflect.Value]() {
			validateStruct(parseFieldValue, parseName+".", parseDepth+1, parseResult)
		}

		parseTag := strings.TrimSpace(parseField.Tag.Get("validate"))
		if parseTag == "" || parseTag == "-" {
			continue
		}
		// `omitempty` makes a field optional: when it is present and the value is the
		// zero value, every other rule on the field is skipped (matching the common
		// struct-tag validator convention).
		parseRules := splitRules(parseTag)
		if slices.Contains(parseRules, "omitempty") && isZero(parseFieldValue) {
			continue
		}
		for _, parseRule := range parseRules {
			parseRuleName, parseArg := splitRule(parseRule)
			if parseRuleName == "omitempty" {
				continue
			}
			if parseErr, parseHasErr := applyRule(parseName, parseRuleName, parseArg, parseFieldValue); parseHasErr {
				parseResult.Errors = append(parseResult.Errors, parseErr)
			}
		}
	}
}

// fieldName returns the field's wire name: the json tag name when present (minus
// options like ",omitempty"), otherwise the Go field name.
func fieldName(parseField reflect.StructField) string {
	parseJSON := parseField.Tag.Get("json")
	if parseJSON == "" {
		return parseField.Name
	}
	parseName := strings.Split(parseJSON, ",")[0]
	if parseName == "" || parseName == "-" {
		return parseField.Name
	}
	return parseName
}

// splitRules splits a validate tag into its trimmed, non-empty rule tokens.
func splitRules(parseTag string) []string {
	var parseRules []string
	for parseRule := range strings.SplitSeq(parseTag, ",") {
		if parseRule = strings.TrimSpace(parseRule); parseRule != "" {
			parseRules = append(parseRules, parseRule)
		}
	}
	return parseRules
}

// splitRule splits "name=arg" into its name and argument ("" when there is no arg).
func splitRule(parseRule string) (string, string) {
	if parseName, parseArg, parseFound := strings.Cut(parseRule, "="); parseFound {
		return strings.TrimSpace(parseName), strings.TrimSpace(parseArg)
	}
	return parseRule, ""
}

// RuleFunc is a custom validation rule registered with RegisterRule. value is the field's value
// (type-assert it, commonly to string — use the comma-ok form, e.g. s, _ := value.(string)); arg
// is the tag argument after '=' (e.g. "5" in validate:"phone=5"), or "" when there is none. Return
// ok=false with a human-readable message to fail the field. A panic inside a RuleFunc is contained:
// it fails the field with a "validation rule panicked" message rather than crashing Struct.
type RuleFunc func(value any, arg string) (ok bool, message string)

var (
	customRulesMu sync.RWMutex
	customRules   = map[string]RuleFunc{}
)

// builtinRules are the rule names handled directly by applyRule's switch; custom rules cannot
// shadow them (RegisterRule ignores these names), and they are never dispatched to the registry.
// Keep in sync with the applyRule switch + the omitempty short-circuit.
var builtinRules = map[string]struct{}{
	"omitempty": {}, "required": {}, "min": {}, "max": {}, "len": {},
	"gt": {}, "gte": {}, "lt": {}, "lte": {}, "eq": {}, "ne": {},
	"oneof": {}, "email": {}, "url": {}, "alpha": {}, "alphanum": {}, "numeric": {},
}

// RegisterRule adds (or replaces) a custom validation rule usable as validate:"name" or
// validate:"name=arg". It is safe for concurrent use. A blank name, a nil fn, or a name that
// collides with a built-in rule is ignored so the core rule set can never be shadowed.
//
//	validate.RegisterRule("phone", func(v any, _ string) (bool, string) {
//	    s, _ := v.(string)
//	    return s == "" || rePhone.MatchString(s), "must be a valid phone number"
//	})
func RegisterRule(parseName string, parseFn RuleFunc) {
	parseName = strings.TrimSpace(parseName)
	if parseName == "" || parseFn == nil {
		return
	}
	if _, parseIsBuiltin := builtinRules[parseName]; parseIsBuiltin {
		return
	}
	customRulesMu.Lock()
	defer customRulesMu.Unlock()
	customRules[parseName] = parseFn
}

// lookupRule returns the registered custom rule for name, if any.
func lookupRule(parseName string) (RuleFunc, bool) {
	customRulesMu.RLock()
	defer customRulesMu.RUnlock()
	parseFn, parseOK := customRules[parseName]
	return parseFn, parseOK
}

// applyRule applies one rule to a field value and returns a FieldError when it
// fails. Built-in rules are handled by the switch; an unrecognized name is dispatched to a
// custom rule registered via RegisterRule, and is otherwise ignored (forward compatible).
func applyRule(parseField, parseRule, parseArg string, parseValue reflect.Value) (FieldError, bool) {
	parseFail := func(parseMessage string) (FieldError, bool) {
		return FieldError{Field: parseField, Rule: parseRule, Message: parseMessage}, true
	}
	switch parseRule {
	case "required":
		if isZero(parseValue) {
			return parseFail("is required")
		}
	case "min":
		if parseN, parseOK := parseFloat(parseArg); parseOK && lessThanMeasure(parseValue, parseN) {
			return parseFail(measureMessage(parseValue, "at least", parseArg))
		}
	case "max":
		if parseN, parseOK := parseFloat(parseArg); parseOK && greaterThanMeasure(parseValue, parseN) {
			return parseFail(measureMessage(parseValue, "at most", parseArg))
		}
	case "len":
		if parseN, parseOK := parseInt(parseArg); parseOK && measureLen(parseValue) != parseN {
			return parseFail(fmt.Sprintf("must be exactly %s long", parseArg))
		}
	case "gt", "gte", "lt", "lte", "eq", "ne":
		return applyNumericCompare(parseRule, parseArg, parseValue, parseFail)
	case "oneof":
		parseOptions := strings.Fields(parseArg)
		if !stringInSet(fmt.Sprintf("%v", valueInterface(parseValue)), parseOptions) {
			return parseFail("must be one of: " + strings.Join(parseOptions, ", "))
		}
	case "email":
		if parseStr, parseOK := stringValue(parseValue); parseOK && parseStr != "" && !reEmail.MatchString(parseStr) {
			return parseFail("must be a valid email address")
		}
	case "url":
		if parseStr, parseOK := stringValue(parseValue); parseOK && parseStr != "" && !reURL.MatchString(parseStr) {
			return parseFail("must be a valid URL")
		}
	case "alpha":
		if parseStr, parseOK := stringValue(parseValue); parseOK && parseStr != "" && !reAlpha.MatchString(parseStr) {
			return parseFail("must contain only letters")
		}
	case "alphanum":
		if parseStr, parseOK := stringValue(parseValue); parseOK && parseStr != "" && !reAlphanum.MatchString(parseStr) {
			return parseFail("must contain only letters and numbers")
		}
	case "numeric":
		if parseStr, parseOK := stringValue(parseValue); parseOK && parseStr != "" && !reNumeric.MatchString(parseStr) {
			return parseFail("must contain only digits")
		}
	default:
		// Not a built-in rule: dispatch to a custom rule if one is registered, else ignore it
		// (forward compatible). Built-in names never reach here, so a custom rule can't shadow them.
		if parseFn, parseOK := lookupRule(parseRule); parseOK {
			if parsePass, parseMessage := callRuleFunc(parseFn, valueInterface(parseValue), parseArg); !parsePass {
				return parseFail(parseMessage)
			}
		}
	}
	return FieldError{}, false
}

// callRuleFunc invokes a custom rule, converting a panic into a validation failure so a buggy rule
// (e.g. a non-comma-ok type assertion against an unexpected field type) fails the field cleanly
// instead of crashing the whole Struct walk (which on the server would 500 a request).
func callRuleFunc(parseFn RuleFunc, parseValue any, parseArg string) (parsePass bool, parseMessage string) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parsePass = false
			parseMessage = fmt.Sprintf("validation rule panicked: %v", parseRecovered)
		}
	}()
	return parseFn(parseValue, parseArg)
}

// applyNumericCompare handles gt/gte/lt/lte/eq/ne against the field's numeric value.
func applyNumericCompare(parseRule, parseArg string, parseValue reflect.Value, parseFail func(string) (FieldError, bool)) (FieldError, bool) {
	parseN, parseOK := parseFloat(parseArg)
	if !parseOK {
		return FieldError{}, false
	}
	parseV, parseIsNum := numericValue(parseValue)
	if !parseIsNum {
		return FieldError{}, false
	}
	switch parseRule {
	case "gt":
		if !(parseV > parseN) {
			return parseFail("must be greater than " + parseArg)
		}
	case "gte":
		if !(parseV >= parseN) {
			return parseFail("must be at least " + parseArg)
		}
	case "lt":
		if !(parseV < parseN) {
			return parseFail("must be less than " + parseArg)
		}
	case "lte":
		if !(parseV <= parseN) {
			return parseFail("must be at most " + parseArg)
		}
	case "eq":
		if parseV != parseN {
			return parseFail("must equal " + parseArg)
		}
	case "ne":
		if parseV == parseN {
			return parseFail("must not equal " + parseArg)
		}
	}
	return FieldError{}, false
}

// --- reflection helpers (all pure, wasm-safe) ---

func isZero(parseValue reflect.Value) bool {
	switch parseValue.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return parseValue.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return parseValue.IsNil()
	case reflect.Bool:
		return !parseValue.Bool()
	default:
		return parseValue.IsZero()
	}
}

// measureLen returns the length used by min/max/len for the value: rune count for
// strings, element count for slices/maps/arrays, else -1.
func measureLen(parseValue reflect.Value) int {
	switch parseValue.Kind() {
	case reflect.String:
		return len([]rune(parseValue.String()))
	case reflect.Slice, reflect.Map, reflect.Array:
		return parseValue.Len()
	default:
		return -1
	}
}

// lessThanMeasure reports whether the value is below n: by length for
// string/slice/map, by numeric value otherwise.
func lessThanMeasure(parseValue reflect.Value, parseN float64) bool {
	if parseLen := measureLen(parseValue); parseLen >= 0 {
		return float64(parseLen) < parseN
	}
	if parseV, parseOK := numericValue(parseValue); parseOK {
		return parseV < parseN
	}
	return false
}

func greaterThanMeasure(parseValue reflect.Value, parseN float64) bool {
	if parseLen := measureLen(parseValue); parseLen >= 0 {
		return float64(parseLen) > parseN
	}
	if parseV, parseOK := numericValue(parseValue); parseOK {
		return parseV > parseN
	}
	return false
}

// measureMessage renders a min/max message appropriate to the value's kind.
func measureMessage(parseValue reflect.Value, parseBound, parseArg string) string {
	if measureLen(parseValue) >= 0 {
		parseUnit := "characters"
		if parseValue.Kind() != reflect.String {
			parseUnit = "items"
		}
		return fmt.Sprintf("must be %s %s %s", parseBound, parseArg, parseUnit)
	}
	return fmt.Sprintf("must be %s %s", parseBound, parseArg)
}

func numericValue(parseValue reflect.Value) (float64, bool) {
	switch parseValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(parseValue.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(parseValue.Uint()), true
	case reflect.Float32, reflect.Float64:
		return parseValue.Float(), true
	default:
		return 0, false
	}
}

func stringValue(parseValue reflect.Value) (string, bool) {
	if parseValue.Kind() == reflect.String {
		return parseValue.String(), true
	}
	return "", false
}

func valueInterface(parseValue reflect.Value) any {
	if !parseValue.IsValid() || !parseValue.CanInterface() {
		return nil
	}
	return parseValue.Interface()
}

func stringInSet(parseValue string, parseSet []string) bool {
	return slices.Contains(parseSet, parseValue)
}

func parseFloat(parseArg string) (float64, bool) {
	parseN, parseErr := strconv.ParseFloat(parseArg, 64)
	return parseN, parseErr == nil
}

func parseInt(parseArg string) (int, bool) {
	parseN, parseErr := strconv.Atoi(parseArg)
	return parseN, parseErr == nil
}
