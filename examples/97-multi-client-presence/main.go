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

func newClientIdentity(parseSurface string, parseRole string) interop.ClientIdentity {
	parseTrimmedSurface := strings.TrimSpace(parseSurface)
	parseTrimmedRole := strings.TrimSpace(parseRole)
	return interop.ClientIdentity{
		ID:      fmt.Sprintf("%s-%d", parseTrimmedSurface, time.Now().UTC().UnixNano()),
		App:     "examples/multi-client-presence",
		Surface: parseTrimmedSurface,
		Role:    parseTrimmedRole,
		Version: "v1",
	}
}

func clonePeerStatuses(parsePrevious map[string]peerStatus) map[string]peerStatus {
	parseNext := make(map[string]peerStatus, len(parsePrevious))
	for parseKey, parseValue := range parsePrevious {
		parseNext[parseKey] = parseValue
	}
	return parseNext
}

func describeError(parsePrefix string, parseErr error) string {
	if parseErr == nil {
		return parsePrefix
	}
	if parseCode, parseOk := interop.CodeOf(parseErr); parseOk {
		return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr)
	}
	return fmt.Sprintf("%s: %v", parsePrefix, parseErr)
}

func describePayload(parsePayload any) string {
	parseText := strings.TrimSpace(fmt.Sprintf("%v", parsePayload))
	if parseText == "" || parseText == "<nil>" {
		return "No payload"
	}
	return parseText
}

func formatSeenAt(parseValue time.Time) string {
	if parseValue.IsZero() {
		return "waiting"
	}
	return parseValue.Local().Format("15:04:05")
}

func renderPeerList(parsePeers map[string]peerStatus) []ui.Node {
	if len(parsePeers) == 0 {
		return []ui.Node{
			html.Li(html.Props{Class: "rounded-2xl border border-dashed border-white/10 bg-slate-950/35 px-4 py-3 text-sm leading-7 text-slate-400"}, html.Text("No remote peers discovered yet. Open this page in a second tab, or reconnect a paused tab to watch hello and result traffic populate the registry.")),
		}
	}

	parseKeys := make([]string, 0, len(parsePeers))
	for parseKey := range parsePeers {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)

	parseNodes := make([]ui.Node, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parsePeer := parsePeers[parseKey2]
		parseLabel := fmt.Sprintf("%s | %s | %s | %s | last seen %s", parsePeer.Identity.Surface, parsePeer.Identity.Role, parsePeer.State, parsePeer.Summary, formatSeenAt(parsePeer.LastSeen))
		parseNodes = append(parseNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseLabel)),
		)
	}
	return parseNodes
}

func renderLogList(parseEntries []string) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseNodes = append(parseNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseEntry)),
		)
	}
	return parseNodes
}

func multiClientPresenceRoot() ui.Node {
	parseLocation, parseErr := interop.GetWindowLocation()
	if parseErr == nil {
		parsePath := strings.ToLower(strings.TrimSpace(parseLocation.Pathname()))
		if strings.Contains(parsePath, "popup") {
			return multiClientPopupSurface()
		}
	}
	return multiClientOpenerSurface()
}

