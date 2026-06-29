//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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

	return shared.ExamplePage(
		"Goroutines",
		"ui.UseTask + ui.UseChannel",
		"Start concurrent work, cancel it safely, and watch goroutine-driven state flow back into the UI.",
		shared.ExamplePanel("Global Controls",
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				html.Button(html.Props{OnClick: parseCleanupAll, Class: "rounded-2xl border border-rose-400/20 bg-rose-400/10 px-4 py-2 text-sm font-medium text-rose-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-rose-400/15 active:translate-y-0 active:scale-95"}, html.Text("Cancel All & Cleanup")),
			),
		),
		shared.ExamplePanel("Background Task",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
				shared.ExampleStat("Status", parseTaskStatus),
				shared.ExampleStat("Progress", fmt.Sprintf("%d%%", parseTaskProgress.Get())),
			),
			html.Div(html.Props{Class: "w-full overflow-hidden rounded-full bg-black/30"},
				html.Div(html.Props{Class: "h-4 rounded-full bg-gradient-to-r from-cyan-400 to-emerald-400 transition-all duration-300", Style: map[string]string{"width": fmt.Sprintf("%d%%", parseTaskProgress.Get())}}),
			),
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				html.Button(html.Props{OnClick: parseStartTask, Disabled: parseCurrentTask4.Running, Class: func() string {
					if parseCurrentTask4.Running {
						return "rounded-2xl border border-white/5 bg-white/5 px-4 py-2 text-sm font-medium text-slate-500"
					}
					return "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95"
				}()}, html.Text("Start Task")),
				html.Button(html.Props{OnClick: parseCancelTask, Class: "rounded-2xl border border-amber-400/20 bg-amber-400/10 px-4 py-2 text-sm font-medium text-amber-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-amber-400/15 active:translate-y-0 active:scale-95"}, html.Text("Cancel Task")),
				html.Button(html.Props{OnClick: resetTask, Class: "rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-200 transition-all duration-200 hover:-translate-y-0.5 hover:border-cyan-300/30 hover:text-cyan-100 active:translate-y-0 active:scale-95"}, html.Text("Reset Task")),
			),
		),
		shared.ExamplePanel("Timer",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
				shared.ExampleStat("Clock", fmt.Sprintf("%02d:%02d", parseCurrentTimer4.Seconds/60, parseCurrentTimer4.Seconds%60)),
				shared.ExampleStat("Running", fmt.Sprintf("%t", parseCurrentTimer4.IsRunning)),
			),
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				html.Button(html.Props{OnClick: parseToggleTimer, Class: func() string {
					if parseCurrentTimer4.IsRunning {
						return "rounded-2xl border border-rose-400/20 bg-rose-400/10 px-4 py-2 text-sm font-medium text-rose-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-rose-400/15 active:translate-y-0 active:scale-95"
					}
					return "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95"
				}()}, html.Text(func() string {
					if parseCurrentTimer4.IsRunning {
						return "Stop Timer"
					}
					return "Start Timer"
				}())),
				html.Button(html.Props{OnClick: resetTimer, Class: "rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-200 transition-all duration-200 hover:-translate-y-0.5 hover:border-cyan-300/30 hover:text-cyan-100 active:translate-y-0 active:scale-95"}, html.Text("Reset Timer")),
			),
		),
		shared.ExamplePanel("Channel Stream",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
				shared.ExampleStat("Latest Event", parseStreamStatus),
				shared.ExampleStat("Stream Running", fmt.Sprintf("%t", parseCurrentStream6.IsRunning)),
			),
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				html.Button(html.Props{OnClick: parseStartStream, Disabled: parseCurrentStream6.IsRunning, Class: func() string {
					if parseCurrentStream6.IsRunning {
						return "rounded-2xl border border-white/5 bg-white/5 px-4 py-2 text-sm font-medium text-slate-500"
					}
					return "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95"
				}()}, html.Text("Start Stream")),
				html.Button(html.Props{OnClick: parseStopStream, Class: "rounded-2xl border border-amber-400/20 bg-amber-400/10 px-4 py-2 text-sm font-medium text-amber-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-amber-400/15 active:translate-y-0 active:scale-95"}, html.Text("Stop Stream")),
				html.Button(html.Props{OnClick: resetStream, Class: "rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-200 transition-all duration-200 hover:-translate-y-0.5 hover:border-cyan-300/30 hover:text-cyan-100 active:translate-y-0 active:scale-95"}, html.Text("Reset Stream")),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(GoroutineExample))
	exampleboot.WaitExampleRuntime()
}
