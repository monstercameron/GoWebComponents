// ./examples/simple_state_examples.go
// Simple examples demonstrating different GoUseState use cases
// KISS principle - no fancy styling, just functionality

//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"
	"strconv"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)

// Simple struct for demonstrating object state
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// BackgroundTask represents a background operation state
type BackgroundTask struct {
	IsRunning  bool      `json:"isRunning"`
	Progress   int       `json:"progress"`
	Status     string    `json:"status"`
	CancelChan chan bool `json:"-"` // Don't serialize the channel
}

// Timer state for the continuous timer
type TimerState struct {
	IsRunning  bool      `json:"isRunning"`
	Seconds    int       `json:"seconds"`
	CancelChan chan bool `json:"-"` // Don't serialize the channel
}

// Global render tracking
var globalRenderCounter int = 0

// Helper function to get next render ID
func getNextRenderID() int {
	globalRenderCounter++
	return globalRenderCounter
}

// Counter component - demonstrates number state
func CounterExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("🔢 CounterExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	count, setCount := GoUseState(0)
	currentCount := count()

	fmt.Printf("🔢 CounterExample [RENDER #%d]: State values - count=%d\n", currentRender, currentCount)

	increment := func(this js.Value, args []js.Value) interface{} {
		newCount := currentCount + 1
		fmt.Printf("🔢 CounterExample [EVENT]: INCREMENT clicked - state transition %d→%d (will trigger render #%d)\n",
			currentCount, newCount, currentRender+1)
		setCount(newCount)
		return nil
	}

	decrement := func(this js.Value, args []js.Value) interface{} {
		newCount := currentCount - 1
		fmt.Printf("🔢 CounterExample [EVENT]: DECREMENT clicked - state transition %d→%d (will trigger render #%d)\n",
			currentCount, newCount, currentRender+1)
		setCount(newCount)
		return nil
	}

	reset := func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("🔢 CounterExample [EVENT]: RESET clicked - state transition %d→0 (will trigger render #%d)\n",
			currentCount, currentRender+1)
		setCount(0)
		return nil
	}

	fmt.Printf("🔢 CounterExample [RENDER #%d]: Generating DOM with count=%d\n", currentRender, currentCount)

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		dom.H3(Attrs{}, dom.Text(fmt.Sprintf("Counter Example (Render #%d)", currentRender))),
		dom.P(Attrs{}, dom.Text(fmt.Sprintf("Count: %d", currentCount))),
		Button(Attrs{
			"onclick": js.FuncOf(increment),
		}, Text("+")),
		Text(" "),
		Button(Attrs{
			"onclick": js.FuncOf(decrement),
		}, Text("-")),
		Text(" "),
		Button(Attrs{
			"onclick": js.FuncOf(reset),
		}, Text("Reset")),
	)
}

// Text input component - demonstrates string state
func TextInputExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("📝 TextInputExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	renderCount, setRenderCount := hooks.UseState(0)
	currentRender := getNextRenderID()
	fmt.Printf("📝 StringExample (Render #%d)\n", currentRender)

	renderCount, setRenderCount := hooks.UseState(0)
	currentRender := getNextRenderID()
	fmt.Printf("🔧 TextInputExample (Render #%d)\n", currentRender)

	text, setText := hooks.UseState("")
	currentText := text()

	fmt.Printf("📝 TextInputExample [RENDER #%d]: State values - text='%s' (length=%d)\n",
		currentRender, currentText, len(currentText))

	handleInput := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newText := args[0].Get("target").Get("value").String()
			fmt.Printf("📝 TextInputExample [EVENT]: INPUT changed - text transition '%s'→'%s' (will trigger render #%d)\n",
				currentText, newText, currentRender+1)
			setText(newText)
		}
		return nil
	}

	clear := func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("📝 TextInputExample [EVENT]: CLEAR clicked - text transition '%s'→'' (will trigger render #%d)\n",
			currentText, currentRender+1)
		setText("")
		return nil
	}

	fmt.Printf("📝 TextInputExample [RENDER #%d]: Generating DOM with text='%s'\n", currentRender, currentText)

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text(fmt.Sprintf("Text Input Example (Render #%d)", currentRender))),
		P(Attrs{}, Text("Type something:")),
		Input(Attrs{
			"type":    "text",
			"value":   currentText,
			"oninput": js.FuncOf(handleInput),
			"style":   "color: black;",
		}),
		Text(" "),
		Button(Attrs{
			"onclick": js.FuncOf(clear),
		}, Text("Clear")),
		P(Attrs{}, Text(fmt.Sprintf("You typed: %s", currentText))),
		P(Attrs{}, Text(fmt.Sprintf("Length: %d", len(currentText)))),
	)
}

// Toggle component - demonstrates boolean state
func ToggleExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("🔘 ToggleExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	isOn, setIsOn := GoUseState(false)
	currentState := isOn()

	fmt.Printf("🔘 ToggleExample [RENDER #%d]: State values - isOn=%v\n", currentRender, currentState)

	toggle := func(this js.Value, args []js.Value) interface{} {
		newState := !currentState
		fmt.Printf("🔘 ToggleExample [EVENT]: TOGGLE clicked - state transition %v→%v (will trigger render #%d)\n",
			currentState, newState, currentRender+1)
		setIsOn(newState)
		return nil
	}

	fmt.Printf("🔘 ToggleExample [RENDER #%d]: Generating DOM with isOn=%v\n", currentRender, currentState)

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text(fmt.Sprintf("Toggle Example (Render #%d)", currentRender))),
		P(Attrs{}, Text(fmt.Sprintf("Switch is: %s", func() string {
			if currentState {
				return "ON"
			}
			return "OFF"
		}()))),
		Button(Attrs{
			"onclick": js.FuncOf(toggle),
			"style": func() string {
				if currentState {
					return "background-color: green; color: white;"
				}
				return "background-color: red; color: white;"
			}(),
		}, Text(func() string {
			if currentState {
				return "Turn OFF"
			}
			return "Turn ON"
		}())),
	)
}

