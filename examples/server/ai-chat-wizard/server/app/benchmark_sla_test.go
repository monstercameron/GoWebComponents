package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/server/provider"
)

const (
	defaultSendSLAMaxCores      = 8
	defaultSendSLAMaxClients    = 64
	defaultSendSLABurstRuns     = 3
	defaultSendSLATargetLatency = 100 * time.Millisecond
)

type sendSweepResult struct {
	latencies []time.Duration
	avg       time.Duration
	p95       time.Duration
	max       time.Duration
}

type coreSweepSummary struct {
	coreCount       int
	bestClientCount int
	clippedByCap    bool
}

type cumulativeSweepPoint struct {
	coreCount         int
	rawClientCount    int
	cumulativeClients int
	clippedByCap      bool
}

type linearRegressionResult struct {
	slope     float64
	intercept float64
}

func TestSendSLASweep(parseT *testing.T) {
	parseMaxCores := parseSendSLAEnvInt("CHAT_WIZARD_BENCH_MAX_CORES", defaultSendSLAMaxCores)
	if parseCpuCount := runtime.NumCPU(); parseMaxCores > parseCpuCount {
		parseMaxCores = parseCpuCount
	}
	if parseMaxCores < 1 {
		parseMaxCores = 1
	}
	parseMaxClients := max(parseSendSLAEnvInt("CHAT_WIZARD_BENCH_MAX_CLIENTS", defaultSendSLAMaxClients), 1)
	parseBurstRuns := max(parseSendSLAEnvInt("CHAT_WIZARD_BENCH_BURST_RUNS", defaultSendSLABurstRuns), 1)
	parseSlaTarget := parseSendSLAEnvDuration("CHAT_WIZARD_BENCH_SLA_MS", defaultSendSLATargetLatency)
	parsePredictionCores := parseSendSLAEnvInt("CHAT_WIZARD_BENCH_PREDICT_CORES", 32)
	if parsePredictionCores < 1 {
		parsePredictionCores = 32
	}

	parseT.Logf("send SLA sweep config: cores=1..%d max_clients=%d burst_runs=%d sla=%s predict_cores=%d", parseMaxCores, parseMaxClients, parseBurstRuns, parseSlaTarget, parsePredictionCores)

	parseSummaries := make([]coreSweepSummary, 0, parseMaxCores)

	for parseCoreCount := 1; parseCoreCount <= parseMaxCores; parseCoreCount++ {
		parseCoreCount2 := parseCoreCount
		parseT.Run(fmt.Sprintf("%d_core", parseCoreCount2), func(parseT2 *testing.T) {
			parsePreviousMaxProcs := runtime.GOMAXPROCS(parseCoreCount2)
			defer runtime.GOMAXPROCS(parsePreviousMaxProcs)

			parseBestClientCount := 0
			isParseClippedByCap := true
			for parseClientCount := 1; parseClientCount <= parseMaxClients; parseClientCount++ {
				parseResult := parseRunSendSweepBurst(parseT2, parseCoreCount2, parseClientCount, parseBurstRuns)
				parseT2.Logf(
					"cores=%d clients=%d avg=%s p95=%s max=%s requests=%d",
					parseCoreCount2,
					parseClientCount,
					parseResult.avg,
					parseResult.p95,
					parseResult.max,
					len(parseResult.latencies),
				)
				if parseResult.p95 > parseSlaTarget {
					parseT2.Logf("SLA breach at cores=%d clients=%d: p95=%s > %s", parseCoreCount2, parseClientCount, parseResult.p95, parseSlaTarget)
					isParseClippedByCap = false
					break
				}
				parseBestClientCount = parseClientCount
			}

			parseT2.Logf("core summary: cores=%d max_clients_under_sla=%d sla=%s", parseCoreCount2, parseBestClientCount, parseSlaTarget)
			parseSummaries = append(parseSummaries, coreSweepSummary{
				coreCount:       parseCoreCount2,
				bestClientCount: parseBestClientCount,
				clippedByCap:    isParseClippedByCap && parseBestClientCount == parseMaxClients,
			})
		})
	}

	if len(parseSummaries) == 0 {
		return
	}

	parseBaseline := parseSummaries[0]
	for _, parseSummary := range parseSummaries {
		parseClientsPerCore := float64(parseSummary.bestClientCount) / float64(parseSummary.coreCount)
		parseScalingFactor := 0.0
		if parseBaseline.bestClientCount > 0 {
			parseScalingFactor = float64(parseSummary.bestClientCount) / float64(parseBaseline.bestClientCount)
		}
		if parseSummary.clippedByCap {
			parseT.Logf(
				"curve point: cores=%d total_clients=%d+ clients_per_core=%.2f scaling_vs_1_core=%.3f clipped_by_cap=true",
				parseSummary.coreCount,
				parseSummary.bestClientCount,
				parseClientsPerCore,
				parseScalingFactor,
			)
			continue
		}
		parseT.Logf(
			"curve point: cores=%d total_clients=%d clients_per_core=%.2f scaling_vs_1_core=%.3f clipped_by_cap=false",
			parseSummary.coreCount,
			parseSummary.bestClientCount,
			parseClientsPerCore,
			parseScalingFactor,
		)
	}

	parseCumulativePoints := buildCumulativeSweepPoints(parseSummaries)
	for _, parsePoint := range parseCumulativePoints {
		parseCumulativeClientsPerCore := float64(parsePoint.cumulativeClients) / float64(parsePoint.coreCount)
		if parsePoint.clippedByCap {
			parseT.Logf(
				"rollup point: cores=%d raw_total_clients=%d+ cumulative_total_clients=%d+ cumulative_clients_per_core=%.2f clipped_by_cap=true",
				parsePoint.coreCount,
				parsePoint.rawClientCount,
				parsePoint.cumulativeClients,
				parseCumulativeClientsPerCore,
			)
			continue
		}
		parseT.Logf(
			"rollup point: cores=%d raw_total_clients=%d cumulative_total_clients=%d cumulative_clients_per_core=%.2f clipped_by_cap=false",
			parsePoint.coreCount,
			parsePoint.rawClientCount,
			parsePoint.cumulativeClients,
			parseCumulativeClientsPerCore,
		)
	}

	parseRegressionPoints := make([]cumulativeSweepPoint, 0, len(parseCumulativePoints))
	for _, parsePoint2 := range parseCumulativePoints {
		if parsePoint2.clippedByCap {
			continue
		}
		parseRegressionPoints = append(parseRegressionPoints, parsePoint2)
	}
	if len(parseRegressionPoints) < 2 {
		parseT.Logf("projection skipped: need at least 2 uncensored cumulative points, got %d", len(parseRegressionPoints))
		return
	}

	parseRegression := parseFitCumulativeLinearRegression(parseRegressionPoints)
	parsePredictedClients := parseRegression.intercept + parseRegression.slope*float64(parsePredictionCores)
	if parsePredictedClients < 0 {
		parsePredictedClients = 0
	}
	parseT.Logf(
		"projection: method=linear_regression_rollup cores=%d predicted_total_clients=%.2f slope=%.4f intercept=%.4f source_points=%d",
		parsePredictionCores,
		parsePredictedClients,
		parseRegression.slope,
		parseRegression.intercept,
		len(parseRegressionPoints),
	)
}

