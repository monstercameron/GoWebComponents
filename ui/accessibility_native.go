//go:build !js || !wasm
// +build !js !wasm

package ui

type FieldErrors map[string]string

type FocusOptions struct {
	PreventScroll bool
}

type FocusTrapOptions struct {
	ContainerSelector     string
	InitialFocusSelector  string
	FallbackFocusSelector string
	Active                bool
	RestoreFocus          bool
}

type FocusManager struct{}

type CompositeItem struct {
	ID       string
	Text     string
	Disabled bool
}

type CompositeNavigationOptions struct {
	Orientation string
	Loop        bool
	InitialIndex int
}

type CompositeNavigation struct{}

type AnnouncementMode string

const (
	AnnouncementPolite   AnnouncementMode = "polite"
	AnnouncementAssertive AnnouncementMode = "assertive"
)

type Announcer struct{}

type AccessibleOverlayProps struct {
	Open                 bool
	Target               PortalTarget
	AppRootSelector      string
	SurfaceID            string
	LabelledBy           string
	DescribedBy          string
	Role                 string
	Modal                bool
	InitialFocusSelector string
	RestoreFocus         bool
	TrapFocus            bool
	CloseOnEscape        bool
	CloseOnOutsideClick  bool
	LockScroll           bool
	BackdropClass        string
	SurfaceClass         string
	Child                Node
	Children             []Node
	OnDismiss            func()
}

func UseFocusManager() FocusManager {
	return FocusManager{}
}

func (m FocusManager) FocusFirstError(errors FieldErrors, fieldIDs map[string]string, order ...string) bool {
	return false
}

func UseCompositeNavigation(items []CompositeItem, options ...CompositeNavigationOptions) CompositeNavigation {
	return CompositeNavigation{}
}

func (n CompositeNavigation) ActiveIndex() int { return -1 }

func (n CompositeNavigation) ActiveID() string { return "" }

func (n CompositeNavigation) ActiveDescendant() string { return "" }

func (n CompositeNavigation) IsActive(index int) bool { return false }

func (n CompositeNavigation) TabIndex(index int) int { return -1 }

func (n CompositeNavigation) SetActive(index int) {}

func (n CompositeNavigation) MoveNext() {}

func (n CompositeNavigation) MovePrevious() {}

func (n CompositeNavigation) MoveHome() {}

func (n CompositeNavigation) MoveEnd() {}

func (n CompositeNavigation) OnKeyDown(event interface{}) {}

func UseAnnouncer() Announcer {
	return Announcer{}
}

func (a Announcer) Announce(mode AnnouncementMode, message string) {}

func (a Announcer) Polite(message string) {}

func (a Announcer) Assertive(message string) {}

func (a Announcer) Clear() {}

func (a Announcer) PoliteID() string { return "" }

func (a Announcer) AssertiveID() string { return "" }

func (a Announcer) Region() Node { return nil }

func AccessibleOverlay(props AccessibleOverlayProps) Node {
	children := make([]Node, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, props.Children...)
	return Fragment(children...)
}

func (m FocusManager) RememberActive() bool {
	return false
}

func (m FocusManager) FocusSelector(selector string, options ...FocusOptions) bool {
	return false
}

func (m FocusManager) FocusByID(id string, options ...FocusOptions) bool {
	return false
}

func (m FocusManager) FocusFirst(containerSelector string, options ...FocusOptions) bool {
	return false
}

func (m FocusManager) Restore(options ...FocusOptions) bool {
	return false
}

func UseFocusTrap(options FocusTrapOptions) {}

func useOverlayEscape(active bool, onDismiss func()) {}

func useOverlayScrollLock(active bool) {}

func useOverlayBackgroundInert(selector string, active bool) {}