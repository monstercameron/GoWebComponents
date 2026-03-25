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
	multiClientBinaryChannelName = "example:multi-client:binary"
	multiClientBinaryPopupName   = "example:multi-client:binary-popup"
	multiClientBinaryPopupURL    = "./multi-client-binary-popup.html"
	binaryPreviewTopic           = "asset:preview"
	binaryAckTopic               = "asset:preview-ack"
	binaryContentType            = "application/octet-stream"
)

func newBinaryIdentity(surface string, role string) interop.ClientIdentity {
	return interop.ClientIdentity{
		ID:      fmt.Sprintf("%s-%d", strings.TrimSpace(surface), time.Now().UTC().UnixNano()),
		App:     "examples/multi-client-binary",
		Surface: strings.TrimSpace(surface),
		Role:    strings.TrimSpace(role),
		Version: "v1",
	}
}

func describeBinaryError(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	if code, ok := interop.CodeOf(err); ok {
		return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

func appendBinaryLog(logs ui.State[[]string], line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	logs.Update(func(previous []string) []string {
		next := append([]string{trimmed}, previous...)
		if len(next) > 10 {
			next = next[:10]
		}
		return next
	})
}

func previewBytes() []byte {
	return []byte{0x47, 0x57, 0x43, 0x00, 0x13, 0x37, 0x42, 0x99, 0x10, 0xAF, 0xC0, 0xDE}
}

func previewSummary(payload any) string {
	bytes, ok := payload.([]byte)
	if !ok {
		return fmt.Sprintf("non-binary payload: %v", payload)
	}
	limit := len(bytes)
	if limit > 6 {
		limit = 6
	}
	parts := make([]string, 0, limit)
	for _, value := range bytes[:limit] {
		parts = append(parts, fmt.Sprintf("%02X", value))
	}
	return fmt.Sprintf("%d bytes [%s]", len(bytes), strings.Join(parts, " "))
}

func renderBinaryLogs(entries []string) []ui.Node {
	nodes := make([]ui.Node, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(entry)),
		)
	}
	return nodes
}

func multiClientBinaryRoot() ui.Node {
	location, err := interop.GetWindowLocation()
	if err == nil {
		path := strings.ToLower(strings.TrimSpace(location.Pathname()))
		if strings.Contains(path, "popup") {
			return multiClientBinaryPopupSurface()
		}
	}
	return multiClientBinaryOpenerSurface()
}

