package app

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseGoogleOIDCProviderKey = "google_oidc"
const parseExternalOIDCProviderType = "oidc"
const parseExternalOIDCStateTTL = 10 * time.Minute
const parseExternalOIDCBootstrapPasswordHash = "external-auth-oidc"

type parseOIDCStartRequest struct {
	ParseProviderKey     string
	ParseWorkspaceID     int64
	ParseReturnToURL     string
	ParseSessionKey      string
	ParseExpectedSubject string
	ParseCreatedByUserID int64
}

type parseOIDCStartResponse struct {
	ParseProviderKey string
	ParseStateToken  string
	ParseNonceToken  string
	ParseReturnToURL string
	ParseSessionKey  string
	ParseExpiresAt   string
}

type parseOIDCCallbackRequest struct {
	ParseProviderKey   string
	ParseStateToken    string
	ParseNonceToken    string
	ParseReturnToURL   string
	ParseSessionKey    string
	ParseProviderSub   string
	ParseVerifiedEmail string
	ParseDisplayName   string
	ParseProfileJSON   string
	IsEmailVerified    bool
	ParseNow           time.Time
}

type parseOIDCCallbackResponse struct {
	ParseAuthToken     string
	ParseUser          authUser
	ParseDecision      string
	IsParseUserLinked  bool
	IsParseUserCreated bool
}

type parseGoogleOIDCStartRequest struct {
	ParseWorkspaceID     int64
	ParseReturnToURL     string
	ParseSessionKey      string
	ParseExpectedSubject string
	ParseCreatedByUserID int64
}

type parseGoogleOIDCStartResponse struct {
	ParseProviderKey string
	ParseStateToken  string
	ParseNonceToken  string
	ParseReturnToURL string
	ParseSessionKey  string
	ParseExpiresAt   string
}

type parseGoogleOIDCCallbackRequest struct {
	ParseStateToken      string
	ParseNonceToken      string
	ParseReturnToURL     string
	ParseSessionKey      string
	ParseProviderSub     string
	ParseVerifiedEmail   string
	ParseDisplayName     string
	ParseProfileJSON     string
	IsParseEmailVerified bool
	ParseNow             time.Time
}

type parseGoogleOIDCCallbackResponse struct {
	ParseAuthToken     string
	ParseUser          authUser
	ParseDecision      string
	IsParseUserLinked  bool
	IsParseUserCreated bool
}

type parseExternalIdentityUserResolutionRequest struct {
	ParseVerifiedEmail string
	ParseDisplayName   string
	IsEmailVerified    bool
}

