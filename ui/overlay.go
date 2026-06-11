package ui

import (
	"sort"
	"sync"
)

// OverlayKind describes the semantic role a layered surface plays in the shared overlay stack.
type OverlayKind string

const (
	OverlayKindDialog  OverlayKind = "dialog"
	OverlayKindPopover OverlayKind = "popover"
	OverlayKindTooltip OverlayKind = "tooltip"
	OverlayKindMenu    OverlayKind = "menu"
	OverlayKindSheet   OverlayKind = "sheet"
	OverlayKindCustom  OverlayKind = "custom"
)

// OverlayStackOptions configures shared stack registration for a layered surface.
type OverlayStackOptions struct {
	ID                  string
	Open                bool
	Kind                OverlayKind
	BaseZIndex          int
	TrapFocus           bool
	CloseOnEscape       bool
	CloseOnOutsideClick bool
}

// OverlayStack reports where the current surface sits in the shared overlay stack.
type OverlayStack struct {
	ID                  string
	Kind                OverlayKind
	Depth               int
	LayerCount          int
	IsTop               bool
	HandlesEscape       bool
	HandlesOutsideClick bool
	TrapFocusActive     bool
	BackdropZIndex      int
	SurfaceZIndex       int
}

// OverlayProps configures a stack-aware layered surface that can optionally render through a portal target.
type OverlayProps struct {
	Open                  bool
	Target                PortalTarget
	AppRootSelector       string
	SurfaceID             string
	Kind                  OverlayKind
	Role                  string
	LabelledBy            string
	DescribedBy           string
	InitialFocusSelector  string
	FallbackFocusSelector string
	Modal                 bool
	Backdrop              bool
	TrapFocus             bool
	RestoreFocus          bool
	CloseOnEscape         bool
	CloseOnOutsideClick   bool
	LockScroll            bool
	BackgroundInert       bool
	BaseZIndex            int
	AnchorSelector        string
	Positioning           string
	BackdropClass         string
	SurfaceClass          string
	BackdropStyle         map[string]string
	SurfaceStyle          map[string]string
	Child                 Node
	Children              []Node
	OnDismiss             func()
}

type overlayManagerRegistration struct {
	ID                  string
	Kind                OverlayKind
	BaseZIndex          int
	TrapFocus           bool
	CloseOnEscape       bool
	CloseOnOutsideClick bool
	Sequence            int
}

type overlaySubscriber struct {
	id     int
	notify func()
}

type overlayStackManager struct {
	mu             sync.Mutex
	entries        map[string]overlayManagerRegistration
	nextSequence   int
	nextSubscriber int
	subscribers    []overlaySubscriber
}

var globalOverlayStackManager = newOverlayStackManager()

// newOverlayStackManager is a core package helper.
func newOverlayStackManager() *overlayStackManager {
	return &overlayStackManager{entries: map[string]overlayManagerRegistration{}, nextSubscriber: 0}
}

// upsert is a core package helper.
func (parseM *overlayStackManager) upsert(parseRegistration overlayManagerRegistration) {
	parseRegistration.Kind = normalizeOverlayKind(parseRegistration.Kind)
	parseRegistration.BaseZIndex = normalizeOverlayBaseZIndex(parseRegistration.BaseZIndex)

	parseM.mu.Lock()
	parsePrevious, parseExists := parseM.entries[parseRegistration.ID]
	if parseExists {
		parseRegistration.Sequence = parsePrevious.Sequence
		if parsePrevious == parseRegistration {
			parseM.mu.Unlock()
			return
		}
	} else {
		parseM.nextSequence++
		parseRegistration.Sequence = parseM.nextSequence
	}
	parseM.entries[parseRegistration.ID] = parseRegistration
	parseSubscribers := append([]overlaySubscriber(nil), parseM.subscribers...)
	parseM.mu.Unlock()
	parseM.notify(parseSubscribers)
}

