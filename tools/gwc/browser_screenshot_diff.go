package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// screenshotDiffResult is the JSON payload returned by `gwc screenshot-diff`.
type screenshotDiffResult struct {
	Baseline     string  `json:"baseline"`
	Current      string  `json:"current"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	TotalPixels  int     `json:"totalPixels"`
	ChangedCount int     `json:"changedPixels"`
	ChangedPct   float64 `json:"changedPct"`
	Threshold    int     `json:"threshold"`
	FailOver     float64 `json:"failOverPct"`
	Exceeded     bool    `json:"exceeded"`
	DiffImage    string  `json:"diffImage,omitempty"`
}

// loadPNG decodes a PNG file into an image.Image.
func loadPNG(parsePath string) (image.Image, error) {
	parseFile, parseErr := os.Open(parsePath)
	if parseErr != nil {
		return nil, parseErr
	}
	defer func() { _ = parseFile.Close() }()
	parseImg, parseErr := png.Decode(parseFile)
	if parseErr != nil {
		return nil, fmt.Errorf("decode %s: %w", parsePath, parseErr)
	}
	return parseImg, nil
}

// channelDelta returns the max absolute per-channel difference (0-255) between
// two colors, comparing R/G/B/A after down-shifting from 16-bit to 8-bit.
func channelDelta(parseA color.Color, parseB color.Color) int {
	parseAR, parseAG, parseAB, parseAA := parseA.RGBA()
	parseBR, parseBG, parseBB, parseBA := parseB.RGBA()
	parseMax := 0
	for _, parsePair := range [][2]uint32{{parseAR, parseBR}, {parseAG, parseBG}, {parseAB, parseBB}, {parseAA, parseBA}} {
		parseD := int(parsePair[0]>>8) - int(parsePair[1]>>8)
		if parseD < 0 {
			parseD = -parseD
		}
		if parseD > parseMax {
			parseMax = parseD
		}
	}
	return parseMax
}

// diffScreenshots compares two images pixel-by-pixel with a per-channel
// tolerance and, when parseWriteDiff is set, returns a highlight image marking
// every changed pixel in magenta over a dimmed copy of the current image. It is
// the testable core of `gwc screenshot-diff`.
func diffScreenshots(parseBaseline image.Image, parseCurrent image.Image, parseThreshold int, parseWriteDiff bool) (parseChanged int, parseTotal int, parseDiff *image.RGBA, parseErr error) {
	parseB := parseBaseline.Bounds()
	parseC := parseCurrent.Bounds()
	if parseB.Dx() != parseC.Dx() || parseB.Dy() != parseC.Dy() {
		return 0, 0, nil, fmt.Errorf("dimension mismatch: baseline %dx%d vs current %dx%d", parseB.Dx(), parseB.Dy(), parseC.Dx(), parseC.Dy())
	}
	parseW, parseH := parseB.Dx(), parseB.Dy()
	parseTotal = parseW * parseH
	if parseWriteDiff {
		parseDiff = image.NewRGBA(image.Rect(0, 0, parseW, parseH))
	}
	for parseY := 0; parseY < parseH; parseY++ {
		for parseX := 0; parseX < parseW; parseX++ {
			parseBC := parseBaseline.At(parseB.Min.X+parseX, parseB.Min.Y+parseY)
			parseCC := parseCurrent.At(parseC.Min.X+parseX, parseC.Min.Y+parseY)
			parseIsChanged := channelDelta(parseBC, parseCC) > parseThreshold
			if parseIsChanged {
				parseChanged++
			}
			if parseWriteDiff {
				if parseIsChanged {
					parseDiff.Set(parseX, parseY, color.RGBA{R: 255, G: 0, B: 255, A: 255})
				} else {
					// Dimmed grayscale of the current pixel for context.
					parseR, parseG, parseBl, _ := parseCC.RGBA()
					parseGray := uint8(((parseR + parseG + parseBl) / 3) >> 8)
					parseGray = parseGray/3 + 40
					parseDiff.Set(parseX, parseY, color.RGBA{R: parseGray, G: parseGray, B: parseGray, A: 255})
				}
			}
		}
	}
	return parseChanged, parseTotal, parseDiff, nil
}

// writePNG encodes an image to a PNG file, creating parent dirs.
func writePNG(parsePath string, parseImg image.Image) error {
	if parseDir := filepath.Dir(parsePath); parseDir != "" {
		_ = os.MkdirAll(parseDir, 0o755)
	}
	parseFile, parseErr := os.Create(parsePath)
	if parseErr != nil {
		return parseErr
	}
	defer func() { _ = parseFile.Close() }()
	return png.Encode(parseFile, parseImg)
}

// runScreenshotDiffCommand compares two PNG screenshots and reports the pixel
// delta — the visual-regression counterpart to structural snapshot-diff. Pure
// stdlib image math, so it ships in the default build (no browser toolchain).
var runScreenshotDiffCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("screenshot-diff", flag.ContinueOnError)
	parseBaseline := parseFs.String("baseline", "", "baseline PNG path (required)")
	parseCurrent := parseFs.String("current", "", "current PNG path (required)")
	parseOut := parseFs.String("out", "", "optional path to write a highlighted diff PNG")
	parseThreshold := parseFs.Int("threshold", 0, "per-channel tolerance 0-255 before a pixel counts as changed")
	parseFailOver := parseFs.Float64("fail-over", 0, "mark the result exceeded (ok=false) when changed%% is over this value")
	parseFs.Bool("json", false, "emit the JSON envelope (always on for this command)")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseBaseline) == "" || strings.TrimSpace(*parseCurrent) == "" {
		return writeAgenticEnvelope("screenshot-diff", false, nil, nil, fmt.Errorf("-baseline and -current are both required"))
	}

	parseBaseImg, parseErr := loadPNG(*parseBaseline)
	if parseErr != nil {
		return writeAgenticEnvelope("screenshot-diff", false, nil, nil, parseErr)
	}
	parseCurImg, parseErr := loadPNG(*parseCurrent)
	if parseErr != nil {
		return writeAgenticEnvelope("screenshot-diff", false, nil, nil, parseErr)
	}

	parseWriteDiff := strings.TrimSpace(*parseOut) != ""
	parseChanged, parseTotal, parseDiffImg, parseErr := diffScreenshots(parseBaseImg, parseCurImg, *parseThreshold, parseWriteDiff)
	if parseErr != nil {
		return writeAgenticEnvelope("screenshot-diff", false, nil, nil, parseErr)
	}
	if parseWriteDiff && parseDiffImg != nil {
		if parseErr := writePNG(*parseOut, parseDiffImg); parseErr != nil {
			return writeAgenticEnvelope("screenshot-diff", false, nil, nil, fmt.Errorf("write diff image: %w", parseErr))
		}
	}

	parsePct := 0.0
	if parseTotal > 0 {
		parsePct = float64(parseChanged) / float64(parseTotal) * 100
	}
	parseExceeded := *parseFailOver > 0 && parsePct > *parseFailOver
	parseBounds := parseBaseImg.Bounds()
	parseResult := screenshotDiffResult{
		Baseline:     *parseBaseline,
		Current:      *parseCurrent,
		Width:        parseBounds.Dx(),
		Height:       parseBounds.Dy(),
		TotalPixels:  parseTotal,
		ChangedCount: parseChanged,
		ChangedPct:   parsePct,
		Threshold:    *parseThreshold,
		FailOver:     *parseFailOver,
		Exceeded:     parseExceeded,
	}
	if parseWriteDiff {
		parseResult.DiffImage = *parseOut
	}
	// ok=false when the diff exceeds the caller's fail-over budget, so an agent
	// can gate a step directly on the envelope.
	return writeAgenticEnvelope("screenshot-diff", !parseExceeded, parseResult, nil, nil)
}
