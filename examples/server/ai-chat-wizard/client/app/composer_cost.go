//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/i18n"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type renderComposerCostRegionProps struct {
	HasThreadCost  bool
	RenderThread   string
	HasAccountCost bool
	RenderAccount  string
}

// parseBuildComposerCostRegionProps formats the display-only composer cost labels.
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

// renderComposerCostRegion renders the display-only composer cost labels.
func renderComposerCostRegion(parseRegionProps renderComposerCostRegionProps) ui.Node {
	return Fragment(
		If(parseRegionProps.HasThreadCost,
			P(ClassStr("text-right text-base text-white/20 mt-1 pr-1"), Text(parseRegionProps.RenderThread)),
		),
		If(parseRegionProps.HasAccountCost,
			P(ClassStr("text-right text-sm text-white/35 mt-0.5 pr-1"), Text(parseRegionProps.RenderAccount)),
		),
	)
}
