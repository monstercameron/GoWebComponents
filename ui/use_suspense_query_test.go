//go:build !js || !wasm

package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/query"
)

// TestUseSuspenseQueryReturnsCachedValueWithoutSuspending proves the resolved path: a fresh cached
// key resolves the suspension immediately, so the component reads the value unconditionally and the
// AsyncBoundary renders the content (not the fallback). The fetcher must not run.
func TestUseSuspenseQueryReturnsCachedValueWithoutSuspending(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("user", "Ada")

	parseComponent := func() Node {
		parseName := UseSuspenseQuery(parseCache, "user", func() (string, error) {
			parseT.Fatal("fetcher must not run for a fresh cached key")
			return "", nil
		})
		return Text(parseName)
	}

	parseMarkup, parseErr := RenderToString(AsyncBoundary(AsyncBoundaryProps{
		Fallback: Text("loading"),
		Content:  CreateElement(parseComponent),
	}))
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	if parseMarkup != "Ada" {
		parseT.Fatalf("expected resolved content %q, got %q", "Ada", parseMarkup)
	}
}

// TestUseSuspenseQuerySuspendsToFallbackWhileLoading proves the suspending path: with no cached
// data and an in-flight fetch, the render suspends and the enclosing AsyncBoundary shows its
// fallback — the whole point of the hook (no Status switch in the component).
func TestUseSuspenseQuerySuspendsToFallbackWhileLoading(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseRelease := make(chan struct{})
	defer close(parseRelease) // let the loader goroutine finish after the assertion

	parseComponent := func() Node {
		parseName := UseSuspenseQuery(parseCache, "user", func() (string, error) {
			<-parseRelease // block so the fetch is genuinely in flight at render time
			return "Ada", nil
		})
		return Text(parseName)
	}

	parseMarkup, parseErr := RenderToString(AsyncBoundary(AsyncBoundaryProps{
		Fallback: Text("loading"),
		Content:  CreateElement(parseComponent),
	}))
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	if parseMarkup != "loading" {
		parseT.Fatalf("expected suspension fallback %q, got %q", "loading", parseMarkup)
	}
}

// TestUseSuspenseQueryResumesWithDataAfterFetch proves the core behavioral contract: after a
// suspended fetch completes, a subsequent render reads the resolved value — the AsyncBoundary
// transitions from fallback to content. (Native UseRef is stateless, so each RenderToString builds
// a fresh box; the second render sees the now-settled cache and resolves immediately.)
func TestUseSuspenseQueryResumesWithDataAfterFetch(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseRelease := make(chan struct{})

	parseComponent := func() Node {
		return Text(UseSuspenseQuery(parseCache, "user", func() (string, error) {
			<-parseRelease
			return "Ada", nil
		}))
	}
	parseTree := func() Node {
		return AsyncBoundary(AsyncBoundaryProps{Fallback: Text("loading"), Content: CreateElement(parseComponent)})
	}

	// First render suspends → fallback, and starts the loader goroutine.
	if parseM1, parseErr := RenderToString(parseTree()); parseErr != nil || parseM1 != "loading" {
		parseT.Fatalf("first render = %q, %v; want loading", parseM1, parseErr)
	}

	// Let the fetch complete and land in the cache.
	close(parseRelease)
	parseLanded := false
	for range 500 {
		if parseV, parseOk := parseCache.Peek("user"); parseOk && parseV == "Ada" {
			parseLanded = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !parseLanded {
		parseT.Fatal("fetch never populated the cache")
	}

	// Second render now sees settled data → content, not fallback.
	if parseM2, parseErr := RenderToString(parseTree()); parseErr != nil || parseM2 != "Ada" {
		parseT.Fatalf("second render = %q, %v; want Ada", parseM2, parseErr)
	}
}

// TestUseSuspenseQuerySurfacesErrorToErrorBoundary proves the error path: a settled cache error is
// thrown to the nearest ErrorBoundary (via Await's panic) rather than re-fetched or swallowed. The
// fetcher must NOT re-run for a settled error.
func TestUseSuspenseQuerySurfacesErrorToErrorBoundary(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseBoom := errors.New("load failed")
	query.Fetch(parseCache, "user", func() (string, error) { return "", parseBoom }) // settle an error

	parseComponent := func() Node {
		return Text(UseSuspenseQuery(parseCache, "user", func() (string, error) {
			parseT.Fatal("fetcher must not re-run for a settled error")
			return "", nil
		}))
	}
	parseNode := CreateElement(ErrorBoundary, ErrorBoundaryProps{
		ErrorFallback: func(parseErr error, parseReset func()) Node {
			if parseErr == nil || !strings.Contains(parseErr.Error(), "load failed") {
				parseT.Fatalf("unexpected boundary error: %v", parseErr)
			}
			return Text("error: " + parseErr.Error())
		},
		Child: AsyncBoundary(AsyncBoundaryProps{Fallback: Text("loading"), Content: CreateElement(parseComponent)}),
	})

	parseMarkup, parseErr := RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "load failed") {
		parseT.Fatalf("expected error-boundary fallback, got %q", parseMarkup)
	}
}

// TestSuspenseDepsEqual proves the dep comparison is shallow (Object.is-style), length-sensitive,
// and panic-safe on non-comparable deps (treated as changed so the query re-loads conservatively).
func TestSuspenseDepsEqual(parseT *testing.T) {
	if !suspenseDepsEqual([]any{"k", 1}, []any{"k", 1}) {
		parseT.Fatal("identical deps must compare equal")
	}
	if suspenseDepsEqual([]any{"k", 1}, []any{"k", 2}) {
		parseT.Fatal("a changed dep must compare unequal")
	}
	if suspenseDepsEqual([]any{"k"}, []any{"k", 1}) {
		parseT.Fatal("different lengths must compare unequal")
	}
	if suspenseDepsEqual([]any{[]int{1}}, []any{[]int{1}}) {
		parseT.Fatal("a non-comparable dep must be treated as changed, not panic")
	}
}
