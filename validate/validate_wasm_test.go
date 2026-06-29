//go:build js && wasm

package validate_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/validate"
)

// TestValidatorRunsInWasm proves the reflection-based validator executes under
// GOOS=js GOARCH=wasm — the whole point of the package is that the SAME validation
// runs client-side (in the browser) and server-side from one struct definition.
func TestValidatorRunsInWasm(parseT *testing.T) {
	type login struct {
		Email string `json:"email" validate:"required,email"`
		PIN   string `json:"pin" validate:"len=4,numeric"`
	}

	parseBad := validate.Struct(login{Email: "nope", PIN: "12"})
	if parseBad.Valid() {
		parseT.Fatal("expected client-side wasm validation to fail an invalid login")
	}
	parseFields := parseBad.Fields()
	if parseFields["email"] != "must be a valid email address" {
		parseT.Fatalf("wasm email rule: %v", parseFields)
	}
	if parseFields["pin"] != "must be exactly 4 long" {
		parseT.Fatalf("wasm len rule: %v", parseFields)
	}

	parseGood := validate.Struct(login{Email: "a@b.com", PIN: "1234"})
	if !parseGood.Valid() {
		parseT.Fatalf("expected valid login under wasm, got %v", parseGood.Errors)
	}
}