// parseHandleOIDCProviderStart validates and stores one OIDC handshake start state for one supported provider key.
func (parseS *chatServer) parseHandleOIDCProviderStart(parseRequest parseOIDCStartRequest) (parseOIDCStartResponse, error) {
	if parseS == nil || parseS.store == nil {
		return parseOIDCStartResponse{}, status.Error(codes.Internal, "oidc start unavailable")
	}
	parseProviderKey := parseNormalizeOIDCProviderKey(parseRequest.ParseProviderKey)
	if parseProviderKey == "" {
		return parseOIDCStartResponse{}, status.Error(codes.InvalidArgument, "oidc start provider key is required")
	}
	if parseRequest.ParseWorkspaceID <= 0 {
		return parseOIDCStartResponse{}, status.Error(codes.InvalidArgument, "oidc start workspace id is required")
	}
	if parseRequest.ParseCreatedByUserID <= 0 {
		return parseOIDCStartResponse{}, status.Error(codes.InvalidArgument, "oidc start creator user id is required")
	}
	parseSessionKey := strings.TrimSpace(parseRequest.ParseSessionKey)
	if parseSessionKey == "" {
		parseSessionKey = parseBuildOpaqueAuthFlowToken()
	}
	parseStartDecision, parseErr := parseAuthorizeExternalHandshakeStart(parseExternalHandshakeStart{
		ParseProviderKey: parseProviderKey,
		ParseReturnToURL: parseRequest.ParseReturnToURL,
		ParseSessionKey:  parseSessionKey,
	})
	if parseErr != nil {
		return parseOIDCStartResponse{}, parseErr
	}
	parseNow := time.Now().UTC()
	parseStateToken := parseBuildOpaqueAuthFlowToken()
	parseNonceToken := parseBuildOpaqueAuthFlowToken()
	parseExpiresAt := parseNow.Add(parseExternalOIDCStateTTL)
	if _, parseErr = parseS.store.parseCreateAuthOIDCState(parseAuthOIDCStateWrite{
		ProviderKey:     parseProviderKey,
		WorkspaceID:     parseRequest.ParseWorkspaceID,
		SessionKey:      parseSessionKey,
		StateTokenHash:  parseBuildAuthFlowTokenHash(parseStateToken),
		NonceTokenHash:  parseBuildAuthFlowTokenHash(parseNonceToken),
		ReturnToURL:     parseStartDecision.ParseReturnToURL,
		ExpectedSubject: strings.TrimSpace(parseRequest.ParseExpectedSubject),
		ExpiresAt:       parseExpiresAt.Format(time.RFC3339),
		CreatedByUserID: parseRequest.ParseCreatedByUserID,
	}); parseErr != nil {
		if errors.Is(parseErr, errStoreSuperuserScopeMissing) {
			return parseOIDCStartResponse{}, status.Error(codes.NotFound, "oidc start scope not found")
		}
		return parseOIDCStartResponse{}, status.Errorf(codes.Internal, "oidc start store write failed: %v", parseErr)
	}
	parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
		ParseActorUserID: parseRequest.ParseCreatedByUserID,
		ParseWorkspaceID: parseRequest.ParseWorkspaceID,
		ParseEventType:   parseExternalAuthAuditEventLoginStart,
		ParseTargetType:  "auth_oidc_state",
		ParseTargetID:    parseBuildAuthFlowTokenHash(parseStateToken),
		ParseSummary:     "External auth login started",
		ParsePayloadJSON: "{}",
	})
	return parseOIDCStartResponse{
		ParseProviderKey: parseProviderKey,
		ParseStateToken:  parseStateToken,
		ParseNonceToken:  parseNonceToken,
		ParseReturnToURL: parseStartDecision.ParseReturnToURL,
		ParseSessionKey:  parseSessionKey,
		ParseExpiresAt:   parseExpiresAt.Format(time.RFC3339),
	}, nil
}

// parseHandleGoogleOIDCStart validates and stores one Google OIDC handshake start state.
func (parseS *chatServer) parseHandleGoogleOIDCStart(parseRequest parseGoogleOIDCStartRequest) (parseGoogleOIDCStartResponse, error) {
	parseResponse, parseErr := parseS.parseHandleOIDCProviderStart(parseOIDCStartRequest{
		ParseProviderKey:     parseGoogleOIDCProviderKey,
		ParseWorkspaceID:     parseRequest.ParseWorkspaceID,
		ParseReturnToURL:     parseRequest.ParseReturnToURL,
		ParseSessionKey:      parseRequest.ParseSessionKey,
		ParseExpectedSubject: parseRequest.ParseExpectedSubject,
		ParseCreatedByUserID: parseRequest.ParseCreatedByUserID,
	})
	if parseErr != nil {
		return parseGoogleOIDCStartResponse{}, parseErr
	}
	return parseGoogleOIDCStartResponse{
		ParseProviderKey: parseResponse.ParseProviderKey,
		ParseStateToken:  parseResponse.ParseStateToken,
		ParseNonceToken:  parseResponse.ParseNonceToken,
		ParseReturnToURL: parseResponse.ParseReturnToURL,
		ParseSessionKey:  parseResponse.ParseSessionKey,
		ParseExpiresAt:   parseResponse.ParseExpiresAt,
	}, nil
}

