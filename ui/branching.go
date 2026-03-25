package ui

// NodeFactory lazily produces a ui.Node when a branch is selected.
type NodeFactory func() Node

// Component is a concise alias for CreateElement when rendering components.
func Component(parseComponent interface{}, parseProps ...interface{}) Node {
	return CreateElement(parseComponent, parseProps...)
}

// If lazily renders one of two branches.
func If(isCondition bool, parseWhenTrue NodeFactory, parseWhenFalse ...NodeFactory) Node {
	if isCondition {
		if parseWhenTrue == nil {
			return nil
		}
		return parseWhenTrue()
	}
	if len(parseWhenFalse) == 0 || parseWhenFalse[0] == nil {
		return nil
	}
	return parseWhenFalse[0]()
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
func (parseBuilder MatchBuilder) When(isCondition bool, render NodeFactory) MatchBuilder {
	if parseBuilder.matched || !isCondition {
		return parseBuilder
	}
	parseBuilder.matched = true
	if render != nil {
		parseBuilder.node = render()
	}
	return parseBuilder
}

// Default resolves the chain to the first matching node or the default branch.
func (parseBuilder MatchBuilder) Default(render NodeFactory) Node {
	if parseBuilder.matched {
		return parseBuilder.node
	}
	if render == nil {
		return nil
	}
	return render()
}
