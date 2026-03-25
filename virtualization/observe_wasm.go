//go:build js && wasm

package virtualization

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/interop"
)

// Subscription tracks one active viewport observation.
type Subscription struct {
	cancel func()
}

// Cancel stops the active viewport observation.
func (parseS Subscription) Cancel() {
	if parseS.cancel != nil {
		parseS.cancel()
	}
}

// ObserveOwnedViewport tracks scroll offset, viewport height, and visible plus
// rendered row ranges for one owned scroll container.
func ObserveOwnedViewport(parseElement interop.Element, parseConfig ViewportConfig, parseHandler func(ViewportState)) (Subscription, error) {
	if _, parseErr := normalizeConfig(parseConfig); parseErr != nil {
		return Subscription{}, parseErr
	}
	if parseHandler == nil {
		return Subscription{}, errors.New("virtualization: viewport handler is nil")
	}

	parsePublish := func() error {
		parseScrollTop, _, parseClientHeight, parseErr2 := parseElement.ScrollMetrics()
		if parseErr2 != nil {
			return parseErr2
		}
		parseState, parseErr2 := ComputeViewportState(parseConfig, parseScrollTop, parseClientHeight)
		if parseErr2 != nil {
			return parseErr2
		}
		parseHandler(parseState)
		return nil
	}

	parseScrollSub, parseErr3 := parseElement.Listen("scroll", func(interop.BrowserEvent) {
		_ = parsePublish()
	})
	if parseErr3 != nil {
		return Subscription{}, parseErr3
	}
	parseResizeSub, parseErr3 := parseElement.ObserveResize(func(interop.ResizeEntry) {
		_ = parsePublish()
	})
	if parseErr3 != nil {
		parseScrollSub.Cancel()
		return Subscription{}, parseErr3
	}
	if parseErr4 := parsePublish(); parseErr4 != nil {
		parseResizeSub.Cancel()
		parseScrollSub.Cancel()
		return Subscription{}, parseErr4
	}

	return Subscription{cancel: func() {
		parseResizeSub.Cancel()
		parseScrollSub.Cancel()
	}}, nil
}
