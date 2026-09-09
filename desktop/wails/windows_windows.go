//go:build windows

package wails

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const maximumOwnedWindows = 16

type nativeWindowEntry struct {
	parseTemplateID string
	parseTitle      string
	parseWindow     application.Window
	parseClosing    atomic.Bool
	parseClosed     atomic.Bool
}

type nativeWindowState struct {
	parseMutex     sync.Mutex
	parseTemplates map[string]WindowTemplate
	parseOwners    map[uint]map[string]*nativeWindowEntry
	parseHooked    map[uint]bool
	parseClosing   map[uint]bool
}

// NewNativeBackendWithWindowTemplates creates an adapter with immutable approved child routes.
func NewNativeBackendWithWindowTemplates(parseTemplates map[string]WindowTemplate) (desktop.NativeBackend, error) {
	parseState, parseErr := newNativeWindowState(parseTemplates)
	if parseErr != nil {
		return nil, parseErr
	}
	return &nativeBackend{parseTrayShortcuts: newNativeTrayShortcutState(), parseWindows: parseState}, nil
}

// newNativeWindowState validates and copies host configuration.
func newNativeWindowState(parseTemplates map[string]WindowTemplate) (*nativeWindowState, error) {
	if len(parseTemplates) == 0 || len(parseTemplates) > 64 {
		return nil, invalid("child-window templates must contain 1 to 64 entries")
	}
	parseCopy := make(map[string]WindowTemplate, len(parseTemplates))
	for parseID, parseTemplate := range parseTemplates {
		if parseID == "" || len(parseID) > 128 || !utf8.ValidString(parseID) || strings.ContainsRune(parseID, 0) {
			return nil, invalid("invalid child-window template id")
		}
		if parseErr := validateWindowTemplate(parseTemplate); parseErr != nil {
			return nil, parseErr
		}
		parseCopy[parseID] = parseTemplate
	}
	return &nativeWindowState{parseTemplates: parseCopy, parseOwners: map[uint]map[string]*nativeWindowEntry{}, parseHooked: map[uint]bool{}, parseClosing: map[uint]bool{}}, nil
}

// validateWindowTemplate restricts navigation to a relative asset-server route.
func validateWindowTemplate(parseTemplate WindowTemplate) error {
	parseURL, parseErr := url.Parse(parseTemplate.Route)
	if parseErr != nil || !strings.HasPrefix(parseTemplate.Route, "/") || strings.HasPrefix(parseTemplate.Route, "//") || strings.Contains(parseTemplate.Route, "\\") || parseURL.IsAbs() || parseURL.Host != "" || parseURL.User != nil {
		return invalid("child-window template route must be same-origin and root-relative")
	}
	if !utf8.ValidString(parseTemplate.Title) || strings.ContainsRune(parseTemplate.Title, 0) || len(parseTemplate.Title) > 256 {
		return invalid("invalid child-window template title")
	}
	if parseTemplate.Width <= 0 || parseTemplate.Height <= 0 || parseTemplate.Width > 4096 || parseTemplate.Height > 4096 {
		return invalid("child-window template dimensions must be between 1 and 4096")
	}
	return nil
}

