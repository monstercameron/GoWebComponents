//go:build !js || !wasm

package interop

import (
	"testing"
)

// TestIntlNativeAvailableReturnsFalse asserts that IntlAvailable always returns
// false on native builds, since the browser Intl API is not present.
func TestIntlNativeAvailableReturnsFalse(parseT *testing.T) {
	if IntlAvailable() {
		parseT.Fatal("IntlAvailable: expected false on native build, got true")
	}
}

// TestIntlNativeFormatNumberReturnsUnavailable asserts that IntlFormatNumber
// returns a CodeUnavailable error on native builds. Callers should fall back to
// the Go i18n formatters on non-browser builds.
func TestIntlNativeFormatNumberReturnsUnavailable(parseT *testing.T) {
	_, parseErr := IntlFormatNumber("en-US", 1234.56)
	if !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("IntlFormatNumber: expected CodeUnavailable, got %v", parseErr)
	}
	parseInteropErr, parseOk := AsError(parseErr)
	if !parseOk || parseInteropErr.Code != CodeUnavailable {
		parseT.Fatalf("IntlFormatNumber: expected structured interop error, got %#v ok=%t", parseInteropErr, parseOk)
	}
	parseCode, parseCodeOk := CodeOf(parseErr)
	if !parseCodeOk || parseCode != CodeUnavailable {
		parseT.Fatalf("IntlFormatNumber: CodeOf expected unavailable, got %q ok=%t", parseCode, parseCodeOk)
	}
}

// TestIntlNativeFormatDateReturnsUnavailable asserts that IntlFormatDate returns
// a CodeUnavailable error on native builds. Callers should fall back to the Go
// i18n formatters on non-browser builds.
func TestIntlNativeFormatDateReturnsUnavailable(parseT *testing.T) {
	_, parseErr := IntlFormatDate("en-US", 1_700_000_000_000)
	if !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("IntlFormatDate: expected CodeUnavailable, got %v", parseErr)
	}
	parseInteropErr, parseOk := AsError(parseErr)
	if !parseOk || parseInteropErr.Code != CodeUnavailable {
		parseT.Fatalf("IntlFormatDate: expected structured interop error, got %#v ok=%t", parseInteropErr, parseOk)
	}
	parseCode, parseCodeOk := CodeOf(parseErr)
	if !parseCodeOk || parseCode != CodeUnavailable {
		parseT.Fatalf("IntlFormatDate: CodeOf expected unavailable, got %q ok=%t", parseCode, parseCodeOk)
	}
}
