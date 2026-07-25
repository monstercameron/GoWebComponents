//go:build js && wasm

package ui

import (
	"fmt"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

var (
	hydrationIslandMu       sync.Mutex
	hydrationIslandActive   int
	hydrationIslandQueue    []func()
	hydrationIslandMaxQueue int
)

// ConfigureHydrationIslandBudget applies the runtime concurrency cap used by
// subsequent HydrateIsland calls. MaxConcurrent <= 0 disables the cap.
func ConfigureHydrationIslandBudget(parseBudget HydrationIslandBudget) {
	hydrationIslandMu.Lock()
	hydrationIslandMaxQueue = parseBudget.MaxConcurrent
	hydrationIslandMu.Unlock()
	drainHydrationIslandQueue()
}

// HydrateIsland schedules an independent SSR island to hydrate when its trigger
// fires. The returned cleanup releases browser listeners if hydration has not
// already started.
func HydrateIsland(parseRoot Node, parseOptions HydrationIslandOptions) (func(), error) {
	parseNormalized, parseErr := NormalizeHydrationIslandOptions(parseOptions)
	if parseErr != nil {
		return nil, parseErr
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() || !parseDocument.Get("querySelector").Truthy() {
		return nil, fmt.Errorf("ui: HydrateIsland could not access document.querySelector")
	}
	parseTarget := parseDocument.Call("querySelector", parseNormalized.Selector)
	if !parseTarget.Truthy() {
		return nil, fmt.Errorf("ui: HydrateIsland could not find target %q", parseNormalized.Selector)
	}

	parseCleanups := []func(){}
	parseStarted := false
	parseCleanup := func() {
		for _, parseRelease := range parseCleanups {
			parseRelease()
		}
		parseCleanups = nil
	}
	parseStart := func() {
		if parseStarted {
			return
		}
		parseStarted = true
		parseCleanup()
		enqueueHydrationIsland(func() {
			if _, parseHydrateErr := HydrateInto(parseRoot, parseTarget, parseNormalized.Hydration); parseHydrateErr != nil {
				runtime.ReportDiagnostic("ui", runtime.DiagnosticWarning, "hydration island failed: "+parseHydrateErr.Error())
			}
		})
	}

	if parseNormalized.Timeout > 0 {
		parseTimer := time.AfterFunc(parseNormalized.Timeout, func() {
			defer runtime.RecoverContainedPanic("ui", "HydrateIsland timeout")
			parseStart()
		})
		parseCleanups = append(parseCleanups, func() {
			parseTimer.Stop()
		})
	}

	switch parseNormalized.Strategy {
	case HydrateImmediately:
		parseStart()
	case HydrateOnIdle:
		parseCleanups = append(parseCleanups, scheduleIslandIdle(parseStart))
	case HydrateOnInteraction:
		for _, parseEventName := range parseNormalized.Events {
			parseCleanups = append(parseCleanups, scheduleIslandEvent(parseTarget, parseEventName, parseStart))
		}
	case HydrateOnVisible:
		parseCleanups = append(parseCleanups, scheduleIslandVisible(parseTarget, parseNormalized.RootMargin, parseStart))
	}

	return func() {
		if !parseStarted {
			parseCleanup()
		}
	}, nil
}

func enqueueHydrationIsland(parseRun func()) {
	if parseRun == nil {
		return
	}
	hydrationIslandMu.Lock()
	if hydrationIslandMaxQueue > 0 && hydrationIslandActive >= hydrationIslandMaxQueue {
		hydrationIslandQueue = append(hydrationIslandQueue, parseRun)
		hydrationIslandMu.Unlock()
		return
	}
	hydrationIslandActive++
	hydrationIslandMu.Unlock()
	go func() {
		defer runtime.RecoverContainedPanic("ui", "HydrateIsland queue")
		defer finishHydrationIslandRun()
		parseRun()
	}()
}

func finishHydrationIslandRun() {
	hydrationIslandMu.Lock()
	if hydrationIslandActive > 0 {
		hydrationIslandActive--
	}
	hydrationIslandMu.Unlock()
	drainHydrationIslandQueue()
}

func drainHydrationIslandQueue() {
	for {
		hydrationIslandMu.Lock()
		if len(hydrationIslandQueue) == 0 || (hydrationIslandMaxQueue > 0 && hydrationIslandActive >= hydrationIslandMaxQueue) {
			hydrationIslandMu.Unlock()
			return
		}
		parseRun := hydrationIslandQueue[0]
		hydrationIslandQueue = append([]func(){}, hydrationIslandQueue[1:]...)
		hydrationIslandActive++
		hydrationIslandMu.Unlock()
		go func() {
			defer runtime.RecoverContainedPanic("ui", "HydrateIsland queue")
			defer finishHydrationIslandRun()
			parseRun()
		}()
	}
}

func scheduleIslandEvent(parseTarget js.Value, parseEventName string, parseStart func()) func() {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "HydrateIsland interaction")
		parseStart()
		return nil
	})
	parseTarget.Call("addEventListener", parseEventName, parseHandler)
	return func() {
		if parseTarget.Truthy() && parseTarget.Get("removeEventListener").Truthy() {
			parseTarget.Call("removeEventListener", parseEventName, parseHandler)
		}
		parseHandler.Release()
	}
}

func scheduleIslandIdle(parseStart func()) func() {
	parseWindow := js.Global().Get("window")
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "HydrateIsland idle")
		parseStart()
		return nil
	})
	if parseWindow.Truthy() && parseWindow.Get("requestIdleCallback").Truthy() {
		parseID := parseWindow.Call("requestIdleCallback", parseHandler)
		return func() {
			if parseWindow.Get("cancelIdleCallback").Truthy() {
				parseWindow.Call("cancelIdleCallback", parseID)
			}
			parseHandler.Release()
		}
	}
	parseID := parseWindow.Call("setTimeout", parseHandler, 0)
	return func() {
		if parseWindow.Truthy() && parseWindow.Get("clearTimeout").Truthy() {
			parseWindow.Call("clearTimeout", parseID)
		}
		parseHandler.Release()
	}
}

func scheduleIslandVisible(parseTarget js.Value, parseRootMargin string, parseStart func()) func() {
	parseWindow := js.Global().Get("window")
	parseObserverCtor := parseWindow.Get("IntersectionObserver")
	if !parseWindow.Truthy() || !parseObserverCtor.Truthy() {
		return scheduleIslandIdle(parseStart)
	}
	var parseObserver js.Value
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "HydrateIsland visible")
		if len(parseArgs) == 0 || parseArgs[0].Length() == 0 {
			return nil
		}
		parseEntry := parseArgs[0].Index(0)
		if parseEntry.Get("isIntersecting").Truthy() {
			if parseObserver.Truthy() {
				parseObserver.Call("disconnect")
			}
			parseStart()
		}
		return nil
	})
	parseOptions := map[string]any{}
	if parseRootMargin != "" {
		parseOptions["rootMargin"] = parseRootMargin
	}
	parseObserver = parseObserverCtor.New(parseHandler, js.ValueOf(parseOptions))
	parseObserver.Call("observe", parseTarget)
	return func() {
		if parseObserver.Truthy() {
			parseObserver.Call("disconnect")
		}
		parseHandler.Release()
	}
}
