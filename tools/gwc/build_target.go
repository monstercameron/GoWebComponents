package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/desktop"
)

// normalizeDesktopFeatures validates and canonicalizes a native feature ceiling.
func normalizeDesktopFeatures(parseFeatures string) (string, error) {
	parseFeatures = strings.TrimSpace(parseFeatures)
	if parseFeatures == "" {
		return "", errors.New("desktop features must not be empty")
	}
	parsePolicy, parseErr := desktop.ParseFeaturePolicy(parseFeatures)
	if parseErr != nil {
		return "", parseErr
	}
	if strings.EqualFold(parseFeatures, "all") || strings.EqualFold(parseFeatures, "none") {
		return strings.ToLower(parseFeatures), nil
	}
	parseNames := parsePolicy.FeatureNames()
	parseValues := make([]string, 0, len(parseNames))
	for _, parseName := range parseNames {
		parseValues = append(parseValues, string(parseName))
	}
	return strings.Join(parseValues, ","), nil
}

// normalizeBuildTarget validates the explicit web or desktop build target.
func normalizeBuildTarget(parseTarget string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(parseTarget)) {
	case "", "web":
		return "web", nil
	case "desktop":
		return "desktop", nil
	default:
		return "", fmt.Errorf("unknown build target %q: want web or desktop", parseTarget)
	}
}

// applyBuildTarget adds the desktop compile tag while preserving profile flags.
func applyBuildTarget(parseProfile buildProfile, parseTarget string) (buildProfile, error) {
	parseNormalized, parseErr := normalizeBuildTarget(parseTarget)
	if parseErr != nil {
		return buildProfile{}, parseErr
	}
	if parseNormalized != "desktop" {
		for _, parseTag := range strings.FieldsFunc(parseProfile.Tags, func(parseRune rune) bool { return parseRune == ',' || parseRune == ' ' }) {
			if parseTag == "gwc_desktop" {
				return buildProfile{}, errors.New("web target cannot use the gwc_desktop build tag")
			}
		}
		return parseProfile, nil
	}
	if strings.EqualFold(strings.TrimSpace(parseProfile.Toolchain), "tinygo") {
		return buildProfile{}, errors.New("desktop target requires the Go toolchain")
	}
	parseTags := strings.FieldsFunc(parseProfile.Tags, func(parseRune rune) bool { return parseRune == ',' || parseRune == ' ' })
	for _, parseTag := range parseTags {
		if parseTag == "gwc_desktop" {
			return parseProfile, nil
		}
	}
	parseTags = append(parseTags, "gwc_desktop")
	parseProfile.Tags = strings.Join(slices.Compact(parseTags), ",")
	return parseProfile, nil
}