func multiClientOpenerSurface() ui.Node {
	parseSelf := ui.UseState(newClientIdentity("storefront-tab", "storefront"))
	parsePresenceOnline := ui.UseState(true)
	parsePresenceTransport := ui.UseState("pending")
	parsePopupStatus := ui.UseState("Closed")
	parsePopupPeerID := ui.UseState("")
	parsePopupResult := ui.UseState("No popup query result yet.")
	parsePeerRegistry := ui.UseState(map[string]peerStatus{})
	parseLogs := ui.UseState([]string{"Open this page in a second tab, then use the popup to watch hello, query, result, lease expiry, and reconnect behavior."})
	parsePresenceChannelRef := ui.UseRef(interop.CrossTabChannel{})
	parsePresenceCancelRef := ui.UseRef((func())(nil))
	parsePopupChannelRef := ui.UseRef(interop.WindowChannel{})
	parsePopupCancelRef := ui.UseRef((func())(nil))

	parseAppendLog := func(parseLine string) {
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

	parseUpsertPeer := func(parseIdentity interop.ClientIdentity, parseState string, parseSummary string) {
		if strings.TrimSpace(parseIdentity.ID) == "" || parseIdentity.ID == parseSelf.Get().ID {
			return
		}
		parseNext2 := clonePeerStatuses(parsePeerRegistry.Get())
		parseEntry := parseNext2[parseIdentity.ID]
		parseEntry.Identity = parseIdentity
		if strings.TrimSpace(parseState) != "" {
			parseEntry.State = parseState
		}
		parseEntry.LastSeen = time.Now().UTC()
		if strings.TrimSpace(parseSummary) != "" {
			parseEntry.Summary = parseSummary
		}
		parseNext2[parseIdentity.ID] = parseEntry
		parsePeerRegistry.Set(parseNext2)
	}

	parseMarkPeerState := func(parseId2 string, parseState2 string, parseSummary2 string) {
		parseTrimmedID := strings.TrimSpace(parseId2)
		if parseTrimmedID == "" {
			return
		}
		parseCurrent := parsePeerRegistry.Get()
		parseEntry2, parseOk := parseCurrent[parseTrimmedID]
		if !parseOk {
			return
		}
		parseNext3 := clonePeerStatuses(parseCurrent)
		parseEntry2.State = parseState2
		if strings.TrimSpace(parseSummary2) != "" {
			parseEntry2.Summary = parseSummary2
		}
		parseNext3[parseTrimmedID] = parseEntry2
		parsePeerRegistry.Set(parseNext3)
	}

	parseReleasePopup := func(isCloseWindow bool) {
		if parseCancel := parsePopupCancelRef.Get(); parseCancel != nil {
			parseCancel()
			parsePopupCancelRef.Set(nil)
		}
		parseChannel := parsePopupChannelRef.Get()
		if isCloseWindow && parseChannel.Name() != "" && !parseChannel.Closed() {
			_ = interop.PublishClientGoodbyeWindow(parseChannel, parseSelf.Get())
			_ = parseChannel.Close()
		}
		parsePopupChannelRef.Set(interop.WindowChannel{})
		parsePopupPeerID.Set("")
	}

	handlePopupMessage := func(parseChannel9 interop.WindowChannel, parseMessage2 interop.ClientMessage, parseErr10 error) {
		if parseErr10 != nil {
			parseAppendLog(describeError("Popup message failed", parseErr10))
			return
		}
		if parseMessage2.Source.ID == parseSelf.Get().ID {
			return
		}

		switch parseMessage2.Kind {
		case interop.ClientHello:
			parsePopupPeerID.Set(parseMessage2.Source.ID)
			parsePopupStatus.Set("Connected")
			parseAppendLog(fmt.Sprintf("Popup hello from %s (%s)", parseMessage2.Source.Surface, parseMessage2.Source.ID))
		case interop.ClientQuery:
			if parseMessage2.Target != "" && parseMessage2.Target != parseSelf.Get().ID {
				return
			}
			parseResponse := interop.ClientMessage{
				Kind:   interop.ClientResult,
				Topic:  parseMessage2.Topic,
				Source: parseSelf.Get(),
				Target: parseMessage2.Source.ID,
				Payload: map[string]string{
					"status":   "opener-ready",
					"presence": fmt.Sprintf("%t", parsePresenceOnline.Get()),
					"surface":  parseSelf.Get().Surface,
				},
			}
			if parsePublishErr := interop.PublishClientWindowMessage(parseChannel9, parseResponse); parsePublishErr != nil {
				parseAppendLog(describeError("Popup query reply failed", parsePublishErr))
				return
			}
			parseAppendLog("Answered popup query with a targeted window result.")
		case interop.ClientResult:
			if parseMessage2.Target != parseSelf.Get().ID {
				return
			}
			parsePopupStatus.Set("Ready")
			parsePopupResult.Set(describePayload(parseMessage2.Payload))
			parseAppendLog(fmt.Sprintf("Popup result on %s: %s", parseMessage2.Topic, describePayload(parseMessage2.Payload)))
		case interop.ClientGoodbye:
			parsePopupStatus.Set("Disconnected")
			parsePopupPeerID.Set("")
			parseAppendLog("Popup sent goodbye and left the targeted channel.")
		}
	}

	ui.UseEffect(func() func() {
		if !parsePresenceOnline.Get() {
			parsePresenceTransport.Set("offline")
			return nil
		}

		parseChannel2, parseErr := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: multiClientPresenceChannelName})
		if parseErr != nil {
			parseAppendLog(describeError("Presence channel unavailable", parseErr))
			parsePresenceTransport.Set("unavailable")
			return nil
		}

		parsePresenceChannelRef.Set(parseChannel2)
		parsePresenceTransport.Set(parseChannel2.Transport())

		parseSubscription, parseErr := interop.SubscribeClientMessages(parseChannel2, func(parseMessage3 interop.ClientMessage, parseReceiveErr error) {
			if parseReceiveErr != nil {
				parseAppendLog(describeError("Presence receive failed", parseReceiveErr))
				return
			}
			if parseMessage3.Source.ID == parseSelf.Get().ID {
				return
			}

			switch parseMessage3.Kind {
			case interop.ClientHello:
				parseUpsertPeer(parseMessage3.Source, "seen", "hello received")
				parseAppendLog(fmt.Sprintf("Peer hello from %s via %s", parseMessage3.Source.Surface, parseChannel2.Transport()))
			case interop.ClientQuery:
				parseUpsertPeer(parseMessage3.Source, "seen", "discovery query received")
				if parseMessage3.Topic != interop.ClientPresenceTopic {
					return
				}
				if parsePublishErr2 := interop.PublishClientResult(parseChannel2, interop.ClientPresenceTopic, parseSelf.Get(), parseMessage3.Source.ID, map[string]string{
					"surface": parseSelf.Get().Surface,
					"role":    parseSelf.Get().Role,
					"state":   "ready",
				}); parsePublishErr2 != nil {
					parseAppendLog(describeError("Presence query reply failed", parsePublishErr2))
					return
				}
				parseAppendLog(fmt.Sprintf("Answered discovery query for %s.", parseMessage3.Source.Surface))
			case interop.ClientResult:
				if parseMessage3.Target != parseSelf.Get().ID {
					return
				}
				parseUpsertPeer(parseMessage3.Source, "ready", describePayload(parseMessage3.Payload))
				parseAppendLog(fmt.Sprintf("Targeted result from %s: %s", parseMessage3.Source.Surface, describePayload(parseMessage3.Payload)))
			case interop.ClientGoodbye:
				parseMarkPeerState(parseMessage3.Source.ID, "disconnected", "goodbye received")
				parseAppendLog(fmt.Sprintf("Peer goodbye from %s.", parseMessage3.Source.Surface))
			}
		})
		if parseErr != nil {
			parseAppendLog(describeError("Presence subscription failed", parseErr))
			_ = parseChannel2.Close()
			parsePresenceChannelRef.Set(interop.CrossTabChannel{})
			return nil
		}
		parsePresenceCancelRef.Set(parseSubscription.Cancel)

		if parseErr2 := interop.PublishClientHello(parseChannel2, parseSelf.Get()); parseErr2 != nil {
			parseAppendLog(describeError("Initial hello failed", parseErr2))
		} else {
			parseAppendLog("Published local hello on the cross-tab presence channel.")
		}

		if parseErr3 := interop.PublishClientQuery(parseChannel2, interop.ClientPresenceTopic, parseSelf.Get()); parseErr3 != nil {
			parseAppendLog(describeError("Initial discovery query failed", parseErr3))
		} else {
			parseAppendLog("Published query(topic=clients) to discover live peers.")
		}

		parseTimer, parseTimerErr := interop.ScheduleInterval(time.Second, func() {
			parseCurrent2 := parsePeerRegistry.Get()
			if len(parseCurrent2) == 0 {
				return
			}

			parseNow := time.Now().UTC()
			parseNext4 := clonePeerStatuses(parseCurrent2)
			parseExpired := make([]string, 0)
			for parseId, parseEntry3 := range parseCurrent2 {
				if parseEntry3.State == "expired" || parseEntry3.State == "disconnected" {
					continue
				}
				if parseNow.Sub(parseEntry3.LastSeen) <= peerLeaseTimeout {
					continue
				}
				parseEntry3.State = "expired"
				parseEntry3.Summary = "lease expired; waiting for fresh hello"
				parseNext4[parseId] = parseEntry3
				parseExpired = append(parseExpired, parseEntry3.Identity.Surface)
			}
			if len(parseExpired) == 0 {
				return
			}
			parsePeerRegistry.Set(parseNext4)
			for _, parseSurface := range parseExpired {
				parseAppendLog("Peer lease expired: " + parseSurface)
			}
		})

		return func() {
			if parseCancel2 := parsePresenceCancelRef.Get(); parseCancel2 != nil {
				parseCancel2()
				parsePresenceCancelRef.Set(nil)
			}
			if parseTimerErr == nil {
				_ = parseTimer.Cancel()
			}
			_ = interop.PublishClientGoodbye(parseChannel2, parseSelf.Get())
			_ = parseChannel2.Close()
			parsePresenceChannelRef.Set(interop.CrossTabChannel{})
		}
	}, parsePresenceOnline.Get())

	ui.UseEffect(func() func() {
		parseTimer2, parseErr4 := interop.ScheduleInterval(500*time.Millisecond, func() {
			parseChannel3 := parsePopupChannelRef.Get()
			if parseChannel3.Name() == "" {
				return
			}
			if !parseChannel3.Closed() {
				return
			}
			parseReleasePopup(false)
			parsePopupStatus.Set("Closed")
			parseAppendLog("Popup closed unexpectedly. Reopen it to restore targeted coordination.")
		})
		return func() {
			if parseErr4 == nil {
				_ = parseTimer2.Cancel()
			}
			parseReleasePopup(true)
		}
	}, true)

	parseReannounceHello := ui.UseEvent(func() {
		parseChannel4 := parsePresenceChannelRef.Get()
		if parseChannel4.Name() == "" {
			parseAppendLog("Reconnect the presence channel before re-announcing hello.")
			return
		}
		if parseErr5 := interop.PublishClientHello(parseChannel4, parseSelf.Get()); parseErr5 != nil {
			parseAppendLog(describeError("Hello publish failed", parseErr5))
			return
		}
		parseAppendLog("Re-announced hello to refresh presence without a full reconnect.")
	})

	parseQueryPeers := ui.UseEvent(func() {
		parseChannel5 := parsePresenceChannelRef.Get()
		if parseChannel5.Name() == "" {
			parseAppendLog("Reconnect the presence channel before querying peers.")
			return
		}
		if parseErr6 := interop.PublishClientQuery(parseChannel5, interop.ClientPresenceTopic, parseSelf.Get()); parseErr6 != nil {
			parseAppendLog(describeError("Discovery query failed", parseErr6))
			return
		}
		parseAppendLog("Published a fresh clients query for late join discovery.")
	})

	parseDisconnectPresence := ui.UseEvent(func() {
		if !parsePresenceOnline.Get() {
			parseAppendLog("Presence channel is already offline.")
			return
		}
		parsePresenceOnline.Set(false)
		parsePresenceTransport.Set("offline")
		parseAppendLog("Disconnected the local presence channel. Other tabs should eventually mark this lease expired.")
	})

	parseReconnectPresence := ui.UseEvent(func() {
		if parsePresenceOnline.Get() {
			parseAppendLog("Presence channel is already connected.")
			return
		}
		parsePresenceOnline.Set(true)
		parseAppendLog("Reconnecting the local presence channel and replaying hello plus discovery.")
	})

	parseOpenPopup := ui.UseEvent(func() {
		parseCurrent3 := parsePopupChannelRef.Get()
		if parseCurrent3.Name() != "" && !parseCurrent3.Closed() {
			_ = parseCurrent3.Focus()
			parsePopupStatus.Set("Connected")
			parseAppendLog("Reused the existing popup and focused it.")
			return
		}

		parseReleasePopup(false)
		parseChannel6, parseErr7 := interop.OpenSecondaryWindowChannel(interop.WindowChannelOptions{
			URL:      multiClientPopupURL,
			Name:     multiClientPopupChannelName,
			Features: "popup=yes,width=560,height=760",
		})
		if parseErr7 != nil {
			parseAppendLog(describeError("Opening popup failed", parseErr7))
			return
		}

		parseSubscription2, parseErr7 := interop.SubscribeClientWindowMessages(parseChannel6, func(parseMessage4 interop.ClientMessage, parseReceiveErr2 error) {
			handlePopupMessage(parseChannel6, parseMessage4, parseReceiveErr2)
		})
		if parseErr7 != nil {
			_ = parseChannel6.Close()
			parseAppendLog(describeError("Popup subscription failed", parseErr7))
			return
		}

		parsePopupChannelRef.Set(parseChannel6)
		parsePopupCancelRef.Set(parseSubscription2.Cancel)
		parsePopupStatus.Set("Connecting")
		parseAppendLog("Opened the popup inspector and subscribed to its window channel.")

		if parseErr8 := interop.PublishClientHelloWindow(parseChannel6, parseSelf.Get()); parseErr8 != nil {
			parseAppendLog(describeError("Popup hello failed", parseErr8))
			return
		}
		parseAppendLog("Sent opener hello on the popup channel.")
	})

	parseQueryPopup := ui.UseEvent(func() {
		parseChannel7 := parsePopupChannelRef.Get()
		if parseChannel7.Name() == "" || parseChannel7.Closed() {
			parsePopupStatus.Set("Closed")
			parseAppendLog("Open the popup before sending a targeted query.")
			return
		}
		parseTarget := strings.TrimSpace(parsePopupPeerID.Get())
		if parseTarget == "" {
			parseAppendLog("Waiting for popup hello before sending a targeted query.")
			return
		}

		parseMessage := interop.ClientMessage{
			Kind:   interop.ClientQuery,
			Topic:  "popup-status",
			Source: parseSelf.Get(),
			Target: parseTarget,
			Payload: map[string]string{
				"request": "status",
			},
		}
		if parseErr9 := interop.PublishClientWindowMessage(parseChannel7, parseMessage); parseErr9 != nil {
			parseAppendLog(describeError("Popup query failed", parseErr9))
			return
		}
		parseAppendLog("Sent a targeted popup query. The reply should return only to this opener.")
	})

	parseClosePopup := ui.UseEvent(func() {
		parseChannel8 := parsePopupChannelRef.Get()
		if parseChannel8.Name() == "" || parseChannel8.Closed() {
			parsePopupStatus.Set("Closed")
			parseAppendLog("Popup is already closed.")
			parseReleasePopup(false)
			return
		}
		parseReleasePopup(true)
		parsePopupStatus.Set("Closed")
		parseAppendLog("Closed the popup and sent goodbye on the window channel.")
	})

	parsePeerNodes := renderPeerList(parsePeerRegistry.Get())
	parseLogNodes := renderLogList(parseLogs.Get())

	return shared.ExamplePage(
		"Multi-Client Presence",
		"interop multi-client helpers",
		"Demonstrate hello, late join discovery, targeted result replies, lease expiry, and reconnect behavior across cross-tab and popup window channels.",
		shared.ExamplePanel("Cross-tab presence",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", parseSelf.Get().Surface),
				shared.ExampleStat("Role", parseSelf.Get().Role),
				shared.ExampleStat("Presence", map[bool]string{true: "connected", false: "offline"}[parsePresenceOnline.Get()]),
				shared.ExampleStat("Transport", parsePresenceTransport.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Open this example in a second tab. Each tab publishes hello on boot, asks query(topic=clients) for late-join discovery, answers with targeted result replies, and expires peers locally when their lease ages out.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Re-announce hello", parseReannounceHello),
				shared.ExampleButton("Query clients", parseQueryPeers),
				shared.ExampleButton("Disconnect presence", parseDisconnectPresence),
				shared.ExampleButton("Reconnect presence", parseReconnectPresence),
			),
		),
		shared.ExamplePanel("Peer registry",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Lease expiry is local state, not a browser-wide scan. If a tab disconnects and does not re-announce, the remaining tab marks it expired and waits for a fresh hello before treating it as ready again.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, parsePeerNodes...),
		),
		shared.ExamplePanel("Popup handshake",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Popup", parsePopupStatus.Get()),
				shared.ExampleStat("Popup peer id", parsePopupPeerID.Get()),
				shared.ExampleStat("Last popup result", parsePopupResult.Get()),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Open popup", parseOpenPopup),
				shared.ExampleButton("Query popup", parseQueryPopup),
				shared.ExampleButton("Close popup", parseClosePopup),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The popup uses WindowOpenerChannel and the opener uses OpenSecondaryWindowChannel. Both sides exchange hello, then targeted query and result messages rather than pretending the popup is just another broadcast peer.")),
		),
		shared.ExamplePanel("Diagnostics",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Use the log below to watch the exact hello, query, result, goodbye, expiry, and reconnect transitions. This is the intended control-plane shape for multi-client coordination in the browser.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, parseLogNodes...),
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
	parseSelf := ui.UseState(newClientIdentity("popup-inspector", "inspector"))
	parseOpenerID := ui.UseState("")
	parseConnection := ui.UseState("Connecting to opener...")
	parseOrphaned := ui.UseState(false)
	parseLastResult := ui.UseState("No opener result yet.")
	parseLogs := ui.UseState([]string{"This popup waits for the opener hello, then answers targeted queries with result messages."})
	parseChannelRef := ui.UseRef(interop.WindowChannel{})
	parseCancelRef := ui.UseRef((func())(nil))

	parseAppendLog := func(parseLine string) {
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

	handleMessage := func(parseChannel3 interop.WindowChannel, parseMessage2 interop.ClientMessage, parseErr4 error) {
		if parseErr4 != nil {
			parseAppendLog(describeError("Opener message failed", parseErr4))
			return
		}
		if parseMessage2.Source.ID == parseSelf.Get().ID {
			return
		}

		switch parseMessage2.Kind {
		case interop.ClientHello:
			parseOpenerID.Set(parseMessage2.Source.ID)
			parseConnection.Set("Connected")
			parseOrphaned.Set(false)
			parseAppendLog(fmt.Sprintf("Received opener hello from %s.", parseMessage2.Source.Surface))
		case interop.ClientQuery:
			if parseMessage2.Target != "" && parseMessage2.Target != parseSelf.Get().ID {
				return
			}
			parseResponse := interop.ClientMessage{
				Kind:   interop.ClientResult,
				Topic:  parseMessage2.Topic,
				Source: parseSelf.Get(),
				Target: parseMessage2.Source.ID,
				Payload: map[string]string{
					"status":   "popup-ready",
					"orphaned": fmt.Sprintf("%t", parseOrphaned.Get()),
					"surface":  parseSelf.Get().Surface,
				},
			}
			if parsePublishErr := interop.PublishClientWindowMessage(parseChannel3, parseResponse); parsePublishErr != nil {
				parseAppendLog(describeError("Popup result publish failed", parsePublishErr))
				return
			}
			parseAppendLog("Answered opener query with a targeted popup result.")
		case interop.ClientResult:
			if parseMessage2.Target != parseSelf.Get().ID {
				return
			}
			parseLastResult.Set(describePayload(parseMessage2.Payload))
			parseAppendLog(fmt.Sprintf("Received targeted result on %s: %s", parseMessage2.Topic, describePayload(parseMessage2.Payload)))
		case interop.ClientGoodbye:
			parseOrphaned.Set(true)
			parseConnection.Set("Opener closed")
			parseAppendLog("Opener sent goodbye. This popup is now orphaned.")
		}
	}

	ui.UseEffect(func() func() {
		parseChannel, parseErr := interop.OpenWindowOpenerChannel(interop.WindowChannelOptions{Name: multiClientPopupChannelName})
		if parseErr != nil {
			parseOrphaned.Set(true)
			parseConnection.Set("Opened without an opener")
			parseAppendLog(describeError("No opener channel available", parseErr))
			return nil
		}

		parseChannelRef.Set(parseChannel)
		parseOrphaned.Set(parseChannel.Closed())
		parseConnection.Set("Connected")

		parseSubscription, parseErr := interop.SubscribeClientWindowMessages(parseChannel, func(parseMessage3 interop.ClientMessage, parseReceiveErr error) {
			handleMessage(parseChannel, parseMessage3, parseReceiveErr)
		})
		if parseErr != nil {
			parseConnection.Set("Subscription failed")
			parseAppendLog(describeError("Popup subscription failed", parseErr))
			return nil
		}
		parseCancelRef.Set(parseSubscription.Cancel)

		if parseErr2 := interop.PublishClientHelloWindow(parseChannel, parseSelf.Get()); parseErr2 != nil {
			parseAppendLog(describeError("Popup hello failed", parseErr2))
		} else {
			parseAppendLog("Published popup hello to the opener channel.")
		}

		parseTimer, parseTimerErr := interop.ScheduleInterval(500*time.Millisecond, func() {
			parseCurrent := parseChannelRef.Get()
			if parseCurrent.Name() == "" {
				return
			}
			if !parseCurrent.Closed() {
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

	parseQueryOpener := ui.UseEvent(func() {
		parseChannel2 := parseChannelRef.Get()
		if parseChannel2.Name() == "" || parseChannel2.Closed() {
			parseOrphaned.Set(true)
			parseConnection.Set("Opener unavailable")
			parseAppendLog("Opener is unavailable; the popup cannot send a targeted query.")
			return
		}
		parseTarget := strings.TrimSpace(parseOpenerID.Get())
		if parseTarget == "" {
			parseAppendLog("Waiting for opener hello before sending a targeted query.")
			return
		}

		parseMessage := interop.ClientMessage{
			Kind:   interop.ClientQuery,
			Topic:  "opener-status",
			Source: parseSelf.Get(),
			Target: parseTarget,
			Payload: map[string]string{
				"request": "status",
			},
		}
		if parseErr3 := interop.PublishClientWindowMessage(parseChannel2, parseMessage); parseErr3 != nil {
			parseAppendLog(describeError("Opener query failed", parseErr3))
			return
		}
		parseAppendLog("Sent a targeted opener query from the popup.")
	})

	parseLogNodes := renderLogList(parseLogs.Get())
	parseOrphanMessage := "The opener is still present and can answer targeted queries."
	if parseOrphaned.Get() {
		parseOrphanMessage = "The opener disappeared or this page was opened directly. Keep diagnostics visible, but do not assume the coordinator still exists."
	}

	return shared.ExamplePage(
		"Popup Inspector",
		"interop.WindowOpenerChannel and multi-client messages",
		"This popup exchanges hello, query, result, and goodbye messages with its opener without pretending the child surface shares the opener's lifecycle or authority.",
		shared.ExamplePanel("Popup state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Client", parseSelf.Get().Surface),
				shared.ExampleStat("Role", parseSelf.Get().Role),
				shared.ExampleStat("Connection", parseConnection.Get()),
				shared.ExampleStat("Opener id", parseOpenerID.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseOrphanMessage)),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Query opener", parseQueryOpener),
			),
		),
		shared.ExamplePanel("Latest opener result",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Targeted result", parseLastResult.Get()),
				shared.ExampleStat("Orphaned", fmt.Sprintf("%t", parseOrphaned.Get())),
			),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, parseLogNodes...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiClientPresenceRoot), "#app")
	select {}
}
