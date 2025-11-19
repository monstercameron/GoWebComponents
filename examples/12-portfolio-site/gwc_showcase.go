//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/router"
)

// GWCShowcaseSection presents GoWebComponents capabilities with feature cards.

// Uses dark theme styling to create visual contrast and highlight the framework's

// professional grade features and developer experience benefits.

func GWCShowcaseSection(_ Attrs) *Element {

	return dom.Section(

		Attrs{

			"id": "gwc-showcase",

			"class": "py-20 bg-gradient-to-br from-gray-900/50 to-purple-900/20 border-b border-white/10 text-white",
		},

		dom.Div(

			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},

			dom.Div(

				Attrs{"class": "text-center mb-16"},

				dom.H2(

					Attrs{"class": "text-4xl font-bold mb-4"},

					"GoWebComponents Showcase",
				),

				dom.P(

					Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"},

					"Discover the innovative features and capabilities that make GoWebComponents unique",
				),
			),

			dom.Div(

				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8"},

				GWCFeatureCard("⚡", "Lightning Fast", "WebAssembly performance with Go's efficiency and memory safety"),

				GWCFeatureCard("🛡️", "Type Safety", "Compile-time error checking eliminates runtime surprises"),

				GWCFeatureCard("🎯", "Developer Experience", "Hot reload, debugging tools, and familiar Go syntax"),

				GWCFeatureCard("🏗️", "Component Architecture", "Reusable, composable components with clear data flow"),

				GWCFeatureCard("🎨", "Modern UI", "Beautiful interfaces with Tailwind CSS integration"),

				GWCFeatureCard("🚀", "Production Ready", "Battle-tested framework with real-world applications"),

				GWCFeatureCard("🔒", "Advanced Form", "Validation • Effects • Memo • Go Routines"),
			),
		),
	)

}

// GWCFeatureCard renders individual framework features with glassmorphism styling.

// Designed for dark backgrounds with semi-transparent cards and hover interactions.

func GWCFeatureCard(icon, title, description string) *Element {

	return dom.Div(

		Attrs{"class": "bg-white/5 backdrop-blur-sm p-6 rounded-xl hover:bg-white/10 transition-all duration-300 border border-white/10"},

		dom.Div(Attrs{"class": "text-3xl mb-4"}, icon),

		dom.H3(Attrs{"class": "text-xl font-semibold mb-3"}, title),

		dom.P(Attrs{"class": "text-gray-400"}, description),
	)

}

// GWCExamplesSection displays interactive mini-applications demonstrating GoWebComponents.

// Features 3D flip cards, source code viewing, and lazy loading for optimal performance.

// Includes an advanced form example with on-demand GitHub source fetching.

func GWCExamplesSection(_ Attrs) *Element {

	return dom.Section(

		Attrs{

			"id": "examples",

			"class": "py-20 bg-transparent",
		},

		dom.Div(

			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},

			dom.Div(

				Attrs{"class": "text-center mb-16"},

				dom.H2(

					Attrs{"class": "text-4xl font-bold text-white mb-4"},

					"Mini Apps Gallery",
				),

				dom.P(

					Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"},

					"7 interactive mini applications showcasing GoWebComponents capabilities",
				),
			),

			dom.Div(

				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-8 max-w-4xl mx-auto"},

				dom.CreateElement(MiniAppCard, Attrs{
					"icon":        "🖱️",
					"title":       "Click Counter",
					"description": "State management basics",
					"component":   MiniClickCounter,
					"sourceCode":  clickCounterSource,
				}),

				dom.CreateElement(MiniAppCard, Attrs{
					"icon":        "🎲",
					"title":       "Random Number",
					"description": "Effects and events",
					"component":   MiniRandomizer,
					"sourceCode":  randomizerSource,
				}),

				dom.CreateElement(MiniAppCard, Attrs{
					"icon":        "📝",
					"title":       "Quick Note",
					"description": "Input handling",
					"component":   MiniNotepad,
					"sourceCode":  notepadSource,
				}),

				dom.CreateElement(MiniAppCard, Attrs{
					"icon":        "🎨",
					"title":       "Color Picker",
					"description": "Dynamic styling",
					"component":   MiniColorPicker,
					"sourceCode":  colorPickerSource,
				}),

				dom.CreateElement(MiniAppCard, Attrs{
					"icon":        "⏱️",
					"title":       "Timer",
					"description": "Real-time updates",
					"component":   MiniTimer,
					"sourceCode":  timerSource,
				}),

				dom.CreateElement(MiniAppCard, Attrs{
					"icon":        "📊",
					"title":       "Vote Counter",
					"description": "Multiple states",
					"component":   MiniVoting,
					"sourceCode":  votingSource,
				}),

				// Full-width advanced example

				dom.Div(

					Attrs{"class": "md:col-span-2"},

					AdvancedFormShowcase(nil),
				),
			),
		),
	)

}

