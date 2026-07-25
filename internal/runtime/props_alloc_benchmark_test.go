package runtime

import "testing"

// v5 PA.1 — instrument the props-clone cost before changing anything (R3).
//
// MICROBENCH_REPORT_arm64.md's central finding is that cloneElementProps is the
// framework-wide allocation bottleneck: 46-56% of allocations across four major
// benchmarks, 53% of component-update allocations. It was left unfixed because
// the clone is a safety contract of the public CreateElement — callers may
// retain the props map they passed — so removing it is architectural, not a
// micro-edit.
//
// These benchmarks isolate the cost by props size, so PA.2 can compare its
// three candidate designs against a real baseline rather than a remembered one.
//
// Run:
//   go test ./internal/runtime -run ^$ -bench PropsClone -benchmem

// propsSink keeps benchmarked clones alive.
//
// Without it, Go's escape analysis proves a discarded clone never leaves the
// loop and stack-allocates it, so small maps report 0 B/op and 0 allocs/op —
// a number that would have handed PA.2 a baseline of "there is nothing to fix".
// Real elements retain their props, so the sink is the honest model.
var propsSink map[string]any

// propsOfSize builds a props map with n string keys, the shape real components
// produce (class, id, style, data-*, aria-*, handlers).
func propsOfSize(parseN int) map[string]any {
	parseProps := make(map[string]any, parseN)
	parseKeys := []string{"class", "id", "style", "title", "role", "data-x", "aria-label", "href"}
	for parseI := range parseN {
		parseProps[parseKeys[parseI%len(parseKeys)]+string(rune('a'+parseI))] = "v"
	}
	return parseProps
}

// BenchmarkPropsCloneBySize is the core measurement: what one CreateElement
// costs purely in props cloning, at the sizes real elements use.
//
// The ≤4-key cases matter most. Go maps carry a high fixed overhead —
// a hashmap header plus at least one bucket regardless of occupancy — so a
// 2-key props map allocates far more than its contents justify. That is the
// observation PA.2's slice-backed small-map candidate is built on.
func BenchmarkPropsCloneBySize(parseB *testing.B) {
	for _, parseSize := range []int{1, 2, 4, 8, 16} {
		parseProps := propsOfSize(parseSize)
		parseB.Run(sizeLabel(parseSize), func(parseInner *testing.B) {
			parseInner.ReportAllocs()
			for parseInner.Loop() {
				propsSink = cloneElementProps(parseProps)
			}
		})
	}
}

// BenchmarkCreateElementVsOwned quantifies exactly what the safety contract
// costs, by comparing the cloning public constructor against the opt-out that
// already exists.
//
// The delta is PA.3's headroom: it is what an app would save today by migrating
// every call site to CreateElementOwned, which is candidate (a) in PA.2.
func BenchmarkCreateElementVsOwned(parseB *testing.B) {
	parseProps := propsOfSize(4)

	parseB.Run("CreateElement_clones", func(parseInner *testing.B) {
		parseInner.ReportAllocs()
		for parseInner.Loop() {
			_ = CreateElement("div", parseProps)
		}
	})

	parseB.Run("CreateElementOwned_no_clone", func(parseInner *testing.B) {
		parseInner.ReportAllocs()
		for parseInner.Loop() {
			_ = CreateElementOwned("div", parseProps)
		}
	})

	// The typed fast lane carries no props map at all. This is the floor any
	// PA.2 candidate is trying to approach for map-built elements.
	parseAttrs := []HostAttr{{Name: "class", Value: "a"}, {Name: "id", Value: "b"}}
	parseB.Run("CompactHost_no_map", func(parseInner *testing.B) {
		parseInner.ReportAllocs()
		for parseInner.Loop() {
			_ = CreateElementCompactHostOwned("div", "", parseAttrs)
		}
	})
}

// BenchmarkPropsCloneEmptyFastPath confirms the zero-prop case is already free,
// so PA.2 does not spend effort on a case that has no cost.
func BenchmarkPropsCloneEmptyFastPath(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		propsSink = cloneElementProps(nil)
	}
}

// BenchmarkComponentUpdateWithProps approximates the scenario the microbench
// report attributes 53% of allocations to: a component subtree re-rendering
// with map-built props on every element.
func BenchmarkComponentUpdateWithProps(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseChildren := make([]any, 0, 16)
		for parseI := range 16 {
			parseChildren = append(parseChildren, CreateElement("li", map[string]any{
				"class": "row",
				"id":    "r",
				"role":  "listitem",
			}, rune('a'+parseI)))
		}
		_ = CreateElement("ul", map[string]any{"class": "list"}, parseChildren...)
	}
}

func sizeLabel(parseN int) string {
	switch parseN {
	case 1:
		return "keys-1"
	case 2:
		return "keys-2"
	case 4:
		return "keys-4"
	case 8:
		return "keys-8"
	default:
		return "keys-16"
	}
}
