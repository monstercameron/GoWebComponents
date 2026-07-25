package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
)

func TestAtlasPrependCSRFTokenRendersHiddenFormControl(parseT *testing.T) {
	parseMarkup := renderAtlasMarkupForTest(parseT, html.Form(html.Props{Action: "/api/app/preferences", Method: "post"},
		prependCSRFToken("atlas-test-token", html.Input(html.Props{Name: "theme", Value: "dark"}))...,
	))

	for _, parseNeedle := range []string{
		`<form`,
		`action="/api/app/preferences"`,
		`name="csrf_token"`,
		`value="atlas-test-token"`,
		`name="theme"`,
	} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected csrf form markup to include %q, got %q", parseNeedle, parseMarkup)
		}
	}
}