// snapshot is a core package helper.
func (parseM *overlayStackManager) snapshot(parseId string, parseFallback overlayManagerRegistration, isIncludeFallback bool) OverlayStack {
	parseM.mu.Lock()
	parseOrdered := make([]overlayManagerRegistration, 0, len(parseM.entries)+1)
	for _, parseEntry := range parseM.entries {
		parseOrdered = append(parseOrdered, parseEntry)
	}
	parseM.mu.Unlock()

	sort.Slice(parseOrdered, func(parseLeft, parseRight int) bool {
		return parseOrdered[parseLeft].Sequence < parseOrdered[parseRight].Sequence
	})

	parseDepth := -1
	for parseIndex, parseEntry2 := range parseOrdered {
		if parseEntry2.ID == parseId {
			parseDepth = parseIndex
			break
		}
	}
	if isIncludeFallback && parseDepth == -1 {
		parseFallback.Kind = normalizeOverlayKind(parseFallback.Kind)
		parseFallback.BaseZIndex = normalizeOverlayBaseZIndex(parseFallback.BaseZIndex)
		parseOrdered = append(parseOrdered, parseFallback)
		parseDepth = len(parseOrdered) - 1
	}

	parseBaseZIndex := normalizeOverlayBaseZIndex(parseFallback.BaseZIndex)
	if parseDepth >= 0 && parseDepth < len(parseOrdered) {
		parseBaseZIndex = parseOrdered[parseDepth].BaseZIndex
	}

	parseStack := OverlayStack{
		ID:             parseId,
		Kind:           normalizeOverlayKind(parseFallback.Kind),
		Depth:          parseDepth,
		LayerCount:     len(parseOrdered),
		BackdropZIndex: parseBaseZIndex,
		SurfaceZIndex:  parseBaseZIndex + 1,
	}
	if parseDepth == -1 {
		return parseStack
	}
	parseStack.Kind = parseOrdered[parseDepth].Kind
	parseStack.BackdropZIndex = parseBaseZIndex + (parseDepth * 2)
	parseStack.SurfaceZIndex = parseStack.BackdropZIndex + 1
	parseStack.IsTop = parseDepth == len(parseOrdered)-1
	parseStack.HandlesEscape = overlayTopMatchID(parseOrdered, func(parseEntry3 overlayManagerRegistration) bool {
		return parseEntry3.CloseOnEscape
	}) == parseId
	parseStack.HandlesOutsideClick = overlayTopMatchID(parseOrdered, func(parseEntry4 overlayManagerRegistration) bool {
		return parseEntry4.CloseOnOutsideClick
	}) == parseId
	parseStack.TrapFocusActive = overlayTopMatchID(parseOrdered, func(parseEntry5 overlayManagerRegistration) bool {
		return parseEntry5.TrapFocus
	}) == parseId
	return parseStack
}

// notify is a core package helper.
func (parseM *overlayStackManager) notify(parseSubscribers []overlaySubscriber) {
	for _, parseSubscriber := range parseSubscribers {
		if parseSubscriber.id > 0 && parseSubscriber.notify != nil {
			parseSubscriber.notify()
		}
	}
}

// overlayTopMatchID is a core package helper.
func overlayTopMatchID(parseEntries []overlayManagerRegistration, parseMatch func(overlayManagerRegistration) bool) string {
	for parseIndex := len(parseEntries) - 1; parseIndex >= 0; parseIndex-- {
		if parseMatch(parseEntries[parseIndex]) {
			return parseEntries[parseIndex].ID
		}
	}
	return ""
}

// normalizeOverlayBaseZIndex is a core package helper.
func normalizeOverlayBaseZIndex(parseBase int) int {
	if parseBase <= 0 {
		return 1000
	}
	return parseBase
}

// normalizeOverlayKind is a core package helper.
func normalizeOverlayKind(parseKind OverlayKind) OverlayKind {
	if parseKind == "" {
		return OverlayKindCustom
	}
	return parseKind
}