// CreateChildWindow creates a child using only a host-registered route template.
func (parseBackend nativeBackend) CreateChildWindow(parseContext context.Context, parseRequest desktop.ChildWindowCreateRequest) (desktop.ChildWindowInfo, error) {
	parseOwner, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.ChildWindowInfo{}, parseErr
	}
	if parseBackend.parseWindows == nil {
		return desktop.ChildWindowInfo{}, unavailable("child windows are not configured")
	}
	parseState := parseBackend.parseWindows
	parseState.parseMutex.Lock()
	parseOwnerID := parseOwner.ID()
	if !parseState.parseHooked[parseOwnerID] {
		// Wails dispatches listeners only after every close hook accepts the event.
		// Keeping the reservation and accepted-close phases separate means a later
		// host hook may veto without poisoning this owner's lifecycle state.
		parseOwner.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
			closeOwnedWindows(parseState, parseOwnerID)
		})
		parseState.parseHooked[parseOwnerID] = true
	}
	if parseState.parseClosing[parseOwnerID] {
		parseState.parseMutex.Unlock()
		return desktop.ChildWindowInfo{}, unavailable("calling window is closing")
	}
	parseTemplate, parseOK := parseState.parseTemplates[parseRequest.TemplateID]
	if !parseOK {
		parseState.parseMutex.Unlock()
		return desktop.ChildWindowInfo{}, invalid("unknown child-window template id")
	}
	parseOwned := parseState.parseOwners[parseOwnerID]
	if parseOwned == nil {
		parseOwned = map[string]*nativeWindowEntry{}
		parseState.parseOwners[parseOwnerID] = parseOwned
	}
	if _, parseExists := parseOwned[parseRequest.ID]; parseExists {
		parseState.parseMutex.Unlock()
		return desktop.ChildWindowInfo{}, invalid("child-window id already exists for caller")
	}
	if len(parseOwned) >= maximumOwnedWindows {
		parseState.parseMutex.Unlock()
		return desktop.ChildWindowInfo{}, invalid("caller child-window limit reached")
	}
	parseOwned[parseRequest.ID] = nil
	parseState.parseMutex.Unlock()
	parseTitle := parseTemplate.Title
	if parseRequest.Title != "" {
		parseTitle = parseRequest.Title
	}
	parseWidth, parseHeight := parseTemplate.Width, parseTemplate.Height
	if parseRequest.Width != 0 {
		parseWidth, parseHeight = parseRequest.Width, parseRequest.Height
	}
	parseApp := application.Get()
	if parseApp == nil {
		removeWindowReservation(parseState, parseOwnerID, parseRequest.ID)
		return desktop.ChildWindowInfo{}, unavailable("Wails application unavailable")
	}
	parseHash := sha256.Sum256([]byte(parseRequest.ID))
	parseWindow := application.NewWindow(application.WebviewWindowOptions{Name: fmt.Sprintf("gwc-child-%d-%x", parseOwnerID, parseHash[:8]), Title: parseTitle, Width: parseWidth, Height: parseHeight, URL: parseTemplate.Route})
	parseEntry := &nativeWindowEntry{parseTemplateID: parseRequest.TemplateID, parseTitle: parseTitle, parseWindow: parseWindow}
	// Register before the window is added or run so neither host callbacks nor
	// user input can close it before accepted-close cleanup exists.
	parseWindow.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		parseEntry.parseClosed.Store(true)
		removeOwnedWindow(parseState, parseOwnerID, parseRequest.ID, parseWindow)
	})
	parseApp.Window.Add(parseWindow)
	parseWindow.Run()
	parseState.parseMutex.Lock()
	parseOwned = parseState.parseOwners[parseOwnerID]
	if parseEntry.parseClosed.Load() {
		if parseOwned != nil {
			delete(parseOwned, parseRequest.ID)
			if len(parseOwned) == 0 {
				delete(parseState.parseOwners, parseOwnerID)
			}
		}
		parseState.parseMutex.Unlock()
		return desktop.ChildWindowInfo{}, unavailable("child window closed during creation")
	}
	if parseOwned == nil {
		parseOwned = map[string]*nativeWindowEntry{}
		parseState.parseOwners[parseOwnerID] = parseOwned
	}
	parseOwned[parseRequest.ID] = parseEntry
	isOwnerClosing := parseState.parseClosing[parseOwnerID]
	parseState.parseMutex.Unlock()
	if isOwnerClosing {
		// Retain ownership until the child's accepted-close listener runs. A
		// child veto therefore cannot turn this failed creation into an orphan.
		parseEntry.parseClosing.Store(true)
		parseWindow.Close()
		return desktop.ChildWindowInfo{}, unavailable("calling window closed during child creation")
	}
	return getNativeChildWindowInfo(parseRequest.ID, parseEntry), nil
}

// ListChildWindows lists a stable snapshot of caller-owned children.
func (parseBackend nativeBackend) ListChildWindows(parseContext context.Context) ([]desktop.ChildWindowInfo, error) {
	parseOwner, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseBackend.parseWindows == nil {
		return nil, unavailable("child windows are not configured")
	}
	parseBackend.parseWindows.parseMutex.Lock()
	parseEntries := make(map[string]*nativeWindowEntry, len(parseBackend.parseWindows.parseOwners[parseOwner.ID()]))
	for parseID, parseEntry := range parseBackend.parseWindows.parseOwners[parseOwner.ID()] {
		if parseEntry != nil {
			parseEntries[parseID] = parseEntry
		}
	}
	parseBackend.parseWindows.parseMutex.Unlock()
	parseIDs := make([]string, 0, len(parseEntries))
	for parseID := range parseEntries {
		parseIDs = append(parseIDs, parseID)
	}
	sort.Strings(parseIDs)
	parseResult := make([]desktop.ChildWindowInfo, 0, len(parseIDs))
	for _, parseID := range parseIDs {
		parseResult = append(parseResult, getNativeChildWindowInfo(parseID, parseEntries[parseID]))
	}
	return parseResult, nil
}

// InspectChildWindow resolves an ID only inside the calling owner's scope.
func (parseBackend nativeBackend) InspectChildWindow(parseContext context.Context, parseRequest desktop.ChildWindowRequest) (desktop.ChildWindowInfo, error) {
	parseOwner, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.ChildWindowInfo{}, parseErr
	}
	parseEntry, parseErr := getOwnedWindow(parseBackend.parseWindows, parseOwner.ID(), parseRequest.ID)
	if parseErr != nil {
		return desktop.ChildWindowInfo{}, parseErr
	}
	return getNativeChildWindowInfo(parseRequest.ID, parseEntry), nil
}

