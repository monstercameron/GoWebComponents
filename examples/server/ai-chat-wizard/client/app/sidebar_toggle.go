//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func parseSidebarToggleIcon(isCollapsed bool) ui.Node {
	return Div(
		ClassStr("relative h-4 w-4 shrink-0 overflow-hidden"),
		Span(ClassStr(ClassNames(
			"absolute top-[2px] h-[12px] w-px rounded-full bg-current transition-all duration-200 ease-out",
			When(isCollapsed, "left-[9px] opacity-20"),
			When(!isCollapsed, "left-[3px] opacity-35"),
		))),
		Span(ClassStr(ClassNames(
			"absolute top-1/2 h-2 w-2 -translate-y-1/2 border-t border-r border-current transition-all duration-200 ease-out",
			When(isCollapsed, "left-[5px] rotate-45"),
			When(!isCollapsed, "left-[7px] rotate-[225deg]"),
		))),
		Span(ClassStr(ClassNames(
			"absolute top-1/2 h-[6px] w-[6px] -translate-y-1/2 rounded-full border border-current transition-all duration-200 ease-out",
			When(isCollapsed, "left-[1px] opacity-55 scale-100"),
			When(!isCollapsed, "left-[11px] opacity-0 scale-50"),
		))),
	)
}
