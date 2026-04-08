package app

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

const parseSupportIDUnknown = "SUP-UNKNOWN"
const parseSupportIDPrefix = "SUP-"

type parseSupportIDExposure struct {
	ParseCustomerSupportID              string
	ParseOperatorRequestID              string
	ParseOperatorCorrelationID          string
	ParseOperatorAuditID                string
	ParseOperatorSessionID              string
	IsParseRequestIDCustomerVisible     bool
	IsParseCorrelationIDCustomerVisible bool
	IsParseAuditIDCustomerVisible       bool
	IsParseSessionIDCustomerVisible     bool
}

// parseResolveSupportIDExposure resolves customer-safe and operator-only identifiers for one request.
func parseResolveSupportIDExposure(parseCtx context.Context, parseAuditID string, parseSessionID string) parseSupportIDExposure {
	parseRequestID, parseCorrelationID := parseExtractCorrelationFromContext(parseCtx)
	parseRequestID = strings.TrimSpace(parseRequestID)
	parseCorrelationID = strings.TrimSpace(parseCorrelationID)
	parseAuditID = strings.TrimSpace(parseAuditID)
	parseSessionID = strings.TrimSpace(parseSessionID)

	parseSupportSeed := parseCorrelationID
	if parseSupportSeed == "" {
		parseSupportSeed = parseRequestID
	}
	if parseSupportSeed == "" {
		parseSupportSeed = parseAuditID
	}
	if parseSupportSeed == "" {
		parseSupportSeed = parseSessionID
	}

	return parseSupportIDExposure{
		ParseCustomerSupportID:              parseBuildCustomerSupportID(parseSupportSeed),
		ParseOperatorRequestID:              parseRequestID,
		ParseOperatorCorrelationID:          parseCorrelationID,
		ParseOperatorAuditID:                parseAuditID,
		ParseOperatorSessionID:              parseSessionID,
		IsParseRequestIDCustomerVisible:     false,
		IsParseCorrelationIDCustomerVisible: false,
		IsParseAuditIDCustomerVisible:       false,
		IsParseSessionIDCustomerVisible:     false,
	}
}

// parseBuildCustomerSupportID derives one stable customer-safe support id from one operator-only seed.
func parseBuildCustomerSupportID(parseSeed string) string {
	parseSeed = strings.TrimSpace(parseSeed)
	if parseSeed == "" {
		return parseSupportIDUnknown
	}
	parseDigest := sha256.Sum256([]byte(parseSeed))
	parseEncodedDigest := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(parseDigest[:])
	parseEncodedDigest = strings.TrimSpace(parseEncodedDigest)
	if parseEncodedDigest == "" {
		return parseSupportIDUnknown
	}
	if len(parseEncodedDigest) > 12 {
		parseEncodedDigest = parseEncodedDigest[:12]
	}
	return parseSupportIDPrefix + parseEncodedDigest
}
