(function () {
    const storeBenchmarkScenarioConfigs = [
        {
            id: "core-render",
            label: "Core Render",
            category: "Initial Render",
            requestedWork: "Render 40 visible core-list rows from empty state.",
            correctnessCheck: "40 .benchmark-core-item nodes must exist and metric-core-count must read 40.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-clear");
                await handleSubjectWait(() => getSubjectNumber("#metric-core-count") === 0 && document.querySelectorAll(".benchmark-core-item").length === 0, 5000, "clear core items");
            },
            async run() {
                handleSubjectClick("#btn-core-render");
                await handleSubjectWait(() => getSubjectNumber("#metric-core-count") === 40 && document.querySelectorAll(".benchmark-core-item").length === 40, 5000, "render core items");
            }
        },
        {
            id: "core-update",
            label: "Core Update",
            category: "Targeted Update",
            requestedWork: "Update the visible text content of the existing 40 core-list rows.",
            correctnessCheck: "All .benchmark-core-item nodes must include '(Updated)' and item count must stay at 40.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-render");
                await handleSubjectWait(() => getSubjectNumber("#metric-core-count") === 40 && document.querySelectorAll(".benchmark-core-item").length === 40, 5000, "prepare core items");
            },
            async run() {
                handleSubjectClick("#btn-core-update");
                await handleSubjectWait(() => {
                    const getItems = Array.from(document.querySelectorAll(".benchmark-core-item"));
                    return getItems.length === 40 && getItems.every((parseNode) => parseNode.textContent.includes("(Updated)"));
                }, 5000, "update core items");
            }
        },
        {
            id: "core-refresh",
            label: "Core Refresh",
            category: "Refresh",
            requestedWork: "Refresh the current core-list view without changing row count.",
            correctnessCheck: "metric-refresh-count must increment by one after the refresh action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-render");
                await handleSubjectWait(() => getSubjectNumber("#metric-core-count") === 40 && document.querySelectorAll(".benchmark-core-item").length === 40, 5000, "prepare core refresh");
                return {
                    getRefreshCount: getSubjectNumber("#metric-refresh-count")
                };
            },
            async run(parseContext) {
                handleSubjectClick("#btn-refresh");
                await handleSubjectWait(() => getSubjectNumber("#metric-refresh-count") === parseContext.getRefreshCount + 1, 5000, "refresh current view");
            }
        },
        {
            id: "content-render",
            label: "Content Render",
            category: "Initial Render",
            requestedWork: "Render 12 nested content cards from empty state.",
            correctnessCheck: "12 .benchmark-content-card nodes must exist and metric-content-count must read 12.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-content-clear");
                await handleSubjectWait(() => getSubjectNumber("#metric-content-count") === 0 && document.querySelectorAll(".benchmark-content-card").length === 0, 5000, "clear content cards");
            },
            async run() {
                handleSubjectClick("#btn-content-render");
                await handleSubjectWait(() => getSubjectNumber("#metric-content-count") === 12 && document.querySelectorAll(".benchmark-content-card").length === 12, 5000, "render content cards");
            }
        },
        {
            id: "content-update",
            label: "Content Update",
            category: "Targeted Update",
            requestedWork: "Update the text and status fields of the 12 rendered content cards.",
            correctnessCheck: "All .benchmark-content-status values must become 'live' and titles must include '(Updated)'.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-content-render");
                await handleSubjectWait(() => getSubjectNumber("#metric-content-count") === 12 && document.querySelectorAll(".benchmark-content-card").length === 12, 5000, "prepare content cards");
            },
            async run() {
                handleSubjectClick("#btn-content-update");
                await handleSubjectWait(() => {
                    const getStatuses = Array.from(document.querySelectorAll(".benchmark-content-status"));
                    const getTitles = Array.from(document.querySelectorAll(".benchmark-content-title"));
                    return getStatuses.length === 12 &&
                        getTitles.length === 12 &&
                        getStatuses.every((parseNode) => parseNode.textContent.trim() === "live") &&
                        getTitles.every((parseNode) => parseNode.textContent.includes("(Updated)"));
                }, 5000, "update content cards");
            }
        },
        {
            id: "deep-render",
            label: "Deep Tree Render",
            category: "Initial Render",
            requestedWork: "Render the recursive deep-tree benchmark view until the benchmark leaf becomes visible.",
            correctnessCheck: "#benchmark-deep-leaf must exist after the render action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-clear");
                await handleSubjectWait(() => getSubjectNumber("#metric-core-count") === 0, 5000, "prepare deep tree");
            },
            async run() {
                handleSubjectClick("#btn-deep-render");
                await handleSubjectWait(() => !!document.querySelector("#benchmark-deep-leaf"), 5000, "render deep tree");
            }
        },
        {
            id: "hooks-render",
            label: "Hook Grid Render",
            category: "Initial Render",
            requestedWork: "Render the 40-cell hook-heavy grid view.",
            correctnessCheck: "Exactly 40 .benchmark-hook-node elements must exist after the render action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-clear");
                await handleSubjectWait(() => getSubjectNumber("#metric-core-count") === 0, 5000, "prepare hook grid");
            },
            async run() {
                handleSubjectClick("#btn-hooks-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-hook-node").length === 40, 5000, "render hook grid");
            }
        }
    ];

    function getSubjectNumber(parseSelector) {
        const getNode = document.querySelector(parseSelector);
        if (!getNode) {
            return 0;
        }
        const getMatch = getNode.textContent.match(/-?\d+/);
        if (!getMatch) {
            return 0;
        }
        return Number.parseInt(getMatch[0], 10);
    }

    function handleSubjectClick(parseSelector) {
        const getNode = document.querySelector(parseSelector);
        if (!getNode) {
            throw new Error("missing benchmark control " + parseSelector);
        }
        getNode.click();
    }

    function handleSubjectDelay(parseDurationMs) {
        return new Promise((parseResolve) => {
            window.setTimeout(parseResolve, parseDurationMs);
        });
    }

    function handleSubjectAnimationFrames(parseCount) {
        return new Promise((parseResolve) => {
            function handleSubjectNext(parseRemaining) {
                if (parseRemaining <= 0) {
                    parseResolve();
                    return;
                }
                window.requestAnimationFrame(() => {
                    handleSubjectNext(parseRemaining - 1);
                });
            }
            handleSubjectNext(parseCount);
        });
    }

    async function handleSubjectWait(parsePredicate, parseTimeoutMs, parseLabel) {
        const getStartedAt = performance.now();
        while ((performance.now() - getStartedAt) < parseTimeoutMs) {
            let isReady = false;
            try {
                isReady = !!parsePredicate();
            } catch (parseErr) {
                window.__example201SubjectError = String(parseErr);
                throw parseErr;
            }
            if (isReady) {
                return;
            }
            await handleSubjectAnimationFrames(1);
        }
        throw new Error("timed out waiting for " + parseLabel);
    }

    function buildSubjectScenarioOrder(parseScenarioIDs, parseSeed) {
        const getScenarioIDs = Array.isArray(parseScenarioIDs) && parseScenarioIDs.length > 0
            ? parseScenarioIDs.slice()
            : storeBenchmarkScenarioConfigs.map((parseConfig) => parseConfig.id);
        let getState = (Number.parseInt(String(parseSeed || "1"), 10) || 1) >>> 0;
        function buildSubjectNextRandom() {
            getState = (getState * 1664525 + 1013904223) >>> 0;
            return getState / 0x100000000;
        }
        for (let parseIndex = getScenarioIDs.length - 1; parseIndex > 0; parseIndex--) {
            const getSwapIndex = Math.floor(buildSubjectNextRandom() * (parseIndex + 1));
            const getTemp = getScenarioIDs[parseIndex];
            getScenarioIDs[parseIndex] = getScenarioIDs[getSwapIndex];
            getScenarioIDs[getSwapIndex] = getTemp;
        }
        return getScenarioIDs;
    }

    function buildSubjectHeapBytes() {
        if (typeof performance.memory?.usedJSHeapSize !== "number") {
            return 0;
        }
        return performance.memory.usedJSHeapSize;
    }

    function buildSubjectLongTaskProbe() {
        if (typeof PerformanceObserver !== "function") {
            return {
                getSupported: false,
                handleStop() {
                    return {
                        getLongTaskCount: 0,
                        getLongTaskDurationMs: 0
                    };
                }
            };
        }
        if (!Array.isArray(PerformanceObserver.supportedEntryTypes) || !PerformanceObserver.supportedEntryTypes.includes("longtask")) {
            return {
                getSupported: false,
                handleStop() {
                    return {
                        getLongTaskCount: 0,
                        getLongTaskDurationMs: 0
                    };
                }
            };
        }
        const getEntries = [];
        const getObserver = new PerformanceObserver((parseList) => {
            getEntries.push(...parseList.getEntries());
        });
        try {
            getObserver.observe({ entryTypes: ["longtask"] });
        } catch (parseErr) {
            return {
                getSupported: false,
                handleStop() {
                    return {
                        getLongTaskCount: 0,
                        getLongTaskDurationMs: 0
                    };
                }
            };
        }
        return {
            getSupported: true,
            handleStop() {
                getObserver.disconnect();
                const getLongTaskDurationMs = getEntries.reduce((parseTotal, parseEntry) => parseTotal + parseEntry.duration, 0);
                return {
                    getLongTaskCount: getEntries.length,
                    getLongTaskDurationMs: Number(getLongTaskDurationMs.toFixed(3))
                };
            }
        };
    }

    function buildSubjectMutationProbe() {
        const getContainerNode = document.querySelector("#benchmark-container") || document.querySelector("#benchmark-app") || document.body;
        const getMutationStats = {
            getMutationRecordCount: 0,
            getChildListMutationCount: 0,
            getAttributeMutationCount: 0,
            getCharacterDataMutationCount: 0,
            getAddedNodeCount: 0,
            getRemovedNodeCount: 0
        };
        const getObserver = new MutationObserver((parseRecords) => {
            getMutationStats.getMutationRecordCount += parseRecords.length;
            for (const parseRecord of parseRecords) {
                if (parseRecord.type === "childList") {
                    getMutationStats.getChildListMutationCount += 1;
                    getMutationStats.getAddedNodeCount += parseRecord.addedNodes.length;
                    getMutationStats.getRemovedNodeCount += parseRecord.removedNodes.length;
                    continue;
                }
                if (parseRecord.type === "attributes") {
                    getMutationStats.getAttributeMutationCount += 1;
                    continue;
                }
                if (parseRecord.type === "characterData") {
                    getMutationStats.getCharacterDataMutationCount += 1;
                }
            }
        });
        getObserver.observe(getContainerNode, {
            childList: true,
            subtree: true,
            attributes: true,
            characterData: true
        });
        return {
            handleStop() {
                getObserver.disconnect();
                return getMutationStats;
            }
        };
    }

    function buildSubjectMetricSummary(parseValues) {
        const getSortedValues = parseValues.slice().sort((parseLeft, parseRight) => parseLeft - parseRight);
        const getMedianIndex = Math.floor(getSortedValues.length / 2);
        const getP95Index = Math.min(getSortedValues.length - 1, Math.floor(getSortedValues.length * 0.95));
        const getSum = getSortedValues.reduce((parseTotal, parseValue) => parseTotal + parseValue, 0);
        return {
            getMean: Number((getSum / getSortedValues.length).toFixed(3)),
            getMedian: Number(getSortedValues[getMedianIndex].toFixed(3)),
            getMin: Number(getSortedValues[0].toFixed(3)),
            getMax: Number(getSortedValues[getSortedValues.length - 1].toFixed(3)),
            getP95: Number(getSortedValues[getP95Index].toFixed(3)),
            getSamples: getSortedValues.map((parseValue) => Number(parseValue.toFixed(3)))
        };
    }

    function buildSubjectSummary(parseSamples, parseScenarioConfig, parseFramework) {
        const getDomReadyStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getDomReadyMs));
        const getPaintVisibleStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getPaintVisibleMs));
        const getPaintAfterDomStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getPaintAfterDomMs));
        const getMutationRecordStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getMutationRecordCount));
        const getChildListStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getChildListMutationCount));
        const getAttributeStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getAttributeMutationCount));
        const getCharacterDataStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getCharacterDataMutationCount));
        const getAddedNodeStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getAddedNodeCount));
        const getRemovedNodeStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getRemovedNodeCount));
        const getLongTaskCountStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getLongTaskCount));
        const getLongTaskDurationStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getLongTaskDurationMs));
        const getHeapDeltaValues = parseSamples
            .map((parseSample) => parseSample.getHeapDeltaBytes)
            .filter((parseValue) => typeof parseValue === "number");
        const getHeapDeltaStats = getHeapDeltaValues.length > 0 ? buildSubjectMetricSummary(getHeapDeltaValues) : null;
        return {
            getFramework: parseFramework,
            getScenarioID: parseScenarioConfig.id,
            getScenarioLabel: parseScenarioConfig.label,
            getCategory: parseScenarioConfig.category,
            getRequestedWork: parseScenarioConfig.requestedWork,
            getCorrectnessCheck: parseScenarioConfig.correctnessCheck,
            getFinishLine: parseScenarioConfig.finishLine,
            getIterationCount: parseSamples.length,
            getDomReadyMeanMs: getDomReadyStats.getMean,
            getDomReadyMedianMs: getDomReadyStats.getMedian,
            getDomReadyMinMs: getDomReadyStats.getMin,
            getDomReadyMaxMs: getDomReadyStats.getMax,
            getDomReadyP95Ms: getDomReadyStats.getP95,
            getDomReadySamplesMs: getDomReadyStats.getSamples,
            getPaintVisibleMeanMs: getPaintVisibleStats.getMean,
            getPaintVisibleMedianMs: getPaintVisibleStats.getMedian,
            getPaintVisibleMinMs: getPaintVisibleStats.getMin,
            getPaintVisibleMaxMs: getPaintVisibleStats.getMax,
            getPaintVisibleP95Ms: getPaintVisibleStats.getP95,
            getPaintVisibleSamplesMs: getPaintVisibleStats.getSamples,
            getPaintAfterDomMeanMs: getPaintAfterDomStats.getMean,
            getPaintAfterDomMedianMs: getPaintAfterDomStats.getMedian,
            getPaintAfterDomP95Ms: getPaintAfterDomStats.getP95,
            getMutationRecordMean: getMutationRecordStats.getMean,
            getChildListMutationMean: getChildListStats.getMean,
            getAttributeMutationMean: getAttributeStats.getMean,
            getCharacterDataMutationMean: getCharacterDataStats.getMean,
            getAddedNodeMean: getAddedNodeStats.getMean,
            getRemovedNodeMean: getRemovedNodeStats.getMean,
            getLongTaskCountMean: getLongTaskCountStats.getMean,
            getLongTaskDurationMeanMs: getLongTaskDurationStats.getMean,
            hasHeapDelta: !!getHeapDeltaStats,
            getHeapDeltaMeanBytes: getHeapDeltaStats ? getHeapDeltaStats.getMean : 0,
            getHeapDeltaMedianBytes: getHeapDeltaStats ? getHeapDeltaStats.getMedian : 0,
            getHeapDeltaP95Bytes: getHeapDeltaStats ? getHeapDeltaStats.getP95 : 0
        };
    }

    async function handleSubjectMeasureScenario(parseScenarioID, parseOptions) {
        const getScenarioConfig = storeBenchmarkScenarioConfigs.find((parseConfig) => parseConfig.id === parseScenarioID);
        if (!getScenarioConfig) {
            throw new Error("unknown benchmark scenario " + parseScenarioID);
        }
        const getIterations = Math.max(1, Number.parseInt(parseOptions?.iterations ?? "7", 10) || 7);
        const getWarmups = Math.max(0, Number.parseInt(parseOptions?.warmups ?? "2", 10) || 2);
        for (let parseIndex = 0; parseIndex < getWarmups; parseIndex++) {
            const getWarmupContext = getScenarioConfig.prepare ? await getScenarioConfig.prepare() : undefined;
            await getScenarioConfig.run(getWarmupContext);
            await handleSubjectAnimationFrames(1);
        }
        const getSamples = [];
        for (let parseIndex = 0; parseIndex < getIterations; parseIndex++) {
            const getScenarioContext = getScenarioConfig.prepare ? await getScenarioConfig.prepare() : undefined;
            const getLongTaskProbe = buildSubjectLongTaskProbe();
            const getMutationProbe = buildSubjectMutationProbe();
            const getHeapBeforeBytes = buildSubjectHeapBytes();
            const getStartedAt = performance.now();
            await getScenarioConfig.run(getScenarioContext);
            const getDomReadyAt = performance.now();
            await handleSubjectAnimationFrames(1);
            const getPaintVisibleAt = performance.now();
            const getMutationStats = getMutationProbe.handleStop();
            const getLongTaskStats = getLongTaskProbe.handleStop();
            const getHeapAfterBytes = buildSubjectHeapBytes();
            getSamples.push({
                getDomReadyMs: getDomReadyAt - getStartedAt,
                getPaintVisibleMs: getPaintVisibleAt - getStartedAt,
                getPaintAfterDomMs: getPaintVisibleAt - getDomReadyAt,
                getMutationRecordCount: getMutationStats.getMutationRecordCount,
                getChildListMutationCount: getMutationStats.getChildListMutationCount,
                getAttributeMutationCount: getMutationStats.getAttributeMutationCount,
                getCharacterDataMutationCount: getMutationStats.getCharacterDataMutationCount,
                getAddedNodeCount: getMutationStats.getAddedNodeCount,
                getRemovedNodeCount: getMutationStats.getRemovedNodeCount,
                getLongTaskCount: getLongTaskStats.getLongTaskCount,
                getLongTaskDurationMs: getLongTaskStats.getLongTaskDurationMs,
                getHeapDeltaBytes: getHeapBeforeBytes > 0 && getHeapAfterBytes > 0 ? getHeapAfterBytes - getHeapBeforeBytes : null
            });
        }
        return buildSubjectSummary(getSamples, getScenarioConfig, document.body.dataset.framework || "unknown");
    }

    async function handleSubjectMeasureAllScenarios(parseOptions) {
        const getScenarioOrder = buildSubjectScenarioOrder(parseOptions?.scenarioIDs, parseOptions?.seed);
        const getScenarioResults = [];
        for (const getScenarioID of getScenarioOrder) {
            getScenarioResults.push(await handleSubjectMeasureScenario(getScenarioID, parseOptions));
        }
        return {
            getScenarioOrder: getScenarioOrder,
            getScenarioResults: getScenarioResults
        };
    }

    async function handleSubjectRegister(parseFramework) {
        await handleSubjectWait(() => !!document.querySelector("#benchmark-app") && !!document.querySelector("#btn-core-render"), 20000, "benchmark subject ready");
        const getWorkerStatusNode = document.querySelector("#metric-worker-status");
        if (getWorkerStatusNode) {
            await handleSubjectWait(() => {
                const getWorkerStatusText = (document.querySelector("#metric-worker-status")?.textContent || "").trim().toLowerCase();
                return getWorkerStatusText === "ready";
            }, 20000, "benchmark worker ready");
        }
        document.body.dataset.framework = parseFramework;
        window.__example201Subject = {
            getFramework: parseFramework,
            getScenarioIDs: storeBenchmarkScenarioConfigs.map((parseConfig) => parseConfig.id),
            getScenarioLabels: storeBenchmarkScenarioConfigs.map((parseConfig) => ({
                getScenarioID: parseConfig.id,
                getScenarioLabel: parseConfig.label,
                getCategory: parseConfig.category,
                getRequestedWork: parseConfig.requestedWork,
                getCorrectnessCheck: parseConfig.correctnessCheck,
                getFinishLine: parseConfig.finishLine
            })),
            isReady: true,
            measureScenario: handleSubjectMeasureScenario,
            measureAllScenarios: handleSubjectMeasureAllScenarios
        };
    }

    window.__example201RegisterSubject = function (parseFramework) {
        return handleSubjectRegister(parseFramework).catch((parseErr) => {
            window.__example201SubjectError = String(parseErr);
            throw parseErr;
        });
    };
})();
