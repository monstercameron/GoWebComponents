//go:build !js || !wasm

package runtime

import (
	"fmt"
	"testing"
	"time"
)

// TestSSRConcurrentHookFibersIsolateRequests forces two component bodies to
// overlap, so serial rendering or sharing the ambient client fiber cannot pass.
func TestSSRConcurrentHookFibersIsolateRequests(parseT *testing.T) {
	parseEntered := make(chan struct{}, 2)
	parseRelease := make(chan struct{})
	parseResults := make(chan error, 2)
	for parseIndex := range 2 {
		go func() {
			parseMarker := fmt.Sprintf("request-%d", parseIndex)
			parseHTML, parseErr := RenderToString(CreateElement(func() *Element {
				parseFiber := GetCurrentFiber()
				parseScope := ResolveRuntime()
				parseGet, _ := GoUseAtomGlobal("overlapping-request", parseMarker)
				parseEntered <- struct{}{}
				<-parseRelease
				if GetCurrentFiber() != parseFiber || ResolveRuntime() != parseScope || !parseScope.ssrRequestScope {
					panic("concurrent SSR replaced request hook context")
				}
				return CreateElement("span", nil, parseGet())
			}, nil))
			if parseErr == nil && parseHTML != "<span>"+parseMarker+"</span>" {
				parseErr = fmt.Errorf("unexpected request output %q", parseHTML)
			}
			if getSSRHookFiber() != nil && parseErr == nil {
				parseErr = fmt.Errorf("request fiber retained after return")
			}
			parseResults <- parseErr
		}()
	}
	for range 2 {
		select {
		case <-parseEntered:
		case <-time.After(2 * time.Second):
			close(parseRelease)
			parseT.Fatal("SSR components did not overlap")
		}
	}
	close(parseRelease)
	for range 2 {
		if parseErr := <-parseResults; parseErr != nil {
			parseT.Error(parseErr)
		}
	}
}

// TestSSRHookFiberRestoresNestedAndPanickingScopes verifies stack-like cleanup
// even when a nested component unwinds with a suspension or other panic.
func TestSSRHookFiberRestoresNestedAndPanickingScopes(parseT *testing.T) {
	parseOuter := &Fiber{}
	parseInner := &Fiber{}
	parseRestore := setSSRHookFiber(parseOuter, 0)
	func() {
		defer func() {
			if recover() != "expected" {
				parseT.Error("panic was swallowed or changed")
			}
		}()
		defer setSSRHookFiber(parseInner, 0)()
		if GetCurrentFiber() != parseInner {
			parseT.Error("nested fiber not installed")
		}
		panic("expected")
	}()
	if GetCurrentFiber() != parseOuter {
		parseT.Error("outer fiber not restored")
	}
	parseRestore()
	if getSSRHookFiber() != nil {
		parseT.Error("top-level fiber not removed")
	}
}

// TestSSRSlabLookupSkipsHookOwner verifies that host construction during SSR
// cannot consult the goroutine-indexed hook map just to reject client slabs.
func TestSSRSlabLookupSkipsHookOwner(parseT *testing.T) {
	parsePrevious := currentFiber
	parseClient := &Fiber{}
	currentFiber = parseClient
	defer func() { currentFiber = parsePrevious }()
	if activeSlabFiber() != parseClient {
		parseT.Fatal("client slab fiber missing before SSR")
	}
	parseRestore := setSSRHookFiber(&Fiber{}, 0)
	if activeSlabFiber() != nil {
		parseT.Error("SSR host construction attempted to use a client slab")
	}
	parseRestore()
	if activeSlabFiber() != parseClient {
		parseT.Error("client slab fiber not restored after SSR")
	}
}
