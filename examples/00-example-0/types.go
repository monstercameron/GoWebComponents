//go:build js && wasm
// +build js,wasm

package main

import "github.com/monstercameron/GoWebComponents/ui"

type sortOption struct {
	Value string
	Label string
}

type docsParam struct {
	Name        string
	Type        string
	Required    string
	Description string
}

type docsSection struct {
	Heading    string
	Paragraphs []string
}

type docsContent struct {
	Kind        string
	AnchorID    string
	EmbedPath   string
	Sections    []docsSection
	Callout     string
	SourcePath  string
	Code        string
	Signature   string
	Summary     string
	Params      []docsParam
	Returns     string
	Example     string
	Notes       []string
	Description string
	Tips        []string
}

type docsItem struct {
	ID         int
	Title      string
	Status     string
	Module     string
	Type       string
	Level      string
	Tags       []string
	SearchTags []string
	Blurb      string
	ReadTime   string
	Content    docsContent
}

type contentPanelProps struct {
	Item            docsItem
	MarkdownBody    string
	MarkdownLoading bool
	MarkdownReady   bool
	MarkdownError   string
	AnchorScrollID  int
	OnRetryMarkdown ui.Handler
}

type catalogHeroProps struct {
	OnBrowseExamples ui.Handler
	OnInspectAPIs    ui.Handler
	TotalItems       int
	ExampleCount     int
	APICount         int
	comparisonGuard  func()
}

type catalogSidebarProps struct {
	SearchQuery          string
	ResultCount          int
	HasActiveFilters     bool
	Statuses             []string
	Levels               []string
	Modules              []string
	SortOptions          []sortOption
	FilterButtons        []ui.Node
	SelectedStatusFilter string
	SelectedLevelFilter  string
	SelectedModuleFilter string
	SelectedSortOrder    string
	ItemNodes            []ui.Node
	OnSearchInput        ui.Handler
	OnStatusChange       ui.Handler
	OnLevelChange        ui.Handler
	OnModuleChange       ui.Handler
	OnSortChange         ui.Handler
	OnResetFilters       ui.Handler
}

type detailPanelProps struct {
	SelectedItem    docsItem
	HasSelectedItem bool
	MarkdownBody    string
	MarkdownLoading bool
	MarkdownReady   bool
	MarkdownError   string
	AnchorScrollID  int
	OnRetryMarkdown ui.Handler
}

type docsCatalog struct {
	Modules     []string     `json:"modules"`
	Statuses    []string     `json:"statuses"`
	Levels      []string     `json:"levels"`
	Filters     []string     `json:"filters"`
	SortOptions []sortOption `json:"sortOptions"`
	Items       []docsItem   `json:"items"`
}
