//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

const (
	multiClientBinaryChannelName = "example:multi-client:binary"
	multiClientBinaryPopupName   = "example:multi-client:binary-popup"
	multiClientBinaryPopupURL    = "./multi-client-binary-popup.html"
	binaryPreviewTopic           = "asset:preview"
	binaryAckTopic               = "asset:preview-ack"
	binaryContentType            = "application/octet-stream"
)

func newBinaryIdentity(parseSurface string, parseRole string) interop.ClientIdentity {
	return interop.ClientIdentity{
		ID:      fmt.Sprintf("%s-%d", strings.TrimSpace(parseSurface), time.Now().UTC().UnixNano()),
		App:     "examples/multi-client-binary",
		Surface: strings.TrimSpace(parseSurface),
		Role:    strings.TrimSpace(parseRole),
		Version: "v1",
	}
}

func describeBinaryError(parsePrefix string, parseErr error) string {
	if parseErr == nil {
		return parsePrefix
	}
	if parseCode, parseOk := interop.CodeOf(parseErr); parseOk {
		return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr)
	}
	return fmt.Sprintf("%s: %v", parsePrefix, parseErr)
}

func appendBinaryLog(parseLogs ui.State[[]string], parseLine string) {
	parseTrimmed := strings.TrimSpace(parseLine)
	if parseTrimmed == "" {
		return
	}
	parseLogs.Update(func(parsePrevious []string) []string {
		parseNext := append([]string{parseTrimmed}, parsePrevious...)
		if len(parseNext) > 10 {
			parseNext = parseNext[:10]
		}
		return parseNext
	})
}

func previewBytes() []byte {
	return []byte{0x47, 0x57, 0x43, 0x00, 0x13, 0x37, 0x42, 0x99, 0x10, 0xAF, 0xC0, 0xDE}
}

func previewSummary(parsePayload any) string {
	parseBytes, parseOk := parsePayload.([]byte)
	if !parseOk {
		return fmt.Sprintf("non-binary payload: %v", parsePayload)
	}
	parseLimit := len(parseBytes)
	if parseLimit > 6 {
		parseLimit = 6
	}
	parseParts := make([]string, 0, parseLimit)
	for _, parseValue := range parseBytes[:parseLimit] {
		parseParts = append(parseParts, fmt.Sprintf("%02X", parseValue))
	}
	return fmt.Sprintf("%d bytes [%s]", len(parseBytes), strings.Join(parseParts, " "))
}

func renderBinaryLogs(parseEntries []string) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseNodes = append(parseNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseEntry)),
		)
	}
	return parseNodes
}

func multiClientBinaryRoot() ui.Node {
	parseLocation, parseErr := interop.GetWindowLocation()
	if parseErr == nil {
		parsePath := strings.ToLower(strings.TrimSpace(parseLocation.Pathname()))
		if strings.Contains(parsePath, "popup") {
			return multiClientBinaryPopupSurface()
		}
	}
	return multiClientBinaryOpenerSurface()
}

