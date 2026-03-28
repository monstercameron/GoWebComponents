package app

import (
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseExternalIdentityLinkActionReject = "reject"
const parseExternalIdentityLinkActionLoginExisting = "login_existing"
const parseExternalIdentityLinkActionLinkExisting = "link_existing"
const parseExternalIdentityLinkActionCreateUser = "create_user"

type parseExternalIdentityLinkRequest struct {
	ParseProviderKey           string
	ParseProviderSubject       string
	ParseVerifiedEmail         string
	IsParseEmailVerified       bool
	IsParsePolicyEnforced      bool
	IsParsePasswordLinkAllowed bool
	IsParseUserCreateAllowed   bool
	ParseSubjectLinkedUserIDs  []int64
	ParseEmailMatchedUsers     []parseExternalIdentityEmailMatch
}

type parseExternalIdentityEmailMatch struct {
	ParseUserID                 int64
	IsParsePasswordAuthEnabled  bool
	ParseProviderLinkedSubjects []string
}

type parseExternalIdentityLinkDecision struct {
	ParseAction                  string
	ParseUserID                  int64
	ParseReason                  string
	IsParsePasswordBootstrapLink bool
}

// parseResolveExternalIdentityLinkDecision enforces canonical account-linking policy for one external identity login.
func parseResolveExternalIdentityLinkDecision(parseRequest parseExternalIdentityLinkRequest) (parseExternalIdentityLinkDecision, error) {
	parseIsPasswordLinkAllowed := true
	parseIsUserCreateAllowed := true
	if parseRequest.IsParsePolicyEnforced {
		parseIsPasswordLinkAllowed = parseRequest.IsParsePasswordLinkAllowed
		parseIsUserCreateAllowed = parseRequest.IsParseUserCreateAllowed
	}
	parseProviderKey := parseNormalizeExternalIdentityProviderKey(parseRequest.ParseProviderKey)
	if parseProviderKey == "" {
		return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.InvalidArgument, "external auth provider key is required")
	}
	parseProviderSubject := strings.TrimSpace(parseRequest.ParseProviderSubject)
	if parseProviderSubject == "" {
		return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.InvalidArgument, "external auth provider subject is required")
	}

	parseSubjectLinkedUserIDs := parseBuildExternalIdentityUniqueUserIDs(parseRequest.ParseSubjectLinkedUserIDs)
	if len(parseSubjectLinkedUserIDs) > 1 {
		return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.PermissionDenied, "external identity subject is linked to multiple users")
	}
	if len(parseSubjectLinkedUserIDs) == 1 {
		return parseExternalIdentityLinkDecision{
			ParseAction: parseExternalIdentityLinkActionLoginExisting,
			ParseUserID: parseSubjectLinkedUserIDs[0],
			ParseReason: "provider_subject_match",
		}, nil
	}

	parseVerifiedEmail := parseNormalizeAuthEmail(parseRequest.ParseVerifiedEmail)
	if !parseRequest.IsParseEmailVerified || parseVerifiedEmail == "" {
		return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.PermissionDenied, "verified email is required for account linking")
	}

	parseEmailMatchedUsers := parseBuildExternalIdentityUniqueEmailMatches(parseRequest.ParseEmailMatchedUsers)
	switch len(parseEmailMatchedUsers) {
	case 0:
		if !parseIsUserCreateAllowed {
			return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.PermissionDenied, "workspace policy blocks external identity user creation")
		}
		return parseExternalIdentityLinkDecision{
			ParseAction: parseExternalIdentityLinkActionCreateUser,
			ParseReason: "verified_email_no_match",
		}, nil
	case 1:
		parseMatchedUser := parseEmailMatchedUsers[0]
		if parseHasExternalIdentityProviderSubjectConflict(parseProviderSubject, parseMatchedUser.ParseProviderLinkedSubjects) {
			return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.PermissionDenied, "provider account conflict for matched email user")
		}
		if parseMatchedUser.IsParsePasswordAuthEnabled && !parseIsPasswordLinkAllowed {
			return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.PermissionDenied, "workspace policy blocks external identity link to password account")
		}
		return parseExternalIdentityLinkDecision{
			ParseAction:                  parseExternalIdentityLinkActionLinkExisting,
			ParseUserID:                  parseMatchedUser.ParseUserID,
			ParseReason:                  "verified_email_single_match",
			IsParsePasswordBootstrapLink: parseMatchedUser.IsParsePasswordAuthEnabled,
		}, nil
	default:
		return parseExternalIdentityLinkDecision{ParseAction: parseExternalIdentityLinkActionReject}, status.Error(codes.PermissionDenied, "verified email matches multiple users")
	}
}

