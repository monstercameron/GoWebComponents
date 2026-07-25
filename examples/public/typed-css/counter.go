//go:build js && wasm

package main

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/v5/css/u"          // styling: bare
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand" // elements: bare
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Hoisted shared button base — folded once. ZERO css./u. qualifiers: everything
// is bare via the two dot-imports.
var buttonBase = Rules(
	InlineFlex, ItemsCenter, JustifyCenter,
	PadX(Spacing5), PadY(Spacing2),
	Rounded(RadiusLg), FontSemibold, Fg(White),
	MinWidth(Px(64)),
	Cursor.Pointer, UserSelect.None,
	Transition(PropAll, Ms(120), Ease),
	Active(Transform(Scale(0.95))),
	FocusVisible(Outline(Px(2), Sky400), OutlineOffset(Px(2))),
)

// Counter — a styled click counter, fully bare (no css. / no u.).
func Counter() ui.Node {
	count := ui.UseState(0)

	return Div(
		Class(
			Flex, FlexCol, ItemsCenter,
			Gap(Spacing5), Pad(Spacing8),
			Rounded(RadiusXl), Bg(Slate900), Border(Slate700),
			MinWidth(Px(260)), Shadow(Shadow2xl),
			// style our own child <span> label via selector composition
			Child(El("span"), Fg(Slate400), FontMedium, TextTransform.Uppercase, Tracking(Ems(0.18))),
			Md(Pad(Spacing10)), // responsive
			Dark(Bg(Black)),    // dark mode
		),
		FromProps(Props{ID: "card"}),

		Span(Class(TextSize(TextSm)), FromProps(Props{ID: "label"}), "Clicks"),

		Div(Class(FontSize(Rem(3)), FontBold, Fg(White),
			FontVariantNumeric.TabularNums, LineHeight(Num(1))),
			strconv.Itoa(count.Get()),
		),

		Div(Class(Flex, Gap(Spacing3)),
			Button(
				Class(buttonBase, Bg(Sky500), Hover(Bg(Sky600))),
				FromProps(Props{ID: "inc"}),
				Props{Type: "button", OnClick: ui.UseEvent(func() { count.Set(count.Get() + 1) })},
				"+1",
			),
			Button(
				Class(buttonBase, Bg(Slate700), Hover(Bg(Slate600))),
				Props{Type: "button", OnClick: ui.UseEvent(func() { count.Set(0) })},
				"Reset",
			),
		),
	)
}
