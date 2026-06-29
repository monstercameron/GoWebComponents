package tokens

import "testing"

func TestTokenNamesAreDefined(parseT *testing.T) {
	if len(ThemeTokenNames) == 0 {
		parseT.Fatal("expected theme token names")
	}
	if len(DensityTokenNames) == 0 {
		parseT.Fatal("expected density token names")
	}
	if len(MotionTokenNames) == 0 {
		parseT.Fatal("expected motion token names")
	}
}
