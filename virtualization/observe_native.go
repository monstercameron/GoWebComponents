//go:build !js || !wasm

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

// ObserveOwnedViewport is unavailable outside browser builds.
func ObserveOwnedViewport(_ interop.Element, config ViewportConfig, _ func(ViewportState)) (Subscription, error) {
	if _, err := normalizeConfig(config); err != nil {
		return Subscription{}, err
	}
	return Subscription{}, errors.New("virtualization: viewport observation is unavailable in this build")
}
