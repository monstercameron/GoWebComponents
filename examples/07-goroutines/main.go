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
	taskProgress := ui.UseState(0)
	taskRunner := ui.UseTask(func(ctx context.Context) (BackgroundTask, error) {
		for i := 0; i <= 100; i += 10 {
			select {
			case <-ctx.Done():
				return BackgroundTask{Progress: i, Status: "Cancelled"}, ctx.Err()
			default:
			}

			select {
			case <-ctx.Done():
				return BackgroundTask{Progress: i, Status: "Cancelled"}, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}

			progress := i
			taskProgress.Set(progress)
		}

		return BackgroundTask{Progress: 100, Status: "Completed!"}, nil
	})
	timer := ui.UseState(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})
	stream := ui.UseState(StreamState{})
	streamValue := ui.UseChannel(stream.Get().Source)

	startTask := ui.UseEvent(func() {
		currentTask := taskRunner.Get()
		if currentTask.Running {
			return
		}
		taskProgress.Set(0)
		taskRunner.Start()
	})

	cancelTask := ui.UseEvent(func() {
		currentTask := taskRunner.Get()
		if !currentTask.Running {
			return
		}
		taskRunner.Cancel()
	})

	resetTask := ui.UseEvent(func() {
		taskRunner.Cancel()
		taskProgress.Set(0)
	})

	toggleTimer := ui.UseEvent(func() {
		currentTimer := timer.Get()
		if !currentTimer.IsRunning {
			cancelChan := make(chan bool, 1)
			timer.Set(TimerState{IsRunning: true, Seconds: currentTimer.Seconds, CancelChan: cancelChan})

			go func() {
				for {
					select {
					case <-cancelChan:
						return
					case <-time.After(1 * time.Second):
					}

					currentState := timer.Get()
					if !currentState.IsRunning {
						return
					}

					select {
					case <-cancelChan:
						return
					default:
					}

					timer.Set(TimerState{IsRunning: true, Seconds: currentState.Seconds + 1, CancelChan: cancelChan})
				}
			}()
			return
		}

		if currentTimer.CancelChan != nil {
			select {
			case currentTimer.CancelChan <- true:
			default:
			}
		}

		timer.Set(TimerState{IsRunning: false, Seconds: currentTimer.Seconds, CancelChan: nil})
	})

	resetTimer := ui.UseEvent(func() {
		currentTimer := timer.Get()
		if currentTimer.IsRunning && currentTimer.CancelChan != nil {
			select {
			case currentTimer.CancelChan <- true:
			default:
			}
		}

		timer.Set(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})
	})

	startStream := ui.UseEvent(func() {
		currentStream := stream.Get()
		if currentStream.IsRunning {
			return
		}

		values := make(chan string)
		stop := make(chan struct{}, 1)
		stream.Set(StreamState{IsRunning: true, Source: values, StopChan: stop})

		go func() {
			defer close(values)

			messages := []string{
				"Connecting worker",
				"Fetching batch",
				"Transforming records",
				"Publishing update",
				"Stream complete",
			}

			for i, message := range messages {
				select {
				case <-stop:
					return
				case <-time.After(750 * time.Millisecond):
				}

				select {
				case <-stop:
					return
				case values <- fmt.Sprintf("%02d. %s", i+1, message):
				}
			}
		}()
	})

	stopStream := ui.UseEvent(func() {
		currentStream := stream.Get()
		if !currentStream.IsRunning || currentStream.StopChan == nil {
			return
		}

		select {
		case currentStream.StopChan <- struct{}{}:
		default:
		}
	})

	resetStream := ui.UseEvent(func() {
		currentStream := stream.Get()
		if currentStream.IsRunning && currentStream.StopChan != nil {
			select {
			case currentStream.StopChan <- struct{}{}:
			default:
			}
		}

		stream.Set(StreamState{})
	})

	ui.UseEffect(func() func() {
		if streamValue.Closed() && stream.Get().IsRunning {
			stream.Set(StreamState{})
		}
		return nil
	}, streamValue.Closed(), stream.Get().IsRunning)

	ui.UseEffect(func() func() {
		currentStream := stream.Get()
		return func() {
			if currentStream.IsRunning && currentStream.StopChan != nil {
				select {
				case currentStream.StopChan <- struct{}{}:
				default:
				}
			}
		}
	}, stream.Get().IsRunning, stream.Get().StopChan)

	cleanupAll := ui.UseEvent(func() {
		currentTask := taskRunner.Get()
		currentTimer := timer.Get()
		currentStream := stream.Get()

		if currentTask.Running {
			taskRunner.Cancel()
		}

		if currentTimer.IsRunning && currentTimer.CancelChan != nil {
			select {
			case currentTimer.CancelChan <- true:
			default:
			}
		}

		if currentStream.IsRunning && currentStream.StopChan != nil {
			select {
			case currentStream.StopChan <- struct{}{}:
			default:
			}
		}

		taskProgress.Set(0)
		timer.Set(TimerState{IsRunning: false, Seconds: 0, CancelChan: nil})
		stream.Set(StreamState{})
	})

	currentTask := taskRunner.Get()
	currentTimer := timer.Get()
	currentStream := stream.Get()
	taskStatus := "Ready"
	if currentTask.Running {
		taskStatus = fmt.Sprintf("Processing... %d%%", taskProgress.Get())
	}
	if currentTask.Cancelled {
		taskStatus = "Cancelled"
	}
	if currentTask.Ready {
		taskStatus = currentTask.Value.Status
	}
	if currentTask.Error != nil && !currentTask.Cancelled {
		taskStatus = currentTask.Error.Error()
	}
	streamStatus := "Waiting to start stream"
	if currentStream.IsRunning && !streamValue.Ok() {
		streamStatus = "Stream connected. Awaiting first value..."
	}
	if streamValue.Ok() {
		streamStatus = streamValue.Get()
	}
	if streamValue.Closed() && !currentStream.IsRunning {
		streamStatus = "Stream finished"
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] text-white p-8"},
		html.Div(html.Props{Class: "max-w-4xl mx-auto"},
			html.H2(html.Props{Class: "text-3xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"}, html.Text("Goroutine Example")),
			html.P(html.Props{Class: "mb-8 text-gray-400"}, html.Text("Demonstrates state updates from goroutines with proper cancellation, cleanup, and channel-driven UI streams.")),
			html.Div(html.Props{Class: "border border-red-500/30 p-6 mb-8 bg-red-500/10 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-lg font-semibold mb-4 text-red-400"}, html.Text("⚠️ Global Controls")),
				html.Button(html.Props{OnClick: cleanupAll, Class: "px-6 py-3 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors font-semibold shadow-lg shadow-red-500/20"}, html.Text("Cancel All & Cleanup")),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 p-6 mb-6 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-xl font-semibold mb-4 text-white"}, html.Text("Background Task")),
				html.P(html.Props{Class: "mb-2 text-gray-400"}, html.Text(fmt.Sprintf("Status: %s", taskStatus))),
				html.P(html.Props{Class: "mb-4 text-gray-400"}, html.Text(fmt.Sprintf("Progress: %d%%", taskProgress.Get()))),
				html.Div(html.Props{Class: "w-full bg-black/30 rounded-full h-4 mb-6 overflow-hidden"},
					html.Div(html.Props{Class: "bg-gradient-to-r from-blue-500 to-purple-600 h-4 rounded-full transition-all duration-300", Style: map[string]string{"width": fmt.Sprintf("%d%%", taskProgress.Get())}}),
				),
				html.Div(html.Props{Class: "flex gap-3"},
					html.Button(html.Props{OnClick: startTask, Disabled: currentTask.Running, Class: func() string {
						if currentTask.Running {
							return "px-4 py-2 bg-white/5 text-gray-500 rounded-lg cursor-not-allowed border border-white/5"
						}
						return "px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors shadow-lg shadow-green-500/20"
					}()}, html.Text("Start Task")),
					html.Button(html.Props{OnClick: cancelTask, Class: "px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition-colors shadow-lg shadow-yellow-500/20"}, html.Text("Cancel Task")),
					html.Button(html.Props{OnClick: resetTask, Class: "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Reset Task")),
				),
			),
			html.Div(html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm"},
				html.H3(html.Props{Class: "text-xl font-semibold mb-4 text-white"}, html.Text("Timer")),
				html.P(html.Props{Class: "mb-6 text-white text-4xl font-mono font-bold tracking-wider"}, html.Text(fmt.Sprintf("%02d:%02d", currentTimer.Seconds/60, currentTimer.Seconds%60))),
				html.Div(html.Props{Class: "flex gap-3"},
					html.Button(html.Props{OnClick: toggleTimer, Class: func() string {
						if currentTimer.IsRunning {
							return "px-6 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors shadow-lg shadow-red-500/20"
						}
						return "px-6 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors shadow-lg shadow-green-500/20"
					}()}, html.Text(func() string {
						if currentTimer.IsRunning {
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
				html.P(html.Props{Class: "mb-6 text-cyan-300 text-2xl font-mono font-semibold min-h-[2rem]"}, html.Text(streamStatus)),
				html.Div(html.Props{Class: "flex gap-3"},
					html.Button(html.Props{OnClick: startStream, Disabled: currentStream.IsRunning, Class: func() string {
						if currentStream.IsRunning {
							return "px-4 py-2 bg-white/5 text-gray-500 rounded-lg cursor-not-allowed border border-white/5"
						}
						return "px-4 py-2 bg-cyan-500 text-black rounded-lg hover:bg-cyan-400 transition-colors font-semibold shadow-lg shadow-cyan-500/20"
					}()}, html.Text("Start Stream")),
					html.Button(html.Props{OnClick: stopStream, Class: "px-4 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 transition-colors shadow-lg shadow-orange-500/20"}, html.Text("Stop Stream")),
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
