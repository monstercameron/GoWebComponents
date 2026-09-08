package desktop

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v5/kvstate"
)

// SubscribeStorage reloads all existing kvstate bindings under a logical database
// name after a storage.changed commit. Reloading all keys makes topic-level burst
// coalescing safe. It never republishes events. Cancel it on unmount/window close.
// Backend selection remains explicit through kvstate.Options.Backend. Set
// Options.ExternalInvalidation=true to avoid browser BroadcastChannel forwarding.
func SubscribeStorage(parseContext context.Context, parseClient Client, parseName string, parseOnError func(error)) (func(), error) {
	return Subscribe[any](parseContext, parseClient, "storage.changed", func(_ any, parseErr error) {
		if parseErr != nil {
			if parseOnError != nil {
				parseOnError(parseErr)
			}
			return
		}
		kvstate.Invalidate(parseName)
	})
}
