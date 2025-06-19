//go:build js && wasm
// +build js,wasm

package website

import (
	"fmt"
	"math/rand"
	"strconv"
	"syscall/js"
	"time"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// GWCShowcaseSection highlights GoWebComponents features
func GWCShowcaseSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "gwc-showcase",
			"class": "py-20 bg-gradient-to-br from-gray-900 to-purple-900 text-white",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold mb-4"},
					"GoWebComponents Showcase",
				),
				P(
					Attrs{"class": "text-xl text-gray-300 max-w-3xl mx-auto"},
					"Discover the innovative features and capabilities that make GoWebComponents unique",
				),
			),

			Div(
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

// GWCFeatureCard creates a feature highlight card
func GWCFeatureCard(icon, title, description string) *Element {
	return Div(
		Attrs{"class": "bg-white/10 backdrop-blur-sm p-6 rounded-xl hover:bg-white/20 transition-all duration-300 border border-white/20"},
		Div(Attrs{"class": "text-3xl mb-4"}, icon),
		H3(Attrs{"class": "text-xl font-semibold mb-3"}, title),
		P(Attrs{"class": "text-gray-300"}, description),
	)
}

// GWCExamplesSection showcases 7 mini apps with source code
func GWCExamplesSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "examples",
			"class": "py-20 bg-gray-50",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Mini Apps Gallery",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"7 interactive mini applications showcasing GoWebComponents capabilities",
				),
			),

			Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-8 max-w-4xl mx-auto"},
				MiniAppCard("🖱️", "Click Counter", "State management basics", MiniClickCounter, clickCounterSource),
				MiniAppCard("🎲", "Random Number", "Effects and events", MiniRandomizer, randomizerSource),
				MiniAppCard("📝", "Quick Note", "Input handling", MiniNotepad, notepadSource),
				MiniAppCard("🎨", "Color Picker", "Dynamic styling", MiniColorPicker, colorPickerSource),
				MiniAppCard("⏱️", "Timer", "Real-time updates", MiniTimer, timerSource),
				MiniAppCard("📊", "Vote Counter", "Multiple states", MiniVoting, votingSource),

				// Full-width advanced example
				Div(
					Attrs{"class": "md:col-span-2"},
					AdvancedFormShowcase(nil),
				),
			),
		),
	)
}

// AdvancedFormShowcase wraps the AdvancedFormExample with on-demand source fetching
func AdvancedFormShowcase(props Attrs) *Element {
	return LazyMiniAppCard("🔒", "Advanced Form", "Validation • Effects • Memo • Go Routines • GoUseFetch", AdvancedFormExample, "https://raw.githubusercontent.com/monstercameron/GoWebComponents/refs/heads/master/website/advanced_form.go")
}