func buildCumulativeSweepPoints(parsePoints []coreSweepSummary) []cumulativeSweepPoint {
	parseCumulative := make([]cumulativeSweepPoint, 0, len(parsePoints))
	parseRunningTotal := 0
	for _, parsePoint := range parsePoints {
		parseRunningTotal += parsePoint.bestClientCount
		parseCumulative = append(parseCumulative, cumulativeSweepPoint{
			coreCount:         parsePoint.coreCount,
			rawClientCount:    parsePoint.bestClientCount,
			cumulativeClients: parseRunningTotal,
			clippedByCap:      parsePoint.clippedByCap,
		})
	}
	return parseCumulative
}

func parseFitCumulativeLinearRegression(parsePoints []cumulativeSweepPoint) linearRegressionResult {
	if len(parsePoints) == 0 {
		return linearRegressionResult{}
	}
	if len(parsePoints) == 1 {
		return linearRegressionResult{
			slope:     0,
			intercept: float64(parsePoints[0].cumulativeClients),
		}
	}

	var parseSumX float64
	var parseSumY float64
	var parseSumXY float64
	var parseSumX2 float64
	for _, parsePoint := range parsePoints {
		parseX := float64(parsePoint.coreCount)
		parseY := float64(parsePoint.cumulativeClients)
		parseSumX += parseX
		parseSumY += parseY
		parseSumXY += parseX * parseY
		parseSumX2 += parseX * parseX
	}
	parseN := float64(len(parsePoints))
	parseDenominator := parseN*parseSumX2 - parseSumX*parseSumX
	if parseDenominator == 0 {
		return linearRegressionResult{
			slope:     0,
			intercept: parseSumY / parseN,
		}
	}
	parseSlope := (parseN*parseSumXY - parseSumX*parseSumY) / parseDenominator
	parseIntercept := (parseSumY - parseSlope*parseSumX) / parseN
	return linearRegressionResult{
		slope:     parseSlope,
		intercept: parseIntercept,
	}
}

