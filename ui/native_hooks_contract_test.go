//go:build !js || !wasm

package ui

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/anim"
)

func TestSafeGoRunsFunctionOnNativeBuild(parseT *testing.T) {
	parseDone := make(chan struct{}, 1)
	SafeGo("background", func() {
		parseDone <- struct{}{}
	})
	select {
	case <-parseDone:
	case <-time.After(time.Second):
		parseT.Fatal("SafeGo did not run function")
	}
}

func TestUseSpringNativeReturnsTargetImmediately(parseT *testing.T) {
	if parseGot := UseSpring(12.5, anim.GentleSpring()); parseGot != 12.5 {
		parseT.Fatalf("UseSpring native = %v, want target", parseGot)
	}
}

func TestStartSpringAnimationSchedulesClampsAndCancels(parseT *testing.T) {
	parseSpring := anim.NewSpring(anim.GentleSpring(), 0)
	parseSpring.SetTarget(10)
	var parsePositions []float64
	var parseFrames []func(float64)
	var parseCanceled []bool
	parseScheduler := func(parseFrame func(float64)) func() {
		parseFrames = append(parseFrames, parseFrame)
		parseCanceled = append(parseCanceled, false)
		parseIndex := len(parseCanceled) - 1
		return func() { parseCanceled[parseIndex] = true }
	}

	parseCleanup := startSpringAnimation(parseSpring, func(parsePosition float64) {
		parsePositions = append(parsePositions, parsePosition)
	}, parseScheduler)
	if len(parseFrames) != 1 {
		parseT.Fatalf("initial scheduled frames = %d, want 1", len(parseFrames))
	}

	parseFrames[0](1000)
	if len(parsePositions) != 1 || parsePositions[0] != 0 {
		parseT.Fatalf("first frame positions = %#v, want unchanged initial position", parsePositions)
	}
	if len(parseFrames) != 2 {
		parseT.Fatalf("frames after unsettled first step = %d, want 2", len(parseFrames))
	}

	parseExpected := anim.NewSpring(anim.GentleSpring(), 0)
	parseExpected.SetTarget(10)
	parseExpected.Step(0)
	parseExpectedPos := parseExpected.Step(maxSpringDt)
	parseFrames[1](2000)
	if len(parsePositions) != 2 || parsePositions[1] != parseExpectedPos {
		parseT.Fatalf("second frame position = %#v, want clamped step %v", parsePositions, parseExpectedPos)
	}

	parseCleanup()
	if len(parseCanceled) < 3 || !parseCanceled[len(parseCanceled)-1] {
		parseT.Fatalf("cleanup should cancel current frame, canceled=%#v", parseCanceled)
	}
	parseFrameCount := len(parseFrames)
	parsePositionCount := len(parsePositions)
	parseFrames[len(parseFrames)-1](2100)
	if len(parseFrames) != parseFrameCount || len(parsePositions) != parsePositionCount {
		parseT.Fatal("frame callback after cleanup should not publish or reschedule")
	}
}

func TestStartSpringAnimationNilInputsAreNoops(parseT *testing.T) {
	startSpringAnimation(nil, nil, nil)()
	parseSpring := anim.NewSpring(anim.GentleSpring(), 1)
	startSpringAnimation(parseSpring, nil, nil)()
}

func TestStartSpringAnimationSettledSpringDoesNotReschedule(parseT *testing.T) {
	parseSpring := anim.NewSpring(anim.GentleSpring(), 3)
	parseScheduled := 0
	parseCleanup := startSpringAnimation(parseSpring, nil, func(parseFrame func(float64)) func() {
		parseScheduled++
		parseFrame(16)
		return nil
	})
	if parseScheduled != 1 {
		parseT.Fatalf("settled spring scheduled %d frames, want 1", parseScheduled)
	}
	parseCleanup()
}

func TestNativeMediaAndPersistedStorageFallbacks(parseT *testing.T) {
	if currentMediaMatch("(prefers-color-scheme: dark)") {
		parseT.Fatal("native currentMediaMatch should default to false")
	}
	if parseGot := loadStoredInitial("missing", "fallback", PersistSession); parseGot != "fallback" {
		parseT.Fatalf("loadStoredInitial native = %q, want fallback", parseGot)
	}
	if _, parseErr := resolveStorage(PersistLocal); parseErr == nil {
		parseT.Fatal("native resolveStorage should report unavailable local storage")
	}
	if _, parseErr := resolveStorage(PersistSession); parseErr == nil {
		parseT.Fatal("native resolveStorage should report unavailable session storage")
	}
}
