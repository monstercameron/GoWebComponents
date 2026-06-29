//go:build js && wasm

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
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

// UseFocusManager creates a FocusManager backed by a component ref for tracking and restoring focus.
func UseFocusManager() FocusManager {
	return FocusManager{remembered: UseRef((interface{})(nil))}
}

// FocusFirstError is a core package helper.
func (parseM FocusManager) FocusFirstError(parseErrors FieldErrors, parseFieldIDs map[string]string, parseOrder ...string) bool {
	for _, parseName := range parseOrder {
		if parseErrors[parseName] == "" {
			continue
		}
		if parseId := parseFieldIDs[parseName]; parseId != "" && parseM.FocusByID(parseId) {
			return true
		}
	}
	for parseName2 := range parseErrors {
		if parseId2 := parseFieldIDs[parseName2]; parseId2 != "" && parseM.FocusByID(parseId2) {
			return true
		}
	}
	return false
}

// UseCompositeNavigation creates a CompositeNavigation with keyboard and typeahead navigation for a list of items.
func UseCompositeNavigation(parseItems []CompositeItem, parseOptions ...CompositeNavigationOptions) CompositeNavigation {
	parseResolved := CompositeNavigationOptions{
		Orientation:  "both",
		Loop:         true,
		InitialIndex: 0,
	}
	if len(parseOptions) > 0 {
		if parseOptions[0].Orientation != "" {
			parseResolved.Orientation = strings.ToLower(parseOptions[0].Orientation)
		}
		parseResolved.Loop = parseOptions[0].Loop
		parseResolved.InitialIndex = parseOptions[0].InitialIndex
	}
	if parseResolved.Orientation == "" {
		parseResolved.Orientation = "both"
	}
	parseCloned := append([]CompositeItem(nil), parseItems...)
	return CompositeNavigation{
		items:     parseCloned,
		options:   parseResolved,
		active:    UseState(compositeInitialIndex(parseCloned, parseResolved.InitialIndex)),
		typeahead: UseRef(compositeTypeaheadState{}),
	}
}

// ActiveIndex is a core package helper.
func (parseN CompositeNavigation) ActiveIndex() int {
	if parseN.active.get == nil {
		return -1
	}
	return compositeNormalizeIndex(parseN.items, parseN.active.Get(), parseN.options.InitialIndex)
}

// ActiveID is a core package helper.
func (parseN CompositeNavigation) ActiveID() string {
	parseIndex := parseN.ActiveIndex()
	if parseIndex < 0 || parseIndex >= len(parseN.items) {
		return ""
	}
	return parseN.items[parseIndex].ID
}

// ActiveDescendant is a core package helper.
func (parseN CompositeNavigation) ActiveDescendant() string {
	return parseN.ActiveID()
}

// IsActive is a core package helper.
func (parseN CompositeNavigation) IsActive(parseIndex int) bool {
	return parseIndex >= 0 && parseIndex == parseN.ActiveIndex()
}

// TabIndex is a core package helper.
func (parseN CompositeNavigation) TabIndex(parseIndex int) int {
	if parseIndex < 0 || parseIndex >= len(parseN.items) || parseN.items[parseIndex].Disabled {
		return -1
	}
	if parseN.IsActive(parseIndex) {
		return 0
	}
	return -1
}

// SetActive is a core package helper.
func (parseN CompositeNavigation) SetActive(parseIndex int) {
	if parseN.active.get == nil {
		return
	}
	if parseIndex < 0 || parseIndex >= len(parseN.items) || parseN.items[parseIndex].Disabled {
		return
	}
	parseN.active.Set(parseIndex)
}

// MoveNext is a core package helper.
func (parseN CompositeNavigation) MoveNext() {
	parseN.move(1)
}

// MovePrevious is a core package helper.
func (parseN CompositeNavigation) MovePrevious() {
	parseN.move(-1)
}

// MoveHome is a core package helper.
func (parseN CompositeNavigation) MoveHome() {
	parseIndex := compositeFirstEnabled(parseN.items)
	if parseIndex >= 0 {
		parseN.SetActive(parseIndex)
	}
}

// MoveEnd is a core package helper.
func (parseN CompositeNavigation) MoveEnd() {
	parseIndex := compositeLastEnabled(parseN.items)
	if parseIndex >= 0 {
		parseN.SetActive(parseIndex)
	}
}

// OnKeyDown is a core package helper.
func (parseN CompositeNavigation) OnKeyDown(parseEvent KeyboardEvent) {
	parseKey := parseEvent.GetKey()
	switch parseKey {
	case "ArrowRight":
		if parseN.options.Orientation == "horizontal" || parseN.options.Orientation == "both" {
			parseEvent.PreventDefault()
			parseN.MoveNext()
		}
	case "ArrowDown":
		if parseN.options.Orientation == "vertical" || parseN.options.Orientation == "both" {
			parseEvent.PreventDefault()
			parseN.MoveNext()
		}
	case "ArrowLeft":
		if parseN.options.Orientation == "horizontal" || parseN.options.Orientation == "both" {
			parseEvent.PreventDefault()
			parseN.MovePrevious()
		}
	case "ArrowUp":
		if parseN.options.Orientation == "vertical" || parseN.options.Orientation == "both" {
			parseEvent.PreventDefault()
			parseN.MovePrevious()
		}
	case "Home":
		parseEvent.PreventDefault()
		parseN.MoveHome()
	case "End":
		parseEvent.PreventDefault()
		parseN.MoveEnd()
	default:
		if compositeIsTypeaheadKey(parseKey) {
			parseEvent.PreventDefault()
			parseN.applyTypeahead(parseKey)
		}
	}
}

