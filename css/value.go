package css

import (
	"math"
	"strconv"
	"strings"
)

// Length is a CSS length/size value that carries its unit in the value text, not
// the type. Build one with Px, Rem, Em, Percent, or the Raw escape hatch.
type Length string

func (l Length) String() string { return string(l) }

// Px is an integer pixel length: Px(8) -> "8px".
func Px(n int) Length { return Length(strconv.Itoa(n) + "px") }

// Rem is a rem length: Rem(0.5) -> "0.5rem".
func Rem(n float64) Length { return Length(trimFloat(n) + "rem") }

// Ems is an em length: Ems(1.5) -> "1.5em".
func Ems(n float64) Length { return Length(trimFloat(n) + "em") }

// Percent is a percentage length: Percent(50) -> "50%".
func Percent(n float64) Length { return Length(trimFloat(n) + "%") }

// Vh / Vw are viewport-relative lengths.
func Vh(n float64) Length { return Length(trimFloat(n) + "vh") }
func Vw(n float64) Length { return Length(trimFloat(n) + "vw") }

// Ch is a character-width length: Ch(68) -> "68ch". One ch is the advance width of
// the "0" glyph in the element's own font, so a ch max-width tracks the measure of the
// text (the ~45-75 character line that reads comfortably) instead of a pixel guess
// that drifts every time the font size changes.
func Ch(n float64) Length { return Length(trimFloat(n) + "ch") }

// Clamp builds clamp(min, preferred, max) — one fluid value instead of a media-query
// staircase. The preferred term is normally viewport-relative (Vw) so it scales, with
// min/max as the hard stops:
//
//	Clamp(Rem(1), Vw(2.5), Rem(1.5)) -> clamp(1rem,2.5vw,1.5rem)
//
// Keep min <= max: an inverted pair is legal CSS but resolves to min, silently
// ignoring the preferred value.
func Clamp(min, preferred, max Length) Length {
	return Length("clamp(" + string(min) + "," + string(preferred) + "," + string(max) + ")")
}

// MinLen / MaxLen build min()/max() over a list of lengths. The commas are function
// arguments, not a declaration-level list, so they are safe wherever a Length goes.
func MinLen(values ...Length) Length { return lengthFn("min", values) }
func MaxLen(values ...Length) Length { return lengthFn("max", values) }

func lengthFn(parseName string, values []Length) Length {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		parts = append(parts, string(v))
	}
	if len(parts) == 0 {
		return Zero
	}
	if len(parts) == 1 {
		return Length(parts[0])
	}
	return Length(parseName + "(" + strings.Join(parts, ",") + ")")
}

// Auto is the keyword length "auto".
const Auto Length = "auto"

// Full is "100%".
const Full Length = "100%"

// Zero is "0".
const Zero Length = "0"

// RawLength is the escape hatch for any length string not covered by a unit
// constructor (e.g. "calc(100% - 8px)", "min(10px, 1rem)").
func RawLength(parseValue string) Length { return Length(parseValue) }

// Duration is a CSS time value (transition/animation timing). Build with Ms or S.
type Duration string

func (d Duration) String() string { return string(d) }

// Ms is a millisecond duration: Ms(120) -> "120ms".
func Ms(n int) Duration { return Duration(strconv.Itoa(n) + "ms") }

// S is a second duration: S(0.2) -> "0.2s".
func S(n float64) Duration { return Duration(trimFloat(n) + "s") }

// RawDuration is the escape hatch for a time value the unit constructors cannot
// express — notably the sub-millisecond "0.01ms" used to neutralize a transition
// under prefers-reduced-motion. 0.01ms rather than 0ms is deliberate there: it still
// fires transitionend, so code awaiting that event does not hang for exactly the
// users who asked for less motion.
func RawDuration(parseValue string) Duration { return Duration(parseValue) }

// Angle is a CSS angle value. Build with Deg or Turn.
type Angle string

func (a Angle) String() string { return string(a) }

// Deg is a degree angle: Deg(45) -> "45deg".
func Deg(n float64) Angle { return Angle(trimFloat(n) + "deg") }

// Turn is a turn angle: Turn(0.5) -> "0.5turn".
func Turn(n float64) Angle { return Angle(trimFloat(n) + "turn") }

// Number is a unitless CSS number (line-height, flex-grow, opacity factors,
// scale factors). Build with Num.
type Number string

func (n Number) String() string { return string(n) }

// Num is a unitless number: Num(1.5) -> "1.5".
func Num(n float64) Number { return Number(trimFloat(n)) }

// Color is a CSS color value. Use the curated token constants, Hex, or RGB/RGBA.
type Color string

func (c Color) String() string { return string(c) }

