package desktop

import (
	"context"
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v8/interop"
)

// WindowEvents identifies typed events from the caller-owned native window.
const WindowEvents Feature = "window-events"

// WindowEventTopic carries portable caller-window lifecycle and geometry events.
const WindowEventTopic = "desktop.window.event"

// WindowDropTopic carries bounded file-drop payloads when the host explicitly enables them.
const WindowDropTopic = "desktop.window.files-dropped"

// WindowEventKind identifies a fixed portable window event; arbitrary backend event IDs are not accepted.
type WindowEventKind string

const (
	WindowFocused        WindowEventKind = "focused"
	WindowBlurred        WindowEventKind = "blurred"
	WindowMoved          WindowEventKind = "moved"
	WindowResized        WindowEventKind = "resized"
	WindowShown          WindowEventKind = "shown"
	WindowHidden         WindowEventKind = "hidden"
	WindowFullscreen     WindowEventKind = "fullscreen"
	WindowUnfullscreened WindowEventKind = "unfullscreen"
	WindowMinimized      WindowEventKind = "minimized"
	WindowUnminimized    WindowEventKind = "unminimized"
	WindowMaximized      WindowEventKind = "maximized"
	WindowUnmaximized    WindowEventKind = "unmaximized"
	WindowRestored       WindowEventKind = "restored"
	WindowDPIChanged     WindowEventKind = "dpi-changed"
	WindowZoomChanged    WindowEventKind = "zoom-changed"
	WindowClosing        WindowEventKind = "closing"
	WindowFilesDropped   WindowEventKind = "files-dropped"
)

// WindowEvent reports only portable state for the subscribing caller's own window.
type WindowEvent struct {
	Kind     WindowEventKind `json:"kind"`
	WindowID string          `json:"windowId"`
	X        int             `json:"x,omitempty"`
	Y        int             `json:"y,omitempty"`
	Width    int             `json:"width,omitempty"`
	Height   int             `json:"height,omitempty"`
	Zoom     float64         `json:"zoom,omitempty"`
	Files    []string        `json:"files,omitempty"`
}

// WindowEventTopics returns the exact topics a host should advertise after installing window handlers.
func WindowEventTopics(isFileDropEnabled bool) []string {
	parseTopics := []string{WindowEventTopic}
	if isFileDropEnabled {
		parseTopics = append(parseTopics, WindowDropTopic)
	}
	return parseTopics
}

// SubscribeWindowEvents observes typed state events for the caller-owned window.
// Closing is best-effort because the runtime may tear down before delivery.
func (parseClient Client) SubscribeWindowEvents(parseContext context.Context, parseHandler func(WindowEvent, error)) (func(), error) {
	if parseHandler == nil {
		return nil, getError(WindowEventTopic, interop.CodeInvalid, errors.New("window event handler required"))
	}
	if parseErr := parseClient.Require(WindowEvents); parseErr != nil {
		return nil, parseErr
	}
	return Subscribe(parseContext, parseClient, WindowEventTopic, func(parseEvent WindowEvent, parseErr error) {
		if parseErr == nil {
			parseErr = validateWindowEvent(parseEvent, false)
		}
		parseHandler(parseEvent, parseErr)
	})
}

// SubscribeWindowDrops observes bounded file drops only when the Wails host opted in and advertised the topic.
func (parseClient Client) SubscribeWindowDrops(parseContext context.Context, parseHandler func(WindowEvent, error)) (func(), error) {
	if parseHandler == nil {
		return nil, getError(WindowDropTopic, interop.CodeInvalid, errors.New("window drop handler required"))
	}
	if parseErr := parseClient.Require(WindowEvents); parseErr != nil {
		return nil, parseErr
	}
	parseCapabilities, parseErr := parseClient.GetCapabilities()
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasName(parseCapabilities.Topics, WindowDropTopic) {
		return nil, getError(WindowDropTopic, interop.CodeUnavailable, errors.New("window file drops are disabled or unavailable"))
	}
	return Subscribe(parseContext, parseClient, WindowDropTopic, func(parseEvent WindowEvent, parseErr error) {
		if parseErr == nil {
			parseErr = validateWindowEvent(parseEvent, true)
		}
		parseHandler(parseEvent, parseErr)
	})
}

// validateWindowEvent rejects malformed or cross-topic payloads from a native host.
func validateWindowEvent(event WindowEvent, isDrop bool) error {
	if event.WindowID == "" || len(event.WindowID) > 256 || !utf8.ValidString(event.WindowID) || strings.ContainsRune(event.WindowID, 0) {
		return getError(WindowEventTopic, interop.CodeDecode, errors.New("invalid window event owner"))
	}
	if event.X < -100000 || event.X > 100000 || event.Y < -100000 || event.Y > 100000 || event.Width < 0 || event.Width > 32768 || event.Height < 0 || event.Height > 32768 || math.IsNaN(event.Zoom) || math.IsInf(event.Zoom, 0) || event.Zoom < 0 || event.Zoom > 100 {
		return getError(WindowEventTopic, interop.CodeDecode, errors.New("invalid window event geometry"))
	}
	if isDrop {
		if event.Kind != WindowFilesDropped || len(event.Files) == 0 || len(event.Files) > 32 {
			return getError(WindowDropTopic, interop.CodeDecode, errors.New("invalid window drop event"))
		}
		for _, file := range event.Files {
			if file == "" || len(file) > 32768 || !utf8.ValidString(file) || strings.ContainsRune(file, 0) {
				return getError(WindowDropTopic, interop.CodeDecode, errors.New("invalid dropped file path"))
			}
		}
		return nil
	}
	if len(event.Files) != 0 {
		return getError(WindowEventTopic, interop.CodeDecode, errors.New("files require the opted-in drop topic"))
	}
	switch event.Kind {
	case WindowFocused, WindowBlurred, WindowMoved, WindowResized, WindowShown, WindowHidden, WindowFullscreen, WindowUnfullscreened, WindowMinimized, WindowUnminimized, WindowMaximized, WindowUnmaximized, WindowRestored, WindowDPIChanged, WindowZoomChanged, WindowClosing:
		return nil
	default:
		return getError(WindowEventTopic, interop.CodeDecode, errors.New("unknown window event kind"))
	}
}