// move is a core package helper.
func (parseN CompositeNavigation) move(parseStep int) {
	if parseN.active.get == nil || len(parseN.items) == 0 {
		return
	}
	parseCurrent := parseN.ActiveIndex()
	if parseCurrent < 0 {
		parseCurrent = compositeInitialIndex(parseN.items, parseN.options.InitialIndex)
	}
	parseNext := compositeNextEnabled(parseN.items, parseCurrent, parseStep, parseN.options.Loop)
	if parseNext >= 0 {
		parseN.active.Set(parseNext)
	}
}

// applyTypeahead is a core package helper.
func (parseN CompositeNavigation) applyTypeahead(parseFragment string) {
	if len(parseN.items) == 0 {
		return
	}
	parseState := parseN.typeahead.Get()
	parseNow := time.Now()
	parseQuery := strings.ToLower(parseFragment)
	if parseNow.Sub(parseState.last) < 700*time.Millisecond {
		parseQuery = parseState.query + parseQuery
	}
	parseState.query = parseQuery
	parseState.last = parseNow
	parseN.typeahead.Set(parseState)

	parseStart := parseN.ActiveIndex()
	if parseStart < 0 {
		parseStart = 0
	}
	for parseOffset := 1; parseOffset <= len(parseN.items); parseOffset++ {
		parseCandidate := (parseStart + parseOffset) % len(parseN.items)
		parseItem := parseN.items[parseCandidate]
		if parseItem.Disabled {
			continue
		}
		if strings.HasPrefix(strings.ToLower(parseItem.Text), parseQuery) {
			parseN.active.Set(parseCandidate)
			return
		}
	}
	for parseIndex, parseItem2 := range parseN.items {
		if parseItem2.Disabled {
			continue
		}
		if strings.HasPrefix(strings.ToLower(parseItem2.Text), strings.ToLower(parseFragment)) {
			parseN.active.Set(parseIndex)
			parseState.query = strings.ToLower(parseFragment)
			parseN.typeahead.Set(parseState)
			return
		}
	}
}

// UseAnnouncer creates an Announcer backed by component state for polite and assertive live-region announcements.
func UseAnnouncer() Announcer {
	parsePoliteID := UseId() + "-polite"
	parseAssertiveID := UseId() + "-assertive"
	return Announcer{
		polite:      UseState(announcementState{}),
		assertive:   UseState(announcementState{}),
		politeID:    parsePoliteID,
		assertiveID: parseAssertiveID,
	}
}

// Announce is a core package helper.
func (parseA Announcer) Announce(parseMode AnnouncementMode, parseMessage string) {
	switch parseMode {
	case AnnouncementAssertive:
		parseA.Assertive(parseMessage)
	default:
		parseA.Polite(parseMessage)
	}
}

// Polite is a core package helper.
func (parseA Announcer) Polite(parseMessage string) {
	if parseA.polite.get == nil {
		return
	}
	parseA.polite.Update(func(parsePrevious announcementState) announcementState {
		parsePrevious.message = parseMessage
		parsePrevious.sequence++
		return parsePrevious
	})
}

// Assertive is a core package helper.
func (parseA Announcer) Assertive(parseMessage string) {
	if parseA.assertive.get == nil {
		return
	}
	parseA.assertive.Update(func(parsePrevious announcementState) announcementState {
		parsePrevious.message = parseMessage
		parsePrevious.sequence++
		return parsePrevious
	})
}

// Clear is a core package helper.
func (parseA Announcer) Clear() {
	if parseA.polite.get != nil {
		parseA.polite.Set(announcementState{})
	}
	if parseA.assertive.get != nil {
		parseA.assertive.Set(announcementState{})
	}
}

// PoliteID is a core package helper.
func (parseA Announcer) PoliteID() string {
	return parseA.politeID
}

// AssertiveID is a core package helper.
func (parseA Announcer) AssertiveID() string {
	return parseA.assertiveID
}

// Region is a core package helper.
func (parseA Announcer) Region() Node {
	parsePoliteState := announcementState{}
	parseAssertiveState := announcementState{}
	if parseA.polite.get != nil {
		parsePoliteState = parseA.polite.Get()
	}
	if parseA.assertive.get != nil {
		parseAssertiveState = parseA.assertive.Get()
	}

	return runtime.CreateElement("div", nil,
		announcementRegionNode(parseA.politeID, string(AnnouncementPolite), parsePoliteState),
		announcementRegionNode(parseA.assertiveID, string(AnnouncementAssertive), parseAssertiveState),
	)
}

