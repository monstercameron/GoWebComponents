//go:build js && wasm
// +build js,wasm

package fiber

import "testing"

func BenchmarkDeriveContextValues(b *testing.B) {
	contextA := CreateContext("a")
	contextB := CreateContext("b")
	contextC := CreateContext("c")

	parent := map[int64]interface{}{
		contextA.id: "theme-dark",
		contextB.id: "density-compact",
		contextC.id: "locale-en-US",
		101:         true,
		102:         42,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = deriveContextValues(parent, contextA.id, "theme-light")
	}
}

func BenchmarkResolveContextValue(b *testing.B) {
	context := CreateContext("fallback")
	fiber := &Fiber{
		contextValues: map[int64]interface{}{
			context.id: "provided",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = resolveContextValue(fiber, context)
	}
}

func BenchmarkContextProviderTraversal(b *testing.B) {
	context := CreateContext("fallback")
	reader := func(props Attrs) *Element {
		return Text(GoUseContext[string](context))
	}
	previousWIPFiber := wipFiber
	defer func() {
		wipFiber = previousWIPFiber
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		root := &Fiber{
			typeOf: "ROOT",
			props: map[string]interface{}{
				"children": []interface{}{
					CreateElement(context.Provider, map[string]interface{}{"value": "provided"},
						CreateElement(reader, nil),
					),
				},
			},
		}

		performUnitOfWork(root)
		performUnitOfWork(root.child)
		performUnitOfWork(root.child.child)
	}
}