// LazyMiniAppCard creates a mini app showcase card with on-demand source code fetching
func LazyMiniAppCard(icon, title, description string, component func(Attrs) *Element, sourceUrl string) *Element {
	showSource, setShowSource := GoUseState(false)
	sourceLoaded, setSourceLoaded := GoUseState(false)
	shouldFetch, setShouldFetch := GoUseState(false)

	// Use GoUseFetch but only when shouldFetch is true
	fetchUrl := func() string {
		if shouldFetch() {
			return sourceUrl
		}
		return "" // Empty URL means no fetch
	}()

	getFetchState, _ := GoUseFetch(fetchUrl)
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

	toggleSource := GoUseFunc(func(event GoEvent) {
		if !showSource() && !sourceLoaded() {
			// First time viewing source - trigger the API call
			setSourceLoaded(true)
			setShouldFetch(true)
			// The fetch will trigger automatically when shouldFetch becomes true
		}
		setShowSource(!showSource())
	})

	copyToClipboard := GoUseFunc(func(event GoEvent) {
		if !isLoading && sourceCode != "" && sourceLoaded() {
			js.Global().Get("navigator").Get("clipboard").Call("writeText", sourceCode)
		}
	})

	return Div(
		Attrs{
			"class": "relative group",
			"style": "perspective: 1000px; min-height: 450px;",
		},

		// 3D Flip Container with Shadow
		Div(
			Attrs{
				"class": "relative w-full min-h-[450px] transition-all duration-700 bg-white rounded-xl shadow-lg hover:shadow-xl border border-gray-200",
				"style": func() string {
					if showSource() {
						return "transform: rotateY(180deg); transform-style: preserve-3d;"
					}
					return "transform: rotateY(0deg); transform-style: preserve-3d;"
				}(),
			},

			// Front Side - App View
			Div(
				Attrs{
					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden flex flex-col",
					"style": "backface-visibility: hidden; transform: rotateY(0deg);",
				},

				// Header
				Div(
					Attrs{"class": "p-6 border-b border-gray-200 bg-gradient-to-r from-indigo-50 to-purple-50"},
					Div(
						Attrs{"class": "flex items-center justify-between"},
						Div(
							Attrs{"class": "flex items-center space-x-2"},
							Span(Attrs{"class": "text-2xl"}, icon),
							Div(nil,
								H3(Attrs{"class": "font-semibold text-gray-900"}, title),
								P(Attrs{"class": "text-xs text-gray-600"}, description),
							),
						),
						Button(
							Attrs{
								"class":   "text-xs px-3 py-1 bg-gray-800 text-white rounded-full hover:bg-gray-700 transition-all duration-300 hover:scale-105",
								"onclick": toggleSource,
							},
							"</> View Code",
						),
					),
				),

				// App Component
				Div(
					Attrs{"class": "flex-1 overflow-y-auto p-6 bg-gray-50"},
					component(nil),
				),
			),

			// Back Side - Code View
			Div(
				Attrs{
					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden bg-gray-900",
					"style": "backface-visibility: hidden; transform: rotateY(-180deg);",
				},

				// Code Header
				Div(
					Attrs{"class": "p-6 border-b border-gray-700 bg-gray-800"},
					Div(
						Attrs{"class": "flex items-center justify-between"},
						Div(
							Attrs{"class": "flex items-center space-x-2"},
							Span(Attrs{"class": "text-green-400 text-lg"}, "{}"),
							H3(Attrs{"class": "font-semibold text-white text-sm"}, title+" Source"),
						),
						Button(
							Attrs{
								"class":   "text-xs px-3 py-1 bg-indigo-600 text-white rounded-full hover:bg-indigo-500 transition-all duration-300 hover:scale-105",
								"onclick": toggleSource,
							},
							"🎨 View App",
						),
					),
				),

				// Code Content with Spinner or Source Code
				Div(
					Attrs{"class": "relative p-6 pb-12 h-full"},
					func() *Element {
						if isLoading {
							// Show loading spinner
							return Div(
								Attrs{"class": "flex items-center justify-center h-full"},
								Div(
									Attrs{"class": "text-center"},
									Div(Attrs{"class": "inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-green-400 mb-4"}),
									P(Attrs{"class": "text-green-400 text-sm"}, "Loading source code from GitHub..."),
								),
							)
						} else {
							// Show source code
							return Pre(Attrs{
								"class": "text-xs text-green-400 font-mono leading-relaxed overflow-x-auto h-full pb-8",
							}, sourceCode)
						}
					}(),

					// Floating Clipboard Button (only show when not loading)
					func() *Element {
						if !isLoading && sourceCode != "" && sourceLoaded() {
							return Button(
								Attrs{
									"class":   "absolute top-8 right-8 p-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg shadow-lg transition-all duration-300 hover:scale-110 opacity-80 hover:opacity-100",
									"onclick": copyToClipboard,
									"title":   "Copy to clipboard",
								},
								Span(Attrs{"class": "text-base"}, "📋"),
							)
						}
						return Div(nil) // Empty div when loading
					}(),
				),
			),
		),
	)
}

