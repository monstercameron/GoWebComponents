package tokens

import "testing"

func TestTokenNamesAreDefined(t *testing.T) {
	if len(ThemeTokenNames) == 0 {
		t.Fatal("expected theme token names")
	}
	if len(DensityTokenNames) == 0 {
		t.Fatal("expected density token names")
	}
	if len(MotionTokenNames) == 0 {
		t.Fatal("expected motion token names")
	}
}