func parseRunSendSweepBurst(parseT *testing.T, parseCoreCount, parseClientCount, parseBurstRuns int) sendSweepResult {
	parseT.Helper()

	store, parseErr := parseOpenChatStore(filepath.Join(parseT.TempDir(), fmt.Sprintf("send-sla-%d-%d.db", parseCoreCount, parseClientCount)))
	if parseErr != nil {
		parseT.Fatalf("openChatStore: %v", parseErr)
	}
	parseT.Cleanup(store.parseClose)

	parseUserID, parseErr := store.parseCreateUser(fmt.Sprintf("send-sla-%d-%d@example.com", parseCoreCount, parseClientCount), "hash", "Benchmark User")
	if parseErr != nil {
		parseT.Fatalf("createUser: %v", parseErr)
	}
	parseUser := authUser{ID: parseUserID, Email: parseNormalizeAuthEmail(fmt.Sprintf("send-sla-%d-%d@example.com", parseCoreCount, parseClientCount))}
	parseMustAssignBillingPlan(parseT, store, parseUser.ID, "free")

	parseServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(benchmarkProvider{model: modelGPT54Mini}),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                parseNewBenchmarkLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 2),
	}

	parseLatencies := make([]time.Duration, 0, parseClientCount*parseBurstRuns)
	for parseRunIndex := range parseBurstRuns {
		parseRunLatencies := make([]time.Duration, parseClientCount)
		parseErrCh := make(chan error, parseClientCount)
		var parseWaitGroup sync.WaitGroup
		parseWaitGroup.Add(parseClientCount)

		for parseClientIndex := range parseClientCount {
			parseClientIndex2 := parseClientIndex
			go func() {
				defer parseWaitGroup.Done()

				parsePeerName := fmt.Sprintf("bench-sla-peer-%d-%d-%d", parseCoreCount, parseRunIndex, parseClientIndex2)
				parseCtx := parseBindAuthUser(parseServer, parsePeerName, parseUser.ID, parseUser.Email)
				defer parseServer.parseUnbindAuthenticatedPeer(parsePeerName)

				parseConversationID, parseCreateErr := store.parseCreateConversation(parseUser.ID)
				if parseCreateErr != nil {
					parseErrCh <- fmt.Errorf("createConversation: %w", parseCreateErr)
					return
				}
				parseHistory := []*chatpb.ChatMessage{
					{Role: "user", Content: "Seed request"},
					{Role: "assistant", Content: "Seed response", ModelId: modelGPT54Mini, PromptTokens: 16, CompletionTokens: 8},
				}
				for _, parseMessage := range parseHistory {
					if parseSaveErr := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, parseMessage.GetRole(), parseMessage.GetContent(), parseMessage.GetModelId(), parseMessage.GetPromptTokens(), parseMessage.GetCompletionTokens()); parseSaveErr != nil {
						parseErrCh <- fmt.Errorf("saveConversationMessage seed: %w", parseSaveErr)
						return
					}
				}

				parseStream := &fakeChatSendStream{ctx: parseCtx}
				parseReq := &chatpb.SendRequest{
					ConversationId:  parseConversationID,
					Model:           modelGPT54Mini,
					Message:         "Profile the bridge under benchmark load.",
					History:         parseHistory,
					Tone:            defaultToneID,
					ThinkingEnabled: true,
					ThinkingEffort:  defaultThinkingEffort,
				}

				parseStartedAt := time.Now()
				if parseSendErr := parseServer.Send(parseReq, parseStream); parseSendErr != nil {
					parseErrCh <- fmt.Errorf("Send: %w", parseSendErr)
					return
				}
				parseRunLatencies[parseClientIndex2] = time.Since(parseStartedAt)
			}()
		}

		parseWaitGroup.Wait()
		close(parseErrCh)
		for parseErr2 := range parseErrCh {
			if parseErr2 != nil {
				parseT.Fatal(parseErr2)
			}
		}
		parseLatencies = append(parseLatencies, parseRunLatencies...)
	}

	slices.Sort(parseLatencies)
	var parseTotal time.Duration
	var parseMaxLatency time.Duration
	for _, parseLatency := range parseLatencies {
		parseTotal += parseLatency
		if parseLatency > parseMaxLatency {
			parseMaxLatency = parseLatency
		}
	}
	parseAvgLatency := time.Duration(0)
	if len(parseLatencies) > 0 {
		parseAvgLatency = parseTotal / time.Duration(len(parseLatencies))
	}
	parseP95Latency := time.Duration(0)
	if len(parseLatencies) > 0 {
		parseP95Index := max(int(math.Ceil(float64(len(parseLatencies))*0.95))-1, 0)
		if parseP95Index >= len(parseLatencies) {
			parseP95Index = len(parseLatencies) - 1
		}
		parseP95Latency = parseLatencies[parseP95Index]
	}

	return sendSweepResult{
		latencies: parseLatencies,
		avg:       parseAvgLatency,
		p95:       parseP95Latency,
		max:       parseMaxLatency,
	}
}

func parseSendSLAEnvInt(parseName string, parseFallback int) int {
	parseRawValue := os.Getenv(parseName)
	if parseRawValue == "" {
		return parseFallback
	}
	parseParsedValue, parseErr := strconv.Atoi(parseRawValue)
	if parseErr != nil {
		return parseFallback
	}
	return parseParsedValue
}

func parseSendSLAEnvDuration(parseName string, parseFallback time.Duration) time.Duration {
	parseParsedMilliseconds := parseSendSLAEnvInt(parseName, int(parseFallback/time.Millisecond))
	if parseParsedMilliseconds <= 0 {
		return parseFallback
	}
	return time.Duration(parseParsedMilliseconds) * time.Millisecond
}