// MiniAppCard creates a mini app showcase card with 3D flip animation and clipboard
func MiniAppCard(icon, title, description string, component func(Attrs) *Element, sourceCode string) *Element {
	showSource, setShowSource := GoUseState(false)

	toggleSource := GoUseFunc(func(event GoEvent) {
		setShowSource(!showSource())
	})

	copyToClipboard := GoUseFunc(func(event GoEvent) {
		js.Global().Get("navigator").Get("clipboard").Call("writeText", sourceCode)
		// Could add a toast notification here
	})

	return Div(
		Attrs{
			"class": "relative group",
			"style": "perspective: 1000px; min-height: 450px;",
		},

		// 3D Flip Container with Shadow
		Div(
			Attrs{
				"class": "relative w-full min-h-[450px] transition-all duration-700 bg-white rounded-xl shadow-lg hover:shadow-xl border border-gray-200",
				"style": func() string {
					if showSource() {
						return "transform: rotateY(180deg); transform-style: preserve-3d;"
					}
					return "transform: rotateY(0deg); transform-style: preserve-3d;"
				}(),
			},

			// Front Side - App View
			Div(
				Attrs{
					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden flex flex-col",
					"style": "backface-visibility: hidden; transform: rotateY(0deg);",
				},

				// Header
				Div(
					Attrs{"class": "p-6 border-b border-gray-200 bg-gradient-to-r from-indigo-50 to-purple-50"},
					Div(
						Attrs{"class": "flex items-center justify-between"},
						Div(
							Attrs{"class": "flex items-center space-x-2"},
							Span(Attrs{"class": "text-2xl"}, icon),
							Div(nil,
								H3(Attrs{"class": "font-semibold text-gray-900"}, title),
								P(Attrs{"class": "text-xs text-gray-600"}, description),
							),
						),
						Button(
							Attrs{
								"class":   "text-xs px-3 py-1 bg-gray-800 text-white rounded-full hover:bg-gray-700 transition-all duration-300 hover:scale-105",
								"onclick": toggleSource,
							},
							"</> View Code",
						),
					),
				),

				// App Component
				Div(
					Attrs{"class": "flex-1 overflow-y-auto p-6 bg-gray-50"},
					component(nil),
				),
			),

			// Back Side - Code View
			Div(
				Attrs{
					"class": "absolute inset-0 w-full h-full rounded-xl overflow-hidden bg-gray-900",
					"style": "backface-visibility: hidden; transform: rotateY(-180deg);",
				},

				// Code Header
				Div(
					Attrs{"class": "p-6 border-b border-gray-700 bg-gray-800"},
					Div(
						Attrs{"class": "flex items-center justify-between"},
						Div(
							Attrs{"class": "flex items-center space-x-2"},
							Span(Attrs{"class": "text-green-400 text-lg"}, "{}"),
							H3(Attrs{"class": "font-semibold text-white text-sm"}, title+" Source"),
						),
						Button(
							Attrs{
								"class":   "text-xs px-3 py-1 bg-indigo-600 text-white rounded-full hover:bg-indigo-500 transition-all duration-300 hover:scale-105",
								"onclick": toggleSource,
							},
							"🎨 View App",
						),
					),
				),

				// Code Content with Floating Clipboard Button
				Div(
					Attrs{"class": "relative p-6 pb-12 h-full"},
					Pre(Attrs{
						"class": "text-xs text-green-400 font-mono leading-relaxed overflow-x-auto h-full pb-8",
					}, sourceCode),

					// Floating Clipboard Button
					Button(
						Attrs{
							"class":   "absolute top-8 right-8 p-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg shadow-lg transition-all duration-300 hover:scale-110 opacity-80 hover:opacity-100",
							"onclick": copyToClipboard,
							"title":   "Copy to clipboard",
						},
						Span(Attrs{"class": "text-base"}, "📋"),
					),
				),
			),
		),
	)
}

// Mini App 1: Click Counter
func MiniClickCounter(props Attrs) *Element {
	clickCount, setClickCount := GoUseState(0)

	incrementClicks := GoUseFunc(func(event GoEvent) {
		setClickCount(clickCount() + 1)
	})

	resetClicks := GoUseFunc(func(event GoEvent) {
		setClickCount(0)
	})

	return Div(
		Attrs{"class": "text-center space-y-3"},
		P(Attrs{"class": "text-2xl font-bold text-indigo-600"}, Text(strconv.Itoa(clickCount()))),
		Div(
			Attrs{"class": "space-x-2"},
			Button(Attrs{"class": "px-3 py-1 bg-indigo-600 text-white text-sm rounded hover:bg-indigo-700", "onclick": incrementClicks}, "+1"),
			Button(Attrs{"class": "px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700", "onclick": resetClicks}, "Reset"),
		),
	)
}

