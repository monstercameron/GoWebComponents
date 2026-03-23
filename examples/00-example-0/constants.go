//go:build js && wasm
// +build js,wasm

package main

const (
	allFilterValue = "all"

	moduleCore      = "core"
	moduleRouter    = "router"
	moduleState     = "state"
	moduleCLI       = "cli"
	moduleData      = "data"
	moduleRendering = "rendering"
	moduleDesign    = "design"
	modulePlugins   = "plugins"
	moduleForms     = "forms"
	moduleMotion    = "motion"
	moduleCommerce  = "commerce"

	statusStable       = "stable"
	statusExperimental = "experimental"
	statusDeprecated   = "deprecated"

	levelBeginner     = "Beginner"
	levelCore         = "Core"
	levelIntermediate = "Intermediate"
	levelAdvanced     = "Advanced"

	kindConcept = "Concept"
	kindAPI     = "API"
	kindExample = "Example"

	contentKindArticle = "article"
	contentKindAPI     = "api"
	contentKindCounter = "counter"

	filterAll = "All"

	sortRelevance = "relevance"
	sortAlpha     = "alpha"
	sortLevel     = "level"

	readTimeReference   = "Reference"
	readTimeInteractive = "Interactive"

	labelRelevance = "Relevance"
	labelAlpha     = "A-Z"
	labelLevel     = "Level"

	labelConceptArticle      = "Concept article"
	labelMarkdownWriteup     = "Markdown-style write-up"
	labelRenderedMarkdown    = "Rendered markdown"
	labelSourceDocument      = "Source document"
	labelAPIReference        = "API reference"
	labelStructuredDocs      = "Structured documentation"
	labelInteractiveExample  = "Interactive example"
	labelReactiveDemoSurface = "Reactive demo surface"
	labelSignature           = "Signature"
	labelParameters          = "Parameters"
	labelReturns             = "Returns"
	labelNotes               = "Notes"
	labelUsageExample        = "Usage example"
	labelWhyThisMatters      = "Why this matters"
	labelExampleMarkdown     = "Example markdown block"
	labelExampleSource       = "Example source"
	labelLiveWidget          = "Live widget"
	labelNothingSelected     = "Nothing selected"
	labelCatalogEntries      = "Catalog entries"
	labelLiveExamples        = "Live examples"
	labelAPIPanes            = "API panes"
	labelSingleStackUI       = "Single stack UI"

	messageNothingSelected = "Nothing selected."
	messageAdjustFilters   = "Adjust the filters or search query to bring results back into view."
	messageNoMatches       = "No matches yet. Try a broader search or switch the active filter."
	messageDocLoading      = "Loading the selected markdown document..."
	messageDocUnavailable  = "This concept does not define a source markdown document."
	messageDocEmpty        = "The selected markdown document loaded, but it did not contain renderable content."

	contentKindLabelArticle = "Markdown article"
	contentKindLabelAPI     = "Structured API"
	contentKindLabelDemo    = "Interactive demo"
	contentKindLabelUnknown = "Unknown"

	toneReady    = "Ready"
	tonePositive = "Positive"
	toneNegative = "Negative"

	buttonBrowseExamples     = "Browse examples"
	buttonInspectPackageAPIs = "Inspect package APIs"
	buttonRetryDocument      = "Retry document"
	buttonDecrement          = "Decrement"
	buttonIncrement          = "Increment"
	buttonReset              = "Reset"

	labelStateTonePrefix   = "State tone: "
	catalogDataRelativeURL = "assets/data/catalog.json"
	catalogCacheKeyPrefix  = "example-0:catalog:"
	markdownCacheKeyPrefix = "example-0:markdown:"
)
