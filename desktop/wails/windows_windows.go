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

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const maximumOwnedWindows = 16

type nativeWindowEntry struct {
	templateID string
	title      string
	window     application.Window
	closing    atomic.Bool
	closed     atomic.Bool
}

type nativeWindowState struct {
	mu        sync.Mutex
	templates map[string]WindowTemplate
	owners    map[uint]map[string]*nativeWindowEntry
	hooked    map[uint]bool
	closing   map[uint]bool
}

// NewNativeBackendWithWindowTemplates creates an adapter with immutable approved child routes.
func NewNativeBackendWithWindowTemplates(templates map[string]WindowTemplate) (desktop.NativeBackend, error) {
	state, err := newNativeWindowState(templates)
	if err != nil {
		return nil, err
	}
	return &nativeBackend{trayShortcuts: newNativeTrayShortcutState(), windows: state}, nil
}

// newNativeWindowState validates and copies host configuration.
func newNativeWindowState(templates map[string]WindowTemplate) (*nativeWindowState, error) {
	if len(templates) == 0 || len(templates) > 64 {
		return nil, invalid("child-window templates must contain 1 to 64 entries")
	}
	copied := make(map[string]WindowTemplate, len(templates))
	for id, tmpl := range templates {
		if id == "" || len(id) > 128 || !utf8.ValidString(id) || strings.ContainsRune(id, 0) {
			return nil, invalid("invalid child-window template id")
		}
		if err := validateWindowTemplate(tmpl); err != nil {
			return nil, err
		}
		copied[id] = tmpl
	}
	return &nativeWindowState{
		templates: copied,
		owners:    map[uint]map[string]*nativeWindowEntry{},
		hooked:    map[uint]bool{},
		closing:   map[uint]bool{},
	}, nil
}

// validateWindowTemplate restricts navigation to a relative asset-server route and bounds geometry.
func validateWindowTemplate(tmpl WindowTemplate) error {
	parsedURL, err := url.Parse(tmpl.Route)
	if err != nil || !strings.HasPrefix(tmpl.Route, "/") || strings.HasPrefix(tmpl.Route, "//") || strings.Contains(tmpl.Route, "\\") || parsedURL.IsAbs() || parsedURL.Host != "" || parsedURL.User != nil {
		return invalid("child-window template route must be same-origin and root-relative")
	}
	if !utf8.ValidString(tmpl.Title) || strings.ContainsRune(tmpl.Title, 0) || len(tmpl.Title) > 256 {
		return invalid("invalid child-window template title")
	}
	if tmpl.Width <= 0 || tmpl.Height <= 0 || tmpl.Width > 4096 || tmpl.Height > 4096 {
		return invalid("child-window template dimensions must be between 1 and 4096")
	}
	for _, bound := range []int{tmpl.MinWidth, tmpl.MinHeight, tmpl.MaxWidth, tmpl.MaxHeight} {
		if bound < 0 || bound > 4096 {
			return invalid("child-window template size constraints must be between 0 and 4096")
		}
	}
	if tmpl.MaxWidth > 0 && tmpl.MinWidth > tmpl.MaxWidth {
		return invalid("child-window template min width exceeds max width")
	}
	if tmpl.MaxHeight > 0 && tmpl.MinHeight > tmpl.MaxHeight {
		return invalid("child-window template min height exceeds max height")
	}
	if _, ok := resolveWindowBackdrop(tmpl.Backdrop); !ok {
		return invalid("unknown child-window template backdrop")
	}
	return nil
}

// resolveWindowBackdrop converts a host-configured backdrop into Wails Windows options.
func resolveWindowBackdrop(backdrop WindowBackdrop) (application.BackdropType, bool) {
	switch backdrop {
	case WindowBackdropDefault, WindowBackdropAuto:
		return application.Auto, true
	case WindowBackdropNone:
		return application.None, true
	case WindowBackdropMica:
		return application.Mica, true
	case WindowBackdropAcrylic:
		return application.Acrylic, true
	case WindowBackdropTabbed:
		return application.Tabbed, true
	default:
		return application.Auto, false
	}
}