// Person form component - demonstrates struct state
func PersonFormExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("👤 PersonFormExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	person, setPerson := GoUseState(Person{Name: "", Age: 0})
	currentPerson := person()

	fmt.Printf("👤 PersonFormExample [RENDER #%d]: State values - person={Name:'%s', Age:%d}\n",
		currentRender, currentPerson.Name, currentPerson.Age)

	updateName := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newName := args[0].Get("target").Get("value").String()
			newPerson := Person{Name: newName, Age: currentPerson.Age}
			fmt.Printf("👤 PersonFormExample [EVENT]: NAME updated - person transition {Name:'%s',Age:%d}→{Name:'%s',Age:%d} (will trigger render #%d)\n",
				currentPerson.Name, currentPerson.Age, newName, currentPerson.Age, currentRender+1)
			setPerson(newPerson)
		}
		return nil
	}

	updateAge := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			ageStr := args[0].Get("target").Get("value").String()
			if age, err := strconv.Atoi(ageStr); err == nil {
				newPerson := Person{Name: currentPerson.Name, Age: age}
				fmt.Printf("👤 PersonFormExample [EVENT]: AGE updated - person transition {Name:'%s',Age:%d}→{Name:'%s',Age:%d} (will trigger render #%d)\n",
					currentPerson.Name, currentPerson.Age, currentPerson.Name, age, currentRender+1)
				setPerson(newPerson)
			} else {
				fmt.Printf("👤 PersonFormExample [ERROR]: Invalid age input '%s' - no state change\n", ageStr)
			}
		}
		return nil
	}

	reset := func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("👤 PersonFormExample [EVENT]: RESET clicked - person transition {Name:'%s',Age:%d}→{Name:'',Age:0} (will trigger render #%d)\n",
			currentPerson.Name, currentPerson.Age, currentRender+1)
		setPerson(Person{Name: "", Age: 0})
		return nil
	}

	fmt.Printf("👤 PersonFormExample [RENDER #%d]: Generating DOM with person={Name:'%s',Age:%d}\n",
		currentRender, currentPerson.Name, currentPerson.Age)

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text(fmt.Sprintf("Person Form Example (Render #%d)", currentRender))),
		P(Attrs{}, Text("Name:")),
		Input(Attrs{
			"type":    "text",
			"value":   currentPerson.Name,
			"oninput": js.FuncOf(updateName),
			"style":   "color: black;",
		}),
		P(Attrs{}, Text("Age:")),
		Input(Attrs{
			"type":    "number",
			"value":   strconv.Itoa(currentPerson.Age),
			"oninput": js.FuncOf(updateAge),
			"style":   "color: black;",
		}),
		Text(" "),
		Button(Attrs{
			"onclick": js.FuncOf(reset),
		}, Text("Reset")),
		P(Attrs{}, Text(fmt.Sprintf("Person: %s, Age: %d", currentPerson.Name, currentPerson.Age))),
	)
}

// Todo list component - demonstrates array/slice state
func TodoListExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("📋 TodoListExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	todos, setTodos := GoUseState([]string{})
	newTodo, setNewTodo := GoUseState("")
	currentTodos := todos()
	currentNewTodo := newTodo()

	fmt.Printf("📋 TodoListExample [RENDER #%d]: State values - todos=%d items %v, newTodo='%s'\n",
		currentRender, len(currentTodos), currentTodos, currentNewTodo)

	addTodo := func(this js.Value, args []js.Value) interface{} {
		todoText := currentNewTodo
		if todoText != "" {
			newTodos := make([]string, len(currentTodos)+1)
			copy(newTodos, currentTodos)
			newTodos[len(currentTodos)] = todoText
			fmt.Printf("📋 TodoListExample [EVENT]: ADD TODO '%s' - todos transition %d→%d items (will trigger render #%d)\n",
				todoText, len(currentTodos), len(newTodos), currentRender+1)
			setTodos(newTodos)
			setNewTodo("")
		} else {
			fmt.Printf("📋 TodoListExample [ERROR]: Cannot add empty todo - no state change\n")
		}
		return nil
	}

	removeTodo := func(index int) js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if index >= 0 && index < len(currentTodos) {
				todoText := currentTodos[index]
				newTodos := make([]string, 0, len(currentTodos)-1)
				for i, todo := range currentTodos {
					if i != index {
						newTodos = append(newTodos, todo)
					}
				}
				fmt.Printf("📋 TodoListExample [EVENT]: REMOVE TODO '%s' at index %d - todos transition %d→%d items (will trigger render #%d)\n",
					todoText, index, len(currentTodos), len(newTodos), currentRender+1)
				setTodos(newTodos)
			} else {
				fmt.Printf("📋 TodoListExample [ERROR]: Invalid remove index %d (total: %d) - no state change\n",
					index, len(currentTodos))
			}
			return nil
		})
	}

	handleInput := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newText := args[0].Get("target").Get("value").String()
			fmt.Printf("📋 TodoListExample [EVENT]: TODO INPUT changed - newTodo transition '%s'→'%s' (will trigger render #%d)\n",
				currentNewTodo, newText, currentRender+1)
			setNewTodo(newText)
		}
		return nil
	}

	clearAll := func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("📋 TodoListExample [EVENT]: CLEAR ALL clicked - todos transition %d→0 items (will trigger render #%d)\n",
			len(currentTodos), currentRender+1)
		setTodos([]string{})
		return nil
	}

	fmt.Printf("📋 TodoListExample [RENDER #%d]: Generating DOM with %d todos\n", currentRender, len(currentTodos))

	// Create todo items
	todoItems := make([]interface{}, 0, len(currentTodos))
	for i, todo := range currentTodos {
		todoItems = append(todoItems, Li(Attrs{
			"key": strconv.Itoa(i),
		},
			Text(todo),
			Text(" "),
			Button(Attrs{
				"onclick": removeTodo(i),
			}, Text("Remove")),
		))
	}

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text(fmt.Sprintf("Todo List Example (Render #%d)", currentRender))),
		P(Attrs{}, Text("Add a new todo:")),
		Input(Attrs{
			"type":    "text",
			"value":   currentNewTodo,
			"oninput": js.FuncOf(handleInput),
			"style":   "color: black;",
		}),
		Text(" "),
		Button(Attrs{
			"onclick": js.FuncOf(addTodo),
		}, Text("Add")),
		Text(" "),
		Button(Attrs{
			"onclick": js.FuncOf(clearAll),
		}, Text("Clear All")),
		P(Attrs{}, Text(fmt.Sprintf("Total todos: %d", len(currentTodos)))),
		Ul(Attrs{}, todoItems...),
	)
}

