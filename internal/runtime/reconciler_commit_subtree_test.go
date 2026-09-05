package runtime

import (
	"strings"
	"testing"
)

func TestWriteSerializedAttrsRoundTrip(t *testing.T) {
	var builder strings.Builder
	attrs := []HostAttr{
		{Name: "data-row-id", Value: "7&8"},
		{Name: "className", Value: "row"},
	}
	if !writeSerializedAttrsRoundTrip(&builder, attrs) {
		t.Fatal("writeSerializedAttrsRoundTrip rejected safe attributes")
	}
	if got, want := builder.String(), ` class="row" data-row-id="7&amp;8"`; got != want {
		t.Fatalf("writeSerializedAttrsRoundTrip output = %q, want %q", got, want)
	}
}

func TestWriteSerializedAttrsRoundTripRejectsUnsafeInputBeforeWriting(t *testing.T) {
	tests := []struct {
		name  string
		attrs []HostAttr
	}{
		{name: "attribute name", attrs: []HostAttr{{Name: `title onmouseover`, Value: "alert(1)"}}},
		{name: "script URL", attrs: []HostAttr{{Name: "href", Value: "javascript:alert(1)"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var builder strings.Builder
			if writeSerializedAttrsRoundTrip(&builder, test.attrs) {
				t.Fatal("writeSerializedAttrsRoundTrip accepted unsafe input")
			}
			if builder.Len() != 0 {
				t.Fatalf("writeSerializedAttrsRoundTrip wrote %q before rejecting input", builder.String())
			}
		})
	}
}
