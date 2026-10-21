//go:build !js || !wasm

package ui

import (
	"testing"
	"time"
)

// TestFormResetInvalidatesInFlightAsyncValidation proves a validation started before Reset can
// never write its result into the form once Reset has run and a newer validation has started:
// Reset must not reuse a sequence number an older in-flight validation still holds.
func TestFormResetInvalidatesInFlightAsyncValidation(parseT *testing.T) {
	parseForm := UseForm(sampleFormModel{Name: "Atlas"})

	parseRelease := make(chan struct{})
	parseStaleDone := make(chan bool, 1)
	parseForm.ValidateAsync(func(parseValue sampleFormModel) (FieldErrors, string) {
		<-parseRelease
		return FieldErrors{"Name": "stale"}, "stale"
	}, func(isOk bool) { parseStaleDone <- isOk })

	parseForm.Reset()

	parseFreshDone := make(chan bool, 1)
	parseForm.ValidateAsync(func(parseValue sampleFormModel) (FieldErrors, string) {
		return nil, ""
	}, func(isOk bool) { parseFreshDone <- isOk })
	select {
	case <-parseFreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("fresh validation never completed")
	}

	close(parseRelease)
	select {
	case <-parseStaleDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("stale validation never completed")
	}

	if parseForm.HasErrors() || parseForm.FormError() != "" {
		parseT.Fatalf("stale pre-Reset validation overwrote the fresh result: errors=%#v form=%q", parseForm.Errors(), parseForm.FormError())
	}
}
