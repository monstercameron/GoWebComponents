package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	logTailSourceServer        = "server"
	logTailSourceClient        = "client"
	logTailSourceAll           = "all"
	logTailDefaultMaxLines     = 200
	logTailMaximumMaxLines     = 2000
	logTailMaximumScanLines    = 10000
	logTailReadChunkBytes      = 4096
	logTailMaximumReadBytes    = 4 * 1024 * 1024
	logTailFilteredScanFactor  = 4
	logTailUnfilteredScanBump  = 20
	logTailReadFallbackMinimum = 100
)

type parseLogTailOptions struct {
	parseSource      string
	parseRequested   int
	parseContains    string
	parseContainsLow string
}

type parseLogTailFile struct {
	parseSource string
	parsePath   string
}

type parseLogTailLine struct {
	parseSource    string
	parsePath      string
	parseTimestamp string
	parseLine      string
	parseOrder     int
}

// GetLogTail returns recent server and client log lines for authenticated superusers.
func (parseS *chatServer) GetLogTail(parseCtx context.Context, parseReq *chatpb.GetLogTailRequest) (*chatpb.GetLogTailResponse, error) {
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
		return nil, parseErr
	}
	if parseReq == nil {
		parseReq = &chatpb.GetLogTailRequest{}
	}

	parseOptions, parseErr := parseNormalizeLogTailOptions(parseReq)
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, parseErr.Error())
	}
	parseLogFiles := parseResolveLogTailFiles(parseOptions.parseSource)
	parseReadLimit := parseResolveLogTailReadLimit(parseOptions.parseRequested, parseOptions.parseContainsLow)

	parseCollected := make([]parseLogTailLine, 0, parseReadLimit)
	parseScannedLines := 0
	parseNextOrder := 0

	for _, parseLogFile := range parseLogFiles {
		parseLines, parseErr2 := parseReadTailLines(parseLogFile.parsePath, parseReadLimit, logTailMaximumReadBytes)
		if parseErr2 != nil {
			if errors.Is(parseErr2, os.ErrNotExist) {
				continue
			}
			return nil, status.Errorf(codes.Internal, "read %s log tail: %v", parseLogFile.parseSource, parseErr2)
		}
		parseScannedLines += len(parseLines)
		for _, parseLine := range parseLines {
			parseTrimmedLine := strings.TrimSpace(parseLine)
			if parseTrimmedLine == "" {
				continue
			}
			parseCollected = append(parseCollected, parseLogTailLine{
				parseSource:    parseLogFile.parseSource,
				parsePath:      parseLogFile.parsePath,
				parseTimestamp: parseExtractLogLineTimestamp(parseTrimmedLine),
				parseLine:      parseTrimmedLine,
				parseOrder:     parseNextOrder,
			})
			parseNextOrder++
		}
	}

	parseSorted := parseSortLogTailLines(parseCollected)
	parseFiltered := parseFilterLogTailLines(parseSorted, parseOptions.parseContainsLow)
	isParseTruncated := len(parseFiltered) > parseOptions.parseRequested
	if isParseTruncated {
		parseFiltered = parseFiltered[len(parseFiltered)-parseOptions.parseRequested:]
	}

	parseResponseEntries := make([]*chatpb.LogTailEntry, 0, len(parseFiltered))
	for _, parseLine := range parseFiltered {
		parseResponseEntries = append(parseResponseEntries, &chatpb.LogTailEntry{
			Source:    parseLine.parseSource,
			FilePath:  parseLine.parsePath,
			Timestamp: parseLine.parseTimestamp,
			Line:      parseLine.parseLine,
		})
	}
	parseFiles := make([]string, 0, len(parseLogFiles))
	for _, parseLogFile := range parseLogFiles {
		parseFiles = append(parseFiles, parseLogFile.parsePath)
	}

	parseLogger := parseS.logger.With(
		slog.String("rpc", "GetLogTail"),
		slog.String("log.source", parseOptions.parseSource),
		slog.Int("requested_max_lines", parseOptions.parseRequested),
	)
	if parseOptions.parseContains != "" {
		parseLogger = parseLogger.With(slog.String("filter.contains", parseOptions.parseContains))
	}
	parseLogger.Info("rpc.GetLogTail: complete",
		slog.Int("scanned_lines", parseScannedLines),
		slog.Int("returned_lines", len(parseResponseEntries)),
		slog.Bool("truncated", isParseTruncated),
	)

	return &chatpb.GetLogTailResponse{
		Entries:           parseResponseEntries,
		Source:            parseOptions.parseSource,
		RequestedMaxLines: int32(parseOptions.parseRequested),
		ScannedLines:      int32(parseScannedLines),
		ReturnedLines:     int32(len(parseResponseEntries)),
		Contains:          parseOptions.parseContains,
		Files:             parseFiles,
		Truncated:         isParseTruncated,
	}, nil
}

