//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
)

// TodoItem represents a single todo item
func TodoItem(props Attrs) *Element {
	id := props["id"].(int)
	text := props["text"].(string)
	completed := props["completed"].(bool)
	toggle := props["toggle"].(func())
	remove := props["remove"].(func())

	toggleHandler := GoUseFunc(toggle)
	removeHandler := GoUseFunc(remove)

	class := ""
	if completed {
		class = "line-through text-gray-400"
	}

	attrs := Attrs{
		"type":     "checkbox",
		"onchange": toggleHandler,
	}
	if completed {
		attrs["checked"] = "checked"
	}

	return Div(Attrs{"class": "todo-item flex items-center justify-between p-2"},
		Div(Attrs{"class": "flex items-center gap-2"},
			Input(attrs),
			Span(Attrs{"id": fmt.Sprintf("todo-text-%d", id), "class": class}, Text(text)),
		),
		Button(Attrs{"class": "px-2 py-1 bg-red-500 text-white rounded", "onclick": removeHandler, "data-id": fmt.Sprintf("%d", id)}, Text("Delete")),
	)
}