// Mini App 2: Random Number Generator
func MiniRandomizer(props Attrs) *Element {
	randomNum, setRandomNum := GoUseState(42)

	// Initialize random seed once when component mounts
	GoUseEffect(func() {
		rand.Seed(time.Now().UnixNano())
		return
	})

	generateRandom := GoUseFunc(func(event GoEvent) {
		// Use Go's native random number generator
		newNum := rand.Intn(100) + 1
		setRandomNum(newNum)
	})

	return Div(
		Attrs{"class": "text-center space-y-3"},
		P(Attrs{"class": "text-2xl font-bold text-purple-600"}, Text(strconv.Itoa(randomNum()))),
		Button(Attrs{"class": "px-4 py-2 bg-purple-600 text-white text-sm rounded hover:bg-purple-700", "onclick": generateRandom}, "Generate"),
	)
}

// Mini App 3: Quick Note
func MiniNotepad(props Attrs) *Element {
	noteText, setNoteText := GoUseState("Sample note text")

	updateNote := GoUseFunc(func(event GoEvent) {
		if noteText() == "Sample note text" {
			setNoteText("Updated note!")
		} else {
			setNoteText("Sample note text")
		}
	})

	return Div(
		Attrs{"class": "space-y-3"},
		Div(
			Attrs{"class": "w-full p-3 border border-gray-300 rounded text-sm bg-gray-50 min-h-16"},
			P(Attrs{"class": "text-gray-800"}, noteText()),
		),
		Div(
			Attrs{"class": "flex justify-between items-center"},
			Button(Attrs{"class": "px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700", "onclick": updateNote}, "Edit Note"),
			P(Attrs{"class": "text-xs text-gray-500"}, Text(strconv.Itoa(len(noteText()))), " characters"),
		),
	)
}

// Mini App 4: Color Picker
func MiniColorPicker(props Attrs) *Element {
	selectedColor, setSelectedColor := GoUseState("bg-blue-500")

	colors := []string{"bg-red-500", "bg-blue-500", "bg-green-500", "bg-yellow-500", "bg-purple-500", "bg-pink-500"}
	colorButtons := make([]interface{}, len(colors))

	for i, color := range colors {
		currentColor := color
		colorButtons[i] = Button(Attrs{
			"class": "w-6 h-6 rounded-full " + color + " hover:scale-110 transition-transform",
			"onclick": GoUseFunc(func(event GoEvent) {
				setSelectedColor(currentColor)
			}),
		})
	}

	return Div(
		Attrs{"class": "space-y-3"},
		Div(Attrs{"class": "w-full h-16 rounded " + selectedColor()}),
		Div(Attrs{"class": "flex space-x-2 justify-center"}, colorButtons...),
	)
}

// Mini App 5: Simple Timer
func MiniTimer(props Attrs) *Element {
	timerCount, setTimerCount := GoUseState(0)
	timerRunning, setTimerRunning := GoUseState(false)

	toggleTimer := GoUseFunc(func(event GoEvent) {
		setTimerRunning(!timerRunning())
	})

	resetTimer := GoUseFunc(func(event GoEvent) {
		setTimerCount(0)
		setTimerRunning(false)
	})

	GoUseEffect(func() {
		if timerRunning() {
			timeoutID := js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				setTimerCount(timerCount() + 1)
				return nil
			}), 1000)
			_ = timeoutID
		}
		return
	})

	return Div(
		Attrs{"class": "text-center space-y-3"},
		P(Attrs{"class": "text-2xl font-bold text-green-600"}, Text(strconv.Itoa(timerCount())), "s"),
		Div(
			Attrs{"class": "space-x-2"},
			Button(Attrs{"class": "px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700", "onclick": toggleTimer}, func() string {
				if timerRunning() {
					return "Stop"
				}
				return "Start"
			}()),
			Button(Attrs{"class": "px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700", "onclick": resetTimer}, "Reset"),
		),
	)
}

// Mini App 6: Vote Counter
func MiniVoting(props Attrs) *Element {
	upvotes, setUpvotes := GoUseState(12)
	downvotes, setDownvotes := GoUseState(3)

	addUpvote := GoUseFunc(func(event GoEvent) {
		setUpvotes(upvotes() + 1)
	})

	addDownvote := GoUseFunc(func(event GoEvent) {
		setDownvotes(downvotes() + 1)
	})

	totalVotes := upvotes() + downvotes()
	upvotePercentage := 0
	if totalVotes > 0 {
		upvotePercentage = (upvotes() * 100) / totalVotes
	}

	return Div(
		Attrs{"class": "space-y-3"},
		Div(
			Attrs{"class": "flex justify-between items-center"},
			Button(Attrs{"class": "flex items-center space-x-1 px-2 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700", "onclick": addUpvote},
				Span(nil, "👍"),
				Span(nil, Text(strconv.Itoa(upvotes()))),
			),
			Button(Attrs{"class": "flex items-center space-x-1 px-2 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700", "onclick": addDownvote},
				Span(nil, "👎"),
				Span(nil, Text(strconv.Itoa(downvotes()))),
			),
		),
		P(Attrs{"class": "text-xs text-gray-600 text-center"}, Text(strconv.Itoa(upvotePercentage)), "% approval"),
	)
}

