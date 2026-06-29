(function () {
    function buildRunnerWorkerLabel(parseWorkerCount) {
        return `Runtime 2 (${parseWorkerCount} Worker${parseWorkerCount === 1 ? "" : "s"})`;
    }

    function buildRunnerWorkerFramework(parseWorkerCount) {
        return `runtime2-workers${parseWorkerCount}`;
    }

    function buildRunnerWorkerCounts(parseQuery) {
        const getCountsText = (parseQuery.get("runtime2WorkerCounts") || "").trim();
        const getLegacyCountText = (parseQuery.get("runtime2Workers") || "").trim();
        const getCountSet = new Set();
        function handleRunnerStoreWorkerCount(parseCount) {
            if (!Number.isFinite(parseCount) || parseCount < 1) {
                return;
            }
            getCountSet.add(Math.max(1, Math.trunc(parseCount)));
        }
        if (getCountsText) {
            for (const getCountToken of getCountsText.split(",")) {
                handleRunnerStoreWorkerCount(Number.parseInt(getCountToken.trim(), 10));
            }
        } else if (getLegacyCountText) {
            handleRunnerStoreWorkerCount(1);
            handleRunnerStoreWorkerCount(Number.parseInt(getLegacyCountText, 10));
        } else {
            handleRunnerStoreWorkerCount(1);
            handleRunnerStoreWorkerCount(2);
            handleRunnerStoreWorkerCount(4);
            handleRunnerStoreWorkerCount(8);
        }
        if (getCountSet.size === 0) {
            handleRunnerStoreWorkerCount(1);
            handleRunnerStoreWorkerCount(2);
            handleRunnerStoreWorkerCount(4);
            handleRunnerStoreWorkerCount(8);
        }
        return Array.from(getCountSet.values()).sort((parseLeft, parseRight) => parseLeft - parseRight);
    }

    function buildRunnerSubjectQuery(parseQuery) {
        const getSubjectQuery = {};
        const getRuntime2WorkScale = (parseQuery.get("runtime2WorkScale") || "").trim();
        const getRuntime2Dispatch = (parseQuery.get("runtime2Dispatch") || "").trim();
        const getRuntime2CoreFastPath = (parseQuery.get("runtime2CoreFastPath") || "").trim();
        if (getRuntime2WorkScale) {
            getSubjectQuery.runtime2WorkScale = getRuntime2WorkScale;
        }
        if (getRuntime2Dispatch) {
            getSubjectQuery.runtime2Dispatch = getRuntime2Dispatch;
        }
        if (getRuntime2CoreFastPath) {
            getSubjectQuery.runtime2CoreFastPath = getRuntime2CoreFastPath;
        }
        return getSubjectQuery;
    }

    function buildRunnerConfig() {
        const getQuery = new URLSearchParams(window.location.search);
        return {
            getIterations: Math.max(1, Number.parseInt(getQuery.get("iterations") || "7", 10) || 7),
            getWarmups: Math.max(0, Number.parseInt(getQuery.get("warmups") || "2", 10) || 2),
            getSeed: Math.max(1, Number.parseInt(getQuery.get("seed") || String(Date.now()), 10) || Date.now()),
            getRuntime2WorkerCounts: buildRunnerWorkerCounts(getQuery),
            getSubjectSet: (getQuery.get("subjectSet") || "").trim().toLowerCase(),
            getSubjectQuery: buildRunnerSubjectQuery(getQuery)
        };
    }

    function buildRunnerSubjectConfigs(parseConfig) {
        const getSubjects = [];
        if (parseConfig.getSubjectSet !== "runtime2-scaling") {
            getSubjects.push(
                {
                    getFramework: "react",
                    getLabel: "React 19.2.4",
                    getURL: "./react/"
                },
                {
                    getFramework: "runtime1",
                    getLabel: "Runtime 1",
                    getURL: "./runtime/"
                }
            );
        }
        for (const getWorkerCount of parseConfig.getRuntime2WorkerCounts) {
            getSubjects.push({
                getFramework: buildRunnerWorkerFramework(getWorkerCount),
                getFrameID: buildRunnerWorkerFramework(getWorkerCount),
                getLabel: buildRunnerWorkerLabel(getWorkerCount),
                getURL: "./runtime2-workers/",
                getWorkerCount: getWorkerCount
            });
        }
        return getSubjects;
    }

    function buildRunnerDomScore(parseFactor) {
        if (!Number.isFinite(parseFactor) || parseFactor <= 0) {
            return 0;
        }
        return Math.round(parseFactor * 100);
    }

    function buildRunnerShuffledList(parseItems, parseSeed) {
        const getItems = parseItems.slice();
        let getState = (Number.parseInt(String(parseSeed || "1"), 10) || 1) >>> 0;
        function buildRunnerNextRandom() {
            getState = (getState * 1664525 + 1013904223) >>> 0;
            return getState / 0x100000000;
        }
        for (let parseIndex = getItems.length - 1; parseIndex > 0; parseIndex--) {
            const getSwapIndex = Math.floor(buildRunnerNextRandom() * (parseIndex + 1));
            const getTemp = getItems[parseIndex];
            getItems[parseIndex] = getItems[getSwapIndex];
            getItems[getSwapIndex] = getTemp;
        }
        return getItems;
    }

    function buildRunnerRotatedList(parseItems, parseOffset) {
        if (!Array.isArray(parseItems) || parseItems.length === 0) {
            return [];
        }
        const getOffset = ((Number.parseInt(String(parseOffset || "0"), 10) || 0) % parseItems.length + parseItems.length) % parseItems.length;
        return parseItems.slice(getOffset).concat(parseItems.slice(0, getOffset));
    }

    function handleRunnerDelay(parseDurationMs) {
        return new Promise((parseResolve) => {
            window.setTimeout(parseResolve, parseDurationMs);
        });
    }

    function getRunnerFrame(parseSubjectConfig) {
        const getFrameID = parseSubjectConfig.getFrameID || parseSubjectConfig.getFramework;
        return document.querySelector(`[data-benchmark-frame="${getFrameID}"]`);
    }

    function renderRunnerSubjectFrames(parseConfig) {
        const getFrameGridNode = document.querySelector("#benchmark-frame-grid");
        if (!getFrameGridNode) {
            return;
        }
        const getSubjectConfigs = buildRunnerSubjectConfigs(parseConfig);
        getFrameGridNode.innerHTML = getSubjectConfigs.map((parseSubjectConfig) => {
            const getFrameID = parseSubjectConfig.getFrameID || parseSubjectConfig.getFramework;
            const getPathLabel = parseSubjectConfig.getURL.replace("./", "");
            return `
                <article class="benchmark-card benchmark-stack" style="padding: 18px;">
                    <div class="benchmark-inline" style="justify-content: space-between;">
                        <h2 style="margin: 0; font-size: 20px;">${parseSubjectConfig.getLabel}</h2>
                        <span class="benchmark-muted">${getPathLabel}</span>
                    </div>
                    <iframe class="benchmark-frame" data-benchmark-frame="${getFrameID}" title="${parseSubjectConfig.getLabel} benchmark subject"></iframe>
                </article>
            `;
        }).join("");
    }

    async function handleRunnerLoadSubject(parseConfig, parseSubjectConfig) {
        const getFrame = getRunnerFrame(parseSubjectConfig);
        if (!getFrame) {
            throw new Error("missing frame for " + parseSubjectConfig.getFramework);
        }
        const getURL = new URL(parseSubjectConfig.getURL, window.location.href);
        getURL.searchParams.set("framework", parseSubjectConfig.getFramework);
        if (parseSubjectConfig.getWorkerCount) {
            getURL.searchParams.set("workers", String(parseSubjectConfig.getWorkerCount));
        }
        for (const [getQueryKey, getQueryValue] of Object.entries(parseConfig.getSubjectQuery || {})) {
            getURL.searchParams.set(getQueryKey, getQueryValue);
        }
        getURL.searchParams.set("v", String(Date.now()));
        getFrame.src = getURL.toString();
        const getStartedAt = performance.now();
        while ((performance.now() - getStartedAt) < 45000) {
            let getSubjectWindow = null;
            try {
                getSubjectWindow = getFrame.contentWindow;
            } catch (parseErr) {
                getSubjectWindow = null;
            }
            if (getSubjectWindow) {
                try {
                    if (getSubjectWindow.__example201SubjectError) {
                        throw new Error(`${parseSubjectConfig.getFramework} subject error: ${getSubjectWindow.__example201SubjectError}`);
                    }
                    if (getSubjectWindow.__example201Subject && getSubjectWindow.__example201Subject.isReady) {
                        return getFrame;
                    }
                } catch (parseErr) {
                    if (parseErr && parseErr.name === "SecurityError") {
                        await handleRunnerDelay(20);
                        continue;
                    }
                    throw parseErr;
                }
            }
            await handleRunnerDelay(20);
        }
        let getDebugURL = "(no contentWindow)";
        try {
            if (getFrame.contentWindow) {
                getDebugURL = String(getFrame.contentWindow.location.href || "(unknown href)");
            }
        } catch (parseErr) {
            getDebugURL = `(unavailable: ${parseErr && parseErr.name ? parseErr.name : "security error"})`;
        }
        let getDebugBody = "(no body)";
        try {
            if (getFrame.contentDocument && getFrame.contentDocument.body) {
                getDebugBody = String(getFrame.contentDocument.body.textContent || "").slice(0, 240);
            }
        } catch (parseErr) {
            getDebugBody = `(unavailable: ${parseErr && parseErr.name ? parseErr.name : "security error"})`;
        }
        throw new Error(`timed out waiting for ${parseSubjectConfig.getFramework} subject load; url=${getDebugURL}; body=${getDebugBody}`);
    }

    function buildRunnerScenarioOrder(parseSubjectWindow, parseSeed) {
        const getScenarioIDs = Array.isArray(parseSubjectWindow?.__example201Subject?.getScenarioIDs)
            ? parseSubjectWindow.__example201Subject.getScenarioIDs.slice()
            : [];
        return buildRunnerShuffledList(getScenarioIDs, parseSeed);
    }

    function buildRunnerScoreReferenceByID(parseScoreReference) {
        const getReferenceByID = new Map();
        for (const getScenarioReference of parseScoreReference?.getScenarioReference || []) {
            if (!getScenarioReference?.getScenarioID) {
                continue;
            }
            getReferenceByID.set(getScenarioReference.getScenarioID, getScenarioReference);
        }
        return getReferenceByID;
    }

    function buildRunnerScoreFactor(parseReferenceMs, parseMeasuredMs) {
        if (!Number.isFinite(parseReferenceMs) || parseReferenceMs <= 0 || !Number.isFinite(parseMeasuredMs) || parseMeasuredMs <= 0) {
            return 0;
        }
        return parseReferenceMs / parseMeasuredMs;
    }

    function buildRunnerDomReadyRepresentativeMs(parseScenarioResult) {
        const getRepresentativeMs = Number(parseScenarioResult?.getDomReadyRepresentativeMs || 0);
        if (Number.isFinite(getRepresentativeMs) && getRepresentativeMs > 0) {
            return getRepresentativeMs;
        }
        const getMeanMs = Number(parseScenarioResult?.getDomReadyMeanMs || 0);
        return Number.isFinite(getMeanMs) && getMeanMs > 0 ? getMeanMs : 0;
    }

    function formatRunnerDomReadyStability(parseFramework) {
        const getCV = Number(parseFramework.getDomReadyCoefficientOfVariation || 0);
        if (!parseFramework.hasDomReadyBimodalSamples) {
            return `CV ${getCV.toFixed(3)}`;
        }
        return `bimodal gap ${parseFramework.getDomReadyBimodalGapMs.toFixed(3)} ms, CV ${getCV.toFixed(3)}`;
    }

    async function handleRunnerLoadScoreReference() {
        const getReferenceURL = new URL("./score-reference.json", window.location.href);
        getReferenceURL.searchParams.set("v", "20260407");
        const getResponse = await window.fetch(getReferenceURL.toString(), {
            cache: "no-store"
        });
        if (!getResponse.ok) {
            throw new Error("load score reference: " + getResponse.status);
        }
        return getResponse.json();
    }

    async function handleRunnerLoadSubjects(parseConfig, parseFrameworkOrder, parseStatusNode) {
        const getLoadedSubjects = [];
        for (const getSubjectConfig of parseFrameworkOrder) {
            const getFrame = await handleRunnerLoadSubject(parseConfig, getSubjectConfig);
            let getSubjectWindow = null;
            try {
                getSubjectWindow = getFrame.contentWindow;
            } catch (parseErr) {
                throw new Error(`subject window unavailable for ${getSubjectConfig.getFramework}: ${parseErr && parseErr.name ? parseErr.name : parseErr}`);
            }
            if (!getSubjectWindow || !getSubjectWindow.__example201Subject || !getSubjectWindow.__example201Subject.isReady) {
                throw new Error(`subject contract missing for ${getSubjectConfig.getFramework}`);
            }
            if (parseStatusNode) {
                parseStatusNode.textContent = "Loaded " + getSubjectConfig.getLabel + ".";
            }
            getLoadedSubjects.push({
                getConfig: getSubjectConfig,
                getWindow: getSubjectWindow
            });
        }
        return getLoadedSubjects;
    }

    async function handleRunnerMeasureScenarios(parseConfig, parseLoadedSubjects, parseScenarioOrder, parseStatusNode) {
        const getScenarioResultsByFramework = new Map();
        const getScenarioFrameworkOrders = {};
        for (const getLoadedSubject of parseLoadedSubjects) {
            getScenarioResultsByFramework.set(getLoadedSubject.getConfig.getFramework, []);
        }
        for (let parseScenarioIndex = 0; parseScenarioIndex < parseScenarioOrder.length; parseScenarioIndex++) {
            const getScenarioID = parseScenarioOrder[parseScenarioIndex];
            const getScenarioLoadedSubjects = buildRunnerRotatedList(parseLoadedSubjects, parseScenarioIndex);
            getScenarioFrameworkOrders[getScenarioID] = getScenarioLoadedSubjects.map((parseLoadedSubject) => parseLoadedSubject.getConfig.getFramework);
            for (const getLoadedSubject of getScenarioLoadedSubjects) {
                if (parseStatusNode) {
                    parseStatusNode.textContent = "Running " + getLoadedSubject.getConfig.getLabel + " / " + getScenarioID + " (" + (parseScenarioIndex + 1) + "/" + parseScenarioOrder.length + ")...";
                }
                const getScenarioResult = await getLoadedSubject.getWindow.__example201Subject.measureScenario(getScenarioID, {
                    iterations: parseConfig.getIterations,
                    warmups: parseConfig.getWarmups
                });
                getScenarioResultsByFramework.get(getLoadedSubject.getConfig.getFramework).push(getScenarioResult);
            }
        }
        return {
            getScenarioFrameworkOrders: getScenarioFrameworkOrders,
            getScenarioResultsByFramework: getScenarioResultsByFramework
        };
    }

    function buildRunnerScenarioRows(parseReport, parseScoreReference) {
        const getScoreReferenceByID = buildRunnerScoreReferenceByID(parseScoreReference);
        const getScenarioRows = [];
        const getScenarioLabels = parseReport.getScenarioLabels || [];
        for (const getScenarioLabel of getScenarioLabels) {
            const getRow = {
                getScenarioID: getScenarioLabel.getScenarioID,
                getScenarioLabel: getScenarioLabel.getScenarioLabel,
                getCategory: getScenarioLabel.getCategory,
                getRequestedWork: getScenarioLabel.getRequestedWork,
                getCorrectnessCheck: getScenarioLabel.getCorrectnessCheck,
                getFinishLine: getScenarioLabel.getFinishLine,
                getFrameworks: []
            };
            let getFastestDomReady = Number.POSITIVE_INFINITY;
            let getFastestPaintVisible = Number.POSITIVE_INFINITY;
            const getScenarioReference = getScoreReferenceByID.get(getScenarioLabel.getScenarioID);
            for (const getFramework of parseReport.getFrameworks) {
                const getScenarioResult = getFramework.getScenarioResults.find((parseResult) => parseResult.getScenarioID === getScenarioLabel.getScenarioID);
                if (!getScenarioResult) {
                    continue;
                }
                const getDomReadyRepresentativeMs = buildRunnerDomReadyRepresentativeMs(getScenarioResult);
                getFastestDomReady = Math.min(getFastestDomReady, getDomReadyRepresentativeMs);
                getFastestPaintVisible = Math.min(getFastestPaintVisible, getScenarioResult.getPaintVisibleMeanMs);
                getRow.getFrameworks.push({
                    getFramework: getFramework.getFramework,
                    getLabel: getFramework.getLabel,
                    getDomReadyRepresentativeMs: getDomReadyRepresentativeMs,
                    getDomReadyRepresentativeSource: getScenarioResult.getDomReadyRepresentativeSource || "mean",
                    getDomReadyMeanMs: getScenarioResult.getDomReadyMeanMs,
                    getDomReadyMedianMs: getScenarioResult.getDomReadyMedianMs,
                    getDomReadyCoefficientOfVariation: getScenarioResult.getDomReadyCoefficientOfVariation || 0,
                    hasDomReadyBimodalSamples: !!getScenarioResult.hasDomReadyBimodalSamples,
                    getDomReadyBimodalGapMs: getScenarioResult.getDomReadyBimodalGapMs || 0,
                    getDomReadyBimodalLowMeanMs: getScenarioResult.getDomReadyBimodalLowMeanMs || 0,
                    getDomReadyBimodalHighMeanMs: getScenarioResult.getDomReadyBimodalHighMeanMs || 0,
                    getDomReadyBimodalLowCount: getScenarioResult.getDomReadyBimodalLowCount || 0,
                    getDomReadyBimodalHighCount: getScenarioResult.getDomReadyBimodalHighCount || 0,
                    getPaintVisibleMeanMs: getScenarioResult.getPaintVisibleMeanMs,
                    getPaintVisibleMedianMs: getScenarioResult.getPaintVisibleMedianMs,
                    getPaintAfterDomMeanMs: getScenarioResult.getPaintAfterDomMeanMs,
                    getChildListMutationMean: getScenarioResult.getChildListMutationMean,
                    getAttributeMutationMean: getScenarioResult.getAttributeMutationMean,
                    getCharacterDataMutationMean: getScenarioResult.getCharacterDataMutationMean,
                    getAddedNodeMean: getScenarioResult.getAddedNodeMean,
                    getRemovedNodeMean: getScenarioResult.getRemovedNodeMean,
                    getLongTaskCountMean: getScenarioResult.getLongTaskCountMean,
                    getLongTaskDurationMeanMs: getScenarioResult.getLongTaskDurationMeanMs,
                    hasWorkerMetrics: !!getScenarioResult.hasWorkerMetrics,
                    getWorkerCountMean: getScenarioResult.getWorkerCountMean || 0,
                    getWorkerBatchCountMean: getScenarioResult.getWorkerBatchCountMean || 0,
                    getWorkerBatchMeanMs: getScenarioResult.getWorkerBatchMeanMs || 0,
                    getWorkerPreparedItemsMean: getScenarioResult.getWorkerPreparedItemsMean || 0
                });
            }
            const getReactFramework = getRow.getFrameworks.find((parseFramework) => parseFramework.getFramework === "react");
            const getReactDomReadyMs = getReactFramework ? getReactFramework.getDomReadyRepresentativeMs : 0;
            for (const getFramework of getRow.getFrameworks) {
                getFramework.getRelativeDomReady = Number((getFramework.getDomReadyRepresentativeMs / getFastestDomReady).toFixed(3));
                if (getScenarioReference?.getDomReadyMeanMs > 0) {
                    getFramework.hasDomScoreReference = true;
                    getFramework.isWorkerRelevant = !!getScenarioReference.isWorkerRelevant;
                    getFramework.getDomReadyReferenceMs = getScenarioReference.getDomReadyMeanMs;
                    getFramework.getDomReadyScoreFactor = Number(buildRunnerScoreFactor(getScenarioReference.getDomReadyMeanMs, getFramework.getDomReadyRepresentativeMs).toFixed(3));
                    getFramework.getDomScore = buildRunnerDomScore(getFramework.getDomReadyScoreFactor);
                } else {
                    getFramework.hasDomScoreReference = false;
                    getFramework.isWorkerRelevant = false;
                    getFramework.getDomReadyReferenceMs = 0;
                    getFramework.getDomReadyScoreFactor = 0;
                    getFramework.getDomScore = 0;
                }
                if (getReactDomReadyMs > 0 && getFramework.getDomReadyRepresentativeMs > 0) {
                    getFramework.hasReactBaseline = true;
                    getFramework.getDomReadyDeltaVsReactMs = Number((getReactDomReadyMs - getFramework.getDomReadyRepresentativeMs).toFixed(3));
                    getFramework.getDomReadySpeedupVsReact = Number((getReactDomReadyMs / getFramework.getDomReadyRepresentativeMs).toFixed(3));
                } else {
                    getFramework.hasReactBaseline = false;
                    getFramework.getDomReadyDeltaVsReactMs = 0;
                    getFramework.getDomReadySpeedupVsReact = 0;
                }
            }
            for (const getFramework of getRow.getFrameworks) {
                getFramework.getRelativePaintVisible = Number((getFramework.getPaintVisibleMeanMs / getFastestPaintVisible).toFixed(3));
            }
            getRow.getFrameworks.sort((parseLeft, parseRight) => {
                if (parseLeft.getDomReadyRepresentativeMs === parseRight.getDomReadyRepresentativeMs) {
                    return parseLeft.getPaintVisibleMeanMs - parseRight.getPaintVisibleMeanMs;
                }
                return parseLeft.getDomReadyRepresentativeMs - parseRight.getDomReadyRepresentativeMs;
            });
            getScenarioRows.push(getRow);
        }
        return getScenarioRows;
    }

    function buildRunnerCategoryRows(parseScenarioRows) {
        const getCategoryMap = new Map();
        for (const getScenarioRow of parseScenarioRows) {
            if (!getCategoryMap.has(getScenarioRow.getCategory)) {
                getCategoryMap.set(getScenarioRow.getCategory, []);
            }
            getCategoryMap.get(getScenarioRow.getCategory).push(getScenarioRow);
        }
        return Array.from(getCategoryMap.entries()).map(([parseCategory, parseRows]) => {
            const getFrameworkMap = new Map();
            for (const getScenarioRow of parseRows) {
                for (let parseIndex = 0; parseIndex < getScenarioRow.getFrameworks.length; parseIndex++) {
                    const getFramework = getScenarioRow.getFrameworks[parseIndex];
                    if (!getFrameworkMap.has(getFramework.getFramework)) {
                        getFrameworkMap.set(getFramework.getFramework, {
                        getFramework: getFramework.getFramework,
                        getLabel: getFramework.getLabel,
                        getDomReadyRelativeProduct: 1,
                        getPaintVisibleRelativeProduct: 1,
                        getDomReadyScoreFactorProduct: 1,
                        getDomReadyRepresentativeSumMs: 0,
                        getDomReadySumMs: 0,
                        getPaintVisibleSumMs: 0,
                        getDomReadyDeltaVsReactSumMs: 0,
                        getScoreScenarioCount: 0,
                        hasReactBaseline: false,
                        getScenarioCount: 0,
                        getReactScenarioCount: 0,
                        getScenarioWins: 0
                        });
                    }
                    const getCategoryFramework = getFrameworkMap.get(getFramework.getFramework);
                    getCategoryFramework.getDomReadyRelativeProduct *= Math.max(getFramework.getRelativeDomReady, 0.0001);
                    getCategoryFramework.getPaintVisibleRelativeProduct *= Math.max(getFramework.getRelativePaintVisible, 0.0001);
                    getCategoryFramework.getDomReadyRepresentativeSumMs += getFramework.getDomReadyRepresentativeMs;
                    getCategoryFramework.getDomReadySumMs += getFramework.getDomReadyMeanMs;
                    getCategoryFramework.getPaintVisibleSumMs += getFramework.getPaintVisibleMeanMs;
                    if (getFramework.hasDomScoreReference) {
                        getCategoryFramework.getScoreScenarioCount += 1;
                        getCategoryFramework.getDomReadyScoreFactorProduct *= Math.max(getFramework.getDomReadyScoreFactor, 0.0001);
                    }
                    if (getFramework.hasReactBaseline) {
                        getCategoryFramework.hasReactBaseline = true;
                        getCategoryFramework.getReactScenarioCount += 1;
                        getCategoryFramework.getDomReadyDeltaVsReactSumMs += getFramework.getDomReadyDeltaVsReactMs;
                    }
                    getCategoryFramework.getScenarioCount += 1;
                    if (parseIndex === 0 || getFramework.getRelativeDomReady === 1) {
                        getCategoryFramework.getScenarioWins += 1;
                    }
                }
            }
            const getFrameworks = Array.from(getFrameworkMap.values()).map((parseFramework) => ({
                getFramework: parseFramework.getFramework,
                getLabel: parseFramework.getLabel,
                getScenarioCount: parseFramework.getScenarioCount,
                getScenarioWins: parseFramework.getScenarioWins,
                hasReactBaseline: parseFramework.hasReactBaseline,
                getDomReadyRepresentativeMs: Number((parseFramework.getDomReadyRepresentativeSumMs / parseFramework.getScenarioCount).toFixed(3)),
                getDomReadyMeanMs: Number((parseFramework.getDomReadySumMs / parseFramework.getScenarioCount).toFixed(3)),
                getPaintVisibleMeanMs: Number((parseFramework.getPaintVisibleSumMs / parseFramework.getScenarioCount).toFixed(3)),
                getDomReadyGeometricRelative: Number(Math.pow(parseFramework.getDomReadyRelativeProduct, 1 / parseFramework.getScenarioCount).toFixed(3)),
                getPaintVisibleGeometricRelative: Number(Math.pow(parseFramework.getPaintVisibleRelativeProduct, 1 / parseFramework.getScenarioCount).toFixed(3)),
                getScoreScenarioCount: parseFramework.getScoreScenarioCount,
                getDomReadyDeltaVsReactMs: parseFramework.getReactScenarioCount > 0
                    ? Number((parseFramework.getDomReadyDeltaVsReactSumMs / parseFramework.getReactScenarioCount).toFixed(3))
                    : 0,
                getDomReadyGeometricScoreFactor: parseFramework.getScoreScenarioCount > 0
                    ? Number(Math.pow(parseFramework.getDomReadyScoreFactorProduct, 1 / parseFramework.getScoreScenarioCount).toFixed(3))
                    : 0,
                getDomScore: parseFramework.getScoreScenarioCount > 0
                    ? buildRunnerDomScore(Math.pow(parseFramework.getDomReadyScoreFactorProduct, 1 / parseFramework.getScoreScenarioCount))
                    : 0
            })).sort((parseLeft, parseRight) => {
                if (parseLeft.getScoreScenarioCount !== parseRight.getScoreScenarioCount) {
                    return parseRight.getScoreScenarioCount - parseLeft.getScoreScenarioCount;
                }
                if (parseLeft.getDomScore === parseRight.getDomScore) {
                    return parseLeft.getFramework.localeCompare(parseRight.getFramework);
                }
                return parseRight.getDomScore - parseLeft.getDomScore;
            });
            return {
                getCategory: parseCategory,
                getScenarioCount: parseRows.length,
                getFrameworks: getFrameworks
            };
        });
    }

    function buildRunnerScalingRows(parseScenarioRows) {
        return parseScenarioRows
            .map((parseScenarioRow) => {
                const getWorkerFrameworks = parseScenarioRow.getFrameworks
                    .filter((parseFramework) => parseFramework.hasWorkerMetrics)
                    .slice()
                    .sort((parseLeft, parseRight) => parseLeft.getWorkerCountMean - parseRight.getWorkerCountMean);
                if (getWorkerFrameworks.length === 0) {
                    return null;
                }
                if (!getWorkerFrameworks.some((parseFramework) => parseFramework.getWorkerBatchMeanMs > 0)) {
                    return null;
                }
                const getBaselineBatchMs = getWorkerFrameworks[0].getWorkerBatchMeanMs || 0;
                const getBaselineDomReadyMs = getWorkerFrameworks[0].getDomReadyRepresentativeMs || 0;
                return {
                    getScenarioID: parseScenarioRow.getScenarioID,
                    getScenarioLabel: parseScenarioRow.getScenarioLabel,
                    getCategory: parseScenarioRow.getCategory,
                    getFrameworks: getWorkerFrameworks.map((parseFramework) => ({
                        getLabel: parseFramework.getLabel,
                        getWorkerCountMean: parseFramework.getWorkerCountMean,
                        getWorkerBatchMeanMs: parseFramework.getWorkerBatchMeanMs,
                        getWorkerPreparedItemsMean: parseFramework.getWorkerPreparedItemsMean,
                        getDomReadyMeanMs: parseFramework.getDomReadyMeanMs,
                        getDomReadyRepresentativeMs: parseFramework.getDomReadyRepresentativeMs,
                        getPaintVisibleMeanMs: parseFramework.getPaintVisibleMeanMs,
                        getRelativeDomReadySpeedup: getBaselineDomReadyMs > 0 && parseFramework.getDomReadyRepresentativeMs > 0
                            ? Number((getBaselineDomReadyMs / parseFramework.getDomReadyRepresentativeMs).toFixed(3))
                            : 0,
                        getRelativeBatchSpeedup: getBaselineBatchMs > 0 && parseFramework.getWorkerBatchMeanMs > 0
                            ? Number((getBaselineBatchMs / parseFramework.getWorkerBatchMeanMs).toFixed(3))
                            : 0
                    }))
                };
            })
            .filter((parseRow) => !!parseRow);
    }

    function renderRunnerSummary(parseReport, parseScoreReference) {
        const getResultsNode = document.querySelector("#benchmark-results");
        const getRawNode = document.querySelector("#benchmark-report-json");
        const getCategoryNode = document.querySelector("#benchmark-category-results");
        const getScalingNode = document.querySelector("#benchmark-scaling-results");
        if (!getResultsNode || !getRawNode || !getCategoryNode || !getScalingNode) {
            return;
        }
        const getScenarioRows = buildRunnerScenarioRows(parseReport, parseScoreReference);
        const getCategoryRows = buildRunnerCategoryRows(getScenarioRows);
        const getScalingRows = buildRunnerScalingRows(getScenarioRows);
        getCategoryNode.innerHTML = getCategoryRows.map((parseCategoryRow) => {
            const getFrameworkRows = parseCategoryRow.getFrameworks.map((parseFramework) => `
                <tr>
                    <td>${parseFramework.getLabel}</td>
                    <td>${parseFramework.getDomScore}</td>
                    <td>${parseFramework.getDomReadyRepresentativeMs.toFixed(3)} ms</td>
                    <td>${parseFramework.getDomReadyMeanMs.toFixed(3)} ms</td>
                    <td>${parseFramework.hasReactBaseline ? `${parseFramework.getDomReadyDeltaVsReactMs >= 0 ? "+" : ""}${parseFramework.getDomReadyDeltaVsReactMs.toFixed(3)} ms` : "n/a"}</td>
                    <td>${parseFramework.getPaintVisibleMeanMs.toFixed(3)} ms</td>
                    <td>${parseFramework.getScoreScenarioCount > 0 ? `${parseFramework.getDomReadyGeometricScoreFactor.toFixed(3)}x` : "n/a"}</td>
                    <td>${parseFramework.getScenarioWins}</td>
                </tr>
            `).join("");
            return `
                <section class="benchmark-card benchmark-stack" style="padding: 18px;">
                    <div class="benchmark-inline" style="justify-content: space-between;">
                        <h3 style="margin: 0; font-size: 18px;">${parseCategoryRow.getCategory}</h3>
                        <span class="benchmark-muted">${parseCategoryRow.getScenarioCount} scenarios</span>
                    </div>
                    <table class="benchmark-table" style="margin-top: 12px;">
                        <thead>
                            <tr>
                                <th>Framework</th>
                                <th>DOM Score</th>
                                <th>Avg DOM Ready Rep.</th>
                                <th>Avg DOM Ready Mean</th>
                                <th>Avg DOM vs React</th>
                                <th>Avg Paint Proxy</th>
                                <th>Geom. DOM Score Factor</th>
                                <th>Wins</th>
                            </tr>
                        </thead>
                        <tbody>${getFrameworkRows}</tbody>
                    </table>
                </section>
            `;
        }).join("");
        getScalingNode.innerHTML = getScalingRows.map((parseScalingRow) => {
            const getFrameworkRows = parseScalingRow.getFrameworks.map((parseFramework) => `
                <tr>
                    <td>${parseFramework.getLabel}</td>
                    <td>${parseFramework.getDomReadyRepresentativeMs.toFixed(3)} ms</td>
                    <td>${parseFramework.getRelativeDomReadySpeedup.toFixed(3)}x</td>
                    <td>${parseFramework.getWorkerBatchMeanMs.toFixed(3)} ms</td>
                    <td>${parseFramework.getRelativeBatchSpeedup.toFixed(3)}x</td>
                    <td>${parseFramework.getWorkerPreparedItemsMean.toFixed(1)}</td>
                    <td>${parseFramework.getPaintVisibleMeanMs.toFixed(3)} ms</td>
                </tr>
            `).join("");
            return `
                <section class="benchmark-card benchmark-stack" style="padding: 18px;">
                    <div class="benchmark-inline" style="justify-content: space-between;">
                        <h3 style="margin: 0; font-size: 18px;">${parseScalingRow.getScenarioLabel}</h3>
                        <span class="benchmark-muted">${parseScalingRow.getScenarioID} / ${parseScalingRow.getCategory}</span>
                    </div>
                    <p class="benchmark-muted" style="margin: 0;">RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.</p>
                    <table class="benchmark-table" style="margin-top: 12px;">
                        <thead>
                            <tr>
                                <th>Framework</th>
                                <th>DOM Ready Rep.</th>
                                <th>DOM Ready Speedup vs Smallest Worker Count</th>
                                <th>Worker Batch</th>
                                <th>Worker Batch Speedup vs Smallest Worker Count</th>
                                <th>Prepared Items</th>
                                <th>Paint Proxy</th>
                            </tr>
                        </thead>
                        <tbody>${getFrameworkRows}</tbody>
                    </table>
                </section>
            `;
        }).join("");
        getResultsNode.innerHTML = getScenarioRows.map((parseRow) => {
            const getFrameworkRows = parseRow.getFrameworks.map((parseFramework) => {
                const isFastest = parseFramework.getRelativeDomReady === 1;
                const getReactText = parseFramework.hasReactBaseline
                    ? `${parseFramework.getDomReadyDeltaVsReactMs >= 0 ? "+" : ""}${parseFramework.getDomReadyDeltaVsReactMs.toFixed(3)} ms / ${parseFramework.getDomReadySpeedupVsReact.toFixed(3)}x`
                    : "n/a";
                return `
                    <tr>
                        <td>${parseFramework.getLabel}</td>
                        <td>${parseFramework.getDomScore}</td>
                        <td>${parseFramework.getDomReadyRepresentativeMs.toFixed(3)} ms</td>
                        <td>${parseFramework.getDomReadyMeanMs.toFixed(3)} ms</td>
                        <td>${formatRunnerDomReadyStability(parseFramework)}</td>
                        <td>${parseFramework.getPaintVisibleMeanMs.toFixed(3)} ms</td>
                        <td>${parseFramework.getPaintAfterDomMeanMs.toFixed(3)} ms</td>
                        <td>${parseFramework.getChildListMutationMean.toFixed(1)} / ${parseFramework.getAttributeMutationMean.toFixed(1)} / ${parseFramework.getCharacterDataMutationMean.toFixed(1)}</td>
                        <td>${parseFramework.getAddedNodeMean.toFixed(1)} / ${parseFramework.getRemovedNodeMean.toFixed(1)}</td>
                        <td>${parseFramework.hasWorkerMetrics ? `${parseFramework.getWorkerBatchCountMean.toFixed(1)} / ${parseFramework.getWorkerBatchMeanMs.toFixed(3)} ms / ${parseFramework.getWorkerPreparedItemsMean.toFixed(1)}` : "n/a"}</td>
                        <td>${parseFramework.getLongTaskCountMean.toFixed(1)} / ${parseFramework.getLongTaskDurationMeanMs.toFixed(3)} ms</td>
                        <td class="${isFastest ? "benchmark-rank-fastest" : "benchmark-rank-slower"}">${getReactText}</td>
                    </tr>
                `;
            }).join("");
            return `
                <section class="benchmark-card benchmark-stack" style="padding: 18px;">
                    <div class="benchmark-inline" style="justify-content: space-between;">
                        <h3 style="margin: 0; font-size: 18px;">${parseRow.getScenarioLabel}</h3>
                        <span class="benchmark-muted">${parseRow.getScenarioID} / ${parseRow.getCategory}</span>
                    </div>
                    <p class="benchmark-muted" style="margin: 0;">Requested work: ${parseRow.getRequestedWork}</p>
                    <p class="benchmark-muted" style="margin: 0;">Correctness check: ${parseRow.getCorrectnessCheck}</p>
                    <p class="benchmark-muted" style="margin: 0;">Finish line: ${parseRow.getFinishLine}</p>
                    <table class="benchmark-table" style="margin-top: 12px;">
                        <thead>
                            <tr>
                                <th>Framework</th>
                                <th>DOM Score</th>
                                <th>DOM Ready Rep.</th>
                                <th>DOM Ready Mean</th>
                                <th>DOM Stability</th>
                                <th>Paint Proxy</th>
                                <th>Paint-After-DOM</th>
                                <th>Mutations C/A/T</th>
                                <th>Nodes + / -</th>
                                <th>Worker Batches / Last Batch / Items</th>
                                <th>Long Tasks</th>
                                <th>DOM vs React</th>
                            </tr>
                        </thead>
                        <tbody>${getFrameworkRows}</tbody>
                    </table>
                </section>
            `;
        }).join("");
        getRawNode.textContent = JSON.stringify(parseReport, null, 2);
    }

    async function handleRunnerStart() {
        const getStatusNode = document.querySelector("#benchmark-status");
        const getConfig = buildRunnerConfig();
        const getFrameworkOrder = buildRunnerShuffledList(buildRunnerSubjectConfigs(getConfig), getConfig.getSeed);
        const getScoreReference = await handleRunnerLoadScoreReference();
        if (getStatusNode) {
            getStatusNode.textContent = "Loading benchmark subjects...";
        }
        const getLoadedSubjects = await handleRunnerLoadSubjects(getConfig, getFrameworkOrder, getStatusNode);
        const getScenarioLabels = Array.isArray(getLoadedSubjects[0]?.getWindow?.__example201Subject?.getScenarioLabels)
            ? getLoadedSubjects[0].getWindow.__example201Subject.getScenarioLabels.slice()
            : [];
        const getScenarioOrder = buildRunnerScenarioOrder(getLoadedSubjects[0]?.getWindow, getConfig.getSeed);
        const getMeasureResult = await handleRunnerMeasureScenarios(getConfig, getLoadedSubjects, getScenarioOrder, getStatusNode);
        const getFrameworks = getFrameworkOrder.map((parseSubjectConfig) => ({
            getFramework: parseSubjectConfig.getFramework,
            getLabel: parseSubjectConfig.getLabel,
            getScenarioResults: getMeasureResult.getScenarioResultsByFramework.get(parseSubjectConfig.getFramework) || []
        }));
        const getReport = {
            generatedAt: new Date().toISOString(),
            getIterations: getConfig.getIterations,
            getWarmups: getConfig.getWarmups,
            getSeed: getConfig.getSeed,
            getScenarioOrder: getScenarioOrder,
            getFrameworkOrder: getFrameworkOrder.map((parseConfig) => parseConfig.getFramework),
            getScenarioFrameworkOrders: getMeasureResult.getScenarioFrameworkOrders,
            getScenarioLabels: getScenarioLabels,
            getFrameworks: getFrameworks
        };
        window.__example201Report = getReport;
        renderRunnerSummary(getReport, getScoreReference);
        if (getStatusNode) {
            getStatusNode.textContent = "Benchmark run complete.";
        }
        return getReport;
    }

    function handleRunnerBind() {
        const getButton = document.querySelector("#btn-run-benchmark");
        const getConfig = buildRunnerConfig();
        renderRunnerSubjectFrames(getConfig);
        if (!getButton) {
            return;
        }
        getButton.addEventListener("click", () => {
            handleRunnerStart().catch((parseErr) => {
                const getStatusNode = document.querySelector("#benchmark-status");
                if (getStatusNode) {
                    getStatusNode.textContent = "Benchmark failed: " + String(parseErr);
                }
                window.__example201ReportError = String(parseErr);
                throw parseErr;
            });
        });
        window.__example201Runner = {
            runBenchmarks: handleRunnerStart
        };
    }

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", handleRunnerBind);
    } else {
        handleRunnerBind();
    }
})();