// Goroutine example - demonstrates state updates from goroutines with cancellation
func GoroutineExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("🚀 GoroutineExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	// Background task state
	task, setTask := GoUseState(BackgroundTask{
		IsRunning:  false,
		Progress:   0,
		Status:     "Ready",
		CancelChan: nil,
	})

	// Timer state
	timer, setTimer := GoUseState(TimerState{
		IsRunning:  false,
		Seconds:    0,
		CancelChan: nil,
	})

	currentTask := task()
	currentTimer := timer()

	fmt.Printf("🚀 GoroutineExample [RENDER #%d]: State values - task={Running:%v,Progress:%d%%,Status:'%s'}, timer={Running:%v,Seconds:%d}\n",
		currentRender, currentTask.IsRunning, currentTask.Progress, currentTask.Status,
		currentTimer.IsRunning, currentTimer.Seconds)

	// Start background task
	startTask := func(this js.Value, args []js.Value) interface{} {
		if currentTask.IsRunning {
			fmt.Printf("🚀 GoroutineExample [EVENT]: START TASK clicked but already running - no state change\n")
			return nil
		}

		fmt.Printf("🚀 GoroutineExample [EVENT]: START TASK clicked - will spawn goroutine and trigger render #%d\n", currentRender+1)

		// Create cancellation channel
		cancelChan := make(chan bool, 1)

		setTask(BackgroundTask{
			IsRunning:  true,
			Progress:   0,
			Status:     "Starting...",
			CancelChan: cancelChan,
		})

		// Spawn goroutine for background task
		go func() {
			fmt.Printf("🔄 BackgroundTask [GOROUTINE]: Started goroutine\n")
			defer func() {
				fmt.Printf("🔄 BackgroundTask [GOROUTINE]: Goroutine cleanup completed\n")
			}()

			for i := 0; i <= 100; i += 10 {
				// Check for cancellation
				select {
				case <-cancelChan:
					fmt.Printf("🔄 BackgroundTask [GOROUTINE]: Received cancellation signal at %d%% - updating state\n", i)
					setTask(BackgroundTask{
						IsRunning:  false,
						Progress:   i,
						Status:     "Cancelled",
						CancelChan: nil,
					})
					return
				default:
					// Continue with work
				}

				// Simulate work
				select {
				case <-cancelChan:
					fmt.Printf("🔄 BackgroundTask [GOROUTINE]: Cancelled during work simulation at %d%% - updating state\n", i)
					setTask(BackgroundTask{
						IsRunning:  false,
						Progress:   i,
						Status:     "Cancelled",
						CancelChan: nil,
					})
					return
				case <-time.After(500 * time.Millisecond):
					// Work completed
				}

				progress := i
				status := fmt.Sprintf("Processing... %d%%", progress)
				if progress == 100 {
					status = "Completed!"
				}

				fmt.Printf("🔄 BackgroundTask [GOROUTINE]: Progress update %d%% - triggering state update (will cause re-render)\n", progress)

				// Update state from goroutine
				setTask(BackgroundTask{
					IsRunning: progress < 100,
					Progress:  progress,
					Status:    status,
					CancelChan: func() chan bool {
						if progress < 100 {
							return cancelChan
						}
						return nil
					}(),
				})

				// Check if we should stop
				if progress >= 100 {
					break
				}
			}

			fmt.Printf("🔄 BackgroundTask [GOROUTINE]: Finished normally\n")
		}()

		return nil
	}

	// Cancel background task
	cancelTask := func(this js.Value, args []js.Value) interface{} {
		if !currentTask.IsRunning || currentTask.CancelChan == nil {
			fmt.Printf("🚀 GoroutineExample [EVENT]: CANCEL TASK clicked but no task to cancel - no state change\n")
			return nil
		}

		fmt.Printf("🚀 GoroutineExample [EVENT]: CANCEL TASK clicked - sending cancellation signal\n")

		// Send cancellation signal
		select {
		case currentTask.CancelChan <- true:
			fmt.Printf("🚀 GoroutineExample [EVENT]: Cancellation signal sent successfully to background task\n")
		default:
			fmt.Printf("🚀 GoroutineExample [ERROR]: Cancellation channel was full or closed\n")
		}

		return nil
	}

	// Reset task
	resetTask := func(this js.Value, args []js.Value) interface{} {
		// Cancel any running task first
		if currentTask.IsRunning && currentTask.CancelChan != nil {
			fmt.Printf("🚀 GoroutineExample [EVENT]: RESET TASK clicked - cancelling running task first\n")
			select {
			case currentTask.CancelChan <- true:
				fmt.Printf("🚀 GoroutineExample [EVENT]: Sent cancellation signal before reset\n")
			default:
				fmt.Printf("🚀 GoroutineExample [ERROR]: Could not send cancellation signal before reset\n")
			}
		}

		fmt.Printf("🚀 GoroutineExample [EVENT]: RESET TASK - resetting to initial state (will trigger render #%d)\n", currentRender+1)
		setTask(BackgroundTask{
			IsRunning:  false,
			Progress:   0,
			Status:     "Ready",
			CancelChan: nil,
		})
		return nil
	}

	// Start/Stop timer
	toggleTimer := func(this js.Value, args []js.Value) interface{} {
		newRunning := !currentTimer.IsRunning

		fmt.Printf("🚀 GoroutineExample [EVENT]: TIMER TOGGLE clicked - state transition %v→%v (will trigger render #%d)\n",
			currentTimer.IsRunning, newRunning, currentRender+1)

		if newRunning {
			// Create cancellation channel
			cancelChan := make(chan bool, 1)

			// Start timer
			setTimer(TimerState{
				IsRunning:  true,
				Seconds:    currentTimer.Seconds,
				CancelChan: cancelChan,
			})

			// Spawn goroutine for timer
			go func() {
				fmt.Printf("⏱️ Timer [GOROUTINE]: Started timer goroutine\n")
				defer func() {
					fmt.Printf("⏱️ Timer [GOROUTINE]: Timer goroutine cleanup completed\n")
				}()

				for {
					// Wait for 1 second or cancellation
					select {
					case <-cancelChan:
						fmt.Printf("⏱️ Timer [GOROUTINE]: Received cancellation signal\n")
						return
					case <-time.After(1 * time.Second):
						// Timer tick
					}

					// Get current state and check if we should continue
					currentState := timer()
					if !currentState.IsRunning {
						fmt.Printf("⏱️ Timer [GOROUTINE]: Stopping (timer turned off in state)\n")
						return
					}

					// Check for cancellation again after state check
					select {
					case <-cancelChan:
						fmt.Printf("⏱️ Timer [GOROUTINE]: Cancelled after state check\n")
						return
					default:
					}

					newSeconds := currentState.Seconds + 1
					fmt.Printf("⏱️ Timer [GOROUTINE]: Tick %d seconds - triggering state update (will cause re-render)\n", newSeconds)

					// Update state from goroutine
					setTimer(TimerState{
						IsRunning:  true,
						Seconds:    newSeconds,
						CancelChan: cancelChan,
					})
				}
			}()
		} else {
			// Stop timer - send cancellation signal first
			if currentTimer.CancelChan != nil {
				fmt.Printf("🚀 GoroutineExample [EVENT]: Sending cancellation signal to timer\n")
				select {
				case currentTimer.CancelChan <- true:
					fmt.Printf("🚀 GoroutineExample [EVENT]: Timer cancellation signal sent\n")
				default:
					fmt.Printf("🚀 GoroutineExample [ERROR]: Timer cancellation channel was full\n")
				}
			}

			// Update state
			setTimer(TimerState{
				IsRunning:  false,
				Seconds:    currentTimer.Seconds,
				CancelChan: nil,
			})
		}

		return nil
	}

	// Reset timer
	resetTimer := func(this js.Value, args []js.Value) interface{} {
		// Cancel any running timer first
		if currentTimer.IsRunning && currentTimer.CancelChan != nil {
			fmt.Printf("🚀 GoroutineExample [EVENT]: RESET TIMER clicked - cancelling running timer first\n")
			select {
			case currentTimer.CancelChan <- true:
				fmt.Printf("🚀 GoroutineExample [EVENT]: Sent timer cancellation signal before reset\n")
			default:
				fmt.Printf("🚀 GoroutineExample [ERROR]: Could not send timer cancellation signal before reset\n")
			}
		}

		fmt.Printf("🚀 GoroutineExample [EVENT]: RESET TIMER - resetting to 0 seconds (will trigger render #%d)\n", currentRender+1)
		setTimer(TimerState{
			IsRunning:  false,
			Seconds:    0,
			CancelChan: nil,
		})
		return nil
	}

	// Cleanup all goroutines
	cleanupAll := func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("🚀 GoroutineExample [EVENT]: CLEANUP ALL clicked - cancelling all goroutines\n")

		// Cancel background task
		if currentTask.IsRunning && currentTask.CancelChan != nil {
			select {
			case currentTask.CancelChan <- true:
				fmt.Printf("🧹 GoroutineExample [CLEANUP]: Background task cancellation signal sent\n")
			default:
				fmt.Printf("🧹 GoroutineExample [ERROR]: Background task channel was full\n")
			}
		}

		// Cancel timer
		if currentTimer.IsRunning && currentTimer.CancelChan != nil {
			select {
			case currentTimer.CancelChan <- true:
				fmt.Printf("🧹 GoroutineExample [CLEANUP]: Timer cancellation signal sent\n")
			default:
				fmt.Printf("🧹 GoroutineExample [ERROR]: Timer channel was full\n")
			}
		}

		// Reset both states
		setTask(BackgroundTask{IsRunning: false, Progress: 0, Status: "Ready", CancelChan: nil})
		setTimer(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})

		fmt.Printf("🧹 GoroutineExample [CLEANUP]: All goroutines cleanup initiated (will trigger render #%d)\n", currentRender+1)
		return nil
	}

	fmt.Printf("🚀 GoroutineExample [RENDER #%d]: Generating DOM with task=%d%% and timer=%ds\n",
		currentRender, currentTask.Progress, currentTimer.Seconds)

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text(fmt.Sprintf("Goroutine Example with Cancellation (Render #%d)", currentRender))),
		P(Attrs{}, Text("This demonstrates state updates from goroutines with proper cancellation and cleanup.")),

		// Global Cleanup Section
		Div(Attrs{
			"style": "border: 2px solid #ff6b6b; padding: 8px; margin: 8px 0; background-color: #ffe0e0;",
		},
			H4(Attrs{}, Text("🧹 Global Controls")),
			Button(Attrs{
				"onclick": js.FuncOf(cleanupAll),
				"style":   "background-color: #ff6b6b; color: white; font-weight: bold;",
			}, Text("Cancel All & Cleanup")),
		),

		// Background Task Section
		Div(Attrs{
			"style": "border: 1px solid #ddd; padding: 8px; margin: 8px 0;",
		},
			H4(Attrs{}, Text("Background Task")),
			P(Attrs{}, Text(fmt.Sprintf("Status: %s", currentTask.Status))),
			P(Attrs{}, Text(fmt.Sprintf("Progress: %d%%", currentTask.Progress))),
			Button(Attrs{
				"onclick":  js.FuncOf(startTask),
				"disabled": currentTask.IsRunning,
				"style": func() string {
					if currentTask.IsRunning {
						return "background-color: #ccc;"
					}
					return "background-color: #4CAF50; color: white;"
				}(),
			}, Text(func() string {
				if currentTask.IsRunning {
					return "Running..."
				}
				return "Start Task"
			}())),
			Text(" "),
			Button(Attrs{
				"onclick":  js.FuncOf(cancelTask),
				"disabled": !currentTask.IsRunning,
				"style": func() string {
					if !currentTask.IsRunning {
						return "background-color: #ccc;"
					}
					return "background-color: #ff9800; color: white;"
				}(),
			}, Text("Cancel")),
			Text(" "),
			Button(Attrs{
				"onclick": js.FuncOf(resetTask),
			}, Text("Reset")),
		),

		// Timer Section
		Div(Attrs{
			"style": "border: 1px solid #ddd; padding: 8px; margin: 8px 0;",
		},
			H4(Attrs{}, Text("Continuous Timer")),
			P(Attrs{}, Text(fmt.Sprintf("Time: %d seconds", currentTimer.Seconds))),
			P(Attrs{}, Text(fmt.Sprintf("Status: %s", func() string {
				if currentTimer.IsRunning {
					return "Running"
				}
				return "Stopped"
			}()))),
			Button(Attrs{
				"onclick": js.FuncOf(toggleTimer),
				"style": func() string {
					if currentTimer.IsRunning {
						return "background-color: #f44336; color: white;"
					}
					return "background-color: #2196F3; color: white;"
				}(),
			}, Text(func() string {
				if currentTimer.IsRunning {
					return "Stop Timer"
				}
				return "Start Timer"
			}())),
			Text(" "),
			Button(Attrs{
				"onclick": js.FuncOf(resetTimer),
			}, Text("Reset Timer")),
		),
	)
}

