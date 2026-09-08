package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestValidatePublicCommentFormCoversInvalidAndValidInputs verifies public comment form validation messages and the valid happy path.
func TestValidatePublicCommentFormCoversInvalidAndValidInputs(parseT *testing.T) {
	parseT.Parallel()

	parseErrors := validatePublicCommentForm(publicCommentFormState{})
	if parseErrors["AuthorName"] == "" || parseErrors["Reaction"] == "" || parseErrors["Subject"] == "" || parseErrors["Body"] == "" {
		parseT.Fatalf("expected all required validation fields, got %#v", parseErrors)
	}

	parseValid := validatePublicCommentForm(publicCommentFormState{
		AuthorName: "Atlas Buyer",
		Reaction:   "up",
		Subject:    "Solid setup",
		Body:       "Installation was smooth and delivery timing matched the estimate.",
	})
	if len(parseValid) != 0 {
		parseT.Fatalf("expected valid form to have no errors, got %#v", parseValid)
	}
}

// TestPublicCommentReactionInputWiresLabelsAndErrors verifies ARIA label wiring and inline error rendering for reaction controls.
func TestPublicCommentReactionInputWiresLabelsAndErrors(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := renderAtlasNodeForTest(publicCommentReactionInput("atlas-reaction", "atlas-reaction-error", "down", ui.Handler{}, "Choose one."))
	if parseErr != nil {
		parseT.Fatalf("publicCommentReactionInput render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "atlas-reaction-legend") {
		parseT.Fatalf("expected legend id wiring, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `aria-labelledby="atlas-reaction-legend"`) {
		parseT.Fatalf("expected aria-labelledby wiring, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Choose one.") {
		parseT.Fatalf("expected inline reaction error message, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Thumbs up") || !strings.Contains(parseMarkup, "Thumbs down") {
		parseT.Fatalf("expected both reaction choices, got %q", parseMarkup)
	}
}

// TestPublicProductFeedbackListShowsModerationAndRefreshingStates verifies moderation badges and refresh-pending copy in the feedback module.
func TestPublicProductFeedbackListShowsModerationAndRefreshingStates(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := renderAtlasNodeForTest(publicProductFeedbackList([]commentRecord{
		{
			ID:         "cmt-1",
			AuthorName: "Atlas Buyer",
			AuthorType: "customer",
			Reaction:   "up",
			Subject:    "Great finish options",
			Body:       "Finish selection matched our palette.",
			Status:     "pending",
			CreatedAt:  "2026-03-24T10:00:00Z",
		},
		{
			ID:         "cmt-2",
			AuthorName: "Ops Review",
			AuthorType: "staff",
			Reaction:   "down",
			Subject:    "Dock damage",
			Body:       "One unit arrived with a damaged edge.",
			Status:     "flagged",
			CreatedAt:  "2026-03-25T11:00:00Z",
		},
	}, true))
	if parseErr != nil {
		parseT.Fatalf("publicProductFeedbackList render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Refreshing buyer notes...") {
		parseT.Fatalf("expected refresh pending copy, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "pending") || !strings.Contains(parseMarkup, "flagged") {
		parseT.Fatalf("expected moderation status badges, got %q", parseMarkup)
	}
}

// TestPublicCommentHelpersCoverPendingAndModerationStates verifies submit-label pending copy, field error mapping, and pending merge behavior.
func TestPublicCommentHelpersCoverPendingAndModerationStates(parseT *testing.T) {
	parseT.Parallel()

	if parseLabel := publicCommentSubmitLabel(false); parseLabel != "Share feedback" {
		parseT.Fatalf("expected default submit label, got %q", parseLabel)
	}
	if parseLabel := publicCommentSubmitLabel(true); parseLabel != "Sending..." {
		parseT.Fatalf("expected pending submit label, got %q", parseLabel)
	}

	parseMapped := normalizePublicCommentFieldErrors(ui.FieldErrors{
		"author_name": "author error",
		"reaction":    "reaction error",
		"subject":     "subject error",
		"body":        "body error",
	})
	if parseMapped["AuthorName"] == "" || parseMapped["Reaction"] == "" || parseMapped["Subject"] == "" || parseMapped["Body"] == "" {
		parseT.Fatalf("expected normalized field errors, got %#v", parseMapped)
	}

	parseMerged := mergePublicCommentList([]commentRecord{{ID: "existing"}}, commentRecord{ID: "pending-new"})
	if len(parseMerged) != 2 || parseMerged[0].ID != "pending-new" {
		parseT.Fatalf("expected pending comment to prepend list, got %#v", parseMerged)
	}
	parseDeduped := mergePublicCommentList(parseMerged, commentRecord{ID: "pending-new"})
	if len(parseDeduped) != 2 {
		parseT.Fatalf("expected duplicate pending comment to be ignored, got %#v", parseDeduped)
	}

	if parseBadge := publicCommentStatusBadge("approved"); parseBadge != nil {
		parseT.Fatal("expected approved status badge to stay hidden")
	}
	parsePendingMarkup, parseErr := renderAtlasNodeForTest(publicCommentStatusBadge("pending"))
	if parseErr != nil {
		parseT.Fatalf("pending status badge render failed: %v", parseErr)
	}
	if !strings.Contains(parsePendingMarkup, "pending") {
		parseT.Fatalf("expected pending status badge text, got %q", parsePendingMarkup)
	}
}
