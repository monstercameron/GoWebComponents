package pluginruntime

import "sync"

// CleanupFunc stores one kernel-managed cleanup callback.
type CleanupFunc func() error

// SubscriptionHandle stores one cleanup-backed subscription handle.
type SubscriptionHandle struct {
	getOnce sync.Once
	getStop CleanupFunc
}

// Stop stops one subscription handle once.
func (parseHandle *SubscriptionHandle) Stop() error {
	if parseHandle == nil {
		return nil
	}
	var getErr error
	parseHandle.getOnce.Do(func() {
		if parseHandle.getStop != nil {
			getErr = parseHandle.getStop()
		}
	})
	return getErr
}
