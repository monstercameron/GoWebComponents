package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed scripts/livereload-client.txt
var embeddedLivereloadClientScript string

func livereloadClientScriptBytes(path string) ([]byte, error) {
	overridePath := strings.TrimSpace(path)
	if overridePath != "" {
		content, err := os.ReadFile(overridePath)
		if err != nil {
			return nil, err
		}
		return content, nil
	}

	if strings.TrimSpace(embeddedLivereloadClientScript) == "" {
		return nil, fmt.Errorf("embedded livereload client script is empty")
	}

	return []byte(embeddedLivereloadClientScript), nil
}
