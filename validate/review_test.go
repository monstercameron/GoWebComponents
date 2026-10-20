package validate_test

import (
	"math"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/validate"
)

// TestStringRulesApplyThroughPointers pins that a *string field is validated by
// the string rules; skipping them silently let any value through for optional
// (pointer) fields on the server.
func TestStringRulesApplyThroughPointers(parseT *testing.T) {
	type form struct {
		Email *string `json:"email" validate:"email"`
		Site  *string `json:"site" validate:"url"`
		Name  *string `json:"name" validate:"alpha"`
		Code  *string `json:"code" validate:"alphanum"`
		Zip   *string `json:"zip" validate:"numeric"`
		Plan  *string `json:"plan" validate:"oneof=free pro"`
	}
	parseBad := "not valid !"
	parseBadPlan := "enterprise"
	parseResult := validate.Struct(form{Email: &parseBad, Site: &parseBad, Name: &parseBad, Code: &parseBad, Zip: &parseBad, Plan: &parseBadPlan})
	for _, parseField := range []string{"email", "site", "name", "code", "zip", "plan"} {
		if _, parseFailed := parseResult.Fields()[parseField]; !parseFailed {
			parseT.Errorf("pointer field %q passed validation with %q", parseField, parseBad)
		}
	}

	parseGoodPlan := "pro"
	parseGoodEmail := "a@b.co"
	if parseOK := validate.Struct(form{Email: &parseGoodEmail, Plan: &parseGoodPlan}); !parseOK.Valid() {
		parseT.Errorf("valid pointer values rejected: %v", parseOK.Error())
	}
}

// TestNaNFailsRangeRules pins that NaN cannot satisfy min/max by comparing false
// both ways.
func TestNaNFailsRangeRules(parseT *testing.T) {
	type form struct {
		Score float64 `json:"score" validate:"min=0,max=100"`
	}
	if parseResult := validate.Struct(form{Score: math.NaN()}); parseResult.Valid() {
		parseT.Fatal("NaN passed min=0,max=100")
	}
}
