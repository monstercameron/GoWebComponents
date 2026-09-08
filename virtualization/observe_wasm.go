//go:build js && wasm

package virtualization

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// Subscription tracks one active viewport observation.
type Subscription struct {
	cancel func()
}

// Cancel stops the active viewport observation.
func (parseSubscription Subscription) Cancel() {
	if parseSubscription.cancel != nil {
		parseSubscription.cancel()
	}
}

// ObserveOwnedViewport tracks scroll offset, viewport height, and visible plus
// rendered row ranges for one owned scroll container.
func ObserveOwnedViewport(parseViewportElement interop.Element, parseViewportConfig ViewportConfig, parseViewportHandler func(ViewportState)) (Subscription, error) {
	if _, parseViewportErr := normalizeConfig(parseViewportConfig); parseViewportErr != nil {
		return Subscription{}, parseViewportErr
	}
	if parseViewportHandler == nil {
		return Subscription{}, errors.New("virtualization: viewport handler is nil")
	}

	parseViewportPublish := func() error {
		parseViewportScrollTop, _, parseViewportClientHeight, parseViewportMetricsErr := parseViewportElement.ScrollMetrics()
		if parseViewportMetricsErr != nil {
			return parseViewportMetricsErr
		}
		parseViewportState, parseViewportStateErr := ComputeViewportState(parseViewportConfig, parseViewportScrollTop, parseViewportClientHeight)
		if parseViewportStateErr != nil {
			return parseViewportStateErr
		}
		parseViewportHandler(parseViewportState)
		return nil
	}

	parseScrollSub, parseScrollErr := parseViewportElement.Listen("scroll", func(interop.BrowserEvent) {
		_ = parseViewportPublish()
	})
	if parseScrollErr != nil {
		return Subscription{}, parseScrollErr
	}
	parseResizeSub, parseResizeErr := parseViewportElement.ObserveResize(func(interop.ResizeEntry) {
		_ = parseViewportPublish()
	})
	if parseResizeErr != nil {
		parseScrollSub.Cancel()
		return Subscription{}, parseResizeErr
	}
	if parseViewportPublishErr := parseViewportPublish(); parseViewportPublishErr != nil {
		parseResizeSub.Cancel()
		parseScrollSub.Cancel()
		return Subscription{}, parseViewportPublishErr
	}

	return Subscription{cancel: func() {
		parseResizeSub.Cancel()
		parseScrollSub.Cancel()
	}}, nil
}
