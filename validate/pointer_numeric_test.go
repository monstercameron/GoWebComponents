package validate_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/validate"
)

// TestNumericRulesApplyThroughPointers pins that min/max/gte/etc. actually fire on
// pointer-to-numeric fields (they previously silently no-op'd → false pass), while
// a nil pointer skips the rule (an unset optional field is `required`'s concern).
func TestNumericRulesApplyThroughPointers(parseT *testing.T) {
	type patch struct {
		Age *int `validate:"gte=18"`
	}
	parseBad := -5
	if validate.Struct(patch{Age: &parseBad}).Valid() {
		parseT.Fatal("expected *int(-5) with gte=18 to be INVALID")
	}
	parseGood := 21
	if !validate.Struct(patch{Age: &parseGood}).Valid() {
		parseT.Fatal("expected *int(21) with gte=18 to be valid")
	}
	// A nil pointer skips the numeric rule (not provided).
	if !validate.Struct(patch{Age: nil}).Valid() {
		parseT.Fatal("expected nil *int to skip gte (unset optional field)")
	}
}
