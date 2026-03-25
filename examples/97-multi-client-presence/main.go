//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sort"
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
	multiClientPresenceChannelName = "example:multi-client:presence"
	multiClientPopupChannelName    = "example:multi-client:popup"
	multiClientPopupURL            = "./multi-client-presence-popup.html"
	peerLeaseTimeout               = 12 * time.Second
)

type peerStatus struct {
	Identity interop.ClientIdentity
	State    string
	LastSeen time.Time
	Summary  string
}

func newClientIdentity(surface string, role string) interop.ClientIdentity {
	trimmedSurface := strings.TrimSpace(surface)
	trimmedRole := strings.TrimSpace(role)
	return interop.ClientIdentity{
		ID:      fmt.Sprintf("%s-%d", trimmedSurface, time.Now().UTC().UnixNano()),
		App:     "examples/multi-client-presence",
		Surface: trimmedSurface,
		Role:    trimmedRole,
		Version: "v1",
	}
}

func clonePeerStatuses(previous map[string]peerStatus) map[string]peerStatus {
	next := make(map[string]peerStatus, len(previous))
	for key, value := range previous {
		next[key] = value
	}
	return next
}

func describeError(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	if code, ok := interop.CodeOf(err); ok {
		return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

func describePayload(payload any) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", payload))
	if text == "" || text == "<nil>" {
		return "No payload"
	}
	return text
}

func formatSeenAt(value time.Time) string {
	if value.IsZero() {
		return "waiting"
	}
	return value.Local().Format("15:04:05")
}

func renderPeerList(peers map[string]peerStatus) []ui.Node {
	if len(peers) == 0 {
		return []ui.Node{
			html.Li(html.Props{Class: "rounded-2xl border border-dashed border-white/10 bg-slate-950/35 px-4 py-3 text-sm leading-7 text-slate-400"}, html.Text("No remote peers discovered yet. Open this page in a second tab, or reconnect a paused tab to watch hello and result traffic populate the registry.")),
		}
	}

	keys := make([]string, 0, len(peers))
	for key := range peers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	nodes := make([]ui.Node, 0, len(keys))
	for _, key := range keys {
		peer := peers[key]
		label := fmt.Sprintf("%s | %s | %s | %s | last seen %s", peer.Identity.Surface, peer.Identity.Role, peer.State, peer.Summary, formatSeenAt(peer.LastSeen))
		nodes = append(nodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(label)),
		)
	}
	return nodes
}

func renderLogList(entries []string) []ui.Node {
	nodes := make([]ui.Node, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(entry)),
		)
	}
	return nodes
}

func multiClientPresenceRoot() ui.Node {
	location, err := interop.GetWindowLocation()
	if err == nil {
		path := strings.ToLower(strings.TrimSpace(location.Pathname()))
		if strings.Contains(path, "popup") {
			return multiClientPopupSurface()
		}
	}
	return multiClientOpenerSurface()
}

