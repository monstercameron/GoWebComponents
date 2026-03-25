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
func Render(source string) (string, error) {
	var htmlBuffer bytes.Buffer
	if err := renderer.Convert([]byte(normalizeFenceLanguageAliases(source)), &htmlBuffer); err != nil {
		return "", err
	}
	return htmlBuffer.String(), nil
}

func normalizeFenceLanguageAliases(source string) string {
	if !strings.Contains(source, "```") && !strings.Contains(source, "~~~") {
		return source
	}

	lines := strings.SplitAfter(source, "\n")
	if len(lines) == 0 {
		return source
	}

	var builder strings.Builder
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for _, line := range lines {
		trimmedLine := strings.TrimRight(line, "\r\n")
		newline := line[len(trimmedLine):]
		leading := len(trimmedLine) - len(strings.TrimLeft(trimmedLine, " "))
		content := trimmedLine
		if leading <= 3 {
			content = strings.TrimLeft(trimmedLine, " ")
		}

		if leading <= 3 && len(content) >= 3 && (content[0] == '`' || content[0] == '~') {
			markerLen := leadingFenceLength(content)
			if markerLen >= 3 {
				markerChar := content[0]
				if !inFence {
					builder.WriteString(trimmedLine[:leading])
					builder.WriteString(strings.Repeat(string(markerChar), markerLen))
					info := strings.TrimSpace(content[markerLen:])
					if info != "" {
						builder.WriteByte(' ')
						builder.WriteString(normalizeFenceInfoString(info))
					}
					builder.WriteString(newline)
					inFence = true
					fenceChar = markerChar
					fenceLen = markerLen
					continue
				}
				if markerChar == fenceChar && markerLen >= fenceLen {
					inFence = false
					fenceChar = 0
					fenceLen = 0
				}
			}
		}

		builder.WriteString(line)
	}

	return builder.String()
}

func leadingFenceLength(content string) int {
	if len(content) == 0 {
		return 0
	}
	markerChar := content[0]
	length := 0
	for length < len(content) && content[length] == markerChar {
		length++
	}
	return length
}

func normalizeFenceInfoString(info string) string {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return info
	}
	if normalized, ok := fenceLanguageAliases[strings.ToLower(fields[0])]; ok {
		fields[0] = normalized
	}
	return strings.Join(fields, " ")
}
