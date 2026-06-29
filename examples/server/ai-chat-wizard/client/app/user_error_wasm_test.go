//go:build js && wasm

package app

import (
	"strings"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestBuildUserErrorTextUsesContractMessageAndReference verifies contract metadata drives chat/settings/dashboard error text and request id.
func TestBuildUserErrorTextUsesContractMessageAndReference(parseT *testing.T) {
	parseStatusErr := status.New(codes.Unavailable, "provider busy")
	parseStatusWithDetails, parseErr := parseStatusErr.WithDetails(&errdetails.ErrorInfo{
		Reason: parseUserErrorInfoReason,
		Domain: parseUserErrorInfoDomain,
		Metadata: map[string]string{
			"scope":                           userErrorScopeChat,
			parseUserErrorMetadataMessage:     "We couldn't send your message right now. Please try again.",
			parseUserErrorMetadataReferenceID: "SUP-TESTREF123",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("WithDetails: %v", parseErr)
	}
	parseMessage := parseBuildUserErrorText(userErrorScopeChat, parseStatusWithDetails.Err())
	if !strings.Contains(parseMessage, "We couldn't send your message right now. Please try again.") {
		parseT.Fatalf("message = %q", parseMessage)
	}
	if !strings.Contains(parseMessage, "Request ID: SUP-TESTREF123.") {
		parseT.Fatalf("message = %q, want contract reference id", parseMessage)
	}
}

// TestBuildUserAuthErrorTextKeepsAuthCopyAndUsesContractReference verifies auth copy still maps through auth-specific messaging while request id uses contract metadata.
func TestBuildUserAuthErrorTextKeepsAuthCopyAndUsesContractReference(parseT *testing.T) {
	parseStatusErr := status.New(codes.Unauthenticated, "invalid email or password")
	parseStatusWithDetails, parseErr := parseStatusErr.WithDetails(&errdetails.ErrorInfo{
		Reason: parseUserErrorInfoReason,
		Domain: parseUserErrorInfoDomain,
		Metadata: map[string]string{
			"scope":                           userErrorScopeAuth,
			parseUserErrorMetadataMessage:     "That email and password didn't match. Try again.",
			parseUserErrorMetadataReferenceID: "SUP-AUTHREF456",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("WithDetails: %v", parseErr)
	}
	parseMessage := parseBuildUserAuthErrorText(authModeLogin, parseStatusWithDetails.Err())
	if !strings.Contains(parseMessage, "That email and password didn't match. Try again.") {
		parseT.Fatalf("message = %q", parseMessage)
	}
	if !strings.Contains(parseMessage, "Request ID: SUP-AUTHREF456.") {
		parseT.Fatalf("message = %q, want contract reference id", parseMessage)
	}
}
