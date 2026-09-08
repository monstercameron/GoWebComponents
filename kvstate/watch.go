package kvstate

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/interop"
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
	hubsMu       sync.Mutex
	hubs         = map[string]*crossTabHub{}
	externalHubs = map[string]*crossTabHub{}
)

// subscribeExternalHub owns a local-only hub until its last binding unsubscribes.
func subscribeExternalHub(parseName, parseKey string, parseOnChange func()) func() {
	hubsMu.Lock()
	parseHub := externalHubs[parseName]
	if parseHub == nil {
		parseHub = &crossTabHub{subs: map[string]map[int]func(){}}
		externalHubs[parseName] = parseHub
	}
	parseStop := subscribeHub(parseHub, parseKey, parseOnChange)
	hubsMu.Unlock()
	return func() {
		hubsMu.Lock()
		defer hubsMu.Unlock()
		parseStop()
		parseHub.mu.Lock()
		defer parseHub.mu.Unlock()
		if len(parseHub.subs) == 0 && externalHubs[parseName] == parseHub {
			delete(externalHubs, parseName)
		}
	}
}

// subscribeBinding selects external invalidation only when explicitly configured.
func subscribeBinding(parseOptions Options, parseKey string, parseOnChange func()) func() {
	if parseOptions.ExternalInvalidation {
		return subscribeExternalHub(parseOptions.Name, parseKey, parseOnChange)
	}
	return subscribeCrossTab(parseOptions.Name, parseKey, parseOnChange)
}

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
	return subscribeHub(parseHub, parseKey, parseOnChange)
}

// subscribeHub registers local reload callbacks with either transport policy.
func subscribeHub(parseHub *crossTabHub, parseKey string, parseOnChange func()) func() {
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
		// Drop the empty inner map too: keys can be dynamic, and the empty
		// husks otherwise accumulate for the hub's lifetime.
		if len(parseHub.subs[parseKey]) == 0 {
			delete(parseHub.subs, parseKey)
		}
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
