package db

import "testing"

// TestReceivingLinesExposeExpectedActualAndClassification verifies receiving lines keep expected-vs-actual quantities with discrepancy reasons.
func TestReceivingLinesExposeExpectedActualAndClassification(parseT *testing.T) {
	parseT.Parallel()

	parseCtx, store, _ := newSeededStore(parseT)
	parseLines, parseErr := store.ReceivingLines(parseCtx, "rcv-illinois-001")
	if parseErr != nil {
		parseT.Fatalf("ReceivingLines: %v", parseErr)
	}
	if len(parseLines) == 0 {
		parseT.Fatal("expected seeded receiving lines")
	}

	parseMismatchCount := 0
	for _, parseLine := range parseLines {
		if parseLine.ExpectedQuantity < 0 || parseLine.ActualQuantity < 0 {
			parseT.Fatalf("expected non-negative quantities: %+v", parseLine)
		}
		if parseLine.ExpectedQuantity != parseLine.ActualQuantity {
			parseMismatchCount++
			if parseLine.DiscrepancyReason == "" {
				parseT.Fatalf("expected discrepancy reason for mismatch line: %+v", parseLine)
			}
		}
	}
	if parseMismatchCount == 0 {
		parseT.Fatal("expected at least one expected-vs-actual mismatch in seeded receiving lines")
	}
}

// TestReconcileReceivingSupportsWorkflowTransitions verifies receiving reconcile status transitions and default closeout behavior.
func TestReconcileReceivingSupportsWorkflowTransitions(parseT *testing.T) {
	parseT.Parallel()

	parseCtx, store, _ := newSeededStore(parseT)

	parseInReview, parseErr := store.ReconcileReceiving(parseCtx, "rcv-nevada-001", ReconcileReceivingInput{
		Status:             "in_review",
		DiscrepancySummary: "Awaiting recount before closeout.",
	})
	if parseErr != nil {
		parseT.Fatalf("ReconcileReceiving(in_review): %v", parseErr)
	}
	if parseInReview.Status != "in_review" {
		parseT.Fatalf("expected in_review status, got %q", parseInReview.Status)
	}

	parseClosed, parseErr := store.ReconcileReceiving(parseCtx, "rcv-nevada-001", ReconcileReceivingInput{
		Status:             "closed",
		DiscrepancySummary: "Recount complete and discrepancy resolved.",
	})
	if parseErr != nil {
		parseT.Fatalf("ReconcileReceiving(closed): %v", parseErr)
	}
	if parseClosed.Status != "closed" {
		parseT.Fatalf("expected closed status, got %q", parseClosed.Status)
	}

	parseDefaultClosed, parseErr := store.ReconcileReceiving(parseCtx, "rcv-nevada-001", ReconcileReceivingInput{
		Status:             "",
		DiscrepancySummary: "Default closeout path.",
	})
	if parseErr != nil {
		parseT.Fatalf("ReconcileReceiving(default): %v", parseErr)
	}
	if parseDefaultClosed.Status != "closed" {
		parseT.Fatalf("expected default closed status, got %q", parseDefaultClosed.Status)
	}
}
