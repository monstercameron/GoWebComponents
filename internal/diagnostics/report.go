package diagnostics

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

const frameworkModulePath = "github.com/monstercameron/GoWebComponents"
const frameworkWorkspaceName = "GoWebComponents"

const (
	visibleAppFrameLimit       = 3
	visibleFrameworkFrameLimit = 4
	visiblePlatformFrameLimit  = 1
)

type frame struct {
	Function string
	File     string
	Line     int
}

type Report struct {
	Summary         string
	Code            string
	Headline        string
	Where           string
	Path            string
	Error           string
	Runtime         string
	Next            string
	Docs            string
	AppFrames       []string
	FrameworkFrames []string
	PlatformFrames  []string
}

type Options struct {
	Summary       string
	Code          string
	Headline      string
	Path          string
	Runtime       string
	Next          string
	Docs          string
	SkipFunctions []string
}

// Build constructs a Report from the given Options, capturing and classifying a stack trace.
func Build(parseOptions Options) Report {
	parseReport := Report{
		Summary:  strings.TrimSpace(parseOptions.Summary),
		Code:     strings.TrimSpace(parseOptions.Code),
		Headline: strings.TrimSpace(parseOptions.Headline),
		Path:     strings.TrimSpace(parseOptions.Path),
		Runtime:  strings.TrimSpace(parseOptions.Runtime),
		Next:     strings.TrimSpace(parseOptions.Next),
		Docs:     strings.TrimSpace(parseOptions.Docs),
	}
	if parseReport.Summary == "" {
		parseReport.Summary = "error without message"
	}
	if parseReport.Error == "" {
		parseReport.Error = parseReport.Summary
	}
	parseFrames := parseFrames(debug.Stack(), parseOptions.SkipFunctions)
	for _, parseCurrent := range parseFrames {
		switch classifyFrame(parseCurrent) {
		case "app":
			parseReport.AppFrames = appendFrame(parseReport.AppFrames, parseCurrent)
		case "framework":
			parseReport.FrameworkFrames = appendFrame(parseReport.FrameworkFrames, parseCurrent)
		default:
			parseReport.PlatformFrames = appendFrame(parseReport.PlatformFrames, parseCurrent)
		}
	}
	if len(parseReport.AppFrames) > 0 {
		parseReport.Where = parseReport.AppFrames[0]
	} else {
		parseReport.Where = parseReport.Path
	}
	if parseReport.Where == "" {
		parseReport.Where = parseReport.Headline
	}
	if parseReport.Path == "" {
		parseReport.Path = parseReport.Where
	}
	parseReport.AppFrames = limitFrames(parseReport.AppFrames, visibleAppFrameLimit, "app")
	parseReport.FrameworkFrames = limitFrames(parseReport.FrameworkFrames, visibleFrameworkFrameLimit, "framework")
	parseReport.PlatformFrames = limitFrames(parseReport.PlatformFrames, visiblePlatformFrameLimit, "platform")
	return parseReport
}

func (parseReport Report) Formatted() string {
	parseLines := []string{parseReport.Summary}
	if parseReport.Code != "" && parseReport.Headline != "" {
		parseLines = append(parseLines, fmt.Sprintf("[%s] %s", parseReport.Code, parseReport.Headline))
	} else if parseReport.Code != "" {
		parseLines = append(parseLines, fmt.Sprintf("[%s]", parseReport.Code))
	} else if parseReport.Headline != "" {
		parseLines = append(parseLines, parseReport.Headline)
	}
	parseLines = append(parseLines,
		"where: "+strings.TrimSpace(parseReport.Where),
		"path: "+strings.TrimSpace(parseReport.Path),
		"error: "+strings.TrimSpace(parseReport.Error),
		"runtime: "+strings.TrimSpace(parseReport.Runtime),
		"next: "+strings.TrimSpace(parseReport.Next),
		"docs: "+strings.TrimSpace(parseReport.Docs),
	)
	if len(parseReport.AppFrames) > 0 || len(parseReport.FrameworkFrames) > 0 || len(parseReport.PlatformFrames) > 0 {
		parseLines = append(parseLines, "stack:")
		if len(parseReport.AppFrames) > 0 {
			parseLines = append(parseLines, "app:")
			parseLines = appendSection(parseLines, parseReport.AppFrames)
		}
		if len(parseReport.FrameworkFrames) > 0 {
			parseLines = append(parseLines, "framework: GWC")
			parseLines = appendSection(parseLines, parseReport.FrameworkFrames)
		}
		if len(parseReport.PlatformFrames) > 0 {
			parseLines = append(parseLines, "platform: GOLANG")
			parseLines = appendSection(parseLines, parseReport.PlatformFrames)
		}
	}
	return strings.Join(parseLines, "\n")
}

