//go:build js && wasm
// +build js,wasm

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type FocusOptions struct {
	PreventScroll bool
}

type FocusTrapOptions struct {
	ContainerSelector     string
	InitialFocusSelector  string
	FallbackFocusSelector string
	Active                bool
	RestoreFocus          bool
}

type FocusManager struct {
	remembered Ref[interface{}]
}

type CompositeItem struct {
	ID       string
	Text     string
	Disabled bool
}

type CompositeNavigationOptions struct {
	Orientation  string
	Loop         bool
	InitialIndex int
}

type CompositeNavigation struct {
	items     []CompositeItem
	options   CompositeNavigationOptions
	active    State[int]
	typeahead Ref[compositeTypeaheadState]
}

type compositeTypeaheadState struct {
	query string
	last  time.Time
}

type AnnouncementMode string

const (
	AnnouncementPolite    AnnouncementMode = "polite"
	AnnouncementAssertive AnnouncementMode = "assertive"
)

type announcementState struct {
	message  string
	sequence int
}

type Announcer struct {
	polite      State[announcementState]
	assertive   State[announcementState]
	politeID    string
	assertiveID string
}

type AccessibleOverlayProps struct {
	Open                  bool
	Target                PortalTarget
	AppRootSelector       string
	SurfaceID             string
	Kind                  OverlayKind
	LabelledBy            string
	DescribedBy           string
	Role                  string
	Modal                 bool
	InitialFocusSelector  string
	FallbackFocusSelector string
	RestoreFocus          bool
	TrapFocus             bool
	CloseOnEscape         bool
	CloseOnOutsideClick   bool
	LockScroll            bool
	BackgroundInert       bool
	Backdrop              bool
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

func UseFocusManager() FocusManager {
	return FocusManager{remembered: UseRef((interface{})(nil))}
}

func (m FocusManager) FocusFirstError(errors FieldErrors, fieldIDs map[string]string, order ...string) bool {
	for _, name := range order {
		if errors[name] == "" {
			continue
		}
		if id := fieldIDs[name]; id != "" && m.FocusByID(id) {
			return true
		}
	}
	for name := range errors {
		if id := fieldIDs[name]; id != "" && m.FocusByID(id) {
			return true
		}
	}
	return false
}

func UseCompositeNavigation(items []CompositeItem, options ...CompositeNavigationOptions) CompositeNavigation {
	resolved := CompositeNavigationOptions{
		Orientation:  "both",
		Loop:         true,
		InitialIndex: 0,
	}
	if len(options) > 0 {
		if options[0].Orientation != "" {
			resolved.Orientation = strings.ToLower(options[0].Orientation)
		}
		resolved.Loop = options[0].Loop
		resolved.InitialIndex = options[0].InitialIndex
	}
	if resolved.Orientation == "" {
		resolved.Orientation = "both"
	}
	cloned := append([]CompositeItem(nil), items...)
	return CompositeNavigation{
		items:     cloned,
		options:   resolved,
		active:    UseState(compositeInitialIndex(cloned, resolved.InitialIndex)),
		typeahead: UseRef(compositeTypeaheadState{}),
	}
}

func (n CompositeNavigation) ActiveIndex() int {
	if n.active.get == nil {
		return -1
	}
	return compositeNormalizeIndex(n.items, n.active.Get(), n.options.InitialIndex)
}

func (n CompositeNavigation) ActiveID() string {
	index := n.ActiveIndex()
	if index < 0 || index >= len(n.items) {
		return ""
	}
	return n.items[index].ID
}

func (n CompositeNavigation) ActiveDescendant() string {
	return n.ActiveID()
}

func (n CompositeNavigation) IsActive(index int) bool {
	return index >= 0 && index == n.ActiveIndex()
}

func (n CompositeNavigation) TabIndex(index int) int {
	if index < 0 || index >= len(n.items) || n.items[index].Disabled {
		return -1
	}
	if n.IsActive(index) {
		return 0
	}
	return -1
}

func (n CompositeNavigation) SetActive(index int) {
	if n.active.get == nil {
		return
	}
	if index < 0 || index >= len(n.items) || n.items[index].Disabled {
		return
	}
	n.active.Set(index)
}

func (n CompositeNavigation) MoveNext() {
	n.move(1)
}

func (n CompositeNavigation) MovePrevious() {
	n.move(-1)
}

func (n CompositeNavigation) MoveHome() {
	index := compositeFirstEnabled(n.items)
	if index >= 0 {
		n.SetActive(index)
	}
}

func (n CompositeNavigation) MoveEnd() {
	index := compositeLastEnabled(n.items)
	if index >= 0 {
		n.SetActive(index)
	}
}

func (n CompositeNavigation) OnKeyDown(event KeyboardEvent) {
	key := event.GetKey()
	switch key {
	case "ArrowRight":
		if n.options.Orientation == "horizontal" || n.options.Orientation == "both" {
			event.PreventDefault()
			n.MoveNext()
		}
	case "ArrowDown":
		if n.options.Orientation == "vertical" || n.options.Orientation == "both" {
			event.PreventDefault()
			n.MoveNext()
		}
	case "ArrowLeft":
		if n.options.Orientation == "horizontal" || n.options.Orientation == "both" {
			event.PreventDefault()
			n.MovePrevious()
		}
	case "ArrowUp":
		if n.options.Orientation == "vertical" || n.options.Orientation == "both" {
			event.PreventDefault()
			n.MovePrevious()
		}
	case "Home":
		event.PreventDefault()
		n.MoveHome()
	case "End":
		event.PreventDefault()
		n.MoveEnd()
	default:
		if compositeIsTypeaheadKey(key) {
			event.PreventDefault()
			n.applyTypeahead(key)
		}
	}
}

func (n CompositeNavigation) move(step int) {
	if n.active.get == nil || len(n.items) == 0 {
		return
	}
	current := n.ActiveIndex()
	if current < 0 {
		current = compositeInitialIndex(n.items, n.options.InitialIndex)
	}
	next := compositeNextEnabled(n.items, current, step, n.options.Loop)
	if next >= 0 {
		n.active.Set(next)
	}
}

func (n CompositeNavigation) applyTypeahead(fragment string) {
	if len(n.items) == 0 {
		return
	}
	state := n.typeahead.Get()
	now := time.Now()
	query := strings.ToLower(fragment)
	if now.Sub(state.last) < 700*time.Millisecond {
		query = state.query + query
	}
	state.query = query
	state.last = now
	n.typeahead.Set(state)

	start := n.ActiveIndex()
	if start < 0 {
		start = 0
	}
	for offset := 1; offset <= len(n.items); offset++ {
		candidate := (start + offset) % len(n.items)
		item := n.items[candidate]
		if item.Disabled {
			continue
		}
		if strings.HasPrefix(strings.ToLower(item.Text), query) {
			n.active.Set(candidate)
			return
		}
	}
	for index, item := range n.items {
		if item.Disabled {
			continue
		}
		if strings.HasPrefix(strings.ToLower(item.Text), strings.ToLower(fragment)) {
			n.active.Set(index)
			state.query = strings.ToLower(fragment)
			n.typeahead.Set(state)
			return
		}
	}
}

func UseAnnouncer() Announcer {
	politeID := UseId() + "-polite"
	assertiveID := UseId() + "-assertive"
	return Announcer{
		polite:      UseState(announcementState{}),
		assertive:   UseState(announcementState{}),
		politeID:    politeID,
		assertiveID: assertiveID,
	}
}

func (a Announcer) Announce(mode AnnouncementMode, message string) {
	switch mode {
	case AnnouncementAssertive:
		a.Assertive(message)
	default:
		a.Polite(message)
	}
}

func (a Announcer) Polite(message string) {
	if a.polite.get == nil {
		return
	}
	a.polite.Update(func(previous announcementState) announcementState {
		previous.message = message
		previous.sequence++
		return previous
	})
}

func (a Announcer) Assertive(message string) {
	if a.assertive.get == nil {
		return
	}
	a.assertive.Update(func(previous announcementState) announcementState {
		previous.message = message
		previous.sequence++
		return previous
	})
}

func (a Announcer) Clear() {
	if a.polite.get != nil {
		a.polite.Set(announcementState{})
	}
	if a.assertive.get != nil {
		a.assertive.Set(announcementState{})
	}
}

func (a Announcer) PoliteID() string {
	return a.politeID
}

func (a Announcer) AssertiveID() string {
	return a.assertiveID
}

func (a Announcer) Region() Node {
	politeState := announcementState{}
	assertiveState := announcementState{}
	if a.polite.get != nil {
		politeState = a.polite.Get()
	}
	if a.assertive.get != nil {
		assertiveState = a.assertive.Get()
	}

	return runtime.CreateElement("div", nil,
		announcementRegionNode(a.politeID, string(AnnouncementPolite), politeState),
		announcementRegionNode(a.assertiveID, string(AnnouncementAssertive), assertiveState),
	)
}

func announcementRegionNode(id string, mode string, state announcementState) Node {
	return runtime.CreateElement("div", map[string]interface{}{
		"id":          id,
		"role":        "status",
		"aria-live":   mode,
		"aria-atomic": "true",
		"style": map[string]string{
			"position":   "absolute",
			"width":      "1px",
			"height":     "1px",
			"padding":    "0",
			"margin":     "-1px",
			"overflow":   "hidden",
			"clip":       "rect(0 0 0 0)",
			"whiteSpace": "nowrap",
			"border":     "0",
		},
	},
		runtime.CreateElement("span", nil, state.message),
		runtime.CreateElement("span", map[string]interface{}{"aria-hidden": "true"}, fmt.Sprintf("-%d", state.sequence)),
	)
}

func AccessibleOverlay(props AccessibleOverlayProps) Node {
	modal := props.Modal
	if props.Role == "dialog" && !props.Modal {
		modal = true
	}
	return Overlay(OverlayProps{
		Open:                  props.Open,
		Target:                props.Target,
		AppRootSelector:       props.AppRootSelector,
		SurfaceID:             props.SurfaceID,
		Kind:                  props.Kind,
		Role:                  props.Role,
		LabelledBy:            props.LabelledBy,
		DescribedBy:           props.DescribedBy,
		InitialFocusSelector:  props.InitialFocusSelector,
		FallbackFocusSelector: props.FallbackFocusSelector,
		Modal:                 modal,
		Backdrop:              props.Backdrop,
		TrapFocus:             props.TrapFocus,
		RestoreFocus:          props.RestoreFocus,
		CloseOnEscape:         props.CloseOnEscape,
		CloseOnOutsideClick:   props.CloseOnOutsideClick,
		LockScroll:            props.LockScroll,
		BackgroundInert:       props.BackgroundInert,
		BaseZIndex:            props.BaseZIndex,
		AnchorSelector:        props.AnchorSelector,
		Positioning:           props.Positioning,
		BackdropClass:         props.BackdropClass,
		SurfaceClass:          props.SurfaceClass,
		BackdropStyle:         props.BackdropStyle,
		SurfaceStyle:          props.SurfaceStyle,
		Child:                 props.Child,
		Children:              props.Children,
		OnDismiss:             props.OnDismiss,
	})
}

func compositeNormalizeIndex(items []CompositeItem, current int, preferred int) int {
	if current >= 0 && current < len(items) && !items[current].Disabled {
		return current
	}
	return compositeInitialIndex(items, preferred)
}

func compositeInitialIndex(items []CompositeItem, preferred int) int {
	if preferred >= 0 && preferred < len(items) && !items[preferred].Disabled {
		return preferred
	}
	return compositeFirstEnabled(items)
}

func compositeFirstEnabled(items []CompositeItem) int {
	for index, item := range items {
		if !item.Disabled {
			return index
		}
	}
	return -1
}

func compositeLastEnabled(items []CompositeItem) int {
	for index := len(items) - 1; index >= 0; index-- {
		if !items[index].Disabled {
			return index
		}
	}
	return -1
}

func compositeNextEnabled(items []CompositeItem, start int, step int, loop bool) int {
	if len(items) == 0 {
		return -1
	}
	index := start
	for attempts := 0; attempts < len(items); attempts++ {
		index += step
		if index < 0 {
			if !loop {
				return compositeFirstEnabled(items)
			}
			index = len(items) - 1
		}
		if index >= len(items) {
			if !loop {
				return compositeLastEnabled(items)
			}
			index = 0
		}
		if !items[index].Disabled {
			return index
		}
	}
	return -1
}

func compositeIsTypeaheadKey(key string) bool {
	if len([]rune(key)) != 1 {
		return false
	}
	r := []rune(key)[0]
	if r == ' ' {
		return false
	}
	return (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
