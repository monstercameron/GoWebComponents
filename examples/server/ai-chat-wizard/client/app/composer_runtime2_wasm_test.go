//go:build js && wasm

package app

import (
	"errors"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestParseRegisterRuntime2RegionsHandlesRegistrationFailure verifies runtime2 registration falls back to inline rendering instead of panicking.
func TestParseRegisterRuntime2RegionsHandlesRegistrationFailure(parseT *testing.T) {
	parseOriginalRegister := handleComposerRuntime2RegionRegister
	parseOriginalReady := isRuntime2RegionRegistrationReady
	storeRuntime2RegionRegistrationOnce = sync.Once{}
	isRuntime2RegionRegistrationReady = false
	handleComposerRuntime2RegionRegister = func(parseRendererID string, parseRender func(renderComposerCostRegionProps) ui.Node) error {
		return errors.New("register failed")
	}
	defer func() {
		handleComposerRuntime2RegionRegister = parseOriginalRegister
		storeRuntime2RegionRegistrationOnce = sync.Once{}
		isRuntime2RegionRegistrationReady = parseOriginalReady
	}()

	parseRegisterRuntime2Regions()

	if isRuntime2RegionRegistrationReady {
		parseT.Fatal("expected runtime2 registration failure to keep inline fallback enabled")
	}
	parseNode := renderComposerCostParallelRegion(composerProps{})
	if parseNode == nil {
		parseT.Fatal("expected inline composer cost fallback node")
	}
}

// TestParseRegisterRuntime2RegionsRunsOnceOnSuccess verifies renderer registration is attempted once and leaves runtime2 rendering enabled.
func TestParseRegisterRuntime2RegionsRunsOnceOnSuccess(parseT *testing.T) {
	parseOriginalRegister := handleComposerRuntime2RegionRegister
	parseOriginalReady := isRuntime2RegionRegistrationReady
	storeRuntime2RegionRegistrationOnce = sync.Once{}
	isRuntime2RegionRegistrationReady = false
	parseRegisterCallCount := 0
	handleComposerRuntime2RegionRegister = func(parseRendererID string, parseRender func(renderComposerCostRegionProps) ui.Node) error {
		parseRegisterCallCount++
		return nil
	}
	defer func() {
		handleComposerRuntime2RegionRegister = parseOriginalRegister
		storeRuntime2RegionRegistrationOnce = sync.Once{}
		isRuntime2RegionRegistrationReady = parseOriginalReady
	}()

	parseRegisterRuntime2Regions()
	parseRegisterRuntime2Regions()

	if !isRuntime2RegionRegistrationReady {
		parseT.Fatal("expected successful runtime2 registration to enable parallel-region rendering")
	}
	if parseRegisterCallCount != 1 {
		parseT.Fatalf("expected one runtime2 registration attempt, got %d", parseRegisterCallCount)
	}
}
