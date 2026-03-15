//go:build js && wasm
// +build js,wasm

package html

import "testing"

func TestDiv_OmitsZeroValueProps(t *testing.T) {
	elem := Div(Props{})
	if elem == nil {
		t.Fatal("expected element")
	}
	if len(elem.Props) != 1 {
		t.Fatalf("expected only runtime children prop, got %#v", elem.Props)
	}
	if _, ok := elem.Props["children"]; !ok {
		t.Fatalf("expected runtime children prop, got %#v", elem.Props)
	}
}

func TestInput_KeepsMeaningfulProps(t *testing.T) {
	elem := Input(Props{
		ID:       "name",
		Value:    "alice",
		Disabled: true,
		Rows:     4,
		Class:    "field",
		Data:     map[string]string{"mode": "demo"},
		Aria:     map[string]string{"label": "Name"},
		Raw:      map[string]interface{}{"tabIndex": 2},
	})
	if elem == nil {
		t.Fatal("expected element")
	}
	if elem.Props["id"] != "name" {
		t.Fatalf("expected id prop, got %#v", elem.Props["id"])
	}
	if elem.Props["value"] != "alice" {
		t.Fatalf("expected value prop, got %#v", elem.Props["value"])
	}
	if elem.Props["disabled"] != true {
		t.Fatalf("expected disabled prop, got %#v", elem.Props["disabled"])
	}
	if elem.Props["rows"] != 4 {
		t.Fatalf("expected rows prop, got %#v", elem.Props["rows"])
	}
	if elem.Props["data-mode"] != "demo" {
		t.Fatalf("expected data attribute, got %#v", elem.Props["data-mode"])
	}
	if elem.Props["aria-label"] != "Name" {
		t.Fatalf("expected aria attribute, got %#v", elem.Props["aria-label"])
	}
	if elem.Props["tabIndex"] != 2 {
		t.Fatalf("expected raw prop override, got %#v", elem.Props["tabIndex"])
	}
	if _, ok := elem.Props["required"]; ok {
		t.Fatalf("expected zero-value bool prop to be omitted, got %#v", elem.Props["required"])
	}
}
