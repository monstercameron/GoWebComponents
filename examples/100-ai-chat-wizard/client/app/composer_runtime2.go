//go:build js && wasm

package app

import (
	"sync"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	renderComposerCostRendererID = "chat-wizard.composer-costs"
	renderComposerCostRegionID   = "chat-wizard.composer-costs.primary"
)

type renderComposerCostRegionProps struct {
	HasThreadCost  bool
	RenderThread   string
	HasAccountCost bool
	RenderAccount  string
}

var storeRuntime2RegionRegistrationOnce sync.Once

// parseRegisterRuntime2Regions registers example 100 display-only runtime2 region renderers.
func parseRegisterRuntime2Regions() {
	storeRuntime2RegionRegistrationOnce.Do(func() {
		parseErr := ui.RegisterParallelRegion(renderComposerCostRendererID, renderComposerCostRegion)
		if parseErr != nil {
			panic(parseErr)
		}
	})
}

// parseBuildComposerCostRegionProps formats display-only composer cost labels for runtime2 props.
func parseBuildComposerCostRegionProps(parseComposerProps composerProps) renderComposerCostRegionProps {
	parseRegionProps := renderComposerCostRegionProps{
		HasThreadCost:  parseComposerProps.ThreadCostSummary.HasAnyExactCosts && parseComposerProps.ThreadCostSummary.TotalCost > 0,
		HasAccountCost: parseComposerProps.AccountCostSummary.HasAnyExactCosts && parseComposerProps.AccountCostSummary.TotalCost > 0,
	}
	if parseRegionProps.HasThreadCost {
		parseThreadCost := formatCostUSD(parseComposerProps.ThreadCostSummary.TotalCost)
		parseRegionProps.RenderThread = parseBuildComposerThreadCostText(
			parseComposerProps.Intl,
			parseThreadCost,
			parseComposerProps.ThreadCostSummary.AllAssistantCostsExact,
		)
	}
	if parseRegionProps.HasAccountCost {
		parseAccountCost := formatCostUSD(parseComposerProps.AccountCostSummary.TotalCost)
		parsePremiumPercent := formatPercentValue(parseComposerProps.AccountCostSummary.PremiumPercent)
		parseRegionProps.RenderAccount = parseBuildComposerAccountCostText(
			parseComposerProps.Intl,
			parseAccountCost,
			parsePremiumPercent,
			parseComposerProps.AccountCostSummary.AllThreadCostsExact,
		)
	}
	return parseRegionProps
}

// parseBuildComposerThreadCostText builds the localized thread cost summary label.
func parseBuildComposerThreadCostText(parseIntl i18n.Runtime, parseCost string, isExact bool) string {
	if isExact {
		return parseIntl.T(chatI18nNamespace, "input.threadTotal", i18n.Arguments{"cost": parseCost})
	}
	return parseIntl.T(chatI18nNamespace, "input.threadTotalPartial", i18n.Arguments{"cost": parseCost})
}

// parseBuildComposerAccountCostText builds the localized account cost summary label.
func parseBuildComposerAccountCostText(parseIntl i18n.Runtime, parseCost string, parsePremiumPercent string, isExact bool) string {
	parseArgs := i18n.Arguments{
		"cost":           parseCost,
		"premiumPercent": parsePremiumPercent,
	}
	if isExact {
		return parseIntl.T(chatI18nNamespace, "input.accountTotal", parseArgs)
	}
	return parseIntl.T(chatI18nNamespace, "input.accountTotalPartial", parseArgs)
}

// renderComposerCostRegion renders the display-only runtime2 region body for composer cost labels.
func renderComposerCostRegion(parseRegionProps renderComposerCostRegionProps) ui.Node {
	return Fragment(
		If(parseRegionProps.HasThreadCost,
			P(Class("text-right text-base text-white/20 mt-1 pr-1"), Text(parseRegionProps.RenderThread)),
		),
		If(parseRegionProps.HasAccountCost,
			P(Class("text-right text-sm text-white/35 mt-0.5 pr-1"), Text(parseRegionProps.RenderAccount)),
		),
	)
}

// renderComposerCostParallelRegion renders the runtime2-backed display-only cost summary region.
func renderComposerCostParallelRegion(parseComposerProps composerProps) ui.Node {
	return ui.ParallelRegion(ui.ParallelRegionSpec[renderComposerCostRegionProps]{
		RendererID:       renderComposerCostRendererID,
		RegionInstanceID: renderComposerCostRegionID,
		Props:            parseBuildComposerCostRegionProps(parseComposerProps),
	})
}
