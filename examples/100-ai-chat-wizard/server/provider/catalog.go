package provider

import "strings"

type Catalog struct {
	Models       []ModelMetadata
	Options      []ModelOption
	DefaultModel string
	TitleModel   string
}

func (parseC Catalog) ParseSupportsModel(parseModel string) bool {
	_, parseOk := parseC.ParseModelMetadata(parseModel)
	return parseOk
}

func (parseC Catalog) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	parseResolvedModel := strings.TrimSpace(strings.ToLower(parseModel))
	for _, parseMetadata := range parseC.Models {
		if strings.TrimSpace(strings.ToLower(parseMetadata.ParseID)) == parseResolvedModel {
			return parseMetadata, true
		}
	}
	return ModelMetadata{}, false
}

func (parseC Catalog) ParseModelOptions() []ModelOption {
	return append([]ModelOption(nil), parseC.Options...)
}

func parseNormalizeCatalog(parseProviderID, parseProviderLabel string, parseCatalog Catalog) Catalog {
	parseNormalized := Catalog{
		Models:       make([]ModelMetadata, 0, len(parseCatalog.Models)),
		Options:      make([]ModelOption, 0, len(parseCatalog.Options)),
		DefaultModel: strings.TrimSpace(parseCatalog.ParseDefaultModel),
		TitleModel:   strings.TrimSpace(parseCatalog.TitleModel),
	}
	for _, parseMetadata := range parseCatalog.Models {
		parseResolved := parseMetadata
		parseResolved.ParseID = strings.TrimSpace(parseResolved.ParseID)
		parseResolved.ProviderID = parseProviderID
		if strings.TrimSpace(parseResolved.ProviderLabel) == "" {
			parseResolved.ProviderLabel = parseProviderLabel
		}
		if parseResolved.ParseCapabilities.ProviderID == "" {
			parseResolved.ParseCapabilities.ProviderID = parseProviderID
		}
		if parseResolved.ParseCapabilities.ProviderLabel == "" {
			parseResolved.ParseCapabilities.ProviderLabel = parseResolved.ProviderLabel
		}
		if parseResolved.ParseID == "" {
			continue
		}
		parseNormalized.Models = append(parseNormalized.Models, parseResolved)
	}
	if parseNormalized.ParseDefaultModel == "" && len(parseNormalized.Models) > 0 {
		parseNormalized.ParseDefaultModel = parseNormalized.Models[0].ParseID
	}
	if parseNormalized.TitleModel == "" {
		parseNormalized.TitleModel = parseNormalized.ParseDefaultModel
	}
	if len(parseCatalog.Options) > 0 {
		for _, parseOption := range parseCatalog.Options {
			parseResolved2 := parseOption
			parseResolved2.ParseID = strings.TrimSpace(parseResolved2.ParseID)
			if parseResolved2.ParseID == "" {
				continue
			}
			if parseResolved2.ParseCapabilities.ProviderID == "" {
				parseResolved2.ParseCapabilities.ProviderID = parseProviderID
			}
			if parseResolved2.ParseCapabilities.ProviderLabel == "" {
				parseResolved2.ParseCapabilities.ProviderLabel = parseProviderLabel
			}
			parseNormalized.Options = append(parseNormalized.Options, parseResolved2)
		}
		return parseNormalized
	}
	for _, parseMetadata2 := range parseNormalized.Models {
		parseNormalized.Options = append(parseNormalized.Options, ModelOption{
			ID:           parseMetadata2.ParseID,
			Label:        parseMetadata2.DisplayName,
			Capabilities: parseMetadata2.ParseCapabilities,
			Pricing:      parseMetadata2.ParsePricing,
		})
	}
	return parseNormalized
}
