package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestResolveExternalIdentityLinkDecision verifies account-linking rules across subject, email-match, and conflict cases.
func TestResolveExternalIdentityLinkDecision(parseT *testing.T) {
	parseLoginDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:          "google_oidc",
		ParseProviderSubject:      "google-subject-1",
		ParseSubjectLinkedUserIDs: []int64{42},
	})
	if parseErr != nil {
		parseT.Fatalf("subject-linked login decision: %v", parseErr)
	}
	if parseLoginDecision.ParseAction != parseExternalIdentityLinkActionLoginExisting || parseLoginDecision.ParseUserID != 42 {
		parseT.Fatalf("unexpected subject-linked decision: %+v", parseLoginDecision)
	}

	if _, parseErr = parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:          "google_oidc",
		ParseProviderSubject:      "google-subject-1",
		ParseSubjectLinkedUserIDs: []int64{42, 99},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("ambiguous subject status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-2",
		ParseVerifiedEmail:   "user@example.com",
		IsParseEmailVerified: false,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("unverified email status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	parseCreateDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-3",
		ParseVerifiedEmail:   "new-user@example.com",
		IsParseEmailVerified: true,
	})
	if parseErr != nil {
		parseT.Fatalf("create-user decision: %v", parseErr)
	}
	if parseCreateDecision.ParseAction != parseExternalIdentityLinkActionCreateUser || parseCreateDecision.ParseUserID != 0 {
		parseT.Fatalf("unexpected create-user decision: %+v", parseCreateDecision)
	}

	parseLinkDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-4",
		ParseVerifiedEmail:   "customer@email.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{ParseUserID: 1001, IsParsePasswordAuthEnabled: true},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("link-existing decision: %v", parseErr)
	}
	if parseLinkDecision.ParseAction != parseExternalIdentityLinkActionLinkExisting || parseLinkDecision.ParseUserID != 1001 || !parseLinkDecision.IsParsePasswordBootstrapLink {
		parseT.Fatalf("unexpected link-existing decision: %+v", parseLinkDecision)
	}

	if _, parseErr = parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-5",
		ParseVerifiedEmail:   "duplicate@example.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{ParseUserID: 1001},
			{ParseUserID: 2002},
		},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("duplicate email match status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-6",
		ParseVerifiedEmail:   "conflict@example.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{ParseUserID: 3003, ParseProviderLinkedSubjects: []string{"other-google-subject"}},
		},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("provider subject conflict status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestResolveExternalIdentityLinkDecisionRejectsInvalidProvider verifies unknown external providers fail closed.
func TestResolveExternalIdentityLinkDecisionRejectsInvalidProvider(parseT *testing.T) {
	_, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "unknown",
		ParseProviderSubject: "subject",
		ParseVerifiedEmail:   "user@example.com",
		IsParseEmailVerified: true,
	})
	if status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("invalid provider status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// BenchmarkResolveExternalIdentityLinkDecision measures account-link decision overhead for one verified email match.
func BenchmarkResolveExternalIdentityLinkDecision(parseB *testing.B) {
	parseRequest := parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-bench",
		ParseVerifiedEmail:   "bench@example.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{
				ParseUserID:                 77,
				IsParsePasswordAuthEnabled:  true,
				ParseProviderLinkedSubjects: nil,
			},
		},
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseRequest)
		if parseErr != nil {
			parseB.Fatalf("parseResolveExternalIdentityLinkDecision: %v", parseErr)
		}
		if parseDecision.ParseAction != parseExternalIdentityLinkActionLinkExisting {
			parseB.Fatalf("unexpected decision action: %s", parseDecision.ParseAction)
		}
	}
}
