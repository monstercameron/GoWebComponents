//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	multiWindowChannelName = "example:multi-window:ops"
	multiWindowPopupURL    = "./multi-window-console-popup.html"
)

func multiWindowExampleRoot() ui.Node {
	location, err := interop.WindowLocation()
	if err == nil {
		path := strings.ToLower(strings.TrimSpace(location.Pathname()))
		if strings.Contains(path, "popup") {
			return popupSurfaceExample()
		}
	}
	return openerSurfaceExample()
}

func openerSurfaceExample() ui.Node {
	session := ui.UseState("signed-in")
	activeRoute := ui.UseState("/inventory?tab=summary")
	activeDocument := ui.UseState("INV-204")
	latestIntent := ui.UseState("Waiting for popup intent.")
	popupStatus := ui.UseState("Closed")
	logs := ui.UseState([]string{"Open the popup to establish the operator-console channel."})
	channelRef := ui.UseRef(interop.WindowChannel{})
	cancelRef := ui.UseRef((func())(nil))

	appendLog := func(line string) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return
		}
		logs.Update(func(previous []string) []string {
			next := append([]string{trimmed}, previous...)
			if len(next) > 8 {
				next = next[:8]
			}
			return next
		})
	}

	describeError := func(prefix string, err error) string {
		if err == nil {
			return prefix
		}
		if code, ok := interop.CodeOf(err); ok {
			return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
		}
		return fmt.Sprintf("%s: %v", prefix, err)
	}

	releaseChannel := func(closeWindow bool) {
		if cancel := cancelRef.Get(); cancel != nil {
			cancel()
			cancelRef.Set(nil)
		}
		channel := channelRef.Get()
		if closeWindow && channel.Name() != "" && !channel.Closed() {
			_ = channel.Close()
		}
		channelRef.Set(interop.WindowChannel{})
	}

	handleSignal := func(message interop.DecodedWindowEnvelope[interop.SurfaceSignal], err error) {
		if err != nil {
			appendLog(describeError("Popup message failed", err))
			return
		}
		switch message.Payload.Kind {
		case interop.SurfaceSignalSession:
			if message.Payload.Session == nil {
				appendLog("Popup session message arrived without payload.")
				return
			}
			session.Set(message.Payload.Session.Status)
			appendLog(fmt.Sprintf("Popup reported session %q: %s", message.Payload.Session.Status, message.Payload.Session.Reason))
		case interop.SurfaceSignalRoute:
			if message.Payload.Route == nil {
				appendLog("Popup route message arrived without payload.")
				return
			}
			nextRoute := message.Payload.Route.Path
			if strings.TrimSpace(message.Payload.Route.Query) != "" {
				nextRoute += "?" + message.Payload.Route.Query
			}
			activeRoute.Set(nextRoute)
			appendLog(fmt.Sprintf("Popup focused route %s", nextRoute))
		case interop.SurfaceSignalSelection:
			if message.Payload.Selection == nil {
				appendLog("Popup selection message arrived without payload.")
				return
			}
			activeDocument.Set(message.Payload.Selection.ID)
			appendLog(fmt.Sprintf("Popup selected %s %s", message.Payload.Selection.Scope, message.Payload.Selection.ID))
		case interop.SurfaceSignalIntent:
			if message.Payload.Intent == nil {
				appendLog("Popup intent message arrived without payload.")
				return
			}
			summary := fmt.Sprintf("%s -> %s", message.Payload.Intent.Action, message.Payload.Intent.Target)
			latestIntent.Set(summary)
			appendLog("Popup requested intent " + summary)
		default:
			appendLog("Popup sent an unknown multi-surface payload.")
		}
	}

	send := func(label string, publish func(interop.WindowChannel) error, after func()) {
		channel := channelRef.Get()
		if channel.Name() == "" || channel.Closed() {
			popupStatus.Set("Closed")
			appendLog("Open the popup before sending a multi-surface signal.")
			return
		}
		if err := publish(channel); err != nil {
			appendLog(describeError(label, err))
			return
		}
		if after != nil {
			after()
		}
		appendLog(label)
	}

	openPopup := ui.UseEvent(func() {
		current := channelRef.Get()
		if current.Name() != "" && !current.Closed() {
			_ = current.Focus()
			popupStatus.Set("Connected")
			appendLog("Reused the existing popup and focused it.")
			return
		}

		releaseChannel(false)
		channel, err := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
			URL:      multiWindowPopupURL,
			Name:     multiWindowChannelName,
			Features: "popup=yes,width=560,height=760",
		})
		if err != nil {
			appendLog(describeError("Opening popup failed", err))
			return
		}
		subscription, err := interop.SubscribeSurfaceSignals(channel, handleSignal)
		if err != nil {
			_ = channel.Close()
			appendLog(describeError("Popup subscription failed", err))
			return
		}
		channelRef.Set(channel)
		cancelRef.Set(subscription.Cancel)
		popupStatus.Set("Connected")
		appendLog("Opened the popup inspector and subscribed to its surface signals.")
	})

	focusPopup := ui.UseEvent(func() {
		send("Focused the popup window.", func(channel interop.WindowChannel) error {
			return channel.Focus()
		}, nil)
	})

	closePopup := ui.UseEvent(func() {
		channel := channelRef.Get()
		if channel.Name() == "" || channel.Closed() {
			popupStatus.Set("Closed")
			appendLog("Popup is already closed.")
			releaseChannel(false)
			return
		}
		if err := channel.Close(); err != nil {
			appendLog(describeError("Closing popup failed", err))
			return
		}
		releaseChannel(false)
		popupStatus.Set("Closed")
		appendLog("Closed the popup from the opener.")
	})

	sendSessionExpired := ui.UseEvent(func() {
		expiresAt := time.Now().UTC().Add(15 * time.Minute).Round(time.Second)
		send("Sent a session-expired signal to the popup.", func(channel interop.WindowChannel) error {
			return interop.PublishSessionExpired(channel, "Re-authentication required in every active surface.", "/login", expiresAt)
		}, func() {
			session.Set("expired")
		})
	})

	sendLogout := ui.UseEvent(func() {
		send("Broadcast a logout signal to the popup.", func(channel interop.WindowChannel) error {
			return interop.PublishLogout(channel, "Operator signed out in the main workspace.")
		}, func() {
			session.Set("signed-out")
		})
	})

	sendRouteFocus := ui.UseEvent(func() {
		send("Sent route focus for /orders/42.", func(channel interop.WindowChannel) error {
			return interop.PublishRouteFocus(channel, "/orders/42", "tab=activity", "order-heading")
		}, func() {
			activeRoute.Set("/orders/42?tab=activity")
		})
	})

	sendSelection := ui.UseEvent(func() {
		send("Sent active-document selection INV-204.", func(channel interop.WindowChannel) error {
			return interop.PublishSelection(channel, "invoice", "INV-204", "rev-12")
		}, func() {
			activeDocument.Set("INV-204")
		})
	})

	sendIntent := ui.UseEvent(func() {
		send("Sent a focus-panel intent to the popup.", func(channel interop.WindowChannel) error {
			return interop.PublishIntent(channel, interop.SurfaceIntentFocusPanel, "inventory-inspector", map[string]string{"tab": "activity"})
		}, func() {
			latestIntent.Set("focus-panel -> inventory-inspector")
		})
	})

	ui.UseEffect(func() func() {
		timer, err := interop.ScheduleInterval(500*time.Millisecond, func() {
			channel := channelRef.Get()
			if channel.Name() == "" {
				return
			}
			if channel.Closed() {
				releaseChannel(false)
				popupStatus.Set("Closed")
				appendLog("Popup closed unexpectedly. Reopen it to restore coordination.")
			}
		})
		return func() {
			if err == nil {
				_ = timer.Cancel()
			}
			releaseChannel(true)
		}
	}, true)

	logNodes := make([]ui.Node, 0, len(logs.Get()))
	for _, entry := range logs.Get() {
		logNodes = append(logNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(entry)),
		)
	}

	return shared.ExamplePage(
		"Multi-Window Console",
		"interop.WindowChannel and multi-surface signals",
		"Open a dedicated operator popup, propagate session and route context, and treat popup loss as ordinary recoverable state instead of a fatal runtime event.",
		shared.ExamplePanel("Opener controls",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Session", session.Get()),
				shared.ExampleStat("Route", activeRoute.Get()),
				shared.ExampleStat("Document", activeDocument.Get()),
				shared.ExampleStat("Popup", popupStatus.Get()),
			),
			html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
				shared.ExampleButton("Open popup", openPopup),
				shared.ExampleButton("Focus popup", focusPopup),
				shared.ExampleButton("Close popup", closePopup),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Expire session", sendSessionExpired),
				shared.ExampleButton("Send logout", sendLogout),
				shared.ExampleButton("Focus /orders/42", sendRouteFocus),
				shared.ExampleButton("Select INV-204", sendSelection),
				shared.ExampleButton("Focus inspector panel", sendIntent),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The opener remains the authoritative owner for popup lifecycle. Signals are typed and explicit, not implicit global state replication.")),
		),
		shared.ExamplePanel("Popup feedback",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Latest intent", latestIntent.Get()),
				shared.ExampleStat("Channel name", multiWindowChannelName),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("If the popup selects a different document or requests a focus change, those signals come back through the same typed channel and update the opener view here.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, logNodes...),
		),
		shared.ExamplePanel("Integration shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Use WindowChannel only for targeted opener or popup workflows. Keep one surface authoritative, send route or selection hints explicitly, and treat popup disconnects as a state transition you can detect and recover from.")),
			shared.ExampleCode(
				`channel, _ := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{URL: "./multi-window-console-popup.html", Name: "example:multi-window:ops"})`,
				`sub, _ := interop.SubscribeSurfaceSignals(channel, func(message interop.DecodedWindowEnvelope[interop.SurfaceSignal], err error) { ... })`,
				`_ = interop.PublishSessionExpired(channel, "Re-auth required.", "/login", time.Now().UTC().Add(15*time.Minute))`,
				`_ = interop.PublishRouteFocus(channel, "/orders/42", "tab=activity", "order-heading")`,
			),
		),
	)
}

