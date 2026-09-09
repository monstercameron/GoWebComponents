//go:build windows

package wails

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// WindowEventOptions configures explicit caller-window event registration.
type WindowEventOptions struct{ EnableFileDrop bool }

// AttachWindowEvents installs the portable event allowlist on exactly one Wails window.
func AttachWindowEvents(parseWindow application.Window, parseOptions WindowEventOptions) (func(), error) {
	if parseWindow == nil {
		return nil, invalid("window event owner required")
	}
	parseCleanups := []func(){registerWindowEvent(parseWindow, events.Common.WindowFocus, desktop.WindowFocused), registerWindowEvent(parseWindow, events.Common.WindowLostFocus, desktop.WindowBlurred), registerWindowEvent(parseWindow, events.Common.WindowDidMove, desktop.WindowMoved), registerWindowEvent(parseWindow, events.Common.WindowDidResize, desktop.WindowResized), registerWindowEvent(parseWindow, events.Common.WindowFullscreen, desktop.WindowFullscreen), registerWindowEvent(parseWindow, events.Common.WindowMinimise, desktop.WindowMinimized), registerWindowEvent(parseWindow, events.Common.WindowMaximise, desktop.WindowMaximized), registerWindowEvent(parseWindow, events.Common.WindowRestore, desktop.WindowRestored), registerWindowEvent(parseWindow, events.Common.WindowClosing, desktop.WindowClosing)}
	if parseOptions.EnableFileDrop {
		parseCleanups = append(parseCleanups, parseWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(parseNativeEvent *application.WindowEvent) {
			parseFiles, isValid := getDroppedFiles(parseNativeEvent)
			if !isValid {
				return
			}
			parseEvent := buildWindowEvent(parseWindow, desktop.WindowFilesDropped)
			parseEvent.Files = parseFiles
			parseWindow.EmitEvent(desktop.WindowDropTopic, parseEvent)
		}))
	}
	var parseOnce sync.Once
	return func() {
		parseOnce.Do(func() {
			for parseIndex := len(parseCleanups) - 1; parseIndex >= 0; parseIndex-- {
				parseCleanups[parseIndex]()
			}
		})
	}, nil
}

// registerWindowEvent maps one fixed Wails event to portable caller-owned state.
func registerWindowEvent(parseWindow application.Window, parseType events.WindowEventType, parseKind desktop.WindowEventKind) func() {
	return parseWindow.OnWindowEvent(parseType, func(_ *application.WindowEvent) {
		parseWindow.EmitEvent(desktop.WindowEventTopic, buildWindowEvent(parseWindow, parseKind))
	})
}

// buildWindowEvent snapshots portable geometry without forwarding native event arguments.
func buildWindowEvent(parseWindow application.Window, parseKind desktop.WindowEventKind) desktop.WindowEvent {
	parseInfo := windowInfo(parseWindow)
	return desktop.WindowEvent{Kind: parseKind, WindowID: parseInfo.ID, X: parseInfo.X, Y: parseInfo.Y, Width: parseInfo.Width, Height: parseInfo.Height}
}

// getDroppedFiles copies and bounds Wails drop paths before they cross the bridge.
func getDroppedFiles(parseEvent *application.WindowEvent) ([]string, bool) {
	if parseEvent == nil || parseEvent.Context() == nil {
		return nil, false
	}
	parseFiles := parseEvent.Context().DroppedFiles()
	if len(parseFiles) == 0 || len(parseFiles) > 32 {
		return nil, false
	}
	parseCopy := make([]string, len(parseFiles))
	for parseIndex, parseFile := range parseFiles {
		if parseFile == "" || len(parseFile) > 32768 || !utf8.ValidString(parseFile) || strings.ContainsRune(parseFile, 0) {
			return nil, false
		}
		parseCopy[parseIndex] = parseFile
	}
	return parseCopy, true
}
