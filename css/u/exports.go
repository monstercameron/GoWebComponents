package u

// This file re-exports the rest of the css surface under the u namespace, so a
// file that dot-imports u (alongside shorthand) never needs to write css. — the
// whole vocabulary is bare and Tailwind-flavored. Where u already defines a
// scale/token-flavored version of a name (Gap, Bg, Rounded, Border, Opacity, W,
// H, P/Pad…), that one wins and the raw css constructor is intentionally not
// re-exported (use the typed value form, e.g. GapV(Px(8)), or css. for the rare
// raw case).

import "github.com/monstercameron/GoWebComponents/v5/css"

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

	// Track is intentionally NOT re-exported: html/shorthand.Track is the <track>
	// element, and this file's whole premise is that u dot-imports cleanly alongside
	// shorthand. Use css.Track for the grid-track type.
	Placement   = css.GridPlacement
	Image       = css.Image
	ColorStop   = css.ColorStop
	FontStack   = css.FontStack
	LineStyle   = css.LineStyle
	ShadowToken = css.ShadowToken
	BgSizeValue = css.BgSizeValue
	Shape       = css.GradientShape
	Direction   = css.GradientDirection
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

	Ch       = css.Ch       // character-width length: Ch(68) -> "68ch"
	Clamp    = css.Clamp    // clamp(min, preferred, max)
	MinLen   = css.MinLen   // min(a, b, …)
	MaxLen   = css.MaxLen   // max(a, b, …)
	ColorMix = css.ColorMix // color-mix(in oklab, a pct%, b)
)

const (
	Auto Length = css.Auto
	Full Length = css.Full
	Zero Length = css.Zero
)

// --- raw escape hatches -------------------------------------------------------

