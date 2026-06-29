package markdownrender

import (
	"bytes"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

var renderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github-dark"),
			highlighting.WithGuessLanguage(true),
			highlighting.WithFormatOptions(
				chromahtml.TabWidth(2),
			),
		),
	),
	goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
)

var fenceLanguageAliases = map[string]string{
	".net":       "csharp",
	"asp.net":    "csharp",
	"c#":         "csharp",
	"c++":        "cpp",
	"cc":         "cpp",
	"cs":         "csharp",
	"cxx":        "cpp",
	"dotnet":     "csharp",
	"golang":     "go",
	"js":         "javascript",
	"mmd":        "mermaid",
	"mermaid":    "mermaid",
	"node":       "javascript",
	"rs":         "rust",
	"sh":         "bash",
	"shell":      "bash",
	"typescript": "typescript",
	"vb.net":     "vbnet",
	"zsh":        "bash",
}

// Render converts markdown into HTML with fenced-code syntax highlighting.
func Render(parseSource string) (string, error) {
	var parseHtmlBuffer bytes.Buffer
	if parseErr := renderer.Convert([]byte(parseNormalizeFenceLanguageAliases(parseSource)), &parseHtmlBuffer); parseErr != nil {
		return "", parseErr
	}
	return parseHtmlBuffer.String(), nil
}

func parseNormalizeFenceLanguageAliases(parseSource string) string {
	if !strings.Contains(parseSource, "```") && !strings.Contains(parseSource, "~~~") {
		return parseSource
	}

	parseLines := strings.SplitAfter(parseSource, "\n")
	if len(parseLines) == 0 {
		return parseSource
	}

	var parseBuilder strings.Builder
	isParseInFence := false
	parseFenceChar := byte(0)
	parseFenceLen := 0

	for _, parseLine := range parseLines {
		parseTrimmedLine := strings.TrimRight(parseLine, "\r\n")
		parseNewline := parseLine[len(parseTrimmedLine):]
		parseLeading := len(parseTrimmedLine) - len(strings.TrimLeft(parseTrimmedLine, " "))
		parseContent := parseTrimmedLine
		if parseLeading <= 3 {
			parseContent = strings.TrimLeft(parseTrimmedLine, " ")
		}

		if parseLeading <= 3 && len(parseContent) >= 3 && (parseContent[0] == '`' || parseContent[0] == '~') {
			parseMarkerLen := parseLeadingFenceLength(parseContent)
			if parseMarkerLen >= 3 {
				parseMarkerChar := parseContent[0]
				if !isParseInFence {
					parseBuilder.WriteString(parseTrimmedLine[:parseLeading])
					parseBuilder.WriteString(strings.Repeat(string(parseMarkerChar), parseMarkerLen))
					parseInfo := strings.TrimSpace(parseContent[parseMarkerLen:])
					if parseInfo != "" {
						parseBuilder.WriteByte(' ')
						parseBuilder.WriteString(parseNormalizeFenceInfoString(parseInfo))
					}
					parseBuilder.WriteString(parseNewline)
					isParseInFence = true
					parseFenceChar = parseMarkerChar
					parseFenceLen = parseMarkerLen
					continue
				}
				if parseMarkerChar == parseFenceChar && parseMarkerLen >= parseFenceLen {
					isParseInFence = false
					parseFenceChar = 0
					parseFenceLen = 0
				}
			}
		}

		parseBuilder.WriteString(parseLine)
	}

	return parseBuilder.String()
}

func parseLeadingFenceLength(parseContent string) int {
	if len(parseContent) == 0 {
		return 0
	}
	parseMarkerChar := parseContent[0]
	parseLength := 0
	for parseLength < len(parseContent) && parseContent[parseLength] == parseMarkerChar {
		parseLength++
	}
	return parseLength
}

func parseNormalizeFenceInfoString(parseInfo string) string {
	parseFields := strings.Fields(parseInfo)
	if len(parseFields) == 0 {
		return parseInfo
	}
	if parseNormalized, parseOk := fenceLanguageAliases[strings.ToLower(parseFields[0])]; parseOk {
		parseFields[0] = parseNormalized
	}
	return strings.Join(parseFields, " ")
}
