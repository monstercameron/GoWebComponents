package runtime2

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var storeCanonicalRenderNodeBenchmarkSink *canonicalRenderNode

// buildRenderNodeMapBenchmarkPayload builds one host-element-heavy map payload used by render-node parse compare benchmarks.
func buildRenderNodeMapBenchmarkPayload() map[string]any {
	buildItems := make([]any, 0, 48)
	for parseIndex := range 48 {
		buildItems = append(buildItems, map[string]any{
			"kind": "host-element",
			"tag":  "li",
			"key":  "row-" + strconv.Itoa(parseIndex),
			"props": map[string]any{
				"class":  "row",
				"data-i": strconv.Itoa(parseIndex),
			},
			"children": []any{
				map[string]any{
					"kind": "text",
					"text": "item-" + strconv.Itoa(parseIndex),
				},
			},
		})
	}
	return map[string]any{
		"kind": "host-element",
		"tag":  "section",
		"props": map[string]any{
			"class":      "panel",
			"aria-label": "benchmark",
		},
		"children": []any{
			map[string]any{
				"kind":     "host-element",
				"tag":      "ul",
				"children": buildItems,
			},
		},
	}
}

// parseBuildCanonicalRenderNodeLegacyBenchmark preserves the previous render-node parse flow for compare benchmarks.
func parseBuildCanonicalRenderNodeLegacyBenchmark(parseValue any) (*canonicalRenderNode, error) {
	switch getValue := parseValue.(type) {
	case nil:
		return &canonicalRenderNode{
			getKind: RenderNodeKindFragment,
		}, nil
	case string:
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: getValue,
		}, nil
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: parseFormatCanonicalScalarValue(getValue),
		}, nil
	case []any:
		buildChildren := make([]*canonicalRenderNode, 0, len(getValue))
		for _, getChildValue := range getValue {
			getChildNode, parseChildErr := parseBuildCanonicalRenderNodeLegacyBenchmark(getChildValue)
			if parseChildErr != nil {
				return nil, parseChildErr
			}
			buildChildren = append(buildChildren, getChildNode)
		}
		return &canonicalRenderNode{
			getKind:     RenderNodeKindFragment,
			getChildren: buildChildren,
		}, nil
	case map[string]any:
		return parseBuildCanonicalRenderNodeFromMapLegacyBenchmark(getValue)
	default:
		parseReflectValue := reflect.ValueOf(parseValue)
		if parseReflectValue.IsValid() && parseReflectValue.Kind() == reflect.Map {
			buildMapValue, parseMapValueErr := parseBuildCanonicalMapValue(parseReflectValue)
			if parseMapValueErr == nil {
				return parseBuildCanonicalRenderNodeFromMapLegacyBenchmark(buildMapValue)
			}
		}
		buildTextPayload, parseTextErr := parseBuildCanonicalJSON(parseValue)
		if parseTextErr != nil {
			return nil, parseTextErr
		}
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: buildTextPayload,
		}, nil
	}
}

// parseBuildCanonicalRenderNodeFromMapLegacyBenchmark preserves the previous map-node kind/tag normalization behavior for compare benchmarks.
func parseBuildCanonicalRenderNodeFromMapLegacyBenchmark(parseMapValue map[string]any) (*canonicalRenderNode, error) {
	buildKind := strings.ToLower(strings.TrimSpace(parseGetCanonicalStringValue(parseMapValue["kind"])))
	if buildKind == "" {
		switch {
		case parseMapValue["tag"] != nil:
			buildKind = "host-element"
		case parseMapValue["text"] != nil && parseMapValue["children"] == nil:
			buildKind = "text"
		case parseMapValue["children"] != nil:
			buildKind = "fragment"
		default:
			buildTextPayload, parseTextErr := parseBuildCanonicalJSON(parseMapValue)
			if parseTextErr != nil {
				return nil, parseTextErr
			}
			return &canonicalRenderNode{
				getKind: RenderNodeKindText,
				getText: buildTextPayload,
			}, nil
		}
	}
	switch buildKind {
	case "text":
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: parseGetCanonicalStringValue(parseMapValue["text"]),
		}, nil
	case "fragment":
		buildChildren, parseChildrenErr := parseBuildCanonicalChildren(parseMapValue["children"])
		if parseChildrenErr != nil {
			return nil, parseChildrenErr
		}
		buildKey := parseGetCanonicalStringValue(parseMapValue["key"])
		return &canonicalRenderNode{
			getKind:     RenderNodeKindFragment,
			getKey:      buildKey,
			hasKey:      hasCanonicalNodeKeyText(buildKey),
			getChildren: buildChildren,
		}, nil
	case "host-element", "element":
		buildTag := strings.TrimSpace(parseGetCanonicalStringValue(parseMapValue["tag"]))
		if parseTagErr := ValidateWorkerRenderableHostTag(buildTag); parseTagErr != nil {
			return nil, parseTagErr
		}
		buildProps, parsePropsErr := parseBuildCanonicalProps(parseMapValue)
		if parsePropsErr != nil {
			return nil, parsePropsErr
		}
		buildChildren, parseChildrenErr := parseBuildCanonicalChildren(parseMapValue["children"])
		if parseChildrenErr != nil {
			return nil, parseChildrenErr
		}
		buildKey := parseGetCanonicalStringValue(parseMapValue["key"])
		return &canonicalRenderNode{
			getKind:     RenderNodeKindHostElement,
			getTag:      buildTag,
			getKey:      buildKey,
			hasKey:      hasCanonicalNodeKeyText(buildKey),
			getProps:    buildProps,
			getChildren: buildChildren,
		}, nil
	default:
		return nil, fmt.Errorf("runtime2: unsupported render node kind %q", buildKind)
	}
}

// BenchmarkParseBuildCanonicalRenderNodeCurrentVsLegacy compares current render-node map parsing against the previous kind/tag normalization path.
func BenchmarkParseBuildCanonicalRenderNodeCurrentVsLegacy(parseB *testing.B) {
	parseRenderNode := buildRenderNodeMapBenchmarkPayload()
	parseB.Run("legacy_kind_tag_normalization", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildNode, parseErr := parseBuildCanonicalRenderNodeLegacyBenchmark(parseRenderNode)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalRenderNodeLegacyBenchmark returned error: %v", parseErr)
			}
			storeCanonicalRenderNodeBenchmarkSink = buildNode
		}
	})
	parseB.Run("current_kind_tag_fast_path", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildNode, parseErr := parseBuildCanonicalRenderNode(parseRenderNode)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalRenderNode returned error: %v", parseErr)
			}
			storeCanonicalRenderNodeBenchmarkSink = buildNode
		}
	})
}