// AdvancedFormShowcase integrates the complex form example with GitHub source loading.

// Demonstrates GoUseFetch hook usage and lazy content loading patterns.

func AdvancedFormShowcase(_ Attrs) *Element {

	return dom.CreateElement(LazyMiniAppCard, Attrs{
		"icon":        "🔒",
		"title":       "Advanced Form",
		"description": "Validation • Effects • Memo • Go Routines • GoUseFetch",
		"component":   AdvancedFormExample,
		"sourceUrl":   "https://raw.githubusercontent.com/monstercameron/GoWebComponents/refs/heads/master/website/advanced_form.go",
	})

}

// LazyMiniAppCard renders a 3D flip card with app demo and source code viewing.

// Features lazy loading of source code from GitHub, loading states, error handling,

// and smooth 3D flip animations between app view and code view.

func LazyMiniAppCard(props Attrs) *Element {
	icon := props["icon"].(string)
	title := props["title"].(string)
	description := props["description"].(string)
	component := props["component"].(func(Attrs) *Element)
	sourceUrl := props["sourceUrl"].(string)

	showSource, setShowSource := hooks.UseState(false)

	sourceLoaded, setSourceLoaded := hooks.UseState(false)

	shouldFetch, setShouldFetch := hooks.UseState(false)

	// Use GoUseFetch but only when shouldFetch is true

	fetchUrl := func() string {

		if shouldFetch() {

			return sourceUrl

		}

		return "" // Empty URL means no fetch

	}()

	getFetchState, _ := hooks.UseFetch(fetchUrl)

	fetchState := getFetchState()

	// Determine current source code and loading state

	var sourceCode string

	var isLoading bool

	if !sourceLoaded() {

		sourceCode = "// Click 'View Code' to load source from GitHub..."

		isLoading = false

	} else if fetchState.Loading {

		sourceCode = ""

		isLoading = true

	} else if fetchState.Error != "" {

		sourceCode = fmt.Sprintf("// Error fetching source code: %s\n// Please check the URL: %s", fetchState.Error, sourceUrl)

		isLoading = false

	} else if fetchState.Data != nil {

		sourceCode = fmt.Sprintf("%v", fetchState.Data)

		isLoading = false

	} else {

		sourceCode = "// Source code not available"

		isLoading = false

	}

	toggleSource := hooks.GoUseFunc(func(event dom.GoEvent) {

		if !showSource() && !sourceLoaded() {

			// First time viewing source - trigger the API call

			setSourceLoaded(true)

			setShouldFetch(true)

			// The fetch will trigger automatically when shouldFetch becomes true

		}

		setShowSource(!showSource())

	})

	copyToClipboard := hooks.GoUseFunc(func(event dom.GoEvent) {

		if !isLoading && sourceCode != "" && sourceLoaded() {

			js.Global().Get("navigator").Get("clipboard").Call("writeText", sourceCode)

		}

	})

	return dom.Div(

		Attrs{

			"class": "relative group",

			"style": "perspective: 1000px; min-height: 450px;",
		},

		// 3D Flip Container with Shadow

		dom.Div(

			Attrs{

				"class": "relative w-full min-h-[450px] transition-all duration-700 bg-white/5 rounded-xl shadow-lg hover:shadow-xl border border-white/10 backdrop-blur-sm",

				"style": func() string {

					if showSource() {

						return "transform: rotateY(180deg); transform-style: preserve-3d;"

					}

					return "transform: rotateY(0deg); transform-style: preserve-3d;"

				}(),
			},

			// Front Side - App View

			dom.Div(

				Attrs{

					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden flex flex-col",

					"style": "-webkit-backface-visibility: hidden; backface-visibility: hidden; transform: rotateY(0deg);",
				},

				// Header

				dom.Div(

					Attrs{"class": "p-6 border-b border-white/10 bg-white/5"},

					dom.Div(

						Attrs{"class": "flex items-center justify-between"},

						dom.Div(

							Attrs{"class": "flex items-center space-x-2"},

							dom.Span(Attrs{"class": "text-2xl"}, icon),

							dom.Div(nil,

								dom.H3(Attrs{"class": "font-semibold text-white"}, title),

								dom.P(Attrs{"class": "text-xs text-gray-400"}, description),
							),
						),

						dom.Button(

							Attrs{

								"class": "text-xs px-3 py-1 bg-white/10 text-white rounded-full hover:bg-white/20 transition-all duration-300 hover:scale-105",

								"onclick": toggleSource,
							},

							"</> View Code",
						),
					),
				),

				// App Component

				dom.Div(

					Attrs{"class": "flex-1 overflow-y-auto p-6 bg-transparent"},

					dom.CreateElement(component, nil),
				),
			),

			// Back Side - Code View

			dom.Div(

				Attrs{

					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden bg-gray-900",

					"style": "-webkit-backface-visibility: hidden; backface-visibility: hidden; transform: rotateY(180deg);",
				},

				dom.Div(

					Attrs{"class": "p-6 border-b border-white/10 bg-white/5"},

					dom.Div(

						Attrs{"class": "flex items-center justify-between"},

						dom.Div(

							Attrs{"class": "flex items-center space-x-2"},

							dom.Span(Attrs{"class": "text-green-400 text-lg"}, "{}"),

							dom.H3(Attrs{"class": "font-semibold text-white text-sm"}, title+" Source"),
						),

						dom.Button(

							Attrs{

								"class": "text-xs px-3 py-1 bg-blue-600 text-white rounded-full hover:bg-blue-500 transition-all duration-300 hover:scale-105",

								"onclick": toggleSource,
							},

							"🎨 View App",
						),
					),
				),

				dom.Div(

					Attrs{"class": "relative p-6 pb-12 h-full"},

					func() *Element {
						// Only render content when showSource is true (back side is visible)
						if !sourceLoaded() {
							// Not loaded yet, show placeholder
							return dom.Pre(
								Attrs{"class": "text-gray-400 text-sm font-mono whitespace-pre-wrap"},
								dom.Code(nil, dom.Text("// Click 'View Code' to load source from GitHub...")),
							)
						}

						if isLoading {

							// Show loading spinner

							return dom.Div(

								Attrs{"class": "flex items-center justify-center h-full"},

								dom.Div(

									Attrs{"class": "text-center"},

									dom.Div(Attrs{"class": "inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-green-400 mb-4"}),

									dom.P(Attrs{"class": "text-green-400 text-sm"}, "Loading source code from GitHub..."),
								),
							)

						}

						// Show source code
						return HighlightGoCode(sourceCode)

					}(),

					// Floating Clipboard Button (only show when not loading)

					func() *Element {
						if !showSource() {
							return nil
						}

						if !isLoading && sourceCode != "" && sourceLoaded() {

							return dom.Button(

								Attrs{

									"class": "absolute top-8 right-8 p-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg shadow-lg transition-all duration-300 hover:scale-110 opacity-80 hover:opacity-100",

									"onclick": copyToClipboard,

									"title": "Copy to clipboard",
								},

								dom.Span(Attrs{"class": "text-base"}, "📋"),
							)

						}

						return dom.Div(nil) // Empty div when loading

					}(),
				),
			),
		),
	)

}