// Raw and Sel are the documented escape hatches for the typed utility surface. The curated
// utilities cover the common surface, not every CSS property; when none fits, drop to a raw
// declaration rather than reaching outside the engine. Raw(property, value) emits one declaration
// and Sel(selector, rules...) emits an arbitrary selector — both fold, dedup, content-hash, and
// emit through the SAME Layer-1 registry/sink as every typed utility, so a raw style still gets a
// stable class name and SSR reuse (it is not an unmanaged inline style). Reach for these only for
// the long tail; prefer a typed utility whenever one exists so the style stays autocompletable
// and compile-checked.
//
//	// no typed aspect-ratio utility → use the raw escape hatch, still registry-managed:
//	css.New(u.Pad(u.Spacing4), u.Raw("aspect-ratio", "16 / 9"))
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

	// Box & scrolling.
	Overflow            = css.Overflow
	OverflowX           = css.OverflowX
	OverflowY           = css.OverflowY
	OverscrollBehavior  = css.OverscrollBehavior
	OverscrollBehaviorX = css.OverscrollBehaviorX
	OverscrollBehaviorY = css.OverscrollBehaviorY
	PointerEvents       = css.PointerEvents
	Appearance          = css.Appearance
	ColorScheme         = css.ColorScheme

	// Flex / grid item alignment.
	FlexWrap     = css.FlexWrap
	AlignSelf    = css.AlignSelf
	JustifySelf  = css.JustifySelf
	AlignContent = css.AlignContent
	JustifyItems = css.JustifyItems

	// Borders & tables.
	BorderStyle    = css.BorderStyle
	BorderCollapse = css.BorderCollapse
	TableLayout    = css.TableLayout

	// Typography.
	TextAlign            = css.TextAlign
	VerticalAlign        = css.VerticalAlign
	WhiteSpace           = css.WhiteSpace
	TextWrap             = css.TextWrap
	OverflowWrap         = css.OverflowWrap
	TextDecoration       = css.TextDecoration
	TextDecorationStyle  = css.TextDecorationStyle
	FontStyle            = css.FontStyle
	FontVariantLigatures = css.FontVariantLigatures

	// Backgrounds.
	BgRepeat   = css.BgRepeat
	BgClip     = css.BgClip
	BgOrigin   = css.BgOrigin
	MaskRepeat = css.MaskRepeat
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

	TransitionProps     = css.TransitionProps
	TransitionLonghands = css.TransitionLonghands
	TransitionDuration  = css.TransitionDuration
	RawShadow           = css.RawShadow
	Shadows             = css.Shadows
	ShadowOf            = css.ShadowOf
	ShadowInset         = css.ShadowInset
	OutlineStyle        = css.OutlineStyle

	// Side-specific borders and corner radii. (u.Border(c) stays the 1px-solid
	// scale-flavored form; these are the Layer-1 width+color constructors.)
	BorderTop     = css.BorderTop
	BorderRight   = css.BorderRight
	BorderBottom  = css.BorderBottom
	BorderLeft    = css.BorderLeft
	BorderX       = css.BorderX
	BorderY       = css.BorderY
	BorderColor   = css.BorderColor
	BorderWidth   = css.BorderWidth
	BorderSpacing = css.BorderSpacing
	// RoundedTopV/… carry the V suffix because u.Rounded already takes a theme
	// Radius token; these take a Layer-1 Length, matching the GapV convention.
	RoundedTopV    = css.RoundedTop
	RoundedBottomV = css.RoundedBottom
	RoundedLeftV   = css.RoundedLeft
	RoundedRightV  = css.RoundedRight

	// Inset offsets & stacking.
	Top              = css.Top
	Right            = css.Right
	Bottom           = css.Bottom
	Left             = css.Left
	Inset            = css.Inset
	InsetX           = css.InsetX
	InsetY           = css.InsetY
	ZIndex           = css.ZIndex
	ScrollPadding    = css.ScrollPadding
	ScrollPaddingTop = css.ScrollPaddingTop

	// Flex item longhands. (u.Flex is Display.Flex, so the css.Flex shorthand is
	// re-exported as FlexOf to keep both reachable.)
	FlexOf     = css.Flex
	FlexGrow   = css.FlexGrow
	FlexShrink = css.FlexShrink
	FlexBasis  = css.FlexBasis
	Order      = css.Order
	RowGap     = css.RowGap
	ColumnGap  = css.ColumnGap

	// Grid.
	GridCols        = css.GridCols
	GridRows        = css.GridRows
	GridAutoRows    = css.GridAutoRows
	GridAutoColumns = css.GridAutoColumns
	GridAreas       = css.GridAreas
	GridArea        = css.GridArea
	GridColumn      = css.GridColumn
	GridRow         = css.GridRow
	Fr              = css.Fr
	TrackLen        = css.TrackLen
	MinMax          = css.MinMax
	FitContent      = css.FitContent
	// Repeat: shadowed by html/shorthand.Repeat (node repetition) — use css.Repeat.
	RepeatFit       = css.RepeatFit
	RepeatFill      = css.RepeatFill
	GridLineAt      = css.GridLineAt
	GridSpan        = css.GridSpan
	GridLineName    = css.GridLineName
	GridRange       = css.GridRange

	// Backgrounds & gradients.
	BgImage                 = css.BgImage
	BgSize                  = css.BgSize
	BgSizeXY                = css.BgSizeXY
	BgPosition              = css.BgPosition
	// LinearGradient / RadialGradient: shadowed by the html/shorthand SVG elements of
	// the same name — use css.LinearGradient / css.RadialGradient. The *To and
	// Repeating* forms below carry no such clash and stay bare.
	LinearGradientTo        = css.LinearGradientTo
	RepeatingLinearGradient = css.RepeatingLinearGradient
	RepeatingRadialGradient = css.RepeatingRadialGradient
	ConicGradient           = css.ConicGradient
	CircleAt                = css.CircleAt
	EllipseAt               = css.EllipseAt
	CircleSizedAt           = css.CircleSizedAt
	URLImage                = css.URLImage
	RawImage                = css.RawImage
	// Stop: shadowed by html/shorthand.Stop (the SVG <stop>) — use css.Stop.
	StopAt                  = css.StopAt
	StopSpan                = css.StopSpan
	ColorHint               = css.ColorHint
	MaskImage               = css.MaskImage
	MaskSize                = css.MaskSize

	// Typography constructors.
	Font                    = css.Font
	FontStackOf             = css.FontStackOf
	VarFontStack            = css.VarFontStack
	RawFontStack            = css.RawFontStack
	TextDecorationThickness = css.TextDecorationThickness
	TextUnderlineOffset     = css.TextUnderlineOffset
	TextDecorationColor     = css.TextDecorationColor
	TextOverflowEllipsis    = css.TextOverflowEllipsis
	TextIndent              = css.TextIndent
	WordSpacing             = css.WordSpacing

	// Custom-property declarations (the typed replacement for Raw("--x", …)).
	Custom          = css.Custom
	CustomColor     = css.CustomColor
	CustomLength    = css.CustomLength
	CustomNumber    = css.CustomNumber
	CustomDuration  = css.CustomDuration
	CustomAngle     = css.CustomAngle
	CustomFontStack = css.CustomFontStack
	CustomShadow    = css.CustomShadow
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

	LineSolid  = css.LineSolid
	LineDashed = css.LineDashed
	LineDotted = css.LineDotted
	LineDouble = css.LineDouble
	LineNone   = css.LineNone
	LineHidden = css.LineHidden

	TrackAuto       = css.TrackAuto
	TrackMinContent = css.TrackMinContent
	TrackMaxContent = css.TrackMaxContent
	GridAuto        = css.GridAuto

	NoImage       = css.NoImage
	// Circle / Ellipse: shadowed by the html/shorthand SVG elements — use css.Circle /
	// css.Ellipse. CircleAt / EllipseAt above are unambiguous and stay bare.
	ToTop         = css.ToTop
	ToBottom      = css.ToBottom
	ToLeft        = css.ToLeft
	ToRight       = css.ToRight
	ToTopLeft     = css.ToTopLeft
	ToTopRight    = css.ToTopRight
	ToBottomLeft  = css.ToBottomLeft
	ToBottomRight = css.ToBottomRight
	BgCover       = css.BgCover
	BgContain     = css.BgContain
	BgAuto        = css.BgAuto

	SansStack  = css.SansStack
	SerifStack = css.SerifStack
	MonoStack  = css.MonoStack
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
