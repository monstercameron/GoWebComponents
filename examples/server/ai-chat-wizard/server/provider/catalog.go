package provider

import "strings"

type Catalog struct {
	Models       []ModelMetadata
	Options      []ModelOption
	DefaultModel string
	TitleModel   string
}

// ParseSupportsModel reports whether the catalog supports the requested model.
func (parseC Catalog) ParseSupportsModel(parseModel string) bool {
	_, parseOk := parseC.ParseModelMetadata(parseModel)
	return parseOk
}

// ParseModelMetadata returns the catalog metadata for one model.
func (parseC Catalog) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	parseResolvedModel := strings.TrimSpace(strings.ToLower(parseModel))
	for _, parseMetadata := range parseC.Models {
		if strings.TrimSpace(strings.ToLower(parseMetadata.ID)) == parseResolvedModel {
			return parseMetadata, true
		}
	}
	return ModelMetadata{}, false
}

// ParseModelOptions returns the catalog model picker options.
func (parseC Catalog) ParseModelOptions() []ModelOption {
	return append([]ModelOption(nil), parseC.Options...)
}

func parseNormalizeCatalog(parseProviderID, parseProviderLabel string, parseCatalog Catalog) Catalog {
	parseNormalized := Catalog{
		Models:       make([]ModelMetadata, 0, len(parseCatalog.Models)),
		Options:      make([]ModelOption, 0, len(parseCatalog.Options)),
		DefaultModel: strings.TrimSpace(parseCatalog.DefaultModel),
		TitleModel:   strings.TrimSpace(parseCatalog.TitleModel),
	}
	for _, parseMetadata := range parseCatalog.Models {
		parseResolved := parseMetadata
		parseResolved.ID = strings.TrimSpace(parseResolved.ID)
		parseResolved.ProviderID = parseProviderID
		if strings.TrimSpace(parseResolved.ProviderLabel) == "" {
			parseResolved.ProviderLabel = parseProviderLabel
		}
		if parseResolved.Capabilities.ProviderID == "" {
			parseResolved.Capabilities.ProviderID = parseProviderID
		}
		if parseResolved.Capabilities.ProviderLabel == "" {
			parseResolved.Capabilities.ProviderLabel = parseResolved.ProviderLabel
		}
		if parseResolved.ID == "" {
			continue
		}
		parseNormalized.Models = append(parseNormalized.Models, parseResolved)
	}
	if parseNormalized.DefaultModel == "" && len(parseNormalized.Models) > 0 {
		parseNormalized.DefaultModel = parseNormalized.Models[0].ID
	}
	if parseNormalized.TitleModel == "" {
		parseNormalized.TitleModel = parseNormalized.DefaultModel
	}
	if len(parseCatalog.Options) > 0 {
		for _, parseOption := range parseCatalog.Options {
			parseResolved2 := parseOption
			parseResolved2.ID = strings.TrimSpace(parseResolved2.ID)
			if parseResolved2.ID == "" {
				continue
			}
			if parseResolved2.Capabilities.ProviderID == "" {
				parseResolved2.Capabilities.ProviderID = parseProviderID
			}
			if parseResolved2.Capabilities.ProviderLabel == "" {
				parseResolved2.Capabilities.ProviderLabel = parseProviderLabel
			}
			parseNormalized.Options = append(parseNormalized.Options, parseResolved2)
		}
		return parseNormalized
	}
	for _, parseMetadata2 := range parseNormalized.Models {
		parseNormalized.Options = append(parseNormalized.Options, ModelOption{
			ID:           parseMetadata2.ID,
			Label:        parseMetadata2.DisplayName,
			Capabilities: parseMetadata2.Capabilities,
			Pricing:      parseMetadata2.Pricing,
		})
	}
	return parseNormalized
}
