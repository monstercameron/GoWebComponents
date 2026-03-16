//go:build js && wasm
// +build js,wasm

package examples

import (
	"fmt"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

var themeContext = CreateContext("light")
var densityContext = CreateContext("comfortable")

func ThemeLabel(props Attrs) *Element {
	theme := GoUseContext[string](themeContext)
	density := GoUseContext[string](densityContext)

	return P(Attrs{}, Text("Theme from GoUseContext: "+theme+" | Density: "+density))
}

func ThemeConsumerLabel(props Attrs) *Element {
	return CreateElement(themeContext.Consumer, map[string]interface{}{
		"render": func(value interface{}) *Element {
			theme, _ := value.(string)
			return P(Attrs{}, Text("Theme from Consumer: "+theme))
		},
	})
}

func ThemeDefaultLabel(props Attrs) *Element {
	return P(Attrs{}, Text("Theme default outside provider: "+GoUseContext[string](themeContext)))
}

func NestedThemeLabel(props Attrs) *Element {
	return P(Attrs{}, Text("Nested provider theme: "+GoUseContext[string](themeContext)))
}

func ContextExample(props Attrs) *Element {
	theme, setTheme := GoUseState("midnight")
	currentTheme := theme()

	toggleTheme := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if currentTheme == "midnight" {
			setTheme("sunrise")
		} else {
			setTheme("midnight")
		}
		return nil
	})

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text("Context API Example")),
		P(Attrs{}, Text("This shows default values, a provider update, and nested provider override behavior.")),
		CreateElement(ThemeDefaultLabel, nil),
		CreateElement(themeContext.Provider, map[string]interface{}{
			"value": currentTheme,
		},
			CreateElement(densityContext.Provider, map[string]interface{}{
				"value": "compact",
			},
				CreateElement(ThemeLabel, nil),
				CreateElement(ThemeConsumerLabel, nil),
				CreateElement(themeContext.Provider, map[string]interface{}{
					"value": "nested-override",
				},
					CreateElement(NestedThemeLabel, nil),
				),
			),
		),
		Button(Attrs{
			"onclick": toggleTheme,
		}, Text("Toggle provider theme")),
		P(Attrs{}, Text("Current provider theme: "+currentTheme)),
	)
}

func ContextExamplesDemo() {
	rootContainer := js.Global().Get("document").Call("getElementById", "app")
	if rootContainer.IsUndefined() || rootContainer.IsNull() {
		fmt.Printf("❌ ContextExamplesDemo [ERROR]: No element with id 'app' found in the DOM!\n")
		return
	}

	Render(CreateElement(ContextExample, nil), rootContainer)
	fmt.Printf("🎉 ContextExamplesDemo [SUCCESS]: Context example rendered\n")
}