func multiClientBinaryOpenerSurface() ui.Node {
	self := ui.UseState(newBinaryIdentity("catalog-tab", "storefront"))
	crossTabTransport := ui.UseState("pending")
	crossTabMode := ui.UseState("No cross-tab preview sent yet.")
	crossTabReceived := ui.UseState("Waiting for another tab or a fallback event.")
	popupStatus := ui.UseState("Closed")
	popupPeerID := ui.UseState("")
	popupAck := ui.UseState("No popup ack yet.")
	logs := ui.UseState([]string{"Open this example in a second tab for cross-tab preview delivery, then open the popup to watch JSON handshake plus binary preview flow."})
	crossTabRef := ui.UseRef(interop.CrossTabChannel{})
	crossTabCancelRef := ui.UseRef((func())(nil))
	popupRef := ui.UseRef(interop.WindowChannel{})
	popupCancelRef := ui.UseRef((func())(nil))

	handleCrossTabMessage := func(message interop.ClientMessage, err error) {
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Cross-tab message failed", err))
			return
		}
		if message.Source.ID == self.Get().ID {
			return
		}
		if message.Kind == interop.ClientHello {
			appendBinaryLog(logs, fmt.Sprintf("Peer hello from %s on %s", message.Source.Surface, crossTabTransport.Get()))
			return
		}
		if message.Kind != interop.ClientEvent || message.Topic != binaryPreviewTopic {
			return
		}
		if message.Encoding == interop.ClientPayloadBinary {
			crossTabReceived.Set("Binary preview received from another tab: " + previewSummary(message.Payload))
			crossTabMode.Set("Last cross-tab delivery used binary over " + crossTabTransport.Get())
			appendBinaryLog(logs, fmt.Sprintf("Received binary preview from %s: %s", message.Source.Surface, previewSummary(message.Payload)))
			return
		}
		crossTabReceived.Set(fmt.Sprintf("JSON fallback event from another tab: %v", message.Payload))
		crossTabMode.Set("Last cross-tab delivery fell back to JSON metadata")
		appendBinaryLog(logs, fmt.Sprintf("Received JSON fallback from %s because the transport could not carry binary.", message.Source.Surface))
	}

	handlePopupMessage := func(channel interop.WindowChannel, message interop.ClientMessage, err error) {
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Popup message failed", err))
			return
		}
		if message.Source.ID == self.Get().ID {
			return
		}
		switch message.Kind {
		case interop.ClientHello:
			popupPeerID.Set(message.Source.ID)
			popupStatus.Set("Connected")
			appendBinaryLog(logs, fmt.Sprintf("Popup hello from %s", message.Source.Surface))
		case interop.ClientResult:
			if message.Topic != binaryAckTopic || message.Target != self.Get().ID {
				return
			}
			popupAck.Set(fmt.Sprintf("Popup ack: %v", message.Payload))
			appendBinaryLog(logs, fmt.Sprintf("Popup acknowledged binary preview with %v", message.Payload))
		case interop.ClientGoodbye:
			popupStatus.Set("Disconnected")
			popupPeerID.Set("")
			appendBinaryLog(logs, "Popup sent goodbye and closed its targeted channel.")
		}
	}

	releasePopup := func(closeWindow bool) {
		if cancel := popupCancelRef.Get(); cancel != nil {
			cancel()
			popupCancelRef.Set(nil)
		}
		channel := popupRef.Get()
		if closeWindow && channel.Name() != "" && !channel.Closed() {
			_ = interop.PublishClientGoodbyeWindow(channel, self.Get())
			_ = channel.Close()
		}
		popupRef.Set(interop.WindowChannel{})
		popupPeerID.Set("")
	}

	ui.UseEffect(func() func() {
		channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: multiClientBinaryChannelName})
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Cross-tab channel unavailable", err))
			crossTabTransport.Set("unavailable")
			return nil
		}
		crossTabRef.Set(channel)
		crossTabTransport.Set(channel.Transport())

		subscription, err := interop.SubscribeClientMessages(channel, handleCrossTabMessage)
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Cross-tab subscription failed", err))
			_ = channel.Close()
			crossTabRef.Set(interop.CrossTabChannel{})
			return nil
		}
		crossTabCancelRef.Set(subscription.Cancel)

		if err := interop.PublishClientHello(channel, self.Get()); err != nil {
			appendBinaryLog(logs, describeBinaryError("Cross-tab hello failed", err))
		} else {
			appendBinaryLog(logs, "Published JSON hello on the cross-tab control plane.")
		}

		return func() {
			if cancel := crossTabCancelRef.Get(); cancel != nil {
				cancel()
				crossTabCancelRef.Set(nil)
			}
			_ = interop.PublishClientGoodbye(channel, self.Get())
			_ = channel.Close()
			crossTabRef.Set(interop.CrossTabChannel{})
		}
	}, true)

	ui.UseEffect(func() func() {
		timer, err := interop.ScheduleInterval(500*time.Millisecond, func() {
			channel := popupRef.Get()
			if channel.Name() == "" || !channel.Closed() {
				return
			}
			releasePopup(false)
			popupStatus.Set("Closed")
			appendBinaryLog(logs, "Popup closed unexpectedly. Reopen it to restore targeted binary preview delivery.")
		})
		return func() {
			if err == nil {
				_ = timer.Cancel()
			}
			releasePopup(true)
		}
	}, true)

	broadcastPreview := ui.UseEvent(func() {
		channel := crossTabRef.Get()
		if channel.Name() == "" {
			appendBinaryLog(logs, "Cross-tab channel is not ready yet.")
			return
		}
		payload := previewBytes()
		if channel.Transport() == "broadcast-channel" {
			if err := interop.PublishClientBinaryCrossTab(channel, binaryPreviewTopic, self.Get(), interop.ClientBinaryPayload{ContentType: binaryContentType, Bytes: payload}); err != nil {
				appendBinaryLog(logs, describeBinaryError("Binary cross-tab publish failed", err))
				return
			}
			crossTabMode.Set("Sent binary preview over BroadcastChannel")
			appendBinaryLog(logs, "Published binary preview to other tabs through BroadcastChannel.")
			return
		}
		fallback := map[string]any{
			"fallback":    "json",
			"contentType": binaryContentType,
			"byteCount":   len(payload),
			"reason":      "resolved cross-tab transport cannot carry binary",
		}
		if err := interop.PublishClientEvent(channel, binaryPreviewTopic, self.Get(), fallback); err != nil {
			appendBinaryLog(logs, describeBinaryError("JSON fallback publish failed", err))
			return
		}
		crossTabMode.Set("Sent JSON fallback because the resolved transport was " + channel.Transport())
		appendBinaryLog(logs, "Cross-tab transport could not carry binary, so the example sent JSON metadata instead.")
	})

	openPopup := ui.UseEvent(func() {
		current := popupRef.Get()
		if current.Name() != "" && !current.Closed() {
			_ = current.Focus()
			popupStatus.Set("Connected")
			appendBinaryLog(logs, "Reused the existing popup and focused it.")
			return
		}

		releasePopup(false)
		channel, err := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
			URL:      multiClientBinaryPopupURL,
			Name:     multiClientBinaryPopupName,
			Features: "popup=yes,width=560,height=760",
		})
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Opening popup failed", err))
			return
		}
		subscription, err := interop.SubscribeClientWindowMessages(channel, func(message interop.ClientMessage, receiveErr error) {
			handlePopupMessage(channel, message, receiveErr)
		})
		if err != nil {
			_ = channel.Close()
			appendBinaryLog(logs, describeBinaryError("Popup subscription failed", err))
			return
		}
		popupRef.Set(channel)
		popupCancelRef.Set(subscription.Cancel)
		popupStatus.Set("Connecting")

		if err := interop.PublishClientHelloWindow(channel, self.Get()); err != nil {
			appendBinaryLog(logs, describeBinaryError("Popup hello failed", err))
			return
		}
		appendBinaryLog(logs, "Sent JSON hello to the popup before any binary payload traffic.")
	})

	sendPopupPreview := ui.UseEvent(func() {
		channel := popupRef.Get()
		if channel.Name() == "" || channel.Closed() {
			appendBinaryLog(logs, "Open the popup before sending a targeted binary preview.")
			popupStatus.Set("Closed")
			return
		}
		target := strings.TrimSpace(popupPeerID.Get())
		if target == "" {
			appendBinaryLog(logs, "Waiting for popup hello before sending the targeted binary preview.")
			return
		}
		if err := interop.PublishClientBinaryWindow(channel, binaryPreviewTopic, self.Get(), target, interop.ClientBinaryPayload{ContentType: binaryContentType, Bytes: previewBytes()}); err != nil {
			appendBinaryLog(logs, describeBinaryError("Popup binary publish failed", err))
			return
		}
		popupStatus.Set("Preview sent")
		appendBinaryLog(logs, "Published binary preview to the popup, with JSON ack expected back on the control plane.")
	})

	closePopup := ui.UseEvent(func() {
		channel := popupRef.Get()
		if channel.Name() == "" || channel.Closed() {
			popupStatus.Set("Closed")
			appendBinaryLog(logs, "Popup is already closed.")
			releasePopup(false)
			return
		}
		releasePopup(true)
		popupStatus.Set("Closed")
		appendBinaryLog(logs, "Closed the popup after targeted preview flow completed.")
	})

	return shared.ExamplePage(
		"Multi-Client Binary",
		"interop mixed JSON and binary client messages",
		"Use JSON for handshake and acknowledgement traffic, use binary only for payload-heavy preview delivery on supported transports, and fall back to JSON metadata when the resolved cross-tab transport cannot carry bytes.",
		shared.ExamplePanel("Cross-tab preview",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", self.Get().Surface),
				shared.ExampleStat("Role", self.Get().Role),
				shared.ExampleStat("Transport", crossTabTransport.Get()),
				shared.ExampleStat("Last mode", crossTabMode.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Open this example in a second tab. The control plane stays JSON, but the preview payload uses binary only when the resolved cross-tab transport is BroadcastChannel. If the browser falls back to storage events, the example publishes JSON metadata instead of pretending bytes are portable there.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Broadcast preview", broadcastPreview),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(crossTabReceived.Get())),
		),
		shared.ExamplePanel("Popup preview",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Popup", popupStatus.Get()),
				shared.ExampleStat("Popup peer", popupPeerID.Get()),
				shared.ExampleStat("Ack", popupAck.Get()),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Open popup", openPopup),
				shared.ExampleButton("Send popup preview", sendPopupPreview),
				shared.ExampleButton("Close popup", closePopup),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The popup channel demonstrates the mixed control/data plane directly: JSON hello and result frames wrap the lifecycle, while the preview bytes themselves travel as structured-clone binary.")),
		),
		shared.ExamplePanel("Diagnostics",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("The log records whether the example used JSON or binary, whether the popup handshake completed, and when the cross-tab flow had to degrade because the resolved transport could not carry raw bytes.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, renderBinaryLogs(logs.Get())...),
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
	self := ui.UseState(newBinaryIdentity("popup-preview", "inspector"))
	openerID := ui.UseState("")
	connection := ui.UseState("Connecting to opener...")
	lastPreview := ui.UseState("No preview received yet.")
	orphaned := ui.UseState(false)
	logs := ui.UseState([]string{"This popup waits for a JSON hello, then accepts targeted binary preview traffic from its opener."})
	channelRef := ui.UseRef(interop.WindowChannel{})
	cancelRef := ui.UseRef((func())(nil))

	handlePopupMessage := func(channel interop.WindowChannel, message interop.ClientMessage, err error) {
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Opener message failed", err))
			return
		}
		if message.Source.ID == self.Get().ID {
			return
		}
		switch message.Kind {
		case interop.ClientHello:
			openerID.Set(message.Source.ID)
			connection.Set("Connected")
			orphaned.Set(false)
			appendBinaryLog(logs, fmt.Sprintf("Received JSON hello from %s", message.Source.Surface))
		case interop.ClientEvent:
			if message.Topic != binaryPreviewTopic || message.Target != self.Get().ID || message.Encoding != interop.ClientPayloadBinary {
				return
			}
			lastPreview.Set(fmt.Sprintf("Binary preview received: %s (%s)", previewSummary(message.Payload), message.ContentType))
			appendBinaryLog(logs, fmt.Sprintf("Received targeted binary preview from %s: %s", message.Source.Surface, previewSummary(message.Payload)))
			ack := interop.ClientMessage{
				Kind:   interop.ClientResult,
				Topic:  binaryAckTopic,
				Source: self.Get(),
				Target: message.Source.ID,
				Payload: map[string]any{
					"contentType": message.ContentType,
					"summary":     previewSummary(message.Payload),
				},
			}
			if publishErr := interop.PublishClientWindowMessage(channel, ack); publishErr != nil {
				appendBinaryLog(logs, describeBinaryError("Popup ack failed", publishErr))
				return
			}
			appendBinaryLog(logs, "Sent JSON ack back to the opener after binary preview receipt.")
		case interop.ClientGoodbye:
			orphaned.Set(true)
			connection.Set("Opener disconnected")
			appendBinaryLog(logs, "Opener sent goodbye. The popup is now orphaned.")
		}
	}

	ui.UseEffect(func() func() {
		channel, err := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: multiClientBinaryPopupName})
		if err != nil {
			orphaned.Set(true)
			connection.Set("Opened without an opener")
			appendBinaryLog(logs, describeBinaryError("No opener channel available", err))
			return nil
		}
		channelRef.Set(channel)
		connection.Set("Connected")
		orphaned.Set(channel.Closed())

		subscription, err := interop.SubscribeClientWindowMessages(channel, func(message interop.ClientMessage, receiveErr error) {
			handlePopupMessage(channel, message, receiveErr)
		})
		if err != nil {
			appendBinaryLog(logs, describeBinaryError("Popup subscription failed", err))
			connection.Set("Subscription failed")
			return nil
		}
		cancelRef.Set(subscription.Cancel)

		if err := interop.PublishClientHelloWindow(channel, self.Get()); err != nil {
			appendBinaryLog(logs, describeBinaryError("Popup hello failed", err))
		} else {
			appendBinaryLog(logs, "Published JSON hello to the opener control plane.")
		}

		timer, timerErr := interop.ScheduleInterval(500*time.Millisecond, func() {
			current := channelRef.Get()
			if current.Name() == "" || !current.Closed() {
				return
			}
			orphaned.Set(true)
			connection.Set("Opener disconnected")
		})

		return func() {
			if cancel := cancelRef.Get(); cancel != nil {
				cancel()
				cancelRef.Set(nil)
			}
			if timerErr == nil {
				_ = timer.Cancel()
			}
			_ = interop.PublishClientGoodbyeWindow(channel, self.Get())
		}
	}, true)

	orphanMessage := "The opener is still available for targeted preview traffic."
	if orphaned.Get() {
		orphanMessage = "The opener disappeared or this page was opened directly. Keep the diagnostic view visible, but do not assume the coordinator is still present."
	}

	return shared.ExamplePage(
		"Popup Binary Preview",
		"interop.OpenWindowOpenerChannel with binary payloads",
		"This popup accepts a targeted binary preview over the window channel and replies with a JSON acknowledgement so the control plane remains easy to inspect.",
		shared.ExamplePanel("Popup state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", self.Get().Surface),
				shared.ExampleStat("Role", self.Get().Role),
				shared.ExampleStat("Connection", connection.Get()),
				shared.ExampleStat("Opener id", openerID.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(orphanMessage)),
		),
		shared.ExamplePanel("Latest preview",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Preview", lastPreview.Get()),
				shared.ExampleStat("Orphaned", fmt.Sprintf("%t", orphaned.Get())),
			),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, renderBinaryLogs(logs.Get())...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiClientBinaryRoot), "#app")
	select {}
}
