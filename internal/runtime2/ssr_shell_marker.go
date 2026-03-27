package runtime2

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// SSRShellMarkerVersionV1 identifies the first runtime2 SSR shell marker schema.
	SSRShellMarkerVersionV1 = "gwc.runtime2.ssr-shell.v1"
	// SSRShellMarkerAttribute identifies the host attribute key used for runtime2 SSR shell marker payloads.
	SSRShellMarkerAttribute = "data-gwc-runtime2-shell"
)

// SSRShellMarker stores one runtime2 SSR shell marker payload.
type SSRShellMarker struct {
	Version          string           `json:"version"`
	RegionInstanceID RegionInstanceID `json:"region_instance_id"`
	RendererID       RendererID       `json:"renderer_id"`
}

// ParseSSRShellMarkerVersion validates one runtime2 SSR shell marker version string.
func ParseSSRShellMarkerVersion(parseRaw string) (string, error) {
	switch parseRaw {
	case SSRShellMarkerVersionV1:
		return parseRaw, nil
	case "":
		return "", fmt.Errorf("runtime2: SSR shell marker version is required")
	default:
		return "", fmt.Errorf("runtime2: SSR shell marker version %q is unsupported", parseRaw)
	}
}

// ValidateSSRShellMarker verifies one runtime2 SSR shell marker is internally consistent.
func ValidateSSRShellMarker(parseMarker SSRShellMarker) error {
	if _, parseErr := ParseSSRShellMarkerVersion(parseMarker.Version); parseErr != nil {
		return parseErr
	}
	if _, parseErr := ParseRegionInstanceID(string(parseMarker.RegionInstanceID)); parseErr != nil {
		return parseErr
	}
	if _, parseErr := ParseRendererID(string(parseMarker.RendererID)); parseErr != nil {
		return parseErr
	}
	return nil
}

// BuildSSRShellMarkerAttributeValue encodes one validated SSR shell marker into the attribute payload format.
func BuildSSRShellMarkerAttributeValue(parseMarker SSRShellMarker) (string, error) {
	if parseErr := ValidateSSRShellMarker(parseMarker); parseErr != nil {
		return "", parseErr
	}
	parsePayload, parseErr := json.Marshal(parseMarker)
	if parseErr != nil {
		return "", fmt.Errorf("runtime2: encode SSR shell marker payload: %w", parseErr)
	}
	return string(parsePayload), nil
}

// ParseSSRShellMarkerAttributeValue decodes and validates one SSR shell marker attribute payload.
func ParseSSRShellMarkerAttributeValue(parsePayload string) (SSRShellMarker, error) {
	if strings.TrimSpace(parsePayload) == "" {
		return SSRShellMarker{}, fmt.Errorf("runtime2: SSR shell marker payload is required")
	}
	var parseMarker SSRShellMarker
	if parseErr := json.Unmarshal([]byte(parsePayload), &parseMarker); parseErr != nil {
		return SSRShellMarker{}, fmt.Errorf("runtime2: decode SSR shell marker payload: %w", parseErr)
	}
	if parseErr := ValidateSSRShellMarker(parseMarker); parseErr != nil {
		return SSRShellMarker{}, parseErr
	}
	return parseMarker, nil
}
