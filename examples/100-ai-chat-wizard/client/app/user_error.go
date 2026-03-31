//go:build js && wasm

package app

import (
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const userErrorScopeAuth = "auth"
const userErrorScopeChat = "chat"
const userErrorScopeSettings = "settings"
const userErrorScopeDashboard = "dashboard"
const parseUserErrorInfoReason = "CUSTOMER_SAFE_ERROR_V1"
const parseUserErrorInfoDomain = "relaydesk.customer_error"
const parseUserErrorMetadataMessage = "user_message"
const parseUserErrorMetadataReferenceID = "reference_id"

type parseUserErrorContract struct {
	ParseScope       string
	ParseUserMessage string
	ParseReferenceID string
}

// parseBuildUserErrorText returns one customer-safe error string with a visible request identifier.
func parseBuildUserErrorText(parseScope string, parseErr error) string {
	parseContract, isParseContract := parseResolveUserErrorContract(parseErr)
	parseMessage := parseResolveUserErrorMessage(parseScope, parseErr)
	if isParseContract {
		if parseContractMessage := strings.TrimSpace(parseContract.ParseUserMessage); parseContractMessage != "" {
			parseMessage = parseContractMessage
		}
	}
	return parseFormatUserErrorTextWithReference(parseMessage, parseBuildUserErrorReferenceID(parseErr))
}

// parseBuildUserAuthErrorText returns one auth-specific customer-safe error string with a request identifier.
func parseBuildUserAuthErrorText(parseMode string, parseErr error) string {
	return parseFormatUserErrorTextWithReference(parseAuthErrorMessage(parseMode, parseErr), parseBuildUserErrorReferenceID(parseErr))
}

// parseResolveUserErrorMessage maps one backend/runtime failure into calm customer-facing copy.
func parseResolveUserErrorMessage(parseScope string, parseErr error) string {
	if parseContract, isParseContract := parseResolveUserErrorContract(parseErr); isParseContract {
		if parseContractMessage := strings.TrimSpace(parseContract.ParseUserMessage); parseContractMessage != "" {
			return parseContractMessage
		}
	}
	if parseErr != nil {
		if parseStatusErr, parseOk := status.FromError(parseErr); parseOk {
			switch parseStatusErr.Code() {
			case codes.Unauthenticated:
				if parseScope == userErrorScopeAuth {
					return parseAuthErrorMessage(authModeLogin, parseErr)
				}
				return authExpiredMessage
			case codes.PermissionDenied:
				return "You do not have permission to complete that action."
			case codes.Unavailable:
				return "The service is temporarily unavailable. Try again in a moment."
			case codes.DeadlineExceeded:
				return "That request took too long. Please try again."
			case codes.Canceled:
				return "The request was canceled. Please try again."
			}
		}
	}
	return parseResolveUserFallbackMessage(parseScope)
}

// parseResolveUserFallbackMessage returns one stable fallback message per product surface.
func parseResolveUserFallbackMessage(parseScope string) string {
	switch strings.TrimSpace(parseScope) {
	case userErrorScopeChat:
		return "We couldn't send your message right now. Please try again."
	case userErrorScopeSettings:
		return "We couldn't save your settings right now. Please try again."
	case userErrorScopeDashboard:
		return "Dashboard data is temporarily unavailable. Please refresh in a moment."
	default:
		return "Something went wrong. Please try again."
	}
}

// parseFormatUserErrorText appends one visible request identifier to customer-facing error copy.
func parseFormatUserErrorText(parseMessage string) string {
	return parseFormatUserErrorTextWithReference(parseMessage, "")
}

// parseFormatUserErrorTextWithReference appends one visible request identifier to customer-facing copy using one explicit preferred reference id when present.
func parseFormatUserErrorTextWithReference(parseMessage string, parseReferenceID string) string {
	parseMessage = strings.TrimSpace(parseMessage)
	if parseMessage == "" {
		parseMessage = parseResolveUserFallbackMessage("")
	}
	if strings.Contains(strings.ToLower(parseMessage), "request id:") {
		return parseMessage
	}
	parseRequestID := strings.TrimSpace(parseReferenceID)
	if parseRequestID == "" {
		parseRequestID = parseBuildUserRequestID()
	}
	if parseRequestID == "" {
		return parseMessage
	}
	return parseMessage + " Request ID: " + parseRequestID + "."
}

// parseBuildUserErrorReferenceID resolves one user-visible reference id from one typed server contract, falling back to local correlation identity.
func parseBuildUserErrorReferenceID(parseErr error) string {
	if parseContract, isParseContract := parseResolveUserErrorContract(parseErr); isParseContract {
		if parseReferenceID := strings.TrimSpace(parseContract.ParseReferenceID); parseReferenceID != "" {
			return parseReferenceID
		}
	}
	return parseBuildUserRequestID()
}

// parseResolveUserErrorContract extracts one typed customer-safe error contract from one RPC status detail payload.
func parseResolveUserErrorContract(parseErr error) (parseUserErrorContract, bool) {
	if parseErr == nil {
		return parseUserErrorContract{}, false
	}
	parseStatusErr, parseOk := status.FromError(parseErr)
	if !parseOk {
		return parseUserErrorContract{}, false
	}
	for _, parseDetail := range parseStatusErr.Details() {
		parseInfo, isParseInfo := parseDetail.(*errdetails.ErrorInfo)
		if !isParseInfo {
			continue
		}
		if strings.TrimSpace(parseInfo.GetReason()) != parseUserErrorInfoReason || strings.TrimSpace(parseInfo.GetDomain()) != parseUserErrorInfoDomain {
			continue
		}
		parseMetadata := parseInfo.GetMetadata()
		return parseUserErrorContract{
			ParseScope:       strings.TrimSpace(parseMetadata["scope"]),
			ParseUserMessage: strings.TrimSpace(parseMetadata[parseUserErrorMetadataMessage]),
			ParseReferenceID: strings.TrimSpace(parseMetadata[parseUserErrorMetadataReferenceID]),
		}, true
	}
	return parseUserErrorContract{}, false
}

// parseBuildUserRequestID returns one user-visible request identifier derived from existing correlation metadata.
func parseBuildUserRequestID() string {
	parseRequestID := strings.TrimSpace(parseEnsurePersistedCorrelationIdentity())
	if parseRequestID != "" {
		return parseRequestID
	}
	return strings.TrimSpace(parseBuildOpaqueMetadataID())
}

// parseUserErrorMessage extracts the customer-facing message portion from one formatted error text string.
func parseUserErrorMessage(parseErrorText string) string {
	parseErrorText = strings.TrimSpace(parseErrorText)
	if parseIdx := strings.Index(parseErrorText, " Request ID: "); parseIdx >= 0 {
		return strings.TrimSpace(parseErrorText[:parseIdx])
	}
	return parseErrorText
}

// parseUserErrorRequestID extracts the embedded request identifier from one formatted error text string.
func parseUserErrorRequestID(parseErrorText string) string {
	if parseIdx := strings.Index(parseErrorText, " Request ID: "); parseIdx >= 0 {
		parseIDRaw := strings.TrimSpace(parseErrorText[parseIdx+len(" Request ID: "):])
		return strings.TrimSuffix(parseIDRaw, ".")
	}
	return ""
}