// Source code strings for each mini app
var clickCounterSource = `func MiniClickCounter(props Attrs) *Element {
    clickCount, setClickCount := GoUseState(0)
    
    incrementClicks := GoUseFunc(func(event GoEvent) {
        setClickCount(clickCount() + 1)
    })
    
    resetClicks := GoUseFunc(func(event GoEvent) {
        setClickCount(0)
    })
    
    return Div(
        Attrs{"class": "text-center space-y-3"},
        P(Attrs{"class": "text-2xl font-bold text-indigo-600"}, 
          Text(strconv.Itoa(clickCount()))),
        Div(Attrs{"class": "space-x-2"},
            Button(Attrs{"onclick": incrementClicks}, "+1"),
            Button(Attrs{"onclick": resetClicks}, "Reset"),
        ),
    )
}`

var randomizerSource = `func MiniRandomizer(props Attrs) *Element {
    randomNum, setRandomNum := GoUseState(42)
    
    GoUseEffect(func() {
        rand.Seed(time.Now().UnixNano())
        return
    })
    
    generateRandom := GoUseFunc(func(event GoEvent) {
        newNum := rand.Intn(100) + 1
        setRandomNum(newNum)
    })
    
    return Div(
        Attrs{"class": "text-center space-y-3"},
        P(Attrs{"class": "text-2xl font-bold text-purple-600"}, 
          Text(strconv.Itoa(randomNum()))),
        Button(Attrs{"onclick": generateRandom}, "Generate"),
    )
}`

var notepadSource = `func MiniNotepad(props Attrs) *Element {
    noteText, setNoteText := GoUseState("Type here...")
    
    handleInput := GoUseFunc(func(event GoEvent) {
        setNoteText(event.Target.Get("value").String())
    })
    
    return Div(Attrs{"class": "space-y-3"},
        Textarea(Attrs{
            "value": noteText(),
            "oninput": handleInput,
            "placeholder": "Type your note...",
        }),
        P(nil, Text(strconv.Itoa(len(noteText()))), " characters"),
    )
}`

var colorPickerSource = `func MiniColorPicker(props Attrs) *Element {
    selectedColor, setSelectedColor := GoUseState("bg-blue-500")
    
    colors := []string{"bg-red-500", "bg-blue-500", 
                      "bg-green-500", "bg-yellow-500"}
    colorButtons := make([]interface{}, len(colors))
    
    for i, color := range colors {
        currentColor := color
        colorButtons[i] = Button(Attrs{
            "class": "w-6 h-6 rounded-full " + color,
            "onclick": GoUseFunc(func(event GoEvent) {
                setSelectedColor(currentColor)
            }),
        })
    }
    
    return Div(Attrs{"class": "space-y-3"},
        Div(Attrs{"class": "w-full h-16 rounded " + selectedColor()}),
        Div(Attrs{"class": "flex space-x-2"}, colorButtons...),
    )
}`

var timerSource = `func MiniTimer(props Attrs) *Element {
    timerCount, setTimerCount := GoUseState(0)
    timerRunning, setTimerRunning := GoUseState(false)
    
    toggleTimer := GoUseFunc(func(event GoEvent) {
        setTimerRunning(!timerRunning())
    })
    
    GoUseEffect(func() {
        if timerRunning() {
            js.Global().Call("setTimeout", js.FuncOf(
                func(this js.Value, args []js.Value) interface{} {
                    setTimerCount(timerCount() + 1)
                    return nil
                }), 1000)
        }
        return
    })
    
    return Div(Attrs{"class": "text-center space-y-3"},
        P(nil, Text(strconv.Itoa(timerCount())), "s"),
        Button(Attrs{"onclick": toggleTimer}, 
               timerRunning() ? "Stop" : "Start"),
    )
}`

