package hookcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func check(t *testing.T, parseSrc string) []Finding {
	t.Helper()
	parseFindings, parseErr := CheckSource("x.go", []byte("package p\n"+parseSrc))
	if parseErr != nil {
		t.Fatalf("parse: %v", parseErr)
	}
	return parseFindings
}

func TestFlagsHookInRangeLoop(t *testing.T) {
	parseFindings := check(t, `
func Row(items []int) {
	for _, x := range items {
		_ = x
		s := UseState(0)
		_ = s
	}
}`)
	if len(parseFindings) != 1 || parseFindings[0].Hook != "UseState" {
		t.Fatalf("expected one UseState finding, got %v", parseFindings)
	}
}

func TestFlagsQualifiedHookInForLoop(t *testing.T) {
	parseFindings := check(t, `
func Row(n int) {
	for i := 0; i < n; i++ {
		ui.UseEffect(func() func() { return nil }, i)
	}
}`)
	if len(parseFindings) != 1 || parseFindings[0].Hook != "UseEffect" {
		t.Fatalf("expected one UseEffect finding, got %v", parseFindings)
	}
}

func TestFlagsUseEventBehindHandlerInLoop(t *testing.T) {
	// The classic gotcha: an On* handler wrapped in ui.UseEvent inside the loop.
	parseFindings := check(t, `
func List(items []int) {
	for range items {
		_ = Button(Props{OnClick: ui.UseEvent(func() {})})
	}
}`)
	if len(parseFindings) != 1 || parseFindings[0].Hook != "UseEvent" {
		t.Fatalf("expected one UseEvent finding, got %v", parseFindings)
	}
}

func TestCleanTopLevelHooksNotFlagged(t *testing.T) {
	parseFindings := check(t, `
func Comp(items []int) {
	a := UseState(0)
	b := ui.UseRef(0)
	_ = a
	_ = b
	for _, x := range items {
		_ = Div(Text(x)) // no hooks in the loop
	}
}`)
	if len(parseFindings) != 0 {
		t.Fatalf("top-level hooks + hookless loop should be clean, got %v", parseFindings)
	}
}

func TestExtractedRowComponentNotFlagged(t *testing.T) {
	// The sanctioned fix: the per-row hook lives in its own component; the loop
	// only creates elements.
	parseFindings := check(t, `
func row(p rowProps) { s := UseState(0); _ = s }
func List(items []int) {
	for _, x := range items {
		_ = CreateElement(row, rowProps{n: x})
	}
}`)
	if len(parseFindings) != 0 {
		t.Fatalf("extracted-row pattern should be clean, got %v", parseFindings)
	}
}

func TestFuncLitBoundaryResetsLoopContext(t *testing.T) {
	// A hook inside an event-handler closure is in a new scope; a loop OUTSIDE the
	// closure must not flag a non-hook call, but a hook called directly in the
	// loop still must. Here the closure body has no hook → only the (none) direct
	// loop hooks count.
	parseFindings := check(t, `
func List(items []int) {
	for range items {
		_ = Button(Props{OnClick: ui.UseEvent(func() {
			doStuff() // not a hook; closure scope
		})})
	}
}`)
	// Only UseEvent (called in the loop) is flagged; doStuff is not a hook.
	if len(parseFindings) != 1 || parseFindings[0].Hook != "UseEvent" {
		t.Fatalf("expected only UseEvent, got %v", parseFindings)
	}
}

func TestHookInLoopInsideEffectClosureIsFlagged(t *testing.T) {
	// A real violation: a hook inside a loop that is itself inside an effect body.
	parseFindings := check(t, `
func Comp(items []int) {
	UseEffect(func() func() {
		for range items {
			_ = UseState(0)
		}
		return nil
	}, items)
}`)
	if len(parseFindings) != 1 || parseFindings[0].Hook != "UseState" {
		t.Fatalf("expected the inner UseState flagged, got %v", parseFindings)
	}
}

func TestNonHookUsePrefixNotFlagged(t *testing.T) {
	// "User"/"Used" are not hooks (Use must be followed by an uppercase letter).
	parseFindings := check(t, `
func Comp(items []int) {
	for range items {
		_ = User()
		_ = Used()
		_ = useThing()
	}
}`)
	if len(parseFindings) != 0 {
		t.Fatalf("Use-prefixed non-hooks should not be flagged, got %v", parseFindings)
	}
}

func TestNestedLoopsFlagOncePerHook(t *testing.T) {
	parseFindings := check(t, `
func Grid(rows, cols []int) {
	for range rows {
		for range cols {
			_ = UseState(0)
		}
	}
}`)
	if len(parseFindings) != 1 {
		t.Fatalf("expected one finding for the single hook, got %v", parseFindings)
	}
}

func TestIgnoreDirectiveSuppresses(t *testing.T) {
	// Same line.
	parseSameLine := check(t, `
func L(xs []int) {
	for range xs {
		_ = UseState(0) //hookcheck:ignore
	}
}`)
	if len(parseSameLine) != 0 {
		t.Fatalf("same-line ignore should suppress, got %v", parseSameLine)
	}
	// Line above.
	parseLineAbove := check(t, `
func L(xs []int) {
	for range xs {
		//hookcheck:ignore
		_ = UseState(0)
	}
}`)
	if len(parseLineAbove) != 0 {
		t.Fatalf("line-above ignore should suppress, got %v", parseLineAbove)
	}
	// Ignore on one hook does not suppress another.
	parseMixed := check(t, `
func L(xs []int) {
	for range xs {
		_ = UseState(0) //hookcheck:ignore
		_ = UseRef(0)
	}
}`)
	if len(parseMixed) != 1 || parseMixed[0].Hook != "UseRef" {
		t.Fatalf("only the un-ignored hook should remain, got %v", parseMixed)
	}
}

func TestCheckDirSkipsTestdataAndAggregates(t *testing.T) {
	parseDir := t.TempDir()
	parseGood := "package p\nfunc C(){ _ = UseState(0) }\n"
	parseBad := "package p\nfunc L(xs []int){ for range xs { _ = UseRef(0) } }\n"
	if parseErr := os.WriteFile(filepath.Join(parseDir, "good.go"), []byte(parseGood), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseDir, "bad.go"), []byte(parseBad), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}
	// A testdata dir must be skipped even though it contains a violation.
	parseTD := filepath.Join(parseDir, "testdata")
	if parseErr := os.MkdirAll(parseTD, 0o755); parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseTD, "skip.go"), []byte(parseBad), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}

	parseFindings, parseErr := CheckDir(parseDir)
	if parseErr != nil {
		t.Fatalf("CheckDir: %v", parseErr)
	}
	if len(parseFindings) != 1 || parseFindings[0].Hook != "UseRef" {
		t.Fatalf("expected exactly one UseRef finding (testdata skipped), got %v", parseFindings)
	}
	if !strings.Contains(parseFindings[0].String(), "bad.go") {
		t.Fatalf("finding should point at bad.go: %s", parseFindings[0])
	}
}