// FormattedPublic returns a client-safe rendering of the report: only the
// app-authored fields (summary, code/headline, next step, docs link). It
// deliberately OMITS where/path/error/runtime and every stack frame — those come
// from debug.Stack() and embed absolute build paths (which leak the OS username
// and the server's filesystem layout) plus internal call structure. That detail
// belongs in the server log (Emit → stderr), never in an HTTP response body sent
// to an untrusted client.
func (parseReport Report) FormattedPublic() string {
	parseLines := []string{parseReport.Summary}
	if parseReport.Code != "" && parseReport.Headline != "" {
		parseLines = append(parseLines, fmt.Sprintf("[%s] %s", parseReport.Code, parseReport.Headline))
	} else if parseReport.Code != "" {
		parseLines = append(parseLines, fmt.Sprintf("[%s]", parseReport.Code))
	} else if parseReport.Headline != "" {
		parseLines = append(parseLines, parseReport.Headline)
	}
	if parseNext := strings.TrimSpace(parseReport.Next); parseNext != "" {
		parseLines = append(parseLines, "next: "+parseNext)
	}
	if parseDocs := strings.TrimSpace(parseReport.Docs); parseDocs != "" {
		parseLines = append(parseLines, "docs: "+parseDocs)
	}
	return strings.Join(parseLines, "\n")
}

// Emit writes the formatted report to stderr.
func Emit(parseReport Report) {
	parseFormatted := strings.TrimSpace(parseReport.Formatted())
	if parseFormatted == "" {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, parseFormatted)
}

func parseFrames(parseStack []byte, parseExtraSkip []string) []frame {
	parseLines := strings.Split(strings.ReplaceAll(string(parseStack), "\r\n", "\n"), "\n")
	parseSkipFunctions := []string{
		"runtime/debug.Stack",
		"github.com/monstercameron/GoWebComponents/v4/internal/diagnostics.parseFrames",
		"github.com/monstercameron/GoWebComponents/v4/internal/diagnostics.Build",
		"github.com/monstercameron/GoWebComponents/v4/internal/diagnostics.WriteHTTPError",
		"github.com/monstercameron/GoWebComponents/v4/internal/diagnostics.Emit",
	}
	parseSkipFunctions = append(parseSkipFunctions, parseExtraSkip...)
	parseFrames := make([]frame, 0, len(parseLines)/2)
	for parseIndex := 1; parseIndex+1 < len(parseLines); parseIndex += 2 {
		parseFunction := strings.TrimSpace(parseLines[parseIndex])
		parseLocation := strings.TrimSpace(parseLines[parseIndex+1])
		if parseFunction == "" || parseLocation == "" {
			continue
		}
		isParseSkip := false
		for _, parseToken := range parseSkipFunctions {
			if parseToken != "" && strings.Contains(parseFunction, parseToken) {
				isParseSkip = true
				break
			}
		}
		if isParseSkip {
			continue
		}
		parseLocation = strings.TrimSpace(strings.SplitN(parseLocation, " +", 2)[0])
		parseLine := 0
		if parseLastColon := strings.LastIndex(parseLocation, ":"); parseLastColon > 0 {
			if parseParsed, parseErr := strconv.Atoi(parseLocation[parseLastColon+1:]); parseErr == nil {
				parseLine = parseParsed
				parseLocation = parseLocation[:parseLastColon]
			}
		}
		parseFrames = append(parseFrames, frame{Function: parseFunction, File: parseLocation, Line: parseLine})
	}
	return parseFrames
}