// MiniAppCard creates a mini app showcase card with 3D flip animation and clipboard

func MiniAppCard(props Attrs) *Element {
	icon := props["icon"].(string)
	title := props["title"].(string)
	description := props["description"].(string)
	component := props["component"].(func(Attrs) *Element)
	sourceCode := props["sourceCode"].(string)

	showSource, setShowSource := hooks.UseState(false)

	toggleSource := hooks.GoUseFunc(func(event dom.GoEvent) {

		setShowSource(!showSource())

	})

	copyToClipboard := hooks.GoUseFunc(func(event dom.GoEvent) {

		js.Global().Get("navigator").Get("clipboard").Call("writeText", sourceCode)

		// Could add a toast notification here

	})

	return dom.Div(

		Attrs{

			"class": "relative group",

			"style": "perspective: 1000px; min-height: 450px;",
		},

		// 3D Flip Container with Shadow

		dom.Div(

			Attrs{

				"class": "relative w-full min-h-[450px] transition-all duration-700 bg-white/5 rounded-xl shadow-lg hover:shadow-xl border border-white/10 backdrop-blur-sm",

				"style": func() string {

					if showSource() {

						return "transform: rotateY(180deg); transform-style: preserve-3d;"

					}

					return "transform: rotateY(0deg); transform-style: preserve-3d;"

				}(),
			},

			// Front Side - App View

			dom.Div(

				Attrs{

					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden flex flex-col",

					"style": "-webkit-backface-visibility: hidden; backface-visibility: hidden; transform: rotateY(0deg);",
				},

				// Header

				dom.Div(

					Attrs{"class": "p-6 border-b border-white/10 bg-white/5"},

					dom.Div(

						Attrs{"class": "flex items-center justify-between"},

						dom.Div(

							Attrs{"class": "flex items-center space-x-2"},

							dom.Span(Attrs{"class": "text-2xl"}, icon),

							dom.Div(nil,

								dom.H3(Attrs{"class": "font-semibold text-white"}, title),

								dom.P(Attrs{"class": "text-xs text-gray-400"}, description),
							),
						),

						dom.Button(

							Attrs{

								"class": "text-xs px-3 py-1 bg-white/10 text-white rounded-full hover:bg-white/20 transition-all duration-300 hover:scale-105",

								"onclick": toggleSource,
							},

							"</> View Code",
						),
					),
				),

				// App Component

				dom.Div(

					Attrs{"class": "flex-1 overflow-y-auto p-6 bg-transparent"},

					dom.CreateElement(component, nil),
				),
			),

			// Back Side - Code View

			dom.Div(

				Attrs{

					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden bg-gray-900",

					"style": "-webkit-backface-visibility: hidden; backface-visibility: hidden; transform: rotateY(180deg);",
				},

				dom.Div(

					Attrs{"class": "p-6 border-b border-white/10 bg-white/5"},

					dom.Div(

						Attrs{"class": "flex items-center justify-between"},

						dom.Div(

							Attrs{"class": "flex items-center space-x-2"},

							dom.Span(Attrs{"class": "text-green-400 text-lg"}, "{}"),

							dom.H3(Attrs{"class": "font-semibold text-white text-sm"}, title+" Source"),
						),

						dom.Button(

							Attrs{

								"class": "text-xs px-3 py-1 bg-blue-600 text-white rounded-full hover:bg-blue-500 transition-all duration-300 hover:scale-105",

								"onclick": toggleSource,
							},

							"🎨 View App",
						),
					),
				),

				dom.Div(

					Attrs{"class": "relative p-6 pb-12 h-full"},

					func() *Element {
						if !showSource() {
							return nil
						}
						return HighlightGoCode(sourceCode)
					}(),

					// Floating Clipboard Button

					func() *Element {
						if !showSource() {
							return nil
						}
						return dom.Button(

							Attrs{

								"class": "absolute top-8 right-8 p-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg shadow-lg transition-all duration-300 hover:scale-110 opacity-80 hover:opacity-100",

								"onclick": copyToClipboard,

								"title": "Copy to clipboard",
							},

							dom.Span(Attrs{"class": "text-base"}, "📋"),
						)
					}(),
				),
			),
		),
	)

}

