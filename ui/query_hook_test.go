//go:build !js || !wasm

package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/query"
)

// TestUseQueryRendersCachedDataToDOMNatively proves UseQuery's snapshot path renders cached
// data into a real DOM in the NATIVE lane too (not only wasm): a component using UseQuery
// against a fresh-seeded cache mounts through the reconciler and shows the cached value.
func TestUseQueryRendersCachedDataToDOMNatively(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("user/1", "Ada")

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")

	parseComponent := func() *runtime.Element {
		parseRes := UseQuery(parseCache, "user/1", func() (string, error) {
			parseT.Fatal("fetcher must not run for a fresh cached key")
			return "", nil
		})
		return runtime.CreateElement("span", map[string]any{}, parseRes.Data)
	}
	if parseErr := parseRuntime.RenderInto(parseRoot, runtime.CreateElement(parseComponent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}

	if parseText := collectQueryDOMText(parseAdapter, parseRoot); !strings.Contains(parseText, "Ada") {
		parseT.Fatalf("expected cached Ada rendered natively, got %q", parseText)
	}
}

func collectQueryDOMText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectQueryDOMText(parseAdapter, parseChild)
	}
	return parseText
}

// TestUseQueryReturnsCachedSnapshot proves the hook renders cached data on first call
// (the pure no-fetch read), so a server-seeded or already-fetched key paints immediately
// with no loading flash.
func TestUseQueryReturnsCachedSnapshot(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("user/42", "Ada")

	parseRes := UseQuery(parseCache, "user/42", func() (string, error) {
		parseT.Fatal("fetcher must not run for a fresh cached key on first read")
		return "", nil
	})
	if parseRes.Data != "Ada" || parseRes.Status != query.StatusSuccess {
		parseT.Fatalf("expected fresh cached Ada, got %+v", parseRes)
	}
}

// TestUseMutationOptimisticRollback proves the hook's mutate function applies optimism
// and rolls back to the prior cached value when the mutation fails.
func TestUseMutationOptimisticRollback(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("count", 10)

	parseMutate := UseMutation[int](parseCache, "count")
	parseBoom := errors.New("save failed")
	parseRes := parseMutate(11, func() (int, error) { return 0, parseBoom })

	if !errors.Is(parseRes.Err, parseBoom) {
		parseT.Fatalf("expected the mutation error to surface, got %+v", parseRes)
	}
	if parsePeek, _ := parseCache.Peek("count"); parsePeek != 10 {
		parseT.Fatalf("expected rollback to 10, cache holds %v", parsePeek)
	}
}