// parseHandleOIDCProviderCallback validates one callback, resolves identity link-or-create, and issues one auth session token.
func (parseS *chatServer) parseHandleOIDCProviderCallback(parseCtx context.Context, parseRequest parseOIDCCallbackRequest) (parseOIDCCallbackResponse, error) {
	if parseS == nil || parseS.store == nil || parseS.authManager == nil {
		return parseOIDCCallbackResponse{}, status.Error(codes.Internal, "oidc callback unavailable")
	}
	parseProviderKey := parseNormalizeOIDCProviderKey(parseRequest.ParseProviderKey)
	if parseProviderKey == "" {
		return parseOIDCCallbackResponse{}, status.Error(codes.InvalidArgument, "oidc callback provider key is required")
	}
	parseAuthMethod := parseResolveOIDCAuthMethod(parseProviderKey)
	if parseAuthMethod == "" {
		return parseOIDCCallbackResponse{}, status.Error(codes.InvalidArgument, "oidc callback provider auth method is unsupported")
	}

	parseStateTokenHash := parseBuildAuthFlowTokenHash(parseRequest.ParseStateToken)
	parseNonceTokenHash := parseBuildAuthFlowTokenHash(parseRequest.ParseNonceToken)
	parseTrackCallbackFailure := func(parseActorUserID, parseWorkspaceID int64, parseSummary string) {
		parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
			ParseActorUserID: parseActorUserID,
			ParseWorkspaceID: parseWorkspaceID,
			ParseEventType:   parseExternalAuthAuditEventCallbackFailure,
			ParseTargetType:  "auth_oidc_state",
			ParseTargetID:    parseStateTokenHash,
			ParseSummary:     parseSummary,
			ParsePayloadJSON: "{}",
		})
	}
	parseStateRow, isParseFound, parseErr := parseS.store.parseGetAuthOIDCStateByStateTokenHash(parseStateTokenHash)
	if parseErr != nil {
		parseTrackCallbackFailure(0, 0, "External auth callback state lookup failed")
		return parseOIDCCallbackResponse{}, status.Errorf(codes.Internal, "oidc callback state lookup failed: %v", parseErr)
	}
	if !isParseFound {
		parseTrackCallbackFailure(0, 0, "External auth callback state not found")
		return parseOIDCCallbackResponse{}, status.Error(codes.PermissionDenied, "oidc callback state not found")
	}
	if _, parseErr = parseAuthorizeExternalHandshakeCallback(parseExternalHandshakeState{
		ParseProviderKey:     parseStateRow.ProviderKey,
		ParseStateToken:      parseStateRow.StateTokenHash,
		ParseNonceToken:      parseStateRow.NonceTokenHash,
		ParseReturnToURL:     parseStateRow.ReturnToURL,
		ParseSessionKey:      parseStateRow.SessionKey,
		ParseBoundSubject:    parseStateRow.ExpectedSubject,
		ParseExpiresAt:       parseParseRFC3339(parseStateRow.ExpiresAt),
		ParseConsumedAt:      parseParseRFC3339(parseStateRow.ConsumedAt),
		IsParseReplayBlocked: true,
	}, parseExternalHandshakeCallback{
		ParseProviderKey: parseProviderKey,
		ParseStateToken:  parseStateTokenHash,
		ParseNonceToken:  parseNonceTokenHash,
		ParseReturnToURL: parseRequest.ParseReturnToURL,
		ParseSessionKey:  parseRequest.ParseSessionKey,
		ParseSubject:     parseRequest.ParseProviderSub,
		ParseNow:         parseRequest.ParseNow,
	}); parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback handshake validation failed")
		return parseOIDCCallbackResponse{}, parseErr
	}
	if isParseConsumed, parseErr := parseS.store.parseConsumeAuthOIDCStateByStateTokenHash(parseStateTokenHash, parseRequest.ParseNow); parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback consume failed")
		return parseOIDCCallbackResponse{}, status.Errorf(codes.Internal, "oidc callback consume failed: %v", parseErr)
	} else if !isParseConsumed {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback replay detected")
		return parseOIDCCallbackResponse{}, status.Error(codes.PermissionDenied, "oidc callback replay detected")
	}

	parseSubjectLinkedUserIDs := make([]int64, 0, 1)
	if parseIdentityRow, isParseIdentityFound, parseErr := parseS.store.parseGetAuthIdentityByProviderSubject(parseProviderKey, parseRequest.ParseProviderSub); parseErr != nil {
		return parseOIDCCallbackResponse{}, status.Errorf(codes.Internal, "oidc callback identity lookup failed: %v", parseErr)
	} else if isParseIdentityFound {
		parseSubjectLinkedUserIDs = append(parseSubjectLinkedUserIDs, parseIdentityRow.UserID)
	}
	parseEmailMatches, parseErr := parseS.parseBuildExternalIdentityEmailMatches(parseProviderKey, parseRequest.ParseVerifiedEmail, parseRequest.IsEmailVerified)
	if parseErr != nil {
		return parseOIDCCallbackResponse{}, parseErr
	}
	parseWorkspacePolicy, parseErr := parseS.parseResolveWorkspaceAuthPolicy(parseCtx, parseStateRow.WorkspaceID)
	if parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback workspace policy resolution failed")
		return parseOIDCCallbackResponse{}, parseErr
	}
	isParseWorkspacePolicyEnforced := parseStateRow.WorkspaceID > 0 && parseWorkspacePolicy.ParsePolicySource == "workspace"
	if isParseWorkspacePolicyEnforced {
		parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
			ParseActorUserID: parseStateRow.CreatedByUserID,
			ParseWorkspaceID: parseStateRow.WorkspaceID,
			ParseEventType:   parseExternalAuthAuditEventWorkspaceSSOEnforcement,
			ParseTargetType:  "workspace",
			ParseTargetID:    strconv.FormatInt(parseStateRow.WorkspaceID, 10),
			ParseSummary:     "Workspace SSO policy evaluated for external auth callback",
			ParsePayloadJSON: "{}",
		})
	}
	parseLinkDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:           parseProviderKey,
		ParseProviderSubject:       parseRequest.ParseProviderSub,
		ParseVerifiedEmail:         parseRequest.ParseVerifiedEmail,
		IsParseEmailVerified:       parseRequest.IsEmailVerified,
		IsParsePolicyEnforced:      isParseWorkspacePolicyEnforced,
		IsParsePasswordLinkAllowed: parseWorkspacePolicy.IsParsePasswordAllowed || parseWorkspacePolicy.IsParseLocalPasswordQAModeAllowed,
		IsParseUserCreateAllowed:   parseWorkspacePolicy.IsParseJITProvisioningAllowed,
		ParseSubjectLinkedUserIDs:  parseSubjectLinkedUserIDs,
		ParseEmailMatchedUsers:     parseEmailMatches,
	})
	if parseErr != nil {
		if isParseWorkspacePolicyEnforced && status.Code(parseErr) == codes.PermissionDenied {
			parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
				ParseActorUserID: parseStateRow.CreatedByUserID,
				ParseWorkspaceID: parseStateRow.WorkspaceID,
				ParseEventType:   parseExternalAuthAuditEventPolicyDeniedLogin,
				ParseTargetType:  "workspace",
				ParseTargetID:    strconv.FormatInt(parseStateRow.WorkspaceID, 10),
				ParseSummary:     "Workspace policy denied external auth login",
				ParsePayloadJSON: "{}",
			})
		}
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback link decision denied")
		return parseOIDCCallbackResponse{}, parseErr
	}
	parseResolvedUserID, isParseUserCreated, isParseUserLinked, parseErr := parseS.parseResolveExternalIdentityUser(parseLinkDecision, parseExternalIdentityUserResolutionRequest{
		ParseVerifiedEmail: parseRequest.ParseVerifiedEmail,
		ParseDisplayName:   parseRequest.ParseDisplayName,
		IsEmailVerified:    parseRequest.IsEmailVerified,
	})
	if parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback user resolution failed")
		return parseOIDCCallbackResponse{}, parseErr
	}
	parseResolvedEmail, parseErr := parseS.parseResolveUserEmailByIdentity(parseResolvedUserID, parseProviderKey, parseRequest.ParseVerifiedEmail)
	if parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback email resolution failed")
		return parseOIDCCallbackResponse{}, parseErr
	}
	if _, parseErr = parseS.store.parseUpsertAuthIdentity(parseAuthIdentityWrite{
		UserID:          parseResolvedUserID,
		ProviderKey:     parseProviderKey,
		ProviderType:    parseExternalOIDCProviderType,
		ProviderSubject: parseRequest.ParseProviderSub,
		Email:           parseResolvedEmail,
		IsEmailVerified: parseRequest.IsEmailVerified,
		ProfileJSON:     parseNormalizeExternalProfileJSON(parseRequest.ParseProfileJSON),
		LastLoginAt:     time.Now().UTC().Format(time.RFC3339),
	}); parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback identity upsert failed")
		return parseOIDCCallbackResponse{}, status.Errorf(codes.Internal, "oidc callback identity upsert failed: %v", parseErr)
	}
	if isParseUserLinked {
		parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
			ParseActorUserID: parseResolvedUserID,
			ParseWorkspaceID: parseStateRow.WorkspaceID,
			ParseEventType:   parseExternalAuthAuditEventIdentityLinked,
			ParseTargetType:  "user",
			ParseTargetID:    parseBuildExternalAuthAuditTargetIDFromUser(parseResolvedUserID),
			ParseSummary:     "External identity linked to existing user",
			ParsePayloadJSON: "{}",
		})
	}
	parseAuthToken, parseErr := parseS.authManager.issueTokenForContextWithAuthMethod(parseCtx, authUser{
		ID:    parseResolvedUserID,
		Email: parseResolvedEmail,
	}, parseStateRow.SessionKey, parseAuthMethod)
	if parseErr != nil {
		parseTrackCallbackFailure(parseStateRow.CreatedByUserID, parseStateRow.WorkspaceID, "External auth callback token issuance failed")
		return parseOIDCCallbackResponse{}, status.Errorf(codes.Internal, "oidc callback auth token issue failed: %v", parseErr)
	}
	parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
		ParseActorUserID: parseResolvedUserID,
		ParseWorkspaceID: parseStateRow.WorkspaceID,
		ParseEventType:   parseExternalAuthAuditEventCallbackSuccess,
		ParseTargetType:  "user",
		ParseTargetID:    parseBuildExternalAuthAuditTargetIDFromUser(parseResolvedUserID),
		ParseSummary:     "External auth callback succeeded",
		ParsePayloadJSON: "{}",
	})
	return parseOIDCCallbackResponse{
		ParseAuthToken:     parseAuthToken,
		ParseUser:          authUser{ID: parseResolvedUserID, Email: parseResolvedEmail},
		ParseDecision:      parseLinkDecision.ParseAction,
		IsParseUserLinked:  isParseUserLinked,
		IsParseUserCreated: isParseUserCreated,
	}, nil
}