// parseNormalizeLogTailOptions normalizes and validates a GetLogTail request payload.
func parseNormalizeLogTailOptions(parseReq *chatpb.GetLogTailRequest) (parseLogTailOptions, error) {
	parseSource := strings.ToLower(strings.TrimSpace(parseReq.GetSource()))
	if parseSource == "" {
		parseSource = logTailSourceServer
	}
	switch parseSource {
	case logTailSourceServer, logTailSourceClient, logTailSourceAll:
	default:
		return parseLogTailOptions{}, errors.New("source must be one of: server, client, all")
	}

	parseRequested := int(parseReq.GetMaxLines())
	if parseRequested <= 0 {
		parseRequested = logTailDefaultMaxLines
	}
	if parseRequested > logTailMaximumMaxLines {
		parseRequested = logTailMaximumMaxLines
	}
	parseContains := strings.TrimSpace(parseReq.GetContains())

	return parseLogTailOptions{
		parseSource:      parseSource,
		parseRequested:   parseRequested,
		parseContains:    parseContains,
		parseContainsLow: strings.ToLower(parseContains),
	}, nil
}

// parseResolveLogTailFiles resolves the log files for one normalized source selector.
func parseResolveLogTailFiles(parseSource string) []parseLogTailFile {
	parseServerPath := filepath.Join(serverLogDir, serverLogFilename)
	parseClientPath := filepath.Join(serverLogDir, clientLogFilename)
	switch parseSource {
	case logTailSourceClient:
		return []parseLogTailFile{{parseSource: logTailSourceClient, parsePath: parseClientPath}}
	case logTailSourceAll:
		return []parseLogTailFile{
			{parseSource: logTailSourceServer, parsePath: parseServerPath},
			{parseSource: logTailSourceClient, parsePath: parseClientPath},
		}
	default:
		return []parseLogTailFile{{parseSource: logTailSourceServer, parsePath: parseServerPath}}
	}
}

// parseResolveLogTailReadLimit chooses a bounded scan depth based on requested lines and filter selectivity.
func parseResolveLogTailReadLimit(parseRequested int, parseContainsLower string) int {
	parseReadLimit := parseRequested + logTailUnfilteredScanBump
	if parseContainsLower != "" {
		parseReadLimit = parseRequested * logTailFilteredScanFactor
	}
	if parseReadLimit < logTailReadFallbackMinimum {
		parseReadLimit = logTailReadFallbackMinimum
	}
	if parseReadLimit > logTailMaximumScanLines {
		parseReadLimit = logTailMaximumScanLines
	}
	return parseReadLimit
}

