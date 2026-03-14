package runtime

import "testing"

func TestReconcileChildren_PreservesHooksForUnchangedFunctionComponent(t *testing.T) {
	fn := func(props map[string]interface{}) *Element { return nil }
	sharedChildren := emptyChildren
	sharedProps := map[string]interface{}{"children": sharedChildren}
	oldHooks := &Hooks{}

	oldFiber := &Fiber{
		typeOf: fn,
		props:  sharedProps,
		hooks:  oldHooks,
	}
	parent := &Fiber{
		alternate: &Fiber{child: oldFiber},
	}
	rt := &Runtime{}

	rt.reconcileChildren(parent, []interface{}{
		&Element{
			Type:  fn,
			Props: sharedProps,
		},
	})

	if parent.child == nil {
		t.Fatal("expected child fiber to be created")
	}
	if parent.child.hooks != oldHooks {
		t.Fatal("expected unchanged function component to retain hook storage")
	}
}
