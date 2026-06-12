package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// runMCPCommand routes the MCP command.
var runMCPCommand func(launcher, []string) error

func init() {
	if runMCPCommand == nil {
		runMCPCommand = defaultRunMCPCommand
	}
}

func defaultRunMCPCommand(parseL launcher, parseArgs []string) error {
	return parseL.runMCP(parseArgs)
}

type mcpManifest struct {
	SchemaVersion string    `json:"schemaVersion"`
	ServerName    string    `json:"serverName"`
	Tools         []mcpTool `json:"tools"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type mcpGwcToolArguments struct {
	Args []string `json:"args,omitempty"`
	JSON *bool    `json:"json,omitempty"`
}

// runMCP starts the MCP stdio server or prints the tool manifest as JSON.
func (parseL launcher) runMCP(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("mcp", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseJSON := parseFlags.Bool("json", false, "Emit the MCP tool manifest instead of starting the stdio server")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseManifest := buildMCPManifest()
	if *parseJSON {
		return writeAgenticEnvelope("mcp", true, parseManifest, nil, nil)
	}
	return serveMCPStdio(parseL, os.Stdin, os.Stdout)
}

// buildMCPManifest builds the MCP tool manifest from the command help registry.
func buildMCPManifest() mcpManifest {
	parseManifest := mcpManifest{
		SchemaVersion: agenticEnvelopeSchemaVersion,
		ServerName:    "gwc",
	}
	for _, parseCommand := range listGwcHelpCommands() {
		if !parseCommand.JSON || parseCommand.Name == "mcp" {
			continue
		}
		parseManifest.Tools = append(parseManifest.Tools, mcpToolForCommand(parseCommand))
	}
	return parseManifest
}

// mcpToolForCommand converts one command registry entry into an MCP tool.
func mcpToolForCommand(parseCommand gwcHelpCommand) mcpTool {
	return mcpTool{
		Name:        "gwc_" + strings.ReplaceAll(parseCommand.Name, "-", "_"),
		Description: parseCommand.Summary,
		InputSchema: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"args": map[string]interface{}{
					"type":        "array",
					"description": "CLI arguments after the command name. Omit -json unless you need to override the default.",
					"items":       map[string]interface{}{"type": "string"},
				},
				"json": map[string]interface{}{
					"type":        "boolean",
					"description": "When true, append -json if the args do not already request JSON.",
					"default":     true,
				},
			},
		},
		Annotations: map[string]interface{}{
			"readOnlyHint":    parseCommand.ReadOnly,
			"destructiveHint": parseCommand.Mutating,
			"title":           parseCommand.Name,
		},
	}
}

// serveMCPStdio serves a small JSON-RPC MCP subset over stdio.
func serveMCPStdio(parseL launcher, parseReader io.Reader, parseWriter io.Writer) error {
	parseBuffered := bufio.NewReader(parseReader)
	for {
		parsePayload, parseErr := readMCPMessage(parseBuffered)
		if errors.Is(parseErr, io.EOF) {
			return nil
		}
		if parseErr != nil {
			return parseErr
		}
		if len(bytes.TrimSpace(parsePayload)) == 0 {
			continue
		}
		parseResponse := handleMCPRequest(parseL, parsePayload)
		if parseResponse == nil {
			continue
		}
		if parseErr := writeMCPMessage(parseWriter, parseResponse); parseErr != nil {
			return parseErr
		}
	}
}

// handleMCPRequest handles one JSON-RPC request payload.
func handleMCPRequest(parseL launcher, parsePayload []byte) *mcpResponse {
	parseRequest := mcpRequest{}
	if parseErr := json.Unmarshal(parsePayload, &parseRequest); parseErr != nil {
		return buildMCPErrorResponse(nil, -32700, "parse error: "+parseErr.Error())
	}
	switch parseRequest.Method {
	case "initialize":
		return &mcpResponse{
			JSONRPC: "2.0",
			ID:      parseRequest.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]interface{}{"name": "gwc", "version": agenticEnvelopeSchemaVersion},
			},
		}
	case "tools/list":
		return &mcpResponse{JSONRPC: "2.0", ID: parseRequest.ID, Result: map[string]interface{}{"tools": buildMCPManifest().Tools}}
	case "tools/call":
		parseResult, parseErr := executeMCPToolCall(parseL, parseRequest.Params)
		if parseErr != nil {
			return buildMCPErrorResponse(parseRequest.ID, -32602, parseErr.Error())
		}
		return &mcpResponse{JSONRPC: "2.0", ID: parseRequest.ID, Result: parseResult}
	case "notifications/initialized":
		return nil
	default:
		return buildMCPErrorResponse(parseRequest.ID, -32601, "method not found: "+parseRequest.Method)
	}
}

// executeMCPToolCall invokes a gwc command and returns MCP content.
func executeMCPToolCall(parseL launcher, parseParams json.RawMessage) (map[string]interface{}, error) {
	parseCall := mcpToolCallParams{}
	if parseErr := json.Unmarshal(parseParams, &parseCall); parseErr != nil {
		return nil, fmt.Errorf("decode tool call params: %w", parseErr)
	}
	parseCommand, parseOK := commandNameForMCPTool(parseCall.Name)
	if !parseOK {
		return nil, fmt.Errorf("unknown gwc MCP tool %q", parseCall.Name)
	}
	parseArguments := mcpGwcToolArguments{JSON: boolPtr(true)}
	if len(parseCall.Arguments) > 0 {
		if parseErr := json.Unmarshal(parseCall.Arguments, &parseArguments); parseErr != nil {
			return nil, fmt.Errorf("decode gwc tool arguments: %w", parseErr)
		}
	}
	parseArgs := append([]string{}, parseArguments.Args...)
	if parseArguments.JSON == nil || *parseArguments.JSON {
		if !launcherJSONRequestedForCommand(parseCommand, parseArgs) {
			parseArgs = append(parseArgs, "-json")
		}
	}
	parseOutput, parseErr := captureMCPCommandOutput(func() error {
		return parseL.dispatchCommand(parseCommand, parseArgs)
	})
	if parseErr != nil && strings.TrimSpace(parseOutput) == "" {
		parseOutput = parseErr.Error()
	}
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": strings.TrimSpace(parseOutput)},
		},
		"isError": parseErr != nil,
	}, nil
}

// commandNameForMCPTool maps an MCP tool name to a gwc command.
func commandNameForMCPTool(parseToolName string) (string, bool) {
	parseToolName = strings.TrimSpace(parseToolName)
	for _, parseCommand := range listGwcHelpCommands() {
		if !parseCommand.JSON || parseCommand.Name == "mcp" {
			continue
		}
		if "gwc_"+strings.ReplaceAll(parseCommand.Name, "-", "_") == parseToolName {
			return parseCommand.Name, true
		}
	}
	return "", false
}

// captureMCPCommandOutput captures os.Stdout while an in-process command runs.
func captureMCPCommandOutput(parseRun func() error) (string, error) {
	parseOriginalStdout := os.Stdout
	parseReader, parseWriter, parseErr := os.Pipe()
	if parseErr != nil {
		return "", parseErr
	}
	os.Stdout = parseWriter
	var parseBuffer bytes.Buffer
	parseDone := make(chan error, 1)
	go func() {
		_, parseCopyErr := io.Copy(&parseBuffer, parseReader)
		parseDone <- parseCopyErr
	}()
	parseRunErr := parseRun()
	_ = parseWriter.Close()
	parseCopyErr := <-parseDone
	os.Stdout = parseOriginalStdout
	_ = parseReader.Close()
	if parseCopyErr != nil {
		return parseBuffer.String(), parseCopyErr
	}
	return parseBuffer.String(), parseRunErr
}

// readMCPMessage reads either Content-Length framed MCP or newline JSON.
func readMCPMessage(parseReader *bufio.Reader) ([]byte, error) {
	parseFirstLine, parseErr := parseReader.ReadString('\n')
	if parseErr != nil {
		if errors.Is(parseErr, io.EOF) && strings.TrimSpace(parseFirstLine) != "" {
			return []byte(parseFirstLine), nil
		}
		return nil, parseErr
	}
	parseTrimmed := strings.TrimSpace(parseFirstLine)
	if strings.HasPrefix(strings.ToLower(parseTrimmed), "content-length:") {
		parseLengthText := strings.TrimSpace(strings.TrimPrefix(parseTrimmed, "Content-Length:"))
		if parseLengthText == parseTrimmed {
			parseLengthText = strings.TrimSpace(strings.TrimPrefix(parseTrimmed, "content-length:"))
		}
		parseLength, parseErr := strconv.Atoi(parseLengthText)
		if parseErr != nil || parseLength < 0 {
			return nil, fmt.Errorf("invalid MCP content length %q", parseLengthText)
		}
		for {
			parseHeaderLine, parseErr := parseReader.ReadString('\n')
			if parseErr != nil {
				return nil, parseErr
			}
			if strings.TrimSpace(parseHeaderLine) == "" {
				break
			}
		}
		parsePayload := make([]byte, parseLength)
		_, parseErr = io.ReadFull(parseReader, parsePayload)
		return parsePayload, parseErr
	}
	return []byte(parseFirstLine), nil
}

// writeMCPMessage writes one Content-Length framed JSON-RPC response.
func writeMCPMessage(parseWriter io.Writer, parseResponse *mcpResponse) error {
	parsePayload, parseErr := json.Marshal(parseResponse)
	if parseErr != nil {
		return parseErr
	}
	_, parseErr = fmt.Fprintf(parseWriter, "Content-Length: %d\r\n\r\n%s", len(parsePayload), parsePayload)
	return parseErr
}

// buildMCPErrorResponse builds a JSON-RPC error response.
func buildMCPErrorResponse(parseID json.RawMessage, parseCode int, parseMessage string) *mcpResponse {
	return &mcpResponse{
		JSONRPC: "2.0",
		ID:      parseID,
		Error:   &mcpError{Code: parseCode, Message: parseMessage},
	}
}

// boolPtr returns a pointer to a bool literal for default MCP args.
func boolPtr(parseValue bool) *bool {
	return &parseValue
}