func multiClientOpenerSurface() ui.Node {
	self := ui.UseState(newClientIdentity("storefront-tab", "storefront"))
	presenceOnline := ui.UseState(true)
	presenceTransport := ui.UseState("pending")
	popupStatus := ui.UseState("Closed")
	popupPeerID := ui.UseState("")
	popupResult := ui.UseState("No popup query result yet.")
	peerRegistry := ui.UseState(map[string]peerStatus{})
	logs := ui.UseState([]string{"Open this page in a second tab, then use the popup to watch hello, query, result, lease expiry, and reconnect behavior."})
	presenceChannelRef := ui.UseRef(interop.CrossTabChannel{})
	presenceCancelRef := ui.UseRef((func())(nil))
	popupChannelRef := ui.UseRef(interop.WindowChannel{})
	popupCancelRef := ui.UseRef((func())(nil))

	appendLog := func(line string) {
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

	upsertPeer := func(identity interop.ClientIdentity, state string, summary string) {
		if strings.TrimSpace(identity.ID) == "" || identity.ID == self.Get().ID {
			return
		}
		next := clonePeerStatuses(peerRegistry.Get())
		entry := next[identity.ID]
		entry.Identity = identity
		if strings.TrimSpace(state) != "" {
			entry.State = state
		}
		entry.LastSeen = time.Now().UTC()
		if strings.TrimSpace(summary) != "" {
			entry.Summary = summary
		}
		next[identity.ID] = entry
		peerRegistry.Set(next)
	}

	markPeerState := func(id string, state string, summary string) {
		trimmedID := strings.TrimSpace(id)
		if trimmedID == "" {
			return
		}
		current := peerRegistry.Get()
		entry, ok := current[trimmedID]
		if !ok {
			return
		}
		next := clonePeerStatuses(current)
		entry.State = state
		if strings.TrimSpace(summary) != "" {
			entry.Summary = summary
		}
		next[trimmedID] = entry
		peerRegistry.Set(next)
	}

	releasePopup := func(closeWindow bool) {
		if cancel := popupCancelRef.Get(); cancel != nil {
			cancel()
			popupCancelRef.Set(nil)
		}
		channel := popupChannelRef.Get()
		if closeWindow && channel.Name() != "" && !channel.Closed() {
			_ = interop.PublishClientGoodbyeWindow(channel, self.Get())
			_ = channel.Close()
		}
		popupChannelRef.Set(interop.WindowChannel{})
		popupPeerID.Set("")
	}

	handlePopupMessage := func(channel interop.WindowChannel, message interop.ClientMessage, err error) {
		if err != nil {
			appendLog(describeError("Popup message failed", err))
			return
		}
		if message.Source.ID == self.Get().ID {
			return
		}

		switch message.Kind {
		case interop.ClientHello:
			popupPeerID.Set(message.Source.ID)
			popupStatus.Set("Connected")
			appendLog(fmt.Sprintf("Popup hello from %s (%s)", message.Source.Surface, message.Source.ID))
		case interop.ClientQuery:
			if message.Target != "" && message.Target != self.Get().ID {
				return
			}
			response := interop.ClientMessage{
				Kind:   interop.ClientResult,
				Topic:  message.Topic,
				Source: self.Get(),
				Target: message.Source.ID,
				Payload: map[string]string{
					"status":   "opener-ready",
					"presence": fmt.Sprintf("%t", presenceOnline.Get()),
					"surface":  self.Get().Surface,
				},
			}
			if publishErr := interop.PublishClientWindowMessage(channel, response); publishErr != nil {
				appendLog(describeError("Popup query reply failed", publishErr))
				return
			}
			appendLog("Answered popup query with a targeted window result.")
		case interop.ClientResult:
			if message.Target != self.Get().ID {
				return
			}
			popupStatus.Set("Ready")
			popupResult.Set(describePayload(message.Payload))
			appendLog(fmt.Sprintf("Popup result on %s: %s", message.Topic, describePayload(message.Payload)))
		case interop.ClientGoodbye:
			popupStatus.Set("Disconnected")
			popupPeerID.Set("")
			appendLog("Popup sent goodbye and left the targeted channel.")
		}
	}

	ui.UseEffect(func() func() {
		if !presenceOnline.Get() {
			presenceTransport.Set("offline")
			return nil
		}

		channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: multiClientPresenceChannelName})
		if err != nil {
			appendLog(describeError("Presence channel unavailable", err))
			presenceTransport.Set("unavailable")
			return nil
		}

		presenceChannelRef.Set(channel)
		presenceTransport.Set(channel.Transport())

		subscription, err := interop.SubscribeClientMessages(channel, func(message interop.ClientMessage, receiveErr error) {
			if receiveErr != nil {
				appendLog(describeError("Presence receive failed", receiveErr))
				return
			}
			if message.Source.ID == self.Get().ID {
				return
			}

			switch message.Kind {
			case interop.ClientHello:
				upsertPeer(message.Source, "seen", "hello received")
				appendLog(fmt.Sprintf("Peer hello from %s via %s", message.Source.Surface, channel.Transport()))
			case interop.ClientQuery:
				upsertPeer(message.Source, "seen", "discovery query received")
				if message.Topic != interop.ClientPresenceTopic {
					return
				}
				if publishErr := interop.PublishClientResult(channel, interop.ClientPresenceTopic, self.Get(), message.Source.ID, map[string]string{
					"surface": self.Get().Surface,
					"role":    self.Get().Role,
					"state":   "ready",
				}); publishErr != nil {
					appendLog(describeError("Presence query reply failed", publishErr))
					return
				}
				appendLog(fmt.Sprintf("Answered discovery query for %s.", message.Source.Surface))
			case interop.ClientResult:
				if message.Target != self.Get().ID {
					return
				}
				upsertPeer(message.Source, "ready", describePayload(message.Payload))
				appendLog(fmt.Sprintf("Targeted result from %s: %s", message.Source.Surface, describePayload(message.Payload)))
			case interop.ClientGoodbye:
				markPeerState(message.Source.ID, "disconnected", "goodbye received")
				appendLog(fmt.Sprintf("Peer goodbye from %s.", message.Source.Surface))
			}
		})
		if err != nil {
			appendLog(describeError("Presence subscription failed", err))
			_ = channel.Close()
			presenceChannelRef.Set(interop.CrossTabChannel{})
			return nil
		}
		presenceCancelRef.Set(subscription.Cancel)

		if err := interop.PublishClientHello(channel, self.Get()); err != nil {
			appendLog(describeError("Initial hello failed", err))
		} else {
			appendLog("Published local hello on the cross-tab presence channel.")
		}

		if err := interop.PublishClientQuery(channel, interop.ClientPresenceTopic, self.Get()); err != nil {
			appendLog(describeError("Initial discovery query failed", err))
		} else {
			appendLog("Published query(topic=clients) to discover live peers.")
		}

		timer, timerErr := interop.ScheduleInterval(time.Second, func() {
			current := peerRegistry.Get()
			if len(current) == 0 {
				return
			}

			now := time.Now().UTC()
			next := clonePeerStatuses(current)
			expired := make([]string, 0)
			for id, entry := range current {
				if entry.State == "expired" || entry.State == "disconnected" {
					continue
				}
				if now.Sub(entry.LastSeen) <= peerLeaseTimeout {
					continue
				}
				entry.State = "expired"
				entry.Summary = "lease expired; waiting for fresh hello"
				next[id] = entry
				expired = append(expired, entry.Identity.Surface)
			}
			if len(expired) == 0 {
				return
			}
			peerRegistry.Set(next)
			for _, surface := range expired {
				appendLog("Peer lease expired: " + surface)
			}
		})

		return func() {
			if cancel := presenceCancelRef.Get(); cancel != nil {
				cancel()
				presenceCancelRef.Set(nil)
			}
			if timerErr == nil {
				_ = timer.Cancel()
			}
			_ = interop.PublishClientGoodbye(channel, self.Get())
			_ = channel.Close()
			presenceChannelRef.Set(interop.CrossTabChannel{})
		}
	}, presenceOnline.Get())

	ui.UseEffect(func() func() {
		timer, err := interop.ScheduleInterval(500*time.Millisecond, func() {
			channel := popupChannelRef.Get()
			if channel.Name() == "" {
				return
			}
			if !channel.Closed() {
				return
			}
			releasePopup(false)
			popupStatus.Set("Closed")
			appendLog("Popup closed unexpectedly. Reopen it to restore targeted coordination.")
		})
		return func() {
			if err == nil {
				_ = timer.Cancel()
			}
			releasePopup(true)
		}
	}, true)

	reannounceHello := ui.UseEvent(func() {
		channel := presenceChannelRef.Get()
		if channel.Name() == "" {
			appendLog("Reconnect the presence channel before re-announcing hello.")
			return
		}
		if err := interop.PublishClientHello(channel, self.Get()); err != nil {
			appendLog(describeError("Hello publish failed", err))
			return
		}
		appendLog("Re-announced hello to refresh presence without a full reconnect.")
	})

	queryPeers := ui.UseEvent(func() {
		channel := presenceChannelRef.Get()
		if channel.Name() == "" {
			appendLog("Reconnect the presence channel before querying peers.")
			return
		}
		if err := interop.PublishClientQuery(channel, interop.ClientPresenceTopic, self.Get()); err != nil {
			appendLog(describeError("Discovery query failed", err))
			return
		}
		appendLog("Published a fresh clients query for late join discovery.")
	})

	disconnectPresence := ui.UseEvent(func() {
		if !presenceOnline.Get() {
			appendLog("Presence channel is already offline.")
			return
		}
		presenceOnline.Set(false)
		presenceTransport.Set("offline")
		appendLog("Disconnected the local presence channel. Other tabs should eventually mark this lease expired.")
	})

	reconnectPresence := ui.UseEvent(func() {
		if presenceOnline.Get() {
			appendLog("Presence channel is already connected.")
			return
		}
		presenceOnline.Set(true)
		appendLog("Reconnecting the local presence channel and replaying hello plus discovery.")
	})

	openPopup := ui.UseEvent(func() {
		current := popupChannelRef.Get()
		if current.Name() != "" && !current.Closed() {
			_ = current.Focus()
			popupStatus.Set("Connected")
			appendLog("Reused the existing popup and focused it.")
			return
		}

		releasePopup(false)
		channel, err := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
			URL:      multiClientPopupURL,
			Name:     multiClientPopupChannelName,
			Features: "popup=yes,width=560,height=760",
		})
		if err != nil {
			appendLog(describeError("Opening popup failed", err))
			return
		}

		subscription, err := interop.SubscribeClientWindowMessages(channel, func(message interop.ClientMessage, receiveErr error) {
			handlePopupMessage(channel, message, receiveErr)
		})
		if err != nil {
			_ = channel.Close()
			appendLog(describeError("Popup subscription failed", err))
			return
		}

		popupChannelRef.Set(channel)
		popupCancelRef.Set(subscription.Cancel)
		popupStatus.Set("Connecting")
		appendLog("Opened the popup inspector and subscribed to its window channel.")

		if err := interop.PublishClientHelloWindow(channel, self.Get()); err != nil {
			appendLog(describeError("Popup hello failed", err))
			return
		}
		appendLog("Sent opener hello on the popup channel.")
	})

	queryPopup := ui.UseEvent(func() {
		channel := popupChannelRef.Get()
		if channel.Name() == "" || channel.Closed() {
			popupStatus.Set("Closed")
			appendLog("Open the popup before sending a targeted query.")
			return
		}
		target := strings.TrimSpace(popupPeerID.Get())
		if target == "" {
			appendLog("Waiting for popup hello before sending a targeted query.")
			return
		}

		message := interop.ClientMessage{
			Kind:   interop.ClientQuery,
			Topic:  "popup-status",
			Source: self.Get(),
			Target: target,
			Payload: map[string]string{
				"request": "status",
			},
		}
		if err := interop.PublishClientWindowMessage(channel, message); err != nil {
			appendLog(describeError("Popup query failed", err))
			return
		}
		appendLog("Sent a targeted popup query. The reply should return only to this opener.")
	})

	closePopup := ui.UseEvent(func() {
		channel := popupChannelRef.Get()
		if channel.Name() == "" || channel.Closed() {
			popupStatus.Set("Closed")
			appendLog("Popup is already closed.")
			releasePopup(false)
			return
		}
		releasePopup(true)
		popupStatus.Set("Closed")
		appendLog("Closed the popup and sent goodbye on the window channel.")
	})

	peerNodes := renderPeerList(peerRegistry.Get())
	logNodes := renderLogList(logs.Get())

	return shared.ExamplePage(
		"Multi-Client Presence",
		"interop multi-client helpers",
		"Demonstrate hello, late join discovery, targeted result replies, lease expiry, and reconnect behavior across cross-tab and popup window channels.",
		shared.ExamplePanel("Cross-tab presence",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", self.Get().Surface),
				shared.ExampleStat("Role", self.Get().Role),
				shared.ExampleStat("Presence", map[bool]string{true: "connected", false: "offline"}[presenceOnline.Get()]),
				shared.ExampleStat("Transport", presenceTransport.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Open this example in a second tab. Each tab publishes hello on boot, asks query(topic=clients) for late-join discovery, answers with targeted result replies, and expires peers locally when their lease ages out.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Re-announce hello", reannounceHello),
				shared.ExampleButton("Query clients", queryPeers),
				shared.ExampleButton("Disconnect presence", disconnectPresence),
				shared.ExampleButton("Reconnect presence", reconnectPresence),
			),
		),
		shared.ExamplePanel("Peer registry",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Lease expiry is local state, not a browser-wide scan. If a tab disconnects and does not re-announce, the remaining tab marks it expired and waits for a fresh hello before treating it as ready again.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, peerNodes...),
		),
		shared.ExamplePanel("Popup handshake",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Popup", popupStatus.Get()),
				shared.ExampleStat("Popup peer id", popupPeerID.Get()),
				shared.ExampleStat("Last popup result", popupResult.Get()),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Open popup", openPopup),
				shared.ExampleButton("Query popup", queryPopup),
				shared.ExampleButton("Close popup", closePopup),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The popup uses WindowOpenerChannel and the opener uses OpenSecondaryWindowChannel. Both sides exchange hello, then targeted query and result messages rather than pretending the popup is just another broadcast peer.")),
		),
		shared.ExamplePanel("Diagnostics",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Use the log below to watch the exact hello, query, result, goodbye, expiry, and reconnect transitions. This is the intended control-plane shape for multi-client coordination in the browser.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, logNodes...),
		),
		shared.ExamplePanel("Integration shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Cross-tab presence answers the late-join question, while targeted popup queries cover sovereign child surfaces. The example deliberately keeps business authority local and uses typed JSON messages instead of shared mutable state.")),
			shared.ExampleCode(
				`channel, _ := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: "example:multi-client:presence"})`,
				`_ = interop.PublishClientHello(channel, self)`,
				`_ = interop.PublishClientQuery(channel, interop.ClientPresenceTopic, self)`,
				`_ = interop.PublishClientResult(channel, interop.ClientPresenceTopic, self, requesterID, payload)`,
				`popup, _ := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{URL: "./multi-client-presence-popup.html", Name: "example:multi-client:popup"})`,
			),
		),
	)
}

