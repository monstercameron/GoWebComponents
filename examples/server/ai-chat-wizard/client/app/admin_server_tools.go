//go:build js && wasm

// admin_server_tools.go holds the server-tools ops surface data types and
// the parseUseAdminServerTools hook that fetches GetSuperuserOpsDiagnostics.
package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v6/logging"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// adminServerToolsPolicyRule is the flattened render-only view of one approved tool entry.
type adminServerToolsPolicyRule struct {
	ToolID           string
	Description      string
	IsEnabled        bool
	Shell            string
	ArgvPrefix       []string
	AllowArgsRegex   []string
	DenyArgsRegex    []string
	AllowCWDPrefixes []string
}

// adminServerToolsPolicy is the flattened policy snapshot from GetSuperuserOpsDiagnostics.
type adminServerToolsPolicy struct {
	IsEnabled         bool
	MaxSessionSeconds int32
	MaxOutputBytes    int32
	ApprovedTools     []adminServerToolsPolicyRule
	Source            string
	UpdatedAt         string
}

// adminServerToolsExecutionRow is one recent server-tool audit entry.
type adminServerToolsExecutionRow struct {
	EventType string
	Summary   string
	CreatedAt string
}

// adminServerToolsData is the full render-only snapshot for the server-tools panel.
type adminServerToolsData struct {
	IsLoading  bool
	IsDenied   bool
	Error      string
	HasData    bool
	LogLines   []string
	Policy     adminServerToolsPolicy
	Executions []adminServerToolsExecutionRow
}

// parseUseAdminServerTools returns an adminServerToolsData snapshot refreshed
// whenever the user holds the superuser role and gRPC is ready.
func parseUseAdminServerTools(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	_ func(error) bool,
) adminServerToolsData {
	parseDataState := ui.UseState(adminServerToolsData{})
	parseRequestSeq := ui.UseRef(uint64(0))

	ui.UseEffect(func() func() {
		// Only superusers may fetch this surface; non-superuser admins see nothing.
		if !parseCurrentState.Authenticated || !parseCurrentState.IsSuperuser {
			parseDataState.Set(adminServerToolsData{})
			return nil
		}
		if !parseCurrentState.GRPCReady {
			parseDataState.Set(adminServerToolsData{
				Error: parseBuildUserErrorText(userErrorScopeDashboard, nil),
			})
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseNextSeq := parseRequestSeq.Get() + 1
		parseRequestSeq.Set(parseNextSeq)
		parseDataState.Set(adminServerToolsData{IsLoading: true})

		go func(parseSeq uint64) {
			parseResp, parseErr := parseClient.GetSuperuserOpsDiagnostics(context.Background(), &chatpb.GetSuperuserOpsDiagnosticsRequest{
				LogMaxLines:    50,
				LogSource:      "server",
				ExecutionLimit: 20,
			})
			if parseRequestSeq.Get() != parseSeq {
				return
			}
			if parseErr != nil {
				parseErrMsg := parseErr.Error()
				isDenied := strings.Contains(parseErrMsg, "PermissionDenied") ||
					strings.Contains(parseErrMsg, "permission denied") ||
					strings.Contains(parseErrMsg, "unauthenticated")
				chatLog.Warn("admin server tools fetch failed", logging.Fields{"error": parseErrMsg, "denied": isDenied})
				parseDataState.Set(adminServerToolsData{
					Error:    parseBuildUserErrorText(userErrorScopeDashboard, parseErr),
					IsDenied: isDenied,
				})
				return
			}
			parseDataState.Set(parseMarshalAdminServerToolsResp(parseResp))
		}(parseNextSeq)
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, parseCurrentState.IsSuperuser)

	return parseDataState.Get()
}

// parseMarshalAdminServerToolsResp converts a GetSuperuserOpsDiagnosticsResponse into the flat snapshot.
func parseMarshalAdminServerToolsResp(parseResp *chatpb.GetSuperuserOpsDiagnosticsResponse) adminServerToolsData {
	if parseResp == nil {
		return adminServerToolsData{Error: parseBuildUserErrorText(userErrorScopeDashboard, nil)}
	}
	parseDiag := parseResp.GetDiagnostics()
	if parseDiag == nil {
		return adminServerToolsData{HasData: true}
	}

	parseData := adminServerToolsData{HasData: true}

	// Log tail lines.
	for _, parseEntry := range parseDiag.GetLogTail() {
		parseMsg := strings.TrimSpace(parseEntry.GetLine())
		if parseMsg != "" {
			parseData.LogLines = append(parseData.LogLines, parseMsg)
		}
	}

	// Tool policy.
	if parsePolicy := parseDiag.GetServerToolPolicy(); parsePolicy != nil {
		parseFlatPolicy := adminServerToolsPolicy{
			IsEnabled:         parsePolicy.GetIsEnabled(),
			MaxSessionSeconds: parsePolicy.GetMaxSessionSeconds(),
			MaxOutputBytes:    parsePolicy.GetMaxOutputBytes(),
			Source:            parsePolicy.GetSource(),
			UpdatedAt:         parsePolicy.GetUpdatedAt(),
		}
		for _, parseTool := range parsePolicy.GetApprovedTools() {
			parseFlatPolicy.ApprovedTools = append(parseFlatPolicy.ApprovedTools, adminServerToolsPolicyRule{
				ToolID:           parseTool.GetToolId(),
				Description:      parseTool.GetDescription(),
				IsEnabled:        parseTool.GetIsEnabled(),
				Shell:            parseTool.GetShell(),
				ArgvPrefix:       append([]string(nil), parseTool.GetArgvPrefix()...),
				AllowArgsRegex:   append([]string(nil), parseTool.GetAllowArgsRegex()...),
				DenyArgsRegex:    append([]string(nil), parseTool.GetDenyArgsRegex()...),
				AllowCWDPrefixes: append([]string(nil), parseTool.GetAllowCwdPrefixes()...),
			})
		}
		parseData.Policy = parseFlatPolicy
	}

	// Recent executions from the audit log.
	for _, parseAudit := range parseDiag.GetRecentServerToolExecutions() {
		parseData.Executions = append(parseData.Executions, adminServerToolsExecutionRow{
			EventType: parseAudit.GetEventType(),
			Summary:   parseAudit.GetSummary(),
			CreatedAt: parseAudit.GetCreatedAt(),
		})
	}

	return parseData
}