// ControlChildWindow shows, hides, or closes an owned child without affecting its parent.
func (parseBackend nativeBackend) ControlChildWindow(parseContext context.Context, parseRequest desktop.ChildWindowControlRequest) (desktop.ChildWindowInfo, error) {
	parseOwner, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.ChildWindowInfo{}, parseErr
	}
	parseEntry, parseErr := getOwnedWindow(parseBackend.parseWindows, parseOwner.ID(), parseRequest.ID)
	if parseErr != nil {
		return desktop.ChildWindowInfo{}, parseErr
	}
	parseInfo := getNativeChildWindowInfo(parseRequest.ID, parseEntry)
	switch parseRequest.Action {
	case "show":
		parseEntry.parseWindow.Show()
		parseInfo.Visible = true
	case "hide":
		parseEntry.parseWindow.Hide()
		parseInfo.Visible = false
	case "close":
		setOwnedWindowClosing(parseBackend.parseWindows, parseOwner.ID(), parseRequest.ID, parseEntry.parseWindow)
		parseEntry.parseWindow.Close()
		parseInfo.Closing = true
	default:
		return desktop.ChildWindowInfo{}, invalid("unknown child-window action")
	}
	return parseInfo, nil
}

// setOwnedWindowClosing marks an accepted close request as pending until WindowClosing runs.
func setOwnedWindowClosing(parseState *nativeWindowState, parseOwnerID uint, parseID string, parseExpected application.Window) {
	parseState.parseMutex.Lock()
	defer parseState.parseMutex.Unlock()
	parseEntry := parseState.parseOwners[parseOwnerID][parseID]
	if parseEntry != nil && parseEntry.parseWindow == parseExpected {
		parseEntry.parseClosing.Store(true)
	}
}

// getOwnedWindow resolves only an owner's own child map.
func getOwnedWindow(parseState *nativeWindowState, parseOwnerID uint, parseID string) (*nativeWindowEntry, error) {
	if parseState == nil {
		return nil, unavailable("child windows are not configured")
	}
	parseState.parseMutex.Lock()
	defer parseState.parseMutex.Unlock()
	parseEntry := parseState.parseOwners[parseOwnerID][parseID]
	if parseEntry == nil {
		return nil, &interop.Error{Op: "desktop", Code: interop.CodeUnavailable, Err: fmt.Errorf("caller does not own child window %q", parseID)}
	}
	return parseEntry, nil
}

// removeWindowReservation rolls back a failed creation.
func removeWindowReservation(parseState *nativeWindowState, parseOwnerID uint, parseID string) {
	removeOwnedWindow(parseState, parseOwnerID, parseID, nil)
}

// removeOwnedWindow removes a child only when the expected window still owns the slot.
func removeOwnedWindow(parseState *nativeWindowState, parseOwnerID uint, parseID string, parseExpected application.Window) {
	parseState.parseMutex.Lock()
	defer parseState.parseMutex.Unlock()
	parseOwned := parseState.parseOwners[parseOwnerID]
	parseEntry, parseOK := parseOwned[parseID]
	if !parseOK || (parseExpected != nil && (parseEntry == nil || parseEntry.parseWindow != parseExpected)) {
		return
	}
	delete(parseOwned, parseID)
	if len(parseOwned) == 0 && parseState.parseClosing[parseOwnerID] {
		delete(parseState.parseOwners, parseOwnerID)
	}
}

// closeOwnedWindows blocks new creates and requests closure without dropping pending ownership.
func closeOwnedWindows(parseState *nativeWindowState, parseOwnerID uint) {
	parseState.parseMutex.Lock()
	parseOwned := parseState.parseOwners[parseOwnerID]
	parseState.parseClosing[parseOwnerID] = true
	parseEntries := make([]*nativeWindowEntry, 0, len(parseOwned))
	for _, parseEntry := range parseOwned {
		if parseEntry != nil {
			parseEntry.parseClosing.Store(true)
			parseEntries = append(parseEntries, parseEntry)
		}
	}
	parseState.parseMutex.Unlock()
	for _, parseEntry := range parseEntries {
		parseEntry.parseWindow.Close()
	}
}

// getNativeChildWindowInfo converts an owned Wails child to portable metadata.
func getNativeChildWindowInfo(parseID string, parseEntry *nativeWindowEntry) desktop.ChildWindowInfo {
	return desktop.ChildWindowInfo{ID: parseID, TemplateID: parseEntry.parseTemplateID, Title: parseEntry.parseTitle, Width: parseEntry.parseWindow.Width(), Height: parseEntry.parseWindow.Height(), Visible: parseEntry.parseWindow.IsVisible(), Closing: parseEntry.parseClosing.Load()}
}
