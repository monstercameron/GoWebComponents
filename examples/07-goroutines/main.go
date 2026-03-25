//go:build js && wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type BackgroundTask struct {
	Progress int
	Status   string
}

type TimerState struct {
	IsRunning  bool
	Seconds    int
	CancelChan chan bool
}

type StreamState struct {
	IsRunning bool
	Source    <-chan string
	StopChan  chan struct{}
}

func GoroutineExample() ui.Node {
	parseTaskProgress := ui.UseState(0)
	parseTaskRunner := ui.UseTask(func(parseCtx context.Context) (BackgroundTask, error) {
		for parseI := 0; parseI <= 100; parseI += 10 {
			select {
			case <-parseCtx.Done():
				return BackgroundTask{Progress: parseI, Status: "Cancelled"}, parseCtx.Err()
			default:
			}

			select {
			case <-parseCtx.Done():
				return BackgroundTask{Progress: parseI, Status: "Cancelled"}, parseCtx.Err()
			case <-time.After(500 * time.Millisecond):
			}

			parseProgress := parseI
			parseTaskProgress.Set(parseProgress)
		}

		return BackgroundTask{Progress: 100, Status: "Completed!"}, nil
	})
	parseTimer := ui.UseState(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})
	parseStream := ui.UseState(StreamState{})
	parseStreamValue := ui.UseChannel(parseStream.Get().Source)

	parseStartTask := ui.UseEvent(func() {
		parseCurrentTask := parseTaskRunner.Get()
		if parseCurrentTask.Running {
			return
		}
		parseTaskProgress.Set(0)
		parseTaskRunner.Start()
	})

	parseCancelTask := ui.UseEvent(func() {
		parseCurrentTask2 := parseTaskRunner.Get()
		if !parseCurrentTask2.Running {
			return
		}
		parseTaskRunner.Cancel()
	})

	resetTask := ui.UseEvent(func() {
		parseTaskRunner.Cancel()
		parseTaskProgress.Set(0)
	})

	parseToggleTimer := ui.UseEvent(func() {
		parseCurrentTimer := parseTimer.Get()
		if !parseCurrentTimer.IsRunning {
			parseCancelChan := make(chan bool, 1)
			parseTimer.Set(TimerState{IsRunning: true, Seconds: parseCurrentTimer.Seconds, CancelChan: parseCancelChan})

			go func() {
				for {
					select {
					case <-parseCancelChan:
						return
					case <-time.After(1 * time.Second):
					}

					parseCurrentState := parseTimer.Get()
					if !parseCurrentState.IsRunning {
						return
					}

					select {
					case <-parseCancelChan:
						return
					default:
					}

					parseTimer.Set(TimerState{IsRunning: true, Seconds: parseCurrentState.Seconds + 1, CancelChan: parseCancelChan})
				}
			}()
			return
		}

		if parseCurrentTimer.CancelChan != nil {
			select {
			case parseCurrentTimer.CancelChan <- true:
			default:
			}
		}

		parseTimer.Set(TimerState{IsRunning: false, Seconds: parseCurrentTimer.Seconds, CancelChan: nil})
	})

	resetTimer := ui.UseEvent(func() {
		parseCurrentTimer2 := parseTimer.Get()
		if parseCurrentTimer2.IsRunning && parseCurrentTimer2.CancelChan != nil {
			select {
			case parseCurrentTimer2.CancelChan <- true:
			default:
			}
		}

		parseTimer.Set(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})
	})

	parseStartStream := ui.UseEvent(func() {
		parseCurrentStream := parseStream.Get()
		if parseCurrentStream.IsRunning {
			return
		}

		parseValues := make(chan string)
		parseStop := make(chan struct{}, 1)
		parseStream.Set(StreamState{IsRunning: true, Source: parseValues, StopChan: parseStop})

		go func() {
			defer close(parseValues)

			parseMessages := []string{
				"Connecting worker",
				"Fetching batch",
				"Transforming records",
				"Publishing update",
				"Stream complete",
			}

			for parseI2, parseMessage := range parseMessages {
				select {
				case <-parseStop:
					return
				case <-time.After(750 * time.Millisecond):
				}

				select {
				case <-parseStop:
					return
				case parseValues <- fmt.Sprintf("%02d. %s", parseI2+1, parseMessage):
				}
			}
		}()
	})

	parseStopStream := ui.UseEvent(func() {
		parseCurrentStream2 := parseStream.Get()
		if !parseCurrentStream2.IsRunning || parseCurrentStream2.StopChan == nil {
			return
		}

		select {
		case parseCurrentStream2.StopChan <- struct{}{}:
		default:
		}
	})

	resetStream := ui.UseEvent(func() {
		parseCurrentStream3 := parseStream.Get()
		if parseCurrentStream3.IsRunning && parseCurrentStream3.StopChan != nil {
			select {
			case parseCurrentStream3.StopChan <- struct{}{}:
			default:
			}
		}

		parseStream.Set(StreamState{})
	})

	ui.UseEffect(func() func() {
		if parseStreamValue.Closed() && parseStream.Get().IsRunning {
			parseStream.Set(StreamState{})
		}
		return nil
	}, parseStreamValue.Closed(), parseStream.Get().IsRunning)

	ui.UseEffect(func() func() {
		parseCurrentStream4 := parseStream.Get()
		return func() {
			if parseCurrentStream4.IsRunning && parseCurrentStream4.StopChan != nil {
				select {
				case parseCurrentStream4.StopChan <- struct{}{}:
				default:
				}
			}
		}
	}, parseStream.Get().IsRunning, parseStream.Get().StopChan)

	parseCleanupAll := ui.UseEvent(func() {
		parseCurrentTask3 := parseTaskRunner.Get()
		parseCurrentTimer3 := parseTimer.Get()
		parseCurrentStream5 := parseStream.Get()

		if parseCurrentTask3.Running {
			parseTaskRunner.Cancel()
		}

		if parseCurrentTimer3.IsRunning && parseCurrentTimer3.CancelChan != nil {
			select {
			case parseCurrentTimer3.CancelChan <- true:
			default:
			}
		}

		if parseCurrentStream5.IsRunning && parseCurrentStream5.StopChan != nil {
			select {
			case parseCurrentStream5.StopChan <- struct{}{}:
			default:
			}
		}

		parseTaskProgress.Set(0)
		parseTimer.Set(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})
		parseStream.Set(StreamState{})
	})

	parseCurrentTask4 := parseTaskRunner.Get()
	parseCurrentTimer4 := parseTimer.Get()
	parseCurrentStream6 := parseStream.Get()
	parseTaskStatus := "Ready"
	if parseCurrentTask4.Running {
		parseTaskStatus = fmt.Sprintf("Processing... %d%%", parseTaskProgress.Get())
	}
	if parseCurrentTask4.Cancelled {
		parseTaskStatus = "Cancelled"
	}
	if parseCurrentTask4.Ready {
		parseTaskStatus = parseCurrentTask4.Value.Status
	}
	if parseCurrentTask4.Error != nil && !parseCurrentTask4.Cancelled {
		parseTaskStatus = parseCurrentTask4.Error.Error()
	}
	parseStreamStatus := "Waiting to start stream"
	if parseCurrentStream6.IsRunning && !parseStreamValue.Ok() {
		parseStreamStatus = "Stream connected. Awaiting first value..."
	}
	if parseStreamValue.Ok() {
		parseStreamStatus = parseStreamValue.Get()
	}
	if parseStreamValue.Closed() && !parseCurrentStream6.IsRunning {
		parseStreamStatus = "Stream finished"
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] text-white p-8"},
		html.Div(html.Props{Class: "max-w-4xl mx-auto"},
			html.H2(html.Props{Class: "text-3xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"}, html.Text("Goroutine Example")),
			html.P(html.Props{Class: "mb-8 text-gray-400"}, html.Text("Demonstrates state updates from goroutines with proper cancellation, cleanup, and channel-driven UI streams.")),
			html.Div(html.Props{Class: "border border-red-500/30 p-6 mb-8 bg-red-500/10 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-lg font-semibold mb-4 text-red-400"}, html.Text("⚠️ Global Controls")),
				html.Button(html.Props{OnClick: parseCleanupAll, Class: "px-6 py-3 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors font-semibold shadow-lg shadow-red-500/20"}, html.Text("Cancel All & Cleanup")),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 p-6 mb-6 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-xl font-semibold mb-4 text-white"}, html.Text("Background Task")),
				html.P(html.Props{Class: "mb-2 text-gray-400"}, html.Text(fmt.Sprintf("Status: %s", parseTaskStatus))),
				html.P(html.Props{Class: "mb-4 text-gray-400"}, html.Text(fmt.Sprintf("Progress: %d%%", parseTaskProgress.Get()))),
				html.Div(html.Props{Class: "w-full bg-black/30 rounded-full h-4 mb-6 overflow-hidden"},
					html.Div(html.Props{Class: "bg-gradient-to-r from-blue-500 to-purple-600 h-4 rounded-full transition-all duration-300", Style: map[string]string{"width": fmt.Sprintf("%d%%", parseTaskProgress.Get())}}),
				),
				html.Div(html.Props{Class: "flex gap-3"},
					html.Button(html.Props{OnClick: parseStartTask, Disabled: parseCurrentTask4.Running, Class: func() string {
						if parseCurrentTask4.Running {
							return "px-4 py-2 bg-white/5 text-gray-500 rounded-lg cursor-not-allowed border border-white/5"
						}
						return "px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors shadow-lg shadow-green-500/20"
					}()}, html.Text("Start Task")),
					html.Button(html.Props{OnClick: parseCancelTask, Class: "px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition-colors shadow-lg shadow-yellow-500/20"}, html.Text("Cancel Task")),
					html.Button(html.Props{OnClick: resetTask, Class: "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Reset Task")),
				),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-xl font-semibold mb-4 text-white"}, html.Text("Timer")),
				html.P(html.Props{Class: "mb-6 text-white text-4xl font-mono font-bold tracking-wider"}, html.Text(fmt.Sprintf("%02d:%02d", parseCurrentTimer4.Seconds/60, parseCurrentTimer4.Seconds%60))),
				html.Div(html.Props{Class: "flex gap-3"},
					html.Button(html.Props{OnClick: parseToggleTimer, Class: func() string {
						if parseCurrentTimer4.IsRunning {
							return "px-6 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors shadow-lg shadow-red-500/20"
						}
						return "px-6 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors shadow-lg shadow-green-500/20"
					}()}, html.Text(func() string {
						if parseCurrentTimer4.IsRunning {
							return "Stop Timer"
						}
						return "Start Timer"
					}())),
					html.Button(html.Props{OnClick: resetTimer, Class: "px-6 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Reset Timer")),
				),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 p-6 mt-6 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-xl font-semibold mb-4 text-white"}, html.Text("Channel Stream")),
				html.P(html.Props{Class: "mb-2 text-gray-400"}, html.Text("Latest event")),
				html.P(html.Props{Class: "mb-6 text-cyan-300 text-2xl font-mono font-semibold min-h-[2rem]"}, html.Text(parseStreamStatus)),
				html.Div(html.Props{Class: "flex gap-3"},
					html.Button(html.Props{OnClick: parseStartStream, Disabled: parseCurrentStream6.IsRunning, Class: func() string {
						if parseCurrentStream6.IsRunning {
							return "px-4 py-2 bg-white/5 text-gray-500 rounded-lg cursor-not-allowed border border-white/5"
						}
						return "px-4 py-2 bg-cyan-500 text-black rounded-lg hover:bg-cyan-400 transition-colors font-semibold shadow-lg shadow-cyan-500/20"
					}()}, html.Text("Start Stream")),
					html.Button(html.Props{OnClick: parseStopStream, Class: "px-4 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 transition-colors shadow-lg shadow-orange-500/20"}, html.Text("Stop Stream")),
					html.Button(html.Props{OnClick: resetStream, Class: "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Reset Stream")),
				),
				html.P(html.Props{Class: "mt-4 text-sm text-gray-500"}, html.Text("This panel uses ui.UseChannel to consume a timed message stream from a goroutine.")),
			),
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(GoroutineExample), "body")
	select {}
}
