//go:build js && wasm
// +build js,wasm

package devtools

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// TestTreeSummaryWasmRendersSelectedNodeAndHierarchy verifies the tree inspector renders selected-node details, cache matches, and nested tree rows.
func TestTreeSummaryWasmRendersSelectedNodeAndHierarchy(parseT *testing.T) {
	parseSnapshot := Snapshot{
		Route: Route{
			Path: "/dashboard",
			Stack: []RouteStack{{
				Path: "/dashboard",
			}},
		},
		Cache: []CacheEntry{{
			Key:             "users:list",
			Ready:           true,
			Stale:           false,
			SubscriberCount: 2,
			OwnerPaths:      []string{"/app/child"},
		}},
		Tree: &Node{
			Name:        "App",
			Path:        "/app",
			Kind:        "component",
			HookCount:   1,
			EffectCount: 1,
			Signature:   "App()",
			Children: []Node{{
				Name:              "Child",
				Path:              "/app/child",
				Kind:              "component",
				HookCount:         2,
				EffectCount:       1,
				Signature:         "Child()",
				FineGrained:       true,
				UpdateOrigin:      "atom",
				ReactiveSource:    "theme",
				RenderDurationNs:  300_000,
				DiffDurationNs:    120_000,
				CommitDurationNs:  110_000,
				SelfDurationNs:    530_000,
				SubtreeDurationNs: 2_500_000,
				Hooks: []Hook{{
					Slot:         1,
					Kind:         "state",
					Value:        "dark",
					Dependencies: "[theme]",
					Status:       "clean",
				}},
				Children: []Node{{
					Name: "Leaf",
					Path: "/app/child/leaf",
					Kind: "host",
				}},
			}},
		},
	}

	parseInspectorMarkup, parseErr := ui.RenderToString(selectedNodeSummary(parseSnapshot, "/app/child"))
	if parseErr != nil {
		parseT.Fatalf("expected selected node summary render, got %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Child",
		"/app/child",
		"origin=atom | source=theme",
		"state#1=dark (clean)",
		"users:list ready=true stale=false subscribers=2",
	} {
		if !strings.Contains(parseInspectorMarkup, parseExpected) {
			parseT.Fatalf("expected selected node markup to include %q, got %q", parseExpected, parseInspectorMarkup)
		}
	}

	parseTreeMarkup, parseErr2 := ui.RenderToString(treeSummary(parseSnapshot.Tree, 0, 2, "/app/child", nil))
	if parseErr2 != nil {
		parseT.Fatalf("expected tree summary render, got %v", parseErr2)
	}
	for _, parseExpected := range []string{
		"App",
		"Child",
		"Leaf",
		"signature: Child()",
		"subtree=2.50ms self=0.53ms render=0.30ms diff=0.12ms commit=0.11ms",
		"mode=fine-grained | origin=atom | source=theme",
	} {
		if !strings.Contains(parseTreeMarkup, parseExpected) {
			parseT.Fatalf("expected tree markup to include %q, got %q", parseExpected, parseTreeMarkup)
		}
	}

	parseTruncatedMarkup, parseErr3 := ui.RenderToString(treeSummary(parseSnapshot.Tree, 0, 0, "", nil))
	if parseErr3 != nil {
		parseT.Fatalf("expected truncated tree render, got %v", parseErr3)
	}
	if !strings.Contains(parseTruncatedMarkup, "1 child nodes hidden at max depth") {
		parseT.Fatalf("expected truncated tree message, got %q", parseTruncatedMarkup)
	}
}

// TestDevtoolsTreeHelpersWasmCoverMappingsAndFormatting verifies the tree helper utilities sort, map, and format debug values consistently.
func TestDevtoolsTreeHelpersWasmCoverMappingsAndFormatting(parseT *testing.T) {
	parseNode := &Node{
		Name: "App",
		Path: "/app",
		Children: []Node{{
			Name: "Child",
			Path: "/app/child",
		}},
	}
	if parseFound := findNodeByPath(parseNode, "/app/child"); parseFound == nil || parseFound.Name != "Child" {
		parseT.Fatalf("expected child node lookup, got %#v", parseFound)
	}
	if parseMissing := findNodeByPath(parseNode, "/missing"); parseMissing != nil {
		parseT.Fatalf("expected missing node lookup to return nil, got %#v", parseMissing)
	}

	parseMatches := cacheEntriesForNode(&Node{Path: "/app/child"}, []CacheEntry{
		{Key: "users", OwnerPaths: []string{"/app/child"}},
		{Key: "settings", OwnerPaths: []string{"/other"}},
	})
	if len(parseMatches) != 1 || parseMatches[0].Key != "users" {
		parseT.Fatalf("expected cache entry match, got %#v", parseMatches)
	}
	if cacheEntriesForNode(nil, []CacheEntry{{Key: "unused"}}) != nil {
		parseT.Fatal("expected nil node cache lookup to return nil")
	}

	parseParams := map[string]string{"tab": "profile"}
	parseParamsClone := cloneParams(parseParams)
	parseParams["tab"] = "security"
	if parseParamsClone["tab"] != "profile" {
		parseT.Fatalf("expected cloneParams to return an independent map, got %#v", parseParamsClone)
	}

	parseStack := mapRouteStack([]router.RouteStackInspection{{
		ID:             "dashboard",
		Path:           "/dashboard",
		Params:         map[string]string{"section": "reports"},
		HasLoader:      true,
		HasBeforeEnter: true,
		Metadata: router.Metadata{
			Title:        "Reports",
			Description:  "Quarterly reports",
			CanonicalURL: "/dashboard",
		},
	}})
	if len(parseStack) != 1 || parseStack[0].Params["section"] != "reports" || !parseStack[0].HasLoader || !parseStack[0].HasBeforeEnter {
		parseT.Fatalf("expected mapped route stack entry, got %#v", parseStack)
	}

	parseLoaders := mapRouteLoaders([]router.RouteLoaderInspection{{
		Key:     "report@/dashboard",
		Path:    "/dashboard",
		Pending: true,
		HasData: false,
		Error:   "loading",
	}})
	if len(parseLoaders) != 1 || parseLoaders[0].Key != "report@/dashboard" || !parseLoaders[0].Pending || parseLoaders[0].Error != "loading" {
		parseT.Fatalf("expected mapped route loader entry, got %#v", parseLoaders)
	}

	if parseGot := formatQuery(map[string][]string{"b": {"2"}, "a": {"1", "3"}}); parseGot != "a=1,3 & b=2" {
		parseT.Fatalf("formatQuery() = %q, want sorted query summary", parseGot)
	}
	if parseGot2 := formatParams(map[string]string{"id": "7", "account": "primary"}); parseGot2 != "account=primary & id=7" {
		parseT.Fatalf("formatParams() = %q, want sorted param summary", parseGot2)
	}
	if parseGot3 := formatStringMap(map[string]string{"role": "admin", "app": "atlas"}); parseGot3 != "app=atlas & role=admin" {
		parseT.Fatalf("formatStringMap() = %q, want sorted map summary", parseGot3)
	}
	if parseGot4 := formatStringMap(nil); parseGot4 != "none" {
		parseT.Fatalf("formatStringMap(nil) = %q, want none", parseGot4)
	}
	if parseGot5 := emptyFallback(" ", "fallback"); parseGot5 != "fallback" {
		parseT.Fatalf("emptyFallback(blank) = %q, want fallback", parseGot5)
	}
	if parseGot6 := emptyFallback("value", "fallback"); parseGot6 != "value" {
		parseT.Fatalf("emptyFallback(value) = %q, want value", parseGot6)
	}
	if parseGot7 := formatDurationNs(2_500_000); parseGot7 != "2.50ms" {
		parseT.Fatalf("formatDurationNs() = %q, want 2.50ms", parseGot7)
	}
	if parseGot8 := formatByteCount(512); parseGot8 != "512 B" {
		parseT.Fatalf("formatByteCount(512) = %q, want 512 B", parseGot8)
	}
	if parseGot9 := formatByteCount(2048); parseGot9 != "2.00 KiB" {
		parseT.Fatalf("formatByteCount(2048) = %q, want 2.00 KiB", parseGot9)
	}
	if parseGot10 := formatByteCount(2 * 1024 * 1024); parseGot10 != "2.00 MiB" {
		parseT.Fatalf("formatByteCount(2MiB) = %q, want 2.00 MiB", parseGot10)
	}
	if parseGot11 := formatTime(time.Time{}); parseGot11 != "n/a" {
		parseT.Fatalf("formatTime(zero) = %q, want n/a", parseGot11)
	}
	if parseGot12 := formatTime(time.Date(2026, 4, 6, 14, 5, 9, 0, time.UTC)); parseGot12 != "14:05:09" {
		parseT.Fatalf("formatTime(fixed) = %q, want 14:05:09", parseGot12)
	}
}