// Mini App 1: Click Counter

func MiniClickCounter(props Attrs) *Element {

	clickCount, setClickCount := hooks.UseState(0)

	incrementClicks := hooks.GoUseFunc(func(event dom.GoEvent) {

		setClickCount(clickCount() + 1)

	})

	resetClicks := hooks.GoUseFunc(func(event dom.GoEvent) {

		setClickCount(0)

	})

	return dom.Div(

		Attrs{"class": "text-center space-y-3"},

		dom.P(Attrs{"class": "text-2xl font-bold text-blue-400"}, dom.Text(strconv.Itoa(clickCount()))),

		dom.Div(

			Attrs{"class": "space-x-2"},

			dom.Button(Attrs{"class": "px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700", "onclick": incrementClicks}, "+1"),

			dom.Button(Attrs{"class": "px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700", "onclick": resetClicks}, "Reset"),
		),
	)

}

// Mini App 2: Random Number Generator

func MiniRandomizer(props Attrs) *Element {

	randomNum, setRandomNum := hooks.UseState(42)

	// Initialize random seed once when component mounts
	hooks.UseEffect(func() func() {
		rand.Seed(time.Now().UnixNano())
		return nil
	})

	generateRandom := hooks.GoUseFunc(func(event dom.GoEvent) {
		// Use Go's native random number generator
		newNum := rand.Intn(100) + 1
		setRandomNum(newNum)
	})

	return dom.Div(

		Attrs{"class": "text-center space-y-3"},

		dom.P(Attrs{"class": "text-2xl font-bold text-purple-400"}, dom.Text(strconv.Itoa(randomNum()))),

		dom.Button(Attrs{"class": "px-4 py-2 bg-purple-600 text-white text-sm rounded hover:bg-purple-700", "onclick": generateRandom}, "Generate"),
	)

}

