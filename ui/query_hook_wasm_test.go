//go:build js && wasm

package ui

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/query"
)

// collectMockNodeText concatenates the text content of a mock DOM subtree, so a test can
// assert what a user would actually see rendered.
func collectMockNodeText(parseNode *mockdom.MockDOMNode) string {
	if parseNode == nil {
		return ""
	}
	parseText := parseNode.TextContent
	for _, parseChild := range parseNode.Children {
		parseText += collectMockNodeText(parseChild)
	}
	return parseText
}

// TestUseQueryRendersCachedDataToDOM is an end-to-end proof that a component using
// UseQuery mounts through the real wasm render path and paints cached data into the DOM
// with no loading flash (the cache is pre-seeded fresh, e.g. from SSR), and without
// invoking the fetcher.
func TestUseQueryRendersCachedDataToDOM(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePreviousInitialized })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("greeting", "Ada")

	parseProbe := func() Node {
		parseRes := UseQuery(parseCache, "greeting", func() (string, error) {
			parseT.Fatal("fetcher must not run for a fresh cached key")
			return "", nil
		})
		return runtime.CreateElement("span", map[string]any{}, parseRes.Data)
	}

	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseRoot, parseOk := parseContainer.(*mockdom.MockDOMNode)
	if !parseOk {
		parseT.Fatalf("expected a *mockdom.MockDOMNode container, got %T", parseContainer)
	}
	if parseGot := collectMockNodeText(parseRoot); !strings.Contains(parseGot, "Ada") {
		parseT.Fatalf("expected rendered DOM to contain %q, got %q", "Ada", parseGot)
	}
}

// TestUseMutationUpdatesCacheAndRerendersDOM is an end-to-end proof of the optimistic
// mutation loop: a successful mutation commits and the bound UseQuery re-renders the new
// value into the DOM, and a failing mutation rolls the DOM back to the prior value.
func TestUseMutationUpdatesCacheAndRerendersDOM(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePreviousInitialized })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("count", 10)

	var parseMutate func(int, func() (int, error)) query.Result[int]
	parseProbe := func() Node {
		parseMutate = UseMutation[int](parseCache, "count")
		parseRes := UseQuery(parseCache, "count", func() (int, error) {
			parseT.Fatal("fetcher must not run for a fresh cached key")
			return 0, nil
		})
		return runtime.CreateElement("span", map[string]any{}, strconv.Itoa(parseRes.Data))
	}

	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseRoot, parseOk := parseContainer.(*mockdom.MockDOMNode)
	if !parseOk {
		parseT.Fatalf("expected a *mockdom.MockDOMNode container, got %T", parseContainer)
	}
	if parseGot := collectMockNodeText(parseRoot); !strings.Contains(parseGot, "10") {
		parseT.Fatalf("expected initial DOM to show 10, got %q", parseGot)
	}

	// Successful mutation: optimistic 12, server confirms 12 → DOM re-renders to 12.
	parseMutate(12, func() (int, error) { return 12, nil })
	parseScheduler.Flush()
	if parseGot := collectMockNodeText(parseRoot); !strings.Contains(parseGot, "12") {
		parseT.Fatalf("expected DOM to update to 12 after a successful mutation, got %q", parseGot)
	}

	// Failing mutation: optimistic 99 then rollback → DOM returns to the committed 12.
	parseMutate(99, func() (int, error) { return 0, errors.New("save failed") })
	parseScheduler.Flush()
	parseGot := collectMockNodeText(parseRoot)
	if !strings.Contains(parseGot, "12") || strings.Contains(parseGot, "99") {
		parseT.Fatalf("expected DOM to roll back to 12 (never showing 99), got %q", parseGot)
	}
}
