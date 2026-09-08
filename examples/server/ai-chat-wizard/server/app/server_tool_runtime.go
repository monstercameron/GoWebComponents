package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc"
)

type parseServerToolRuntimeResult struct {
	ParseExitCode   int32
	ParseFinishedAt string
	ParseErrorText  string
}

type parseServerToolEventSender struct {
	parseStream grpc.BidiStreamingServer[chatpb.RunServerToolRequest, chatpb.RunServerToolEvent]
	parseMutex  sync.Mutex
}

type parseServerToolRuntimeSession struct {
	parseSessionID      string
	parseCommandTokens  []string
	parseCommand        *exec.Cmd
	parseStdin          io.WriteCloser
	parseSender         *parseServerToolEventSender
	parseCancel         context.CancelFunc
	parseDone           chan parseServerToolRuntimeResult
	parseMaxOutputBytes int64
	parseOutputBytes    int64
	parseWriteMutex     sync.Mutex
	isParseStdinClosed  bool
	parseErrorOnce      sync.Once
}

// parseSendEvent serializes one stream event send across concurrent producer goroutines.
func (parseS *parseServerToolEventSender) parseSendEvent(parseEvent *chatpb.RunServerToolEvent) error {
	if parseS == nil || parseS.parseStream == nil {
		return errors.New("server tool event sender stream is required")
	}
	if parseEvent == nil {
		return errors.New("server tool event is required")
	}
	parseS.parseMutex.Lock()
	defer parseS.parseMutex.Unlock()
	return parseS.parseStream.Send(parseEvent)
}

// parseSendErrorEvent emits one typed run-server-tool error frame.
func (parseS *parseServerToolEventSender) parseSendErrorEvent(parseSessionID string, parseCode string, parseMessage string) error {
	return parseS.parseSendEvent(&chatpb.RunServerToolEvent{
		Payload: &chatpb.RunServerToolEvent_Error{
			Error: &chatpb.RunServerToolError{
				SessionId: strings.TrimSpace(parseSessionID),
				Code:      strings.TrimSpace(parseCode),
				Message:   strings.TrimSpace(parseMessage),
			},
		},
	})
}

// parseSendExitEvent emits one terminal run-server-tool exit frame.
func (parseS *parseServerToolEventSender) parseSendExitEvent(parseSessionID string, parseResult parseServerToolRuntimeResult) error {
	return parseS.parseSendEvent(&chatpb.RunServerToolEvent{
		Payload: &chatpb.RunServerToolEvent_Exit{
			Exit: &chatpb.RunServerToolExit{
				SessionId:  strings.TrimSpace(parseSessionID),
				ExitCode:   parseResult.ParseExitCode,
				FinishedAt: strings.TrimSpace(parseResult.ParseFinishedAt),
				Error:      strings.TrimSpace(parseResult.ParseErrorText),
			},
		},
	})
}

