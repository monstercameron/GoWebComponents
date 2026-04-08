//go:build js && wasm
// +build js,wasm

package main

const (
	allFilterValue = "all"

	moduleCore      = "core"
	moduleState     = "state"
	moduleData      = "data"
	moduleRendering = "rendering"
	modulePlugins   = "plugins"
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

	contentKindExample = "example"
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
	labelSourceFirstExample  = "Source-first example"
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
	labelConceptFocus        = "Concept focus"
	labelStudyPrompts        = "What to notice"
	labelLivePreview         = "Live preview"
	labelPreviewMode         = "Isolated runtime preview"
	labelStandalonePreview   = "Open standalone"
	labelReferenceSearch     = "Reference search"
	labelNothingSelected     = "Choose an example"
	labelCatalogEntries      = "Runnable examples"
	labelLiveExamples        = "Stable builds"
	labelAPIPanes            = "Advanced patterns"
	labelSingleStackUI       = "Preview mode"

	messageNothingSelected = "Choose an example from the gallery."
	messageAdjustFilters   = "Adjust the search or filters to bring another example into view."
	messageNoMatches       = "No matches yet. Try a broader search or switch the active filter."
	messageDocLoading      = "Loading the selected example source..."
	messageDocUnavailable  = "This concept does not define a source markdown document."
	messageDocEmpty        = "The selected markdown document loaded, but it did not contain renderable content."
	messageReferenceSearch = "Search functions in this reference..."
	messageSourceFirstDemo = "This example builds to wasm, but the gallery keeps it source-first because it depends on standalone document scaffolding or browser-global behavior."

	contentKindLabelArticle = "Markdown article"
	contentKindLabelAPI     = "Structured API"
	contentKindLabelDemo    = "Interactive demo"
	contentKindLabelUnknown = "Unknown"

	toneReady    = "Ready"
	tonePositive = "Positive"
	toneNegative = "Negative"

	buttonBrowseExamples     = "Start with counter"
	buttonInspectPackageAPIs = "Explore advanced patterns"
	buttonResetFilters       = "Reset filters"
	buttonRetryDocument      = "Retry document"
	buttonDecrement          = "Decrement"
	buttonIncrement          = "Increment"
	buttonReset              = "Reset"

	labelStateTonePrefix   = "State tone: "
	catalogDataRelativeURL = "assets/data/catalog.json"
	catalogCacheKeyPrefix  = "public-examples-site:catalog:"
	markdownCacheKeyPrefix = "public-examples-site:markdown:"
)
