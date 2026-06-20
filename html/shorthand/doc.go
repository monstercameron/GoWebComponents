// Package shorthand provides a mixed-argument companion surface for html.
//
// The parent html package keeps the stable typed builders such as html.Div and
// html.Button. This package reuses the same option helpers and child
// normalization, but exposes host-tag functions that accept a single shared
// argument list of prop options and children:
//
//	return shorthand.Div(
//		shorthand.ClassStr("panel"),
//		shorthand.H2("Counter"),
//		shorthand.Button(shorthand.OnClick(increment), "Increment"),
//	)
//
// Strings and other text-like child values are normalized through html.Text,
// and option helpers keep the same last-write-wins semantics as html.PropsOf.
// The supported path for full-struct props is shorthand.FromProps(...), which
// preserves explicit zero values before later options are applied.
package shorthand
