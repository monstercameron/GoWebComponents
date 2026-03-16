package ui

type HydrationOptions struct {
	ScriptID          string
	ReferenceScriptID string
	Bootstrap         SSRBootstrap
	BootstrapRef      SSRBootstrapReference
}

func resolveHydrationOptions(options []HydrationOptions) HydrationOptions {
	if len(options) == 0 {
		return HydrationOptions{}
	}
	return options[0]
}

var _ = resolveHydrationOptions
