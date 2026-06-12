//go:build !js || !wasm

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
	Orientation  string
	Loop         bool
	InitialIndex int
}

type CompositeNavigation struct{}

type AnnouncementMode string

const (
	AnnouncementPolite    AnnouncementMode = "polite"
	AnnouncementAssertive AnnouncementMode = "assertive"
)

type Announcer struct{}

type AccessibleOverlayProps struct {
	Open                  bool
	Target                PortalTarget
	AppRootSelector       string
	SurfaceID             string
	Kind                  OverlayKind
	LabelledBy            string
	DescribedBy           string
	Role                  string
	Modal                 bool
	InitialFocusSelector  string
	FallbackFocusSelector string
	RestoreFocus          bool
	TrapFocus             bool
	CloseOnEscape         bool
	CloseOnOutsideClick   bool
	LockScroll            bool
	BackgroundInert       bool
	Backdrop              bool
	BaseZIndex            int
	AnchorSelector        string
	Positioning           string
	BackdropClass         string
	SurfaceClass          string
	BackdropStyle         map[string]string
	SurfaceStyle          map[string]string
	Child                 Node
	Children              []Node
	OnDismiss             func()
}

// UseFocusManager returns a no-op FocusManager for non-browser targets.
func UseFocusManager() FocusManager {
	return FocusManager{}
}

// FocusFirstError is a core package helper.
func (parseM FocusManager) FocusFirstError(parseErrors FieldErrors, parseFieldIDs map[string]string, parseOrder ...string) bool {
	return false
}

// UseCompositeNavigation returns a no-op CompositeNavigation for non-browser targets.
func UseCompositeNavigation(parseItems []CompositeItem, parseOptions ...CompositeNavigationOptions) CompositeNavigation {
	return CompositeNavigation{}
}

// ActiveIndex is a core package helper.
func (parseN CompositeNavigation) ActiveIndex() int { return -1 }

// ActiveID is a core package helper.
func (parseN CompositeNavigation) ActiveID() string { return "" }

// ActiveDescendant is a core package helper.
func (parseN CompositeNavigation) ActiveDescendant() string { return "" }

// IsActive is a core package helper.
func (parseN CompositeNavigation) IsActive(parseIndex int) bool { return false }

// TabIndex is a core package helper.
func (parseN CompositeNavigation) TabIndex(parseIndex int) int { return -1 }

// SetActive is a core package helper.
func (parseN CompositeNavigation) SetActive(parseIndex int) {}

// MoveNext is a core package helper.
func (parseN CompositeNavigation) MoveNext() {}

// MovePrevious is a core package helper.
func (parseN CompositeNavigation) MovePrevious() {}

// MoveHome is a core package helper.
func (parseN CompositeNavigation) MoveHome() {}

// MoveEnd is a core package helper.
func (parseN CompositeNavigation) MoveEnd() {}

// OnKeyDown is a core package helper.
func (parseN CompositeNavigation) OnKeyDown(parseEvent any) {}

// UseAnnouncer returns a no-op Announcer for non-browser targets.
func UseAnnouncer() Announcer {
	return Announcer{}
}

// Announce is a core package helper.
func (parseA Announcer) Announce(parseMode AnnouncementMode, parseMessage string) {}

// Polite is a core package helper.
func (parseA Announcer) Polite(parseMessage string) {}

// Assertive is a core package helper.
func (parseA Announcer) Assertive(parseMessage string) {}

// Clear is a core package helper.
func (parseA Announcer) Clear() {}

// PoliteID is a core package helper.
func (parseA Announcer) PoliteID() string { return "" }

// AssertiveID is a core package helper.
func (parseA Announcer) AssertiveID() string { return "" }

// Region is a core package helper.
func (parseA Announcer) Region() Node { return nil }

// AccessibleOverlay renders an accessible overlay node delegating to Overlay for server rendering.
func AccessibleOverlay(parseProps AccessibleOverlayProps) Node {
	return Overlay(OverlayProps{
		Open:                  parseProps.Open,
		Target:                parseProps.Target,
		AppRootSelector:       parseProps.AppRootSelector,
		SurfaceID:             parseProps.SurfaceID,
		Kind:                  parseProps.Kind,
		Role:                  parseProps.Role,
		LabelledBy:            parseProps.LabelledBy,
		DescribedBy:           parseProps.DescribedBy,
		InitialFocusSelector:  parseProps.InitialFocusSelector,
		FallbackFocusSelector: parseProps.FallbackFocusSelector,
		Modal:                 parseProps.Modal,
		Backdrop:              parseProps.Backdrop,
		TrapFocus:             parseProps.TrapFocus,
		RestoreFocus:          parseProps.RestoreFocus,
		CloseOnEscape:         parseProps.CloseOnEscape,
		CloseOnOutsideClick:   parseProps.CloseOnOutsideClick,
		LockScroll:            parseProps.LockScroll,
		BackgroundInert:       parseProps.BackgroundInert,
		BaseZIndex:            parseProps.BaseZIndex,
		AnchorSelector:        parseProps.AnchorSelector,
		Positioning:           parseProps.Positioning,
		BackdropClass:         parseProps.BackdropClass,
		SurfaceClass:          parseProps.SurfaceClass,
		BackdropStyle:         parseProps.BackdropStyle,
		SurfaceStyle:          parseProps.SurfaceStyle,
		Child:                 parseProps.Child,
		Children:              parseProps.Children,
		OnDismiss:             parseProps.OnDismiss,
	})
}

// RememberActive is a core package helper.
func (parseM FocusManager) RememberActive() bool {
	return false
}

// FocusSelector is a core package helper.
func (parseM FocusManager) FocusSelector(parseSelector string, parseOptions ...FocusOptions) bool {
	return false
}

// FocusByID is a core package helper.
func (parseM FocusManager) FocusByID(parseId string, parseOptions ...FocusOptions) bool {
	return false
}

// FocusFirst is a core package helper.
func (parseM FocusManager) FocusFirst(parseContainerSelector string, parseOptions ...FocusOptions) bool {
	return false
}

// Restore is a core package helper.
func (parseM FocusManager) Restore(parseOptions ...FocusOptions) bool {
	return false
}

// UseFocusTrap is a no-op for non-browser targets.
func UseFocusTrap(parseOptions FocusTrapOptions) {}
