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

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
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
	coreCount        int
	rawClientCount   int
	cumulativeClients int
	clippedByCap     bool
}

type linearRegressionResult struct {
	slope     float64
	intercept float64
}

func TestSendSLASweep(t *testing.T) {
	maxCores := min(sendSLAEnvInt("CHAT_WIZARD_BENCH_MAX_CORES", defaultSendSLAMaxCores), runtime.NumCPU())
	if maxCores < 1 {
		maxCores = 1
	}
	maxClients := sendSLAEnvInt("CHAT_WIZARD_BENCH_MAX_CLIENTS", defaultSendSLAMaxClients)
	if maxClients < 1 {
		maxClients = 1
	}
	burstRuns := sendSLAEnvInt("CHAT_WIZARD_BENCH_BURST_RUNS", defaultSendSLABurstRuns)
	if burstRuns < 1 {
		burstRuns = 1
	}
	slaTarget := sendSLAEnvDuration("CHAT_WIZARD_BENCH_SLA_MS", defaultSendSLATargetLatency)
	predictionCores := sendSLAEnvInt("CHAT_WIZARD_BENCH_PREDICT_CORES", 32)
	if predictionCores < 1 {
		predictionCores = 32
	}

	t.Logf("send SLA sweep config: cores=1..%d max_clients=%d burst_runs=%d sla=%s predict_cores=%d", maxCores, maxClients, burstRuns, slaTarget, predictionCores)

	summaries := make([]coreSweepSummary, 0, maxCores)

	for coreCount := 1; coreCount <= maxCores; coreCount++ {
		coreCount := coreCount
		t.Run(fmt.Sprintf("%d_core", coreCount), func(t *testing.T) {
			previousMaxProcs := runtime.GOMAXPROCS(coreCount)
			defer runtime.GOMAXPROCS(previousMaxProcs)

			bestClientCount := 0
			clippedByCap := true
			for clientCount := 1; clientCount <= maxClients; clientCount++ {
				result := runSendSweepBurst(t, coreCount, clientCount, burstRuns)
				t.Logf(
					"cores=%d clients=%d avg=%s p95=%s max=%s requests=%d",
					coreCount,
					clientCount,
					result.avg,
					result.p95,
					result.max,
					len(result.latencies),
				)
				if result.p95 > slaTarget {
					t.Logf("SLA breach at cores=%d clients=%d: p95=%s > %s", coreCount, clientCount, result.p95, slaTarget)
					clippedByCap = false
					break
				}
				bestClientCount = clientCount
			}

			t.Logf("core summary: cores=%d max_clients_under_sla=%d sla=%s", coreCount, bestClientCount, slaTarget)
			summaries = append(summaries, coreSweepSummary{
				coreCount:       coreCount,
				bestClientCount: bestClientCount,
				clippedByCap:    clippedByCap && bestClientCount == maxClients,
			})
		})
	}

	if len(summaries) == 0 {
		return
	}

	baseline := summaries[0]
	for _, summary := range summaries {
		clientsPerCore := float64(summary.bestClientCount) / float64(summary.coreCount)
		scalingFactor := 0.0
		if baseline.bestClientCount > 0 {
			scalingFactor = float64(summary.bestClientCount) / float64(baseline.bestClientCount)
		}
		if summary.clippedByCap {
			t.Logf(
				"curve point: cores=%d total_clients=%d+ clients_per_core=%.2f scaling_vs_1_core=%.3f clipped_by_cap=true",
				summary.coreCount,
				summary.bestClientCount,
				clientsPerCore,
				scalingFactor,
			)
			continue
		}
		t.Logf(
			"curve point: cores=%d total_clients=%d clients_per_core=%.2f scaling_vs_1_core=%.3f clipped_by_cap=false",
			summary.coreCount,
			summary.bestClientCount,
			clientsPerCore,
			scalingFactor,
		)
	}

	cumulativePoints := buildCumulativeSweepPoints(summaries)
	for _, point := range cumulativePoints {
		cumulativeClientsPerCore := float64(point.cumulativeClients) / float64(point.coreCount)
		if point.clippedByCap {
			t.Logf(
				"rollup point: cores=%d raw_total_clients=%d+ cumulative_total_clients=%d+ cumulative_clients_per_core=%.2f clipped_by_cap=true",
				point.coreCount,
				point.rawClientCount,
				point.cumulativeClients,
				cumulativeClientsPerCore,
			)
			continue
		}
		t.Logf(
			"rollup point: cores=%d raw_total_clients=%d cumulative_total_clients=%d cumulative_clients_per_core=%.2f clipped_by_cap=false",
			point.coreCount,
			point.rawClientCount,
			point.cumulativeClients,
			cumulativeClientsPerCore,
		)
	}

	regressionPoints := make([]cumulativeSweepPoint, 0, len(cumulativePoints))
	for _, point := range cumulativePoints {
		if point.clippedByCap {
			continue
		}
		regressionPoints = append(regressionPoints, point)
	}
	if len(regressionPoints) < 2 {
		t.Logf("projection skipped: need at least 2 uncensored cumulative points, got %d", len(regressionPoints))
		return
	}

	regression := fitCumulativeLinearRegression(regressionPoints)
	predictedClients := regression.intercept + regression.slope*float64(predictionCores)
	if predictedClients < 0 {
		predictedClients = 0
	}
	t.Logf(
		"projection: method=linear_regression_rollup cores=%d predicted_total_clients=%.2f slope=%.4f intercept=%.4f source_points=%d",
		predictionCores,
		predictedClients,
		regression.slope,
		regression.intercept,
		len(regressionPoints),
	)
}