func popupSurfaceExample() ui.Node {
	session := ui.UseState("waiting")
	activeRoute := ui.UseState("Waiting for opener route focus.")
	activeDocument := ui.UseState("No active document")
	latestIntent := ui.UseState("Waiting for opener intent.")
	connection := ui.UseState("Connecting to opener...")
	orphaned := ui.UseState(false)
	logs := ui.UseState([]string{"This popup is waiting for the opener channel."})
	channelRef := ui.UseRef(interop.WindowChannel{})
	cancelRef := ui.UseRef((func())(nil))

	appendLog := func(line string) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return
		}
		logs.Update(func(previous []string) []string {
			next := append([]string{trimmed}, previous...)
			if len(next) > 8 {
				next = next[:8]
			}
			return next
		})
	}

	describeError := func(prefix string, err error) string {
		if err == nil {
			return prefix
		}
		if code, ok := interop.CodeOf(err); ok {
			return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
		}
		return fmt.Sprintf("%s: %v", prefix, err)
	}

	send := func(label string, publish func(interop.WindowChannel) error, after func()) {
		channel := channelRef.Get()
		if channel.Name() == "" || channel.Closed() {
			orphaned.Set(true)
			connection.Set("Opener unavailable")
			appendLog("Opener is unavailable; this popup is now orphaned.")
			return
		}
		if err := publish(channel); err != nil {
			appendLog(describeError(label, err))
			return
		}
		if after != nil {
			after()
		}
		appendLog(label)
	}

	handleSignal := func(message interop.DecodedWindowEnvelope[interop.SurfaceSignal], err error) {
		if err != nil {
			appendLog(describeError("Opener message failed", err))
			return
		}
		switch message.Payload.Kind {
		case interop.SurfaceSignalSession:
			if message.Payload.Session == nil {
				return
			}
			session.Set(message.Payload.Session.Status)
			connection.Set("Connected")
			orphaned.Set(false)
			appendLog(fmt.Sprintf("Received session %q from opener.", message.Payload.Session.Status))
		case interop.SurfaceSignalRoute:
			if message.Payload.Route == nil {
				return
			}
			nextRoute := message.Payload.Route.Path
			if strings.TrimSpace(message.Payload.Route.Query) != "" {
				nextRoute += "?" + message.Payload.Route.Query
			}
			activeRoute.Set(nextRoute)
			appendLog("Received route focus " + nextRoute)
		case interop.SurfaceSignalSelection:
			if message.Payload.Selection == nil {
				return
			}
			activeDocument.Set(message.Payload.Selection.ID)
			appendLog(fmt.Sprintf("Received active %s %s", message.Payload.Selection.Scope, message.Payload.Selection.ID))
		case interop.SurfaceSignalIntent:
			if message.Payload.Intent == nil {
				return
			}
			latestIntent.Set(fmt.Sprintf("%s -> %s", message.Payload.Intent.Action, message.Payload.Intent.Target))
			appendLog(fmt.Sprintf("Received opener intent %s", message.Payload.Intent.Action))
		}
	}

	ui.UseEffect(func() func() {
		channel, err := interop.WindowOpenerChannel(interop.WindowChannelOptions{Name: multiWindowChannelName})
		if err != nil {
			orphaned.Set(true)
			connection.Set("Opened without an opener")
			appendLog(describeError("No opener channel available", err))
			return nil
		}
		channelRef.Set(channel)
		connection.Set("Connected")
		orphaned.Set(channel.Closed())

		subscription, err := interop.SubscribeSurfaceSignals(channel, handleSignal)
		if err != nil {
			connection.Set("Subscription failed")
			appendLog(describeError("Popup subscription failed", err))
			return nil
		}
		cancelRef.Set(subscription.Cancel)
		appendLog("Connected to the opener and subscribed to surface signals.")

		timer, timerErr := interop.ScheduleInterval(500*time.Millisecond, func() {
			current := channelRef.Get()
			if current.Name() == "" {
				return
			}
			if current.Closed() {
				orphaned.Set(true)
				connection.Set("Opener disconnected")
			}
		})

		return func() {
			if cancel := cancelRef.Get(); cancel != nil {
				cancel()
				cancelRef.Set(nil)
			}
			if timerErr == nil {
				_ = timer.Cancel()
			}
		}
	}, true)

	requestOrdersFocus := ui.UseEvent(func() {
		send("Requested route focus back to /inventory/alerts.", func(channel interop.WindowChannel) error {
			return interop.PublishRouteFocus(channel, "/inventory/alerts", "panel=exceptions", "alerts-heading")
		}, func() {
			activeRoute.Set("/inventory/alerts?panel=exceptions")
		})
	})

	requestSelection := ui.UseEvent(func() {
		send("Selected SKU-42 in the popup.", func(channel interop.WindowChannel) error {
			return interop.PublishSelection(channel, "sku", "SKU-42", "popup-rev-3")
		}, func() {
			activeDocument.Set("SKU-42")
		})
	})

	requestFocusIntent := ui.UseEvent(func() {
		send("Requested the opener focus its audit log panel.", func(channel interop.WindowChannel) error {
			return interop.PublishIntent(channel, interop.SurfaceIntentFocusPanel, "audit-log", map[string]string{"tab": "alerts"})
		}, func() {
			latestIntent.Set("focus-panel -> audit-log")
		})
	})

	acknowledgeLogout := ui.UseEvent(func() {
		send("Acknowledged sign-out back to the opener.", func(channel interop.WindowChannel) error {
			return interop.PublishLogout(channel, "Popup acknowledged remote sign-out.")
		}, func() {
			session.Set("signed-out")
		})
	})

	logNodes := make([]ui.Node, 0, len(logs.Get()))
	for _, entry := range logs.Get() {
		logNodes = append(logNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(entry)),
		)
	}

	orphanMessage := "This popup is attached to the opener channel."
	if orphaned.Get() {
		orphanMessage = "The opener disappeared or this page was opened directly. Keep the UI visible, but disable assumptions that the coordinator is still present."
	}

	return shared.ExamplePage(
		"Popup Surface",
		"interop.WindowOpenerChannel",
		"This is the popup side of the same example. It treats the opener as authoritative, listens for session or route signals, and degrades into an orphaned surface when the opener goes away.",
		shared.ExamplePanel("Popup state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Connection", connection.Get()),
				shared.ExampleStat("Session", session.Get()),
				shared.ExampleStat("Route", activeRoute.Get()),
				shared.ExampleStat("Document", activeDocument.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(orphanMessage)),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Focus alerts route in opener", requestOrdersFocus),
				shared.ExampleButton("Select SKU-42", requestSelection),
				shared.ExampleButton("Focus audit log", requestFocusIntent),
				shared.ExampleButton("Ack sign-out", acknowledgeLogout),
			),
		),
		shared.ExamplePanel("Latest opener intent",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Intent", latestIntent.Get()),
				shared.ExampleStat("Orphaned", fmt.Sprintf("%t", orphaned.Get())),
			),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, logNodes...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiWindowExampleRoot), "#app")
	select {}
}