// Fetch example - demonstrates GoUseFetch hook with API calls
func FetchExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("🌐 FetchExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	// URL state for the fetch request
	url, setUrl := GoUseState("https://jsonplaceholder.typicode.com/posts/1")
	currentUrl := url()

	// Use GoUseFetch hook for data fetching
	getFetchState, refetch := GoUseFetch(currentUrl)
	fetchState := getFetchState()

	fmt.Printf("🌐 FetchExample [RENDER #%d]: State values - url='%s', loading=%v, hasError=%v, hasData=%v\n",
		currentRender, currentUrl, fetchState.Loading, fetchState.Error != "", fetchState.Data != nil)

	// Log fetch state details
	if fetchState.Error != "" {
		fmt.Printf("🌐 FetchExample [RENDER #%d]: Error state - %s\n", currentRender, fetchState.Error)
	}
	if fetchState.Data != nil {
		fmt.Printf("🌐 FetchExample [RENDER #%d]: Data received - %+v\n", currentRender, fetchState.Data)
	}

	// Manual fetch button handler
	manualFetch := func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("🌐 FetchExample [EVENT]: FETCH clicked - triggering manual fetch from '%s' (will trigger render #%d)\n",
			currentUrl, currentRender+1)
		refetch()
		return nil
	}

	// URL input handler
	handleUrlChange := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newUrl := args[0].Get("target").Get("value").String()
			fmt.Printf("🌐 FetchExample [EVENT]: URL changed - transition '%s'→'%s' (will trigger render #%d)\n",
				currentUrl, newUrl, currentRender+1)
			setUrl(newUrl)
		}
		return nil
	}

	// Preset URL buttons
	setJsonPlaceholder := func(this js.Value, args []js.Value) interface{} {
		newUrl := "https://jsonplaceholder.typicode.com/posts/1"
		fmt.Printf("🌐 FetchExample [EVENT]: JSON PLACEHOLDER clicked - setting URL to '%s' (will trigger render #%d)\n",
			newUrl, currentRender+1)
		setUrl(newUrl)
		return nil
	}

	setHttpBin := func(this js.Value, args []js.Value) interface{} {
		newUrl := "https://httpbin.org/json"
		fmt.Printf("🌐 FetchExample [EVENT]: HTTPBIN clicked - setting URL to '%s' (will trigger render #%d)\n",
			newUrl, currentRender+1)
		setUrl(newUrl)
		return nil
	}

	setRandomUser := func(this js.Value, args []js.Value) interface{} {
		newUrl := "https://randomuser.me/api/"
		fmt.Printf("🌐 FetchExample [EVENT]: RANDOM USER clicked - setting URL to '%s' (will trigger render #%d)\n",
			newUrl, currentRender+1)
		setUrl(newUrl)
		return nil
	}

	fmt.Printf("🌐 FetchExample [RENDER #%d]: Generating DOM with fetch state - loading=%v\n", currentRender, fetchState.Loading)

	// Render response data
	var responseContent interface{}
	if fetchState.Loading {
		responseContent = P(Attrs{
			"style": "color: #63b3ed; font-style: italic;",
		}, Text("🔄 Loading..."))
	} else if fetchState.Error != "" {
		responseContent = P(Attrs{
			"style": "color: #f56565; font-weight: bold;",
		}, Text(fmt.Sprintf("❌ Error: %s", fetchState.Error)))
	} else if fetchState.Data != nil {
		responseContent = Div(Attrs{},
			P(Attrs{
				"style": "color: #48bb78; font-weight: bold; margin-bottom: 8px;",
			}, Text("✅ Success! Data received:")),
			Pre(Attrs{
				"style": "background-color: #2d3748; color: #e2e8f0; padding: 10px; border: 1px solid #4a5568; border-radius: 4px; overflow-x: auto; font-size: 12px; font-family: 'Courier New', monospace;",
			}, Text(fmt.Sprintf("%+v", fetchState.Data))),
		)
	} else {
		responseContent = P(Attrs{
			"style": "color: #a0aec0; font-style: italic;",
		}, Text("🔗 Click 'Fetch Data' to make a request"))
	}

	return Div(Attrs{
		"style": "border: 1px solid #4a5568; background-color: #2d3748; padding: 10px; margin: 10px; border-radius: 8px;",
	},
		H3(Attrs{
			"style": "color: #f7fafc; margin-bottom: 8px;",
		}, Text(fmt.Sprintf("🌐 Fetch Example (Render #%d)", currentRender))),
		P(Attrs{
			"style": "color: #e2e8f0; margin-bottom: 12px;",
		}, Text("This demonstrates the GoUseFetch hook for API calls.")),

		// URL Input Section
		Div(Attrs{
			"style": "border: 1px solid #4a5568; background-color: #1a202c; padding: 8px; margin: 8px 0; border-radius: 6px;",
		},
			H4(Attrs{
				"style": "color: #63b3ed; margin-bottom: 6px;",
			}, Text("📡 API Endpoint")),
			P(Attrs{
				"style": "color: #e2e8f0; margin-bottom: 4px;",
			}, Text("URL:")),
			Input(Attrs{
				"type":    "url",
				"value":   currentUrl,
				"oninput": js.FuncOf(handleUrlChange),
				"style":   "background-color: #2d3748; color: #f7fafc; border: 1px solid #4a5568; padding: 6px; border-radius: 4px; width: 100%; max-width: 500px;",
			}),
			Div(Attrs{
				"style": "margin: 8px 0;",
			},
				Span(Attrs{
					"style": "color: #e2e8f0;",
				}, Text("Quick presets: ")),
				Button(Attrs{
					"onclick": js.FuncOf(setJsonPlaceholder),
					"style":   "background-color: #4a5568; color: #f7fafc; border: 1px solid #63b3ed; padding: 4px 8px; margin: 2px; font-size: 12px; border-radius: 4px; cursor: pointer;",
				}, Text("JSONPlaceholder")),
				Text(" "),
				Button(Attrs{
					"onclick": js.FuncOf(setHttpBin),
					"style":   "background-color: #4a5568; color: #f7fafc; border: 1px solid #63b3ed; padding: 4px 8px; margin: 2px; font-size: 12px; border-radius: 4px; cursor: pointer;",
				}, Text("HTTPBin")),
				Text(" "),
				Button(Attrs{
					"onclick": js.FuncOf(setRandomUser),
					"style":   "background-color: #4a5568; color: #f7fafc; border: 1px solid #63b3ed; padding: 4px 8px; margin: 2px; font-size: 12px; border-radius: 4px; cursor: pointer;",
				}, Text("Random User")),
			),
		),

		// Fetch Controls Section
		Div(Attrs{
			"style": "border: 1px solid #4a5568; background-color: #1a202c; padding: 8px; margin: 8px 0; border-radius: 6px;",
		},
			H4(Attrs{
				"style": "color: #63b3ed; margin-bottom: 6px;",
			}, Text("🚀 Fetch Controls")),
			Button(Attrs{
				"onclick": js.FuncOf(manualFetch),
				"style": func() string {
					if fetchState.Loading {
						return "background-color: #4a5568; color: #a0aec0; cursor: not-allowed; padding: 8px 16px; border-radius: 4px; border: 1px solid #4a5568;"
					}
					return "background-color: #3182ce; color: #f7fafc; font-weight: bold; padding: 8px 16px; border-radius: 4px; border: 1px solid #63b3ed; cursor: pointer;"
				}(),
				"disabled": fetchState.Loading,
			}, Text(func() string {
				if fetchState.Loading {
					return "🔄 Fetching..."
				}
				return "🚀 Fetch Data"
			}())),
			P(Attrs{
				"style": "font-size: 12px; color: #a0aec0; margin: 4px 0;",
			}, Text("Note: GoUseFetch automatically fetches when URL changes. Use button for manual refetch.")),
		),

		// Response Section
		Div(Attrs{
			"style": "border: 1px solid #4a5568; background-color: #1a202c; padding: 8px; margin: 8px 0; border-radius: 6px;",
		},
			H4(Attrs{
				"style": "color: #63b3ed; margin-bottom: 6px;",
			}, Text("📋 Response")),
			responseContent,
		),
	)
}