// parseNormalizeExternalIdentityProviderKey normalizes one external auth provider key into one canonical value.
func parseNormalizeExternalIdentityProviderKey(parseProviderKey string) string {
	switch strings.TrimSpace(strings.ToLower(parseProviderKey)) {
	case "google_oidc", "oidc", "saml":
		return strings.TrimSpace(strings.ToLower(parseProviderKey))
	default:
		return ""
	}
}

// parseBuildExternalIdentityUniqueUserIDs returns one unique, sorted user-id set.
func parseBuildExternalIdentityUniqueUserIDs(parseRawUserIDs []int64) []int64 {
	parseUniqueUserIDs := make([]int64, 0, len(parseRawUserIDs))
	for _, parseRawUserID := range parseRawUserIDs {
		if parseRawUserID <= 0 || slices.Contains(parseUniqueUserIDs, parseRawUserID) {
			continue
		}
		parseUniqueUserIDs = append(parseUniqueUserIDs, parseRawUserID)
	}
	slices.Sort(parseUniqueUserIDs)
	return parseUniqueUserIDs
}

// parseBuildExternalIdentityUniqueEmailMatches returns one unique user match list from one email-match candidate set.
func parseBuildExternalIdentityUniqueEmailMatches(parseRawMatches []parseExternalIdentityEmailMatch) []parseExternalIdentityEmailMatch {
	parseMatches := make([]parseExternalIdentityEmailMatch, 0, len(parseRawMatches))
	parseMatchedUserIDs := make(map[int64]struct{}, len(parseRawMatches))
	for _, parseRawMatch := range parseRawMatches {
		if parseRawMatch.ParseUserID <= 0 {
			continue
		}
		if _, hasParseMatchedUser := parseMatchedUserIDs[parseRawMatch.ParseUserID]; hasParseMatchedUser {
			continue
		}
		parseMatchedUserIDs[parseRawMatch.ParseUserID] = struct{}{}
		parseMatches = append(parseMatches, parseExternalIdentityEmailMatch{
			ParseUserID:                 parseRawMatch.ParseUserID,
			IsParsePasswordAuthEnabled:  parseRawMatch.IsParsePasswordAuthEnabled,
			ParseProviderLinkedSubjects: parseBuildExternalIdentityUniqueSubjects(parseRawMatch.ParseProviderLinkedSubjects),
		})
	}
	return parseMatches
}

// parseBuildExternalIdentityUniqueSubjects returns one deduplicated normalized subject list.
func parseBuildExternalIdentityUniqueSubjects(parseRawSubjects []string) []string {
	parseSubjects := make([]string, 0, len(parseRawSubjects))
	for _, parseRawSubject := range parseRawSubjects {
		parseSubject := strings.TrimSpace(parseRawSubject)
		if parseSubject == "" || slices.Contains(parseSubjects, parseSubject) {
			continue
		}
		parseSubjects = append(parseSubjects, parseSubject)
	}
	slices.Sort(parseSubjects)
	return parseSubjects
}

// parseHasExternalIdentityProviderSubjectConflict reports whether one email-matched user already has one different subject bound for the same provider.
func parseHasExternalIdentityProviderSubjectConflict(parseProviderSubject string, parseLinkedProviderSubjects []string) bool {
	parseProviderSubject = strings.TrimSpace(parseProviderSubject)
	if parseProviderSubject == "" {
		return true
	}
	for _, parseLinkedProviderSubject := range parseBuildExternalIdentityUniqueSubjects(parseLinkedProviderSubjects) {
		if parseLinkedProviderSubject == parseProviderSubject {
			return false
		}
	}
	return len(parseLinkedProviderSubjects) > 0
}
