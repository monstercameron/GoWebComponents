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
func (s Subscription) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

// ObserveOwnedViewport tracks scroll offset, viewport height, and visible plus
// rendered row ranges for one owned scroll container.
func ObserveOwnedViewport(element interop.Element, config ViewportConfig, handler func(ViewportState)) (Subscription, error) {
	if _, err := normalizeConfig(config); err != nil {
		return Subscription{}, err
	}
	if handler == nil {
		return Subscription{}, errors.New("virtualization: viewport handler is nil")
	}

	publish := func() error {
		scrollTop, _, clientHeight, err := element.ScrollMetrics()
		if err != nil {
			return err
		}
		state, err := ComputeViewportState(config, scrollTop, clientHeight)
		if err != nil {
			return err
		}
		handler(state)
		return nil
	}

	scrollSub, err := element.Listen("scroll", func(interop.BrowserEvent) {
		_ = publish()
	})
	if err != nil {
		return Subscription{}, err
	}
	resizeSub, err := element.ObserveResize(func(interop.ResizeEntry) {
		_ = publish()
	})
	if err != nil {
		scrollSub.Cancel()
		return Subscription{}, err
	}
	if err := publish(); err != nil {
		resizeSub.Cancel()
		scrollSub.Cancel()
		return Subscription{}, err
	}

	return Subscription{cancel: func() {
		resizeSub.Cancel()
		scrollSub.Cancel()
	}}, nil
}
