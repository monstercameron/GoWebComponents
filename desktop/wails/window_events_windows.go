//go:build windows

package wails

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// WindowEventOptions configures explicit caller-window event registration.
type WindowEventOptions struct{ EnableFileDrop bool }

// AttachWindowEvents installs the portable event allowlist on exactly one Wails window.
func AttachWindowEvents(win application.Window, opts WindowEventOptions) (func(), error) {
	if win == nil {
		return nil, invalid("window event owner required")
	}
	cleanups := []func(){
		registerWindowEvent(win, events.Common.WindowFocus, desktop.WindowFocused),
		registerWindowEvent(win, events.Common.WindowLostFocus, desktop.WindowBlurred),
		registerWindowEvent(win, events.Common.WindowDidMove, desktop.WindowMoved),
		registerWindowEvent(win, events.Common.WindowDidResize, desktop.WindowResized),
		registerWindowEvent(win, events.Common.WindowShow, desktop.WindowShown),
		registerWindowEvent(win, events.Common.WindowHide, desktop.WindowHidden),
		registerWindowEvent(win, events.Common.WindowFullscreen, desktop.WindowFullscreen),
		registerWindowEvent(win, events.Common.WindowUnFullscreen, desktop.WindowUnfullscreened),
		registerWindowEvent(win, events.Common.WindowMinimise, desktop.WindowMinimized),
		registerWindowEvent(win, events.Common.WindowUnMinimise, desktop.WindowUnminimized),
		registerWindowEvent(win, events.Common.WindowMaximise, desktop.WindowMaximized),
		registerWindowEvent(win, events.Common.WindowUnMaximise, desktop.WindowUnmaximized),
		registerWindowEvent(win, events.Common.WindowRestore, desktop.WindowRestored),
		registerWindowEvent(win, events.Common.WindowDPIChanged, desktop.WindowDPIChanged),
		registerWindowEvent(win, events.Common.WindowZoom, desktop.WindowZoomChanged),
		registerWindowEvent(win, events.Common.WindowZoomIn, desktop.WindowZoomChanged),
		registerWindowEvent(win, events.Common.WindowZoomOut, desktop.WindowZoomChanged),
		registerWindowEvent(win, events.Common.WindowZoomReset, desktop.WindowZoomChanged),
		registerWindowEvent(win, events.Common.WindowClosing, desktop.WindowClosing),
	}
	if opts.EnableFileDrop {
		cleanups = append(cleanups, win.OnWindowEvent(events.Common.WindowFilesDropped, func(nativeEvent *application.WindowEvent) {
			files, isValid := getDroppedFiles(nativeEvent)
			if !isValid {
				return
			}
			event := buildWindowEvent(win, desktop.WindowFilesDropped)
			event.Files = files
			win.EmitEvent(desktop.WindowDropTopic, event)
		}))
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
			}
		})
	}, nil
}

// registerWindowEvent maps one fixed Wails event to portable caller-owned state.
func registerWindowEvent(win application.Window, eventType events.WindowEventType, kind desktop.WindowEventKind) func() {
	return win.OnWindowEvent(eventType, func(_ *application.WindowEvent) {
		win.EmitEvent(desktop.WindowEventTopic, buildWindowEvent(win, kind))
	})
}

// buildWindowEvent snapshots portable geometry without forwarding native event arguments.
func buildWindowEvent(win application.Window, kind desktop.WindowEventKind) desktop.WindowEvent {
	x, y := win.Position()
	event := desktop.WindowEvent{
		Kind:     kind,
		WindowID: fmt.Sprint(win.ID()),
		X:        x,
		Y:        y,
		Width:    win.Width(),
		Height:   win.Height(),
	}
	if kind == desktop.WindowZoomChanged {
		event.Zoom = win.GetZoom()
	}
	return event
}

// getDroppedFiles copies and bounds Wails drop paths before they cross the bridge.
func getDroppedFiles(event *application.WindowEvent) ([]string, bool) {
	if event == nil || event.Context() == nil {
		return nil, false
	}
	files := event.Context().DroppedFiles()
	if len(files) == 0 || len(files) > 32 {
		return nil, false
	}
	copied := make([]string, len(files))
	for i, file := range files {
		if file == "" || len(file) > 32768 || !utf8.ValidString(file) || strings.ContainsRune(file, 0) {
			return nil, false
		}
		copied[i] = file
	}
	return copied, true
}