// Hex builds a color from a hex string, tolerating a missing leading '#':
// Hex("0af") and Hex("#0af") both -> "#0af". As a typed constructor it is
// safe-by-construction: any non-hex characters are dropped, so an untrusted string
// can never inject CSS or break out (e.g. Hex("</style>") -> "#e"). An empty/all-
// invalid input falls back to "#000000".
func Hex(parseValue string) Color {
	var b strings.Builder
	for i := 0; i < len(parseValue); i++ {
		c := parseValue[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			b.WriteByte(c)
			if b.Len() == 8 { // cap at #RRGGBBAA
				break
			}
		}
	}
	// Only 3/4/6/8-digit forms are legal CSS hex colors; anything else (e.g.
	// Hex("12345")) the browser silently drops. Fall back to black rather than
	// emit a dead value.
	switch b.Len() {
	case 3, 4, 6, 8:
		return Color("#" + b.String())
	default:
		return Color("#000000")
	}
}

// clampChannel clamps an RGB channel to the valid [0,255] range.
func clampChannel(parseV int) int {
	if parseV < 0 {
		return 0
	}
	if parseV > 255 {
		return 255
	}
	return parseV
}

// RGB builds an rgb() color; channels are clamped to [0,255] (matching how RGBA
// clamps alpha) so the formatter emits canonical, in-range output.
func RGB(r, g, b int) Color {
	return Color("rgb(" + strconv.Itoa(clampChannel(r)) + "," + strconv.Itoa(clampChannel(g)) + "," + strconv.Itoa(clampChannel(b)) + ")")
}

// RGBA builds an rgba() color; alpha is clamped to [0,1].
func RGBA(r, g, b int, a float64) Color {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return Color("rgba(" + strconv.Itoa(clampChannel(r)) + "," + strconv.Itoa(clampChannel(g)) + "," + strconv.Itoa(clampChannel(b)) + "," + trimFloat(a) + ")")
}

// ColorMix blends two colors: ColorMix(Sky500, White, 20) is "20% Sky500, the rest
// White". pct is the share of the FIRST color and is clamped to [0,100].
//
// Mixing happens in oklab, not sRGB, because sRGB interpolation runs through a muddy
// desaturated midpoint (the classic grey band halfway between blue and yellow) while
// oklab keeps perceived lightness and chroma steady. This is how a design system
// derives a hover tint or a disabled tone from ONE token instead of shipping a whole
// second palette — and because the mix resolves at paint time, it composes with
// var() tokens a runtime theme engine rewrites.
func ColorMix(a, b Color, pct float64) Color {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return Color("color-mix(in oklab," + string(a) + " " + trimFloat(pct) + "%," + string(b) + ")")
}

// Var references a CSS custom property, the bridge for runtime/dynamic values:
// Var("--accent") -> "var(--accent)". As a typed constructor it sanitizes the name
// to CSS identifier characters (letters, digits, '-', '_'), so an untrusted name
// can never close the var() call or inject CSS.
func Var(parseName string) Color {
	return Color(varExpr(parseName))
}

// VarLength / VarDuration / VarAngle / VarNumber are the typed siblings of Var for custom
// properties used in non-color positions, so a CSS variable flows into W/FontSize/Gap (Length),
// Transition/Animation (Duration), Rotate (Angle), or LineHeight/Opacity (Number) with full type
// safety instead of falling through to Raw(...). Same sanitizing as Var.
func VarLength(parseName string) Length     { return Length(varExpr(parseName)) }
func VarDuration(parseName string) Duration { return Duration(varExpr(parseName)) }
func VarAngle(parseName string) Angle       { return Angle(varExpr(parseName)) }
func VarNumber(parseName string) Number     { return Number(varExpr(parseName)) }

// varExpr sanitizes a custom-property name to CSS identifier characters and wraps it in var(),
// so an untrusted name can never close the var() call or inject CSS. Shared by every Var* form.
//
// It normalizes the name through customPropertyName — the same helper the Custom*
// declaration constructors use — so a declaration and its reference can never
// disagree about the spelling of a token ("accent", "--accent" and "-accent" all name
// the single property --accent).
func varExpr(parseName string) string {
	return "var(" + customPropertyName(parseName) + ")"
}

// Curated v1 color tokens (a Tailwind-shaped slice of the default palette).
const (
	Transparent Color = "transparent"
	CurrentCo   Color = "currentColor"
	White       Color = "#ffffff"
	Black       Color = "#000000"

	Slate50  Color = "#f8fafc"
	Slate100 Color = "#f1f5f9"
	Slate200 Color = "#e2e8f0"
	Slate300 Color = "#cbd5e1"
	Slate400 Color = "#94a3b8"
	Slate500 Color = "#64748b"
	Slate600 Color = "#475569"
	Slate700 Color = "#334155"
	Slate800 Color = "#1e293b"
	Slate900 Color = "#0f172a"

	Sky400 Color = "#38bdf8"
	Sky500 Color = "#0ea5e9"
	Sky600 Color = "#0284c7"

	Red500   Color = "#ef4444"
	Red600   Color = "#dc2626"
	Green500 Color = "#22c55e"
	Amber500 Color = "#f59e0b"
)

// trimFloat formats a float without a trailing ".0" or trailing zeros, so the
// canonical serialization stays stable and compact. Non-finite inputs (NaN, ±Inf)
// would serialize to "NaN"/"+Inf" — invalid CSS — so they collapse to "0".
func trimFloat(n float64) string {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return "0"
	}
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
