package runtime

// SelectorResolves reports whether parseSelector resolves to a live DOM
// container in the current runtime. The agent bridge uses it to reject a
// bridge.mount onto a non-existent selector up front: RenderTo signals a
// missing selector by panicking, but the production panic policy suppresses
// that signal, so callers cannot otherwise tell a bad selector from success.
func (parseRt *Runtime) SelectorResolves(parseSelector string) bool {
	if parseRt == nil {
		return false
	}
	return !IsDOMNodeNull(parseRt.queryContainer(parseSelector))
}