// parseHandleGoogleOIDCCallback validates one callback, resolves identity link-or-create, and issues one auth session token.
func (parseS *chatServer) parseHandleGoogleOIDCCallback(parseCtx context.Context, parseRequest parseGoogleOIDCCallbackRequest) (parseGoogleOIDCCallbackResponse, error) {
	parseResponse, parseErr := parseS.parseHandleOIDCProviderCallback(parseCtx, parseOIDCCallbackRequest{
		ParseProviderKey:   parseGoogleOIDCProviderKey,
		ParseStateToken:    parseRequest.ParseStateToken,
		ParseNonceToken:    parseRequest.ParseNonceToken,
		ParseReturnToURL:   parseRequest.ParseReturnToURL,
		ParseSessionKey:    parseRequest.ParseSessionKey,
		ParseProviderSub:   parseRequest.ParseProviderSub,
		ParseVerifiedEmail: parseRequest.ParseVerifiedEmail,
		ParseDisplayName:   parseRequest.ParseDisplayName,
		ParseProfileJSON:   parseRequest.ParseProfileJSON,
		IsEmailVerified:    parseRequest.IsParseEmailVerified,
		ParseNow:           parseRequest.ParseNow,
	})
	if parseErr != nil {
		return parseGoogleOIDCCallbackResponse{}, parseErr
	}
	return parseGoogleOIDCCallbackResponse{
		ParseAuthToken:     parseResponse.ParseAuthToken,
		ParseUser:          parseResponse.ParseUser,
		ParseDecision:      parseResponse.ParseDecision,
		IsParseUserLinked:  parseResponse.IsParseUserLinked,
		IsParseUserCreated: parseResponse.IsParseUserCreated,
	}, nil
}

