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

func resolveHydrationOptions(options []HydrationOptions) HydrationOptions {
	if len(options) == 0 {
		return HydrationOptions{}
	}
	return options[0]
}
