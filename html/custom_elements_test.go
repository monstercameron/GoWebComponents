package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestCustomElementSeparatesAttributesFromProperties(parseT *testing.T) {
	parseNode := CustomElement("demo-rating-card", CustomElementProps{
		Props: Props{
			ID:    "rating-card",
			Class: "widget-shell",
		},
		Attributes: map[string]string{
			"palette": "ocean",
		},
		Presence: map[string]bool{
			"interactive": true,
			"disabled":    false,
		},
		Properties: map[string]any{
			"score":  4,
			"config": map[string]any{"accent": "cyan"},
		},
	}, Span(Props{Slot: "actions"}, Text("Refresh")))
	if parseNode == nil {
		parseT.Fatal("expected custom element node")
	}
	if parseNode.Type != "demo-rating-card" {
		parseT.Fatalf("expected custom element tag, got %#v", parseNode.Type)
	}
	if parseNode.Props["id"] != "rating-card" {
		parseT.Fatalf("expected id prop, got %#v", parseNode.Props["id"])
	}
	if parseNode.Props["palette"] != "ocean" {
		parseT.Fatalf("expected reflected palette attribute, got %#v", parseNode.Props["palette"])
	}
	if parseNode.Props["interactive"] != "" {
		parseT.Fatalf("expected presence attribute, got %#v", parseNode.Props["interactive"])
	}
	if _, parseOk := parseNode.Props["disabled"]; parseOk {
		parseT.Fatalf("expected disabled presence attribute to be omitted, got %#v", parseNode.Props["disabled"])
	}
	if parseNode.Props[customElementPropertyPrefix+"score"] != 4 {
		parseT.Fatalf("expected score property, got %#v", parseNode.Props[customElementPropertyPrefix+"score"])
	}
	if parseConfig, parseOk2 := parseNode.Props[customElementPropertyPrefix+"config"].(map[string]any); !parseOk2 || parseConfig["accent"] != "cyan" {
		parseT.Fatalf("expected config property, got %#v", parseNode.Props[customElementPropertyPrefix+"config"])
	}
	if len(parseNode.Children) != 1 {
		parseT.Fatalf("expected one slotted child, got %d", len(parseNode.Children))
	}
	parseChild, parseOk3 := parseNode.Children[0].(*ui.Element)
	if !parseOk3 || runtime.EnsureElementProps(parseChild)["slot"] != "actions" {
		parseT.Fatalf("expected child slot attribute, got %#v", parseNode.Children[0])
	}
}

func TestCustomElementSkipsPropertyOnlyValuesDuringSSR(parseT *testing.T) {
	parseNode := CustomElement("demo-rating-card", CustomElementProps{
		Props: Props{
			ID: "rating-card",
		},
		Attributes: map[string]string{
			"palette": "sunset",
		},
		Properties: map[string]any{
			"score":  7,
			"config": map[string]any{"series": []int{2, 4, 7}},
		},
	}, Span(Props{Slot: "summary"}, Text("Healthy demand")))

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("expected custom element SSR render, got %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `<demo-rating-card`) {
		parseT.Fatalf("expected custom element tag in markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `palette="sunset"`) {
		parseT.Fatalf("expected reflected attribute in markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `slot="summary"`) {
		parseT.Fatalf("expected slotted child markup, got %q", parseMarkup)
	}
	if strings.Contains(parseMarkup, "score=") || strings.Contains(parseMarkup, "config=") || strings.Contains(parseMarkup, "__gwc_prop__:") {
		parseT.Fatalf("expected property-only values to stay out of SSR markup, got %q", parseMarkup)
	}
}