var votingSource = `func MiniVoting(props Attrs) *Element {
    upvotes, setUpvotes := GoUseState(12)
    downvotes, setDownvotes := GoUseState(3)
    
    addUpvote := GoUseFunc(func(event GoEvent) {
        setUpvotes(upvotes() + 1)
    })
    
    addDownvote := GoUseFunc(func(event GoEvent) {
        setDownvotes(downvotes() + 1)
    })
    
    totalVotes := upvotes() + downvotes()
    upvotePercentage := (upvotes() * 100) / totalVotes
    
    return Div(Attrs{"class": "space-y-3"},
        Div(Attrs{"class": "flex justify-between"},
            Button(Attrs{"onclick": addUpvote}, "👍 ", upvotes()),
            Button(Attrs{"onclick": addDownvote}, "👎 ", downvotes()),
        ),
        P(nil, Text(strconv.Itoa(upvotePercentage)), "% approval"),
    )
}`

// WhyGoWebComponentsSection explains the benefits and rationale behind GoWebComponents
func WhyGoWebComponentsSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "api",
			"class": "py-20 bg-gradient-to-br from-gray-900 to-indigo-900",
		},
		Div(
			Attrs{"class": "container mx-auto px-6"},
			Div(
				Attrs{"class": "max-w-4xl mx-auto text-center mb-16"},
				H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-8 text-white"}, "Why GoWebComponents?"),
				P(
					Attrs{"class": "text-xl text-gray-300 leading-relaxed mb-8"},
					"Born from the need to build complex, performant web applications without the JavaScript ecosystem's complexity. ",
					"GoWebComponents brings Go's elegance, safety, and performance to the frontend.",
				),
			),

			// Comparison cards
			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12 max-w-6xl mx-auto"},

				// Traditional approach
				Div(
					Attrs{"class": "bg-red-900/20 border border-red-500/30 rounded-2xl p-8 backdrop-blur-sm"},
					H3(Attrs{"class": "text-2xl font-bold text-red-300 mb-6 flex items-center"},
						Span(Attrs{"class": "mr-3"}, "❌"),
						"Traditional Web Development"),
					Ul(Attrs{"class": "space-y-4 text-red-200"},
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Complex build pipelines and toolchains"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Runtime errors and type coercion issues"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Separate backend/frontend codebases"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Heavy node_modules dependencies"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"State management complexity"),
					),
				),

				// GoWebComponents approach
				Div(
					Attrs{"class": "bg-green-900/20 border border-green-500/30 rounded-2xl p-8 backdrop-blur-sm"},
					H3(Attrs{"class": "text-2xl font-bold text-green-300 mb-6 flex items-center"},
						Span(Attrs{"class": "mr-3"}, "✅"),
						"GoWebComponents Approach"),
					Ul(Attrs{"class": "space-y-4 text-green-200"},
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Single Go codebase for everything"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Compile-time error checking"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Shared types and logic"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Zero external dependencies"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Built-in state management"),
					),
				),
			),

			// Stats section
			Div(
				Attrs{"class": "mt-16 grid grid-cols-1 md:grid-cols-3 gap-8 max-w-4xl mx-auto"},
				StatCard("10x", "Faster Development", "No build tools, instant feedback"),
				StatCard("100%", "Type Safe", "Go's compiler catches all errors"),
				StatCard("0", "Dependencies", "Pure Go, no node_modules"),
			),

			// Documentation CTA
			Div(
				Attrs{"class": "mt-16 text-center"},
				func() *Element {
					// Store GoUseFunc result in variable for proper event handling
					navigateToDocs := GoUseFunc(func(event GoEvent) {
						Navigate("/docs")
					})

					return Div(
						Attrs{"class": "bg-white/10 backdrop-blur-sm rounded-2xl p-8 shadow-lg border border-white/20 max-w-2xl mx-auto"},
						H3(
							Attrs{"class": "text-2xl font-bold text-white mb-4"},
							"Ready to Get Started?",
						),
						P(
							Attrs{"class": "text-gray-300 mb-6"},
							"Explore our comprehensive documentation with API references, tutorials, and best practices to build your next web application with GoWebComponents.",
						),
						Button(
							Attrs{
								"class":   "px-8 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 font-semibold text-lg cursor-pointer",
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
