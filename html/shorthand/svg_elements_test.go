package shorthand_test

import (
	"strings"
	"testing"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// TestSVGChartElementsRender proves the chart-primitive SVG helpers (G8) render
// their elements through SSR — enabling native-Go charting without a JS shim.
func TestSVGChartElementsRender(parseT *testing.T) {
	parseNode := Svg(
		Attr("viewBox", "0 0 100 100"),
		Defs(
			LinearGradient(Attr("id", "grad"),
				GradientStop(Attr("offset", "0%")),
				GradientStop(Attr("offset", "100%")),
			),
			SvgPattern(Attr("id", "pat")),
		),
		G(
			Ellipse(Attr("cx", "50"), Attr("cy", "50"), Attr("rx", "40"), Attr("ry", "20")),
			Line(Attr("x1", "0"), Attr("y1", "0"), Attr("x2", "100"), Attr("y2", "100")),
			Polyline(Attr("points", "0,0 10,10")),
			Polygon(Attr("points", "0,0 10,10 0,10")),
			TSpan("label"),
		),
		RadialGradient(Attr("id", "rg")),
		ClipPath(Attr("id", "clip")),
		Mask(Attr("id", "m")),
		Marker(Attr("id", "mk")),
		Symbol(Attr("id", "sym")),
		ForeignObject(Attr("width", "10")),
		SvgImage(Attr("href", "x.png")),
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	for _, parseTag := range []string{
		"<ellipse", "<line", "<polyline", "<polygon", "<tspan",
		"<lineargradient", "<radialgradient", "<stop", "<clippath",
		"<mask", "<pattern", "<marker", "<symbol", "<foreignobject", "<image",
	} {
		if !strings.Contains(strings.ToLower(parseMarkup), parseTag) {
			parseT.Fatalf("expected %q in SVG markup:\n%s", parseTag, parseMarkup)
		}
	}
}
