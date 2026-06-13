package main

import (
	"image"
	"image/color"
	"testing"
)

// solidImage returns a w×h image filled with one color.
func solidImage(parseW int, parseH int, parseC color.RGBA) *image.RGBA {
	parseImg := image.NewRGBA(image.Rect(0, 0, parseW, parseH))
	for parseY := 0; parseY < parseH; parseY++ {
		for parseX := 0; parseX < parseW; parseX++ {
			parseImg.Set(parseX, parseY, parseC)
		}
	}
	return parseImg
}

// TestDiffScreenshotsIdentical pins zero-diff for identical images.
func TestDiffScreenshotsIdentical(t *testing.T) {
	parseA := solidImage(20, 10, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	parseB := solidImage(20, 10, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	parseChanged, parseTotal, _, parseErr := diffScreenshots(parseA, parseB, 0, false)
	if parseErr != nil {
		t.Fatalf("diff: %v", parseErr)
	}
	if parseTotal != 200 {
		t.Fatalf("total = %d, want 200", parseTotal)
	}
	if parseChanged != 0 {
		t.Fatalf("changed = %d, want 0 for identical images", parseChanged)
	}
}

// TestDiffScreenshotsChangedRegion pins that exactly the changed pixels are
// counted and marked in the diff image.
func TestDiffScreenshotsChangedRegion(t *testing.T) {
	parseA := solidImage(20, 10, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	parseB := solidImage(20, 10, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	// Flip a 3×2 block to white.
	for parseY := 0; parseY < 2; parseY++ {
		for parseX := 0; parseX < 3; parseX++ {
			parseB.Set(parseX, parseY, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	parseChanged, _, parseDiff, parseErr := diffScreenshots(parseA, parseB, 0, true)
	if parseErr != nil {
		t.Fatalf("diff: %v", parseErr)
	}
	if parseChanged != 6 {
		t.Fatalf("changed = %d, want 6", parseChanged)
	}
	if parseDiff == nil {
		t.Fatal("expected a diff image")
	}
	// Changed pixels are magenta; unchanged are not.
	parseR, parseG, parseB2, _ := parseDiff.At(0, 0).RGBA()
	if !(parseR>>8 == 255 && parseG>>8 == 0 && parseB2>>8 == 255) {
		t.Fatalf("changed pixel not marked magenta: got %d,%d,%d", parseR>>8, parseG>>8, parseB2>>8)
	}
	parseR2, parseG2, parseB3, _ := parseDiff.At(10, 5).RGBA()
	if parseR2>>8 == 255 && parseG2>>8 == 0 && parseB3>>8 == 255 {
		t.Fatal("unchanged pixel wrongly marked magenta")
	}
}

// TestDiffScreenshotsThreshold pins that a small per-channel delta is ignored
// under tolerance.
func TestDiffScreenshotsThreshold(t *testing.T) {
	parseA := solidImage(8, 8, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	parseB := solidImage(8, 8, color.RGBA{R: 105, G: 100, B: 100, A: 255}) // +5 on R
	parseChanged, _, _, parseErr := diffScreenshots(parseA, parseB, 8, false)
	if parseErr != nil {
		t.Fatalf("diff: %v", parseErr)
	}
	if parseChanged != 0 {
		t.Fatalf("changed = %d, want 0 under threshold 8", parseChanged)
	}
	parseChanged2, _, _, _ := diffScreenshots(parseA, parseB, 2, false)
	if parseChanged2 != 64 {
		t.Fatalf("changed = %d, want 64 under threshold 2", parseChanged2)
	}
}

// TestDiffScreenshotsDimensionMismatch pins a clean error, not a crash.
func TestDiffScreenshotsDimensionMismatch(t *testing.T) {
	parseA := solidImage(10, 10, color.RGBA{A: 255})
	parseB := solidImage(12, 10, color.RGBA{A: 255})
	if _, _, _, parseErr := diffScreenshots(parseA, parseB, 0, false); parseErr == nil {
		t.Fatal("expected dimension-mismatch error")
	}
}
