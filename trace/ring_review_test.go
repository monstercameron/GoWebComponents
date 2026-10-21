package trace

import (
	"fmt"
	"math/rand"
	"testing"
)

// TestRecorderRingMatchesReferenceModel drives random Start/End sequences through a small ring and
// compares every observable (resident spans oldest-first, Dropped, End errors for evicted or
// closed spans) against a trivial slice-based reference model.
func TestRecorderRingMatchesReferenceModel(parseT *testing.T) {
	for parseSeed := int64(1); parseSeed <= 200; parseSeed++ {
		parseRand := rand.New(rand.NewSource(parseSeed))
		parseCapacity := 1 + parseRand.Intn(6)
		parseRecorder, parseErr := NewRecorder(ThreadRender, parseCapacity)
		if parseErr != nil {
			parseT.Fatal(parseErr)
		}

		type refSpan struct {
			id     SpanID
			ended  bool
			endAt  int64
			errStr string
		}
		var parseModel []refSpan
		parseDropped := 0
		var parseAllIDs []SpanID

		for parseStep := 0; parseStep < 80; parseStep++ {
			if parseRand.Intn(3) != 0 || len(parseAllIDs) == 0 {
				parseID, parseStartErr := parseRecorder.Start("op", CorrelationID("c1"), "", int64(parseStep+1))
				if parseStartErr != nil {
					parseT.Fatal(parseStartErr)
				}
				parseAllIDs = append(parseAllIDs, parseID)
				parseModel = append(parseModel, refSpan{id: parseID})
				if len(parseModel) > parseCapacity {
					parseModel = parseModel[1:]
					parseDropped++
				}
				continue
			}
			parseTarget := parseAllIDs[parseRand.Intn(len(parseAllIDs))]
			parseEndErr := parseRecorder.End(parseTarget, int64(1000+parseStep), fmt.Sprintf("e%d", parseStep))
			parseIndex := -1
			for parseI := range parseModel {
				if parseModel[parseI].id == parseTarget {
					parseIndex = parseI
				}
			}
			switch {
			case parseIndex < 0:
				if parseEndErr == nil {
					parseT.Fatalf("seed %d: End of evicted span %s succeeded", parseSeed, parseTarget)
				}
			case parseModel[parseIndex].ended:
				if parseEndErr == nil {
					parseT.Fatalf("seed %d: second End of %s succeeded", parseSeed, parseTarget)
				}
			default:
				if parseEndErr != nil {
					parseT.Fatalf("seed %d: End of resident open span %s failed: %v", parseSeed, parseTarget, parseEndErr)
				}
				parseModel[parseIndex].ended = true
				parseModel[parseIndex].endAt = int64(1000 + parseStep)
				parseModel[parseIndex].errStr = fmt.Sprintf("e%d", parseStep)
			}
		}

		parseSpans := parseRecorder.Spans()
		if len(parseSpans) != len(parseModel) {
			parseT.Fatalf("seed %d: %d resident spans, model has %d", parseSeed, len(parseSpans), len(parseModel))
		}
		for parseI, parseSpan := range parseSpans {
			parseWant := parseModel[parseI]
			if parseSpan.ID != parseWant.id || (parseWant.ended && (parseSpan.EndNanos != parseWant.endAt || parseSpan.Err != parseWant.errStr)) || (!parseWant.ended && !parseSpan.Open()) {
				parseT.Fatalf("seed %d: span %d = %+v, model %+v", parseSeed, parseI, parseSpan, parseWant)
			}
		}
		if parseRecorder.Dropped() != parseDropped {
			parseT.Fatalf("seed %d: Dropped=%d, model %d", parseSeed, parseRecorder.Dropped(), parseDropped)
		}
		if parseMerged := Merge(parseRecorder); parseMerged.Dropped() != parseDropped {
			parseT.Fatalf("seed %d: merged Dropped=%d, want %d", parseSeed, parseMerged.Dropped(), parseDropped)
		}
	}
}

// TestLifecycleOrdersParentsBeforeChildrenAcrossThreadsAndSurvivesCycles pins causal ordering with
// per-thread sequence ties and a malformed parent cycle (must terminate).
func TestLifecycleOrdersParentsBeforeChildrenAcrossThreadsAndSurvivesCycles(parseT *testing.T) {
	parseRender, _ := NewRecorder(ThreadRender, 8)
	parseDomain, _ := NewRecorder(ThreadDomain, 8)
	parseRoot, _ := parseRender.Start("send", "cmd-1", "", 500)
	parseChild, _ := parseDomain.Start("handle", "cmd-1", parseRoot, 3) // earlier clock, later cause
	parseGrand, _ := parseRender.Start("apply", "cmd-1", parseChild, 600)
	_ = parseRender.End(parseRoot, 510, "")
	_ = parseDomain.End(parseChild, 9, "")
	_ = parseRender.End(parseGrand, 610, "")

	parseOrdered := Merge(parseRender, parseDomain).Lifecycle("cmd-1")
	parseNames := []string{}
	for _, parseSpan := range parseOrdered {
		parseNames = append(parseNames, parseSpan.Name)
	}
	if fmt.Sprint(parseNames) != "[send handle apply]" {
		parseT.Fatalf("causal order = %v, want [send handle apply]", parseNames)
	}

	// A two-span parent cycle must not hang or crash.
	parseCycle := causalOrder([]Span{{ID: "a", Parent: "b", Thread: ThreadRender, Seq: 1}, {ID: "b", Parent: "a", Thread: ThreadRender, Seq: 2}})
	if len(parseCycle) != 2 {
		parseT.Fatalf("cycle produced %d spans", len(parseCycle))
	}
}
