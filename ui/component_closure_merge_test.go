// This file deliberately carries NO build tag so it runs on BOTH targets.
//
// It pins component identity when the toolchain merges or renames closures.
// Go 1.27 simplifies function-literal names and may merge function literals
// with identical bodies, so two literals from different call sites can report
// one code pointer and one qualified name. Identity must stay correct then:
// distinct captures render distinct output, and merged literals may share a
// logical component type only because their code is identical.

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// buildMergeProbeAlpha returns a component literal with the same body as
// buildMergeProbeBeta but a different call site.
//
//go:noinline
func buildMergeProbeAlpha(parseRt *runtime.Runtime, parseName string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		_ = ui.UseId()
		_, _ = runtime.GoUseState(parseRt, parseName)
		return html.Input(html.Props{Type: "text", Name: parseName})
	})
}

// buildMergeProbeBeta returns a component literal with the same body as
// buildMergeProbeAlpha but a different call site.
//
//go:noinline
func buildMergeProbeBeta(parseRt *runtime.Runtime, parseName string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		_ = ui.UseId()
		_, _ = runtime.GoUseState(parseRt, parseName)
		return html.Input(html.Props{Type: "text", Name: parseName})
	})
}

// TestComponentIdentityIdenticalLiteralsAtDistinctSitesRenderOwnCaptures proves
// that merging identical literals never leaks one site's captured props into the
// other, whatever names the toolchain assigns.
func TestComponentIdentityIdenticalLiteralsAtDistinctSitesRenderOwnCaptures(parseT *testing.T) {
	parseH := newIdentityHarness()
	parseH.rt.RenderInto(parseH.root, html.Form(html.Props{},
		buildMergeProbeAlpha(parseH.rt, "alpha-one"),
		buildMergeProbeBeta(parseH.rt, "beta-one"),
		buildMergeProbeAlpha(parseH.rt, "alpha-two"),
		buildMergeProbeBeta(parseH.rt, "beta-two"),
	))
	expectNames(parseT, "merge probe mount", parseH.names(), "alpha-one", "beta-one", "alpha-two", "beta-two")
}
