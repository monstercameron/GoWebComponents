package hotreload

// Config controls how the hot reload bridge captures state.
type Config struct {
	// AtomIDs limits exported atom state to the provided ids.
	// When empty, all exported atom state is included.
	AtomIDs []string

	// ResetKey opts into explicit snapshot invalidation. When the reset key
	// changes between builds, previously exported local and shared state is
	// discarded instead of being restored into the new module.
	ResetKey string
}

// Enable installs the default hot reload bridge for development builds.
func Enable() {
	Configure(Config{})
}