// parseReadTailLines reads the latest lines from one file by scanning from the end with bounded memory.
func parseReadTailLines(parsePath string, parseMaxLines int, parseMaxBytes int64) ([]string, error) {
	if parseMaxLines <= 0 {
		return nil, nil
	}
	parseFile, parseErr := os.Open(parsePath)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseFile.Close()

	parseInfo, parseErr2 := parseFile.Stat()
	if parseErr2 != nil {
		return nil, parseErr2
	}
	parseFileSize := parseInfo.Size()
	if parseFileSize <= 0 {
		return nil, nil
	}
	if parseMaxBytes <= 0 {
		parseMaxBytes = logTailMaximumReadBytes
	}

	parseOffset := parseFileSize
	parseRemaining := parseMaxBytes
	parseBuffer := make([]byte, 0, parseMinInt64(parseFileSize, parseMaxBytes))
	for parseOffset > 0 && parseRemaining > 0 {
		parseChunkSize := min(min(int64(logTailReadChunkBytes), parseOffset), parseRemaining)
		parseOffset -= parseChunkSize
		parseChunk := make([]byte, parseChunkSize)
		if _, parseErr3 := parseFile.ReadAt(parseChunk, parseOffset); parseErr3 != nil && !errors.Is(parseErr3, io.EOF) {
			return nil, parseErr3
		}
		parseBuffer = append(parseChunk, parseBuffer...)
		parseRemaining -= parseChunkSize
		if bytes.Count(parseBuffer, []byte{'\n'}) >= parseMaxLines+1 {
			break
		}
	}

	if len(parseBuffer) == 0 {
		return nil, nil
	}
	isParsePartialFirstLine := parseOffset > 0 && parseBuffer[0] != '\n'
	parseContent := strings.ReplaceAll(string(parseBuffer), "\r\n", "\n")
	parseContent = strings.TrimRight(parseContent, "\n")
	if strings.TrimSpace(parseContent) == "" {
		return nil, nil
	}

	parseLines := strings.Split(parseContent, "\n")
	if isParsePartialFirstLine && len(parseLines) > 0 {
		parseLines = parseLines[1:]
	}
	if len(parseLines) > parseMaxLines {
		parseLines = parseLines[len(parseLines)-parseMaxLines:]
	}
	return parseLines, nil
}

// parseExtractLogLineTimestamp extracts the standard "timestamp" field from one JSON log line when available.
func parseExtractLogLineTimestamp(parseLine string) string {
	if !strings.HasPrefix(parseLine, "{") {
		return ""
	}
	parseEnvelope := struct {
		Timestamp string `json:"timestamp"`
	}{}
	if parseErr := json.Unmarshal([]byte(parseLine), &parseEnvelope); parseErr != nil {
		return ""
	}
	return strings.TrimSpace(parseEnvelope.Timestamp)
}

// parseSortLogTailLines returns a stable oldest-to-newest ordering with timestamp-aware merge behavior.
func parseSortLogTailLines(parseLines []parseLogTailLine) []parseLogTailLine {
	parseSorted := append([]parseLogTailLine(nil), parseLines...)
	sort.SliceStable(parseSorted, func(parseI int, parseJ int) bool {
		parseTimeI, isParseTimeIOk := parseParseLogTimestamp(parseSorted[parseI].parseTimestamp)
		parseTimeJ, isParseTimeJOk := parseParseLogTimestamp(parseSorted[parseJ].parseTimestamp)
		if isParseTimeIOk && isParseTimeJOk && !parseTimeI.Equal(parseTimeJ) {
			return parseTimeI.Before(parseTimeJ)
		}
		if isParseTimeIOk && !isParseTimeJOk {
			return true
		}
		if !isParseTimeIOk && isParseTimeJOk {
			return false
		}
		return parseSorted[parseI].parseOrder < parseSorted[parseJ].parseOrder
	})
	return parseSorted
}

// parseParseLogTimestamp parses one RFC3339-like timestamp value from a structured log line.
func parseParseLogTimestamp(parseTimestamp string) (time.Time, bool) {
	parseTimestamp = strings.TrimSpace(parseTimestamp)
	if parseTimestamp == "" {
		return time.Time{}, false
	}
	parseTime, parseErr := time.Parse(time.RFC3339Nano, parseTimestamp)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseTime, true
}

// parseFilterLogTailLines applies one case-insensitive substring filter to log lines.
func parseFilterLogTailLines(parseLines []parseLogTailLine, parseContainsLower string) []parseLogTailLine {
	if parseContainsLower == "" {
		return parseLines
	}
	parseFiltered := make([]parseLogTailLine, 0, len(parseLines))
	for _, parseLine := range parseLines {
		if strings.Contains(strings.ToLower(parseLine.parseLine), parseContainsLower) {
			parseFiltered = append(parseFiltered, parseLine)
		}
	}
	return parseFiltered
}

// parseMinInt64 returns the smaller of two int64 values.
func parseMinInt64(parseLeft int64, parseRight int64) int64 {
	if parseLeft < parseRight {
		return parseLeft
	}
	return parseRight
}