// Hook Order Validation Test - demonstrates proper hook order validation
func HookOrderTestExample(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("🔍 HookOrderTestExample [RENDER #%d|Global #%d]: Component rendered\n", currentRender, globalRender)

	// Test mode state - controls whether we violate hook order
	testMode, setTestMode := GoUseState("normal")
	currentMode := testMode()

	fmt.Printf("🔍 HookOrderTestExample [RENDER #%d]: Test mode = '%s'\n", currentRender, currentMode)

	// Normal hooks that should always be called
	normalCount, setNormalCount := GoUseState(0)
	currentNormalCount := normalCount()

	// Conditional hook violation test - this will trigger hook order errors
	var conditionalCount func() int
	var setConditionalCount func(int)

	if currentMode == "violate" {
		// This violates hook order rules - hooks should not be called conditionally
		fmt.Printf("🚨 HookOrderTestExample [RENDER #%d]: About to violate hook order by calling conditional hook\n", currentRender)
		conditionalCount, setConditionalCount = GoUseState(100)
		fmt.Printf("🚨 HookOrderTestExample [RENDER #%d]: Conditional hook called - this should trigger validation error\n", currentRender)
	} else {
		// Provide dummy functions when not violating
		conditionalCount = func() int { return -1 }
		setConditionalCount = func(int) {}
	}

	// Another normal hook that should always be called
	message, setMessage := GoUseState("Hook order is normal")
	currentMessage := message()

	// Effect hook for testing effect order validation
	GoUseEffect(func() {
		fmt.Printf("🔍 HookOrderTestExample [EFFECT]: Effect ran for render #%d in mode '%s'\n", currentRender, currentMode)
	}, []interface{}{currentRender, currentMode})

	// Conditional effect - this will also violate hook order
	if currentMode == "violate" {
		fmt.Printf("🚨 HookOrderTestExample [RENDER #%d]: About to violate hook order with conditional effect\n", currentRender)
		GoUseEffect(func() {
			fmt.Printf("🚨 HookOrderTestExample [CONDITIONAL_EFFECT]: This effect should trigger validation error\n")
		}, []interface{}{})
	}

	// Event handlers
	toggleMode := func(this js.Value, args []js.Value) interface{} {
		newMode := "normal"
		if currentMode == "normal" {
			newMode = "violate"
		}
		fmt.Printf("🔍 HookOrderTestExample [EVENT]: Switching mode %s→%s (will trigger render #%d)\n",
			currentMode, newMode, currentRender+1)
		setTestMode(newMode)

		// Update message based on mode
		if newMode == "violate" {
			setMessage("⚠️ Hook order violation mode - check console for errors!")
		} else {
			setMessage("✅ Hook order is normal")
		}
		return nil
	}

	incrementNormal := func(this js.Value, args []js.Value) interface{} {
		newCount := currentNormalCount + 1
		fmt.Printf("🔍 HookOrderTestExample [EVENT]: Normal count %d→%d (will trigger render #%d)\n",
			currentNormalCount, newCount, currentRender+1)
		setNormalCount(newCount)
		return nil
	}

	incrementConditional := func(this js.Value, args []js.Value) interface{} {
		if currentMode == "violate" {
			currentConditional := conditionalCount()
			newCount := currentConditional + 1
			fmt.Printf("🔍 HookOrderTestExample [EVENT]: Conditional count %d→%d (will trigger render #%d)\n",
				currentConditional, newCount, currentRender+1)
			setConditionalCount(newCount)
		} else {
			fmt.Printf("🔍 HookOrderTestExample [EVENT]: Cannot increment conditional count in normal mode\n")
		}
		return nil
	}

	fmt.Printf("🔍 HookOrderTestExample [RENDER #%d]: Generating DOM with mode='%s', normalCount=%d\n",
		currentRender, currentMode, currentNormalCount)

	return Div(Attrs{
		"style": "border: 1px solid #ccc; padding: 10px; margin: 10px;",
	},
		H3(Attrs{}, Text(fmt.Sprintf("🔍 Hook Order Validation Test (Render #%d)", currentRender))),
		P(Attrs{}, Text("This component tests hook order validation by conditionally calling hooks.")),
		P(Attrs{}, Text("⚠️ WARNING: 'Violate' mode will intentionally break hook order rules!")),

		Div(Attrs{
			"style": "margin: 10px 0; padding: 10px; border: 1px solid #ddd;",
		},
			P(Attrs{}, Text(fmt.Sprintf("Current Mode: %s", currentMode))),
			P(Attrs{}, Text(fmt.Sprintf("Message: %s", currentMessage))),
			P(Attrs{}, Text(fmt.Sprintf("Normal Count: %d", currentNormalCount))),
			P(Attrs{}, Text(fmt.Sprintf("Conditional Count: %s", func() string {
				if currentMode == "violate" {
					return fmt.Sprintf("%d", conditionalCount())
				}
				return "N/A (not in violation mode)"
			}()))),
		),

		Div(Attrs{
			"style": "margin: 10px 0;",
		},
			Button(Attrs{
				"onclick": js.FuncOf(toggleMode),
				"style": func() string {
					if currentMode == "violate" {
						return "background-color: green; color: white;"
					}
					return "background-color: red; color: white;"
				}(),
			}, Text(func() string {
				if currentMode == "violate" {
					return "Switch to Normal Mode"
				}
				return "Switch to Violation Mode"
			}())),
			Text(" "),
			Button(Attrs{
				"onclick": js.FuncOf(incrementNormal),
			}, Text("Increment Normal")),
			Text(" "),
			Button(Attrs{
				"onclick":  js.FuncOf(incrementConditional),
				"disabled": currentMode != "violate",
			}, Text("Increment Conditional")),
		),

		P(Attrs{}, Text("💡 Check the browser console for hook order validation messages when switching to violation mode.")),
	)
}