// announcementRegionNode is a core package helper.
func announcementRegionNode(parseId string, parseMode string, parseState announcementState) Node {
	return runtime.CreateElementOwned("div", map[string]interface{}{
		"id":          parseId,
		"role":        "status",
		"aria-live":   parseMode,
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
		runtime.CreateElement("span", nil, parseState.message),
		runtime.CreateElementOwned("span", map[string]interface{}{"aria-hidden": "true"}, fmt.Sprintf("-%d", parseState.sequence)),
	)
}

// AccessibleOverlay renders an overlay with ARIA roles, focus management, and optional modal behaviour.
func AccessibleOverlay(parseProps AccessibleOverlayProps) Node {
	parseModal := parseProps.Modal
	if parseProps.Role == "dialog" && !parseProps.Modal {
		parseModal = true
	}
	return Overlay(OverlayProps{
		Open:                  parseProps.Open,
		Target:                parseProps.Target,
		AppRootSelector:       parseProps.AppRootSelector,
		SurfaceID:             parseProps.SurfaceID,
		Kind:                  parseProps.Kind,
		Role:                  parseProps.Role,
		LabelledBy:            parseProps.LabelledBy,
		DescribedBy:           parseProps.DescribedBy,
		InitialFocusSelector:  parseProps.InitialFocusSelector,
		FallbackFocusSelector: parseProps.FallbackFocusSelector,
		Modal:                 parseModal,
		Backdrop:              parseProps.Backdrop,
		TrapFocus:             parseProps.TrapFocus,
		RestoreFocus:          parseProps.RestoreFocus,
		CloseOnEscape:         parseProps.CloseOnEscape,
		CloseOnOutsideClick:   parseProps.CloseOnOutsideClick,
		LockScroll:            parseProps.LockScroll,
		BackgroundInert:       parseProps.BackgroundInert,
		BaseZIndex:            parseProps.BaseZIndex,
		AnchorSelector:        parseProps.AnchorSelector,
		Positioning:           parseProps.Positioning,
		BackdropClass:         parseProps.BackdropClass,
		SurfaceClass:          parseProps.SurfaceClass,
		BackdropStyle:         parseProps.BackdropStyle,
		SurfaceStyle:          parseProps.SurfaceStyle,
		Child:                 parseProps.Child,
		Children:              parseProps.Children,
		OnDismiss:             parseProps.OnDismiss,
	})
}

// compositeNormalizeIndex is a core package helper.
func compositeNormalizeIndex(parseItems []CompositeItem, parseCurrent int, parsePreferred int) int {
	if parseCurrent >= 0 && parseCurrent < len(parseItems) && !parseItems[parseCurrent].Disabled {
		return parseCurrent
	}
	return compositeInitialIndex(parseItems, parsePreferred)
}

// compositeInitialIndex is a core package helper.
func compositeInitialIndex(parseItems []CompositeItem, parsePreferred int) int {
	if parsePreferred >= 0 && parsePreferred < len(parseItems) && !parseItems[parsePreferred].Disabled {
		return parsePreferred
	}
	return compositeFirstEnabled(parseItems)
}

// compositeFirstEnabled is a core package helper.
func compositeFirstEnabled(parseItems []CompositeItem) int {
	for parseIndex, parseItem := range parseItems {
		if !parseItem.Disabled {
			return parseIndex
		}
	}
	return -1
}

// compositeLastEnabled is a core package helper.
func compositeLastEnabled(parseItems []CompositeItem) int {
	for parseIndex := len(parseItems) - 1; parseIndex >= 0; parseIndex-- {
		if !parseItems[parseIndex].Disabled {
			return parseIndex
		}
	}
	return -1
}

// compositeNextEnabled is a core package helper.
func compositeNextEnabled(parseItems []CompositeItem, parseStart int, parseStep int, isLoop bool) int {
	if len(parseItems) == 0 {
		return -1
	}
	parseIndex := parseStart
	for parseAttempts := 0; parseAttempts < len(parseItems); parseAttempts++ {
		parseIndex += parseStep
		if parseIndex < 0 {
			if !isLoop {
				return compositeFirstEnabled(parseItems)
			}
			parseIndex = len(parseItems) - 1
		}
		if parseIndex >= len(parseItems) {
			if !isLoop {
				return compositeLastEnabled(parseItems)
			}
			parseIndex = 0
		}
		if !parseItems[parseIndex].Disabled {
			return parseIndex
		}
	}
	return -1
}

// compositeIsTypeaheadKey is a core package helper.
func compositeIsTypeaheadKey(parseKey string) bool {
	if len([]rune(parseKey)) != 1 {
		return false
	}
	parseR := []rune(parseKey)[0]
	if parseR == ' ' {
		return false
	}
	return (parseR >= '0' && parseR <= '9') || (parseR >= 'A' && parseR <= 'Z') || (parseR >= 'a' && parseR <= 'z')
}