// Mini App 3: Quick Note

func MiniNotepad(props Attrs) *Element {

	noteText, setNoteText := hooks.UseState("Sample note text")

	updateNote := hooks.GoUseFunc(func(event dom.GoEvent) {

		if noteText() == "Sample note text" {

			setNoteText("Updated note!")

		} else {

			setNoteText("Sample note text")

		}

	})

	return dom.Div(

		Attrs{"class": "space-y-3"},

		dom.Div(

			Attrs{"class": "w-full p-3 border border-white/10 rounded text-sm bg-black/20 min-h-16"},

			dom.P(Attrs{"class": "text-white"}, noteText()),
		),

		dom.Div(

			Attrs{"class": "flex justify-between items-center"},

			dom.Button(Attrs{"class": "px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700", "onclick": updateNote}, "Edit Note"),

			dom.P(Attrs{"class": "text-xs text-gray-400"}, dom.Text(strconv.Itoa(len(noteText()))), " characters"),
		),
	)

}

// Mini App 4: Color Picker

func MiniColorPicker(props Attrs) *Element {

	selectedColor, setSelectedColor := hooks.UseState("bg-blue-500")

	colors := []string{"bg-red-500", "bg-blue-500", "bg-green-500", "bg-yellow-500", "bg-purple-500", "bg-pink-500"}

	colorButtons := make([]interface{}, len(colors))

	for i, color := range colors {

		currentColor := color

		colorButtons[i] = dom.Button(Attrs{

			"class": "w-6 h-6 rounded-full " + color + " hover:scale-110 transition-transform",

			"onclick": hooks.GoUseFunc(func(event dom.GoEvent) {

				setSelectedColor(currentColor)

			}),
		})

	}

	return dom.Div(

		Attrs{"class": "space-y-3"},

		dom.Div(Attrs{"class": "w-full h-16 rounded " + selectedColor()}),

		dom.Div(Attrs{"class": "flex space-x-2 justify-center"}, colorButtons...),
	)

}

