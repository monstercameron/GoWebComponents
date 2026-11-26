//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/events"
	. "github.com/monstercameron/GoWebComponents/v8/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v8/kvstate"
	"github.com/monstercameron/GoWebComponents/v8/router"
	"github.com/monstercameron/GoWebComponents/v8/state"
	"github.com/monstercameron/GoWebComponents/v8/ui"
)

const (
	labActivityTopic     = "desktop.lab.activity"
	labLastActivityAtom  = "desktop.lastActivity"
	labMaxTelemetryItems = 12
)

// labActivityEvent carries typed in-app and native desktop telemetry across routes.
type labActivityEvent struct {
	Seq     int
	Source  string
	Action  string
	Outcome string
	Detail  string
	At      string
}

var labActivitySeq int

// publishLabActivity fans out one typed event on the in-app bus and updates the shared atom.
func publishLabActivity(source, action, outcome, detail string) {
	labActivitySeq++
	ev := labActivityEvent{
		Seq:     labActivitySeq,
		Source:  source,
		Action:  action,
		Outcome: outcome,
		Detail:  detail,
		At:      time.Now().Format("15:04:05"),
	}
	events.Publish(labActivityTopic, ev)
}

// renderThemeToggle renders a reactive GWC v8 theme switcher seeded from OS preference.
func renderThemeToggle() ui.Node {
	preferred := ui.UsePrefersColorScheme()
	reducedMotion := ui.UsePrefersReducedMotion()
	theme, setTheme := ui.UseTheme(string(preferred))
	toggle := ui.UseEvent(func() {
		if theme == "dark" {
			setTheme("light")
			publishLabActivity("ui", "theme", "completed", "switched to light")
		} else {
			setTheme("dark")
			publishLabActivity("ui", "theme", "completed", "switched to dark")
		}
	})
	label := "Theme: Light"
	if theme == "dark" {
		label = "Theme: Dark"
	}
	motionText := "Motion: full"
	if reducedMotion {
		motionText = "Motion: reduced"
	}
	return Div(ClassStr("lab-meta-controls"),
		Span(ClassStr("lab-pill"), motionText),
		Button(ID("theme-toggle"), ClassStr("lab-pill-btn"), OnClick(toggle), label),
	)
}

// renderRouteBoundary wraps a route in GWC's crash-containment error boundary.
func renderRouteBoundary(routeKey string, child ui.Node) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ResetKeys: []any{routeKey},
		ErrorFallback: func(err error, reset func()) ui.Node {
			onReset := ui.UseEvent(func() { reset() })
			return Section(ClassStr("counter-shell"), Role("alert"),
				H1("View recovered from a render panic"),
				P(err.Error()),
				Button(OnClick(onReset), "Retry view"),
			)
		},
		Child: child,
	})
}

