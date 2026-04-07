package pluginruntime

func init() {
	_ = RegisterBuiltinService(ServiceRegistration{Key: ServiceKeyDOM, Value: buildDefaultDOMService{}})
	_ = RegisterBuiltinService(ServiceRegistration{Key: ServiceKeyStyle, Value: buildDefaultStyleService{}})
	_ = RegisterBuiltinService(ServiceRegistration{Key: ServiceKeyEvents, Value: buildDefaultEventService{}})
	_ = RegisterBuiltinService(ServiceRegistration{Key: ServiceKeyAssets, Value: buildDefaultAssetService{}})
	_ = RegisterBuiltinService(ServiceRegistration{Key: ServiceKeySecurity, Value: buildDefaultSecurityService{}})
	_ = RegisterBuiltinService(ServiceRegistration{Key: ServiceKeyCapture, Value: buildDefaultCaptureService{}})
}

type buildDefaultDOMService struct{}

type buildDefaultStyleService struct{}

type buildDefaultStylePatchHandle struct{}

type buildDefaultEventService struct{}

type buildDefaultAssetService struct{}

type buildDefaultSecurityService struct{}

type buildDefaultCaptureService struct{}

// GetDOMSnapshot returns one empty default DOM snapshot.
func (buildDefaultDOMService) GetDOMSnapshot(parseBudget QueryBudget) (DOMSnapshot, error) {
	return DOMSnapshot{Meta: BuildSnapshotMeta(BackendIDNative, false)}, nil
}

// HighlightNode is a no-op for the default DOM service.
func (buildDefaultDOMService) HighlightNode(parseNodeID string, parseOptions CommandOptions) error {
	return nil
}

// ScrollNodeIntoView is a no-op for the default DOM service.
func (buildDefaultDOMService) ScrollNodeIntoView(parseNodeID string, parseOptions CommandOptions) error {
	return nil
}

// GetStyleSnapshot returns one empty default style snapshot.
func (buildDefaultStyleService) GetStyleSnapshot(parseBudget QueryBudget) (StyleSnapshot, error) {
	return StyleSnapshot{Meta: BuildSnapshotMeta(BackendIDNative, false)}, nil
}

// ApplyStyleVariables applies no changes for the default style service.
func (buildDefaultStyleService) ApplyStyleVariables(parseVariables map[string]string, parseOptions CommandOptions) (StylePatchHandle, error) {
	return buildDefaultStylePatchHandle{}, nil
}

// RemoveStylePatch removes one default no-op style patch.
func (buildDefaultStylePatchHandle) RemoveStylePatch() error {
	return nil
}

// GetEventSnapshot returns one empty default event snapshot.
func (buildDefaultEventService) GetEventSnapshot(parseBudget QueryBudget) (EventSnapshot, error) {
	return EventSnapshot{Meta: BuildSnapshotMeta(BackendIDNative, false)}, nil
}

// GetAssetSnapshot returns one empty default asset snapshot.
func (buildDefaultAssetService) GetAssetSnapshot(parseBudget QueryBudget) (AssetSnapshot, error) {
	return AssetSnapshot{Meta: BuildSnapshotMeta(BackendIDNative, false)}, nil
}

// RefreshAssets is a no-op for the default asset service.
func (buildDefaultAssetService) RefreshAssets(parseOptions CommandOptions) error {
	return nil
}

// GetSecuritySnapshot returns one empty default security snapshot.
func (buildDefaultSecurityService) GetSecuritySnapshot(parseBudget QueryBudget) (SecuritySnapshot, error) {
	return SecuritySnapshot{Meta: BuildSnapshotMeta(BackendIDNative, false)}, nil
}

// GetCaptureSnapshot returns one empty default capture snapshot.
func (buildDefaultCaptureService) GetCaptureSnapshot(parseBudget QueryBudget) (CaptureSnapshot, error) {
	return CaptureSnapshot{Meta: BuildSnapshotMeta(BackendIDNative, false)}, nil
}