// Mini App 5: Simple Timer

func MiniTimer(props Attrs) *Element {

	timerCount, setTimerCount := hooks.UseState(0)

	timerRunning, setTimerRunning := hooks.UseState(false)

	toggleTimer := hooks.GoUseFunc(func(event dom.GoEvent) {

		setTimerRunning(!timerRunning())

	})

	resetTimer := hooks.GoUseFunc(func(event dom.GoEvent) {

		setTimerCount(0)

		setTimerRunning(false)
	})

	// Define tick handler using GoUseFunc
	tick := hooks.GoUseFunc(func() {
		setTimerCount(timerCount() + 1)
	})

	hooks.UseEffect(func() func() {
		if timerRunning() {
			timeoutID := js.Global().Call("setTimeout", tick, 1000)
			_ = timeoutID
		}
		return nil
	})

	return dom.Div(
		Attrs{"class": "text-center space-y-3"},
		dom.P(Attrs{"class": "text-2xl font-bold text-green-400"}, dom.Text(strconv.Itoa(timerCount())), "s"),
		dom.Div(

			Attrs{"class": "space-x-2"},

			dom.Button(Attrs{"class": "px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700", "onclick": toggleTimer}, func() string {

				if timerRunning() {

					return "Stop"

				}

				return "Start"

			}()),

			dom.Button(Attrs{"class": "px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700", "onclick": resetTimer}, "Reset"),
		),
	)

}

// Mini App 6: Vote Counter

func MiniVoting(props Attrs) *Element {

	upvotes, setUpvotes := hooks.UseState(12)

	downvotes, setDownvotes := hooks.UseState(3)

	addUpvote := hooks.GoUseFunc(func(event dom.GoEvent) {

		setUpvotes(upvotes() + 1)

	})

	addDownvote := hooks.GoUseFunc(func(event dom.GoEvent) {

		setDownvotes(downvotes() + 1)

	})

	totalVotes := upvotes() + downvotes()

	upvotePercentage := 0

	if totalVotes > 0 {

		upvotePercentage = (upvotes() * 100) / totalVotes

	}

	return dom.Div(

		Attrs{"class": "space-y-3"},

		dom.Div(

			Attrs{"class": "flex justify-between items-center"},

			dom.Button(Attrs{"class": "flex items-center space-x-1 px-2 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700", "onclick": addUpvote},

				dom.Span(nil, "👍"),

				dom.Span(nil, dom.Text(strconv.Itoa(upvotes()))),
			),

			dom.Button(Attrs{"class": "flex items-center space-x-1 px-2 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700", "onclick": addDownvote},

				dom.Span(nil, "👎"),

				dom.Span(nil, dom.Text(strconv.Itoa(downvotes()))),
			),
		),

		dom.P(Attrs{"class": "text-xs text-gray-400 text-center"}, dom.Text(strconv.Itoa(upvotePercentage)), "% approval"),
	)

}

// Source code strings for each mini app

var clickCounterSource = `func MiniClickCounter(props Attrs) *Element {

    clickCount, setClickCount := hooks.UseState(0)

    incrementClicks := hooks.GoUseFunc(func(event dom.GoEvent) {

        setClickCount(clickCount() + 1)

    })

    resetClicks := hooks.GoUseFunc(func(event dom.GoEvent) {

        setClickCount(0)

    })

    return dom.Div(

        Attrs{"class": "text-center space-y-3"},

        dom.P(Attrs{"class": "text-2xl font-bold text-indigo-600"}, 

          dom.Text(strconv.Itoa(clickCount()))),

        dom.Div(Attrs{"class": "space-x-2"},

            dom.Button(Attrs{"onclick": incrementClicks}, "+1"),

            dom.Button(Attrs{"onclick": resetClicks}, "Reset"),

        ),

    )

}`