// parseBuildExternalIdentityEmailMatches builds one email-match set for external identity linking.
func (parseS *chatServer) parseBuildExternalIdentityEmailMatches(parseProviderKey, parseVerifiedEmail string, isParseEmailVerified bool) ([]parseExternalIdentityEmailMatch, error) {
	parseVerifiedEmail = parseNormalizeAuthEmail(parseVerifiedEmail)
	if !isParseEmailVerified || parseVerifiedEmail == "" {
		return []parseExternalIdentityEmailMatch{}, nil
	}
	parseAuthRecord, parseErr := parseS.store.getUserAuthByEmail(parseVerifiedEmail)
	if parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return []parseExternalIdentityEmailMatch{}, nil
		}
		return nil, status.Errorf(codes.Internal, "external identity email match lookup failed: %v", parseErr)
	}
	parseIdentityRows, parseErr := parseS.store.parseListAuthIdentitiesByUser(parseAuthRecord.ID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "external identity linked-subject lookup failed: %v", parseErr)
	}
	parseProviderSubjects := make([]string, 0, len(parseIdentityRows))
	for _, parseIdentityRow := range parseIdentityRows {
		if parseNormalizeAuthProviderKey(parseIdentityRow.ProviderKey) != parseNormalizeAuthProviderKey(parseProviderKey) {
			continue
		}
		parseProviderSubjects = append(parseProviderSubjects, parseIdentityRow.ProviderSubject)
	}
	return []parseExternalIdentityEmailMatch{{
		ParseUserID:                 parseAuthRecord.ID,
		IsParsePasswordAuthEnabled:  strings.TrimSpace(parseAuthRecord.PasswordHash) != "",
		ParseProviderLinkedSubjects: parseProviderSubjects,
	}}, nil
}

