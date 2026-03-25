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
	parseLocation, parseErr := interop.GetWindowLocation()
	if parseErr == nil {
		parsePath := strings.ToLower(strings.TrimSpace(parseLocation.Pathname()))
		if strings.Contains(parsePath, "popup") {
			return popupSurfaceExample()
		}
	}
	return openerSurfaceExample()
}

func openerSurfaceExample() ui.Node {
	parseSession := ui.UseState("signed-in")
	parseActiveRoute := ui.UseState("/inventory?tab=summary")
	parseActiveDocument := ui.UseState("INV-204")
	parseLatestIntent := ui.UseState("Waiting for popup intent.")
	parsePopupStatus := ui.UseState("Closed")
	parseLogs := ui.UseState([]string{"Open the popup to establish the operator-console channel."})
	parseChannelRef := ui.UseRef(interop.WindowChannel{})
	parseCancelRef := ui.UseRef((func())(nil))

	parseAppendLog := func(parseLine string) {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			return
		}
		parseLogs.Update(func(parsePrevious []string) []string {
			parseNext := append([]string{parseTrimmed}, parsePrevious...)
			if len(parseNext) > 8 {
				parseNext = parseNext[:8]
			}
			return parseNext
		})
	}

	parseDescribeError := func(parsePrefix string, parseErr5 error) string {
		if parseErr5 == nil {
			return parsePrefix
		}
		if parseCode, parseOk := interop.CodeOf(parseErr5); parseOk {
			return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr5)
		}
		return fmt.Sprintf("%s: %v", parsePrefix, parseErr5)
	}

	parseReleaseChannel := func(isCloseWindow bool) {
		if parseCancel := parseCancelRef.Get(); parseCancel != nil {
			parseCancel()
			parseCancelRef.Set(nil)
		}
		parseChannel := parseChannelRef.Get()
		if isCloseWindow && parseChannel.Name() != "" && !parseChannel.Closed() {
			_ = parseChannel.Close()
		}
		parseChannelRef.Set(interop.WindowChannel{})
	}

	handleSignal := func(parseMessage interop.DecodedWindowEnvelope[interop.SurfaceSignal], parseErr6 error) {
		if parseErr6 != nil {
			parseAppendLog(parseDescribeError("Popup message failed", parseErr6))
			return
		}
		switch parseMessage.Payload.Kind {
		case interop.SurfaceSignalSession:
			if parseMessage.Payload.Session == nil {
				parseAppendLog("Popup session message arrived without payload.")
				return
			}
			parseSession.Set(parseMessage.Payload.Session.Status)
			parseAppendLog(fmt.Sprintf("Popup reported session %q: %s", parseMessage.Payload.Session.Status, parseMessage.Payload.Session.Reason))
		case interop.SurfaceSignalRoute:
			if parseMessage.Payload.Route == nil {
				parseAppendLog("Popup route message arrived without payload.")
				return
			}
			parseNextRoute := parseMessage.Payload.Route.Path
			if strings.TrimSpace(parseMessage.Payload.Route.Query) != "" {
				parseNextRoute += "?" + parseMessage.Payload.Route.Query
			}
			parseActiveRoute.Set(parseNextRoute)
			parseAppendLog(fmt.Sprintf("Popup focused route %s", parseNextRoute))
		case interop.SurfaceSignalSelection:
			if parseMessage.Payload.Selection == nil {
				parseAppendLog("Popup selection message arrived without payload.")
				return
			}
			parseActiveDocument.Set(parseMessage.Payload.Selection.ID)
			parseAppendLog(fmt.Sprintf("Popup selected %s %s", parseMessage.Payload.Selection.Scope, parseMessage.Payload.Selection.ID))
		case interop.SurfaceSignalIntent:
			if parseMessage.Payload.Intent == nil {
				parseAppendLog("Popup intent message arrived without payload.")
				return
			}
			parseSummary := fmt.Sprintf("%s -> %s", parseMessage.Payload.Intent.Action, parseMessage.Payload.Intent.Target)
			parseLatestIntent.Set(parseSummary)
			parseAppendLog("Popup requested intent " + parseSummary)
		default:
			parseAppendLog("Popup sent an unknown multi-surface payload.")
		}
	}

	parseSend := func(parseLabel string, parsePublish func(interop.WindowChannel) error, parseAfter func()) {
		parseChannel2 := parseChannelRef.Get()
		if parseChannel2.Name() == "" || parseChannel2.Closed() {
			parsePopupStatus.Set("Closed")
			parseAppendLog("Open the popup before sending a multi-surface signal.")
			return
		}
		if parseErr := parsePublish(parseChannel2); parseErr != nil {
			parseAppendLog(parseDescribeError(parseLabel, parseErr))
			return
		}
		if parseAfter != nil {
			parseAfter()
		}
		parseAppendLog(parseLabel)
	}

	parseOpenPopup := ui.UseEvent(func() {
		parseCurrent := parseChannelRef.Get()
		if parseCurrent.Name() != "" && !parseCurrent.Closed() {
			_ = parseCurrent.Focus()
			parsePopupStatus.Set("Connected")
			parseAppendLog("Reused the existing popup and focused it.")
			return
		}

		parseReleaseChannel(false)
		parseChannel3, parseErr2 := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
			URL:      multiWindowPopupURL,
			Name:     multiWindowChannelName,
			Features: "popup=yes,width=560,height=760",
		})
		if parseErr2 != nil {
			parseAppendLog(parseDescribeError("Opening popup failed", parseErr2))
			return
		}
		parseSubscription, parseErr2 := interop.SubscribeSurfaceSignals(parseChannel3, handleSignal)
		if parseErr2 != nil {
			_ = parseChannel3.Close()
			parseAppendLog(parseDescribeError("Popup subscription failed", parseErr2))
			return
		}
		parseChannelRef.Set(parseChannel3)
		parseCancelRef.Set(parseSubscription.Cancel)
		parsePopupStatus.Set("Connected")
		parseAppendLog("Opened the popup inspector and subscribed to its surface signals.")
	})

	parseFocusPopup := ui.UseEvent(func() {
		parseSend("Focused the popup window.", func(parseChannel6 interop.WindowChannel) error {
			return parseChannel6.Focus()
		}, nil)
	})

	parseClosePopup := ui.UseEvent(func() {
		parseChannel4 := parseChannelRef.Get()
		if parseChannel4.Name() == "" || parseChannel4.Closed() {
			parsePopupStatus.Set("Closed")
			parseAppendLog("Popup is already closed.")
			parseReleaseChannel(false)
			return
		}
		if parseErr3 := parseChannel4.Close(); parseErr3 != nil {
			parseAppendLog(parseDescribeError("Closing popup failed", parseErr3))
			return
		}
		parseReleaseChannel(false)
		parsePopupStatus.Set("Closed")
		parseAppendLog("Closed the popup from the opener.")
	})

	parseSendSessionExpired := ui.UseEvent(func() {
		parseExpiresAt := time.Now().UTC().Add(15 * time.Minute).Round(time.Second)
		parseSend("Sent a session-expired signal to the popup.", func(parseChannel7 interop.WindowChannel) error {
			return interop.PublishSessionExpired(parseChannel7, "Re-authentication required in every active surface.", "/login", parseExpiresAt)
		}, func() {
			parseSession.Set("expired")
		})
	})

	parseSendLogout := ui.UseEvent(func() {
		parseSend("Broadcast a logout signal to the popup.", func(parseChannel8 interop.WindowChannel) error {
			return interop.PublishLogout(parseChannel8, "Operator signed out in the main workspace.")
		}, func() {
			parseSession.Set("signed-out")
		})
	})

	parseSendRouteFocus := ui.UseEvent(func() {
		parseSend("Sent route focus for /orders/42.", func(parseChannel9 interop.WindowChannel) error {
			return interop.PublishRouteFocus(parseChannel9, "/orders/42", "tab=activity", "order-heading")
		}, func() {
			parseActiveRoute.Set("/orders/42?tab=activity")
		})
	})

	parseSendSelection := ui.UseEvent(func() {
		parseSend("Sent active-document selection INV-204.", func(parseChannel10 interop.WindowChannel) error {
			return interop.PublishSelection(parseChannel10, "invoice", "INV-204", "rev-12")
		}, func() {
			parseActiveDocument.Set("INV-204")
		})
	})

	parseSendIntent := ui.UseEvent(func() {
		parseSend("Sent a focus-panel intent to the popup.", func(parseChannel11 interop.WindowChannel) error {
			return interop.PublishIntent(parseChannel11, interop.SurfaceIntentFocusPanel, "inventory-inspector", map[string]string{"tab": "activity"})
		}, func() {
			parseLatestIntent.Set("focus-panel -> inventory-inspector")
		})
	})

	ui.UseEffect(func() func() {
		parseTimer, parseErr4 := interop.ScheduleInterval(500*time.Millisecond, func() {
			parseChannel5 := parseChannelRef.Get()
			if parseChannel5.Name() == "" {
				return
			}
			if parseChannel5.Closed() {
				parseReleaseChannel(false)
				parsePopupStatus.Set("Closed")
				parseAppendLog("Popup closed unexpectedly. Reopen it to restore coordination.")
			}
		})
		return func() {
			if parseErr4 == nil {
				_ = parseTimer.Cancel()
			}
			parseReleaseChannel(true)
		}
	}, true)

	parseLogNodes := make([]ui.Node, 0, len(parseLogs.Get()))
	for _, parseEntry := range parseLogs.Get() {
		parseLogNodes = append(parseLogNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseEntry)),
		)
	}

	return shared.ExamplePage(
		"Multi-Window Console",
		"interop.WindowChannel and multi-surface signals",
		"Open a dedicated operator popup, propagate session and route context, and treat popup loss as ordinary recoverable state instead of a fatal runtime event.",
		shared.ExamplePanel("Opener controls",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Session", parseSession.Get()),
				shared.ExampleStat("Route", parseActiveRoute.Get()),
				shared.ExampleStat("Document", parseActiveDocument.Get()),
				shared.ExampleStat("Popup", parsePopupStatus.Get()),
			),
			html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
				shared.ExampleButton("Open popup", parseOpenPopup),
				shared.ExampleButton("Focus popup", parseFocusPopup),
				shared.ExampleButton("Close popup", parseClosePopup),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Expire session", parseSendSessionExpired),
				shared.ExampleButton("Send logout", parseSendLogout),
				shared.ExampleButton("Focus /orders/42", parseSendRouteFocus),
				shared.ExampleButton("Select INV-204", parseSendSelection),
				shared.ExampleButton("Focus inspector panel", parseSendIntent),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The opener remains the authoritative owner for popup lifecycle. Signals are typed and explicit, not implicit global state replication.")),
		),
		shared.ExamplePanel("Popup feedback",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Latest intent", parseLatestIntent.Get()),
				shared.ExampleStat("Channel name", multiWindowChannelName),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("If the popup selects a different document or requests a focus change, those signals come back through the same typed channel and update the opener view here.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, parseLogNodes...),
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
	parseSession := ui.UseState("waiting")
	parseActiveRoute := ui.UseState("Waiting for opener route focus.")
	parseActiveDocument := ui.UseState("No active document")
	parseLatestIntent := ui.UseState("Waiting for opener intent.")
	parseConnection := ui.UseState("Connecting to opener...")
	parseOrphaned := ui.UseState(false)
	parseLogs := ui.UseState([]string{"This popup is waiting for the opener channel."})
	parseChannelRef := ui.UseRef(interop.WindowChannel{})
	parseCancelRef := ui.UseRef((func())(nil))

	parseAppendLog := func(parseLine string) {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			return
		}
		parseLogs.Update(func(parsePrevious []string) []string {
			parseNext := append([]string{parseTrimmed}, parsePrevious...)
			if len(parseNext) > 8 {
				parseNext = parseNext[:8]
			}
			return parseNext
		})
	}

	parseDescribeError := func(parsePrefix string, parseErr3 error) string {
		if parseErr3 == nil {
			return parsePrefix
		}
		if parseCode, parseOk := interop.CodeOf(parseErr3); parseOk {
			return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr3)
		}
		return fmt.Sprintf("%s: %v", parsePrefix, parseErr3)
	}

	parseSend := func(parseLabel string, parsePublish func(interop.WindowChannel) error, parseAfter func()) {
		parseChannel := parseChannelRef.Get()
		if parseChannel.Name() == "" || parseChannel.Closed() {
			parseOrphaned.Set(true)
			parseConnection.Set("Opener unavailable")
			parseAppendLog("Opener is unavailable; this popup is now orphaned.")
			return
		}
		if parseErr := parsePublish(parseChannel); parseErr != nil {
			parseAppendLog(parseDescribeError(parseLabel, parseErr))
			return
		}
		if parseAfter != nil {
			parseAfter()
		}
		parseAppendLog(parseLabel)
	}

	handleSignal := func(parseMessage interop.DecodedWindowEnvelope[interop.SurfaceSignal], parseErr4 error) {
		if parseErr4 != nil {
			parseAppendLog(parseDescribeError("Opener message failed", parseErr4))
			return
		}
		switch parseMessage.Payload.Kind {
		case interop.SurfaceSignalSession:
			if parseMessage.Payload.Session == nil {
				return
			}
			parseSession.Set(parseMessage.Payload.Session.Status)
			parseConnection.Set("Connected")
			parseOrphaned.Set(false)
			parseAppendLog(fmt.Sprintf("Received session %q from opener.", parseMessage.Payload.Session.Status))
		case interop.SurfaceSignalRoute:
			if parseMessage.Payload.Route == nil {
				return
			}
			parseNextRoute := parseMessage.Payload.Route.Path
			if strings.TrimSpace(parseMessage.Payload.Route.Query) != "" {
				parseNextRoute += "?" + parseMessage.Payload.Route.Query
			}
			parseActiveRoute.Set(parseNextRoute)
			parseAppendLog("Received route focus " + parseNextRoute)
		case interop.SurfaceSignalSelection:
			if parseMessage.Payload.Selection == nil {
				return
			}
			parseActiveDocument.Set(parseMessage.Payload.Selection.ID)
			parseAppendLog(fmt.Sprintf("Received active %s %s", parseMessage.Payload.Selection.Scope, parseMessage.Payload.Selection.ID))
		case interop.SurfaceSignalIntent:
			if parseMessage.Payload.Intent == nil {
				return
			}
			parseLatestIntent.Set(fmt.Sprintf("%s -> %s", parseMessage.Payload.Intent.Action, parseMessage.Payload.Intent.Target))
			parseAppendLog(fmt.Sprintf("Received opener intent %s", parseMessage.Payload.Intent.Action))
		}
	}

	ui.UseEffect(func() func() {
		parseChannel2, parseErr2 := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: multiWindowChannelName})
		if parseErr2 != nil {
			parseOrphaned.Set(true)
			parseConnection.Set("Opened without an opener")
			parseAppendLog(parseDescribeError("No opener channel available", parseErr2))
			return nil
		}
		parseChannelRef.Set(parseChannel2)
		parseConnection.Set("Connected")
		parseOrphaned.Set(parseChannel2.Closed())

		parseSubscription, parseErr2 := interop.SubscribeSurfaceSignals(parseChannel2, handleSignal)
		if parseErr2 != nil {
			parseConnection.Set("Subscription failed")
			parseAppendLog(parseDescribeError("Popup subscription failed", parseErr2))
			return nil
		}
		parseCancelRef.Set(parseSubscription.Cancel)
		parseAppendLog("Connected to the opener and subscribed to surface signals.")

		parseTimer, parseTimerErr := interop.ScheduleInterval(500*time.Millisecond, func() {
			parseCurrent := parseChannelRef.Get()
			if parseCurrent.Name() == "" {
				return
			}
			if parseCurrent.Closed() {
				parseOrphaned.Set(true)
				parseConnection.Set("Opener disconnected")
			}
		})

		return func() {
			if parseCancel := parseCancelRef.Get(); parseCancel != nil {
				parseCancel()
				parseCancelRef.Set(nil)
			}
			if parseTimerErr == nil {
				_ = parseTimer.Cancel()
			}
		}
	}, true)

	parseRequestOrdersFocus := ui.UseEvent(func() {
		parseSend("Requested route focus back to /inventory/alerts.", func(parseChannel3 interop.WindowChannel) error {
			return interop.PublishRouteFocus(parseChannel3, "/inventory/alerts", "panel=exceptions", "alerts-heading")
		}, func() {
			parseActiveRoute.Set("/inventory/alerts?panel=exceptions")
		})
	})

	parseRequestSelection := ui.UseEvent(func() {
		parseSend("Selected SKU-42 in the popup.", func(parseChannel4 interop.WindowChannel) error {
			return interop.PublishSelection(parseChannel4, "sku", "SKU-42", "popup-rev-3")
		}, func() {
			parseActiveDocument.Set("SKU-42")
		})
	})

	parseRequestFocusIntent := ui.UseEvent(func() {
		parseSend("Requested the opener focus its audit log panel.", func(parseChannel5 interop.WindowChannel) error {
			return interop.PublishIntent(parseChannel5, interop.SurfaceIntentFocusPanel, "audit-log", map[string]string{"tab": "alerts"})
		}, func() {
			parseLatestIntent.Set("focus-panel -> audit-log")
		})
	})

	parseAcknowledgeLogout := ui.UseEvent(func() {
		parseSend("Acknowledged sign-out back to the opener.", func(parseChannel6 interop.WindowChannel) error {
			return interop.PublishLogout(parseChannel6, "Popup acknowledged remote sign-out.")
		}, func() {
			parseSession.Set("signed-out")
		})
	})

	parseLogNodes := make([]ui.Node, 0, len(parseLogs.Get()))
	for _, parseEntry := range parseLogs.Get() {
		parseLogNodes = append(parseLogNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseEntry)),
		)
	}

	parseOrphanMessage := "This popup is attached to the opener channel."
	if parseOrphaned.Get() {
		parseOrphanMessage = "The opener disappeared or this page was opened directly. Keep the UI visible, but disable assumptions that the coordinator is still present."
	}

	return shared.ExamplePage(
		"Popup Surface",
		"interop.WindowOpenerChannel",
		"This is the popup side of the same example. It treats the opener as authoritative, listens for session or route signals, and degrades into an orphaned surface when the opener goes away.",
		shared.ExamplePanel("Popup state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Connection", parseConnection.Get()),
				shared.ExampleStat("Session", parseSession.Get()),
				shared.ExampleStat("Route", parseActiveRoute.Get()),
				shared.ExampleStat("Document", parseActiveDocument.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseOrphanMessage)),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Focus alerts route in opener", parseRequestOrdersFocus),
				shared.ExampleButton("Select SKU-42", parseRequestSelection),
				shared.ExampleButton("Focus audit log", parseRequestFocusIntent),
				shared.ExampleButton("Ack sign-out", parseAcknowledgeLogout),
			),
		),
		shared.ExamplePanel("Latest opener intent",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Intent", parseLatestIntent.Get()),
				shared.ExampleStat("Orphaned", fmt.Sprintf("%t", parseOrphaned.Get())),
			),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, parseLogNodes...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiWindowExampleRoot), "#app")
	select {}
}