var randomizerSource = `func MiniRandomizer(props Attrs) *Element {

    randomNum, setRandomNum := hooks.UseState(42)

    hooks.UseEffect(func() {

        rand.Seed(time.Now().UnixNano())

        return

    })

    generateRandom := hooks.GoUseFunc(func(event dom.GoEvent) {

        newNum := rand.Intn(100) + 1

        setRandomNum(newNum)

    })

    return dom.Div(

        Attrs{"class": "text-center space-y-3"},

        dom.P(Attrs{"class": "text-2xl font-bold text-purple-600"}, 

          dom.Text(strconv.Itoa(randomNum()))),

        dom.Button(Attrs{"onclick": generateRandom}, "Generate"),

    )

}`

var notepadSource = `func MiniNotepad(props Attrs) *Element {

    noteText, setNoteText := hooks.UseState("Type here...")

    handleInput := hooks.GoUseFunc(func(event dom.GoEvent) {

        setNoteText(event.Target.Get("value").String())

    })

    return dom.Div(Attrs{"class": "space-y-3"},

        Textarea(Attrs{

            "value": noteText(),

            "oninput": handleInput,

            "placeholder": "Type your note...",

        }),

        dom.P(nil, dom.Text(strconv.Itoa(len(noteText()))), " characters"),

    )

}`

var colorPickerSource = `func MiniColorPicker(props Attrs) *Element {

    selectedColor, setSelectedColor := hooks.UseState("bg-blue-500")

    colors := []string{"bg-red-500", "bg-blue-500", 

                      "bg-green-500", "bg-yellow-500"}

    colorButtons := make([]interface{}, len(colors))

    for i, color := range colors {

        currentColor := color

        colorButtons[i] = dom.Button(Attrs{

            "class": "w-6 h-6 rounded-full " + color,

            "onclick": hooks.GoUseFunc(func(event dom.GoEvent) {

                setSelectedColor(currentColor)

            }),

        })

    }

    return dom.Div(Attrs{"class": "space-y-3"},

        dom.Div(Attrs{"class": "w-full h-16 rounded " + selectedColor()}),

        dom.Div(Attrs{"class": "flex space-x-2"}, colorButtons...),

    )

}`

var timerSource = `func MiniTimer(props Attrs) *Element {

    timerCount, setTimerCount := hooks.UseState(0)

    timerRunning, setTimerRunning := hooks.UseState(false)

    toggleTimer := hooks.GoUseFunc(func(event dom.GoEvent) {

        setTimerRunning(!timerRunning())

    })

    hooks.UseEffect(func() {

        if timerRunning() {

            js.Global().Call("setTimeout", js.FuncOf(

                func(this js.Value, args []js.Value) interface{} {

                    setTimerCount(timerCount() + 1)

                    return nil

                }), 1000)

        }

        return

    })

    return dom.Div(Attrs{"class": "text-center space-y-3"},

        dom.P(nil, dom.Text(strconv.Itoa(timerCount())), "s"),

        dom.Button(Attrs{"onclick": toggleTimer}, 

               timerRunning() ? "Stop" : "Start"),

    )

}`

var votingSource = `func MiniVoting(props Attrs) *Element {

    upvotes, setUpvotes := hooks.UseState(12)

    downvotes, setDownvotes := hooks.UseState(3)

    addUpvote := hooks.GoUseFunc(func(event dom.GoEvent) {

        setUpvotes(upvotes() + 1)

    })

    addDownvote := hooks.GoUseFunc(func(event dom.GoEvent) {

        setDownvotes(downvotes() + 1)

    })

    totalVotes := upvotes() + downvotes()

    upvotePercentage := (upvotes() * 100) / totalVotes

    return dom.Div(Attrs{"class": "space-y-3"},

        dom.Div(Attrs{"class": "flex justify-between"},

            dom.Button(Attrs{"onclick": addUpvote}, "👍 ", upvotes()),

            dom.Button(Attrs{"onclick": addDownvote}, "👎 ", downvotes()),

        ),

        dom.P(nil, dom.Text(strconv.Itoa(upvotePercentage)), "% approval"),

    )

}`

