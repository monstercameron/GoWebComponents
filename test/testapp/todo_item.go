//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
)

// TodoItem represents a single todo item
func TodoItem(parseProps Attrs) *Element {
	parseId := parseProps["id"].(int)
	parseText := parseProps["text"].(string)
	parseCompleted := parseProps["completed"].(bool)
	parseToggle := parseProps["toggle"].(func())
	parseRemove := parseProps["remove"].(func())

	parseToggleHandler := GoUseFunc(parseToggle)
	parseRemoveHandler := GoUseFunc(parseRemove)

	parseClass := ""
	if parseCompleted {
		parseClass = "line-through text-gray-400"
	}

	parseAttrs := Attrs{
		"type":     "checkbox",
		"onchange": parseToggleHandler,
	}
	if parseCompleted {
		parseAttrs["checked"] = "checked"
	}

	return Div(Attrs{"class": "todo-item flex items-center justify-between p-2"},
		Div(Attrs{"class": "flex items-center gap-2"},
			Input(parseAttrs),
			Span(Attrs{"id": fmt.Sprintf("todo-text-%d", parseId), "class": parseClass}, Text(parseText)),
		),
		Button(Attrs{"class": "px-2 py-1 bg-red-500 text-white rounded", "onclick": parseRemoveHandler, "data-id": fmt.Sprintf("%d", parseId)}, Text("Delete")),
	)
}