// parseResolveExternalIdentityUser resolves one auth user ID from one link-or-create decision and callback payload.
func (parseS *chatServer) parseResolveExternalIdentityUser(parseDecision parseExternalIdentityLinkDecision, parseRequest parseExternalIdentityUserResolutionRequest) (int64, bool, bool, error) {
	switch parseDecision.ParseAction {
	case parseExternalIdentityLinkActionLoginExisting:
		if parseDecision.ParseUserID <= 0 {
			return 0, false, false, status.Error(codes.PermissionDenied, "external identity login target is missing")
		}
		return parseDecision.ParseUserID, false, false, nil
	case parseExternalIdentityLinkActionLinkExisting:
		if parseDecision.ParseUserID <= 0 {
			return 0, false, false, status.Error(codes.PermissionDenied, "external identity link target is missing")
		}
		return parseDecision.ParseUserID, false, true, nil
	case parseExternalIdentityLinkActionCreateUser:
		parseVerifiedEmail := parseNormalizeAuthEmail(parseRequest.ParseVerifiedEmail)
		if parseVerifiedEmail == "" || !parseRequest.IsEmailVerified {
			return 0, false, false, status.Error(codes.PermissionDenied, "verified email is required for external identity user creation")
		}
		parseDisplayName := parseResolveExternalDisplayName(parseRequest.ParseDisplayName, parseVerifiedEmail)
		parseUserID, parseErr := parseS.store.parseCreateUser(parseVerifiedEmail, parseExternalOIDCBootstrapPasswordHash, parseDisplayName)
		if parseErr != nil {
			if errors.Is(parseErr, errUserAlreadyExists) {
				parseAuthRecord, parseLookupErr := parseS.store.getUserAuthByEmail(parseVerifiedEmail)
				if parseLookupErr != nil {
					return 0, false, false, status.Errorf(codes.Internal, "external identity create-user lookup failed: %v", parseLookupErr)
				}
				return parseAuthRecord.ID, false, true, nil
			}
			return 0, false, false, status.Errorf(codes.Internal, "external identity create-user failed: %v", parseErr)
		}
		return parseUserID, true, false, nil
	default:
		return 0, false, false, status.Error(codes.PermissionDenied, "external identity link decision rejected")
	}
}

