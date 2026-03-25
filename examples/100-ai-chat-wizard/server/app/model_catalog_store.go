package app

import (
	"database/sql"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
)

type modelCatalogRow struct {
	ID                     string
	ProviderID             string
	ProviderLabel          string
	Label                  string
	Note                   string
	Description            string
	SupportsThinking       bool
	SupportsSpeech         bool
	InputPerMillionUSD     float64
	OutputPerMillionUSD    float64
	PricingCurrency        string
	MaxOutputTokens        int64
	ThroughputTokensPerSec float64
	OnboardingReady        bool
	IsDefault              bool
	UseForTitleGeneration  bool
	UseForMemoryExtraction bool
	SortOrder              int64
}

type modelCatalogConfig struct {
	ProviderCatalogs      map[string]provider.Catalog
	DefaultModel          string
	MemoryExtractionModel string
}

func (r modelCatalogRow) capabilities() provider.ModelCapabilities {
	return provider.ModelCapabilities{
		ProviderID:       r.ProviderID,
		ProviderLabel:    r.ProviderLabel,
		SupportsThinking: r.SupportsThinking,
		SupportsSpeech:   r.SupportsSpeech,
	}
}

func (r modelCatalogRow) metadata() provider.ModelMetadata {
	return provider.ModelMetadata{
		ID:                        r.ID,
		DisplayName:               r.Label,
		Description:               r.Description,
		ProviderID:                r.ProviderID,
		ProviderLabel:             r.ProviderLabel,
		ProviderFamily:            r.ProviderID,
		Capabilities:              r.capabilities(),
		StreamingSupported:        true,
		ReasoningSupported:        r.SupportsThinking,
		ToolUseSupported:          false,
		MaxOutputTokens:           r.MaxOutputTokens,
		ThroughputTokensPerSecond: r.ThroughputTokensPerSec,
		OnboardingReady:           r.OnboardingReady,
		Pricing: provider.ModelPricing{
			InputPerMillionUSD:  r.InputPerMillionUSD,
			OutputPerMillionUSD: r.OutputPerMillionUSD,
			Currency:            r.PricingCurrency,
		},
	}
}

func (r modelCatalogRow) option() provider.ModelOption {
	return provider.ModelOption{
		ID:           r.ID,
		Label:        r.Label,
		Note:         r.Note,
		Capabilities: r.capabilities(),
		Pricing: provider.ModelPricing{
			InputPerMillionUSD:  r.InputPerMillionUSD,
			OutputPerMillionUSD: r.OutputPerMillionUSD,
			Currency:            r.PricingCurrency,
		},
	}
}

func loadModelCatalogConfig(store *Store) (modelCatalogConfig, error) {
	rows, err := loadModelCatalogRows(store)
	if err != nil {
		return modelCatalogConfig{}, err
	}

	config := modelCatalogConfig{
		ProviderCatalogs: make(map[string]provider.Catalog),
	}
	firstModel := ""
	for _, row := range rows {
		modelID := normalizeSelectedModelID(row.ID)
		if modelID == "" {
			continue
		}
		row.ID = modelID
		providerID := strings.TrimSpace(strings.ToLower(row.ProviderID))
		if providerID == "" {
			continue
		}
		if firstModel == "" {
			firstModel = modelID
		}

		catalog := config.ProviderCatalogs[providerID]
		if catalog.DefaultModel == "" {
			catalog.DefaultModel = modelID
		}
		if row.IsDefault {
			catalog.DefaultModel = modelID
			config.DefaultModel = modelID
		}
		if row.UseForTitleGeneration && catalog.TitleModel == "" {
			catalog.TitleModel = modelID
		}
		if row.UseForMemoryExtraction && config.MemoryExtractionModel == "" {
			config.MemoryExtractionModel = modelID
		}
		catalog.Models = append(catalog.Models, row.metadata())
		catalog.Options = append(catalog.Options, row.option())
		config.ProviderCatalogs[providerID] = catalog
	}
	if config.DefaultModel == "" {
		config.DefaultModel = firstModel
	}
	if config.MemoryExtractionModel == "" {
		config.MemoryExtractionModel = config.DefaultModel
	}
	return config, nil
}

func loadModelCatalogRows(store *Store) ([]modelCatalogRow, error) {
	if store != nil {
		return store.listModelCatalog()
	}

	queries, err := loadStoreQueries()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", "file:chat-model-catalog?mode=memory&cache=private")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if _, err := db.Exec(queries.schema); err != nil {
		return nil, err
	}

	tempStore := &Store{db: db, queries: queries}
	return tempStore.listModelCatalog()
}