func sanitizeFunction(parseFunction string) string {
	parseTrimmed := strings.TrimSpace(parseFunction)
	if parseTrimmed == "" {
		return ""
	}
	if parseIndex := strings.Index(parseTrimmed, "("); parseIndex > 0 {
		parseTrimmed = parseTrimmed[:parseIndex]
	}
	return strings.TrimSpace(parseTrimmed)
}

func classifyFrame(parseCurrent frame) string {
	parseFunction := strings.ReplaceAll(sanitizeFunction(parseCurrent.Function), "\\", "/")
	parseFile := strings.ReplaceAll(parseCurrent.File, "\\", "/")
	if strings.HasSuffix(parseFile, "_test.go") || strings.Contains(parseFunction, ".Test") {
		return "app"
	}
	if strings.Contains(parseFunction, frameworkModulePath+"/") {
		if strings.Contains(parseFunction, frameworkModulePath+"/examples/") || strings.Contains(parseFunction, frameworkModulePath+"/test/") {
			return "app"
		}
		return "framework"
	}
	if strings.Contains(parseFile, "/"+frameworkWorkspaceName+"/") {
		if strings.Contains(parseFile, "/"+frameworkWorkspaceName+"/examples/") || strings.Contains(parseFile, "/"+frameworkWorkspaceName+"/test/") {
			return "app"
		}
		return "framework"
	}
	if strings.HasPrefix(parseFunction, "runtime.") ||
		strings.HasPrefix(parseFunction, "syscall/js.") ||
		strings.HasPrefix(parseFunction, "testing.") ||
		strings.Contains(parseFile, "/src/runtime/") ||
		strings.Contains(parseFile, "/src/testing/") ||
		strings.Contains(parseFile, "/src/syscall/js/") {
		return "platform"
	}
	return "app"
}

func shortenFilePath(parsePath string) string {
	parseNormalized := strings.ReplaceAll(strings.TrimSpace(parsePath), "\\", "/")
	if parseNormalized == "" {
		return ""
	}
	parseMarker := "/" + frameworkWorkspaceName + "/"
	if _, after, ok := strings.Cut(parseNormalized, parseMarker); ok {
		return after
	}
	parseParts := strings.Split(parseNormalized, "/")
	if len(parseParts) <= 3 {
		return parseNormalized
	}
	return strings.Join(parseParts[len(parseParts)-3:], "/")
}

func formatFrame(parseCurrent frame) string {
	parseFunction := sanitizeFunction(parseCurrent.Function)
	parseLocation := shortenFilePath(parseCurrent.File)
	if parseLocation == "" {
		return parseFunction
	}
	if parseCurrent.Line > 0 {
		return fmt.Sprintf("%s at %s:%d", parseFunction, parseLocation, parseCurrent.Line)
	}
	return fmt.Sprintf("%s at %s", parseFunction, parseLocation)
}

func appendFrame(parseTarget []string, parseCurrent frame) []string {
	parseFormatted := formatFrame(parseCurrent)
	if parseFormatted == "" {
		return parseTarget
	}
	return append(parseTarget, parseFormatted)
}

func limitFrames(parseValues []string, parseLimit int, parseLabel string) []string {
	if parseLimit <= 0 || len(parseValues) <= parseLimit {
		return parseValues
	}
	parseTrimmed := append([]string(nil), parseValues[:parseLimit]...)
	parseTrimmed = append(parseTrimmed, fmt.Sprintf("... %d more %s frames omitted", len(parseValues)-parseLimit, parseLabel))
	return parseTrimmed
}

func appendSection(parseLines []string, parseValues []string) []string {
	for _, parseValue := range parseValues {
		parseLines = append(parseLines, "  "+parseValue)
	}
	return parseLines
}
