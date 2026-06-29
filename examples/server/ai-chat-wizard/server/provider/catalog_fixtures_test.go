package provider

func parseTestOpenAICatalog() Catalog {
	parseModels := []ModelMetadata{
		{
			ID:            "gpt-5.4",
			DisplayName:   "GPT-5.4",
			ProviderID:    "openai",
			ProviderLabel: "OpenAI",
			Capabilities:  ModelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
			Pricing:       ModelPricing{InputPerMillionUSD: 1.25, OutputPerMillionUSD: 10.00, Currency: "USD"},
		},
		{
			ID:            "gpt-5.4-mini",
			DisplayName:   "GPT-5.4 mini",
			ProviderID:    "openai",
			ProviderLabel: "OpenAI",
			Capabilities:  ModelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
			Pricing:       ModelPricing{InputPerMillionUSD: 0.25, OutputPerMillionUSD: 2.00, Currency: "USD"},
		},
		{
			ID:            "gpt-5.4-nano",
			DisplayName:   "GPT-5.4 nano",
			ProviderID:    "openai",
			ProviderLabel: "OpenAI",
			Capabilities:  ModelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
			Pricing:       ModelPricing{InputPerMillionUSD: 0.05, OutputPerMillionUSD: 0.40, Currency: "USD"},
		},
	}
	return Catalog{
		Models: []ModelMetadata{parseModels[0], parseModels[1], parseModels[2]},
		Options: []ModelOption{
			ParseModelOptionFromMetadata(parseModels[0], "Best"),
			ParseModelOptionFromMetadata(parseModels[1], "Fast"),
			ParseModelOptionFromMetadata(parseModels[2], "Cheap"),
		},
		DefaultModel: "gpt-5.4-mini",
		TitleModel:   "gpt-5.4-nano",
	}
}

func parseTestAnthropicCatalog() Catalog {
	parseModels := []ModelMetadata{
		{
			ID:              "claude-sonnet-4-5",
			DisplayName:     "Claude Sonnet 4.5",
			ProviderID:      "anthropic",
			ProviderLabel:   "Anthropic",
			Capabilities:    ModelCapabilities{ProviderID: "anthropic", ProviderLabel: "Anthropic", SupportsThinking: true, SupportsSpeech: false},
			MaxOutputTokens: anthropicMaxTokens,
		},
		{
			ID:              "claude-haiku-4-5",
			DisplayName:     "Claude Haiku 4.5",
			ProviderID:      "anthropic",
			ProviderLabel:   "Anthropic",
			Capabilities:    ModelCapabilities{ProviderID: "anthropic", ProviderLabel: "Anthropic", SupportsThinking: true, SupportsSpeech: false},
			MaxOutputTokens: anthropicMaxTokens,
		},
	}
	return Catalog{
		Models: []ModelMetadata{parseModels[0], parseModels[1]},
		Options: []ModelOption{
			ParseModelOptionFromMetadata(parseModels[0], "Reasoning"),
			ParseModelOptionFromMetadata(parseModels[1], "Fast"),
		},
		DefaultModel: "claude-sonnet-4-5",
		TitleModel:   "claude-haiku-4-5",
	}
}

func parseTestCerebrasCatalog() Catalog {
	parseModels := []ModelMetadata{
		{
			ID:              "gpt-oss-120b",
			DisplayName:     "GPT OSS 120B",
			ProviderID:      "cerebras",
			ProviderLabel:   "Cerebras",
			Capabilities:    ModelCapabilities{ProviderID: "cerebras", ProviderLabel: "Cerebras", SupportsThinking: true, SupportsSpeech: false},
			MaxOutputTokens: cerebrasMaxCompletionTokens,
		},
		{
			ID:              "llama3.1-8b",
			DisplayName:     "Llama 3.1 8B",
			ProviderID:      "cerebras",
			ProviderLabel:   "Cerebras",
			Capabilities:    ModelCapabilities{ProviderID: "cerebras", ProviderLabel: "Cerebras", SupportsThinking: false, SupportsSpeech: false},
			MaxOutputTokens: cerebrasMaxCompletionTokens,
		},
		{
			ID:              "qwen-3-235b-a22b-instruct-2507",
			DisplayName:     "Qwen 3 235B Instruct",
			ProviderID:      "cerebras",
			ProviderLabel:   "Cerebras",
			Capabilities:    ModelCapabilities{ProviderID: "cerebras", ProviderLabel: "Cerebras", SupportsThinking: false, SupportsSpeech: false},
			MaxOutputTokens: cerebrasMaxCompletionTokens,
		},
		{
			ID:              "zai-glm-4.7",
			DisplayName:     "Z.ai GLM 4.7",
			ProviderID:      "cerebras",
			ProviderLabel:   "Cerebras",
			Capabilities:    ModelCapabilities{ProviderID: "cerebras", ProviderLabel: "Cerebras", SupportsThinking: true, SupportsSpeech: false},
			MaxOutputTokens: cerebrasMaxCompletionTokens,
		},
	}
	return Catalog{
		Models: []ModelMetadata{parseModels[0], parseModels[1], parseModels[2], parseModels[3]},
		Options: []ModelOption{
			ParseModelOptionFromMetadata(parseModels[0], "Reasoning"),
			ParseModelOptionFromMetadata(parseModels[1], "Fast"),
			ParseModelOptionFromMetadata(parseModels[2], "Preview"),
			ParseModelOptionFromMetadata(parseModels[3], "Preview"),
		},
		DefaultModel: "gpt-oss-120b",
		TitleModel:   "llama3.1-8b",
	}
}
