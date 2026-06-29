package app

import (
	"context"
	"log/slog"
	"strings"
)

// parseResolveAuthRPCLogger returns one traceability-enriched logger for auth RPC flows.
func parseResolveAuthRPCLogger(parseS *chatServer, parseCtx context.Context, parseRPC string) *slog.Logger {
	parseRPC = strings.TrimSpace(parseRPC)
	if parseRPC == "" {
		parseRPC = "AuthRPC"
	}
	if parseS != nil && parseS.logger != nil {
		return parseS.logger.With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{ParseRPC: parseRPC})...)
	}
	return slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{ParseRPC: parseRPC})...)
}

// parseIsStoreUnavailableError reports whether one error string indicates store-availability loss.
func parseIsStoreUnavailableError(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(parseErr.Error())), "store unavailable")
}

// parseResolveAdminMutationTargetScope resolves one stable target scope label for one admin mutation action.
func parseResolveAdminMutationTargetScope(parseAction parseAdminMutationAction) string {
	switch parseAction {
	case parseAdminMutationSuspendWorkspace, parseAdminMutationRestoreWorkspace:
		return "workspace"
	default:
		return "user"
	}
}
