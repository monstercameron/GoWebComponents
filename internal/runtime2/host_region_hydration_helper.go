package runtime2

import "fmt"

// HostRegionHydrationAttachHelper exposes a narrow host-side hydration attach surface without exposing mutable adapter internals.
type HostRegionHydrationAttachHelper struct {
	storeHostRegionAdapter *HostRegionAdapter
}

// BuildHostRegionHydrationAttachHelper builds one hydration helper from one mounted host region adapter.
func BuildHostRegionHydrationAttachHelper(parseHostRegionAdapter *HostRegionAdapter) (HostRegionHydrationAttachHelper, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionHydrationAttachHelper{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	return HostRegionHydrationAttachHelper{
		storeHostRegionAdapter: parseHostRegionAdapter,
	}, nil
}

// GetHostRegionInstanceID reports the helper-owned host region instance ID.
func (parseHostRegionHydrationAttachHelper HostRegionHydrationAttachHelper) GetHostRegionInstanceID() RegionInstanceID {
	if parseHostRegionHydrationAttachHelper.storeHostRegionAdapter == nil {
		return ""
	}
	return parseHostRegionHydrationAttachHelper.storeHostRegionAdapter.GetHostRegionInstanceID()
}

// HandleHostRegionHydrationComplete marks hydration complete through the helper-owned adapter.
func (parseHostRegionHydrationAttachHelper HostRegionHydrationAttachHelper) HandleHostRegionHydrationComplete() error {
	if parseHostRegionHydrationAttachHelper.storeHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	return parseHostRegionHydrationAttachHelper.storeHostRegionAdapter.HandleHostRegionHydrationComplete()
}

// HandleHostRegionRegisterHydratedShellAnchor registers one hydrated shell anchor through the helper-owned adapter.
func (parseHostRegionHydrationAttachHelper HostRegionHydrationAttachHelper) HandleHostRegionRegisterHydratedShellAnchor(parseNodeID uint64, parseTag string) error {
	if parseHostRegionHydrationAttachHelper.storeHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	return parseHostRegionHydrationAttachHelper.storeHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(parseNodeID, parseTag)
}

// HandleHostRegionPostHydrationAttach performs post-hydration attach through the helper-owned adapter.
func (parseHostRegionHydrationAttachHelper HostRegionHydrationAttachHelper) HandleHostRegionPostHydrationAttach() (HostRegionHydrationAttachResult, error) {
	if parseHostRegionHydrationAttachHelper.storeHostRegionAdapter == nil {
		return HostRegionHydrationAttachResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	return parseHostRegionHydrationAttachHelper.storeHostRegionAdapter.HandleHostRegionPostHydrationAttach()
}

// GetHostRegionIsHydrationComplete reports hydration completion from helper-owned adapter state.
func (parseHostRegionHydrationAttachHelper HostRegionHydrationAttachHelper) GetHostRegionIsHydrationComplete() bool {
	if parseHostRegionHydrationAttachHelper.storeHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionHydrationAttachHelper.storeHostRegionAdapter.GetHostRegionIsHydrationComplete()
}

// HasHostRegionPostHydrationAttached reports post-hydration attach status from helper-owned adapter state.
func (parseHostRegionHydrationAttachHelper HostRegionHydrationAttachHelper) HasHostRegionPostHydrationAttached() bool {
	if parseHostRegionHydrationAttachHelper.storeHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionHydrationAttachHelper.storeHostRegionAdapter.HasHostRegionPostHydrationAttached()
}
