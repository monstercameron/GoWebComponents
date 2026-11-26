package runtime

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

// TestFiberSizeRemainsCompact pins the Fiber struct size after moving cold
// profiling/boundary/reactive fields behind *fiberCold and packing Cache Line 0.
func TestFiberSizeRemainsCompact(t *testing.T) {
	got := unsafe.Sizeof(Fiber{})
	if got > 264 {
		t.Fatalf("Fiber size regressed: got %d bytes, want <= 264", got)
	}
	t.Logf("Fiber size = %d bytes, fiberCold size = %d bytes", got, unsafe.Sizeof(fiberCold{}))
}

// TestKeyedHeadTailPrependAndDeleteZeroDOMReadback verifies head/tail keyed
// trimming and fiber-derived child-order repair produce exact DOM order with
// zero GetFirstChild/GetNextSibling read-back calls on direct host lists.
func TestKeyedHeadTailPrependAndDeleteZeroDOMReadback(t *testing.T) {
	adapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	rt := NewRuntime(Config{DOMAdapter: adapter})
	container := adapter.CreateElement("root")

	renderRows := func(ids ...string) *Element {
		children := make([]any, 0, len(ids))
		for _, id := range ids {
			children = append(children, CreateElement("div", map[string]any{"id": id, "key": id}, id))
		}
		return CreateElement("section", map[string]any{"id": "list"}, children...)
	}

	assertOrder := func(want ...string) {
		t.Helper()
		rootChildren := adapter.GetChildren(container)
		if len(rootChildren) != 1 {
			t.Fatalf("expected 1 host child, got %d", len(rootChildren))
		}
		host := rootChildren[0].(*testDOMNode)
		if len(host.children) != len(want) {
			t.Fatalf("host child count = %d, want %d", len(host.children), len(want))
		}
		for i, id := range want {
			child := host.children[i].(*testDOMNode)
			if child.attributes["id"] != id {
				t.Fatalf("child[%d] id = %q, want %q (full order mismatch)", i, child.attributes["id"], id)
			}
		}
	}

	rt.Render(renderRows("a", "b", "c", "d"), container)
	assertOrder("a", "b", "c", "d")

	// 1. Prepend: pure head insertion (suffix [a,b,c,d] trimmed, 1 InsertBefore, 0 DOM reads).
	adapter.resetCounts()
	rt.Render(renderRows("z", "a", "b", "c", "d"), container)
	assertOrder("z", "a", "b", "c", "d")
	if adapter.getFirstChildCount != 0 || adapter.getNextSiblingCount != 0 {
		t.Fatalf("expected 0 DOM read-backs on prepend, got firstChild=%d nextSibling=%d", adapter.getFirstChildCount, adapter.getNextSiblingCount)
	}
	if adapter.insertBeforeCount != 1 {
		t.Fatalf("expected 1 InsertBefore on prepend, got %d", adapter.insertBeforeCount)
	}

	// 2. Mid-list deletion: pure middle delete (needsChildOrder=false, 0 InsertBefore, 0 DOM reads).
	adapter.resetCounts()
	rt.Render(renderRows("z", "a", "c", "d"), container)
	assertOrder("z", "a", "c", "d")
	if adapter.getFirstChildCount != 0 || adapter.getNextSiblingCount != 0 || adapter.insertBeforeCount != 0 {
		t.Fatalf("expected 0 read-backs/moves on mid-list delete, got first=%d next=%d insert=%d",
			adapter.getFirstChildCount, adapter.getNextSiblingCount, adapter.insertBeforeCount)
	}

	// 3. Adjacent swap inside middle window: 1 InsertBefore, 0 DOM reads.
	adapter.resetCounts()
	rt.Render(renderRows("z", "c", "a", "d"), container)
	assertOrder("z", "c", "a", "d")
	if adapter.getFirstChildCount != 0 || adapter.getNextSiblingCount != 0 {
		t.Fatalf("expected 0 DOM read-backs on swap, got firstChild=%d nextSibling=%d", adapter.getFirstChildCount, adapter.getNextSiblingCount)
	}
	if adapter.insertBeforeCount != 1 {
		t.Fatalf("expected 1 InsertBefore on adjacent swap, got %d", adapter.insertBeforeCount)
	}
}

// TestWriteSerializedEscapedStringMatchesHTML verifies the fast-path escaper
// matches Go's html.EscapeString across plain ASCII, UTF-8, and all special chars.
func TestWriteSerializedEscapedStringMatchesHTML(t *testing.T) {
	cases := []string{
		"",
		"plain ascii text 12345",
		"café naïve 日本語 🚀",
		`<script>alert("xss" & 'quotes')</script>`,
		`a&b<c>d"e'f`,
		`&`,
		`>`,
		`trailing&`,
	}
	for _, tc := range cases {
		var b strings.Builder
		writeSerializedEscapedString(&b, tc)
		want := strings.NewReplacer(
			"&", "&amp;",
			"<", "&lt;",
			">", "&gt;",
			`"`, "&#34;",
			"'", "&#39;",
		).Replace(tc)
		if b.String() != want {
			t.Fatalf("writeSerializedEscapedString(%q) = %q, want %q", tc, b.String(), want)
		}
	}
}

// BenchmarkReconcileKeyedPrepend256 measures prepending 1 row onto a 256-row keyed list.
func BenchmarkReconcileKeyedPrepend256(b *testing.B) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter})
	container := adapter.CreateElement("root")

	buildList := func(withHead bool) *Element {
		children := make([]any, 0, 257)
		if withHead {
			children = append(children, CreateElement("div", map[string]any{"key": "head"}, "head"))
		}
		for i := range 256 {
			k := fmt.Sprintf("row-%d", i)
			children = append(children, CreateElement("div", map[string]any{"key": k}, k))
		}
		return CreateElement("section", nil, children...)
	}

	listBase := buildList(false)
	listPrepended := buildList(true)
	rt.Render(listBase, container)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			rt.Render(listPrepended, container)
		} else {
			rt.Render(listBase, container)
		}
	}
}