// WhyGoWebComponentsSection explains the benefits and rationale behind GoWebComponents

func WhyGoWebComponentsSection(_ Attrs) *Element {

	return dom.Section(

		Attrs{

			"id": "api",

			"class": "py-20 bg-gradient-to-br from-gray-900/50 to-blue-900/20 border-t border-white/10",
		},

		dom.Div(

			Attrs{"class": "container mx-auto px-6"},

			dom.Div(

				Attrs{"class": "max-w-4xl mx-auto text-center mb-16"},

				dom.H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-8 text-white"}, "Why GoWebComponents?"),

				dom.P(

					Attrs{"class": "text-xl text-gray-300 leading-relaxed mb-8"},

					"Born from the need to build complex, performant web applications without the JavaScript ecosystem's complexity. ",

					"GoWebComponents brings Go's elegance, safety, and performance to the frontend.",
				),
			),

			// Comparison cards

			dom.Div(

				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12 max-w-6xl mx-auto"},

				// Traditional approach

				dom.Div(

					Attrs{"class": "bg-red-900/10 border border-red-500/20 rounded-2xl p-8 backdrop-blur-sm"},

					dom.H3(Attrs{"class": "text-2xl font-bold text-red-300 mb-6 flex items-center"},

						dom.Span(Attrs{"class": "mr-3"}, "❌"),

						"Traditional Web Development"),

					dom.Ul(Attrs{"class": "space-y-4 text-red-200"},

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Complex build pipelines and toolchains"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Runtime errors and type coercion issues"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Separate backend/frontend codebases"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Heavy node_modules dependencies"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"State management complexity"),
					),
				),

				// GoWebComponents approach

				dom.Div(

					Attrs{"class": "bg-green-900/10 border border-green-500/20 rounded-2xl p-8 backdrop-blur-sm"},

					dom.H3(Attrs{"class": "text-2xl font-bold text-green-300 mb-6 flex items-center"},

						dom.Span(Attrs{"class": "mr-3"}, "✅"),

						"GoWebComponents Approach"),

					dom.Ul(Attrs{"class": "space-y-4 text-green-200"},

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Single Go codebase for everything"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Compile-time error checking"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Shared types and logic"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Zero external dependencies"),

						dom.Li(Attrs{"class": "flex items-start"},

							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),

							"Built-in state management"),
					),
				),
			),

			// Stats section

			dom.Div(

				Attrs{"class": "mt-16 grid grid-cols-1 md:grid-cols-3 gap-8 max-w-4xl mx-auto"},

				StatCard("10x", "Faster Development", "No build tools, instant feedback"),

				StatCard("100%", "Type Safe", "Go's compiler catches all errors"),

				StatCard("0", "Dependencies", "Pure Go, no node_modules"),
			),

			// Documentation CTA

			dom.Div(

				Attrs{"class": "mt-16 text-center"},

				func() *Element {

					// Store hooks.GoUseFunc result in variable for proper event handling

					navigateToDocs := hooks.GoUseFunc(func(event dom.GoEvent) {

						router.Navigate("/docs")

					})

					return dom.Div(

						Attrs{"class": "bg-white/5 backdrop-blur-sm rounded-2xl p-8 shadow-lg border border-white/10 max-w-2xl mx-auto"},

						dom.H3(

							Attrs{"class": "text-2xl font-bold text-white mb-4"},

							"Ready to Get Started?",
						),

						dom.P(

							Attrs{"class": "text-gray-300 mb-6"},

							"Explore our comprehensive documentation with API references, tutorials, and best practices to build your next web application with GoWebComponents.",
						),

						dom.Button(

							Attrs{

								"class": "px-8 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 font-semibold text-lg cursor-pointer",

								"onclick": navigateToDocs,
							},

							"📚 View Documentation",
						),
					)

				}(),
			),
		),
	)

}
