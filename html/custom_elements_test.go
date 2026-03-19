package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestCustomElementSeparatesAttributesFromProperties(t *testing.T) {
	node := CustomElement("demo-rating-card", CustomElementProps{
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
		Properties: map[string]interface{}{
			"score":  4,
			"config": map[string]interface{}{"accent": "cyan"},
		},
	}, Span(Props{Slot: "actions"}, Text("Refresh")))
	if node == nil {
		t.Fatal("expected custom element node")
	}
	if node.Type != "demo-rating-card" {
		t.Fatalf("expected custom element tag, got %#v", node.Type)
	}
	if node.Props["id"] != "rating-card" {
		t.Fatalf("expected id prop, got %#v", node.Props["id"])
	}
	if node.Props["palette"] != "ocean" {
		t.Fatalf("expected reflected palette attribute, got %#v", node.Props["palette"])
	}
	if node.Props["interactive"] != "" {
		t.Fatalf("expected presence attribute, got %#v", node.Props["interactive"])
	}
	if _, ok := node.Props["disabled"]; ok {
		t.Fatalf("expected disabled presence attribute to be omitted, got %#v", node.Props["disabled"])
	}
	if node.Props[customElementPropertyPrefix+"score"] != 4 {
		t.Fatalf("expected score property, got %#v", node.Props[customElementPropertyPrefix+"score"])
	}
	if config, ok := node.Props[customElementPropertyPrefix+"config"].(map[string]interface{}); !ok || config["accent"] != "cyan" {
		t.Fatalf("expected config property, got %#v", node.Props[customElementPropertyPrefix+"config"])
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected one slotted child, got %d", len(node.Children))
	}
	child, ok := node.Children[0].(*ui.Element)
	if !ok || child.Props["slot"] != "actions" {
		t.Fatalf("expected child slot attribute, got %#v", node.Children[0])
	}
}

func TestCustomElementSkipsPropertyOnlyValuesDuringSSR(t *testing.T) {
	node := CustomElement("demo-rating-card", CustomElementProps{
		Props: Props{
			ID: "rating-card",
		},
		Attributes: map[string]string{
			"palette": "sunset",
		},
		Properties: map[string]interface{}{
			"score":  7,
			"config": map[string]interface{}{"series": []int{2, 4, 7}},
		},
	}, Span(Props{Slot: "summary"}, Text("Healthy demand")))

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("expected custom element SSR render, got %v", err)
	}
	if !strings.Contains(markup, `<demo-rating-card`) {
		t.Fatalf("expected custom element tag in markup, got %q", markup)
	}
	if !strings.Contains(markup, `palette="sunset"`) {
		t.Fatalf("expected reflected attribute in markup, got %q", markup)
	}
	if !strings.Contains(markup, `slot="summary"`) {
		t.Fatalf("expected slotted child markup, got %q", markup)
	}
	if strings.Contains(markup, "score=") || strings.Contains(markup, "config=") || strings.Contains(markup, "__gwc_prop__:") {
		t.Fatalf("expected property-only values to stay out of SSR markup, got %q", markup)
	}
}
