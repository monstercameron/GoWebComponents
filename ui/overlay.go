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

func newOverlayStackManager() *overlayStackManager {
	return &overlayStackManager{entries: map[string]overlayManagerRegistration{}}
}

func (m *overlayStackManager) subscribe(notify func()) func() {
	if notify == nil {
		return func() {}
	}
	m.mu.Lock()
	m.nextSubscriber++
	subscriberID := m.nextSubscriber
	m.subscribers = append(m.subscribers, overlaySubscriber{id: subscriberID, notify: notify})
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for index, subscriber := range m.subscribers {
			if subscriber.id == subscriberID {
				m.subscribers = append(m.subscribers[:index], m.subscribers[index+1:]...)
				return
			}
		}
	}
}

func (m *overlayStackManager) upsert(registration overlayManagerRegistration) {
	registration.Kind = normalizeOverlayKind(registration.Kind)
	registration.BaseZIndex = normalizeOverlayBaseZIndex(registration.BaseZIndex)

	m.mu.Lock()
	previous, exists := m.entries[registration.ID]
	if exists {
		registration.Sequence = previous.Sequence
		if previous == registration {
			m.mu.Unlock()
			return
		}
	} else {
		m.nextSequence++
		registration.Sequence = m.nextSequence
	}
	m.entries[registration.ID] = registration
	subscribers := append([]overlaySubscriber(nil), m.subscribers...)
	m.mu.Unlock()
	m.notify(subscribers)
}

func (m *overlayStackManager) remove(id string) {
	if id == "" {
		return
	}
	m.mu.Lock()
	if _, exists := m.entries[id]; !exists {
		m.mu.Unlock()
		return
	}
	delete(m.entries, id)
	subscribers := append([]overlaySubscriber(nil), m.subscribers...)
	m.mu.Unlock()
	m.notify(subscribers)
}

func (m *overlayStackManager) snapshot(id string, fallback overlayManagerRegistration, includeFallback bool) OverlayStack {
	m.mu.Lock()
	ordered := make([]overlayManagerRegistration, 0, len(m.entries)+1)
	for _, entry := range m.entries {
		ordered = append(ordered, entry)
	}
	m.mu.Unlock()

	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].Sequence < ordered[right].Sequence
	})

	depth := -1
	for index, entry := range ordered {
		if entry.ID == id {
			depth = index
			break
		}
	}
	if includeFallback && depth == -1 {
		fallback.Kind = normalizeOverlayKind(fallback.Kind)
		fallback.BaseZIndex = normalizeOverlayBaseZIndex(fallback.BaseZIndex)
		ordered = append(ordered, fallback)
		depth = len(ordered) - 1
	}

	baseZIndex := normalizeOverlayBaseZIndex(fallback.BaseZIndex)
	if depth >= 0 && depth < len(ordered) {
		baseZIndex = ordered[depth].BaseZIndex
	}

	stack := OverlayStack{
		ID:             id,
		Kind:           normalizeOverlayKind(fallback.Kind),
		Depth:          depth,
		LayerCount:     len(ordered),
		BackdropZIndex: baseZIndex,
		SurfaceZIndex:  baseZIndex + 1,
	}
	if depth == -1 {
		return stack
	}
	stack.Kind = ordered[depth].Kind
	stack.BackdropZIndex = baseZIndex + (depth * 2)
	stack.SurfaceZIndex = stack.BackdropZIndex + 1
	stack.IsTop = depth == len(ordered)-1
	stack.HandlesEscape = overlayTopMatchID(ordered, func(entry overlayManagerRegistration) bool {
		return entry.CloseOnEscape
	}) == id
	stack.HandlesOutsideClick = overlayTopMatchID(ordered, func(entry overlayManagerRegistration) bool {
		return entry.CloseOnOutsideClick
	}) == id
	stack.TrapFocusActive = overlayTopMatchID(ordered, func(entry overlayManagerRegistration) bool {
		return entry.TrapFocus
	}) == id
	return stack
}

func (m *overlayStackManager) notify(subscribers []overlaySubscriber) {
	for _, subscriber := range subscribers {
		if subscriber.notify != nil {
			callback := subscriber.notify
			go callback()
		}
	}
}

func overlayTopMatchID(entries []overlayManagerRegistration, match func(overlayManagerRegistration) bool) string {
	for index := len(entries) - 1; index >= 0; index-- {
		if match(entries[index]) {
			return entries[index].ID
		}
	}
	return ""
}

func normalizeOverlayBaseZIndex(base int) int {
	if base <= 0 {
		return 1000
	}
	return base
}

func normalizeOverlayKind(kind OverlayKind) OverlayKind {
	if kind == "" {
		return OverlayKindCustom
	}
	return kind
}

func overlayRole(kind OverlayKind, explicit string) string {
	if explicit != "" {
		return explicit
	}
	switch kind {
	case OverlayKindTooltip:
		return "tooltip"
	case OverlayKindMenu:
		return "menu"
	default:
		return "dialog"
	}
}

func cloneOverlayStyle(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(source)+1)
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