// CreateChildWindow creates a child using only a host-registered route template.
func (b nativeBackend) CreateChildWindow(ctx context.Context, req desktop.ChildWindowCreateRequest) (desktop.ChildWindowInfo, error) {
	owner, err := getCaller(ctx)
	if err != nil {
		return desktop.ChildWindowInfo{}, err
	}
	if b.windows == nil {
		return desktop.ChildWindowInfo{}, unavailable("child windows are not configured")
	}
	state := b.windows
	state.mu.Lock()
	ownerID := owner.ID()
	if !state.hooked[ownerID] {
		// Wails dispatches listeners only after every close hook accepts the event.
		// Keeping the reservation and accepted-close phases separate means a later
		// host hook may veto without poisoning this owner's lifecycle state.
		owner.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
			closeOwnedWindows(state, ownerID)
		})
		state.hooked[ownerID] = true
	}
	if state.closing[ownerID] {
		state.mu.Unlock()
		return desktop.ChildWindowInfo{}, unavailable("calling window is closing")
	}
	tmpl, ok := state.templates[req.TemplateID]
	if !ok {
		state.mu.Unlock()
		return desktop.ChildWindowInfo{}, invalid("unknown child-window template id")
	}
	owned := state.owners[ownerID]
	if owned == nil {
		owned = map[string]*nativeWindowEntry{}
		state.owners[ownerID] = owned
	}
	if _, exists := owned[req.ID]; exists {
		state.mu.Unlock()
		return desktop.ChildWindowInfo{}, invalid("child-window id already exists for caller")
	}
	if len(owned) >= maximumOwnedWindows {
		state.mu.Unlock()
		return desktop.ChildWindowInfo{}, invalid("caller child-window limit reached")
	}
	owned[req.ID] = nil
	state.mu.Unlock()

	title := tmpl.Title
	if req.Title != "" {
		title = req.Title
	}
	width, height := tmpl.Width, tmpl.Height
	if req.Width != 0 {
		width, height = req.Width, req.Height
	}
	app := application.Get()
	if app == nil {
		removeWindowReservation(state, ownerID, req.ID)
		return desktop.ChildWindowInfo{}, unavailable("Wails application unavailable")
	}
	hash := sha256.Sum256([]byte(req.ID))
	opts := application.WebviewWindowOptions{
		Name:        fmt.Sprintf("gwc-child-%d-%x", ownerID, hash[:8]),
		Title:       title,
		Width:       width,
		Height:      height,
		MinWidth:    tmpl.MinWidth,
		MinHeight:   tmpl.MinHeight,
		MaxWidth:    tmpl.MaxWidth,
		MaxHeight:   tmpl.MaxHeight,
		Frameless:   tmpl.Frameless,
		AlwaysOnTop: tmpl.AlwaysOnTop,
		URL:         tmpl.Route,
	}
	if tmpl.Backdrop != WindowBackdropDefault {
		backdropType, _ := resolveWindowBackdrop(tmpl.Backdrop)
		opts.BackgroundType = application.BackgroundTypeTranslucent
		opts.Windows.BackdropType = backdropType
	}
	win := application.NewWindow(opts)
	entry := &nativeWindowEntry{templateID: req.TemplateID, title: title, window: win}
	// Register before the window is added or run so neither host callbacks nor
	// user input can close it before accepted-close cleanup exists.
	win.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		entry.closed.Store(true)
		removeOwnedWindow(state, ownerID, req.ID, win)
	})
	app.Window.Add(win)
	win.Run()

	state.mu.Lock()
	owned = state.owners[ownerID]
	if entry.closed.Load() {
		if owned != nil {
			delete(owned, req.ID)
			if len(owned) == 0 {
				delete(state.owners, ownerID)
			}
		}
		state.mu.Unlock()
		return desktop.ChildWindowInfo{}, unavailable("child window closed during creation")
	}
	if owned == nil {
		owned = map[string]*nativeWindowEntry{}
		state.owners[ownerID] = owned
	}
	owned[req.ID] = entry
	isOwnerClosing := state.closing[ownerID]
	state.mu.Unlock()
	if isOwnerClosing {
		// Retain ownership until the child's accepted-close listener runs. A
		// child veto therefore cannot turn this failed creation into an orphan.
		entry.closing.Store(true)
		win.Close()
		return desktop.ChildWindowInfo{}, unavailable("calling window closed during child creation")
	}
	return buildNativeChildWindowInfo(state, req.ID, entry), nil
}

// ListChildWindows lists a stable snapshot of caller-owned children.
func (b nativeBackend) ListChildWindows(ctx context.Context) ([]desktop.ChildWindowInfo, error) {
	owner, err := getCaller(ctx)
	if err != nil {
		return nil, err
	}
	if b.windows == nil {
		return nil, unavailable("child windows are not configured")
	}
	b.windows.mu.Lock()
	entries := make(map[string]*nativeWindowEntry, len(b.windows.owners[owner.ID()]))
	for id, entry := range b.windows.owners[owner.ID()] {
		if entry != nil {
			entries[id] = entry
		}
	}
	b.windows.mu.Unlock()
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]desktop.ChildWindowInfo, 0, len(ids))
	for _, id := range ids {
		result = append(result, buildNativeChildWindowInfo(b.windows, id, entries[id]))
	}
	return result, nil
}

// InspectChildWindow resolves an ID only inside the calling owner's scope.
func (b nativeBackend) InspectChildWindow(ctx context.Context, req desktop.ChildWindowRequest) (desktop.ChildWindowInfo, error) {
	owner, err := getCaller(ctx)
	if err != nil {
		return desktop.ChildWindowInfo{}, err
	}
	entry, err := getOwnedWindow(b.windows, owner.ID(), req.ID)
	if err != nil {
		return desktop.ChildWindowInfo{}, err
	}
	return buildNativeChildWindowInfo(b.windows, req.ID, entry), nil
}

