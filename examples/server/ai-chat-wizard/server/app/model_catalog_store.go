package app

import (
	"database/sql"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/server/provider"
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

func (parseR modelCatalogRow) parseCapabilities() provider.ModelCapabilities {
	return provider.ModelCapabilities{
		ProviderID:       parseR.ProviderID,
		ProviderLabel:    parseR.ProviderLabel,
		SupportsThinking: parseR.SupportsThinking,
		SupportsSpeech:   parseR.SupportsSpeech,
	}
}

func (parseR modelCatalogRow) parseMetadata() provider.ModelMetadata {
	return provider.ModelMetadata{
		ID:                        parseR.ID,
		DisplayName:               parseR.Label,
		Description:               parseR.Description,
		ProviderID:                parseR.ProviderID,
		ProviderLabel:             parseR.ProviderLabel,
		ProviderFamily:            parseR.ProviderID,
		Capabilities:              parseR.parseCapabilities(),
		StreamingSupported:        true,
		ReasoningSupported:        parseR.SupportsThinking,
		ToolUseSupported:          false,
		MaxOutputTokens:           parseR.MaxOutputTokens,
		ThroughputTokensPerSecond: parseR.ThroughputTokensPerSec,
		OnboardingReady:           parseR.OnboardingReady,
		Pricing: provider.ModelPricing{
			InputPerMillionUSD:  parseR.InputPerMillionUSD,
			OutputPerMillionUSD: parseR.OutputPerMillionUSD,
			Currency:            parseR.PricingCurrency,
		},
	}
}

func (parseR modelCatalogRow) parseOption() provider.ModelOption {
	return provider.ModelOption{
		ID:           parseR.ID,
		Label:        parseR.Label,
		Note:         parseR.Note,
		Capabilities: parseR.parseCapabilities(),
		Pricing: provider.ModelPricing{
			InputPerMillionUSD:  parseR.InputPerMillionUSD,
			OutputPerMillionUSD: parseR.OutputPerMillionUSD,
			Currency:            parseR.PricingCurrency,
		},
	}
}

func parseLoadModelCatalogConfig(store *Store) (modelCatalogConfig, error) {
	parseRows, parseErr := parseLoadModelCatalogRows(store)
	if parseErr != nil {
		return modelCatalogConfig{}, parseErr
	}

	parseConfig := modelCatalogConfig{
		ProviderCatalogs: make(map[string]provider.Catalog),
	}
	parseFirstModel := ""
	for _, parseRow := range parseRows {
		parseModelID := parseNormalizeSelectedModelID(parseRow.ID)
		if parseModelID == "" {
			continue
		}
		parseRow.ID = parseModelID
		parseProviderID := strings.TrimSpace(strings.ToLower(parseRow.ProviderID))
		if parseProviderID == "" {
			continue
		}
		if parseFirstModel == "" {
			parseFirstModel = parseModelID
		}

		parseCatalog := parseConfig.ProviderCatalogs[parseProviderID]
		if parseCatalog.DefaultModel == "" {
			parseCatalog.DefaultModel = parseModelID
		}
		if parseRow.IsDefault {
			parseCatalog.DefaultModel = parseModelID
			parseConfig.DefaultModel = parseModelID
		}
		if parseRow.UseForTitleGeneration && parseCatalog.TitleModel == "" {
			parseCatalog.TitleModel = parseModelID
		}
		if parseRow.UseForMemoryExtraction && parseConfig.MemoryExtractionModel == "" {
			parseConfig.MemoryExtractionModel = parseModelID
		}
		parseCatalog.Models = append(parseCatalog.Models, parseRow.parseMetadata())
		parseCatalog.Options = append(parseCatalog.Options, parseRow.parseOption())
		parseConfig.ProviderCatalogs[parseProviderID] = parseCatalog
	}
	if parseConfig.DefaultModel == "" {
		parseConfig.DefaultModel = parseFirstModel
	}
	if parseConfig.MemoryExtractionModel == "" {
		parseConfig.MemoryExtractionModel = parseConfig.DefaultModel
	}
	return parseConfig, nil
}

func parseLoadModelCatalogRows(store *Store) ([]modelCatalogRow, error) {
	if store != nil {
		return store.parseListModelCatalog()
	}

	parseQueries, parseErr := parseLoadStoreQueries()
	if parseErr != nil {
		return nil, parseErr
	}
	parseDb, parseErr := sql.Open("sqlite3", "file:chat-model-catalog?mode=memory&cache=private")
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseDb.Close()

	if _, parseErr2 := parseDb.Exec(parseQueries.schema); parseErr2 != nil {
		return nil, parseErr2
	}

	parseTempStore := &Store{db: parseDb, queries: parseQueries}
	return parseTempStore.parseListModelCatalog()
}
