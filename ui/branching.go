package ui

// NodeFactory lazily produces a ui.Node when a branch is selected.
type NodeFactory func() Node

// Component is a concise alias for CreateElement when rendering components.
func Component(component interface{}, props ...interface{}) Node {
	return CreateElement(component, props...)
}

// If lazily renders one of two branches.
func If(condition bool, whenTrue NodeFactory, whenFalse ...NodeFactory) Node {
	if condition {
		if whenTrue == nil {
			return nil
		}
		return whenTrue()
	}
	if len(whenFalse) == 0 || whenFalse[0] == nil {
		return nil
	}
	return whenFalse[0]()
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
func (builder MatchBuilder) When(condition bool, render NodeFactory) MatchBuilder {
	if builder.matched || !condition {
		return builder
	}
	builder.matched = true
	if render != nil {
		builder.node = render()
	}
	return builder
}

// Default resolves the chain to the first matching node or the default branch.
func (builder MatchBuilder) Default(render NodeFactory) Node {
	if builder.matched {
		return builder.node
	}
	if render == nil {
		return nil
	}
	return render()
}