// ControlChildWindow applies a bounded control to an owned child without affecting its parent.
func (b nativeBackend) ControlChildWindow(ctx context.Context, req desktop.ChildWindowControlRequest) (desktop.ChildWindowInfo, error) {
	owner, err := getCaller(ctx)
	if err != nil {
		return desktop.ChildWindowInfo{}, err
	}
	entry, err := getOwnedWindow(b.windows, owner.ID(), req.ID)
	if err != nil {
		return desktop.ChildWindowInfo{}, err
	}
	switch req.Action {
	case "show":
		entry.window.Show()
		info := buildNativeChildWindowInfo(b.windows, req.ID, entry)
		info.Visible = true
		return info, nil
	case "hide":
		entry.window.Hide()
		info := buildNativeChildWindowInfo(b.windows, req.ID, entry)
		info.Visible = false
		return info, nil
	case "close":
		info := buildNativeChildWindowInfo(b.windows, req.ID, entry)
		setOwnedWindowClosing(b.windows, owner.ID(), req.ID, entry.window)
		entry.window.Close()
		info.Closing = true
		return info, nil
	case "focus":
		entry.window.Focus()
	case "minimize":
		entry.window.Minimise()
	case "unminimize":
		entry.window.UnMinimise()
	case "maximize":
		entry.window.Maximise()
	case "unmaximize":
		entry.window.UnMaximise()
	case "restore":
		entry.window.Restore()
	case "center":
		entry.window.Center()
	case "resize":
		if req.Width <= 0 || req.Height <= 0 {
			return desktop.ChildWindowInfo{}, invalid("child-window dimensions must be positive")
		}
		entry.window.SetSize(req.Width, req.Height)
	case "set-title":
		entry.window.SetTitle(req.Title)
		b.windows.mu.Lock()
		entry.title = req.Title
		b.windows.mu.Unlock()
	case "set-bounds":
		if req.Width <= 0 || req.Height <= 0 {
			return desktop.ChildWindowInfo{}, invalid("child-window dimensions must be positive")
		}
		entry.window.SetBounds(application.Rect{X: req.X, Y: req.Y, Width: req.Width, Height: req.Height})
	default:
		return desktop.ChildWindowInfo{}, invalid("unknown child-window action")
	}
	return buildNativeChildWindowInfo(b.windows, req.ID, entry), nil
}

// setOwnedWindowClosing marks an accepted close request as pending until WindowClosing runs.
func setOwnedWindowClosing(state *nativeWindowState, ownerID uint, id string, expected application.Window) {
	state.mu.Lock()
	defer state.mu.Unlock()
	entry := state.owners[ownerID][id]
	if entry != nil && entry.window == expected {
		entry.closing.Store(true)
	}
}

// getOwnedWindow resolves only an owner's own child map.
func getOwnedWindow(state *nativeWindowState, ownerID uint, id string) (*nativeWindowEntry, error) {
	if state == nil {
		return nil, unavailable("child windows are not configured")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	entry := state.owners[ownerID][id]
	if entry == nil {
		return nil, &interop.Error{Op: "desktop", Code: interop.CodeUnavailable, Err: fmt.Errorf("caller does not own child window %q", id)}
	}
	return entry, nil
}

// removeWindowReservation rolls back a failed creation.
func removeWindowReservation(state *nativeWindowState, ownerID uint, id string) {
	removeOwnedWindow(state, ownerID, id, nil)
}

// removeOwnedWindow removes a child only when the expected window still owns the slot.
func removeOwnedWindow(state *nativeWindowState, ownerID uint, id string, expected application.Window) {
	state.mu.Lock()
	defer state.mu.Unlock()
	owned := state.owners[ownerID]
	entry, ok := owned[id]
	if !ok || (expected != nil && (entry == nil || entry.window != expected)) {
		return
	}
	delete(owned, id)
	if len(owned) == 0 && state.closing[ownerID] {
		delete(state.owners, ownerID)
	}
}

// closeOwnedWindows blocks new creates and requests closure without dropping pending ownership.
func closeOwnedWindows(state *nativeWindowState, ownerID uint) {
	state.mu.Lock()
	owned := state.owners[ownerID]
	state.closing[ownerID] = true
	entries := make([]*nativeWindowEntry, 0, len(owned))
	for _, entry := range owned {
		if entry != nil {
			entry.closing.Store(true)
			entries = append(entries, entry)
		}
	}
	state.mu.Unlock()
	for _, entry := range entries {
		entry.window.Close()
	}
}

// buildNativeChildWindowInfo converts an owned Wails child to portable metadata.
func buildNativeChildWindowInfo(state *nativeWindowState, id string, entry *nativeWindowEntry) desktop.ChildWindowInfo {
	state.mu.Lock()
	title := entry.title
	state.mu.Unlock()
	x, y := entry.window.Position()
	return desktop.ChildWindowInfo{
		ID:         id,
		TemplateID: entry.templateID,
		Title:      title,
		X:          x,
		Y:          y,
		Width:      entry.window.Width(),
		Height:     entry.window.Height(),
		Visible:    entry.window.IsVisible(),
		Focused:    entry.window.IsFocused(),
		Minimised:  entry.window.IsMinimised(),
		Maximised:  entry.window.IsMaximised(),
		Closing:    entry.closing.Load(),
	}
}
