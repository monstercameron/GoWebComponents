package app

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

// TestResolveSupportIDExposurePolicy verifies customer-safe support id derivation and operator-only id exposure policy.
func TestResolveSupportIDExposurePolicy(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-bridge-123",
		correlationIDMetadataKey, "corr-bridge-456",
	))
	parseExposure := parseResolveSupportIDExposure(parseCtx, "audit-evt-789", "session-xyz")

	if !strings.HasPrefix(parseExposure.ParseCustomerSupportID, parseSupportIDPrefix) {
		parseT.Fatalf("customer support id %q missing prefix %q", parseExposure.ParseCustomerSupportID, parseSupportIDPrefix)
	}
	if strings.Contains(parseExposure.ParseCustomerSupportID, "req-bridge-123") || strings.Contains(parseExposure.ParseCustomerSupportID, "corr-bridge-456") {
		parseT.Fatalf("customer support id leaked operator ids: %q", parseExposure.ParseCustomerSupportID)
	}
	if parseExposure.ParseOperatorRequestID != "req-bridge-123" || parseExposure.ParseOperatorCorrelationID != "corr-bridge-456" || parseExposure.ParseOperatorAuditID != "audit-evt-789" || parseExposure.ParseOperatorSessionID != "session-xyz" {
		parseT.Fatalf("unexpected operator exposure payload: %+v", parseExposure)
	}
	if parseExposure.IsParseRequestIDCustomerVisible || parseExposure.IsParseCorrelationIDCustomerVisible || parseExposure.IsParseAuditIDCustomerVisible || parseExposure.IsParseSessionIDCustomerVisible {
		parseT.Fatalf("expected operator identifiers to remain customer-hidden, got %+v", parseExposure)
	}
}

// TestBuildCustomerSupportIDFallbackAndDeterminism verifies unknown fallback and deterministic hash behavior.
func TestBuildCustomerSupportIDFallbackAndDeterminism(parseT *testing.T) {
	parseUnknown := parseBuildCustomerSupportID("")
	if parseUnknown != parseSupportIDUnknown {
		parseT.Fatalf("empty seed support id = %q, want %q", parseUnknown, parseSupportIDUnknown)
	}

	parseFirst := parseBuildCustomerSupportID("corr-bridge-456")
	parseSecond := parseBuildCustomerSupportID("corr-bridge-456")
	parseDifferent := parseBuildCustomerSupportID("corr-bridge-457")

	if parseFirst != parseSecond {
		parseT.Fatalf("expected deterministic support ids, first=%q second=%q", parseFirst, parseSecond)
	}
	if parseFirst == parseDifferent {
		parseT.Fatalf("expected different seeds to produce different support ids, both=%q", parseFirst)
	}
}
