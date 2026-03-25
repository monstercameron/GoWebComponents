package runtime

import "testing"

func TestReconcileChildren_PreservesHooksForUnchangedFunctionComponent(parseT *testing.T) {
	parseFn := func(parseProps map[string]interface{}) *Element { return nil }
	parseSharedChildren := emptyChildren
	parseSharedProps := map[string]interface{}{"children": parseSharedChildren}
	parseOldHooks := &Hooks{}

	parseOldFiber := &Fiber{
		typeOf: parseFn,
		props:  parseSharedProps,
		hooks:  parseOldHooks,
	}
	parseParent := &Fiber{
		alternate: &Fiber{child: parseOldFiber},
	}
	parseRt := &Runtime{}

	parseRt.reconcileChildren(parseParent, []interface{}{
		&Element{
			Type:  parseFn,
			Props: parseSharedProps,
		},
	})

	if parseParent.child == nil {
		parseT.Fatal("expected child fiber to be created")
	}
	if parseParent.child.hooks != parseOldHooks {
		parseT.Fatal("expected unchanged function component to retain hook storage")
	}
}
