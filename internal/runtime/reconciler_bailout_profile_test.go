package runtime

import "testing"

// BenchmarkSanitizeFiberSubtreeStableRows measures the clean-subtree bailout
// on a wide list whose rows each own an unchanged nested branch.
func BenchmarkSanitizeFiberSubtreeStableRows(parseB *testing.B) {
	parseRootA := &Fiber{}
	parseRootB := &Fiber{}
	var parsePrev *Fiber
	for range 128 {
		parseRow := &Fiber{parent: parseRootA}
		parseRoot := parseRow
		for range 8 {
			parseChild := &Fiber{parent: parseRoot}
			parseRoot.child = parseChild
			parseRoot = parseChild
		}
		if parsePrev == nil {
			parseRootA.child = parseRow
		} else {
			parsePrev.sibling = parseRow
		}
		parsePrev = parseRow
	}

	parseRt := &Runtime{}
	parseCurrent := parseRootA
	parseNext := parseRootB
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		parseNext.alternate = parseCurrent
		parseRt.reuseFiberChildSubtree(parseNext)
		parseCurrent, parseNext = parseNext, parseCurrent
	}
}

// TestSanitizeFiberSubtreeStopsAtRepairedCleanBoundary checks that relinking a
// clean row does not disturb its already-correct descendants or hook owners.
func TestSanitizeFiberSubtreeStopsAtRepairedCleanBoundary(parseT *testing.T) {
	parseHooks := &Hooks{}
	parseGrandchild := &Fiber{}
	parseChild := &Fiber{child: parseGrandchild, hooks: parseHooks}
	parseHooks.owner = parseChild
	parseGrandchild.parent = parseChild
	parseOldParent := &Fiber{child: parseChild}
	parseChild.parent = parseOldParent
	parseNewParent := &Fiber{alternate: parseOldParent}

	(&Runtime{}).reuseFiberChildSubtree(parseNewParent)

	if parseChild.parent != parseNewParent {
		parseT.Fatal("clean child was not relinked to the work-in-progress parent")
	}
	if parseGrandchild.parent != parseChild {
		parseT.Fatal("clean descendant parent link changed")
	}
	if parseHooks.owner != parseChild {
		parseT.Fatal("clean child hook owner changed")
	}
}

// TestSanitizeFiberSubtreeVisitsDirtyBoundary checks that dirty descendants
// still force traversal through a shallow relink.
func TestSanitizeFiberSubtreeVisitsDirtyBoundary(parseT *testing.T) {
	parseGrandchild := &Fiber{dirty: true}
	parseChild := &Fiber{child: parseGrandchild, subtreeDirty: true}
	parseGrandchild.parent = parseChild
	parseOldParent := &Fiber{child: parseChild}
	parseChild.parent = parseOldParent
	parseNewParent := &Fiber{}

	(&Runtime{}).sanitizeFiberSubtree(parseChild, parseNewParent)

	if parseChild.parent != parseNewParent {
		parseT.Fatal("dirty child was not relinked to the work-in-progress parent")
	}
	if parseGrandchild.dirty {
		parseT.Fatal("dirty descendant was skipped during sanitization")
	}
}
