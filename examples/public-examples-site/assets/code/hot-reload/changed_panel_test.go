package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestChangedCounterPanelRendersMarkup verifies the changed panel renders its expected initial markup.
func TestChangedCounterPanelRendersMarkup(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(ChangedCounterPanel))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ChangedCounterPanel) error = %v", parseErr)
	}

	for _, parseNeedle := range []string{
		"Changed sibling subtree",
		"Changed subtree version: v1",
		"Changed count: 0",
		"Increment Changed Counter",
		"id=\"changed-panel\"",
		"id=\"changed-version\"",
		"id=\"changed-count\"",
	} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected changed panel markup to contain %q\n%s", parseNeedle, parseMarkup)
		}
	}
}

// TestChangedCounterPanelHandlesIncrement verifies the stored increment handler executes.
func TestChangedCounterPanelHandlesIncrement(parseT *testing.T) {
	parseNode := ChangedCounterPanel()
	parseButton := findHotReloadNodeByID(parseNode, "changed-increment")
	if parseButton == nil {
		parseT.Fatal("expected changed increment button node")
	}

	parseOnClick, parseOk := parseButton.Props["onclick"].(func())
	if !parseOk {
		parseT.Fatalf("expected onclick handler func(), got %#v", parseButton.Props["onclick"])
	}

	parseOnClick()
}

// findHotReloadNodeByID walks a rendered node tree and returns the first node with the matching id prop.
func findHotReloadNodeByID(parseNode *runtime.Element, parseID string) *runtime.Element {
	if parseNode == nil {
		return nil
	}
	if parseNode.Props["id"] == parseID {
		return parseNode
	}
	for _, parseChild := range parseNode.Children {
		parseElement, parseOk := parseChild.(*runtime.Element)
		if !parseOk {
			continue
		}
		if parseMatch := findHotReloadNodeByID(parseElement, parseID); parseMatch != nil {
			return parseMatch
		}
	}
	return nil
}
