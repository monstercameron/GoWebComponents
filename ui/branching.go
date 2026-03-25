package ui

// NodeFactory lazily produces a ui.Node when a branch is selected.
type NodeFactory func() Node

// Component is a concise alias for CreateElement when rendering components.
func Component(parseComponentType interface{}, parseComponentProps ...interface{}) Node {
	return CreateElement(parseComponentType, parseComponentProps...)
}

// If lazily renders one of two branches.
func If(isCondition bool, parseBranchTrue NodeFactory, parseBranchFalse ...NodeFactory) Node {
	if isCondition {
		if parseBranchTrue == nil {
			return nil
		}
		return parseBranchTrue()
	}
	if len(parseBranchFalse) == 0 || parseBranchFalse[0] == nil {
		return nil
	}
	return parseBranchFalse[0]()
}

// MatchBuilder lazily resolves the first matching branch and a default.
type MatchBuilder struct {
	matched bool
	node    Node
}

// Match starts a lazy conditional chain.
func Match() MatchBuilder {
	return MatchBuilder{}
}

// When records the first matching branch in the chain.
func (parseMatchBuilder MatchBuilder) When(isCondition bool, parseBranchRender NodeFactory) MatchBuilder {
	if parseMatchBuilder.matched || !isCondition {
		return parseMatchBuilder
	}
	parseMatchBuilder.matched = true
	if parseBranchRender != nil {
		parseMatchBuilder.node = parseBranchRender()
	}
	return parseMatchBuilder
}

// Default resolves the chain to the first matching node or the default branch.
func (parseMatchBuilder MatchBuilder) Default(parseBranchRender NodeFactory) Node {
	if parseMatchBuilder.matched {
		return parseMatchBuilder.node
	}
	if parseBranchRender == nil {
		return nil
	}
	return parseBranchRender()
}