// parseStartServerToolRuntimeSession starts one command execution session after policy authorization.
func parseStartServerToolRuntimeSession(parseCtx context.Context, parseSender *parseServerToolEventSender, parseStart *chatpb.RunServerToolStart, parsePolicy *chatpb.GetServerToolPolicyResponse) (*parseServerToolRuntimeSession, error) {
	if parseSender == nil {
		return nil, errors.New("server tool sender is required")
	}
	if parseStart == nil {
		return nil, errors.New("run server tool start payload is required")
	}
	parseCommandTokens, parseErr := parseSplitServerToolCommandTokens(parseStart.GetCommand())
	if parseErr != nil {
		return nil, parseErr
	}
	parseSessionID := strings.TrimSpace(parseStart.GetSessionId())
	if parseSessionID == "" {
		parseSessionID = uuid.NewString()
	}
	parseTimeoutSeconds := parseStart.GetTimeoutSeconds()
	if parseTimeoutSeconds <= 0 && parsePolicy != nil {
		parseTimeoutSeconds = parsePolicy.GetMaxSessionSeconds()
	}
	if parseTimeoutSeconds <= 0 {
		parseTimeoutSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	parseRuntimeCtx, parseCancel := context.WithTimeout(parseCtx, time.Duration(parseTimeoutSeconds)*time.Second)
	parseCommand := exec.CommandContext(parseRuntimeCtx, parseCommandTokens[0], parseCommandTokens[1:]...)
	parseCommand.Dir = parseResolveServerToolRuntimeCWD(parseStart.GetCwd())
	parseStdout, parseErr := parseCommand.StdoutPipe()
	if parseErr != nil {
		parseCancel()
		return nil, parseErr
	}
	parseStderr, parseErr := parseCommand.StderrPipe()
	if parseErr != nil {
		parseCancel()
		return nil, parseErr
	}
	parseStdin, parseErr := parseCommand.StdinPipe()
	if parseErr != nil {
		parseCancel()
		return nil, parseErr
	}
	if parseErr = parseCommand.Start(); parseErr != nil {
		parseCancel()
		return nil, parseErr
	}
	parseStartedAt := time.Now().UTC().Format(time.RFC3339)
	if parseErr = parseSender.parseSendEvent(&chatpb.RunServerToolEvent{
		Payload: &chatpb.RunServerToolEvent_Started{
			Started: &chatpb.RunServerToolStarted{
				SessionId: parseSessionID,
				Shell:     parseBuildServerToolPolicyShell(parseStart.GetShell()),
				Cwd:       parseCommand.Dir,
				StartedAt: parseStartedAt,
			},
		},
	}); parseErr != nil {
		parseCancel()
		_ = parseCommand.Process.Kill()
		_, _ = parseCommand.Process.Wait()
		return nil, parseErr
	}

	parseMaxOutputBytes := int64(serverToolPolicyDefaultMaxOutputBytes)
	if parsePolicy != nil && parsePolicy.GetMaxOutputBytes() > 0 {
		parseMaxOutputBytes = int64(parsePolicy.GetMaxOutputBytes())
	}
	parseSession := &parseServerToolRuntimeSession{
		parseSessionID:      parseSessionID,
		parseCommandTokens:  parseCommandTokens,
		parseCommand:        parseCommand,
		parseStdin:          parseStdin,
		parseSender:         parseSender,
		parseCancel:         parseCancel,
		parseDone:           make(chan parseServerToolRuntimeResult, 1),
		parseMaxOutputBytes: parseMaxOutputBytes,
	}
	go parseSession.parseRunCommand(parseStdout, parseStderr)
	return parseSession, nil
}

// parseRunCommand drains stdout/stderr, waits for process completion, and publishes one terminal result.
func (parseS *parseServerToolRuntimeSession) parseRunCommand(parseStdout io.ReadCloser, parseStderr io.ReadCloser) {
	parseStdoutDone := make(chan struct{})
	parseStderrDone := make(chan struct{})
	go parseS.parseDrainOutputPipe(parseStdout, true, parseStdoutDone)
	go parseS.parseDrainOutputPipe(parseStderr, false, parseStderrDone)
	parseWaitErr := parseS.parseCommand.Wait()
	<-parseStdoutDone
	<-parseStderrDone
	parseResult := parseBuildServerToolRuntimeResult(parseS.parseCommand, parseWaitErr)
	parseS.parseDone <- parseResult
}

// parseDrainOutputPipe streams one output pipe onto the gRPC event stream with output-byte cap enforcement.
func (parseS *parseServerToolRuntimeSession) parseDrainOutputPipe(parsePipe io.ReadCloser, isParseStdout bool, parseDone chan<- struct{}) {
	defer close(parseDone)
	if parsePipe == nil {
		return
	}
	defer parsePipe.Close()
	parseBuffer := make([]byte, 4096)
	for {
		parseReadCount, parseReadErr := parsePipe.Read(parseBuffer)
		if parseReadCount > 0 {
			parseChunk := append([]byte(nil), parseBuffer[:parseReadCount]...)
			parseExceeded, parseSendErr := parseS.parseSendOutputChunk(parseChunk, isParseStdout)
			if parseSendErr != nil {
				parseS.parseEmitRuntimeError("stream_send_failed", parseSendErr.Error())
				parseS.parseCancel()
				return
			}
			if parseExceeded {
				parseS.parseEmitRuntimeError("output_limit_exceeded", "server tool output exceeded policy max_output_bytes")
				parseS.parseCancel()
				return
			}
		}
		if parseReadErr == io.EOF {
			_ = parseS.parseSender.parseSendEvent(parseBuildServerToolOutputEvent(parseS.parseSessionID, nil, true, isParseStdout))
			return
		}
		if parseReadErr != nil {
			parseS.parseEmitRuntimeError("stream_read_failed", parseReadErr.Error())
			parseS.parseCancel()
			return
		}
	}
}

// parseSendOutputChunk emits one output chunk and returns true when output cap was exceeded.
func (parseS *parseServerToolRuntimeSession) parseSendOutputChunk(parseChunk []byte, isParseStdout bool) (bool, error) {
	if len(parseChunk) == 0 {
		return false, nil
	}
	parseS.parseWriteMutex.Lock()
	parseRemainingBytes := parseS.parseMaxOutputBytes - parseS.parseOutputBytes
	if parseRemainingBytes <= 0 {
		parseS.parseWriteMutex.Unlock()
		return true, nil
	}
	parseChunkLen := int64(len(parseChunk))
	isParseExceeded := false
	if parseChunkLen > parseRemainingBytes {
		parseChunk = parseChunk[:parseRemainingBytes]
		parseChunkLen = parseRemainingBytes
		isParseExceeded = true
	}
	parseS.parseOutputBytes += parseChunkLen
	parseS.parseWriteMutex.Unlock()
	if parseChunkLen > 0 {
		if parseErr := parseS.parseSender.parseSendEvent(parseBuildServerToolOutputEvent(parseS.parseSessionID, parseChunk, false, isParseStdout)); parseErr != nil {
			return isParseExceeded, parseErr
		}
	}
	return isParseExceeded, nil
}

// parseWriteInputChunk writes one stdin chunk for one active session and optionally closes stdin.
func (parseS *parseServerToolRuntimeSession) parseWriteInputChunk(parseSessionID string, parseChunk []byte, isParseClose bool) error {
	if parseS == nil {
		return errors.New("server tool session is required")
	}
	if strings.TrimSpace(parseSessionID) != "" && strings.TrimSpace(parseSessionID) != parseS.parseSessionID {
		return errors.New("stdin session id does not match active session")
	}
	parseS.parseWriteMutex.Lock()
	defer parseS.parseWriteMutex.Unlock()
	if parseS.isParseStdinClosed {
		return errors.New("stdin is already closed")
	}
	if len(parseChunk) > 0 {
		if _, parseErr := parseS.parseStdin.Write(parseChunk); parseErr != nil {
			return parseErr
		}
	}
	if isParseClose {
		if parseErr := parseS.parseStdin.Close(); parseErr != nil {
			return parseErr
		}
		parseS.isParseStdinClosed = true
	}
	return nil
}

// parseSendSignal dispatches one process signal for one active runtime session.
func (parseS *parseServerToolRuntimeSession) parseSendSignal(parseSessionID string, parseSignalName string) error {
	if parseS == nil || parseS.parseCommand == nil || parseS.parseCommand.Process == nil {
		return errors.New("server tool process is not running")
	}
	if strings.TrimSpace(parseSessionID) != "" && strings.TrimSpace(parseSessionID) != parseS.parseSessionID {
		return errors.New("signal session id does not match active session")
	}
	parseSignal, parseErr := parseResolveServerToolSignal(parseSignalName)
	if parseErr != nil {
		return parseErr
	}
	return parseS.parseCommand.Process.Signal(parseSignal)
}

// parseCloseSession requests graceful session shutdown by closing stdin and canceling the process context.
func (parseS *parseServerToolRuntimeSession) parseCloseSession(parseSessionID string) error {
	if parseS == nil {
		return errors.New("server tool session is required")
	}
	if strings.TrimSpace(parseSessionID) != "" && strings.TrimSpace(parseSessionID) != parseS.parseSessionID {
		return errors.New("close session id does not match active session")
	}
	_ = parseS.parseWriteInputChunk(parseS.parseSessionID, nil, true)
	parseS.parseCancel()
	return nil
}

// parseEmitRuntimeError emits one stream error frame only once for one runtime session.
func (parseS *parseServerToolRuntimeSession) parseEmitRuntimeError(parseCode string, parseMessage string) {
	if parseS == nil || parseS.parseSender == nil {
		return
	}
	parseS.parseErrorOnce.Do(func() {
		_ = parseS.parseSender.parseSendErrorEvent(parseS.parseSessionID, parseCode, parseMessage)
	})
}

// parseResolveServerToolRuntimeCWD normalizes one runtime cwd value for command execution.
func parseResolveServerToolRuntimeCWD(parseCWD string) string {
	parseCWD = strings.TrimSpace(parseCWD)
	if parseCWD == "" {
		return "."
	}
	return parseCWD
}

// parseBuildServerToolOutputEvent builds one stdout or stderr event frame.
func parseBuildServerToolOutputEvent(parseSessionID string, parseChunk []byte, isParseEOF bool, isParseStdout bool) *chatpb.RunServerToolEvent {
	parseOutput := &chatpb.RunServerToolOutput{
		SessionId: strings.TrimSpace(parseSessionID),
		Chunk:     parseChunk,
		Eof:       isParseEOF,
	}
	if isParseStdout {
		return &chatpb.RunServerToolEvent{
			Payload: &chatpb.RunServerToolEvent_Stdout{Stdout: parseOutput},
		}
	}
	return &chatpb.RunServerToolEvent{
		Payload: &chatpb.RunServerToolEvent_Stderr{Stderr: parseOutput},
	}
}

// parseBuildServerToolRuntimeResult resolves one process wait result into one exit payload.
func parseBuildServerToolRuntimeResult(parseCommand *exec.Cmd, parseWaitErr error) parseServerToolRuntimeResult {
	parseResult := parseServerToolRuntimeResult{ParseFinishedAt: time.Now().UTC().Format(time.RFC3339)}
	if parseCommand != nil && parseCommand.ProcessState != nil {
		parseResult.ParseExitCode = int32(parseCommand.ProcessState.ExitCode())
	}
	if parseWaitErr == nil {
		return parseResult
	}
	parseResult.ParseErrorText = strings.TrimSpace(parseWaitErr.Error())
	var parseExitErr *exec.ExitError
	if errors.As(parseWaitErr, &parseExitErr) {
		parseResult.ParseExitCode = int32(parseExitErr.ExitCode())
		return parseResult
	}
	parseResult.ParseExitCode = -1
	return parseResult
}

// parseResolveServerToolSignal maps one signal selector to one OS signal value.
func parseResolveServerToolSignal(parseSignalName string) (os.Signal, error) {
	switch strings.ToLower(strings.TrimSpace(parseSignalName)) {
	case "", "interrupt", "int", "sigint":
		return os.Interrupt, nil
	case "terminate", "term", "sigterm":
		return os.Interrupt, nil
	case "kill", "sigkill":
		return os.Kill, nil
	default:
		return nil, fmt.Errorf("unsupported signal %q", strings.TrimSpace(parseSignalName))
	}
}