func multiClientBinaryOpenerSurface() ui.Node {
	parseSelf := ui.UseState(newBinaryIdentity("catalog-tab", "storefront"))
	parseCrossTabTransport := ui.UseState("pending")
	parseCrossTabMode := ui.UseState("No cross-tab preview sent yet.")
	parseCrossTabReceived := ui.UseState("Waiting for another tab or a fallback event.")
	parsePopupStatus := ui.UseState("Closed")
	parsePopupPeerID := ui.UseState("")
	parsePopupAck := ui.UseState("No popup ack yet.")
	parseLogs := ui.UseState([]string{"Open this example in a second tab for cross-tab preview delivery, then open the popup to watch JSON handshake plus binary preview flow."})
	parseCrossTabRef := ui.UseRef(interop.CrossTabChannel{})
	parseCrossTabCancelRef := ui.UseRef((func())(nil))
	parsePopupRef := ui.UseRef(interop.WindowChannel{})
	parsePopupCancelRef := ui.UseRef((func())(nil))

	handleCrossTabMessage := func(parseMessage interop.ClientMessage, parseErr9 error) {
		if parseErr9 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Cross-tab message failed", parseErr9))
			return
		}
		if parseMessage.Source.ID == parseSelf.Get().ID {
			return
		}
		if parseMessage.Kind == interop.ClientHello {
			appendBinaryLog(parseLogs, fmt.Sprintf("Peer hello from %s on %s", parseMessage.Source.Surface, parseCrossTabTransport.Get()))
			return
		}
		if parseMessage.Kind != interop.ClientEvent || parseMessage.Topic != binaryPreviewTopic {
			return
		}
		if parseMessage.Encoding == interop.ClientPayloadBinary {
			parseCrossTabReceived.Set("Binary preview received from another tab: " + previewSummary(parseMessage.Payload))
			parseCrossTabMode.Set("Last cross-tab delivery used binary over " + parseCrossTabTransport.Get())
			appendBinaryLog(parseLogs, fmt.Sprintf("Received binary preview from %s: %s", parseMessage.Source.Surface, previewSummary(parseMessage.Payload)))
			return
		}
		parseCrossTabReceived.Set(fmt.Sprintf("JSON fallback event from another tab: %v", parseMessage.Payload))
		parseCrossTabMode.Set("Last cross-tab delivery fell back to JSON metadata")
		appendBinaryLog(parseLogs, fmt.Sprintf("Received JSON fallback from %s because the transport could not carry binary.", parseMessage.Source.Surface))
	}

	handlePopupMessage := func(_ interop.WindowChannel, parseMessage2 interop.ClientMessage, parseErr10 error) {
		if parseErr10 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Popup message failed", parseErr10))
			return
		}
		if parseMessage2.Source.ID == parseSelf.Get().ID {
			return
		}
		switch parseMessage2.Kind {
		case interop.ClientHello:
			parsePopupPeerID.Set(parseMessage2.Source.ID)
			parsePopupStatus.Set("Connected")
			appendBinaryLog(parseLogs, fmt.Sprintf("Popup hello from %s", parseMessage2.Source.Surface))
		case interop.ClientResult:
			if parseMessage2.Topic != binaryAckTopic || parseMessage2.Target != parseSelf.Get().ID {
				return
			}
			parsePopupAck.Set(fmt.Sprintf("Popup ack: %v", parseMessage2.Payload))
			appendBinaryLog(parseLogs, fmt.Sprintf("Popup acknowledged binary preview with %v", parseMessage2.Payload))
		case interop.ClientGoodbye:
			parsePopupStatus.Set("Disconnected")
			parsePopupPeerID.Set("")
			appendBinaryLog(parseLogs, "Popup sent goodbye and closed its targeted channel.")
		}
	}

	parseReleasePopup := func(isCloseWindow bool) {
		if parseCancel := parsePopupCancelRef.Get(); parseCancel != nil {
			parseCancel()
			parsePopupCancelRef.Set(nil)
		}
		parseChannel := parsePopupRef.Get()
		if isCloseWindow && parseChannel.Name() != "" && !parseChannel.Closed() {
			_ = interop.PublishClientGoodbyeWindow(parseChannel, parseSelf.Get())
			_ = parseChannel.Close()
		}
		parsePopupRef.Set(interop.WindowChannel{})
		parsePopupPeerID.Set("")
	}

	ui.UseEffect(func() func() {
		parseChannel2, parseErr := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: multiClientBinaryChannelName})
		if parseErr != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Cross-tab channel unavailable", parseErr))
			parseCrossTabTransport.Set("unavailable")
			return nil
		}
		parseCrossTabRef.Set(parseChannel2)
		parseCrossTabTransport.Set(parseChannel2.Transport())

		parseSubscription, parseErr := interop.SubscribeClientMessages(parseChannel2, handleCrossTabMessage)
		if parseErr != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Cross-tab subscription failed", parseErr))
			_ = parseChannel2.Close()
			parseCrossTabRef.Set(interop.CrossTabChannel{})
			return nil
		}
		parseCrossTabCancelRef.Set(parseSubscription.Cancel)

		if parseErr2 := interop.PublishClientHello(parseChannel2, parseSelf.Get()); parseErr2 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Cross-tab hello failed", parseErr2))
		} else {
			appendBinaryLog(parseLogs, "Published JSON hello on the cross-tab control plane.")
		}

		return func() {
			if parseCancel2 := parseCrossTabCancelRef.Get(); parseCancel2 != nil {
				parseCancel2()
				parseCrossTabCancelRef.Set(nil)
			}
			_ = interop.PublishClientGoodbye(parseChannel2, parseSelf.Get())
			_ = parseChannel2.Close()
			parseCrossTabRef.Set(interop.CrossTabChannel{})
		}
	}, true)

	ui.UseEffect(func() func() {
		parseTimer, parseErr3 := interop.ScheduleInterval(500*time.Millisecond, func() {
			parseChannel3 := parsePopupRef.Get()
			if parseChannel3.Name() == "" || !parseChannel3.Closed() {
				return
			}
			parseReleasePopup(false)
			parsePopupStatus.Set("Closed")
			appendBinaryLog(parseLogs, "Popup closed unexpectedly. Reopen it to restore targeted binary preview delivery.")
		})
		return func() {
			if parseErr3 == nil {
				_ = parseTimer.Cancel()
			}
			parseReleasePopup(true)
		}
	}, true)

	parseBroadcastPreview := ui.UseEvent(func() {
		parseChannel4 := parseCrossTabRef.Get()
		if parseChannel4.Name() == "" {
			appendBinaryLog(parseLogs, "Cross-tab channel is not ready yet.")
			return
		}
		parsePayload := previewBytes()
		if parseChannel4.Transport() == "broadcast-channel" {
			if parseErr4 := interop.PublishClientBinaryCrossTab(parseChannel4, binaryPreviewTopic, parseSelf.Get(), interop.ClientBinaryPayload{ContentType: binaryContentType, Bytes: parsePayload}); parseErr4 != nil {
				appendBinaryLog(parseLogs, describeBinaryError("Binary cross-tab publish failed", parseErr4))
				return
			}
			parseCrossTabMode.Set("Sent binary preview over BroadcastChannel")
			appendBinaryLog(parseLogs, "Published binary preview to other tabs through BroadcastChannel.")
			return
		}
		parseFallback := map[string]any{
			"fallback":    "json",
			"contentType": binaryContentType,
			"byteCount":   len(parsePayload),
			"reason":      "resolved cross-tab transport cannot carry binary",
		}
		if parseErr5 := interop.PublishClientEvent(parseChannel4, binaryPreviewTopic, parseSelf.Get(), parseFallback); parseErr5 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("JSON fallback publish failed", parseErr5))
			return
		}
		parseCrossTabMode.Set("Sent JSON fallback because the resolved transport was " + parseChannel4.Transport())
		appendBinaryLog(parseLogs, "Cross-tab transport could not carry binary, so the example sent JSON metadata instead.")
	})

	parseOpenPopup := ui.UseEvent(func() {
		parseCurrent := parsePopupRef.Get()
		if parseCurrent.Name() != "" && !parseCurrent.Closed() {
			_ = parseCurrent.Focus()
			parsePopupStatus.Set("Connected")
			appendBinaryLog(parseLogs, "Reused the existing popup and focused it.")
			return
		}

		parseReleasePopup(false)
		parseChannel5, parseErr6 := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
			URL:      multiClientBinaryPopupURL,
			Name:     multiClientBinaryPopupName,
			Features: "popup=yes,width=560,height=760",
		})
		if parseErr6 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Opening popup failed", parseErr6))
			return
		}
		parseSubscription2, parseErr6 := interop.SubscribeClientWindowMessages(parseChannel5, func(parseMessage3 interop.ClientMessage, parseReceiveErr error) {
			handlePopupMessage(parseChannel5, parseMessage3, parseReceiveErr)
		})
		if parseErr6 != nil {
			_ = parseChannel5.Close()
			appendBinaryLog(parseLogs, describeBinaryError("Popup subscription failed", parseErr6))
			return
		}
		parsePopupRef.Set(parseChannel5)
		parsePopupCancelRef.Set(parseSubscription2.Cancel)
		parsePopupStatus.Set("Connecting")

		if parseErr7 := interop.PublishClientHelloWindow(parseChannel5, parseSelf.Get()); parseErr7 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Popup hello failed", parseErr7))
			return
		}
		appendBinaryLog(parseLogs, "Sent JSON hello to the popup before any binary payload traffic.")
	})

	parseSendPopupPreview := ui.UseEvent(func() {
		parseChannel6 := parsePopupRef.Get()
		if parseChannel6.Name() == "" || parseChannel6.Closed() {
			appendBinaryLog(parseLogs, "Open the popup before sending a targeted binary preview.")
			parsePopupStatus.Set("Closed")
			return
		}
		parseTarget := strings.TrimSpace(parsePopupPeerID.Get())
		if parseTarget == "" {
			appendBinaryLog(parseLogs, "Waiting for popup hello before sending the targeted binary preview.")
			return
		}
		if parseErr8 := interop.PublishClientBinaryWindow(parseChannel6, binaryPreviewTopic, parseSelf.Get(), parseTarget, interop.ClientBinaryPayload{ContentType: binaryContentType, Bytes: previewBytes()}); parseErr8 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Popup binary publish failed", parseErr8))
			return
		}
		parsePopupStatus.Set("Preview sent")
		appendBinaryLog(parseLogs, "Published binary preview to the popup, with JSON ack expected back on the control plane.")
	})

	parseClosePopup := ui.UseEvent(func() {
		parseChannel7 := parsePopupRef.Get()
		if parseChannel7.Name() == "" || parseChannel7.Closed() {
			parsePopupStatus.Set("Closed")
			appendBinaryLog(parseLogs, "Popup is already closed.")
			parseReleasePopup(false)
			return
		}
		parseReleasePopup(true)
		parsePopupStatus.Set("Closed")
		appendBinaryLog(parseLogs, "Closed the popup after targeted preview flow completed.")
	})

	return shared.ExamplePage(
		"Multi-Client Binary",
		"interop mixed JSON and binary client messages",
		"Use JSON for handshake and acknowledgement traffic, use binary only for payload-heavy preview delivery on supported transports, and fall back to JSON metadata when the resolved cross-tab transport cannot carry bytes.",
		shared.ExamplePanel("Cross-tab preview",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", parseSelf.Get().Surface),
				shared.ExampleStat("Role", parseSelf.Get().Role),
				shared.ExampleStat("Transport", parseCrossTabTransport.Get()),
				shared.ExampleStat("Last mode", parseCrossTabMode.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Open this example in a second tab. The control plane stays JSON, but the preview payload uses binary only when the resolved cross-tab transport is BroadcastChannel. If the browser falls back to storage events, the example publishes JSON metadata instead of pretending bytes are portable there.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Broadcast preview", parseBroadcastPreview),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseCrossTabReceived.Get())),
		),
		shared.ExamplePanel("Popup preview",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Popup", parsePopupStatus.Get()),
				shared.ExampleStat("Popup peer", parsePopupPeerID.Get()),
				shared.ExampleStat("Ack", parsePopupAck.Get()),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Open popup", parseOpenPopup),
				shared.ExampleButton("Send popup preview", parseSendPopupPreview),
				shared.ExampleButton("Close popup", parseClosePopup),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The popup channel demonstrates the mixed control/data plane directly: JSON hello and result frames wrap the lifecycle, while the preview bytes themselves travel as structured-clone binary.")),
		),
		shared.ExamplePanel("Diagnostics",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("The log records whether the example used JSON or binary, whether the popup handshake completed, and when the cross-tab flow had to degrade because the resolved transport could not carry raw bytes.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, renderBinaryLogs(parseLogs.Get())...),
		),
		shared.ExamplePanel("Integration shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The intended pattern is simple: keep presence, ack, timeout, and authority traffic JSON-shaped; reserve binary for the payload-heavy topic itself; and treat unsupported transport as a normal fallback decision rather than a surprise runtime failure.")),
			shared.ExampleCode(
				`if channel.Transport() == "broadcast-channel" { _ = interop.PublishClientBinaryCrossTab(channel, "asset:preview", self, payload) } else { _ = interop.PublishClientEvent(channel, "asset:preview", self, fallbackMetadata) }`,
				`_ = interop.PublishClientHelloWindow(windowChannel, self)`,
				`_ = interop.PublishClientBinaryWindow(windowChannel, "asset:preview", self, popupPeerID, payload)`,
				`_ = interop.PublishClientWindowMessage(windowChannel, interop.ClientMessage{Kind: interop.ClientResult, Topic: "asset:preview-ack", Source: popupSelf, Target: openerID, Payload: ack})`,
			),
		),
	)
}

