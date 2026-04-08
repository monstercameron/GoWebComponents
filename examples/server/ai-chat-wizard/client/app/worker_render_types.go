//go:build js && wasm

package app

type renderWorkerMessageMetadataMessageRequest struct {
	GetMessageIndex int    `json:"messageIndex"`
	GetContentBytes []byte `json:"contentBytes"`
	GetThoughtBytes []byte `json:"thoughtBytes"`
	GetContentText  string `json:"-"`
	GetThoughtText  string `json:"-"`
}

type renderWorkerMessageMetadataChunkRequest struct {
	GetChunkIndex   int                                         `json:"chunkIndex"`
	GetMessageItems []renderWorkerMessageMetadataMessageRequest `json:"messageItems"`
}

type renderWorkerMessageMetadataBatchRequest struct {
	GetGeneration   uint64                                    `json:"generation"`
	GetChunkRequest []renderWorkerMessageMetadataChunkRequest `json:"chunkRequest"`
}

type renderWorkerThoughtSectionResult struct {
	GetHeadingBytes []byte `json:"headingBytes"`
	GetBodyBytes    []byte `json:"bodyBytes"`
}

type renderWorkerCanvasArtifactResult struct {
	GetIDBytes    []byte `json:"idBytes"`
	GetLabelBytes []byte `json:"labelBytes"`
}

type renderWorkerMessageMetadataMessageResult struct {
	GetMessageIndex   int                                `json:"messageIndex"`
	GetThoughtSection []renderWorkerThoughtSectionResult `json:"thoughtSection"`
	GetCanvasArtifact []renderWorkerCanvasArtifactResult `json:"canvasArtifact"`
}

type renderWorkerMessageMetadataChunkResult struct {
	GetGeneration uint64                                     `json:"generation"`
	GetChunkIndex int                                        `json:"chunkIndex"`
	GetMessage    []renderWorkerMessageMetadataMessageResult `json:"message"`
}

type renderWorkerMessageMetadataBatchResult struct {
	GetGeneration uint64                                   `json:"generation"`
	GetChunk      []renderWorkerMessageMetadataChunkResult `json:"chunk"`
}

type renderWorkerCostMessageRequest struct {
	GetMessageIndex     int    `json:"messageIndex"`
	GetModelIDBytes     []byte `json:"modelIDBytes"`
	GetPromptTokens     int    `json:"promptTokens"`
	GetCompletionTokens int    `json:"completionTokens"`
}

type renderWorkerCostModelRequest struct {
	GetModelIDBytes            []byte  `json:"modelIDBytes"`
	GetInputDollarsPerMillion  float64 `json:"inputDollarsPerMillion"`
	GetOutputDollarsPerMillion float64 `json:"outputDollarsPerMillion"`
	GetCurrencyBytes           []byte  `json:"currencyBytes"`
}

type renderWorkerThreadCostSummaryRequest struct {
	GetGeneration uint64                           `json:"generation"`
	GetMessage    []renderWorkerCostMessageRequest `json:"message"`
	GetModel      []renderWorkerCostModelRequest   `json:"model"`
}

type renderWorkerAssistantMessageCostResult struct {
	GetMessageIndex     int     `json:"messageIndex"`
	GetModelIDBytes     []byte  `json:"modelIDBytes"`
	GetPromptTokens     int     `json:"promptTokens"`
	GetCompletionTokens int     `json:"completionTokens"`
	GetCost             float64 `json:"cost"`
	GetHasExactCost     bool    `json:"hasExactCost"`
}

type renderWorkerThreadCostSummaryResult struct {
	GetGeneration             uint64                                   `json:"generation"`
	GetTotalCost              float64                                  `json:"totalCost"`
	GetAssistantMessageCost   []renderWorkerAssistantMessageCostResult `json:"assistantMessageCost"`
	GetHasAnyExactCosts       bool                                     `json:"hasAnyExactCosts"`
	GetAllAssistantCostsExact bool                                     `json:"allAssistantCostsExact"`
}

type renderWorkerSignatureMessageRequest struct {
	GetMessageIndex     int    `json:"messageIndex"`
	GetContentBytes     []byte `json:"contentBytes"`
	GetThoughtBytes     []byte `json:"thoughtBytes"`
	GetModelIDBytes     []byte `json:"modelIDBytes"`
	GetPromptTokens     int    `json:"promptTokens"`
	GetCompletionTokens int    `json:"completionTokens"`
}

type renderWorkerSignatureRequest struct {
	GetGeneration uint64                                `json:"generation"`
	GetMessage    []renderWorkerSignatureMessageRequest `json:"message"`
	GetModel      []renderWorkerCostModelRequest        `json:"model"`
}

type renderWorkerSignatureResult struct {
	GetGeneration                      uint64 `json:"generation"`
	GetCompletedMarkdownSignatureBytes []byte `json:"completedMarkdownSignatureBytes"`
	GetAssistantMetadataSignatureBytes []byte `json:"assistantMetadataSignatureBytes"`
	GetThreadCostSignatureBytes        []byte `json:"threadCostSignatureBytes"`
}

type renderWorkerSignatureState struct {
	GetCompletedMarkdownSignature string
	GetAssistantMetadataSignature string
	GetThreadCostSignature        string
}

type renderWorkerThoughtCacheEntry struct {
	GetThoughtText string
	GetSection     []thoughtSection
}

type renderWorkerCanvasCacheEntry struct {
	GetContentText string
	GetArtifact    []canvasArtifact
}

// parseBuildRenderSignatureState derives one local signature state using synchronous app helpers.
func parseBuildRenderSignatureState(parseMessages []message, parseModels []modelOption) renderWorkerSignatureState {
	return renderWorkerSignatureState{
		GetCompletedMarkdownSignature: parseCompletedAssistantMessagesMarkdownSignature(parseMessages),
		GetAssistantMetadataSignature: parseBuildAssistantMessageMetadataSignature(parseMessages),
		GetThreadCostSignature:        parseThreadCostSummarySignature(parseMessages, parseModels),
	}
}

// parseBuildAssistantMessageMetadataSignature builds one stable signature for off-thread thought/canvas metadata derivation.
func parseBuildAssistantMessageMetadataSignature(parseMessages []message) string {
	parseHash := parseSignatureSeed
	hasParseAssistantMessage := false
	for parseMessageIndex, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		hasParseAssistantMessage = true
		parseApplySignatureByte(&parseHash, 'm')
		parseApplySignatureInt(&parseHash, parseMessageIndex)
		parseApplySignatureString(&parseHash, parseMessageItem.Content)
		parseApplySignatureString(&parseHash, parseMessageItem.Thought)
	}
	if !hasParseAssistantMessage {
		return ""
	}
	return parseBuildSignatureString(parseHash)
}

// parseBuildMessageIndexString converts one index into a compact base-10 signature fragment.
func parseBuildMessageIndexString(parseIndex int) string {
	if parseIndex == 0 {
		return "0"
	}
	isParseNegative := parseIndex < 0
	if isParseNegative {
		parseIndex = -parseIndex
	}
	parseDigits := [20]byte{}
	parseWrite := len(parseDigits)
	for parseIndex > 0 {
		parseWrite--
		parseDigits[parseWrite] = byte('0' + (parseIndex % 10))
		parseIndex /= 10
	}
	if isParseNegative {
		parseWrite--
		parseDigits[parseWrite] = '-'
	}
	return string(parseDigits[parseWrite:])
}
