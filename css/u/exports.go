package u

// This file re-exports the rest of the css surface under the u namespace, so a
// file that dot-imports u (alongside shorthand) never needs to write css. — the
// whole vocabulary is bare and Tailwind-flavored. Where u already defines a
// scale/token-flavored version of a name (Gap, Bg, Rounded, Border, Opacity, W,
// H, P/Pad…), that one wins and the raw css constructor is intentionally not
// re-exported (use the typed value form, e.g. GapV(Px(8)), or css. for the rare
// raw case).

import "github.com/monstercameron/GoWebComponents/css"

// --- types (aliases so signatures read bare) ---------------------------------

type (
	Rule      = css.Rule
	Sheet     = css.Sheet
	Color     = css.Color
	Length    = css.Length
	Duration  = css.Duration
	Angle     = css.Angle
	Number    = css.Number
	Selector  = css.Selector
	MediaQ    = css.MediaQuery
	Theme     = css.Theme
	Sink      = css.Sink
	Frame     = css.Frame
	NthArg    = css.NthArg
	Easing    = css.Easing
	TransProp = css.TransitionProperty
)

// --- entry points & registry --------------------------------------------------

var (
	Class         = css.Class
	New           = css.New
	Rules         = css.Rules
	DefineUtility = css.DefineUtility
	Utility       = css.Utility
	DefineVariant = css.DefineVariant
	UseTheme      = css.UseTheme
	DefaultTheme  = css.DefaultTheme
	Reset         = css.Reset
	Harvest       = css.Harvest // available on both build lanes
	Seed          = css.Seed
	SetSink       = css.SetSink
	MarkImportant = css.MarkImportant
	// SSR-only helpers (StyleBlock/HarvestedClasses) stay qualified as css.* —
	// they're native-build-only and a wasm app never calls them.
)

// --- value constructors -------------------------------------------------------

var (
	Px        = css.Px // length: Px(8) -> "8px"
	Rem       = css.Rem
	Ems       = css.Ems
	Percent   = css.Percent
	Vh        = css.Vh
	Vw        = css.Vw
	RawLength = css.RawLength
	Num       = css.Num
	Ms        = css.Ms
	S         = css.S
	Deg       = css.Deg
	Turn      = css.Turn
	Hex       = css.Hex
	RGB       = css.RGB
	RGBA      = css.RGBA
	CSSVar    = css.Var
	FontSize  = css.FontSize
	MinWidth  = css.MinWidth
	MaxWidth  = css.MaxWidth
	MinHeight = css.MinHeight
	MaxHeight = css.MaxHeight
)

const (
	Auto Length = css.Auto
	Full Length = css.Full
	Zero Length = css.Zero
)

// --- raw escape hatches -------------------------------------------------------

var (
	Raw = css.Raw
	Sel = css.Sel
)

// --- variants (the ones u doesn't already define) -----------------------------

var (
	Media        = css.Media
	MinW         = css.MinW
	MaxW         = css.MaxW
	RawMedia     = css.RawMedia
	FocusVisible = css.FocusVisible
	Before       = css.Before
	After        = css.After
	Keyframes    = css.Keyframes
	At           = css.At
	Animation    = css.Animation

	Child      = css.Child
	Descendant = css.Descendant
	Adjacent   = css.Adjacent
	Sibling    = css.Sibling
	Not        = css.Not
	Has        = css.Has
	Is         = css.Is
	NthChild   = css.NthChild
	NthOfType  = css.NthOfType
	AnB        = css.AnB
	Nth        = css.Nth
)

const (
	Odd  NthArg = css.Odd
	Even NthArg = css.Even
)

// --- selector targets ---------------------------------------------------------

var (
	El       = css.El
	SheetRef = css.SheetRef
	ClassSel = css.ClassSel
	AttrSel  = css.AttrSel
	AttrEq   = css.AttrEq
)

// --- typed property namespaces ------------------------------------------------

var (
	Cursor             = css.Cursor
	UserSelect         = css.UserSelect
	TextTransform      = css.TextTransform
	FontVariantNumeric = css.FontVariantNumeric
)

// --- raw property constructors (no scale-flavored u equivalent) ---------------

var (
	Shadow        = css.Shadow
	Transition    = css.Transition
	Prop          = css.Prop
	Transform     = css.Transform
	Scale         = css.Scale
	ScaleXY       = css.ScaleXY
	Rotate        = css.Rotate
	TranslateX    = css.TranslateX
	TranslateY    = css.TranslateY
	Outline       = css.Outline
	OutlineOffset = css.OutlineOffset
	Tracking      = css.Tracking
	LineHeight    = css.LineHeight
	LineHeightLen = css.LineHeightLen
	CubicBezier   = css.CubicBezier
)

const (
	ShadowNone = css.ShadowNone
	ShadowSm   = css.ShadowSm
	ShadowMd   = css.ShadowMd
	ShadowLg   = css.ShadowLg
	ShadowXl   = css.ShadowXl
	Shadow2xl  = css.Shadow2xl

	PropAll       = css.PropAll
	PropColors    = css.PropColors
	PropOpacity   = css.PropOpacity
	PropTransform = css.PropTransform
	PropShadow    = css.PropShadow

	Linear    = css.Linear
	Ease      = css.Ease
	EaseIn    = css.EaseIn
	EaseOut   = css.EaseOut
	EaseInOut = css.EaseInOut
)

// --- the rest of the color palette (u already exports a curated subset) -------

const (
	Transparent  Color = css.Transparent
	CurrentColor Color = css.CurrentCo
	Slate50      Color = css.Slate50
	Slate200     Color = css.Slate200
	Slate300     Color = css.Slate300
	Slate400     Color = css.Slate400
	Slate600     Color = css.Slate600
	Slate700     Color = css.Slate700
	Sky400       Color = css.Sky400
	Sky600       Color = css.Sky600
	Red600       Color = css.Red600
	Green500     Color = css.Green500
	Amber500     Color = css.Amber500
)
