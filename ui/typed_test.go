package ui

import (
	"testing"
)

type typedTestProps struct {
	GetLabel string
}

func renderTypedTestComponent(parseProps typedTestProps) Node {
	return Text(parseProps.GetLabel)
}

// TestTypedSharesIdentityWithCreateElement pins that a ui.Typed constructor
// and plain ui.CreateElement produce elements with the SAME component handle:
// mixing the two must reconcile as the same component (no remount, hooks
// preserved), and the Typed registration's static renderer must survive a
// later CreateElement of the same function value.
func TestTypedSharesIdentityWithCreateElement(parseT *testing.T) {
	getTypedConstructor := Typed(renderTypedTestComponent)
	getTypedNode := getTypedConstructor(typedTestProps{GetLabel: "a"})
	getPlainNode := CreateElement(renderTypedTestComponent, typedTestProps{GetLabel: "b"})
	if getTypedNode == nil || getPlainNode == nil {
		parseT.Fatal("expected both construction paths to build elements")
	}
	if getTypedNode.Type != getPlainNode.Type {
		parseT.Fatalf("Typed and CreateElement produced different component handles: %#v vs %#v",
			getTypedNode.Type, getPlainNode.Type)
	}
}

// TestTypedConstructorRendersProps pins that the static trampoline delivers
// the typed props to the component.
func TestTypedConstructorRendersProps(parseT *testing.T) {
	getConstructor := Typed(renderTypedTestComponent)
	getNode := getConstructor(typedTestProps{GetLabel: "hello-typed"})
	getRendered, parseErr := RenderToString(getNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if getRendered == "" || !containsText(getRendered, "hello-typed") {
		parseT.Fatalf("expected rendered output to contain typed props label, got %q", getRendered)
	}
}

func TestTypedPropsEqualKeepsDynamicInterfaceAndSliceSemantics(parseT *testing.T) {
	getInterfaceEqual := buildTypedPropsEqual[struct{ GetValue any }]()
	getSlice := []int{1, 2}
	if !getInterfaceEqual(struct{ GetValue any }{GetValue: 7}, struct{ GetValue any }{GetValue: 7}) {
		parseT.Fatal("equal comparable values in interface props should compare equal")
	}
	if !getInterfaceEqual(struct{ GetValue any }{GetValue: getSlice}, struct{ GetValue any }{GetValue: getSlice}) {
		parseT.Fatal("the same slice identity in interface props should compare equal")
	}
	if getInterfaceEqual(struct{ GetValue any }{GetValue: getSlice}, struct{ GetValue any }{GetValue: []int{1, 2}}) {
		parseT.Fatal("distinct slice identities in interface props should compare unequal")
	}
	getMap := map[string]int{"key": 1}
	if !getInterfaceEqual(struct{ GetValue any }{GetValue: getMap}, struct{ GetValue any }{GetValue: getMap}) {
		parseT.Fatal("the same map identity in interface props should compare equal")
	}
	if getInterfaceEqual(struct{ GetValue any }{GetValue: getMap}, struct{ GetValue any }{GetValue: map[string]int{"key": 1}}) {
		parseT.Fatal("distinct map identities in interface props should compare unequal")
	}

	getSliceEqual := buildTypedPropsEqual[struct{ GetValues []int }]()
	if !getSliceEqual(struct{ GetValues []int }{GetValues: getSlice}, struct{ GetValues []int }{GetValues: getSlice}) {
		parseT.Fatal("the same slice identity in non-comparable props should compare equal")
	}
	getNestedEqual := buildTypedPropsEqual[struct{ GetRows [1]struct{ Values []int } }]()
	getNestedLeft := struct{ GetRows [1]struct{ Values []int } }{GetRows: [1]struct{ Values []int }{{Values: getSlice}}}
	getNestedRight := struct{ GetRows [1]struct{ Values []int } }{GetRows: [1]struct{ Values []int }{{Values: getSlice}}}
	if !getNestedEqual(getNestedLeft, getNestedRight) {
		parseT.Fatal("nested arrays and structs should preserve shallow slice identity equality")
	}
	getNestedRight.GetRows[0].Values = []int{1, 2}
	if getNestedEqual(getNestedLeft, getNestedRight) {
		parseT.Fatal("nested arrays and structs with distinct slices should compare unequal")
	}
}

func containsText(parseHaystack, parseNeedle string) bool {
	for parseI := 0; parseI+len(parseNeedle) <= len(parseHaystack); parseI++ {
		if parseHaystack[parseI:parseI+len(parseNeedle)] == parseNeedle {
			return true
		}
	}
	return false
}