func multiClientPopupSurface() ui.Node {
	self := ui.UseState(newClientIdentity("popup-inspector", "inspector"))
	openerID := ui.UseState("")
	connection := ui.UseState("Connecting to opener...")
	orphaned := ui.UseState(false)
	lastResult := ui.UseState("No opener result yet.")
	logs := ui.UseState([]string{"This popup waits for the opener hello, then answers targeted queries with result messages."})
	channelRef := ui.UseRef(interop.WindowChannel{})
	cancelRef := ui.UseRef((func())(nil))

	appendLog := func(line string) {
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

	handleMessage := func(channel interop.WindowChannel, message interop.ClientMessage, err error) {
		if err != nil {
			appendLog(describeError("Opener message failed", err))
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
			appendLog(fmt.Sprintf("Received opener hello from %s.", message.Source.Surface))
		case interop.ClientQuery:
			if message.Target != "" && message.Target != self.Get().ID {
				return
			}
			response := interop.ClientMessage{
				Kind:   interop.ClientResult,
				Topic:  message.Topic,
				Source: self.Get(),
				Target: message.Source.ID,
				Payload: map[string]string{
					"status":   "popup-ready",
					"orphaned": fmt.Sprintf("%t", orphaned.Get()),
					"surface":  self.Get().Surface,
				},
			}
			if publishErr := interop.PublishClientWindowMessage(channel, response); publishErr != nil {
				appendLog(describeError("Popup result publish failed", publishErr))
				return
			}
			appendLog("Answered opener query with a targeted popup result.")
		case interop.ClientResult:
			if message.Target != self.Get().ID {
				return
			}
			lastResult.Set(describePayload(message.Payload))
			appendLog(fmt.Sprintf("Received targeted result on %s: %s", message.Topic, describePayload(message.Payload)))
		case interop.ClientGoodbye:
			orphaned.Set(true)
			connection.Set("Opener closed")
			appendLog("Opener sent goodbye. This popup is now orphaned.")
		}
	}

	ui.UseEffect(func() func() {
		channel, err := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: multiClientPopupChannelName})
		if err != nil {
			orphaned.Set(true)
			connection.Set("Opened without an opener")
			appendLog(describeError("No opener channel available", err))
			return nil
		}

		channelRef.Set(channel)
		orphaned.Set(channel.Closed())
		connection.Set("Connected")

		subscription, err := interop.SubscribeClientWindowMessages(channel, func(message interop.ClientMessage, receiveErr error) {
			handleMessage(channel, message, receiveErr)
		})
		if err != nil {
			connection.Set("Subscription failed")
			appendLog(describeError("Popup subscription failed", err))
			return nil
		}
		cancelRef.Set(subscription.Cancel)

		if err := interop.PublishClientHelloWindow(channel, self.Get()); err != nil {
			appendLog(describeError("Popup hello failed", err))
		} else {
			appendLog("Published popup hello to the opener channel.")
		}

		timer, timerErr := interop.ScheduleInterval(500*time.Millisecond, func() {
			current := channelRef.Get()
			if current.Name() == "" {
				return
			}
			if !current.Closed() {
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

	queryOpener := ui.UseEvent(func() {
		channel := channelRef.Get()
		if channel.Name() == "" || channel.Closed() {
			orphaned.Set(true)
			connection.Set("Opener unavailable")
			appendLog("Opener is unavailable; the popup cannot send a targeted query.")
			return
		}
		target := strings.TrimSpace(openerID.Get())
		if target == "" {
			appendLog("Waiting for opener hello before sending a targeted query.")
			return
		}

		message := interop.ClientMessage{
			Kind:   interop.ClientQuery,
			Topic:  "opener-status",
			Source: self.Get(),
			Target: target,
			Payload: map[string]string{
				"request": "status",
			},
		}
		if err := interop.PublishClientWindowMessage(channel, message); err != nil {
			appendLog(describeError("Opener query failed", err))
			return
		}
		appendLog("Sent a targeted opener query from the popup.")
	})

	logNodes := renderLogList(logs.Get())
	orphanMessage := "The opener is still present and can answer targeted queries."
	if orphaned.Get() {
		orphanMessage = "The opener disappeared or this page was opened directly. Keep diagnostics visible, but do not assume the coordinator still exists."
	}

	return shared.ExamplePage(
		"Popup Inspector",
		"interop.WindowOpenerChannel and multi-client messages",
		"This popup exchanges hello, query, result, and goodbye messages with its opener without pretending the child surface shares the opener's lifecycle or authority.",
		shared.ExamplePanel("Popup state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", self.Get().Surface),
				shared.ExampleStat("Role", self.Get().Role),
				shared.ExampleStat("Connection", connection.Get()),
				shared.ExampleStat("Opener id", openerID.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(orphanMessage)),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Query opener", queryOpener),
			),
		),
		shared.ExamplePanel("Latest opener result",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Targeted result", lastResult.Get()),
				shared.ExampleStat("Orphaned", fmt.Sprintf("%t", orphaned.Get())),
			),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, logNodes...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiClientPresenceRoot), "#app")
	select {}
}
