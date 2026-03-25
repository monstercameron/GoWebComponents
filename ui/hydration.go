package ui

// HydrationOptions configures how a server-rendered tree is resumed in the browser.
type HydrationOptions struct {
	ScriptID          string
	ReferenceScriptID string
	Bootstrap         SSRBootstrap
	BootstrapRef      SSRBootstrapReference
	Strict            bool
	Observability     SSRObservabilityOptions
}

// resolveHydrationOptions is a core package helper.
func resolveHydrationOptions(parseHydrationOptions []HydrationOptions) HydrationOptions {
	if len(parseHydrationOptions) == 0 {
		return HydrationOptions{}
	}
	return parseHydrationOptions[0]
}