func buildCumulativeSweepPoints(points []coreSweepSummary) []cumulativeSweepPoint {
	cumulative := make([]cumulativeSweepPoint, 0, len(points))
	runningTotal := 0
	for _, point := range points {
		runningTotal += point.bestClientCount
		cumulative = append(cumulative, cumulativeSweepPoint{
			coreCount:         point.coreCount,
			rawClientCount:    point.bestClientCount,
			cumulativeClients: runningTotal,
			clippedByCap:      point.clippedByCap,
		})
	}
	return cumulative
}

func fitCumulativeLinearRegression(points []cumulativeSweepPoint) linearRegressionResult {
	if len(points) == 0 {
		return linearRegressionResult{}
	}
	if len(points) == 1 {
		return linearRegressionResult{
			slope:     0,
			intercept: float64(points[0].cumulativeClients),
		}
	}

	var sumX float64
	var sumY float64
	var sumXY float64
	var sumX2 float64
	for _, point := range points {
		x := float64(point.coreCount)
		y := float64(point.cumulativeClients)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	n := float64(len(points))
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return linearRegressionResult{
			slope:     0,
			intercept: sumY / n,
		}
	}
	slope := (n*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / n
	return linearRegressionResult{
		slope:     slope,
		intercept: intercept,
	}
}

func fitLinearRegression(points []coreSweepSummary) linearRegressionResult {
	if len(points) == 0 {
		return linearRegressionResult{}
	}
	if len(points) == 1 {
		return linearRegressionResult{
			slope:     0,
			intercept: float64(points[0].bestClientCount),
		}
	}

	var sumX float64
	var sumY float64
	var sumXY float64
	var sumX2 float64
	for _, point := range points {
		x := float64(point.coreCount)
		y := float64(point.bestClientCount)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	n := float64(len(points))
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return linearRegressionResult{
			slope:     0,
			intercept: sumY / n,
		}
	}
	slope := (n*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / n
	return linearRegressionResult{
		slope:     slope,
		intercept: intercept,
	}
}

func runSendSweepBurst(t *testing.T, coreCount, clientCount, burstRuns int) sendSweepResult {
	t.Helper()

	store, err := openChatStore(filepath.Join(t.TempDir(), fmt.Sprintf("send-sla-%d-%d.db", coreCount, clientCount)))
	if err != nil {
		t.Fatalf("openChatStore: %v", err)
	}
	t.Cleanup(store.close)

	userID, err := store.createUser(fmt.Sprintf("send-sla-%d-%d@example.com", coreCount, clientCount), "hash", "Benchmark User")
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}
	user := authUser{ID: userID, Email: normalizeAuthEmail(fmt.Sprintf("send-sla-%d-%d@example.com", coreCount, clientCount))}

	server := &chatServer{
		providerRegistry:      provider.NewRegistry(benchmarkProvider{model: modelGPT54Mini}),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                newBenchmarkLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 2),
	}

	latencies := make([]time.Duration, 0, clientCount*burstRuns)
	for runIndex := 0; runIndex < burstRuns; runIndex++ {
		runLatencies := make([]time.Duration, clientCount)
		errCh := make(chan error, clientCount)
		var waitGroup sync.WaitGroup
		waitGroup.Add(clientCount)

		for clientIndex := 0; clientIndex < clientCount; clientIndex++ {
			clientIndex := clientIndex
			go func() {
				defer waitGroup.Done()

				peerName := fmt.Sprintf("bench-sla-peer-%d-%d-%d", coreCount, runIndex, clientIndex)
				ctx := bindAuthUser(server, peerName, user.ID, user.Email)
				defer server.unbindAuthenticatedPeer(peerName)

				conversationID, createErr := store.createConversation(user.ID)
				if createErr != nil {
					errCh <- fmt.Errorf("createConversation: %w", createErr)
					return
				}
				history := []*chatpb.ChatMessage{
					{Role: "user", Content: "Seed request"},
					{Role: "assistant", Content: "Seed response", ModelId: modelGPT54Mini, PromptTokens: 16, CompletionTokens: 8},
				}
				for _, message := range history {
					if saveErr := store.saveConversationMessage(user.ID, conversationID, message.GetRole(), message.GetContent(), message.GetModelId(), message.GetPromptTokens(), message.GetCompletionTokens()); saveErr != nil {
						errCh <- fmt.Errorf("saveConversationMessage seed: %w", saveErr)
						return
					}
				}

				stream := &fakeChatSendStream{ctx: ctx}
				req := &chatpb.SendRequest{
					ConversationId:  conversationID,
					Model:           modelGPT54Mini,
					Message:         "Profile the bridge under benchmark load.",
					History:         history,
					Tone:            defaultToneID,
					ThinkingEnabled: true,
					ThinkingEffort:  defaultThinkingEffort,
				}

				startedAt := time.Now()
				if sendErr := server.Send(req, stream); sendErr != nil {
					errCh <- fmt.Errorf("Send: %w", sendErr)
					return
				}
				runLatencies[clientIndex] = time.Since(startedAt)
			}()
		}

		waitGroup.Wait()
		close(errCh)
		for err := range errCh {
			if err != nil {
				t.Fatal(err)
			}
		}
		latencies = append(latencies, runLatencies...)
	}

	slices.Sort(latencies)
	var total time.Duration
	var maxLatency time.Duration
	for _, latency := range latencies {
		total += latency
		if latency > maxLatency {
			maxLatency = latency
		}
	}
	avgLatency := time.Duration(0)
	if len(latencies) > 0 {
		avgLatency = total / time.Duration(len(latencies))
	}
	p95Latency := time.Duration(0)
	if len(latencies) > 0 {
		p95Index := int(math.Ceil(float64(len(latencies))*0.95)) - 1
		if p95Index < 0 {
			p95Index = 0
		}
		if p95Index >= len(latencies) {
			p95Index = len(latencies) - 1
		}
		p95Latency = latencies[p95Index]
	}

	return sendSweepResult{
		latencies: latencies,
		avg:       avgLatency,
		p95:       p95Latency,
		max:       maxLatency,
	}
}

func sendSLAEnvInt(name string, fallback int) int {
	rawValue := os.Getenv(name)
	if rawValue == "" {
		return fallback
	}
	parsedValue, err := strconv.Atoi(rawValue)
	if err != nil {
		return fallback
	}
	return parsedValue
}

func sendSLAEnvDuration(name string, fallback time.Duration) time.Duration {
	parsedMilliseconds := sendSLAEnvInt(name, int(fallback/time.Millisecond))
	if parsedMilliseconds <= 0 {
		return fallback
	}
	return time.Duration(parsedMilliseconds) * time.Millisecond
}
