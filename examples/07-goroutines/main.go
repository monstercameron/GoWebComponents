//go:build js && wasm

package main

import (
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = dom.Element

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

	startTask := hooks.GoUseFunc(func(event dom.GoEvent) {
		if currentTask.IsRunning {
			return
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
	})

	cancelTask := hooks.GoUseFunc(func(event dom.GoEvent) {
		if !currentTask.IsRunning || currentTask.CancelChan == nil {
			return
		}

		select {
		case currentTask.CancelChan <- true:
		default:
		}
	})

	resetTask := hooks.GoUseFunc(func(event dom.GoEvent) {
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
	})

	toggleTimer := hooks.GoUseFunc(func(event dom.GoEvent) {
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
	})

	resetTimer := hooks.GoUseFunc(func(event dom.GoEvent) {
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
	})

	cleanupAll := hooks.GoUseFunc(func(event dom.GoEvent) {
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
	})

	return dom.Div(Attrs{
		"class": "min-h-screen bg-[#0a0a0a] text-white p-8",
	},
		dom.Div(Attrs{
			"class": "max-w-4xl mx-auto",
		},
			dom.H2(Attrs{
				"class": "text-3xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, dom.Text("Goroutine Example")),

			dom.P(Attrs{
				"class": "mb-8 text-gray-400",
			}, dom.Text("Demonstrates state updates from goroutines with proper cancellation and cleanup.")),

			// Global Cleanup Section
			dom.Div(Attrs{
				"class": "border border-red-500/30 p-6 mb-8 bg-red-500/10 rounded-xl backdrop-blur-sm",
			},
				dom.H3(Attrs{
					"class": "text-lg font-semibold mb-4 text-red-400",
				}, dom.Text("⚠️ Global Controls")),
				dom.Button(Attrs{
					"onclick": cleanupAll,
					"class":   "px-6 py-3 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors font-semibold shadow-lg shadow-red-500/20",
				}, dom.Text("Cancel All & Cleanup")),
			),

			// Background Task Section
			dom.Div(Attrs{
				"class": "bg-white/5 border border-white/10 p-6 mb-6 rounded-xl backdrop-blur-sm",
			},
				dom.H3(Attrs{
					"class": "text-xl font-semibold mb-4 text-white",
				}, dom.Text("Background Task")),
				dom.P(Attrs{
					"class": "mb-2 text-gray-400",
				}, dom.Text(fmt.Sprintf("Status: %s", currentTask.Status))),
				dom.P(Attrs{
					"class": "mb-4 text-gray-400",
				}, dom.Text(fmt.Sprintf("Progress: %d%%", currentTask.Progress))),

				dom.Div(Attrs{"class": "w-full bg-black/30 rounded-full h-4 mb-6 overflow-hidden"},
					dom.Div(Attrs{
						"class": "bg-gradient-to-r from-blue-500 to-purple-600 h-4 rounded-full transition-all duration-300",
						"style": fmt.Sprintf("width: %d%%", currentTask.Progress),
					}),
				),

				dom.Div(Attrs{"class": "flex gap-3"},
					dom.Button(Attrs{
						"onclick":  startTask,
						"disabled": currentTask.IsRunning,
						"class": func() string {
							if currentTask.IsRunning {
								return "px-4 py-2 bg-white/5 text-gray-500 rounded-lg cursor-not-allowed border border-white/5"
							}
							return "px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors shadow-lg shadow-green-500/20"
						}(),
					}, dom.Text("Start Task")),
					dom.Button(Attrs{
						"onclick": cancelTask,
						"class":   "px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition-colors shadow-lg shadow-yellow-500/20",
					}, dom.Text("Cancel Task")),
					dom.Button(Attrs{
						"onclick": resetTask,
						"class":   "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10",
					}, dom.Text("Reset Task")),
				),
			),

			// Timer Section
			dom.Div(Attrs{
				"class": "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm",
			},
				dom.H3(Attrs{
					"class": "text-xl font-semibold mb-4 text-white",
				}, dom.Text("Timer")),
				dom.P(Attrs{
					"class": "mb-6 text-white text-4xl font-mono font-bold tracking-wider",
				}, dom.Text(fmt.Sprintf("%02d:%02d", currentTimer.Seconds/60, currentTimer.Seconds%60))),

				dom.Div(Attrs{"class": "flex gap-3"},
					dom.Button(Attrs{
						"onclick": toggleTimer,
						"class": func() string {
							if currentTimer.IsRunning {
								return "px-6 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors shadow-lg shadow-red-500/20"
							}
							return "px-6 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors shadow-lg shadow-green-500/20"
						}(),
					}, dom.Text(func() string {
						if currentTimer.IsRunning {
							return "Stop Timer"
						}
						return "Start Timer"
					}())),
					dom.Button(Attrs{
						"onclick": resetTimer,
						"class":   "px-6 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10",
					}, dom.Text("Reset Timer")),
				),
			),
		),
	)
}

func main() {
	render.To(dom.CreateElement(GoroutineExample, nil), "body")
	select {}
}
