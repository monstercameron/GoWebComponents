package css

import (
	"sort"
	"strconv"
	"sync"
)

// Theme is the typed analog of tailwind.config.js: the named scales the Layer-2
// utility engine resolves against. Swap or extend it with UseTheme; utilities and
// responsive variants read the active theme.
type Theme struct {
	// Spacing maps a scale index (Tailwind's 0,1,2,3,…) to a Length. The default
	// follows Tailwind: index n -> n*0.25rem.
	Spacing map[int]Length
	// Colors maps a token name ("slate-900") to a Color.
	Colors map[string]Color
	// FontSizes maps a type-scale name ("sm","base","lg") to a Length.
	FontSizes map[string]Length
	// Breakpoints maps a responsive name ("sm","md","lg","xl","2xl") to its
	// min-width Length and drives the responsive variants.
	Breakpoints map[string]Length
	// Radii maps a rounding name ("sm","md","lg","full") to a Length.
	Radii map[string]Length
}

// DefaultTheme returns Tailwind's default scales (the curated v1 subset).
func DefaultTheme() Theme {
	spacing := map[int]Length{}
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24, 32, 40, 48, 56, 64, 80, 96} {
		if n == 0 {
			spacing[0] = Zero
			continue
		}
		spacing[n] = Rem(float64(n) * 0.25)
	}
	return Theme{
		Spacing: spacing,
		Colors: map[string]Color{
			"transparent": Transparent,
			"current":     CurrentCo,
			"white":       White,
			"black":       Black,
			"slate-50":    Slate50,
			"slate-100":   Slate100,
			"slate-200":   Slate200,
			"slate-300":   Slate300,
			"slate-400":   Slate400,
			"slate-500":   Slate500,
			"slate-600":   Slate600,
			"slate-700":   Slate700,
			"slate-800":   Slate800,
			"slate-900":   Slate900,
			"sky-400":     Sky400,
			"sky-500":     Sky500,
			"sky-600":     Sky600,
			"red-500":     Red500,
			"red-600":     Red600,
			"green-500":   Green500,
			"amber-500":   Amber500,
		},
		FontSizes: map[string]Length{
			"xs":   Rem(0.75),
			"sm":   Rem(0.875),
			"base": Rem(1),
			"lg":   Rem(1.125),
			"xl":   Rem(1.25),
			"2xl":  Rem(1.5),
			"3xl":  Rem(1.875),
			"4xl":  Rem(2.25),
		},
		Breakpoints: map[string]Length{
			"sm":  Px(640),
			"md":  Px(768),
			"lg":  Px(1024),
			"xl":  Px(1280),
			"2xl": Px(1536),
		},
		Radii: map[string]Length{
			"none": Zero,
			"sm":   Rem(0.125),
			"md":   Rem(0.375),
			"lg":   Rem(0.5),
			"xl":   Rem(0.75),
			"full": Length("9999px"),
		},
	}
}

var (
	themeMu     sync.RWMutex
	activeTheme = DefaultTheme()
)

// RootRules returns the theme's scales as typed custom-property declarations
// (CSS2): --color-<name>, --space-<index>, --text-<name>, --radius-<name>. It is
// the bridge between the typed Theme (which the utility engine resolves at
// class-generation time) and a live CSS-variable palette: emit these into :root
// and reference them with Var, then a runtime element.style.setProperty reskins
// every reference without regenerating classes.
//
//	css.Root(css.DefaultTheme().RootRules()...)               // :root palette
//	css.New(css.DataTheme("light", lightTheme.RootRules()...)) // a scoped theme
//
// Breakpoints are intentionally omitted — they drive @media queries, where
// var() is not reliably supported.
func (parseTheme Theme) RootRules() []Rule {
	parseRules := make([]Rule, 0,
		len(parseTheme.Colors)+len(parseTheme.Spacing)+len(parseTheme.FontSizes)+len(parseTheme.Radii))

	parseColorNames := make([]string, 0, len(parseTheme.Colors))
	for parseName := range parseTheme.Colors {
		parseColorNames = append(parseColorNames, parseName)
	}
	sort.Strings(parseColorNames)
	for _, parseName := range parseColorNames {
		parseRules = append(parseRules, Raw("--color-"+parseName, string(parseTheme.Colors[parseName])))
	}

	parseSpacingKeys := make([]int, 0, len(parseTheme.Spacing))
	for parseIndex := range parseTheme.Spacing {
		parseSpacingKeys = append(parseSpacingKeys, parseIndex)
	}
	sort.Ints(parseSpacingKeys)
	for _, parseIndex := range parseSpacingKeys {
		parseRules = append(parseRules, Raw("--space-"+strconv.Itoa(parseIndex), string(parseTheme.Spacing[parseIndex])))
	}

	parseFontNames := make([]string, 0, len(parseTheme.FontSizes))
	for parseName := range parseTheme.FontSizes {
		parseFontNames = append(parseFontNames, parseName)
	}
	sort.Strings(parseFontNames)
	for _, parseName := range parseFontNames {
		parseRules = append(parseRules, Raw("--text-"+parseName, string(parseTheme.FontSizes[parseName])))
	}

	parseRadiusNames := make([]string, 0, len(parseTheme.Radii))
	for parseName := range parseTheme.Radii {
		parseRadiusNames = append(parseRadiusNames, parseName)
	}
	sort.Strings(parseRadiusNames)
	for _, parseName := range parseRadiusNames {
		parseRules = append(parseRules, Raw("--radius-"+parseName, string(parseTheme.Radii[parseName])))
	}

	return parseRules
}

// EmitThemeTokens emits the theme's RootRules into :root in one call — the
// convenience form of css.Root(theme.RootRules()...).
func EmitThemeTokens(parseTheme Theme) {
	if parseRules := parseTheme.RootRules(); len(parseRules) > 0 {
		Root(parseRules...)
	}
}

// UseTheme swaps the active theme that the utility engine resolves against.
func UseTheme(parseTheme Theme) {
	themeMu.Lock()
	defer themeMu.Unlock()
	activeTheme = parseTheme
}

// ActiveTheme returns a snapshot reference to the active theme.
func ActiveTheme() Theme {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return activeTheme
}

// SpacingValue resolves a spacing-scale index against the active theme, falling
// back to Tailwind's n*0.25rem formula for indices outside the curated map.
func SpacingValue(parseIndex int) Length {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if v, ok := activeTheme.Spacing[parseIndex]; ok {
		return v
	}
	if parseIndex == 0 {
		return Zero
	}
	return Rem(float64(parseIndex) * 0.25)
}

// ColorValue resolves a color token name against the active theme; ok reports
// whether the token exists.
func ColorValue(parseName string) (Color, bool) {
	themeMu.RLock()
	defer themeMu.RUnlock()
	c, ok := activeTheme.Colors[parseName]
	return c, ok
}

// FontSizeValue resolves a type-scale name against the active theme.
func FontSizeValue(parseName string) (Length, bool) {
	themeMu.RLock()
	defer themeMu.RUnlock()
	v, ok := activeTheme.FontSizes[parseName]
	return v, ok
}

// BreakpointValue resolves a responsive breakpoint name to its min-width.
func BreakpointValue(parseName string) (Length, bool) {
	themeMu.RLock()
	defer themeMu.RUnlock()
	v, ok := activeTheme.Breakpoints[parseName]
	return v, ok
}

// RadiusValue resolves a rounding name against the active theme.
func RadiusValue(parseName string) (Length, bool) {
	themeMu.RLock()
	defer themeMu.RUnlock()
	v, ok := activeTheme.Radii[parseName]
	return v, ok
}
