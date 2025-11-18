//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

type BackgroundTask struct {
	IsRunning  bool
	Progress   int
	Status     string
	CancelChan chan bool
}

type TimerState struct {
	IsRunning  bool
	Seconds    int
	CancelChan chan bool
}

func GoroutineExample(_ Attrs) *Element {
	task, setTask := hooks.UseState(BackgroundTask{
		IsRunning:  false,
		Progress:   0,
		Status:     "Ready",
		CancelChan: nil,
	})

	timer, setTimer := hooks.UseState(TimerState{
		IsRunning:  false,
		Seconds:    0,
		CancelChan: nil,
	})

	currentTask := task()
	currentTimer := timer()

	startTask := func(this js.Value, args []js.Value) interface{} {
		if currentTask.IsRunning {
			return nil
		}

		cancelChan := make(chan bool, 1)

		setTask(BackgroundTask{
			IsRunning:  true,
			Progress:   0,
			Status:     "Starting...",
			CancelChan: cancelChan,
		})

		go func() {
			for i := 0; i <= 100; i += 10 {
				select {
				case <-cancelChan:
					setTask(BackgroundTask{
						IsRunning:  false,
						Progress:   i,
						Status:     "Cancelled",
						CancelChan: nil,
					})
					return
				default:
				}

				select {
				case <-cancelChan:
					setTask(BackgroundTask{
						IsRunning:  false,
						Progress:   i,
						Status:     "Cancelled",
						CancelChan: nil,
					})
					return
				case <-time.After(500 * time.Millisecond):
				}

				progress := i
				status := fmt.Sprintf("Processing... %d%%", progress)
				if progress == 100 {
					status = "Completed!"
				}

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

				if progress >= 100 {
					break
				}
			}
		}()

		return nil
	}

	cancelTask := func(this js.Value, args []js.Value) interface{} {
		if !currentTask.IsRunning || currentTask.CancelChan == nil {
			return nil
		}

		select {
		case currentTask.CancelChan <- true:
		default:
		}

		return nil
	}

	resetTask := func(this js.Value, args []js.Value) interface{} {
		if currentTask.IsRunning && currentTask.CancelChan != nil {
			select {
			case currentTask.CancelChan <- true:
			default:
			}
		}

		setTask(BackgroundTask{
			IsRunning:  false,
			Progress:   0,
			Status:     "Ready",
			CancelChan: nil,
		})
		return nil
	}

	toggleTimer := func(this js.Value, args []js.Value) interface{} {
		newRunning := !currentTimer.IsRunning

		if newRunning {
			cancelChan := make(chan bool, 1)

			setTimer(TimerState{
				IsRunning:  true,
				Seconds:    currentTimer.Seconds,
				CancelChan: cancelChan,
			})

			go func() {
				for {
					select {
					case <-cancelChan:
						return
					case <-time.After(1 * time.Second):
					}

					currentState := timer()
					if !currentState.IsRunning {
						return
					}

					select {
					case <-cancelChan:
						return
					default:
					}

					newSeconds := currentState.Seconds + 1

					setTimer(TimerState{
						IsRunning:  true,
						Seconds:    newSeconds,
						CancelChan: cancelChan,
					})
				}
			}()
		} else {
			if currentTimer.CancelChan != nil {
				select {
				case currentTimer.CancelChan <- true:
				default:
				}
			}

			setTimer(TimerState{
				IsRunning:  false,
				Seconds:    currentTimer.Seconds,
				CancelChan: nil,
			})
		}

		return nil
	}

	resetTimer := func(this js.Value, args []js.Value) interface{} {
		if currentTimer.IsRunning && currentTimer.CancelChan != nil {
			select {
			case currentTimer.CancelChan <- true:
			default:
			}
		}

		setTimer(TimerState{
			IsRunning:  false,
			Seconds:    0,
			CancelChan: nil,
		})
		return nil
	}

	cleanupAll := func(this js.Value, args []js.Value) interface{} {
		if currentTask.IsRunning && currentTask.CancelChan != nil {
			select {
			case currentTask.CancelChan <- true:
			default:
			}
		}

		if currentTimer.IsRunning && currentTimer.CancelChan != nil {
			select {
			case currentTimer.CancelChan <- true:
			default:
			}
		}

		setTask(BackgroundTask{IsRunning: false, Progress: 0, Status: "Ready", CancelChan: nil})
		setTimer(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})

		return nil
	}

	return dom.Div(Attrs{
		"class": "max-w-4xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold mb-6 text-gray-800",
		}, dom.Text("Goroutine Example")),

		dom.P(Attrs{
			"class": "mb-6 text-gray-600",
		}, dom.Text("Demonstrates state updates from goroutines with proper cancellation and cleanup.")),

		// Global Cleanup Section
		dom.Div(Attrs{
			"class": "border-2 border-red-400 p-4 mb-6 bg-red-50 rounded-lg",
		},
			dom.H3(Attrs{
				"class": "text-lg font-semibold mb-3 text-gray-800",
			}, dom.Text("🧹 Global Controls")),
			dom.Button(Attrs{
				"onclick": js.FuncOf(cleanupAll),
				"class":   "px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors font-semibold",
			}, dom.Text("Cancel All & Cleanup")),
		),

		// Background Task Section
		dom.Div(Attrs{
			"class": "border border-gray-300 p-4 mb-4 rounded-lg",
		},
			dom.H3(Attrs{
				"class": "text-lg font-semibold mb-3 text-gray-800",
			}, dom.Text("Background Task")),
			dom.P(Attrs{
				"class": "mb-2 text-gray-700",
			}, dom.Text(fmt.Sprintf("Status: %s", currentTask.Status))),
			dom.P(Attrs{
				"class": "mb-3 text-gray-700",
			}, dom.Text(fmt.Sprintf("Progress: %d%%", currentTask.Progress))),

			dom.Div(Attrs{"class": "w-full bg-gray-200 rounded-full h-4 mb-4"},
				dom.Div(Attrs{
					"class": "bg-blue-500 h-4 rounded-full transition-all duration-300",
					"style": fmt.Sprintf("width: %d%%", currentTask.Progress),
				}),
			),

			dom.Div(Attrs{"class": "flex gap-2"},
				dom.Button(Attrs{
					"onclick": js.FuncOf(startTask),
					"disabled": func() string {
						if currentTask.IsRunning {
							return "true"
						}
						return ""
					}(),
					"class": func() string {
						if currentTask.IsRunning {
							return "px-4 py-2 bg-gray-300 text-gray-500 rounded-lg cursor-not-allowed"
						}
						return "px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors"
					}(),
				}, dom.Text("Start Task")),
				dom.Button(Attrs{
					"onclick": js.FuncOf(cancelTask),
					"class":   "px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition-colors",
				}, dom.Text("Cancel Task")),
				dom.Button(Attrs{
					"onclick": js.FuncOf(resetTask),
					"class":   "px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors",
				}, dom.Text("Reset Task")),
			),
		),

		// Timer Section
		dom.Div(Attrs{
			"class": "border border-gray-300 p-4 rounded-lg",
		},
			dom.H3(Attrs{
				"class": "text-lg font-semibold mb-3 text-gray-800",
			}, dom.Text("Timer")),
			dom.P(Attrs{
				"class": "mb-3 text-gray-700 text-2xl font-mono",
			}, dom.Text(fmt.Sprintf("%d seconds", currentTimer.Seconds))),

			dom.Div(Attrs{"class": "flex gap-2"},
				dom.Button(Attrs{
					"onclick": js.FuncOf(toggleTimer),
					"class": func() string {
						if currentTimer.IsRunning {
							return "px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors"
						}
						return "px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors"
					}(),
				}, func() interface{} {
					if currentTimer.IsRunning {
						return dom.Text("Stop Timer")
					}
					return dom.Text("Start Timer")
				}()),
				dom.Button(Attrs{
					"onclick": js.FuncOf(resetTimer),
					"class":   "px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors",
				}, dom.Text("Reset Timer")),
			),
		),
	)
}

func main() {
	container := js.Global().Get("document").Call("getElementById", "app")
	element := dom.CreateElement(GoroutineExample, nil)
	render.ToElement(element, container)
	select {}
}
