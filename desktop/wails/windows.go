package wails

// WindowBackdrop identifies the Windows DWM backdrop material for a host-configured template.
type WindowBackdrop string

const (
	// WindowBackdropDefault leaves the window background solid without translucent backdrop effects.
	WindowBackdropDefault WindowBackdrop = ""
	// WindowBackdropNone requests translucent mode with no blur material.
	WindowBackdropNone WindowBackdrop = "none"
	// WindowBackdropAuto lets Windows choose the default translucent material.
	WindowBackdropAuto WindowBackdrop = "auto"
	// WindowBackdropMica requests Windows 11 Mica backdrop material.
	WindowBackdropMica WindowBackdrop = "mica"
	// WindowBackdropAcrylic requests Windows Acrylic backdrop material.
	WindowBackdropAcrylic WindowBackdrop = "acrylic"
	// WindowBackdropTabbed requests Windows 11 Tabbed Mica backdrop material.
	WindowBackdropTabbed WindowBackdrop = "tabbed"
)

// WindowTemplate is host-only configuration for an approved same-origin route.
type WindowTemplate struct {
	Route       string
	Title       string
	Width       int
	Height      int
	MinWidth    int
	MinHeight   int
	MaxWidth    int
	MaxHeight   int
	Frameless   bool
	AlwaysOnTop bool
	Backdrop    WindowBackdrop
}
