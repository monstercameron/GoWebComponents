//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

func FetchExample(_ Attrs) *Element {
	url, setUrl := hooks.UseState("https://jsonplaceholder.typicode.com/posts/1")
	currentUrl := url()

	getFetchState, refetch := hooks.UseFetch(currentUrl)
	fetchState := getFetchState()

	manualFetch := func(this js.Value, args []js.Value) interface{} {
		refetch()
		return nil
	}

	handleUrlChange := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newUrl := args[0].Get("target").Get("value").String()
			setUrl(newUrl)
		}
		return nil
	}

	setJsonPlaceholder := func(this js.Value, args []js.Value) interface{} {
		setUrl("https://jsonplaceholder.typicode.com/posts/1")
		return nil
	}

	setHttpBin := func(this js.Value, args []js.Value) interface{} {
		setUrl("https://httpbin.org/json")
		return nil
	}

	setRandomUser := func(this js.Value, args []js.Value) interface{} {
		setUrl("https://randomuser.me/api/")
		return nil
	}

	// Render response data
	var responseContent interface{}
	if fetchState.Loading {
		responseContent = dom.P(Attrs{
			"class": "text-blue-500 italic",
		}, dom.Text("🔄 Loading..."))
	} else if fetchState.Error != "" {
		responseContent = dom.P(Attrs{
			"class": "text-red-500 font-bold",
		}, dom.Text(fmt.Sprintf("❌ Error: %s", fetchState.Error)))
	} else if fetchState.Data != nil {
		responseContent = dom.Div(Attrs{},
			dom.P(Attrs{
				"class": "text-green-500 font-bold mb-2",
			}, dom.Text("✅ Success! Data received:")),
			dom.Pre(Attrs{
				"class": "bg-gray-800 text-gray-100 p-4 rounded-lg overflow-x-auto font-mono text-sm border border-gray-600",
			}, dom.Text(fmt.Sprintf("%+v", fetchState.Data))),
		)
	} else {
		responseContent = dom.P(Attrs{
			"class": "text-gray-400 italic",
		}, dom.Text("🔗 Click 'Fetch Data' to make a request"))
	}

	return dom.Div(Attrs{
		"class": "max-w-4xl mx-auto mt-8 p-6 bg-gray-800 rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold mb-4 text-white",
		}, dom.Text("🌐 Fetch Example")),

		dom.P(Attrs{
			"class": "mb-6 text-gray-300",
		}, dom.Text("Demonstrates the hooks.UseFetch hook for API calls.")),

		// URL Input Section
		dom.Div(Attrs{
			"class": "border border-gray-600 bg-gray-900 p-4 mb-4 rounded-lg",
		},
			dom.H3(Attrs{
				"class": "text-lg font-semibold mb-3 text-blue-400",
			}, dom.Text("📡 API Endpoint")),

			dom.Label(Attrs{
				"class": "block text-gray-300 mb-2",
			}, dom.Text("URL:")),

			dom.Input(Attrs{
				"type":        "url",
				"value":       currentUrl,
				"oninput":     js.FuncOf(handleUrlChange),
				"class":       "w-full bg-gray-700 text-white border border-gray-600 px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				"placeholder": "Enter API URL",
			}),

			dom.Div(Attrs{
				"class": "mt-4",
			},
				dom.Span(Attrs{
					"class": "text-gray-300 mr-2",
				}, dom.Text("Quick presets:")),
				dom.Button(Attrs{
					"onclick": js.FuncOf(setJsonPlaceholder),
					"class":   "bg-gray-700 text-white border border-blue-400 px-3 py-1 rounded hover:bg-gray-600 transition-colors text-sm mr-2",
				}, dom.Text("JSONPlaceholder")),
				dom.Button(Attrs{
					"onclick": js.FuncOf(setHttpBin),
					"class":   "bg-gray-700 text-white border border-blue-400 px-3 py-1 rounded hover:bg-gray-600 transition-colors text-sm mr-2",
				}, dom.Text("HTTPBin")),
				dom.Button(Attrs{
					"onclick": js.FuncOf(setRandomUser),
					"class":   "bg-gray-700 text-white border border-blue-400 px-3 py-1 rounded hover:bg-gray-600 transition-colors text-sm",
				}, dom.Text("Random User")),
			),
		),

		// Fetch Controls Section
		dom.Div(Attrs{
			"class": "border border-gray-600 bg-gray-900 p-4 mb-4 rounded-lg",
		},
			dom.H3(Attrs{
				"class": "text-lg font-semibold mb-3 text-blue-400",
			}, dom.Text("🚀 Fetch Controls")),

			dom.Button(Attrs{
				"onclick": js.FuncOf(manualFetch),
				"disabled": func() string {
					if fetchState.Loading {
						return "true"
					}
					return ""
				}(),
				"class": func() string {
					if fetchState.Loading {
						return "bg-gray-600 text-gray-400 cursor-not-allowed px-6 py-2 rounded-lg"
					}
					return "bg-blue-500 text-white px-6 py-2 rounded-lg hover:bg-blue-600 transition-colors"
				}(),
			}, func() interface{} {
				if fetchState.Loading {
					return dom.Text("Fetching...")
				}
				return dom.Text("Fetch Data")
			}()),
		),

		// Response Section
		dom.Div(Attrs{
			"class": "border border-gray-600 bg-gray-900 p-4 rounded-lg",
		},
			dom.H3(Attrs{
				"class": "text-lg font-semibold mb-3 text-blue-400",
			}, dom.Text("📦 Response")),
			responseContent,
		),
	)
}

func main() {
	container := js.Global().Get("document").Call("getElementById", "app")
	element := dom.CreateElement(FetchExample, nil)
	render.ToElement(element, container)
	select {}
}
