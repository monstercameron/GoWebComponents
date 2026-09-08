//go:build js && wasm

package router

import (
	"context"
	"errors"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

func TestRouteChunkGatesRouteFactoryUntilLoaded(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRelease := make(chan struct{})
	parseFactoryCalls := 0
	parseR := NewHashRouter()
	parseR.RegisterLazy("/reports", func(parseAttrs Attrs) *Element {
		parseFactoryCalls++
		return runtime.Div(nil, runtime.Text("reports ready"))
	}, RouteChunk{
		ID: "reports",
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) error {
			select {
			case <-parseCtx.Done():
				return parseCtx.Err()
			case <-parseRelease:
				return nil
			}
		},
		Loading: func(parseAttrs Attrs) *Element {
			if parseAttrs["chunk"] != true {
				parseT.Fatalf("expected chunk loading props, got %#v", parseAttrs)
			}
			return runtime.Div(nil, runtime.Text("loading route chunk"))
		},
	})
	js.Global().Get("location").Set("hash", "/reports")

	parseInitial := parseR.Current()
	if parseGot := collectElementText(parseInitial); parseGot != "loading route chunk" {
		parseT.Fatalf("initial route text = %q", parseGot)
	}
	if parseFactoryCalls != 0 {
		parseT.Fatalf("factory called before chunk loaded: %d", parseFactoryCalls)
	}
	close(parseRelease)
	waitForCondition(parseT, func() bool {
		return collectElementText(parseR.Current()) == "reports ready"
	})
	if parseFactoryCalls == 0 {
		parseT.Fatal("factory was not called after chunk loaded")
	}
}

func TestRouteChunkErrorUsesChunkErrorRenderer(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	parseR.RegisterLazy("/broken", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("unreachable"))
	}, RouteChunk{
		ID:     "broken",
		Loader: func(context.Context, RouteContext) error { return errors.New("chunk boom") },
		Error: func(parseAttrs Attrs) *Element {
			return runtime.Div(nil, runtime.Text("chunk error:"+parseAttrs["error"].(string)))
		},
	})
	js.Global().Get("location").Set("hash", "/broken")
	_ = parseR.Current()

	waitForCondition(parseT, func() bool {
		return collectElementText(parseR.Current()) == "chunk error:chunk boom"
	})
}
