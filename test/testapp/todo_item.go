//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)

// TodoItem represents a single todo item
func TodoItem(props dom.Attrs) *dom.Element {
	id := props["id"].(int)
	text := props["text"].(string)
	completed := props["completed"].(bool)
	toggle := props["toggle"].(func())
	remove := props["remove"].(func())

	toggleHandler := hooks.GoUseFunc(toggle)
	removeHandler := hooks.GoUseFunc(remove)

	class := ""
	if completed {
		class = "line-through text-gray-400"
	}

	attrs := dom.Attrs{
		"type":     "checkbox",
		"onchange": toggleHandler,
	}
	if completed {
		attrs["checked"] = "checked"
	}

	return dom.Div(dom.Attrs{"class": "todo-item flex items-center justify-between p-2"},
		dom.Div(dom.Attrs{"class": "flex items-center gap-2"},
			dom.Input(attrs),
			dom.Span(dom.Attrs{"id": fmt.Sprintf("todo-text-%d", id), "class": class}, dom.Text(text)),
		),
		dom.Button(dom.Attrs{"class": "px-2 py-1 bg-red-500 text-white rounded", "onclick": removeHandler, "data-id": fmt.Sprintf("%d", id)}, dom.Text("Delete")),
	)
}
