package css_test

import "github.com/monstercameron/GoWebComponents/v5/css"

// The css sink API must be identical on native and wasm builds so portable app
// code compiles on both targets. This file is build-tag-free: referencing each
// function with its exact signature forces BOTH targets to expose it, so a
// target-specific regression (e.g. StyleBlock/HarvestedClasses native-only or
// SeedFromDocument wasm-only, the #42 gap) fails to compile here.
var (
	_ func(string, string) = css.Inject
	_ func()               = css.Reset
	_ func() string        = css.Harvest
	_ func() []string      = css.HarvestedClasses
	_ func() string        = css.CriticalCSS
	_ func() string        = css.StyleBlock
	_ func()               = css.SeedFromDocument
)
