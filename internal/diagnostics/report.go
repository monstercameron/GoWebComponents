package diagnostics

import (
	"fmt"
	"net/http"
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
func Build(options Options) Report {
	report := Report{
		Summary:  strings.TrimSpace(options.Summary),
		Code:     strings.TrimSpace(options.Code),
		Headline: strings.TrimSpace(options.Headline),
		Path:     strings.TrimSpace(options.Path),
		Runtime:  strings.TrimSpace(options.Runtime),
		Next:     strings.TrimSpace(options.Next),
		Docs:     strings.TrimSpace(options.Docs),
	}
	if report.Summary == "" {
		report.Summary = "error without message"
	}
	if report.Error == "" {
		report.Error = report.Summary
	}
	frames := parseFrames(debug.Stack(), options.SkipFunctions)
	for _, current := range frames {
		switch classifyFrame(current) {
		case "app":
			report.AppFrames = appendFrame(report.AppFrames, current)
		case "framework":
			report.FrameworkFrames = appendFrame(report.FrameworkFrames, current)
		default:
			report.PlatformFrames = appendFrame(report.PlatformFrames, current)
		}
	}
	if len(report.AppFrames) > 0 {
		report.Where = report.AppFrames[0]
	} else {
		report.Where = report.Path
	}
	if report.Where == "" {
		report.Where = report.Headline
	}
	if report.Path == "" {
		report.Path = report.Where
	}
	report.AppFrames = limitFrames(report.AppFrames, visibleAppFrameLimit, "app")
	report.FrameworkFrames = limitFrames(report.FrameworkFrames, visibleFrameworkFrameLimit, "framework")
	report.PlatformFrames = limitFrames(report.PlatformFrames, visiblePlatformFrameLimit, "platform")
	return report
}

func (report Report) Formatted() string {
	lines := []string{report.Summary}
	if report.Code != "" && report.Headline != "" {
		lines = append(lines, fmt.Sprintf("[%s] %s", report.Code, report.Headline))
	} else if report.Code != "" {
		lines = append(lines, fmt.Sprintf("[%s]", report.Code))
	} else if report.Headline != "" {
		lines = append(lines, report.Headline)
	}
	lines = append(lines,
		"where: "+strings.TrimSpace(report.Where),
		"path: "+strings.TrimSpace(report.Path),
		"error: "+strings.TrimSpace(report.Error),
		"runtime: "+strings.TrimSpace(report.Runtime),
		"next: "+strings.TrimSpace(report.Next),
		"docs: "+strings.TrimSpace(report.Docs),
	)
	if len(report.AppFrames) > 0 || len(report.FrameworkFrames) > 0 || len(report.PlatformFrames) > 0 {
		lines = append(lines, "stack:")
		if len(report.AppFrames) > 0 {
			lines = append(lines, "app:")
			lines = appendSection(lines, report.AppFrames)
		}
		if len(report.FrameworkFrames) > 0 {
			lines = append(lines, "framework: GWC")
			lines = appendSection(lines, report.FrameworkFrames)
		}
		if len(report.PlatformFrames) > 0 {
			lines = append(lines, "platform: GOLANG")
			lines = appendSection(lines, report.PlatformFrames)
		}
	}
	return strings.Join(lines, "\n")
}

// Emit writes the formatted report to stderr.
func Emit(report Report) {
	formatted := strings.TrimSpace(report.Formatted())
	if formatted == "" {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, formatted)
}

// WriteHTTPError emits the report and writes it as a plain-text HTTP error response.
func WriteHTTPError(w http.ResponseWriter, status int, report Report) {
	Emit(report)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(report.Formatted()))
}

