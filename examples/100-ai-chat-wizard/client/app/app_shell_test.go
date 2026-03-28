//go:build js && wasm

package app

import "testing"

func TestShouldRenderLandingShellEarly(parseT *testing.T) {
	parseTests := []struct {
		name string
		view appViewState
		want bool
	}{
		{
			name: "landing route unauthenticated renders immediately",
			view: appViewState{
				CurrentPath:   authLandingRoute,
				Authenticated: false,
				AuthResolved:  false,
				GRPCReady:     false,
			},
			want: true,
		},
		{
			name: "pricing route unauthenticated renders immediately",
			view: appViewState{
				CurrentPath:   marketingPricingRoute,
				Authenticated: false,
			},
			want: true,
		},
		{
			name: "app route still waits for auth bootstrap",
			view: appViewState{
				CurrentPath:   chatRouteRoot,
				Authenticated: false,
				AuthResolved:  false,
				GRPCReady:     false,
			},
			want: false,
		},
		{
			name: "authenticated landing route does not use early landing shell",
			view: appViewState{
				CurrentPath:   authLandingRoute,
				Authenticated: true,
			},
			want: false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldRenderLandingShellEarly(parseTest.view); parseGot != parseTest.want {
				parseT2.Fatalf("shouldRenderLandingShellEarly(%+v) = %v, want %v", parseTest.view, parseGot, parseTest.want)
			}
		})
	}
}