// renderCounter renders local state, durable multi-window kvstate, typed in-app events, and lifecycle-owned native tasks.
func renderCounter() ui.Node {
	count := ui.UseState(0)
	progress := ui.UseState(0)
	eventError := ui.UseState("")
	lastActivity := state.UseAtom(labLastActivityAtom, labActivityEvent{})

	backend := ui.UseMemo(func() kvstate.PersistenceBackend {
		client, err := desktop.Connect()
		if err != nil || !client.Supports(desktop.PersistentStorage) {
			return nil
		}
		return desktop.NewStorageBackend(client)
	})
	shared := kvstate.UsePersistedState("shared-counter", 0, kvstate.Options{
		Name:                 "wails-counter",
		Backend:              backend,
		Strategy:             kvstate.Immediate{},
		ExternalInvalidation: true,
	})

	events.UseTopic(labActivityTopic, func(ev labActivityEvent) {
		lastActivity.Set(ev)
	}, events.WithReplayLast())

	ui.UseEffect(func() func() {
		client, err := desktop.Connect()
		if err != nil || !client.Supports(desktop.PersistentStorage) {
			if err != nil {
				eventError.Set(err.Error())
			}
			return nil
		}
		stop, err := desktop.SubscribeStorage(context.Background(), client, "wails-counter", func(err error) {
			eventError.Set(err.Error())
		})
		if err != nil {
			eventError.Set(err.Error())
			return nil
		}
		return stop
	}, true)

	saveClick := ui.UseEvent(func() {
		next := shared.Get() + 1
		shared.Set(next)
		publishLabActivity("kvstate", "shared-counter", "completed", fmt.Sprintf("saved=%d", next))
	})
	bridge := ui.UseTask(func(ctx context.Context) (desktop.Client, error) {
		return desktop.Connect()
	})
	ui.UseEffect(func() func() { bridge.Start(); return nil }, true)
	bridgeState := bridge.Get()

	nativeTask := ui.UseTask(func(ctx context.Context) (contracts.CounterState, error) {
		res, err := desktop.Call[contracts.CounterState](ctx, bridge.Get().Value, "counter.increment")
		if err == nil {
			publishLabActivity("native", "counter.increment", "completed", fmt.Sprintf("value=%d", res.Value))
		}
		return res, err
	})
	dialogTask := ui.UseTask(func(ctx context.Context) (string, error) {
		selection, err := bridge.Get().Value.OpenFile(ctx, desktop.FileDialogOptions{Title: "Choose a file"})
		if err != nil || selection.Cancelled {
			return "", err
		}
		publishLabActivity("native", "open-file", "completed", selection.Paths[0])
		return selection.Paths[0], nil
	})
	progressTask := ui.UseTask(func(ctx context.Context) (any, error) {
		return desktop.Call[any](ctx, bridge.Get().Value, "counter.progress")
	})
	rejectTask := ui.UseTask(func(ctx context.Context) (any, error) {
		return desktop.Call[any](ctx, bridge.Get().Value, "counter.reject")
	})
	workTask := ui.UseTask(func(ctx context.Context) (any, error) {
		return desktop.Call[any](ctx, bridge.Get().Value, "counter.work")
	})

	ui.UseEffect(func() func() {
		client, err := desktop.Connect()
		if err != nil {
			eventError.Set(err.Error())
			return nil
		}
		stop, err := desktop.Subscribe[contracts.Progress](context.Background(), client, "counter.progress", func(ev contracts.Progress, decodeErr error) {
			if decodeErr != nil {
				eventError.Set(decodeErr.Error())
				return
			}
			progress.Set(ev.Percent)
		})
		if err != nil {
			eventError.Set(err.Error())
			return nil
		}
		return stop
	}, true)

	increment := ui.UseEvent(func() {
		count.Update(func(prev int) int { return prev + 1 })
	})
	nativeClick := ui.UseEvent(func() { nativeTask.Start() })
	dialogClick := ui.UseEvent(func() { dialogTask.Start() })
	progressClick := ui.UseEvent(func() { progressTask.Start() })
	rejectClick := ui.UseEvent(func() { rejectTask.Start() })
	workClick := ui.UseEvent(func() { workTask.Start() })
	cancelClick := ui.UseEvent(func() { workTask.Cancel() })

	message := "Native bridge loading…"
	if bridgeState.Ready {
		message = "Native bridge ready"
	}
	if bridgeState.Error != nil {
		message = "Native bridge unavailable: " + bridgeState.Error.Error()
	}
	nativeState := nativeTask.Get()
	if nativeState.Ready {
		message = fmt.Sprintf("Native response: %d", nativeState.Value.Value)
	}
	if nativeState.Error != nil {
		message = "Native call failed: " + nativeState.Error.Error()
	}

	dialogState := dialogTask.Get()
	dialogMessage := "No file selected"
	if dialogState.Ready {
		dialogMessage = "Selected: " + dialogState.Value
		if dialogState.Value == "" {
			dialogMessage = "Dialog cancelled"
		}
	}
	if dialogState.Error != nil {
		dialogMessage = "Dialog failed: " + dialogState.Error.Error()
	}

	errorMessage := eventError.Get()
	if progressTask.Get().Error != nil {
		errorMessage = progressTask.Get().Error.Error()
	}
	if rejectTask.Get().Error != nil {
		errorMessage = rejectTask.Get().Error.Error()
	}
	if shared.Err() != nil {
		errorMessage = shared.Err().Error()
	}

	recent := lastActivity.Get()

	return Div(ClassStr("counter-shell"),
		Header(ClassStr("tester-topbar"),
			Div(
				Span(ClassStr("tester-kicker"), "GWC v8 · WAILS v3.1.0 · COUNTER"),
				H1(ID("route-title"), "Wails Counter"),
				P("Window-local state, durable multi-window kvstate, typed in-app events, and cancellable native RPC."),
			),
			Div(ClassStr("tester-topbar-right"),
				ui.CreateElement(renderThemeToggle),
				Nav(ClassStr("tester-links"), Aria("label", "Routes"),
					A(ID("tester-link"), Href("#/tester"), "Windows API Lab"),
					A(ID("about-link"), Href("#/about"), "About this desktop example"),
				),
			),
		),
		Main(ClassStr("counter-main"),
			Section(ClassStr("tester-panel"),
				H2("01 / Local & native round trips"),
				P(ID("local-count"), ClassStr("counter-metric"), Textf("Count: %d", count.Get())),
				Div(ClassStr("tester-controls"),
					Button(ID("local-increment"), OnClick(increment), "Increment"),
					Button(ID("native-increment"), OnClick(nativeClick), Disabled(!bridgeState.Ready || nativeState.Running), "Native round trip"),
					Button(ID("native-open-file"), OnClick(dialogClick), Disabled(!bridgeState.Ready || !bridgeState.Value.Supports(desktop.FileDialogs) || dialogState.Running), "Open file"),
					Button(ID("native-progress"), OnClick(progressClick), Disabled(!bridgeState.Ready || progressTask.Get().Running), "Progress"),
					Button(ID("native-reject"), OnClick(rejectClick), Disabled(!bridgeState.Ready || rejectTask.Get().Running), "Test service error"),
				),
				P(ID("native-status"), ClassStr("tester-case__hint"), Role("status"), message),
				P(ID("dialog-status"), ClassStr("tester-case__hint"), Role("status"), dialogMessage),
				P(ID("native-error"), ClassStr(ClassNames("tester-case__hint", When(errorMessage != "", "is-error-text"))), Role("status"), errorMessage),
			),
			Section(ClassStr("tester-panel"),
				H2("02 / Durable cross-window kvstate"),
				P(ID("shared-count"), ClassStr("counter-metric"), Textf("Saved count: %d", shared.Get())),
				Div(ClassStr("tester-controls"),
					Button(ID("shared-increment"), OnClick(saveClick), Disabled(shared.Loading()), "Increment shared saved count"),
				),
			),
			Section(ClassStr("tester-panel"),
				H2("03 / Cancellable native work & progress stream"),
				Div(ClassStr("tester-controls"),
					Button(ID("native-work"), OnClick(workClick), Disabled(!bridgeState.Ready || workTask.Get().Running), "Start 10-second work"),
					Button(ID("native-cancel"), OnClick(cancelClick), Disabled(!workTask.Get().Running), "Cancel native work"),
				),
				P(ID("progress-status"), ClassStr("tester-case__hint"), Role("status"), Textf("Native progress: %d%%", progress.Get())),
				Div(ClassStr("progress"),
					Div(ID("progress-value"), Style(map[string]string{"width": fmt.Sprintf("%d%%", progress.Get())})),
				),
			),
			If(recent.Seq > 0,
				Section(ClassStr("tester-panel"),
					H2("04 / Last cross-route activity (events.UseTopic + state.UseAtom)"),
					P(ID("counter-last-activity"), ClassStr("tester-case__hint"), Textf("[%s] %s · %s (%s) %s", recent.At, recent.Source, recent.Action, recent.Outcome, recent.Detail)),
				),
			),
		),
	)
}

