package app

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

// TestIsStoreUnavailableError verifies centralized store-outage detection semantics.
func TestIsStoreUnavailableError(parseT *testing.T) {
	if !parseIsStoreUnavailableError(errors.New("Store Unavailable: db closed")) {
		parseT.Fatal("expected store unavailable detection for store-outage error")
	}
	if parseIsStoreUnavailableError(errors.New("permission denied")) {
		parseT.Fatal("expected non-store errors to remain false")
	}
}

// TestResolveAdminMutationTargetScope verifies stable scope labels for user/workspace mutation actions.
func TestResolveAdminMutationTargetScope(parseT *testing.T) {
	if parseScope := parseResolveAdminMutationTargetScope(parseAdminMutationDisableUser); parseScope != "user" {
		parseT.Fatalf("disable-user target scope = %q, want user", parseScope)
	}
	if parseScope := parseResolveAdminMutationTargetScope(parseAdminMutationSuspendWorkspace); parseScope != "workspace" {
		parseT.Fatalf("suspend-workspace target scope = %q, want workspace", parseScope)
	}
}

// TestResolveAuthRPCLoggerIncludesTraceability verifies auth RPC logger helper carries rpc + request metadata attributes.
func TestResolveAuthRPCLoggerIncludesTraceability(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseServer := &chatServer{
		logger: parseNewOTELLogger(&parseOutput, serverServiceName),
	}
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-auth-helper-123",
		correlationIDMetadataKey, "corr-auth-helper-456",
	))
	parseLogger := parseResolveAuthRPCLogger(parseServer, parseCtx, "Signup")
	parseLogger.Info("auth helper traceability test")

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected logger output")
	}
	if !strings.Contains(parseLogLine, `"rpc":"Signup"`) || !strings.Contains(parseLogLine, `"request.id":"req-auth-helper-123"`) || !strings.Contains(parseLogLine, `"correlation.id":"corr-auth-helper-456"`) {
		parseT.Fatalf("expected rpc/request/correlation attrs in log line, got %q", parseLogLine)
	}
}
