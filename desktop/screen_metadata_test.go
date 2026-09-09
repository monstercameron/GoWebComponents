package desktop

import (
	"math"
	"testing"
)

// TestScreenMetadata validates extended physical and logical monitor metadata.
func TestScreenMetadata(parseTest *testing.T) {
	parseValid := ScreenInfo{ID: "left", X: -1920, Width: 1920, Height: 1080, Scale: 1.5, Rotation: 90, WorkArea: ScreenBounds{X: -1920, Width: 1920, Height: 1040}}
	if parseErr := validateScreens([]ScreenInfo{parseValid}); parseErr != nil { parseTest.Fatal(parseErr) }
	for _, parseRotation := range []float32{-1, 360, float32(math.NaN()), float32(math.Inf(1))} {
		parseInvalid := parseValid
		parseInvalid.Rotation = parseRotation
		if validateScreens([]ScreenInfo{parseInvalid}) == nil { parseTest.Errorf("accepted rotation %v", parseRotation) }
	}
	parseValid.PhysicalBounds.Width = -1
	if validateScreens([]ScreenInfo{parseValid}) == nil { parseTest.Fatal("accepted negative physical width") }
}