func multiClientBinaryPopupSurface() ui.Node {
	parseSelf := ui.UseState(newBinaryIdentity("popup-preview", "inspector"))
	parseOpenerID := ui.UseState("")
	parseConnection := ui.UseState("Connecting to opener...")
	parseLastPreview := ui.UseState("No preview received yet.")
	parseOrphaned := ui.UseState(false)
	parseLogs := ui.UseState([]string{"This popup waits for a JSON hello, then accepts targeted binary preview traffic from its opener."})
	parseChannelRef := ui.UseRef(interop.WindowChannel{})
	parseCancelRef := ui.UseRef((func())(nil))

	handlePopupMessage := func(parseChannel2 interop.WindowChannel, parseMessage interop.ClientMessage, parseErr3 error) {
		if parseErr3 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Opener message failed", parseErr3))
			return
		}
		if parseMessage.Source.ID == parseSelf.Get().ID {
			return
		}
		switch parseMessage.Kind {
		case interop.ClientHello:
			parseOpenerID.Set(parseMessage.Source.ID)
			parseConnection.Set("Connected")
			parseOrphaned.Set(false)
			appendBinaryLog(parseLogs, fmt.Sprintf("Received JSON hello from %s", parseMessage.Source.Surface))
		case interop.ClientEvent:
			if parseMessage.Topic != binaryPreviewTopic || parseMessage.Target != parseSelf.Get().ID || parseMessage.Encoding != interop.ClientPayloadBinary {
				return
			}
			parseLastPreview.Set(fmt.Sprintf("Binary preview received: %s (%s)", previewSummary(parseMessage.Payload), parseMessage.ContentType))
			appendBinaryLog(parseLogs, fmt.Sprintf("Received targeted binary preview from %s: %s", parseMessage.Source.Surface, previewSummary(parseMessage.Payload)))
			parseAck := interop.ClientMessage{
				Kind:   interop.ClientResult,
				Topic:  binaryAckTopic,
				Source: parseSelf.Get(),
				Target: parseMessage.Source.ID,
				Payload: map[string]any{
					"contentType": parseMessage.ContentType,
					"summary":     previewSummary(parseMessage.Payload),
				},
			}
			if parsePublishErr := interop.PublishClientWindowMessage(parseChannel2, parseAck); parsePublishErr != nil {
				appendBinaryLog(parseLogs, describeBinaryError("Popup ack failed", parsePublishErr))
				return
			}
			appendBinaryLog(parseLogs, "Sent JSON ack back to the opener after binary preview receipt.")
		case interop.ClientGoodbye:
			parseOrphaned.Set(true)
			parseConnection.Set("Opener disconnected")
			appendBinaryLog(parseLogs, "Opener sent goodbye. The popup is now orphaned.")
		}
	}

	ui.UseEffect(func() func() {
		parseChannel, parseErr := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: multiClientBinaryPopupName})
		if parseErr != nil {
			parseOrphaned.Set(true)
			parseConnection.Set("Opened without an opener")
			appendBinaryLog(parseLogs, describeBinaryError("No opener channel available", parseErr))
			return nil
		}
		parseChannelRef.Set(parseChannel)
		parseConnection.Set("Connected")
		parseOrphaned.Set(parseChannel.Closed())

		parseSubscription, parseErr := interop.SubscribeClientWindowMessages(parseChannel, func(parseMessage2 interop.ClientMessage, parseReceiveErr error) {
			handlePopupMessage(parseChannel, parseMessage2, parseReceiveErr)
		})
		if parseErr != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Popup subscription failed", parseErr))
			parseConnection.Set("Subscription failed")
			return nil
		}
		parseCancelRef.Set(parseSubscription.Cancel)

		if parseErr2 := interop.PublishClientHelloWindow(parseChannel, parseSelf.Get()); parseErr2 != nil {
			appendBinaryLog(parseLogs, describeBinaryError("Popup hello failed", parseErr2))
		} else {
			appendBinaryLog(parseLogs, "Published JSON hello to the opener control plane.")
		}

		parseTimer, parseTimerErr := interop.ScheduleInterval(500*time.Millisecond, func() {
			parseCurrent := parseChannelRef.Get()
			if parseCurrent.Name() == "" || !parseCurrent.Closed() {
				return
			}
			parseOrphaned.Set(true)
			parseConnection.Set("Opener disconnected")
		})

		return func() {
			if parseCancel := parseCancelRef.Get(); parseCancel != nil {
				parseCancel()
				parseCancelRef.Set(nil)
			}
			if parseTimerErr == nil {
				_ = parseTimer.Cancel()
			}
			_ = interop.PublishClientGoodbyeWindow(parseChannel, parseSelf.Get())
		}
	}, true)

	parseOrphanMessage := "The opener is still available for targeted preview traffic."
	if parseOrphaned.Get() {
		parseOrphanMessage = "The opener disappeared or this page was opened directly. Keep the diagnostic view visible, but do not assume the coordinator is still present."
	}

	return shared.ExamplePage(
		"Popup Binary Preview",
		"interop.OpenWindowOpenerChannel with binary payloads",
		"This popup accepts a targeted binary preview over the window channel and replies with a JSON acknowledgement so the control plane remains easy to inspect.",
		shared.ExamplePanel("Popup state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", parseSelf.Get().Surface),
				shared.ExampleStat("Role", parseSelf.Get().Role),
				shared.ExampleStat("Connection", parseConnection.Get()),
				shared.ExampleStat("Opener id", parseOpenerID.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseOrphanMessage)),
		),
		shared.ExamplePanel("Latest preview",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Preview", parseLastPreview.Get()),
				shared.ExampleStat("Orphaned", fmt.Sprintf("%t", parseOrphaned.Get())),
			),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, renderBinaryLogs(parseLogs.Get())...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(multiClientBinaryRoot))
	exampleboot.WaitExampleRuntime()
}