// Main application component
func SimpleStateExamplesApp(props Attrs) *Element {
	// Track render count for this component
	renderCount, setRenderCount := GoUseState(0)
	currentRender := renderCount() + 1
	setRenderCount(currentRender)

	globalRender := getNextRenderID()

	fmt.Printf("🚀 SimpleStateExamplesApp [RENDER #%d|Global #%d]: Main app component rendered\n", currentRender, globalRender)
	fmt.Printf("🚀 SimpleStateExamplesApp [RENDER #%d]: Rendering all child components\n", currentRender)

	return Div(Attrs{},
		H1(Attrs{}, Text(fmt.Sprintf("Simple GoUseState Examples (App Render #%d | Global #%d)", currentRender, globalRender))),
		P(Attrs{}, Text("These examples demonstrate various use cases of GoUseState with minimal styling.")),

		FetchExample(nil),
		HookOrderTestExample(nil),
		CounterExample(nil),
		TextInputExample(nil),
		ToggleExample(nil),
		PersonFormExample(nil),
		TodoListExample(nil),
		GoroutineExample(nil),
	)
}

// GetSimpleStateExamplesApp returns the main app component function
func GetSimpleStateExamplesApp() func(Attrs) *Element {
	// Create a wrapper component function
	appPage := func(props Attrs) *Element {
		fmt.Printf("🎯 SimpleStateExamplesDemo [INIT]: Wrapper component called - about to render main app\n")
		return SimpleStateExamplesApp(nil)
	}

	return appPage
}