// parseResolveUserEmailByIdentity resolves one user email with identity-first fallback for token issuance.
func (parseS *chatServer) parseResolveUserEmailByIdentity(parseUserID int64, parseProviderKey, parseFallbackEmail string) (string, error) {
	parseIdentityRows, parseErr := parseS.store.parseListAuthIdentitiesByUser(parseUserID)
	if parseErr != nil {
		return "", status.Errorf(codes.Internal, "external identity email resolution failed: %v", parseErr)
	}
	for _, parseIdentityRow := range parseIdentityRows {
		if parseNormalizeAuthProviderKey(parseIdentityRow.ProviderKey) != parseNormalizeAuthProviderKey(parseProviderKey) {
			continue
		}
		if parseIdentityEmail := parseNormalizeAuthEmail(parseIdentityRow.Email); parseIdentityEmail != "" {
			return parseIdentityEmail, nil
		}
	}
	parseFallbackEmail = parseNormalizeAuthEmail(parseFallbackEmail)
	if parseFallbackEmail == "" {
		return "", status.Error(codes.PermissionDenied, "external identity email is missing")
	}
	return parseFallbackEmail, nil
}

// parseNormalizeOIDCProviderKey normalizes one OIDC provider key into one supported runtime value.
func parseNormalizeOIDCProviderKey(parseProviderKey string) string {
	parseProviderKey = parseNormalizeExternalIdentityProviderKey(parseProviderKey)
	switch parseProviderKey {
	case parseWorkspaceAuthMethodGoogleOIDC, parseWorkspaceAuthMethodOIDC:
		return parseProviderKey
	default:
		return ""
	}
}

// parseResolveOIDCAuthMethod resolves one token auth-method claim from one OIDC provider key.
func parseResolveOIDCAuthMethod(parseProviderKey string) string {
	switch parseNormalizeOIDCProviderKey(parseProviderKey) {
	case parseWorkspaceAuthMethodGoogleOIDC:
		return parseWorkspaceAuthMethodGoogleOIDC
	case parseWorkspaceAuthMethodOIDC:
		return parseWorkspaceAuthMethodOIDC
	default:
		return ""
	}
}

// parseResolveExternalDisplayName resolves one safe display name for new external-auth users.
func parseResolveExternalDisplayName(parseDisplayName, parseFallbackEmail string) string {
	parseDisplayName = strings.TrimSpace(parseDisplayName)
	if parseDisplayName != "" {
		return parseDisplayName
	}
	return parseDefaultDisplayNameFromEmail(parseFallbackEmail)
}

// parseNormalizeExternalProfileJSON normalizes profile payload JSON with one safe empty-object fallback.
func parseNormalizeExternalProfileJSON(parseProfileJSON string) string {
	parseProfileJSON = strings.TrimSpace(parseProfileJSON)
	if parseProfileJSON == "" {
		return "{}"
	}
	return parseProfileJSON
}

// parseParseRFC3339 parses one RFC3339 timestamp and returns a zero time on parse failure.
func parseParseRFC3339(parseTimestamp string) time.Time {
	parseTimestamp = strings.TrimSpace(parseTimestamp)
	if parseTimestamp == "" {
		return time.Time{}
	}
	parseParsedTimestamp, parseErr := time.Parse(time.RFC3339, parseTimestamp)
	if parseErr != nil {
		return time.Time{}
	}
	return parseParsedTimestamp.UTC()
}
