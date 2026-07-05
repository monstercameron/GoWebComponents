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

func containsText(parseHaystack, parseNeedle string) bool {
	for parseI := 0; parseI+len(parseNeedle) <= len(parseHaystack); parseI++ {
		if parseHaystack[parseI:parseI+len(parseNeedle)] == parseNeedle {
			return true
		}
	}
	return false
}