// renderAbout provides a separate route so navigation exercises unmount cleanup and shared reactive atoms.
func renderAbout() ui.Node {
	lastActivity := state.UseAtom(labLastActivityAtom, labActivityEvent{})
	recent := lastActivity.Get()
	return Div(ClassStr("counter-shell"),
		Header(ClassStr("tester-topbar"),
			Div(
				Span(ClassStr("tester-kicker"), "GWC v8 · WAILS v3.1.0 · ARCHITECTURE"),
				H1(ID("route-title"), "About this desktop example"),
				P("GWC renders the UI inside a native Wails v3.1.0 WebView2 window with crash containment and typed RPC."),
			),
			Div(ClassStr("tester-topbar-right"),
				ui.CreateElement(renderThemeToggle),
				Nav(ClassStr("tester-links"), Aria("label", "Routes"),
					A(ID("counter-link"), Href("#/counter"), "Back to counter"),
					A(ID("tester-link"), Href("#/tester"), "Windows API Lab"),
				),
			),
		),
		Main(ClassStr("counter-main"),
			Section(ClassStr("tester-panel"),
				H2("Active framework & host surfaces"),
				P("Markup: html/shorthand DSL (MapKeyed, If, IfElse, ClassNames, When) + ui.ErrorBoundary + ui.UseTheme + ui.UseDeferredValue + ui.UseGlobalKey + ui.UseId."),
				P("Events & state: typed events.UseTopic (WithReplayLast), state.UseAtom, durable cross-window kvstate.UsePersistedState, and Wails v3.1.0 native event streams."),
				If(recent.Seq > 0,
					P(ClassStr("tester-case__hint"), Textf("Latest observed event across routes: [%s] %s · %s (%s)", recent.At, recent.Source, recent.Action, recent.Outcome)),
				),
			),
		),
	)
}

// main mounts the framework hash router, which renders through ui.Render.
func main() {
	hashRouter := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/counter"})
	hashRouter.Register("/counter", func(router.Attrs) *router.Element {
		return renderRouteBoundary("/counter", ui.CreateElement(renderCounter))
	})
	hashRouter.Register("/about", func(router.Attrs) *router.Element {
		return renderRouteBoundary("/about", ui.CreateElement(renderAbout))
	})
	hashRouter.Register("/tester", func(router.Attrs) *router.Element {
		return renderRouteBoundary("/tester", ui.CreateElement(renderTester))
	})
	hashRouter.Mount("#app")
	select {}
}
