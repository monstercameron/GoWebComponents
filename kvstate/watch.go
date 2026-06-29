package kvstate

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// Cross-tab consistency uses a BroadcastChannel (interop.OpenCrossTabChannel),
// not "storage" events — SQLite writes do not fire those. After a durable Save a
// tab broadcasts {key, version}; other tabs re-Load that key. On native the
// interop channel degrades and these become no-ops.

type crossTabHub struct {
	ch        interop.CrossTabChannel
	available bool
	mu        sync.Mutex
	nextID    int
	subs      map[string]map[int]func()
}

var (
	hubsMu sync.Mutex
	hubs   = map[string]*crossTabHub{}
)

func getHub(parseName string) *crossTabHub {
	hubsMu.Lock()
	defer hubsMu.Unlock()
	if parseHub, parseOK := hubs[parseName]; parseOK {
		return parseHub
	}

	parseHub := &crossTabHub{subs: map[string]map[int]func(){}}
	parseChannel, parseErr := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{
		Name: "kvstate:" + parseName,
	})
	if parseErr == nil {
		parseHub.ch = parseChannel
		parseHub.available = true
		parseChannel.Subscribe(func(parseEnv interop.CrossTabEnvelope, parseSubErr error) {
			if parseSubErr != nil {
				return
			}
			parsePayload, parseOK := parseEnv.Payload.(map[string]any)
			if !parseOK {
				return
			}
			parseKey, _ := parsePayload["key"].(string)
			parseHub.fire(parseKey)
		})
	}
	hubs[parseName] = parseHub
	return parseHub
}

func (parseH *crossTabHub) fire(parseKey string) {
	parseH.mu.Lock()
	parseCallbacks := make([]func(), 0, len(parseH.subs[parseKey]))
	for _, parseFn := range parseH.subs[parseKey] {
		parseCallbacks = append(parseCallbacks, parseFn)
	}
	parseH.mu.Unlock()
	for _, parseFn := range parseCallbacks {
		parseFn()
	}
}

// subscribeCrossTab registers onChange for cross-tab writes to key; the returned
// function unsubscribes.
func subscribeCrossTab(parseName, parseKey string, parseOnChange func()) func() {
	parseHub := getHub(parseName)
	parseHub.mu.Lock()
	parseID := parseHub.nextID
	parseHub.nextID++
	if parseHub.subs[parseKey] == nil {
		parseHub.subs[parseKey] = map[int]func(){}
	}
	parseHub.subs[parseKey][parseID] = parseOnChange
	parseHub.mu.Unlock()

	return func() {
		parseHub.mu.Lock()
		delete(parseHub.subs[parseKey], parseID)
		parseHub.mu.Unlock()
	}
}

// broadcastCrossTab notifies other tabs that key changed.
func broadcastCrossTab(parseName, parseKey string, parseVersion int64) {
	parseHub := getHub(parseName)
	if !parseHub.available {
		return
	}
	parseHub.ch.Publish(map[string]any{"key": parseKey, "version": parseVersion})
}