// SimpleStateExamplesDemo initializes and starts the simple state examples
func SimpleStateExamplesDemo() {
	fmt.Printf("🎯 SimpleStateExamplesDemo [INIT]: Starting initialization...\n")

	appPage := GetSimpleStateExamplesApp()

	fmt.Printf("🎯 SimpleStateExamplesDemo [INIT]: Looking for DOM element with id 'app'\n")

	// Use RenderTo which will handle state restoration automatically
	fmt.Printf("✅ SimpleStateExamplesDemo [INIT]: Found DOM container, creating element...\n")
	fmt.Printf("✅ SimpleStateExamplesDemo [INIT]: Element created, rendering to DOM...\n")
	RenderTo("#app", appPage)

	fmt.Printf("🎉 SimpleStateExamplesDemo [SUCCESS]: Simple GoUseState Examples are now running!\n")
	fmt.Printf("📊 SimpleStateExamplesDemo [INFO]: Examples included:\n")
	fmt.Printf("   - 🌐 Fetch Example (GoUseFetch hook with API calls)\n")
	fmt.Printf("   - 🔍 Hook Order Validation Test (demonstrates hook order validation)\n")
	fmt.Printf("   - 🔢 Counter (number state)\n")
	fmt.Printf("   - 📝 Text Input (string state)\n")
	fmt.Printf("   - 🔘 Toggle (boolean state)\n")
	fmt.Printf("   - 👤 Person Form (struct state)\n")
	fmt.Printf("   - 📋 Todo List (array/slice state)\n")
	fmt.Printf("   - 🚀 Goroutine (async state updates with cancellation)\n")
	fmt.Printf("🔍 SimpleStateExamplesDemo [DEBUG]: Watch console for detailed render tracking and state change logs!\n")
	fmt.Printf("⚠️  SimpleStateExamplesDemo [HOOK_VALIDATION]: Hook order validation is now active - violations will be logged!\n")
	fmt.Printf("📈 SimpleStateExamplesDemo [PERFORMANCE]: Global render counter started - track re-renders across all components\n")
}
