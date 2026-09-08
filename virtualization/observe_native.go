//go:build !js || !wasm

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

// ObserveOwnedViewport is unavailable outside browser builds.
func ObserveOwnedViewport(_ interop.Element, parseViewportConfig ViewportConfig, _ func(ViewportState)) (Subscription, error) {
	if _, parseViewportErr := normalizeConfig(parseViewportConfig); parseViewportErr != nil {
		return Subscription{}, parseViewportErr
	}
	return Subscription{}, errors.New("virtualization: viewport observation is unavailable in this build")
}
