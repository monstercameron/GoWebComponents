//go:build js && wasm

package router

import (
	"context"
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// RouteChunkLoader loads one route-level code artifact before the route factory
// is invoked.
type RouteChunkLoader func(context.Context, RouteContext) error

// RouteChunk describes a lazy route artifact. Apps can provide Loader for
// custom wasm/module bootstrapping, or Scripts for the default browser loader.
type RouteChunk struct {
	ID      string
	Scripts []string
	Loader  RouteChunkLoader
	Loading interface{}
	Error   interface{}
}

func (parseChunk RouteChunk) empty() bool {
	return strings.TrimSpace(parseChunk.ID) == "" && len(parseChunk.Scripts) == 0 && parseChunk.Loader == nil
}

func (parseChunk RouteChunk) key(parseRouteID string) string {
	parseID := strings.TrimSpace(parseChunk.ID)
	if parseID == "" {
		parseID = strings.Join(parseChunk.Scripts, "|")
	}
	if parseID == "" {
		parseID = parseRouteID
	}
	return "chunk:" + parseID
}

func (parseChunk RouteChunk) loadingComponent(parseOption Options) interface{} {
	if parseChunk.Loading != nil {
		return parseChunk.Loading
	}
	return parseOption.Loading
}

func (parseChunk RouteChunk) errorComponent(parseOption Options) interface{} {
	if parseChunk.Error != nil {
		return parseChunk.Error
	}
	return parseOption.Error
}

// RegisterLazy registers a route that loads chunk before rendering component.
func (parseR *Router) RegisterLazy(parsePath string, parseComponent interface{}, parseChunk RouteChunk, parseOptions ...Options) {
	parseOption := Options{}
	if len(parseOptions) > 0 {
		parseOption = parseOptions[0]
	}
	parseOption.Chunk = parseChunk
	parseR.Register(parsePath, parseComponent, parseOption)
}

// RegisterLazyRoute registers a route on the global router with a route chunk.
func RegisterLazyRoute(parsePath string, parseComponent interface{}, parseChunk RouteChunk, parseOptions ...Options) {
	GetRouter().RegisterLazy(parsePath, parseComponent, parseChunk, parseOptions...)
}

func (parseR *Router) prepareChunkState(parseActiveKeys []string) {
	parseR.chunkState.mu.Lock()
	defer parseR.chunkState.mu.Unlock()
	if parseR.chunkState.entries == nil {
		parseR.chunkState.entries = make(map[string]*routeChunkEntry)
	}
	parseNextActive := make(map[string]struct{}, len(parseActiveKeys))
	for _, parseKey := range parseActiveKeys {
		parseNextActive[parseKey] = struct{}{}
	}
	for parseKey, parseEntry := range parseR.chunkState.entries {
		if _, parseKeep := parseNextActive[parseKey]; parseKeep {
			continue
		}
		if parseEntry != nil && parseEntry.cancel != nil {
			parseEntry.cancel()
		}
		delete(parseR.chunkState.entries, parseKey)
	}
	parseR.chunkState.active = parseNextActive
}

func (parseR *Router) cancelRouteChunksIfActive() {
	parseR.chunkState.mu.Lock()
	defer parseR.chunkState.mu.Unlock()
	for parseKey, parseEntry := range parseR.chunkState.entries {
		if parseEntry != nil && parseEntry.cancel != nil {
			parseEntry.cancel()
		}
		delete(parseR.chunkState.entries, parseKey)
	}
	parseR.chunkState.active = make(map[string]struct{})
}

func (parseR *Router) ensureRouteChunkResult(parseKey string, parseChunk RouteChunk, parseRouteCtx RouteContext) struct {
	pending bool
	err     error
} {
	parseR.chunkState.mu.Lock()
	if parseR.chunkState.entries == nil {
		parseR.chunkState.entries = make(map[string]*routeChunkEntry)
	}
	parseEntry := parseR.chunkState.entries[parseKey]
	if parseEntry != nil {
		parseState := struct {
			pending bool
			err     error
		}{pending: parseEntry.pending, err: parseEntry.err}
		parseR.chunkState.mu.Unlock()
		return parseState
	}
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseEntry = &routeChunkEntry{pending: true, cancel: parseCancel}
	parseEntry.version++
	parseVersion := parseEntry.version
	parseR.chunkState.entries[parseKey] = parseEntry
	parseR.chunkState.mu.Unlock()

	runtime.ReportProfilingEvent("router", "chunk", "start", parseRouteCtx.Path, 0, map[string]string{
		"key": parseKey,
	})
	go func() {
		parseStarted := time.Now()
		var parseErr error
		func() {
			defer func() {
				if parseRecovered := recover(); parseRecovered != nil {
					parseErr = fmt.Errorf("router: route chunk loader panicked: %v", parseRecovered)
				}
			}()
			parseErr = loadRouteChunk(parseCtx, parseChunk, parseRouteCtx)
		}()
		parseDurationNs := time.Since(parseStarted).Nanoseconds()

		parseR.chunkState.mu.Lock()
		parseCurrent := parseR.chunkState.entries[parseKey]
		if parseCtx.Err() != nil || parseCurrent == nil || parseCurrent != parseEntry || parseVersion != parseCurrent.version {
			parseR.chunkState.mu.Unlock()
			runtime.ReportProfilingEvent("router", "chunk", "cancelled", parseRouteCtx.Path, parseDurationNs, map[string]string{"key": parseKey})
			return
		}
		parseCurrent.pending = false
		parseCurrent.err = parseErr
		parseCurrent.cancel = nil
		parseR.chunkState.mu.Unlock()

		if parseErr != nil {
			runtime.ReportProfilingEvent("router", "chunk", "error", parseRouteCtx.Path, parseDurationNs, map[string]string{
				"key":   parseKey,
				"error": parseErr.Error(),
			})
		} else {
			runtime.ReportProfilingEvent("router", "chunk", "finish", parseRouteCtx.Path, parseDurationNs, map[string]string{"key": parseKey})
		}
		if parseR.disposed {
			return
		}
		parseR.renderCurrentRoute(false)
	}()

	return struct {
		pending bool
		err     error
	}{pending: true}
}

func loadRouteChunk(parseCtx context.Context, parseChunk RouteChunk, parseRouteCtx RouteContext) error {
	if parseChunk.Loader != nil {
		return parseChunk.Loader(parseCtx, parseRouteCtx)
	}
	if len(parseChunk.Scripts) == 0 {
		return nil
	}
	for _, parseScriptURL := range parseChunk.Scripts {
		if parseErr := loadRouteChunkScript(parseCtx, strings.TrimSpace(parseScriptURL)); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

func loadRouteChunkScript(parseCtx context.Context, parseScriptURL string) error {
	if parseScriptURL == "" {
		return fmt.Errorf("router: route chunk script URL is required")
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return fmt.Errorf("router: route chunk loader could not access document")
	}
	parseScript := parseDocument.Call("createElement", "script")
	parseDone := make(chan error, 1)
	var parseLoad js.Func
	var parseError js.Func
	parseLoad = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		parseDone <- nil
		return nil
	})
	parseError = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		parseDone <- fmt.Errorf("router: route chunk script failed: %s", parseScriptURL)
		return nil
	})
	defer parseLoad.Release()
	defer parseError.Release()
	parseScript.Set("src", parseScriptURL)
	parseScript.Set("async", true)
	parseScript.Call("addEventListener", "load", parseLoad)
	parseScript.Call("addEventListener", "error", parseError)
	parseDocument.Get("head").Call("appendChild", parseScript)
	select {
	case <-parseCtx.Done():
		// Detach the listeners BEFORE the deferred Release runs. Removing the
		// <script> does not reliably cancel an already-queued load/error event;
		// if one fires after Release, invoking a released js.Func panics.
		parseScript.Call("removeEventListener", "load", parseLoad)
		parseScript.Call("removeEventListener", "error", parseError)
		if parseScript.Get("remove").Truthy() {
			parseScript.Call("remove")
		}
		return parseCtx.Err()
	case parseErr := <-parseDone:
		return parseErr
	}
}
