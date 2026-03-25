package provider

import "strings"

type Catalog struct {
	Models       []ModelMetadata
	Options      []ModelOption
	DefaultModel string
	TitleModel   string
}

func (c Catalog) SupportsModel(model string) bool {
	_, ok := c.ModelMetadata(model)
	return ok
}

func (c Catalog) ModelMetadata(model string) (ModelMetadata, bool) {
	resolvedModel := strings.TrimSpace(strings.ToLower(model))
	for _, metadata := range c.Models {
		if strings.TrimSpace(strings.ToLower(metadata.ID)) == resolvedModel {
			return metadata, true
		}
	}
	return ModelMetadata{}, false
}

func (c Catalog) ModelOptions() []ModelOption {
	return append([]ModelOption(nil), c.Options...)
}

func normalizeCatalog(providerID, providerLabel string, catalog Catalog) Catalog {
	normalized := Catalog{
		Models:       make([]ModelMetadata, 0, len(catalog.Models)),
		Options:      make([]ModelOption, 0, len(catalog.Options)),
		DefaultModel: strings.TrimSpace(catalog.DefaultModel),
		TitleModel:   strings.TrimSpace(catalog.TitleModel),
	}
	for _, metadata := range catalog.Models {
		resolved := metadata
		resolved.ID = strings.TrimSpace(resolved.ID)
		resolved.ProviderID = providerID
		if strings.TrimSpace(resolved.ProviderLabel) == "" {
			resolved.ProviderLabel = providerLabel
		}
		if resolved.Capabilities.ProviderID == "" {
			resolved.Capabilities.ProviderID = providerID
		}
		if resolved.Capabilities.ProviderLabel == "" {
			resolved.Capabilities.ProviderLabel = resolved.ProviderLabel
		}
		if resolved.ID == "" {
			continue
		}
		normalized.Models = append(normalized.Models, resolved)
	}
	if normalized.DefaultModel == "" && len(normalized.Models) > 0 {
		normalized.DefaultModel = normalized.Models[0].ID
	}
	if normalized.TitleModel == "" {
		normalized.TitleModel = normalized.DefaultModel
	}
	if len(catalog.Options) > 0 {
		for _, option := range catalog.Options {
			resolved := option
			resolved.ID = strings.TrimSpace(resolved.ID)
			if resolved.ID == "" {
				continue
			}
			if resolved.Capabilities.ProviderID == "" {
				resolved.Capabilities.ProviderID = providerID
			}
			if resolved.Capabilities.ProviderLabel == "" {
				resolved.Capabilities.ProviderLabel = providerLabel
			}
			normalized.Options = append(normalized.Options, resolved)
		}
		return normalized
	}
	for _, metadata := range normalized.Models {
		normalized.Options = append(normalized.Options, ModelOption{
			ID:           metadata.ID,
			Label:        metadata.DisplayName,
			Capabilities: metadata.Capabilities,
			Pricing:      metadata.Pricing,
		})
	}
	return normalized
}