func parseFrames(stack []byte, extraSkip []string) []frame {
	lines := strings.Split(strings.ReplaceAll(string(stack), "\r\n", "\n"), "\n")
	skipFunctions := []string{
		"runtime/debug.Stack",
		"github.com/monstercameron/GoWebComponents/internal/diagnostics.parseFrames",
		"github.com/monstercameron/GoWebComponents/internal/diagnostics.Build",
		"github.com/monstercameron/GoWebComponents/internal/diagnostics.WriteHTTPError",
		"github.com/monstercameron/GoWebComponents/internal/diagnostics.Emit",
	}
	skipFunctions = append(skipFunctions, extraSkip...)
	frames := make([]frame, 0, len(lines)/2)
	for index := 1; index+1 < len(lines); index += 2 {
		function := strings.TrimSpace(lines[index])
		location := strings.TrimSpace(lines[index+1])
		if function == "" || location == "" {
			continue
		}
		skip := false
		for _, token := range skipFunctions {
			if token != "" && strings.Contains(function, token) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		location = strings.TrimSpace(strings.SplitN(location, " +", 2)[0])
		line := 0
		if lastColon := strings.LastIndex(location, ":"); lastColon > 0 {
			if parsed, err := strconv.Atoi(location[lastColon+1:]); err == nil {
				line = parsed
				location = location[:lastColon]
			}
		}
		frames = append(frames, frame{Function: function, File: location, Line: line})
	}
	return frames
}

func sanitizeFunction(function string) string {
	trimmed := strings.TrimSpace(function)
	if trimmed == "" {
		return ""
	}
	if index := strings.Index(trimmed, "("); index > 0 {
		trimmed = trimmed[:index]
	}
	return strings.TrimSpace(trimmed)
}

func classifyFrame(current frame) string {
	function := strings.ReplaceAll(sanitizeFunction(current.Function), "\\", "/")
	file := strings.ReplaceAll(current.File, "\\", "/")
	if strings.HasSuffix(file, "_test.go") || strings.Contains(function, ".Test") {
		return "app"
	}
	if strings.Contains(function, frameworkModulePath+"/") {
		if strings.Contains(function, frameworkModulePath+"/examples/") || strings.Contains(function, frameworkModulePath+"/test/") {
			return "app"
		}
		return "framework"
	}
	if strings.Contains(file, "/"+frameworkWorkspaceName+"/") {
		if strings.Contains(file, "/"+frameworkWorkspaceName+"/examples/") || strings.Contains(file, "/"+frameworkWorkspaceName+"/test/") {
			return "app"
		}
		return "framework"
	}
	if strings.HasPrefix(function, "runtime.") ||
		strings.HasPrefix(function, "syscall/js.") ||
		strings.HasPrefix(function, "testing.") ||
		strings.Contains(file, "/src/runtime/") ||
		strings.Contains(file, "/src/testing/") ||
		strings.Contains(file, "/src/syscall/js/") {
		return "platform"
	}
	return "app"
}

func shortenFilePath(path string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	if normalized == "" {
		return ""
	}
	marker := "/" + frameworkWorkspaceName + "/"
	if index := strings.Index(normalized, marker); index >= 0 {
		return normalized[index+len(marker):]
	}
	parts := strings.Split(normalized, "/")
	if len(parts) <= 3 {
		return normalized
	}
	return strings.Join(parts[len(parts)-3:], "/")
}

func formatFrame(current frame) string {
	function := sanitizeFunction(current.Function)
	location := shortenFilePath(current.File)
	if location == "" {
		return function
	}
	if current.Line > 0 {
		return fmt.Sprintf("%s at %s:%d", function, location, current.Line)
	}
	return fmt.Sprintf("%s at %s", function, location)
}

func appendFrame(target []string, current frame) []string {
	formatted := formatFrame(current)
	if formatted == "" {
		return target
	}
	return append(target, formatted)
}

func limitFrames(values []string, limit int, label string) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	trimmed := append([]string(nil), values[:limit]...)
	trimmed = append(trimmed, fmt.Sprintf("... %d more %s frames omitted", len(values)-limit, label))
	return trimmed
}

func appendSection(lines []string, values []string) []string {
	for _, value := range values {
		lines = append(lines, "  "+value)
	}
	return lines
}
