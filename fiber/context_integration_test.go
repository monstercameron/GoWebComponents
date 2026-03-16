//go:build js && wasm
// +build js,wasm

package fiber

import "testing"

func TestContextProviderHookIntegration(t *testing.T) {
	context := CreateContext("fallback")
	previousWIPFiber := wipFiber
	defer func() {
		wipFiber = previousWIPFiber
	}()

	reader := func(props Attrs) *Element {
		return Text(GoUseContext[string](context))
	}

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
	providerFiber := root.child
	if providerFiber == nil {
		t.Fatal("expected provider fiber to be created")
	}

	performUnitOfWork(providerFiber)
	readerFiber := providerFiber.child
	if readerFiber == nil {
		t.Fatal("expected reader fiber to be created")
	}

	performUnitOfWork(readerFiber)
	textFiber := readerFiber.child
	if textFiber == nil {
		t.Fatal("expected text fiber to be created")
	}

	if got := textFiber.props["nodeValue"]; got != "provided" {
		t.Fatalf("expected provider value to reach hook consumer, got %v", got)
	}

	if got := readerFiber.contextValues[context.id]; got != "provided" {
		t.Fatalf("expected reader fiber context value provided, got %v", got)
	}
}

func TestNestedContextProvidersIntegration(t *testing.T) {
	context := CreateContext("fallback")
	previousWIPFiber := wipFiber
	defer func() {
		wipFiber = previousWIPFiber
	}()

	reader := func(props Attrs) *Element {
		return Text(GoUseContext[string](context))
	}

	root := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(context.Provider, map[string]interface{}{"value": "outer"},
					CreateElement(context.Provider, map[string]interface{}{"value": "inner"},
						CreateElement(reader, nil),
					),
				),
			},
		},
	}

	performUnitOfWork(root)
	outerProviderFiber := root.child
	if outerProviderFiber == nil {
		t.Fatal("expected outer provider fiber to be created")
	}

	performUnitOfWork(outerProviderFiber)
	innerProviderFiber := outerProviderFiber.child
	if innerProviderFiber == nil {
		t.Fatal("expected inner provider fiber to be created")
	}

	performUnitOfWork(innerProviderFiber)
	readerFiber := innerProviderFiber.child
	if readerFiber == nil {
		t.Fatal("expected nested reader fiber to be created")
	}

	performUnitOfWork(readerFiber)
	textFiber := readerFiber.child
	if textFiber == nil {
		t.Fatal("expected nested text fiber to be created")
	}

	if got := textFiber.props["nodeValue"]; got != "inner" {
		t.Fatalf("expected nested provider override to win, got %v", got)
	}

	if got := readerFiber.contextValues[context.id]; got != "inner" {
		t.Fatalf("expected nested reader context value inner, got %v", got)
	}
}

func TestContextConsumerIntegration(t *testing.T) {
	context := CreateContext("fallback")

	root := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(context.Provider, map[string]interface{}{"value": "consumer-value"},
					CreateElement(context.Consumer, map[string]interface{}{
						"render": func(value interface{}) *Element {
							text, _ := value.(string)
							return Text(text)
						},
					}),
				),
			},
		},
	}

	performUnitOfWork(root)
	providerFiber := root.child
	if providerFiber == nil {
		t.Fatal("expected provider fiber to be created")
	}

	performUnitOfWork(providerFiber)
	consumerFiber := providerFiber.child
	if consumerFiber == nil {
		t.Fatal("expected consumer fiber to be created")
	}

	performUnitOfWork(consumerFiber)
	textFiber := consumerFiber.child
	if textFiber == nil {
		t.Fatal("expected consumer output fiber to be created")
	}

	if got := textFiber.props["nodeValue"]; got != "consumer-value" {
		t.Fatalf("expected consumer render value, got %v", got)
	}
}

func TestContextProviderWithoutValueUsesDefaultIntegration(t *testing.T) {
	context := CreateContext("fallback")
	previousWIPFiber := wipFiber
	defer func() {
		wipFiber = previousWIPFiber
	}()

	reader := func(props Attrs) *Element {
		return Text(GoUseContext[string](context))
	}

	root := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(context.Provider, nil,
					CreateElement(reader, nil),
				),
			},
		},
	}

	performUnitOfWork(root)
	providerFiber := root.child
	if providerFiber == nil {
		t.Fatal("expected provider fiber to be created")
	}

	performUnitOfWork(providerFiber)
	readerFiber := providerFiber.child
	if readerFiber == nil {
		t.Fatal("expected reader fiber to be created")
	}

	performUnitOfWork(readerFiber)
	textFiber := readerFiber.child
	if textFiber == nil {
		t.Fatal("expected text fiber to be created")
	}

	if got := textFiber.props["nodeValue"]; got != "fallback" {
		t.Fatalf("expected provider without explicit value to use default, got %v", got)
	}
}

func TestContextConsumerFallsBackToDefaultIntegration(t *testing.T) {
	context := CreateContext("fallback")

	root := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(context.Consumer, map[string]interface{}{
					"render": func(value interface{}) *Element {
						text, _ := value.(string)
						return Text(text)
					},
				}),
			},
		},
	}

	performUnitOfWork(root)
	consumerFiber := root.child
	if consumerFiber == nil {
		t.Fatal("expected consumer fiber to be created")
	}

	performUnitOfWork(consumerFiber)
	textFiber := consumerFiber.child
	if textFiber == nil {
		t.Fatal("expected default consumer output fiber to be created")
	}

	if got := textFiber.props["nodeValue"]; got != "fallback" {
		t.Fatalf("expected consumer without provider to use default value, got %v", got)
	}
}
