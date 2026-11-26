//go:build windows

package wails

import (
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type fakeEventWindow struct {
	fakeWindow
	mu        sync.Mutex
	handlers  map[events.WindowEventType]func(*application.WindowEvent)
	removed   int
	emitted   []desktop.WindowEvent
	dropTopic []desktop.WindowEvent
}

// OnWindowEvent registers a handler for one event type and counts idempotent unregistrations.
func (w *fakeEventWindow) OnWindowEvent(eventType events.WindowEventType, handler func(*application.WindowEvent)) func() {
	w.mu.Lock()
	if w.handlers == nil {
		w.handlers = make(map[events.WindowEventType]func(*application.WindowEvent))
	}
	w.handlers[eventType] = handler
	w.mu.Unlock()
	return func() {
		w.mu.Lock()
		delete(w.handlers, eventType)
		w.removed++
		w.mu.Unlock()
	}
}

// EmitEvent records emitted portable window and drop events.
func (w *fakeEventWindow) EmitEvent(topic string, data ...any) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(data) == 1 {
		if event, ok := data[0].(desktop.WindowEvent); ok {
			if topic == desktop.WindowDropTopic {
				w.dropTopic = append(w.dropTopic, event)
			} else if topic == desktop.WindowEventTopic {
				w.emitted = append(w.emitted, event)
			}
		}
	}
	return true
}

// TestAttachWindowEventsRejectsNilOwner verifies event attachment fails closed without a window.
func TestAttachWindowEventsRejectsNilOwner(t *testing.T) {
	if _, err := AttachWindowEvents(nil, WindowEventOptions{}); !interop.IsCode(err, interop.CodeInvalid) {
		t.Fatalf("nil owner error=%v", err)
	}
}

// TestAttachWindowEventsDispatchesPortableKindsAndCleansUp verifies v3.1.0 window events and teardown.
func TestAttachWindowEventsDispatchesPortableKindsAndCleansUp(t *testing.T) {
	win := &fakeEventWindow{
		fakeWindow: fakeWindow{id: 42, name: "lab", x: 10, y: 20, width: 800, height: 600, zoom: 1.5},
	}
	cleanup, err := AttachWindowEvents(win, WindowEventOptions{EnableFileDrop: true})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[events.WindowEventType]desktop.WindowEventKind{
		events.Common.WindowShow:         desktop.WindowShown,
		events.Common.WindowHide:         desktop.WindowHidden,
		events.Common.WindowUnFullscreen: desktop.WindowUnfullscreened,
		events.Common.WindowUnMinimise:   desktop.WindowUnminimized,
		events.Common.WindowUnMaximise:   desktop.WindowUnmaximized,
		events.Common.WindowDPIChanged:   desktop.WindowDPIChanged,
		events.Common.WindowZoom:         desktop.WindowZoomChanged,
	}
	for nativeType, wantKind := range expected {
		handler := win.handlers[nativeType]
		if handler == nil {
			t.Fatalf("missing handler for native event %d", nativeType)
		}
		handler(application.NewWindowEvent())
		last := win.emitted[len(win.emitted)-1]
		wantZoom := 0.0
		if wantKind == desktop.WindowZoomChanged {
			wantZoom = 1.5
		}
		if last.Kind != wantKind || last.WindowID != "42" || last.Zoom != wantZoom {
			t.Fatalf("unexpected event for %d: %+v", nativeType, last)
		}
	}
	totalRegistered := len(win.handlers)
	cleanup()
	cleanup()
	if win.removed != totalRegistered || len(win.handlers) != 0 {
		t.Fatalf("cleanup not idempotent or incomplete: removed=%d total=%d remaining=%d", win.removed, totalRegistered, len(win.handlers))
	}
}
