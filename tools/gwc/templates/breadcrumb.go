package components

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// Crumb is one breadcrumb entry.
type Crumb struct {
	// Label is the visible text.
	Label string
	// Href is the link target; the last crumb usually has an empty Href (current page).
	Href string
}

// BreadcrumbProps configures a Breadcrumb.
type BreadcrumbProps struct {
	// Items are the trail, root first.
	Items []Crumb
}

// Breadcrumb is an accessible navigation trail implementing the WAI-ARIA breadcrumb pattern:
// a <nav aria-label="Breadcrumb"> wrapping an ordered list of links, with the final entry
// marked aria-current="page" and rendered as plain text rather than a link. Copied into your
// repo by `gwc add breadcrumb`; restyle via the gwc-breadcrumb classes.
func Breadcrumb(parseProps BreadcrumbProps) ui.Node {
	parseItems := make([]ui.Node, 0, len(parseProps.Items))
	for parseIndex, parseCrumb := range parseProps.Items {
		parseIsLast := parseIndex == len(parseProps.Items)-1

		var parseInner ui.Node
		if parseIsLast || parseCrumb.Href == "" {
			parseInner = html.Tag("span", html.Props{
				Class: "gwc-breadcrumb-current",
				Aria:  map[string]string{"current": "page"},
			}, html.Text(parseCrumb.Label))
		} else {
			parseInner = html.A(html.Props{Class: "gwc-breadcrumb-link", Href: parseCrumb.Href}, html.Text(parseCrumb.Label))
		}

		parseItems = append(parseItems, html.Tag("li", html.Props{Class: "gwc-breadcrumb-item"}, parseInner))
	}

	return html.Nav(html.Props{
		Class: "gwc-breadcrumb",
		Aria:  map[string]string{"label": "Breadcrumb"},
	}, html.Tag("ol", html.Props{Class: "gwc-breadcrumb-list"}, parseItems...))
}
